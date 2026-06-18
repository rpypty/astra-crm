import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { CalendarDays, Copy, Eye, FileText, History, Pencil, Plus, UserRound, X } from "lucide-react";
import { useCallback, useDeferredValue, useEffect, useMemo, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { Bar, CartesianGrid, Cell, ComposedChart, Line, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { z } from "zod";
import { DateTimeCell } from "@/shared/ui/date-time-cell";
import { EmptyState } from "@/shared/ui/empty-state";
import { FormField } from "@/shared/ui/form-field";
import { AcceptMismatchDialog, ImportCsvDialog, MismatchAlert } from "@/features/import-csv/ui/import-components";
import { MoneyCell } from "@/entities/order/ui/money-cell";
import { OrderDashboard } from "@/widgets/order-dashboard/ui/order-dashboard";
import { PageHeader } from "@/shared/ui/page-header";
import { PeriodFilterBar } from "@/features/period-filter/ui/period-filter-bar";
import { ReportReconciliationDetails } from "@/features/reconciliation/ui/report-reconciliation-details";
import { RequisiteCell } from "@/entities/requisite/ui/requisite-cell";
import { StatusBadge } from "@/entities/status/ui/status-badge";
import { UserCell } from "@/entities/user/ui/user-cell";
import { DataTable } from "@/shared/ui/data-table";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/shared/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { SearchableSelect, type SearchableSelectOption } from "@/shared/ui/searchable-select";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type {
  AccountingPeriod,
  AuditLogEntry,
  Bank,
  Order,
  OrderDirection,
  Requisite,
  RequisiteAssignmentEvent,
  RequisiteAssignmentWorkRow,
  RequisiteReport,
  RequisiteReportShift,
  ShiftReport,
  ShiftRequisite,
  Trader,
} from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { filterOrdersBySearch } from "@/shared/lib/order-filters";
import type { PeriodFilter } from "@/shared/lib/period-filter";
import { usePersistentPeriodFilter } from "@/shared/lib/period-filter";
import { queryKeys } from "@/shared/api/query-keys";
import {
  bpsToPercent,
  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  isValidRussianPhone,
  normalizeRussianPhone,
  parseMoneyToMinor,
  percentToBps,
} from "@/shared/lib/utils";

const traderSchema = z
  .object({
    id: z.number().optional(),
    login: z.string().min(1, "Введите логин"),
    password: z.string().optional(),
    externalWorkerName: z.string().min(1, "Введите external worker name"),
    salaryPercent: z.coerce.number().min(0, "Минимум 0").max(100, "Максимум 100"),
    status: z.enum(["active", "disabled"]),
  })
  .superRefine((values, context) => {
    if (!values.id && !values.password) {
      context.addIssue({ code: "custom", path: ["password"], message: "Пароль обязателен при создании" });
    }
  });

const requisiteSchema = z.object({
  id: z.number().optional(),
  phone: z.string().min(1, "Введите телефон").refine(isValidRussianPhone, "Введите телефон в формате +7 (XXX) XXX-XX-XX"),
  bankCode: z.string().min(1, "Выберите банк"),
  proxy: z.string().min(1, "Введите proxy"),
  employeeComment: z.string().optional(),
  assignedTraderId: z.string(),
  status: z.enum(["active", "archived"]),
});

const planSchema = z.object({
  assignmentId: z.number().optional(),
  requisiteId: z.string().min(1, "Выберите реквизит"),
  traderId: z.string().min(1, "Выберите трейдера"),
  assignedForDate: z.string().min(1, "Выберите дату"),
  targetTurnover: z.string().min(1, "Введите лимит").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
  comment: z.string().optional(),
});

type TraderForm = z.infer<typeof traderSchema>;
type RequisiteForm = z.infer<typeof requisiteSchema>;
type PlanForm = z.infer<typeof planSchema>;

type TeamleadRequisiteTab = "all" | "activity" | "planning";
type RequisiteReportTarget = {
  id: number;
  phone: string;
  bankName: string;
};

const TEAMLEAD_PERIOD_FILTER_STORAGE_KEY = "astra-crm:teamlead-period-filter";

export function TeamleadDashboardPage() {
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TEAMLEAD_PERIOD_FILTER_STORAGE_KEY);
  const inboundDashboardQuery = useQuery({
    queryKey: queryKeys.teamlead.dashboard("inbound", periodFilter),
    queryFn: () => api.orders.dashboard("teamlead", "inbound", periodFilter),
  });
  const outboundDashboardQuery = useQuery({
    queryKey: queryKeys.teamlead.dashboard("outbound", periodFilter),
    queryFn: () => api.orders.dashboard("teamlead", "outbound", periodFilter),
  });
  const blockedBalanceMinor =
    inboundDashboardQuery.data?.summary.blockedBalanceMinor ?? outboundDashboardQuery.data?.summary.blockedBalanceMinor ?? 0;
  const isBlockedBalanceLoading = inboundDashboardQuery.isLoading && outboundDashboardQuery.isLoading;

  return (
    <div className="space-y-6">
      <PageHeader title="Аналитика" description="Сводка по выбранному периоду, инвойсам и выплатам." />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <div className="grid gap-4 xl:grid-cols-2">
        <OrderDashboard
          title="Инвойсы"
          dashboard={inboundDashboardQuery.data}
          direction="inbound"
          isLoading={inboundDashboardQuery.isLoading}
          error={inboundDashboardQuery.error instanceof Error ? inboundDashboardQuery.error : null}
          showUnknownStatuses={false}
        />
        <OrderDashboard
          title="Выплаты"
          dashboard={outboundDashboardQuery.data}
          direction="outbound"
          isLoading={outboundDashboardQuery.isLoading}
          error={outboundDashboardQuery.error instanceof Error ? outboundDashboardQuery.error : null}
          showUnknownStatuses={false}
        />
      </div>
      <ProfitLossPlaceholder
        periodFilter={periodFilter}
        blockedBalanceMinor={blockedBalanceMinor}
        isBlockedBalanceLoading={isBlockedBalanceLoading}
      />
    </div>
  );
}

function ProfitLossPlaceholder({
  periodFilter,
  blockedBalanceMinor,
  isBlockedBalanceLoading,
}: {
  periodFilter: PeriodFilter;
  blockedBalanceMinor: number;
  isBlockedBalanceLoading: boolean;
}) {
  const data = useMemo(() => buildProfitLossPlaceholder(periodFilter), [periodFilter]);
  const totalMinor = data.length > 0 ? data[data.length - 1].cumulativeMinor : 0;
  const isProfit = totalMinor >= 0;
  const profitableDays = data.filter((item) => item.amountMinor > 0).length;
  const lossDays = data.filter((item) => item.amountMinor < 0).length;
  const bestDay = data.reduce((max, item) => Math.max(max, item.amountMinor), 0);
  const worstDay = data.reduce((min, item) => Math.min(min, item.amountMinor), 0);

  return (
    <Card className="overflow-hidden">
      <CardContent className="space-y-5 p-4">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <div className="text-sm font-medium text-muted-foreground">P&L</div>
              <span className="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700">заглушка</span>
            </div>
            <div className={isProfit ? "mt-2 text-3xl font-semibold text-emerald-700" : "mt-2 text-3xl font-semibold text-red-700"}>
              {formatMoneyMinor(totalMinor)}
            </div>
            <div className="mt-1 text-sm text-muted-foreground">Прибыль/убыток за выбранный период</div>
          </div>
          <div className="grid w-full gap-2 sm:grid-cols-2 xl:w-auto xl:grid-cols-4">
            <ProfitLossMetric label="Прибыльных дней" value={String(profitableDays)} />
            <ProfitLossMetric label="Лучший день" value={formatMoneyMinor(bestDay)} tone="positive" />
            <ProfitLossMetric label="Худший день" value={formatMoneyMinor(worstDay)} tone="negative" />
            <ProfitLossMetric
              label="Заблокированный остаток"
              value={isBlockedBalanceLoading ? "Загрузка" : formatMoneyMinor(blockedBalanceMinor)}
            />
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-sm bg-emerald-600" />
            Дневная прибыль
          </span>
          <span className="inline-flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-sm bg-red-500" />
            Дневной убыток
          </span>
          <span className="inline-flex items-center gap-2">
            <span className="h-0.5 w-5 rounded-full bg-amber-500" />
            Накопленный P&L
          </span>
          <span>Убыточных дней: {lossDays}</span>
        </div>
        <div className="h-80 min-w-0 rounded-md border bg-white p-3">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={data} margin={{ top: 8, right: 14, left: 0, bottom: 0 }}>
              <CartesianGrid stroke="#e5e7eb" strokeDasharray="3 3" vertical={false} />
              <ReferenceLine y={0} stroke="#94a3b8" strokeWidth={1.5} />
              <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} tick={{ fill: "#64748b", fontSize: 12 }} />
              <YAxis
                tickLine={false}
                axisLine={false}
                width={88}
                tick={{ fill: "#64748b", fontSize: 12 }}
                tickFormatter={(value) => formatMoneyAxis(Number(value))}
              />
              <Tooltip
                cursor={{ fill: "rgba(148, 163, 184, 0.14)" }}
                contentStyle={{ borderRadius: 8, borderColor: "#dbe3ee", boxShadow: "0 10px 30px rgba(15, 23, 42, 0.12)" }}
                formatter={(value, name) => [
                  formatMoneyMinor(Number(value)),
                  name === "amountMinor" ? "Дневной P&L" : "Накопленный P&L",
                ]}
                labelFormatter={(label) => `Дата: ${label}`}
              />
              <Bar dataKey="amountMinor" barSize={18} radius={[3, 3, 0, 0]}>
                {data.map((item) => (
                  <Cell key={item.date} fill={item.amountMinor >= 0 ? "#059669" : "#dc2626"} />
                ))}
              </Bar>
              <Line
                type="monotone"
                dataKey="cumulativeMinor"
                stroke="#f59e0b"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4, stroke: "#f59e0b", strokeWidth: 2, fill: "#ffffff" }}
              />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}

function ProfitLossMetric({
  label,
  value,
  tone = "neutral",
}: {
  label: string;
  value: string;
  tone?: "neutral" | "positive" | "negative";
}) {
  const valueClass =
    tone === "positive" ? "text-emerald-700" : tone === "negative" ? "text-red-700" : "text-foreground";

  return (
    <div className="rounded-md border bg-muted/20 px-3 py-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className={`mt-1 whitespace-nowrap text-sm font-semibold tabular-nums ${valueClass}`}>{value}</div>
    </div>
  );
}

function buildProfitLossPlaceholder(periodFilter: PeriodFilter) {
  const today = startOfLocalDay(new Date());
  const end = parseISODate(periodFilter.dateTo) ?? today;
  const start = parseISODate(periodFilter.dateFrom) ?? addDays(end, -13);
  const normalizedStart = start > end ? end : start;
  const daysCount = Math.min(diffDays(normalizedStart, end) + 1, 45);
  const firstDay = addDays(end, -(daysCount - 1));
  let cumulativeMinor = 0;

  return Array.from({ length: daysCount }, (_, index) => {
    const date = addDays(firstDay, index);
    const direction = index % 6 === 1 || index % 6 === 4 ? -1 : 1;
    const baseMinor = (180_000 + ((index * 37) % 420_000)) * 100;
    const amountMinor = direction * baseMinor;
    cumulativeMinor += amountMinor;

    return {
      date: toISODate(date),
      label: new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit" }).format(date),
      amountMinor,
      cumulativeMinor,
    };
  });
}

function formatMoneyAxis(value: number) {
  const absolute = Math.abs(value);
  const sign = value < 0 ? "-" : "";

  if (absolute >= 100_000_000) {
    return `${sign}${(absolute / 100_000_000).toFixed(1).replace(".", ",")} млн ₽`;
  }

  if (absolute >= 100_000) {
    return `${sign}${Math.round(absolute / 100_000)} тыс ₽`;
  }

  return formatMoneyMinor(value).replace(",00", "");
}

function parseISODate(value?: string) {
  if (!value) return undefined;
  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day) return undefined;
  const date = new Date(year, month - 1, day);
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) return undefined;
  return startOfLocalDay(date);
}

function startOfLocalDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function addDays(date: Date, days: number) {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

function diffDays(start: Date, end: Date) {
  return Math.max(0, Math.round((end.getTime() - start.getTime()) / 86_400_000));
}

function toISODate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
