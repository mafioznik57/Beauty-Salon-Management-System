package services

import (
	"time"

	"go-rbac-h3-sim/internal/db"

	"github.com/uber/h3-go/v4"
)

type AnalyticsService struct {
	db *db.DB
}

func NewAnalytics(d *db.DB) *AnalyticsService {
	return &AnalyticsService{db: d}
}

func (s *AnalyticsService) RecomputeCounts(targetRes int) error {
	rows, err := s.db.SQL.Query(`SELECT h3_index FROM cells`)
	if err != nil {
		return err
	}
	defer rows.Close()

	counts := map[string]int{}

	for rows.Next() {
		var h3s string
		if err := rows.Scan(&h3s); err != nil {
			return err
		}
		var cell h3.Cell
		if err := cell.UnmarshalText([]byte(h3s)); err != nil {
			continue
		}
		parent := cell.Parent(targetRes).String()
		counts[parent]++
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.SQL.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO cell_analytics(h3_parent, resolution, cells_count, updated_at)
        VALUES($1,$2,$3,$4)
        ON CONFLICT(h3_parent, resolution) DO UPDATE SET cells_count=excluded.cells_count, updated_at=excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for parent, c := range counts {
		if _, err := stmt.Exec(parent, targetRes, c, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

type AnalyticsRow struct {
	H3Parent   string `json:"h3Parent"`
	Resolution int    `json:"resolution"`
	CellsCount int    `json:"cellsCount"`
	UpdatedAt  string `json:"updatedAt"`
}

func (s *AnalyticsService) ListCounts(targetRes int, limit int) ([]AnalyticsRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.SQL.Query(`SELECT h3_parent, resolution, cells_count, updated_at
        FROM cell_analytics WHERE resolution=$1 ORDER BY cells_count DESC LIMIT $2`, targetRes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AnalyticsRow{}
	for rows.Next() {
		var r AnalyticsRow
		if err := rows.Scan(&r.H3Parent, &r.Resolution, &r.CellsCount, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
