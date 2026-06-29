import { zodResolver } from "@hookform/resolvers/zod";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import type { Order, OrderDirection, Payout, PayoutTransfer, ReconciliationItem, ReconciliationSummary, ShiftReport, ShiftRequisite } from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import type { PeriodFilter } from "@/shared/lib/period-filter";
import { usePersistentPeriodFilter } from "@/shared/lib/period-filter";
import { queryKeys } from "@/shared/api/query-keys";
import { paginationToQuery } from "@/shared/lib/pagination";
import {

  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  normalizeCardNumber,
  parseMoneyToMinor,
} from "@/shared/lib/utils";
import { ReadOnlyField } from "@/shared/ui/read-only-field";

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

export function TraderOrdersPage({ direction }: { direction: "inbound" | "outbound" }) {
  return <TraderTransactionsPage initialDirection={direction} />;
}

export function TraderTransactionsPage({ initialDirection = "inbound" }: { initialDirection?: OrderDirection }) {
  const [activeOrdersDirection, setActiveOrdersDirection] = useState<OrderDirection>(initialDirection);
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TRADER_PERIOD_FILTER_STORAGE_KEY);
  const activeLabel = activeOrdersDirection === "inbound" ? "Инвойсы" : "Выплаты";

  useEffect(() => {
    setActiveOrdersDirection(initialDirection);
  }, [initialDirection]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Транзакции"
        description="Импорт CSV, сверка и ордера текущей смены."
        actions={
          <ImportCsvDialog
            scopeLabel={`${activeLabel} текущей смены`}
            scope="trader"
            direction={activeOrdersDirection}
          />
        }
      />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <div className="flex justify-end">
        <div className="inline-flex rounded-md border border-border bg-card p-1">
          <Button
            type="button"
            variant={activeOrdersDirection === "inbound" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveOrdersDirection("inbound")}
          >
            Инвойсы
          </Button>
          <Button
            type="button"
            variant={activeOrdersDirection === "outbound" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveOrdersDirection("outbound")}
          >
            Выплаты
          </Button>
        </div>
      </div>
      <TraderOrdersDirectionContent direction={activeOrdersDirection} periodFilter={periodFilter} />
    </div>
  );
}

function TraderOrdersDirectionContent({ direction, periodFilter }: { direction: OrderDirection; periodFilter: PeriodFilter }) {
  const confirmedPeriodFilter = useMemo(() => ({ ...periodFilter, confirmedOnly: true }), [periodFilter]);
  const dashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard(direction, confirmedPeriodFilter),
    queryFn: () => api.orders.dashboard("trader", direction, confirmedPeriodFilter),
  });
  const reconciliationQuery = useQuery({
    queryKey: ["trader", direction, "reconciliation"],
    queryFn: () => api.orders.reconciliation("trader", direction),
  });

  return (
    <div className="space-y-6">
      <OrderDashboard
        dashboard={dashboardQuery.data}
        direction={direction}
        isLoading={dashboardQuery.isLoading}
        error={dashboardQuery.error instanceof Error ? dashboardQuery.error : null}
      />
      {reconciliationQuery.data ? <MismatchAlert summary={reconciliationQuery.data} /> : null}
      {reconciliationQuery.data?.status === "mismatch" && reconciliationQuery.data.runId ? (
        <AcceptMismatchDialog scope="trader" direction={direction} runId={reconciliationQuery.data.runId} />
      ) : null}
      <TraderOrdersTable direction={direction} periodFilter={periodFilter} />
    </div>
  );
}

function TraderOrdersTable({ direction, periodFilter }: { direction: "inbound" | "outbound"; periodFilter: PeriodFilter }) {
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsOrder, setDetailsOrder] = useState<Order | null>(null);
  const confirmedPeriodFilter = useMemo(
    () => ({ ...periodFilter, confirmedOnly: true, search: deferredSearch }),
    [deferredSearch, periodFilter],
  );
  const orderServerFilters = useMemo(
    () => ({ ...confirmedPeriodFilter, ...paginationToQuery(pagination) }),
    [confirmedPeriodFilter, pagination],
  );
  const ordersQuery = useQuery({
    queryKey: queryKeys.trader.orders(direction, orderServerFilters),
    queryFn: () => api.orders.list("trader", direction, orderServerFilters),
    placeholderData: keepPreviousData,
  });
  const data = ordersQuery.data?.items ?? [];
  useEffect(() => {
    setPagination((current) => (current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }));
  }, [deferredSearch, direction, confirmedPeriodFilter]);
  const columns = useMemo<ColumnDef<Order>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
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
      {
        accessorKey: "tlReconciliationStatus",
        header: "TL",
        cell: ({ row }) => <StatusBadge status={row.original.tlReconciliationStatus} />,
      },
      { accessorKey: "innerId", header: "innerId" },
    ],
    [],
  );
  return (
    <>
      <DataTable
        columns={columns}
        data={data}
        rowCount={ordersQuery.data?.total ?? 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        serverSidePagination
        search={search}
        onSearchChange={setSearch}
        isLoading={ordersQuery.isLoading}
        isFetching={ordersQuery.isFetching}
        onRowClick={setDetailsOrder}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsOrder(row) }]}
      />
      <TraderOrderDetailsDialog order={detailsOrder} onClose={() => setDetailsOrder(null)} />
    </>
  );
}

function TraderOrderDetailsDialog({ order, onClose }: { order: Order | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(order)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Ордер {order?.innerId}</DialogTitle>
          <DialogDescription>Данные активного import scope.</DialogDescription>
        </DialogHeader>
        {order ? (
          <div className="grid gap-3 md:grid-cols-2">
            <ReadOnlyField label="Время" value={new Date(order.createdAt).toLocaleString("ru-RU")} />
            <ReadOnlyField label="Сумма" value={formatMoneyMinor(order.amountMinor)} />
            <ReadOnlyField label="Реквизит" value={formatRussianPhone(order.requisite)} />
            <ReadOnlyField label="Метод/банк" value={order.bankName || order.method || "—"} />
            <ReadOnlyField label="Статус" value={order.rawStatus} />
            <div className="rounded-md border border-border px-3 py-2">
              <div className="mb-1 text-xs text-muted-foreground">TL-сверка</div>
              <StatusBadge status={order.tlReconciliationStatus} />
            </div>
            <ReadOnlyField label="innerId" value={order.innerId} />
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
