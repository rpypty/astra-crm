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

export function TeamleadAuditPage() {
  const auditQuery = useQuery({ queryKey: queryKeys.teamlead.audit(), queryFn: api.audit.list });
  const [search, setSearch] = useState("");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsEntry, setDetailsEntry] = useState<AuditLogEntry | null>(null);
  const auditItems = auditQuery.data ?? [];
  const normalizedSearch = search.trim().toLowerCase();
  const filteredAuditItems = normalizedSearch
    ? auditItems.filter((item) =>
        [item.actorLogin, item.action, item.entityType, item.entityId, item.comment ?? ""].some((value) =>
          value.toLowerCase().includes(normalizedSearch),
        ),
      )
    : auditItems;
  const actorsCount = new Set(auditItems.map((item) => item.actorLogin)).size;
  const columns = useMemo<ColumnDef<AuditLogEntry>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "actorLogin", header: "Автор" },
      { accessorKey: "action", header: "Действие" },
      { accessorKey: "entityType", header: "Сущность" },
      { accessorKey: "entityId", header: "ID" },
      { accessorKey: "comment", header: "Комментарий", cell: ({ row }) => row.original.comment ?? "—" },
    ],
    [],
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Аудит" description="Журнал изменений по команде." />
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="События" value={String(auditItems.length)} />
        <MetricCard label="Авторы" value={String(actorsCount)} />
        <MetricCard label="Последнее событие" value={auditItems[0] ? formatDateTime(auditItems[0].createdAt) : "—"} />
      </div>
      <DataTable
        columns={columns}
        data={filteredAuditItems}
        rowCount={filteredAuditItems.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        isLoading={auditQuery.isLoading}
        error={auditQuery.error instanceof Error ? auditQuery.error.message : null}
        emptyTitle="Событий аудита нет"
        emptyDescription="Мутации в системе будут отображаться здесь."
        onRowClick={setDetailsEntry}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsEntry(row) }]}
      />
      <AuditDetailsDialog entry={detailsEntry} onClose={() => setDetailsEntry(null)} />
    </div>
  );
}

function AuditDetailsDialog({ entry, onClose }: { entry: AuditLogEntry | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(entry)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Аудит #{entry?.id}</DialogTitle>
          <DialogDescription>Payload отображается в том виде, в котором backend вернул read model.</DialogDescription>
        </DialogHeader>
        {entry ? (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Время" value={formatDateTime(entry.createdAt)} />
              <ReadOnlyField label="Автор" value={entry.actorLogin} />
              <ReadOnlyField label="Действие" value={entry.action} />
              <ReadOnlyField label="Сущность" value={`${entry.entityType} #${entry.entityId}`} />
            </div>
            {entry.comment ? <ReadOnlyField label="Комментарий" value={entry.comment} /> : null}
            <pre className="max-h-96 overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-50">
              {JSON.stringify(entry.maskedPayload, null, 2)}
            </pre>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

