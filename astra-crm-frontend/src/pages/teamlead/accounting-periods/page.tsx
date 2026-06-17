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
import { MetricCard } from "@/shared/ui/metric-card";
import { ReadOnlyField } from "@/shared/ui/read-only-field";

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
  targetTurnover: z.string().min(1, "Введите целевой оборот").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
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

export function TeamleadPeriodsPage() {
  const periodsQuery = useQuery({ queryKey: ["teamlead", "periods"], queryFn: api.periods.list });
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsPeriod, setDetailsPeriod] = useState<AccountingPeriod | null>(null);
  const periods = periodsQuery.data ?? [];
  const openCount = periods.filter((period) => period.status === "open").length;
  const mismatchCount = periods.filter(
    (period) => period.inboundStatus === "mismatch" || period.outboundStatus === "mismatch",
  ).length;
  const discrepancyCount = periods.filter((period) => period.status === "closed_with_discrepancy").length;
  const columns = useMemo<ColumnDef<AccountingPeriod>[]>(
    () => [
      { accessorKey: "title", header: "Сверка" },
      { accessorKey: "dateRange", header: "Даты" },
      { accessorKey: "inboundStatus", header: "Инвойсы", cell: ({ row }) => <StatusBadge status={row.original.inboundStatus} /> },
      { accessorKey: "outboundStatus", header: "Выплаты", cell: ({ row }) => <StatusBadge status={row.original.outboundStatus} /> },
      { accessorKey: "status", header: "Статус", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
    ],
    [],
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Сверка" description="Инвойсы и выплаты по учетным периодам." />
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Открытые периоды" value={String(openCount)} />
        <MetricCard label="Расхождения" value={String(mismatchCount)} warning={mismatchCount > 0} />
        <MetricCard label="Закрыты с расхождением" value={String(discrepancyCount)} warning={discrepancyCount > 0} />
      </div>
      <DataTable
        columns={columns}
        data={periods}
        rowCount={periods.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={periodsQuery.isLoading}
        error={periodsQuery.error instanceof Error ? periodsQuery.error.message : null}
        emptyTitle="Сверок пока нет"
        emptyDescription="Записи появятся после создания accounting period."
        onRowClick={setDetailsPeriod}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsPeriod(row) }]}
      />
      <PeriodDetailsDialog period={detailsPeriod} onClose={() => setDetailsPeriod(null)} />
    </div>
  );
}

function PeriodDetailsDialog({ period, onClose }: { period: AccountingPeriod | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(period)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{period?.title}</DialogTitle>
          <DialogDescription>Состояние итоговой сверки по учетному периоду.</DialogDescription>
        </DialogHeader>
        {period ? (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Даты" value={period.dateRange} />
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Статус периода</div>
                <StatusBadge status={period.status} />
              </div>
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Инвойсы</div>
                <StatusBadge status={period.inboundStatus} />
              </div>
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Выплаты</div>
                <StatusBadge status={period.outboundStatus} />
              </div>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}


