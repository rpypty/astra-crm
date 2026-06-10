package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/audit"
	"github.com/ashpak/astra-crm-backend/internal/auth"
	"github.com/ashpak/astra-crm-backend/internal/config"
	"github.com/ashpak/astra-crm-backend/internal/httpserver"
	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/orders"
	"github.com/ashpak/astra-crm-backend/internal/payouts"
	"github.com/ashpak/astra-crm-backend/internal/platform/logger"
	"github.com/ashpak/astra-crm-backend/internal/platform/postgres"
	"github.com/ashpak/astra-crm-backend/internal/readmodels"
	"github.com/ashpak/astra-crm-backend/internal/reconciliation"
	"github.com/ashpak/astra-crm-backend/internal/requisites"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
	"github.com/ashpak/astra-crm-backend/internal/users"
	db "github.com/ashpak/astra-crm-backend/sqlc/generated"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		fallback := logger.New("development")
		fallback.Error("failed to load config", slog.Any("error", err))
		return 1
	}

	log := logger.New(cfg.AppEnv)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var dbPool *postgres.Pool
	if cfg.DatabaseURL != "" {
		dbPool, err = postgres.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Error("failed to initialize database pool", slog.Any("error", err))
			return 1
		}
		defer dbPool.Close()
	}

	var authService *auth.Service
	var traderService *users.TraderService
	var requisiteService *requisites.Service
	var shiftService *shifts.Service
	var payoutService *payouts.Service
	var importService *imports.Service
	var orderReadService *orders.Service
	var readmodelService *readmodels.Service
	var reconciliationService *reconciliation.Service
	if dbPool != nil {
		queries := db.New(dbPool.Raw())
		userRepository := users.NewRepository(queries)
		requisiteRepository := requisites.NewRepository(queries)
		shiftRepository := shifts.NewRepository(queries)
		payoutRepository := payouts.NewRepository(queries)
		importRepository := imports.NewRepository(dbPool.Raw())
		orderReadRepository := orders.NewRepository(queries)
		reconciliationRepository := reconciliation.NewRepository(dbPool.Raw())
		sessionRepository := auth.NewSessionRepository(queries)
		auditRepository := audit.NewRepository(queries)
		auditService := audit.NewService(auditRepository)

		authService = auth.NewService(userRepository, sessionRepository)
		traderService = users.NewTraderService(userRepository, auditService, auth.HashPassword, auth.NewSessionToken)
		requisiteService = requisites.NewService(requisiteRepository, userRepository, auditService)
		reconciliationService = reconciliation.NewService(reconciliationRepository, auditService)
		shiftService = shifts.NewService(shiftRepository, auditService, reconciliationService)
		payoutService = payouts.NewService(payoutRepository, auditService, reconciliationService)
		importService = imports.NewService(importRepository, auditService, reconciliationService)
		orderReadService = orders.NewService(orderReadRepository)
		readmodelService = readmodels.NewService(dbPool.Raw())
	}

	router := httpserver.NewRouter(log, httpserver.RouterConfig{
		ReadyPinger:       dbPool,
		AuthService:       authService,
		TraderService:     traderService,
		RequisiteService:  requisiteService,
		ShiftService:      shiftService,
		PayoutService:     payoutService,
		ImportService:     importService,
		OrderReadService:  orderReadService,
		ReadmodelService:  readmodelService,
		ReconcileService:  reconciliationService,
		SessionCookieName: cfg.SessionCookieName,
		SessionSecure:     cfg.SessionSecure,
	})
	server := httpserver.New(cfg.HTTPAddr, router)

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", slog.String("addr", cfg.HTTPAddr), slog.String("app_env", cfg.AppEnv))
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("failed to gracefully shutdown http server", slog.Any("error", err))
			return 1
		}

		log.Info("http server stopped")
		return 0
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return 0
		}

		log.Error("http server failed", slog.Any("error", err))
		return 1
	}
}
