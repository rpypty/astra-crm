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

export function TeamleadTransactionsPage({ initialDirection = "inbound" }: { initialDirection?: OrderDirection }) {
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TEAMLEAD_PERIOD_FILTER_STORAGE_KEY);
  const [activeOrdersDirection, setActiveOrdersDirection] = useState<OrderDirection>(initialDirection);
  const [, startOrdersTabTransition] = useTransition();
  const activeLabel = activeOrdersDirection === "inbound" ? "Инвойсы" : "Выплаты";

  useEffect(() => {
    setActiveOrdersDirection(initialDirection);
  }, [initialDirection]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Транзакции"
        description="Ордера по инвойсам и выплатам в одном разделе."
        actions={
          <ImportCsvDialog
            scopeLabel={`Сверка тимлида: ${activeLabel.toLowerCase()}`}
            scope="teamlead"
            direction={activeOrdersDirection}
          />
        }
      />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <section className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-lg font-semibold">{activeLabel}</h2>
          <div className="inline-flex rounded-md border border-border bg-card p-1">
            <Button
              type="button"
              variant={activeOrdersDirection === "inbound" ? "default" : "ghost"}
              size="sm"
              onClick={() => startOrdersTabTransition(() => setActiveOrdersDirection("inbound"))}
            >
              Инвойсы
            </Button>
            <Button
              type="button"
              variant={activeOrdersDirection === "outbound" ? "default" : "ghost"}
              size="sm"
              onClick={() => startOrdersTabTransition(() => setActiveOrdersDirection("outbound"))}
            >
              Выплаты
            </Button>
          </div>
        </div>
        <OrdersPage
          direction={activeOrdersDirection}
          scope="teamlead"
          embedded
          periodFilter={periodFilter}
          showReconciliation={false}
        />
      </section>
    </div>
  );
}

export function OrdersPage({
  direction,
  scope,
  embedded,
  periodFilter: externalPeriodFilter,
  showReconciliation = true,
}: {
  direction: "inbound" | "outbound";
  scope: "teamlead" | "trader";
  embedded?: boolean;
  periodFilter?: PeriodFilter;
  showReconciliation?: boolean;
}) {
  const [storedPeriodFilter, setStoredPeriodFilter] = usePersistentPeriodFilter(
    scope === "teamlead" ? TEAMLEAD_PERIOD_FILTER_STORAGE_KEY : "astra-crm:trader-period-filter",
  );
  const periodFilter = externalPeriodFilter ?? storedPeriodFilter;
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [status, setStatus] = useState("all");
  const [selectedTraderIds, setSelectedTraderIds] = useState<number[]>([]);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsOrder, setDetailsOrder] = useState<Order | null>(null);
  const dashboardFilters = useMemo(
    () => ({ ...periodFilter, traderIds: selectedTraderIds }),
    [periodFilter, selectedTraderIds],
  );
  const orderServerFilters = useMemo(
    () => ({ status, ...periodFilter, traderIds: selectedTraderIds }),
    [periodFilter, selectedTraderIds, status],
  );
  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ status: "active" }),
    queryFn: () => api.traders.list({ status: "active" }),
    enabled: scope === "teamlead",
  });
  const dashboardQuery = useQuery({
    queryKey:
      scope === "teamlead"
        ? queryKeys.teamlead.dashboard(direction, dashboardFilters)
        : queryKeys.trader.dashboard(direction, dashboardFilters),
    queryFn: () => api.orders.dashboard(scope, direction, dashboardFilters),
    enabled: !embedded,
  });
  const ordersQuery = useQuery({
    queryKey:
      scope === "teamlead"
        ? queryKeys.teamlead.orders(direction, orderServerFilters)
        : queryKeys.trader.orders(direction, orderServerFilters),
    queryFn: () => api.orders.list(scope, direction, orderServerFilters),
  });
  const reconciliationQuery = useQuery({
    queryKey: [scope, direction, "reconciliation"],
    queryFn: () => api.orders.reconciliation(scope, direction),
    enabled: showReconciliation,
  });
  useEffect(() => {
    setPagination((current) => (current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }));
  }, [deferredSearch, direction, orderServerFilters]);
  const columns = useMemo<ColumnDef<Order>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "trader", header: "Трейдер", cell: ({ row }) => <UserCell login={row.original.trader} secondary={row.original.workerName} /> },
      { accessorKey: "requisite", header: "Реквизит", cell: ({ row }) => <RequisiteCell phone={row.original.requisite} method={row.original.method} /> },
      { accessorKey: "bankName", header: "Банк" },
      { accessorKey: "amountMinor", header: () => <div className="text-right">Сумма</div>, cell: ({ row }) => <MoneyCell valueMinor={row.original.amountMinor} /> },
      {
        accessorKey: "normalizedStatus",
        header: "Статус",
        cell: ({ row }) => (
          <div className="space-y-1">
            <StatusBadge status={row.original.normalizedStatus} />
            {row.original.rawStatus !== row.original.normalizedStatus ? (
              <div className="text-xs text-muted-foreground">{row.original.rawStatus}</div>
            ) : null}
          </div>
        ),
      },
      { accessorKey: "innerId", header: "innerId" },
    ],
    [],
  );
  const data = useMemo(
    () => filterOrdersBySearch(ordersQuery.data ?? [], deferredSearch),
    [deferredSearch, ordersQuery.data],
  );
  const title = direction === "inbound" ? "Инвойсы" : "Выплаты";

  return (
    <div className="space-y-6">
      {!embedded ? (
        <PageHeader
          title={title}
          description="Ордера, импорт CSV и состояние сверки."
          actions={
            <ImportCsvDialog
              scopeLabel={direction === "inbound" ? "Сверка тимлида: инвойсы" : "Сверка тимлида: выплаты"}
              scope={scope}
              direction={direction}
            />
          }
        />
      ) : null}
      {!embedded ? <PeriodFilterBar value={periodFilter} onChange={setStoredPeriodFilter} /> : null}
      {!embedded ? (
        <OrderDashboard
          dashboard={dashboardQuery.data}
          direction={direction}
          isLoading={dashboardQuery.isLoading}
          error={dashboardQuery.error instanceof Error ? dashboardQuery.error : null}
        />
      ) : null}
      {showReconciliation && reconciliationQuery.data ? <MismatchAlert summary={reconciliationQuery.data} /> : null}
      {showReconciliation && scope === "trader" && reconciliationQuery.data?.status === "mismatch" && reconciliationQuery.data.runId ? (
        <AcceptMismatchDialog scope={scope} direction={direction} runId={reconciliationQuery.data.runId} />
      ) : null}
      {showReconciliation && reconciliationQuery.data?.status !== "mismatch" && reconciliationQuery.data ? (
        <Card>
          <CardContent className="flex flex-wrap items-center justify-between gap-3 p-4">
            <div className="flex items-center gap-3">
              <StatusBadge status={reconciliationQuery.data.status} />
              <span className="text-sm text-muted-foreground">
                Ожидалось {formatMoneyMinor(reconciliationQuery.data.expectedMinor)}, факт{" "}
                {formatMoneyMinor(reconciliationQuery.data.actualMinor)}
              </span>
            </div>
            <span className="text-sm font-medium">Diff: {formatMoneyMinor(reconciliationQuery.data.diffMinor)}</span>
          </CardContent>
        </Card>
      ) : null}
      <DataTable
        columns={columns}
        data={data}
        rowCount={data.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        toolbarFilters={
          <div className="flex flex-wrap items-center gap-2">
            <Select className="w-44" value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="all">Все статусы</option>
              <option value="success">Успех</option>
              <option value="corrected">Исправлен</option>
              <option value="failed">Неуспех</option>
              <option value="cancelled">Отменен</option>
              <option value="unknown">Неизвестно</option>
            </Select>
            {scope === "teamlead" ? (
              <TraderFilterDropdown
                traders={tradersQuery.data ?? []}
                selectedTraderIds={selectedTraderIds}
                isLoading={tradersQuery.isLoading}
                onChange={setSelectedTraderIds}
              />
            ) : null}
          </div>
        }
        isLoading={ordersQuery.isLoading}
        error={ordersQuery.error instanceof Error ? ordersQuery.error.message : null}
        emptyTitle="Ордеров пока нет"
        emptyDescription="После CSV-импорта активного scope здесь появятся ордера."
        onRowClick={setDetailsOrder}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsOrder(row) }]}
      />
      <OrderDetailsDialog order={detailsOrder} onClose={() => setDetailsOrder(null)} />
    </div>
  );
}

function TraderFilterDropdown({
  traders,
  selectedTraderIds,
  isLoading,
  onChange,
}: {
  traders: Trader[];
  selectedTraderIds: number[];
  isLoading?: boolean;
  onChange: (traderIds: number[]) => void;
}) {
  const [search, setSearch] = useState("");
  const selected = useMemo(() => new Set(selectedTraderIds), [selectedTraderIds]);
  const filteredTraders = useMemo(() => {
    const normalizedSearch = search.trim().toLowerCase();
    if (!normalizedSearch) {
      return traders;
    }

    return traders.filter((trader) =>
      [trader.login, trader.externalWorkerName].some((value) => value.toLowerCase().includes(normalizedSearch)),
    );
  }, [search, traders]);
  const label = selectedTraderIds.length === 0 ? "Все трейдеры" : `Трейдеры: ${selectedTraderIds.length}`;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="outline" className="min-w-44 justify-between">
          {label}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="flex max-h-[min(28rem,var(--radix-dropdown-menu-content-available-height))] w-80 max-w-[calc(100vw-2rem)] flex-col overflow-hidden"
      >
        <div className="space-y-2 border-b border-border p-2">
          <DropdownMenuLabel className="px-0 py-0">Трейдеры</DropdownMenuLabel>
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            onKeyDown={(event) => event.stopPropagation()}
            placeholder="Найти трейдера"
          />
          {selectedTraderIds.length > 0 ? (
            <Button type="button" variant="ghost" size="sm" className="w-full justify-start px-2" onClick={() => onChange([])}>
              Сбросить выбор
            </Button>
          ) : null}
        </div>
        <div className="min-h-0 overflow-y-auto p-1">
          {isLoading ? <DropdownMenuItem disabled>Загрузка...</DropdownMenuItem> : null}
          {!isLoading && traders.length === 0 ? <DropdownMenuItem disabled>Нет активных трейдеров</DropdownMenuItem> : null}
          {!isLoading && traders.length > 0 && filteredTraders.length === 0 ? (
            <DropdownMenuItem disabled>Ничего не найдено</DropdownMenuItem>
          ) : null}
          {filteredTraders.map((trader) => (
            <DropdownMenuCheckboxItem
              key={trader.id}
              checked={selected.has(trader.id)}
              className="max-w-full"
              onCheckedChange={(checked) => {
                if (checked) {
                  onChange([...selectedTraderIds, trader.id]);
                  return;
                }
                onChange(selectedTraderIds.filter((traderId) => traderId !== trader.id));
              }}
              onSelect={(event) => event.preventDefault()}
            >
              <span className="min-w-0 truncate" title={trader.login}>
                {trader.login}
              </span>
            </DropdownMenuCheckboxItem>
          ))}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function OrderDetailsDialog({ order, onClose }: { order: Order | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(order)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Ордер {order?.innerId}</DialogTitle>
          <DialogDescription>Данные активного import scope из backend.</DialogDescription>
        </DialogHeader>
        {order ? (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Время" value={formatDateTime(order.createdAt)} />
              <ReadOnlyField label="Сумма" value={formatMoneyMinor(order.amountMinor)} />
              <ReadOnlyField label="Трейдер" value={order.trader} />
              <ReadOnlyField label="Worker" value={order.workerName} />
              <ReadOnlyField label="Реквизит" value={order.requisite || "—"} />
              <ReadOnlyField label="Метод/банк" value={order.bankName || order.method || "—"} />
              <ReadOnlyField label="External ID" value={order.externalId} />
              <ReadOnlyField label="Import batch" value={String(order.importBatchId)} />
            </div>
            <div className="rounded-md border border-border p-3">
              <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Статус</div>
              <div className="flex flex-wrap items-center gap-3">
                <StatusBadge status={order.normalizedStatus} />
                <span className="text-sm text-muted-foreground">raw: {order.rawStatus}</span>
              </div>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
