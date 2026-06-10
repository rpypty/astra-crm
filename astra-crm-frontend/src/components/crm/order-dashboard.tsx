import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { EmptyState } from "@/components/crm/empty-state";
import { StatusBadge } from "@/components/crm/status-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { OrderDashboard as OrderDashboardData, OrderDirection } from "@/lib/domain";
import { formatDateTime, formatMoneyMinor } from "@/lib/utils";

type OrderDashboardProps = {
  dashboard?: OrderDashboardData;
  direction: OrderDirection;
  title?: string;
  isLoading?: boolean;
  error?: Error | null;
};

const statusLabels: Record<string, string> = {
  success: "Успех",
  corrected: "Исправлен",
  failed: "Ошибка",
  cancelled: "Отменен",
  unknown: "Неизвестно",
};

export function OrderDashboard({ dashboard, direction, title, isLoading, error }: OrderDashboardProps) {
  if (isLoading) {
    return <EmptyState title="Загружаем показатели" />;
  }

  if (error) {
    return <EmptyState title="Не удалось загрузить показатели" description={error.message} />;
  }

  if (!dashboard) {
    return <EmptyState title="Нет данных для dashboard" />;
  }

  const summary = dashboard.summary;
  const hasOrders = summary.totalCount > 0;
  const conversion = summary.totalCount > 0 ? (summary.successCount / summary.totalCount) * 100 : 0;
  const chartData = dashboard.statusBreakdown.map((item) => ({
    name: statusLabels[item.normalizedStatus] ?? item.rawStatus,
    amount: item.amountMinor,
    count: item.count,
    rawStatus: item.rawStatus,
  }));

  return (
    <div className="space-y-4">
      {title ? <h2 className="text-lg font-semibold">{title}</h2> : null}
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Всего" value={formatMoneyMinor(summary.totalAmountMinor)} detail={`${summary.totalCount} ордеров`} />
        <MetricCard
          label="Успешный оборот"
          value={formatMoneyMinor(summary.successAmountMinor)}
          detail={`${summary.successCount} ордеров`}
        />
        <MetricCard
          label={direction === "inbound" ? "Неуспешный оборот" : "Проблемные выплаты"}
          value={formatMoneyMinor(summary.failedAmountMinor + summary.unknownAmountMinor)}
          detail={`${summary.failedCount + summary.unknownCount} ордеров`}
          warning={summary.failedCount + summary.unknownCount > 0}
        />
        <MetricCard label="Конверсия" value={`${conversion.toFixed(1)}%`} detail="success + corrected" />
      </div>

      {dashboard.unknownStatuses.length ? (
        <Card className="border-amber-200 bg-amber-50">
          <CardContent className="p-4 text-sm text-amber-950">
            Неизвестные CSV-статусы: {dashboard.unknownStatuses.join(", ")}
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Оборот по статусам</CardTitle>
        </CardHeader>
        <CardContent>
          {hasOrders && chartData.length > 0 ? (
            <div className="h-72">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} margin={{ top: 8, right: 16, left: 8, bottom: 8 }}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="name" tickLine={false} axisLine={false} />
                  <YAxis
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(value) => formatMoneyMinor(Number(value)).replace(",00", "")}
                    width={96}
                  />
                  <Tooltip
                    formatter={(value) => [formatMoneyMinor(Number(value)), "Оборот"]}
                    labelFormatter={(_, payload) => payload?.[0]?.payload?.rawStatus ?? ""}
                  />
                  <Legend />
                  <Bar dataKey="amount" fill="#2563eb" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="flex h-72 flex-col items-center justify-center gap-1 text-center">
              <div className="text-base font-semibold">Нет данных для графика</div>
              <div className="text-sm text-muted-foreground">График появится после CSV-импорта в этот scope.</div>
            </div>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Последние импорты</CardTitle>
        </CardHeader>
        <CardContent>
          {dashboard.recentImports.length ? (
            <div className="space-y-3">
              {dashboard.recentImports.slice(0, 5).map((item) => (
                <div key={item.id} className="flex items-start justify-between gap-3 border-b border-border pb-3 last:border-0 last:pb-0">
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium">{item.fileName}</div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      {formatDateTime(item.appliedAt ?? item.createdAt)} · {item.rowsCount} строк
                    </div>
                  </div>
                  <StatusBadge status={item.status} />
                </div>
              ))}
            </div>
          ) : (
            <div className="flex min-h-48 flex-col items-center justify-center gap-1 text-center">
              <div className="text-base font-semibold">Импортов пока нет</div>
              <div className="text-sm text-muted-foreground">Загрузите CSV, чтобы увидеть историю.</div>
            </div>
          )}
        </CardContent>
      </Card>
      </div>
    </div>
  );
}

function MetricCard({ label, value, detail, warning }: { label: string; value: string; detail: string; warning?: boolean }) {
  return (
    <Card className={warning ? "border-amber-200 bg-amber-50" : undefined}>
      <CardHeader>
        <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-semibold">{value}</div>
        <div className="mt-1 text-sm text-muted-foreground">{detail}</div>
      </CardContent>
    </Card>
  );
}
