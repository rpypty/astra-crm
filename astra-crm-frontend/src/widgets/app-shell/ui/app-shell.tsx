import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import { BarChart3, ClipboardList, CreditCard, Landmark, ReceiptText, Users } from "lucide-react";
import { useAuth } from "@/app/auth";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type Role = "teamlead" | "trader";

const teamleadNavigation = [
  { label: "Аналитика", to: "/teamlead/dashboard", icon: BarChart3 },
  { label: "Реквизиты", to: "/teamlead/requisites", icon: CreditCard },
  { label: "Сотрудники", to: "/teamlead/traders", icon: Users },
  { label: "Транзакции", to: "/teamlead/transactions", icon: ReceiptText },
  { label: "Сверка", to: "/teamlead/periods", icon: ClipboardList },
  { label: "Отчеты", to: "/teamlead/reports", icon: ClipboardList },
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
  const navigation = role === "teamlead" ? teamleadNavigation : traderNavigation;
  const roleLabel = role === "teamlead" ? "TEAMLEAD" : "TRADER";

  return (
    <div className="min-h-screen bg-background">
      <aside className="fixed inset-y-0 left-0 z-20 flex w-60 flex-col border-r border-border bg-card">
        <div className="flex h-16 items-center border-b border-border px-5">
          <div>
            <div className="text-sm font-semibold uppercase tracking-normal text-primary">Astra CRM</div>
            <div className="text-xs text-muted-foreground">P2P operations</div>
          </div>
        </div>
        <nav className="flex-1 space-y-1 p-3">
          {navigation.map((item) => {
            const Icon = item.icon;
            const active = location.pathname === item.to;

            return (
              <Link key={item.to} to={item.to} className="block">
                <span
                  className={cn(
                    "flex h-9 items-center gap-2 rounded-md px-3 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-foreground",
                    active && "bg-accent text-foreground",
                  )}
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </span>
              </Link>
            );
          })}
        </nav>
      </aside>

      <div className="pl-60">
        <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b border-border bg-card/95 px-6 backdrop-blur">
          <div className="flex items-center gap-3 text-sm">
            <span className="rounded-md border border-border bg-background px-2 py-1 font-medium">{roleLabel}</span>
            <span className="text-muted-foreground">Рабочий период: текущий</span>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={() => void auth.logout()}>
            {auth.user?.login ?? "Пользователь"} · Выйти
          </Button>
        </header>
        <main className="p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
