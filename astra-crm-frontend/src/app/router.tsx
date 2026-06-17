import { Navigate, Outlet, createRootRoute, createRoute, createRouter } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { useAuth } from "@/features/auth/model/auth";
import { AppShell } from "@/widgets/app-shell/ui/app-shell";
import { LoadingSkeleton } from "@/shared/ui/loading-skeleton";
import { LoginPage } from "@/pages/login";
import {
  TeamleadAuditPage,
  TeamleadDashboardPage,
  TeamleadPeriodsPage,
  TeamleadReportsPage,
  TeamleadRequisitesPage,
  TeamleadTransactionsPage,
  TeamleadTradersPage,
} from "@/pages/teamlead";
import {
  TraderAnalyticsPage,
  TraderPayoutsPage,
  TraderReportsPage,
  TraderRequisitesPage,
  TraderTransactionsPage,
} from "@/pages/trader";

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: IndexRedirect,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "login",
  component: LoginRedirect,
});

const teamleadRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "teamlead",
  component: () => (
    <ProtectedRole role="teamlead">
      <AppShell role="teamlead" />
    </ProtectedRole>
  ),
});

const teamleadDashboardRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "dashboard",
  component: TeamleadDashboardPage,
});

const teamleadRequisitesRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "requisites",
  component: TeamleadRequisitesPage,
});

const teamleadTradersRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "traders",
  component: TeamleadTradersPage,
});

const teamleadInboundRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "inbound",
  component: () => <TeamleadTransactionsPage initialDirection="inbound" />,
});

const teamleadOutboundRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "outbound",
  component: () => <TeamleadTransactionsPage initialDirection="outbound" />,
});

const teamleadTransactionsRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "transactions",
  component: TeamleadTransactionsPage,
});

const teamleadPeriodsRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "periods",
  component: TeamleadPeriodsPage,
});

const teamleadReportsRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "reports",
  component: TeamleadReportsPage,
});

const teamleadAuditRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "audit",
  component: TeamleadAuditPage,
});

const traderRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "trader",
  component: () => (
    <ProtectedRole role="trader">
      <AppShell role="trader" />
    </ProtectedRole>
  ),
});

const traderRequisitesRoute = createRoute({
  getParentRoute: () => traderRoute,
  path: "requisites",
  component: TraderRequisitesPage,
});

const traderInboundRoute = createRoute({
  getParentRoute: () => traderRoute,
  path: "inbound",
  component: () => <TraderTransactionsPage initialDirection="inbound" />,
});

const traderOutboundRoute = createRoute({
  getParentRoute: () => traderRoute,
  path: "outbound",
  component: () => <TraderTransactionsPage initialDirection="outbound" />,
});

const traderTransactionsRoute = createRoute({
  getParentRoute: () => traderRoute,
  path: "transactions",
  component: TraderTransactionsPage,
});

const traderReportsRoute = createRoute({
  getParentRoute: () => traderRoute,
  path: "reports",
  component: TraderReportsPage,
});

const traderPayoutsRoute = createRoute({
  getParentRoute: () => traderRoute,
  path: "payouts",
  component: TraderPayoutsPage,
});

const traderAnalyticsRoute = createRoute({
  getParentRoute: () => traderRoute,
  path: "analytics",
  component: TraderAnalyticsPage,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  teamleadRoute.addChildren([
    teamleadDashboardRoute,
    teamleadRequisitesRoute,
    teamleadTradersRoute,
    teamleadTransactionsRoute,
    teamleadInboundRoute,
    teamleadOutboundRoute,
    teamleadPeriodsRoute,
    teamleadReportsRoute,
    teamleadAuditRoute,
  ]),
  traderRoute.addChildren([
    traderRequisitesRoute,
    traderTransactionsRoute,
    traderInboundRoute,
    traderOutboundRoute,
    traderReportsRoute,
    traderPayoutsRoute,
    traderAnalyticsRoute,
  ]),
]);

export const router = createRouter({
  routeTree,
  defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

function IndexRedirect() {
  const auth = useAuth();
  if (auth.isLoading) return <FullPageLoading />;
  if (!auth.user) return <Navigate to="/login" />;
  return <Navigate to={auth.user.role === "teamlead" ? "/teamlead/dashboard" : "/trader/analytics"} />;
}

function LoginRedirect() {
  const auth = useAuth();
  if (auth.isLoading) return <FullPageLoading />;
  if (auth.user) return <Navigate to={auth.user.role === "teamlead" ? "/teamlead/dashboard" : "/trader/analytics"} />;
  return <LoginPage />;
}

function ProtectedRole({ role, children }: { role: "teamlead" | "trader"; children: ReactNode }) {
  const auth = useAuth();
  if (auth.isLoading) return <FullPageLoading />;
  if (!auth.user) return <Navigate to="/login" />;
  if (auth.user.role !== role) {
    return <Navigate to={auth.user.role === "teamlead" ? "/teamlead/dashboard" : "/trader/analytics"} />;
  }
  return children;
}

function FullPageLoading() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <div className="w-full max-w-lg">
        <LoadingSkeleton rows={4} />
      </div>
    </div>
  );
}
