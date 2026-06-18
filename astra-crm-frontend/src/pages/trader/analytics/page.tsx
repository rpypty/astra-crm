import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertTriangle, CalendarDays, CheckCircle2, Eye, FileText, History, Plus, RefreshCw, Upload } from "lucide-react";
import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AcceptMismatchDialog, ImportCsvDialog, MismatchAlert } from "@/features/import-csv/ui/import-components";
import { ConfirmDialog } from "@/shared/ui/confirm-dialog";
import { DateTimeCell } from "@/shared/ui/date-time-cell";
import { EmptyState } from "@/shared/ui/empty-state";
import { FormField } from "@/shared/ui/form-field";
import { MoneyCell } from "@/entities/order/ui/money-cell";
import { OrderDashboard } from "@/widgets/order-dashboard/ui/order-dashboard";
import { PageHeader } from "@/shared/ui/page-header";
import { PeriodFilterBar } from "@/features/period-filter/ui/period-filter-bar";
import { ReportReconciliationDetails } from "@/features/reconciliation/ui/report-reconciliation-details";
import { RequisiteCell } from "@/entities/requisite/ui/requisite-cell";
import { StatusBadge } from "@/entities/status/ui/status-badge";
import { DataTable } from "@/shared/ui/data-table";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { SearchableSelect, type SearchableSelectOption } from "@/shared/ui/searchable-select";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type { OrderDirection, Payout, PayoutTransfer, ReconciliationItem, ReconciliationSummary, ShiftReport, ShiftRequisite } from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { filterOrdersBySearch } from "@/shared/lib/order-filters";
import type { PeriodFilter } from "@/shared/lib/period-filter";
import { usePersistentPeriodFilter } from "@/shared/lib/period-filter";
import { queryKeys } from "@/shared/api/query-keys";
import {

  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  normalizeCardNumber,
  parseMoneyToMinor,
} from "@/shared/lib/utils";
import { MetricCard } from "@/shared/ui/metric-card";

const takeSchema = z.object({
  cardNumber: z.string().min(8, "Введите номер карты"),
  holderName: z.string().min(1, "Введите держателя"),
});

const closeRequisiteSchema = z.object({
  inboundTurnover: z.string().min(1, "Введите оборот по оплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  outboundTurnover: z.string().min(1, "Введите оборот по выплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  closingBalance: z.string().min(1, "Введите остаток").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  blocked: z.boolean(),
  comment: z.string().optional(),
});

const payoutSchema = z.object({
  destinationBank: z.string().min(1, "Введите банк"),
  destinationRequisite: z.string().min(1, "Введите реквизит получателя"),
  amount: z.string().min(1, "Введите сумму").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
});

const transferSchema = z.object({
  sourceShiftRequisiteId: z.coerce.number().min(1, "Выберите источник"),
  amount: z.string().min(1, "Введите сумму").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
  comment: z.string().optional(),
});

const TRADER_PERIOD_FILTER_STORAGE_KEY = "astra-crm:trader-period-filter";
type TraderRequisiteTab = "current" | "future" | "history";

export function TraderAnalyticsPage() {
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TRADER_PERIOD_FILTER_STORAGE_KEY);
  const confirmedPeriodFilter = useMemo(() => ({ ...periodFilter, confirmedOnly: true }), [periodFilter]);
  const profileQuery = useQuery({
    queryKey: queryKeys.trader.profile(periodFilter),
    queryFn: () => api.traderProfile.get(periodFilter),
  });
  const inboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("inbound", confirmedPeriodFilter),
    queryFn: () => api.orders.dashboard("trader", "inbound", confirmedPeriodFilter),
  });
  const outboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("outbound", confirmedPeriodFilter),
    queryFn: () => api.orders.dashboard("trader", "outbound", confirmedPeriodFilter),
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Аналитика" description="Показатели текущей и прошлых смен." />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <TraderSalaryAnalytics
        profile={profileQuery.data}
        isLoading={profileQuery.isLoading}
        error={profileQuery.error instanceof Error ? profileQuery.error.message : null}
      />
      <div className="grid gap-4 xl:grid-cols-2">
        <OrderDashboard
          title="Инвойсы"
          dashboard={inboundDashboardQuery.data}
          direction="inbound"
          isLoading={inboundDashboardQuery.isLoading}
          error={inboundDashboardQuery.error instanceof Error ? inboundDashboardQuery.error : null}
        />
        <OrderDashboard
          title="Выплаты"
          dashboard={outboundDashboardQuery.data}
          direction="outbound"
          isLoading={outboundDashboardQuery.isLoading}
          error={outboundDashboardQuery.error instanceof Error ? outboundDashboardQuery.error : null}
        />
      </div>
    </div>
  );
}

function TraderSalaryAnalytics({
  profile,
  isLoading,
  error,
}: {
  profile?: Awaited<ReturnType<typeof api.traderProfile.get>>;
  isLoading?: boolean;
  error?: string | null;
}) {
  if (isLoading) return <EmptyState title="Загружаем ЗП" />;
  if (error) return <EmptyState title="Не удалось загрузить ЗП" description={error} />;
  if (!profile) return null;

  return (
    <section>
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="border-emerald-200 bg-emerald-50 md:col-span-2">
          <CardContent className="flex min-h-32 flex-col justify-center p-4">
            <div className="text-xs font-medium uppercase tracking-normal text-emerald-800">ЗП за выбранный период</div>
            <div className="mt-3 text-3xl font-semibold text-emerald-700 tabular-nums">
              {formatMoneyMinor(profile.periodSalaryMinor)}
            </div>
            <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-emerald-900">
              <span>
                Сегодня: <span className="font-semibold tabular-nums">{formatMoneyMinor(profile.currentShiftSalaryMinor)}</span>
              </span>
              <span>Ставка {profile.salaryRateBps / 100}%</span>
            </div>
          </CardContent>
        </Card>
        <MetricCard layout="header" label="Ставка" value={`${profile.salaryRateBps / 100}%`} />
      </div>
    </section>
  );
}
