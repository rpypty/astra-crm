import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import {
  BarChart3,
  ChevronLeft,
  ChevronRight,
  ClipboardList,
  CreditCard,
  FileSpreadsheet,
  Landmark,
  LogOut,
  ReceiptText,
  Users,
} from "lucide-react";
import { useState } from "react";
import { useAuth } from "@/features/auth/model/auth";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/lib/utils";

type Role = "teamlead" | "trader";

const teamleadNavigation = [
  { label: "Аналитика", to: "/teamlead/dashboard", icon: BarChart3 },
  { label: "Реквизиты", to: "/teamlead/requisites", icon: CreditCard },
  { label: "Сотрудники", to: "/teamlead/traders", icon: Users },
  { label: "Банки", to: "/teamlead/banks", icon: Landmark },
  { label: "Транзакции", to: "/teamlead/transactions", icon: ReceiptText },
  { label: "Сверка", to: "/teamlead/periods", icon: ClipboardList },
  { label: "Отчеты", to: "/teamlead/reports", icon: ClipboardList },
  { label: "Импорт отчета Fin_ALL", to: "/teamlead/debug", icon: FileSpreadsheet },
] as const;

const traderNavigation = [
  { label: "Аналитика", to: "/trader/analytics", icon: BarChart3 },
  { label: "Мои реквизиты", to: "/trader/requisites", icon: CreditCard },
  { label: "Транзакции", to: "/trader/transactions", icon: ReceiptText },
  { label: "Ручные выплаты", to: "/trader/payouts", icon: Landmark },
  { label: "Отчеты", to: "/trader/reports", icon: ClipboardList },
] as const;

export function AppShell({ role }: { role: Role }) {
  const auth = useAuth();
  const location = useRouterState({ select: (state) => state.location });
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const navigation = role === "teamlead" ? teamleadNavigation : traderNavigation;
  const roleLabel = role === "teamlead" ? "TEAMLEAD" : "TRADER";

  return (
    <div className="min-h-screen bg-background">
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-20 flex flex-col border-r border-border bg-card transition-[width] duration-200",
          sidebarCollapsed ? "w-20" : "w-60",
        )}
      >
        <div
          className={cn(
            "flex h-16 items-center border-b border-border px-4",
            sidebarCollapsed ? "justify-center" : "justify-between",
          )}
        >
          <div className={cn("min-w-0", sidebarCollapsed && "hidden")}>
            <div className="text-sm font-semibold uppercase tracking-normal text-primary">Astra CRM</div>
            <div className="text-xs text-muted-foreground">P2P operations</div>
          </div>
          <div
            className={cn(
              "hidden h-9 w-9 items-center justify-center rounded-md border border-primary/20 bg-primary/10 text-sm font-semibold text-primary",
              sidebarCollapsed && "flex",
            )}
          >
            A
          </div>
        </div>
        <nav className={cn("flex-1 space-y-1 p-3", sidebarCollapsed && "px-2")}>
          {navigation.map((item) => {
            const Icon = item.icon;
            const active = location.pathname === item.to;

            return (
              <Link key={item.to} to={item.to} className="block">
                <span
                  className={cn(
                    "flex h-9 items-center gap-2 rounded-md text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
                    sidebarCollapsed ? "justify-center px-0" : "px-3",
                    active && "bg-accent text-foreground shadow-sm",
                  )}
                  title={sidebarCollapsed ? item.label : undefined}
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  <span className={cn("truncate", sidebarCollapsed && "sr-only")}>{item.label}</span>
                </span>
              </Link>
            );
          })}
        </nav>
        <div className={cn("border-t border-border p-3", sidebarCollapsed && "px-2")}>
          <Button
            type="button"
            variant="ghost"
            size={sidebarCollapsed ? "icon" : "default"}
            className={cn("w-full justify-start text-muted-foreground", sidebarCollapsed && "justify-center")}
            onClick={() => setSidebarCollapsed((value) => !value)}
            aria-label={sidebarCollapsed ? "Развернуть сайдбар" : "Свернуть сайдбар"}
            title={sidebarCollapsed ? "Развернуть сайдбар" : "Свернуть сайдбар"}
          >
            {sidebarCollapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
            {!sidebarCollapsed && <span>Свернуть</span>}
          </Button>
        </div>
      </aside>

      <div className={cn("transition-[padding] duration-200", sidebarCollapsed ? "pl-20" : "pl-60")}>
        <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b border-border bg-card/95 px-8 shadow-sm shadow-slate-200/40 backdrop-blur">
          <div className="flex min-w-0 items-center gap-4 text-sm">
            <span className="rounded-md border border-border bg-background px-3 py-1.5 text-sm font-semibold shadow-sm">
              {roleLabel}
            </span>
            <span className="truncate text-sm text-muted-foreground">Рабочий период: текущий</span>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-9 shrink-0 px-3 text-sm font-semibold shadow-sm"
            onClick={() => void auth.logout()}
          >
            <span>{auth.user?.login ?? "Пользователь"}</span>
            <span className="text-muted-foreground">·</span>
            <LogOut className="h-4 w-4" />
            <span>Выйти</span>
          </Button>
        </header>
        <main className="p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
