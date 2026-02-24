package app

import (
	"context"
	"time"

	"go-rbac-h3-sim/internal/db"
	"go-rbac-h3-sim/internal/services"
)

type Config struct {
	DBPath    string
	JWTSecret []byte
	TokenTTL  time.Duration
}

type App struct {
	DB *db.DB

	Auth      *services.AuthService
	Audit     *services.AuditService
	Analytics *services.AnalyticsService
	Sim       *services.Engine
	Cancel    context.CancelFunc
}

func New(cfg Config) (*App, error) {
	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	if err := d.Migrate(); err != nil {
		return nil, err
	}

	authSvc := services.NewAuth(cfg.JWTSecret, cfg.TokenTTL)
	auditSvc := services.NewAudit(d)
	analyticsSvc := services.NewAnalytics(d)

	ctx, cancel := context.WithCancel(context.Background())
	simEngine := services.NewEngine(d, auditSvc)
	simEngine.Start(ctx)

	return &App{
		DB:        d,
		Auth:      authSvc,
		Audit:     auditSvc,
		Analytics: analyticsSvc,
		Sim:       simEngine,
		Cancel:    cancel,
	}, nil
}
