package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"rental-architecture-back/internal/db"
)

type EventType string

const (
	EventTick EventType = "TICK"
	EventNoop EventType = "NOOP"
)

type Event struct {
	RunID   int64
	Type    EventType
	Payload map[string]any
}

type Engine struct {
	db    *db.DB
	audit *AuditService
	ch    chan Event
}

func NewEngine(d *db.DB, auditSvc *AuditService) *Engine {
	return &Engine{
		db:    d,
		audit: auditSvc,
		ch:    make(chan Event, 256),
	}
}

func (e *Engine) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-e.ch:
				_ = e.handle(ev)
			}
		}
	}()
}

func (e *Engine) Enqueue(ev Event) {
	e.ch <- ev
}

func (e *Engine) CreateRun(requestedBy int64, ticks int) (int64, error) {
	if ticks <= 0 {
		ticks = 5
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var runID int64
	err := e.db.SQL.QueryRow(
		`INSERT INTO sim_runs(requested_by, status, events_total, events_processed, created_at, updated_at)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		requestedBy, "queued", ticks, 0, now, now,
	).Scan(&runID)
	if err != nil {
		return 0, err
	}

	for i := 0; i < ticks; i++ {
		e.Enqueue(Event{RunID: runID, Type: EventTick, Payload: map[string]any{"i": i + 1}})
	}
	return runID, nil
}

func (e *Engine) GetRun(runID int64) (map[string]any, error) {
	var status string
	var total, processed int
	var createdAt, updatedAt string
	err := e.db.SQL.QueryRow(
		`SELECT status, events_total, events_processed, created_at, updated_at FROM sim_runs WHERE id=$1`,
		runID,
	).Scan(&status, &total, &processed, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": runID, "status": status, "eventsTotal": total, "eventsProcessed": processed,
		"createdAt": createdAt, "updatedAt": updatedAt,
	}, nil
}

func (e *Engine) handle(ev Event) error {
	_, _ = e.db.SQL.Exec(`UPDATE sim_runs SET status=$1, updated_at=$2 WHERE id=$3 AND status='queued'`,
		"running", time.Now().UTC().Format(time.RFC3339), ev.RunID,
	)

	time.Sleep(120 * time.Millisecond)
	tx, err := e.db.SQL.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var total, processed int
	err = tx.QueryRow(`SELECT events_total, events_processed FROM sim_runs WHERE id=$1`, ev.RunID).Scan(&total, &processed)
	if err != nil {
		return err
	}
	processed++

	status := "running"
	if processed >= total {
		status = "done"
	}

	_, err = tx.Exec(`UPDATE sim_runs SET events_processed=$1, status=$2, updated_at=$3 WHERE id=$4`,
		processed, status, time.Now().UTC().Format(time.RFC3339), ev.RunID,
	)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if status == "done" {
		_ = e.audit.Log(nil, "simulation.completed", fmt.Sprintf("run:%d", ev.RunID))
	}
	return nil
}

var _ = sql.ErrNoRows
