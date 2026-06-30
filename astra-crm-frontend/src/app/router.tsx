import { Navigate, Outlet, createRootRoute, createRoute, createRouter } from "@tanstack/react-router";
import { Suspense, lazy, type ReactNode } from "react";
import { useAuth } from "@/features/auth/model/auth";
import { AppShell } from "@/widgets/app-shell/ui/app-shell";
import { LoadingSkeleton } from "@/shared/ui/loading-skeleton";

const LoginPage = lazy(() => import("@/pages/login").then((module) => ({ default: module.LoginPage })));
const TeamleadAuditPage = lazy(() => import("@/pages/teamlead/audit/page").then((module) => ({ default: module.TeamleadAuditPage })));
const TeamleadBanksPage = lazy(() => import("@/pages/teamlead/banks/page").then((module) => ({ default: module.TeamleadBanksPage })));
const TeamleadDashboardPage = lazy(() =>
  import("@/pages/teamlead/dashboard/page").then((module) => ({ default: module.TeamleadDashboardPage })),
);
const TeamleadDebugPage = lazy(() => import("@/pages/teamlead/debug/page").then((module) => ({ default: module.TeamleadDebugPage })));
const TeamleadPeriodsPage = lazy(() =>
  import("@/pages/teamlead/accounting-periods/page").then((module) => ({ default: module.TeamleadPeriodsPage })),
);
const TeamleadReportsPage = lazy(() =>
  import("@/pages/teamlead/reports/page").then((module) => ({ default: module.TeamleadReportsPage })),
);
const TeamleadRequisitesPage = lazy(() =>
  import("@/pages/teamlead/requisites/page").then((module) => ({ default: module.TeamleadRequisitesPage })),
);
const TeamleadTransactionsPage = lazy(() =>
  import("@/pages/teamlead/transactions/page").then((module) => ({ default: module.TeamleadTransactionsPage })),
);
const TeamleadTradersPage = lazy(() =>
  import("@/pages/teamlead/employees/page").then((module) => ({ default: module.TeamleadTradersPage })),
);
const TraderAnalyticsPage = lazy(() =>
  import("@/pages/trader/analytics/page").then((module) => ({ default: module.TraderAnalyticsPage })),
);
const TraderPayoutsPage = lazy(() => import("@/pages/trader/payouts/page").then((module) => ({ default: module.TraderPayoutsPage })));
const TraderReportsPage = lazy(() => import("@/pages/trader/reports/page").then((module) => ({ default: module.TraderReportsPage })));
const TraderRequisitesPage = lazy(() =>
  import("@/pages/trader/requisites/page").then((module) => ({ default: module.TraderRequisitesPage })),
);
const TraderTransactionsPage = lazy(() =>
  import("@/pages/trader/transactions/page").then((module) => ({ default: module.TraderTransactionsPage })),
);

const rootRoute = createRootRoute({
  component: () => (
    <Suspense fallback={<FullPageLoading />}>
      <Outlet />
    </Suspense>
  ),
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

const teamleadBanksRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "banks",
  component: TeamleadBanksPage,
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

const teamleadDebugRoute = createRoute({
  getParentRoute: () => teamleadRoute,
  path: "debug",
  component: TeamleadDebugPage,
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
    teamleadBanksRoute,
    teamleadTransactionsRoute,
    teamleadInboundRoute,
    teamleadOutboundRoute,
    teamleadPeriodsRoute,
    teamleadReportsRoute,
    teamleadAuditRoute,
    teamleadDebugRoute,
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
