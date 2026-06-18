import {
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
} from "recharts";
import type { ReactNode } from "react";
import { EmptyState } from "@/shared/ui/empty-state";
import { Card, CardContent } from "@/shared/ui/card";
import type { OrderDashboard as OrderDashboardData, OrderDirection } from "@/shared/model/domain";
import { formatMoneyMinor } from "@/shared/lib/utils";

type OrderDashboardProps = {
  dashboard?: OrderDashboardData;
  direction: OrderDirection;
  title?: string;
  isLoading?: boolean;
  error?: Error | null;
  showUnknownStatuses?: boolean;
};

const statusLabels: Record<string, string> = {
  success: "Успех",
  corrected: "Исправлен",
  failed: "Ошибка",
  cancelled: "Отменен",
  unknown: "Неизвестно",
};

const statusColors: Record<string, string> = {
  success: "#059669",
  corrected: "#2563eb",
  failed: "#dc2626",
  cancelled: "#64748b",
  unknown: "#d97706",
};

export function OrderDashboard({ dashboard, direction, title, isLoading, error, showUnknownStatuses = true }: OrderDashboardProps) {
  if (isLoading) {
    return <EmptyState title="Загружаем показатели" />;
  }

  if (error) {
    return <EmptyState title="Не удалось загрузить показатели" description={error.message} />;
  }

  if (!dashboard) {
    return <EmptyState title="Нет данных для аналитики" />;
  }

  const summary = dashboard.summary;
  const statusBreakdown = dashboard.statusBreakdown ?? [];
  const unknownStatuses = dashboard.unknownStatuses ?? [];
  const problemAmountMinor = summary.failedAmountMinor + summary.unknownAmountMinor;
  const problemCount = summary.failedCount + summary.unknownCount;
  const conversion = summary.totalCount > 0 ? (summary.successCount / summary.totalCount) * 100 : 0;
  const conversionChartData = [
    {
      name: "Успешно",
      count: summary.successCount,
      color: statusColors.success,
    },
    {
      name: direction === "inbound" ? "Неуспешно" : "Проблемно",
      count: problemCount,
      color: statusColors.failed,
    },
    {
      name: "Прочее",
      count: Math.max(summary.totalCount - summary.successCount - problemCount, 0),
      color: "#64748b",
    },
  ].filter((item) => item.count > 0);
  const statusSummary = buildStatusSummary(statusBreakdown);

  return (
    <Card>
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            {title ? <h2 className="text-lg font-semibold">{title}</h2> : null}
            <div className="mt-1 text-sm text-muted-foreground">
              {summary.totalCount} транзакций · {formatMoneyMinor(summary.totalAmountMinor)}
            </div>
          </div>
          {statusSummary ? <div className="max-w-md text-right text-xs text-muted-foreground">{statusSummary}</div> : null}
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <MetricCard
            label="Всего"
            value={formatMoneyMinor(summary.totalAmountMinor)}
            detail={`${summary.totalCount} транзакций`}
          />
          <MetricCard
            label="Конверсия"
            value={`${conversion.toFixed(1)}%`}
            chart={conversionChartData.length ? <ConversionChart data={conversionChartData} /> : null}
          />
          <MetricCard
            label="Успешный оборот"
            value={formatMoneyMinor(summary.successAmountMinor)}
            detail={`${summary.successCount} транзакций`}
          />
          <MetricCard
            label={direction === "inbound" ? "Неуспешный оборот" : "Проблемные выплаты"}
            value={formatMoneyMinor(problemAmountMinor)}
            detail={`${problemCount} транзакций`}
          />
        </div>

        {showUnknownStatuses && unknownStatuses.length ? (
          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-950">
            Неизвестные CSV-статусы: {unknownStatuses.join(", ")}
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function buildStatusSummary(items: OrderDashboardData["statusBreakdown"]) {
  const grouped = new Map<string, number>();
  items.forEach((item) => {
    const label = statusLabels[item.rawStatus] ?? statusLabels[item.normalizedStatus] ?? item.rawStatus;
    grouped.set(label, (grouped.get(label) ?? 0) + item.count);
  });

  return Array.from(grouped.entries())
    .map(([label, count]) => `${label}: ${count}`)
    .join(" · ");
}

function MetricCard({
  label,
  value,
  detail,
  chart,
}: {
  label: string;
  value: string;
  detail?: string;
  chart?: ReactNode;
}) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="flex min-h-24 items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
          <div className="mt-2 break-words text-xl font-semibold tabular-nums">{value}</div>
          {detail ? <div className="mt-1 text-sm text-muted-foreground">{detail}</div> : null}
        </div>
        {chart ? <div className="h-20 w-20 shrink-0">{chart}</div> : null}
      </div>
    </div>
  );
}

function ConversionChart({ data }: { data: Array<{ name: string; count: number; color: string }> }) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <PieChart margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
        <Pie
          data={data}
          dataKey="count"
          nameKey="name"
          innerRadius="58%"
          outerRadius="88%"
          paddingAngle={2}
          stroke="#ffffff"
          strokeWidth={2}
          isAnimationActive={false}
        >
          {data.map((item) => (
            <Cell key={item.name} fill={item.color} />
          ))}
        </Pie>
        <Tooltip formatter={(value) => [Number(value), "Транзакций"]} />
      </PieChart>
    </ResponsiveContainer>
  );
}
