package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ashpak/astra-crm-backend/internal/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterConfig struct {
	ReadyPinger ReadyPinger
	AuthService interface {
		AuthService
		Authenticator
	}
	TraderService    TeamleadTraderService
	RequisiteService TeamleadRequisiteService
	ShiftService     TraderShiftService
	PayoutService    TraderPayoutService
	ImportService    ImportService
	OrderReadService OrderReadService
	ReadmodelService TeamleadReadmodelService
	ReconcileService interface {
		TraderReconciliationService
		TeamleadReconciliationService
	}
	SessionCookieName      string
	SessionSecure          bool
	CSRFAllowedOrigins     []string
	LoginRateLimitRequests int
	LoginRateLimitWindow   time.Duration
}

func NewRouter(log *slog.Logger, cfg RouterConfig) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(RequestLogger(log))
	router.Use(Recoverer(log))

	router.Get("/health", HealthHandler())
	router.Get("/ready", ReadyHandler(cfg.ReadyPinger))

	authHandler := NewAuthHandler(cfg.AuthService, cfg.SessionCookieName, cfg.SessionSecure)
	tradersHandler := NewTeamleadTradersHandler(cfg.TraderService)
	requisitesHandler := NewTeamleadRequisitesHandler(cfg.RequisiteService)
	traderShiftHandler := NewTraderShiftHandler(cfg.ShiftService)
	traderPayoutHandler := NewTraderPayoutHandler(cfg.PayoutService)
	importHandler := NewImportHandler(cfg.ImportService, cfg.ShiftService)
	orderReadHandler := NewOrderReadHandler(cfg.OrderReadService)
	readmodelHandler := NewTeamleadReadmodelHandler(cfg.ReadmodelService)
	traderReconciliationHandler := NewTraderReconciliationHandler(cfg.ReconcileService, cfg.ShiftService)
	teamleadReconciliationHandler := NewTeamleadReconciliationHandler(cfg.ReconcileService)
	loginRateLimiter := NewLoginRateLimiter(cfg.LoginRateLimitRequests, cfg.LoginRateLimitWindow)
	router.Route("/api/v1", func(r chi.Router) {
		r.Use(CSRFOriginGuard(cfg.CSRFAllowedOrigins))
		r.With(loginRateLimiter.Middleware).Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(cfg.AuthService, cfg.SessionCookieName))
			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me", authHandler.Me)
		})

		r.Route("/teamlead", func(r chi.Router) {
			r.Use(AuthMiddleware(cfg.AuthService, cfg.SessionCookieName))
			r.Use(RequireRole(users.RoleTeamlead))
			r.Get("/traders", tradersHandler.List)
			r.Post("/traders", tradersHandler.Create)
			r.Get("/traders/{traderId}", tradersHandler.Get)
			r.Patch("/traders/{traderId}", tradersHandler.Patch)
			r.Delete("/traders/{traderId}", tradersHandler.Delete)
			r.Post("/traders/{traderId}/reset-password", tradersHandler.ResetPassword)
			r.Get("/requisites", requisitesHandler.List)
			r.Post("/requisites", requisitesHandler.Create)
			r.Get("/requisites/{requisiteId}", requisitesHandler.Get)
			r.Patch("/requisites/{requisiteId}", requisitesHandler.Patch)
			r.Delete("/requisites/{requisiteId}", requisitesHandler.Delete)
			r.Post("/requisites/{requisiteId}/assign", requisitesHandler.Assign)
			r.Post("/requisites/{requisiteId}/unassign", requisitesHandler.Unassign)
			r.Get("/requisites/{requisiteId}/assignment-history", requisitesHandler.AssignmentHistory)
			r.Get("/inbound/dashboard", orderReadHandler.TeamleadInboundDashboard)
			r.Get("/inbound/orders", orderReadHandler.TeamleadInboundOrders)
			r.Post("/inbound/import", importHandler.TeamleadInbound)
			r.Get("/inbound/reconciliation/latest", teamleadReconciliationHandler.LatestInbound)
			r.Get("/outbound/dashboard", orderReadHandler.TeamleadOutboundDashboard)
			r.Get("/outbound/orders", orderReadHandler.TeamleadOutboundOrders)
			r.Post("/outbound/import", importHandler.TeamleadOutbound)
			r.Get("/periods", readmodelHandler.Periods)
			r.Get("/audit", readmodelHandler.Audit)
			r.Get("/periods/{periodId}/reconciliation/inbound", teamleadReconciliationHandler.PeriodInbound)
			r.Get("/periods/{periodId}/reconciliation/items", teamleadReconciliationHandler.PeriodItems)
			r.NotFound(func(w http.ResponseWriter, r *http.Request) {
				RespondError(w, NotFoundError())
			})
		})

		r.Route("/trader", func(r chi.Router) {
			r.Use(AuthMiddleware(cfg.AuthService, cfg.SessionCookieName))
			r.Use(RequireRole(users.RoleTrader))
			r.Get("/shift/current", traderShiftHandler.Current)
			r.Post("/shift/current/close", traderShiftHandler.CloseCurrent)
			r.Get("/shift/current/checklist", traderShiftHandler.CloseChecklist)
			r.Get("/shift/current/turnovers", traderShiftHandler.LatestTurnovers)
			r.Post("/shift/current/turnovers", traderShiftHandler.CreateTurnover)
			r.Get("/requisites", traderShiftHandler.AssignedRequisites)
			r.Post("/requisites/{requisiteId}/take", traderShiftHandler.TakeRequisite)
			r.Get("/shift-requisites", traderShiftHandler.ShiftRequisites)
			r.Patch("/shift-requisites/{shiftRequisiteId}", traderShiftHandler.UpdateShiftRequisite)
			r.Get("/shift-requisites/{shiftRequisiteId}/turnovers", traderShiftHandler.TurnoversByShiftRequisite)
			r.Get("/payouts", traderPayoutHandler.List)
			r.Post("/payouts", traderPayoutHandler.Create)
			r.Get("/payouts/{payoutId}", traderPayoutHandler.Get)
			r.Patch("/payouts/{payoutId}", traderPayoutHandler.Patch)
			r.Delete("/payouts/{payoutId}", traderPayoutHandler.Delete)
			r.Post("/payouts/{payoutId}/transfers", traderPayoutHandler.AddTransfer)
			r.Delete("/payouts/{payoutId}/transfers/{transferId}", traderPayoutHandler.DeleteTransfer)
			r.Get("/inbound/dashboard", orderReadHandler.TraderInboundDashboard)
			r.Get("/inbound/orders", orderReadHandler.TraderInboundOrders)
			r.Post("/inbound/import", importHandler.TraderInbound)
			r.Get("/inbound/reconciliation/latest", traderReconciliationHandler.LatestInbound)
			r.Post("/inbound/reconciliation/{runId}/accept", traderReconciliationHandler.AcceptInbound)
			r.Get("/outbound/dashboard", orderReadHandler.TraderOutboundDashboard)
			r.Get("/outbound/orders", orderReadHandler.TraderOutboundOrders)
			r.Post("/outbound/import", importHandler.TraderOutbound)
			r.Get("/outbound/reconciliation/latest", traderReconciliationHandler.LatestOutbound)
			r.Post("/outbound/reconciliation/{runId}/accept", traderReconciliationHandler.AcceptOutbound)
			r.NotFound(func(w http.ResponseWriter, r *http.Request) {
				RespondError(w, NotFoundError())
			})
		})
	})

	return router
}
