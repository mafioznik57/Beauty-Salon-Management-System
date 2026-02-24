package services

import (
	"time"

	"go-rbac-h3-sim/internal/db"
)

type AuditService struct {
	db *db.DB
}

func NewAudit(d *db.DB) *AuditService {
	return &AuditService{db: d}
}

func (s *AuditService) Log(actorUserID *int64, action, target string) error {
	_, err := s.db.SQL.Exec(
		`INSERT INTO audit_logs(actor_user_id, action, target, created_at) VALUES($1,$2,$3,$4)`,
		actorUserID, action, target, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

type AuditEntry struct {
	ID          int64  `json:"id"`
	ActorUserID *int64 `json:"actorUserId,omitempty"`
	Action      string `json:"action"`
	Target      string `json:"target"`
	CreatedAt   string `json:"createdAt"`
}

func (s *AuditService) List(limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.SQL.Query(`SELECT id, actor_user_id, action, target, created_at FROM audit_logs ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AuditEntry, 0)
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.Action, &e.Target, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
