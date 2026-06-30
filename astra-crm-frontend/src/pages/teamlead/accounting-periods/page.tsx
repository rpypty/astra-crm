import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertTriangle, ArrowDownLeft, ArrowUpRight, CheckCircle2, Copy, Eye, FileText, RefreshCw, Upload, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { CopyToast, type CopyHandler } from "@/entities/requisite/ui/requisite-phone-menu";
import { AcceptMismatchDialog } from "@/features/import-csv/ui/import-components";
import { StatusBadge } from "@/entities/status/ui/status-badge";
import { PageHeader } from "@/shared/ui/page-header";
import { EmptyState } from "@/shared/ui/empty-state";
import { DataTable } from "@/shared/ui/data-table";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { ConfirmDialog } from "@/shared/ui/confirm-dialog";
import { DatePickerField } from "@/shared/ui/date-picker-field";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
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
import type {
  OrderDirection,
  OrderImportHistoryItem,
  ReconciliationItem,
  ReconciliationSummary,
  Requisite,
  TeamleadReconciliationItem,
  TeamleadReconciliationRun,
  Trader,
} from "@/shared/model/domain";
import { api, type TeamleadReconciliationItemFilters } from "@/shared/api/api";
import { queryKeys } from "@/shared/api/query-keys";
import { paginationToQuery } from "@/shared/lib/pagination";
import { cardDigits, cn, formatCardNumber, formatDateTime, formatMoneyMinor, formatRussianPhone, normalizeRussianPhone } from "@/shared/lib/utils";

type ToastMessage = {
  id: number;
  title: string;
  message: string;
};

type TeamleadMismatchRow = {
  id: string;
  issueType: string;
  issueLabel: string;
  requisite: string;
  trader: string;
  innerId: string;
  csvAmountMinor?: number;
  crmAmountMinor?: number;
  diffMinor?: number;
  csvCount?: number;
  crmCount?: number;
  csvStatus: string;
  crmStatus: string;
  createdAt: string;
};

type TeamleadReconciliationHistoryRow = {
  id: number;
  direction: OrderDirection;
  summary: ReconciliationSummary;
};

type SelectedHistoryRun = {
  direction: OrderDirection;
  summary: ReconciliationSummary;
};

type TeamleadReconciliationTableRow = {
  id: string;
  direction: OrderDirection;
  stage: string;
  stages: string[];
  severity: TeamleadReconciliationItem["severity"];
  statusLabel: string;
  statusTitle: string;
  innerId: string;
  trader: string;
  requisite: TeamleadReconciliationRequisiteCellData;
  amountMinor?: number;
  bankMethod: string;
  reasons: string[];
  isBlocking: boolean;
};

type TeamleadReconciliationRequisiteCellData = {
  displayValue: string;
  rawValue?: string;
  kind: "phone" | "card" | "raw" | "empty";
  mode: "crm_requisite" | "csv_recipient";
  requisiteId?: number;
  bankCode?: string;
  bankName?: string;
  methodName?: string;
  methodType?: string;
  recipientName?: string;
};

export function TeamleadPeriodsPage() {
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [selectedRunID, setSelectedRunID] = useState<number | null>(null);
  const [activeDirection, setActiveDirection] = useState<OrderDirection>("inbound");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 15 });
  const [itemPagination, setItemPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 15 });
  const listParams = paginationToQuery(pagination);
  const runsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationV2(listParams),
    queryFn: () => api.teamleadReconciliations.list(listParams),
    placeholderData: keepPreviousData,
  });
  const selectedRunQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationV2Run(selectedRunID ?? undefined),
    queryFn: () => api.teamleadReconciliations.get(selectedRunID ?? 0),
    enabled: selectedRunID !== null,
  });
  const itemsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationV2Items(selectedRunID ?? undefined, { direction: activeDirection, scope: "all" }),
    queryFn: () => fetchAllTeamleadReconciliationItems(selectedRunID ?? 0, activeDirection),
    enabled: selectedRunID !== null,
  });
  const columns = useTeamleadV2HistoryColumns();
  const latestRun = runsQuery.data?.items[0];

  useEffect(() => {
    setItemPagination({ pageIndex: 0, pageSize: 15 });
  }, [selectedRunID, activeDirection]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Сверка"
        description="История сверок за период, расхождения и применение TL CSV к транзакциям."
        actions={
          <Button type="button" onClick={() => setUploadDialogOpen(true)}>
            <Upload className="h-4 w-4" />
            Загрузить сверку
          </Button>
        }
      />

      <TeamleadV2Overview run={latestRun} total={runsQuery.data?.total ?? 0} />

      <DataTable
        columns={columns}
        data={runsQuery.data?.items ?? []}
        rowCount={runsQuery.data?.total ?? 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={runsQuery.isLoading}
        isFetching={runsQuery.isFetching}
        serverSidePagination
        emptyTitle="Сверок пока нет"
        emptyDescription="Загрузите CSV входов или выходов за период, чтобы создать первый run."
        actions={[
          {
            label: "Открыть",
            onSelect: (row) => {
              setSelectedRunID(row.id);
              setActiveDirection(row.inboundImportBatchId ? "inbound" : "outbound");
            },
          },
        ]}
        onRowClick={(row) => {
          setSelectedRunID(row.id);
          setActiveDirection(row.inboundImportBatchId ? "inbound" : "outbound");
        }}
      />

      <TeamleadReconciliationV2UploadDialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen} />
      <TeamleadReconciliationV2DetailsDialog
        run={selectedRunQuery.data ?? null}
        items={itemsQuery.data ?? []}
        itemPagination={itemPagination}
        onItemPaginationChange={setItemPagination}
        activeDirection={activeDirection}
        onDirectionChange={setActiveDirection}
        open={selectedRunID !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedRunID(null);
        }}
        isLoading={selectedRunQuery.isLoading}
        isItemsLoading={itemsQuery.isLoading}
        isItemsFetching={itemsQuery.isFetching}
      />
    </div>
  );
}

async function fetchAllTeamleadReconciliationItems(runId: number, direction: OrderDirection) {
  const pageSize = 200;
  const items: TeamleadReconciliationItem[] = [];
  let page = 1;
  let total = Number.POSITIVE_INFINITY;

  while (items.length < total) {
    const filters: TeamleadReconciliationItemFilters = { direction, page, pageSize };
    const response = await api.teamleadReconciliations.items(runId, filters);
    items.push(...response.items);
    total = response.total;
    if (response.items.length === 0) break;
    page += 1;
  }

  return items;
}

function TeamleadV2Overview({ run, total }: { run?: TeamleadReconciliationRun; total: number }) {
  const inboundAmount = run ? summaryNumber(run.inboundSummary, "successAmountMinor") : undefined;
  const outboundAmount = run ? summaryNumber(run.outboundSummary, "successAmountMinor") : undefined;

  return (
    <div className="grid gap-3 md:grid-cols-4">
      <Card>
        <CardContent className="p-4">
          <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">Run'ов</div>
          <div className="mt-2 text-2xl font-semibold">{total}</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4">
          <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">Последний статус</div>
          <div className="mt-2">{run ? <StatusBadge status={run.status} /> : <span className="text-sm text-muted-foreground">Нет run'ов</span>}</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4">
          <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">Входы последнего run</div>
          <div className="mt-2 text-xl font-semibold">{inboundAmount === undefined ? "—" : formatMoneyMinor(inboundAmount)}</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4">
          <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">Выходы последнего run</div>
          <div className="mt-2 text-xl font-semibold">{outboundAmount === undefined ? "—" : formatMoneyMinor(outboundAmount)}</div>
        </CardContent>
      </Card>
    </div>
  );
}

function useTeamleadV2HistoryColumns() {
  return useMemo<ColumnDef<TeamleadReconciliationRun>[]>(
    () => [
      {
        accessorKey: "id",
        header: "Run",
        cell: ({ row }) => <span className="font-medium">#{row.original.id}</span>,
      },
      {
        id: "period",
        header: "Период",
        cell: ({ row }) => `${row.original.dateFrom} — ${row.original.dateTo}`,
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        id: "directions",
        header: "Направления",
        cell: ({ row }) => (
          <div className="flex flex-wrap gap-1">
            {row.original.inboundImportBatchId ? <span className="rounded-md bg-emerald-50 px-2 py-1 text-xs text-emerald-700">Входы</span> : null}
            {row.original.outboundImportBatchId ? <span className="rounded-md bg-sky-50 px-2 py-1 text-xs text-sky-700">Выходы</span> : null}
          </div>
        ),
      },
      {
        accessorKey: "comment",
        header: "Комментарий",
        cell: ({ row }) => (
          <span className="block max-w-[280px] truncate text-muted-foreground" title={row.original.comment ?? ""}>
            {row.original.comment?.trim() || "—"}
          </span>
        ),
      },
      {
        id: "summary",
        header: "Расхождения",
        cell: ({ row }) => (
          <span className={row.original.mismatchCount > 0 ? "font-semibold text-red-700" : "text-muted-foreground"}>
            {row.original.mismatchCount}
          </span>
        ),
      },
      {
        accessorKey: "createdAt",
        header: "Запущена",
        cell: ({ row }) => formatDateTime(row.original.createdAt),
      },
    ],
    [],
  );
}

function TeamleadReconciliationV2UploadDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const queryClient = useQueryClient();
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [selectedTraderIds, setSelectedTraderIds] = useState<number[]>([]);
  const [inboundFile, setInboundFile] = useState<File | null>(null);
  const [outboundFile, setOutboundFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ status: "active" }),
    queryFn: () => api.traders.list({ status: "active", page: 1, pageSize: 200 }),
    enabled: open,
  });
  const createMutation = useMutation({
    mutationFn: () => api.teamleadReconciliations.create({ dateFrom, dateTo, traderIds: selectedTraderIds, inboundFile, outboundFile }),
    onSuccess: async () => {
      setDateFrom("");
      setDateTo("");
      setSelectedTraderIds([]);
      setInboundFile(null);
      setOutboundFile(null);
      setError(null);
      onOpenChange(false);
      await queryClient.invalidateQueries({ queryKey: ["teamlead", "reconciliations"] });
    },
    onError: (mutationError) => setError(mutationError instanceof Error ? mutationError.message : "Не удалось создать сверку"),
  });
  const canSubmit = dateFrom !== "" && dateTo !== "" && (inboundFile !== null || outboundFile !== null);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[1120px] p-6">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Загрузить сверку</DialogTitle>
          <DialogDescription>Выберите период выгрузки и CSV входов, выходов или оба файла.</DialogDescription>
        </DialogHeader>

        <div className="space-y-5">
          <section className="space-y-3">
            <h2 className="text-sm font-semibold">Период выгрузки</h2>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <div className="text-sm font-medium">Дата начала</div>
                <DatePickerField
                  value={dateFrom}
                  placeholder="YYYY-MM-DD"
                  max={dateTo}
                  onChange={(value) => setDateFrom(value ?? "")}
                />
              </div>
              <div className="space-y-2">
                <div className="text-sm font-medium">Дата окончания</div>
                <DatePickerField
                  value={dateTo}
                  placeholder="YYYY-MM-DD"
                  min={dateFrom}
                  onChange={(value) => setDateTo(value ?? "")}
                />
              </div>
            </div>
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">Трейдеры</h2>
            <TeamleadReconciliationTraderPicker
              traders={tradersQuery.data?.items ?? []}
              selectedTraderIds={selectedTraderIds}
              isLoading={tradersQuery.isLoading}
              onChange={setSelectedTraderIds}
            />
            <div className="text-xs text-muted-foreground">
              Если никого не выбрать, сверка будет выполнена по всем активным трейдерам.
            </div>
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">CSV файлы</h2>
            <div className="grid gap-4 md:grid-cols-2">
              <ReportFileDropzone
                label="CSV входов"
                help="Файл входящих транзакций тимлида за выбранный период."
                selectedFile={inboundFile}
                onFileChange={setInboundFile}
              />
              <ReportFileDropzone
                label="CSV выходов"
                help="Файл выплат/выходов тимлида за выбранный период."
                selectedFile={outboundFile}
                onFileChange={setOutboundFile}
              />
            </div>
          </section>

          {error ? <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div> : null}
        </div>

        <div className="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button type="button" disabled={!canSubmit || createMutation.isPending} onClick={() => createMutation.mutate()}>
            {createMutation.isPending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
            Создать run
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function TeamleadReconciliationTraderPicker({
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
    if (!normalizedSearch) return traders;
    return traders.filter((trader) =>
      [trader.login, trader.externalWorkerName].some((value) => value.toLowerCase().includes(normalizedSearch)),
    );
  }, [search, traders]);
  const label = selectedTraderIds.length === 0 ? "Все трейдеры" : `Трейдеры: ${selectedTraderIds.length}`;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="outline" className="w-full justify-between md:w-80">
          <span className="truncate">{label}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="flex max-h-[min(28rem,var(--radix-dropdown-menu-content-available-height))] w-80 max-w-[calc(100vw-2rem)] flex-col overflow-hidden"
      >
        <div className="space-y-2 border-b border-border p-2">
          <DropdownMenuLabel className="px-0 py-0">Трейдеры для сверки</DropdownMenuLabel>
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
              <span className="min-w-0 truncate" title={`${trader.login} / ${trader.externalWorkerName}`}>
                {trader.login}
                <span className="ml-2 text-muted-foreground">{trader.externalWorkerName}</span>
              </span>
            </DropdownMenuCheckboxItem>
          ))}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function TeamleadReconciliationV2DetailsDialog({
  run,
  items,
  itemPagination,
  onItemPaginationChange,
  activeDirection,
  onDirectionChange,
  open,
  onOpenChange,
  isLoading,
  isItemsLoading,
  isItemsFetching,
}: {
  run: TeamleadReconciliationRun | null;
  items: TeamleadReconciliationItem[];
  itemPagination: PaginationState;
  onItemPaginationChange: (pagination: PaginationState) => void;
  activeDirection: OrderDirection;
  onDirectionChange: (direction: OrderDirection) => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isLoading?: boolean;
  isItemsLoading?: boolean;
  isItemsFetching?: boolean;
}) {
  const queryClient = useQueryClient();
  const [comment, setComment] = useState("");
  const [copyToast, setCopyToast] = useState<{ id: number; message: string } | null>(null);
  const copyToClipboard = useCallback<CopyHandler>((value, label) => {
    const normalized = value?.trim();
    if (!normalized || normalized === "—") return;
    void navigator.clipboard?.writeText(normalized);
    setCopyToast({ id: Date.now(), message: copySuccessMessage(label) });
  }, []);
  useEffect(() => {
    if (!copyToast) return;
    const timeout = window.setTimeout(() => setCopyToast(null), 1800);
    return () => window.clearTimeout(timeout);
  }, [copyToast]);
  const confirmMutation = useMutation({
    mutationFn: () => api.teamleadReconciliations.confirm({ runId: run?.id ?? 0, comment: comment.trim() }),
    onSuccess: async (updatedRun) => {
      setComment("");
      queryClient.setQueryData(queryKeys.teamlead.reconciliationV2Run(updatedRun.id), updatedRun);
      await queryClient.invalidateQueries({ queryKey: ["teamlead", "reconciliations"] });
    },
  });
  const rejectMutation = useMutation({
    mutationFn: () => api.teamleadReconciliations.reject({ runId: run?.id ?? 0, comment: comment.trim() }),
    onSuccess: async (updatedRun) => {
      setComment("");
      queryClient.setQueryData(queryKeys.teamlead.reconciliationV2Run(updatedRun.id), updatedRun);
      await queryClient.invalidateQueries({ queryKey: ["teamlead", "reconciliations"] });
    },
  });
  const canDecide = run?.status === "matched" || run?.status === "mismatch" || run?.status === "apply_failed";
  const hasInbound = Boolean(run?.inboundImportBatchId);
  const hasOutbound = Boolean(run?.outboundImportBatchId);
  const hasDecisionIssues = Boolean(run && (run.mismatchCount > 0 || run.conflictCount > 0 || run.blockedCount > 0));
  const trimmedComment = comment.trim();
  const canConfirm = Boolean(run) && !confirmMutation.isPending && (!hasDecisionIssues || trimmedComment !== "");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] !w-[calc(100vw-96px)] !max-w-[1280px] overflow-y-auto p-0">
        <DialogHeader className="sticky top-0 z-20 border-b border-border bg-white px-6 py-4">
          <div className="flex flex-wrap items-center justify-between gap-3 pr-8">
            <div className="min-w-0 space-y-1">
              <DialogTitle>{run ? `Сверка #${run.id}` : "Сверка"}</DialogTitle>
              <DialogDescription>{run ? `${run.dateFrom} — ${run.dateTo}` : "Загружаем детали run"}</DialogDescription>
            </div>
            {run ? (
              <div className="flex items-center self-center">
                <StatusBadge status={run.status} />
              </div>
            ) : null}
          </div>
        </DialogHeader>
        {isLoading || !run ? (
          <div className="p-6">
            <EmptyState title="Загружаем детали" />
          </div>
        ) : (
          <div className="space-y-4 px-6 pb-6 pt-5">
            {run.status === "apply_failed" ? <TeamleadApplyFailedNotice run={run} /> : null}
            <TeamleadV2Pipeline run={run} items={items} activeDirection={activeDirection} />
            <TeamleadV2DirectionTabs activeDirection={activeDirection} hasInbound={hasInbound} hasOutbound={hasOutbound} onDirectionChange={onDirectionChange} />
            <TeamleadV2DirectionSummary run={run} direction={activeDirection} />
            <TeamleadDecisionComment run={run} />
            <TeamleadV2ItemsTable
              direction={activeDirection}
              items={items}
              pagination={itemPagination}
              onPaginationChange={onItemPaginationChange}
              onCopy={copyToClipboard}
              isLoading={isItemsLoading}
              isFetching={isItemsFetching}
            />
            {canDecide ? (
              <div className="space-y-2 rounded-md border border-border p-3">
                <textarea
                  className="min-h-20 w-full rounded-md border border-border px-3 py-2 text-sm"
                  value={comment}
                  onChange={(event) => setComment(event.target.value)}
                  placeholder={hasDecisionIssues ? "Комментарий обязателен для расхождений, конфликтов или блокеров" : "Комментарий к решению"}
                />
                <div className="flex flex-wrap justify-end gap-2">
                  <Button type="button" variant="outline" disabled={rejectMutation.isPending || trimmedComment === ""} onClick={() => rejectMutation.mutate()}>
                    Отклонить
                  </Button>
                  <ConfirmDialog
                    trigger={
                      <Button type="button" disabled={!canConfirm}>
                        <CheckCircle2 className="h-4 w-4" />
                        Подтвердить и применить
                      </Button>
                    }
                    title="Подтвердить и применить сверку?"
                    description="После подтверждения TL CSV обновит транзакции CRM и повлияет на аналитику."
                    confirmText={confirmMutation.isPending ? "Применяем..." : "Подтвердить"}
                    onConfirm={() => confirmMutation.mutate()}
                  >
                    <TeamleadConfirmPreview run={run} comment={trimmedComment} />
                  </ConfirmDialog>
                </div>
              </div>
            ) : null}
            <CopyToast message={copyToast?.message ?? null} />
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function TeamleadV2DirectionTabs({
  activeDirection,
  hasInbound,
  hasOutbound,
  onDirectionChange,
}: {
  activeDirection: OrderDirection;
  hasInbound: boolean;
  hasOutbound: boolean;
  onDirectionChange: (direction: OrderDirection) => void;
}) {
  const tabs: Array<{ direction: OrderDirection; label: string; icon: typeof ArrowDownLeft; disabled: boolean }> = [
    { direction: "inbound", label: "Входы", icon: ArrowDownLeft, disabled: !hasInbound },
    { direction: "outbound", label: "Выходы", icon: ArrowUpRight, disabled: !hasOutbound },
  ];

  return (
    <div className="border-b border-border">
      <div className="grid w-full grid-cols-2">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          const isActive = activeDirection === tab.direction;
          return (
            <button
              key={tab.direction}
              type="button"
              disabled={tab.disabled}
              className={cn(
                "flex h-11 items-center justify-center gap-2 border-b-2 px-4 text-sm font-semibold transition",
                isActive ? "border-primary bg-primary/5 text-primary" : "border-transparent text-muted-foreground hover:bg-slate-50 hover:text-foreground",
                tab.disabled ? "cursor-not-allowed opacity-45 hover:bg-transparent hover:text-muted-foreground" : "",
              )}
              onClick={() => onDirectionChange(tab.direction)}
            >
              <Icon className="h-4 w-4" />
              {tab.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function TeamleadApplyFailedNotice({ run }: { run: TeamleadReconciliationRun }) {
  const reason = teamleadApplyFailureReason(run);
  const result = run.applyResult;
  return (
    <details className="group rounded-md border border-red-200 bg-red-50/80 text-red-950">
      <summary className="flex min-h-12 cursor-pointer list-none items-center gap-3 px-4 py-2.5 text-sm">
        <AlertTriangle className="h-4 w-4 shrink-0 text-red-700" />
        <span className="font-semibold">Не удалось применить сверку</span>
        <span className="ml-auto font-medium text-red-700 group-open:hidden">Подробнее</span>
        <span className="ml-auto hidden font-medium text-red-700 group-open:inline">Свернуть</span>
      </summary>
      <div className="space-y-3 border-t border-red-200 px-4 pb-4 pt-3">
        <div className="flex gap-3">
          <div className="w-5 shrink-0" />
          <div className="min-w-0 flex-1 space-y-2">
            <div className="text-sm text-red-900">{reason.message}</div>
            {reason.action ? <div className="text-sm text-red-900">{reason.action}</div> : null}
            {result && Object.keys(result).length > 0 ? (
              <div className="grid gap-2 pt-1 text-sm sm:grid-cols-4">
                <ApplyResultMetric label="Создано" value={summaryNumber(result, "createdOrders") ?? 0} />
                <ApplyResultMetric label="Обновлено" value={summaryNumber(result, "updatedOrders") ?? 0} />
                <ApplyResultMetric label="Подтверждено" value={summaryNumber(result, "confirmedOrders") ?? 0} />
                <ApplyResultMetric label="С расхождением" value={summaryNumber(result, "discrepancyOrders") ?? 0} />
              </div>
            ) : null}
            {reason.technical ? (
              <details className="pt-1 text-xs text-red-900">
                <summary className="cursor-pointer font-medium">Техническая деталь</summary>
                <div className="mt-1 break-words rounded-md bg-white/70 p-2 font-mono">{reason.technical}</div>
              </details>
            ) : null}
          </div>
        </div>
      </div>
    </details>
  );
}

function ApplyResultMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-red-200 bg-white/70 px-3 py-2">
      <div className="text-xs font-medium uppercase tracking-normal text-red-700">{label}</div>
      <div className="mt-1 font-semibold">{value}</div>
    </div>
  );
}

function teamleadApplyFailureReason(run: TeamleadReconciliationRun) {
  const raw = run.errorMessage?.trim();
  const rowMatch = raw?.match(/^row\s+(\d+):\s+unmatched trader\s+"(.+)"$/i);
  if (rowMatch) {
    return {
      message: `Строка ${rowMatch[1]} CSV: трейдер «${rowMatch[2]}» не найден среди активных трейдеров CRM.`,
      action: "Проверьте поле workerName в CSV или externalWorkerName у трейдера, затем повторите применение сверки.",
      technical: raw,
    };
  }

  const rowMessageMatch = raw?.match(/^row\s+(\d+):\s+(.+)$/i);
  if (rowMessageMatch) {
    return {
      message: `Строка ${rowMessageMatch[1]} CSV: ${translateTeamleadMessage(rowMessageMatch[2]) ?? rowMessageMatch[2]}`,
      action: "Исправьте блокирующий матчинг в CRM или CSV, затем повторите применение сверки.",
      technical: raw,
    };
  }

  if (run.blockedCount > 0) {
    return {
      message: `В сверке осталось ${run.blockedCount} блокирующих проблем. Обычно это несопоставленные трейдеры или реквизиты.`,
      action: "Откройте строки со статусом «Блокер», исправьте трейдеров/реквизиты и повторите применение.",
      technical: raw,
    };
  }

  return {
    message: raw ? translateTeamleadMessage(raw) ?? raw : "Применение остановилось из-за ошибки, но backend не вернул подробную причину.",
    action: "Проверьте техническую деталь ниже или повторите применение после исправления данных сверки.",
    technical: raw,
  };
}

type TeamleadPipelineFactView = {
  label: string;
  value: unknown;
};

type TeamleadPipelineStepView = {
  key: string;
  direction?: OrderDirection;
  stage: string;
  status: string;
  issuesCount: number;
  facts: TeamleadPipelineFactView[];
};

type TeamleadPipelineFlowView = {
  key: string;
  label: string;
  direction?: OrderDirection;
  steps: TeamleadPipelineStepView[];
};

function TeamleadV2Pipeline({ run, items, activeDirection }: { run: TeamleadReconciliationRun; items: TeamleadReconciliationItem[]; activeDirection: OrderDirection }) {
  const stages = useMemo(() => normalizeTeamleadPipeline(run), [run]);
  const flows = useMemo(() => buildTeamleadPipelineFlows(run, stages), [run, stages]);
  const pipelineRef = useRef<HTMLDivElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const [popover, setPopover] = useState<{ step: TeamleadPipelineStepView; left: number; top: number } | null>(null);

  useEffect(() => {
    if (!popover) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setPopover(null);
    };
    const handleResize = () => setPopover(null);
    const handleScroll = () => setPopover(null);
    const previousBodyOverflow = document.body.style.overflow;
    const previousDocumentOverscroll = document.documentElement.style.overscrollBehavior;
    const dialogContent = pipelineRef.current?.closest("[role='dialog']") as HTMLElement | null;
    const previousDialogOverflow = dialogContent?.style.overflow;

    document.body.style.overflow = "hidden";
    document.documentElement.style.overscrollBehavior = "contain";
    if (dialogContent) {
      dialogContent.style.overflow = "hidden";
    }
    document.addEventListener("keydown", handleKeyDown);
    document.addEventListener("scroll", handleScroll, true);
    window.addEventListener("resize", handleResize);
    return () => {
      document.body.style.overflow = previousBodyOverflow;
      document.documentElement.style.overscrollBehavior = previousDocumentOverscroll;
      if (dialogContent) {
        dialogContent.style.overflow = previousDialogOverflow ?? "";
      }
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("scroll", handleScroll, true);
      window.removeEventListener("resize", handleResize);
    };
  }, [popover]);

  if (!flows.length) return null;

  return (
    <div ref={pipelineRef} className="py-5">
      <div className="grid gap-5">
        {flows.map((flow) => (
          <div key={flow.key} className="min-w-0 rounded-lg bg-slate-50/50 px-3 py-3">
            <div className="mb-2 text-xs font-semibold uppercase tracking-normal text-muted-foreground">{flow.label}</div>
            <div className="overflow-x-auto pb-1">
              <div className="flex w-max min-w-full items-center gap-2">
                {flow.steps.map((step, index) => {
                  const label = stageLabel(step.stage);
                  const effectiveIssuesCount = pipelineStepIssueCount(run, step);
                  const isMismatch = pipelineStepIsMismatch(run, step);
                  const isPreview = step.stage === "preview";
                  const isOpen = popover?.step.key === step.key;
                  return (
                    <div key={step.key} className="flex shrink-0 items-center gap-2">
                      <button
                        type="button"
                        className={cn(
                          "flex min-w-0 items-center gap-2 rounded-full border px-2 py-1.5 text-left transition hover:bg-slate-50",
                          isOpen ? "border-primary bg-primary/5" : "border-border",
                        )}
                        onClick={(event) => {
                          if (isOpen) {
                            setPopover(null);
                            return;
                          }
                          const rect = event.currentTarget.getBoundingClientRect();
                          const popoverWidth = step.stage === "preview" ? 520 : 420;
                          const left = Math.min(
                            Math.max(rect.left + rect.width / 2 - popoverWidth / 2, 12),
                            Math.max(12, window.innerWidth - popoverWidth - 12),
                          );
                          setPopover({ step, left, top: rect.bottom + 8 });
                        }}
                      >
                        <span
                          className={cn(
                            "inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full border",
                            isPreview
                              ? "border-blue-200 bg-blue-50 text-blue-700"
                              : isMismatch
                                ? "border-red-200 bg-red-50 text-red-700"
                                : "border-emerald-200 bg-emerald-50 text-emerald-700",
                          )}
                        >
                          {isPreview ? <FileText className="h-4 w-4" /> : isMismatch ? <AlertTriangle className="h-4 w-4" /> : <CheckCircle2 className="h-4 w-4" />}
                        </span>
                        <span className="min-w-0">
                          <span className="block max-w-[190px] truncate text-xs font-semibold">{label}</span>
                          <span className={cn("block text-[11px] leading-tight", isPreview ? "text-blue-700" : isMismatch ? "text-red-700" : "text-emerald-700")}>
                            {pipelineStepCaption(run, step, effectiveIssuesCount)}
                          </span>
                        </span>
                      </button>
                      {index < flow.steps.length - 1 ? <div className="h-px w-5 shrink-0 bg-border" /> : null}
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        ))}
      </div>
      {popover && typeof document !== "undefined"
        ? createPortal(
            <>
              <div
                className="pointer-events-auto fixed inset-0 z-[1000] cursor-default bg-transparent"
                onPointerDown={() => setPopover(null)}
                onWheel={(event) => {
                  event.preventDefault();
                  setPopover(null);
                }}
                onTouchMove={(event) => {
                  event.preventDefault();
                  setPopover(null);
                }}
              />
              <div
                ref={popoverRef}
                className={cn(
                  "pointer-events-auto fixed z-[1010] select-text rounded-lg border border-border bg-white p-4 text-sm text-foreground shadow-lg",
                  popover.step.stage === "preview" ? "w-[520px]" : "w-[420px]",
                )}
                style={{ left: popover.left, top: popover.top }}
                onPointerDown={(event) => event.stopPropagation()}
                onWheel={(event) => event.stopPropagation()}
                onTouchMove={(event) => event.stopPropagation()}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="font-semibold">{stageLabel(popover.step.stage)}</div>
                    <div className="mt-1 text-xs text-muted-foreground">{stageDescription(popover.step.stage, popover.step.direction)}</div>
                  </div>
                  <StatusBadge status={pipelineStepStatus(run, popover.step)} />
                </div>
                <PipelineStepFacts run={run} step={popover.step} items={items} activeDirection={activeDirection} />
              </div>
            </>,
            document.body,
          )
        : null}
    </div>
  );
}

function PipelineStepFacts({
  run,
  step,
  items,
  activeDirection,
}: {
  run: TeamleadReconciliationRun;
  step: TeamleadPipelineStepView;
  items: TeamleadReconciliationItem[];
  activeDirection: OrderDirection;
}) {
  if (step.stage === "preview") {
    const facts = pipelineStepFacts(run, step).filter((fact) => fact.label !== "Отчет");
    return (
      <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-5">
        {facts.map((fact) => (
          <PipelineCompactMetric key={fact.label} label={fact.label} value={fact.value} />
        ))}
      </div>
    );
  }

  const review = step.direction === activeDirection ? buildPipelineStageReview(run, step, items) : null;
  if (review) {
    return (
      <div className="mt-3 space-y-3">
        <div className="rounded-md border border-border bg-slate-50 px-3 py-2">
          <div className="text-[11px] font-medium uppercase tracking-normal text-muted-foreground">Отчет</div>
          <div className="mt-1 text-sm leading-snug text-muted-foreground">{review.text}</div>
        </div>
        {review.statuses.length > 0 ? (
          <div className="grid gap-2 sm:grid-cols-2">
            {review.statuses.map((status) => (
              <PipelineCompactMetric key={status.label} label={status.label} value={String(status.count)} />
            ))}
          </div>
        ) : null}
        {review.reasons.length > 0 ? (
          <div className="rounded-md border border-border bg-white">
            <div className="border-b border-border px-3 py-2 text-[11px] font-medium uppercase tracking-normal text-muted-foreground">Причины</div>
            <div className="divide-y divide-border">
              {review.reasons.map((reason) => (
                <div key={reason.label} className="flex items-start justify-between gap-3 px-3 py-2 text-sm">
                  <span className="min-w-0 text-muted-foreground">{reason.label}</span>
                  <span className="shrink-0 font-semibold">{reason.count}</span>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </div>
    );
  }

  const facts = pipelineStepFacts(run, step).filter((fact) => !(step.stage === "transaction_check" && pipelineTransactionPreviewFactLabels.has(fact.label)));
  return (
    <div className="mt-3 grid gap-2">
      {facts.map((fact) => (
        <div key={fact.label} className="rounded-md border border-border bg-slate-50 px-3 py-2">
          <div className="text-[11px] font-medium uppercase tracking-normal text-muted-foreground">{fact.label}</div>
          <div className={cn("mt-1", isPipelineReportFact(fact) ? "text-sm leading-snug text-muted-foreground" : "font-semibold")}>{fact.value}</div>
        </div>
      ))}
    </div>
  );
}

function PipelineCompactMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md border border-border bg-slate-50 px-3 py-2">
      <div className="truncate text-[11px] font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-base font-semibold leading-tight">{value}</div>
    </div>
  );
}

function TeamleadV2DirectionSummary({ run, direction }: { run: TeamleadReconciliationRun; direction: OrderDirection }) {
  const summary = direction === "inbound" ? run.inboundSummary : run.outboundSummary;
  const preview = directionPreview(run, direction);
  if (!summary || Object.keys(summary).length === 0) return <EmptyState title="Направление не загружалось" />;
  return (
    <div className="rounded-lg border border-border bg-white px-4 py-3">
      <div className="grid gap-x-6 gap-y-3 md:grid-cols-[0.8fr_1fr_1fr_1.4fr]">
        <SummaryField label="Строк в периоде" value={String(summaryNumber(summary, "rowsInPeriod") ?? 0)} />
        <SummaryField label="Сумма TL" value={formatMoneyMinor(summaryNumber(summary, "successAmountMinor") ?? 0)} />
        <SummaryField label="Сумма CRM" value={formatMoneyMinor(summaryNumber(summary, "crmAmountMinor") ?? 0)} />
        <ApplyPreviewField preview={preview} summary={summary} direction={direction} />
      </div>
    </div>
  );
}

function SummaryField({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-[11px] font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-base font-semibold leading-tight">{value}</div>
    </div>
  );
}

function ApplyPreviewField({ preview, summary, direction }: { preview?: Record<string, unknown> | null; summary?: Record<string, unknown> | null; direction: OrderDirection }) {
  const previewSummary = preview ?? undefined;
  const applyRowsCount = summaryNumber(previewSummary, "applyRowsCount") ?? summaryNumber(summary ?? undefined, "applyRowsCount") ?? 0;
  const createCount = summaryNumber(previewSummary, "createCount") ?? 0;
  const updateCount = summaryNumber(previewSummary, "updateCount") ?? 0;
  const unchangedCount = summaryNumber(previewSummary, "unchangedCount") ?? 0;
  const applyLabel = direction === "outbound" ? "строк в базу" : "к применению";

  return (
    <div className="min-w-0">
      <div className="text-[11px] font-medium uppercase tracking-normal text-muted-foreground">После подтверждения</div>
      <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-sm leading-tight">
        <span>
          <span className="text-muted-foreground">{applyLabel}</span> <span className="font-semibold">{applyRowsCount}</span>
        </span>
        <span>
          <span className="text-muted-foreground">создать</span> <span className="font-semibold">{createCount}</span>
        </span>
        <span>
          <span className="text-muted-foreground">обновить</span> <span className="font-semibold">{updateCount}</span>
        </span>
        <span>
          <span className="text-muted-foreground">без изменений</span> <span className="font-semibold">{unchangedCount}</span>
        </span>
      </div>
    </div>
  );
}

function TeamleadDecisionComment({ run }: { run: TeamleadReconciliationRun }) {
  const comment = run.comment?.trim();
  if (!comment) return null;

  const isRejected = run.status === "rejected";
  const date = isRejected ? run.rejectedAt : run.confirmedAt ?? run.appliedAt ?? run.applyQueuedAt;

  return (
    <div className="rounded-md border border-border bg-slate-50 px-4 py-3 text-sm">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="font-medium">{isRejected ? "Комментарий отклонения" : "Комментарий подтверждения"}</span>
        {date ? <span className="text-xs text-muted-foreground">{formatDateTime(date)}</span> : null}
      </div>
      <div className="mt-1 whitespace-pre-wrap text-muted-foreground">{comment}</div>
    </div>
  );
}

function TeamleadConfirmPreview({ run, comment }: { run: TeamleadReconciliationRun; comment: string }) {
  const directions: Array<{ direction: OrderDirection; summary?: Record<string, unknown>; enabled: boolean }> = [
    { direction: "inbound" as const, summary: run.inboundSummary, enabled: Boolean(run.inboundImportBatchId) },
    { direction: "outbound" as const, summary: run.outboundSummary, enabled: Boolean(run.outboundImportBatchId) },
  ].filter((item) => item.enabled);

  return (
    <div className="space-y-3 text-sm">
      <div className="rounded-md border border-border bg-slate-50 px-3 py-2">
        <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">Период</div>
        <div className="mt-1 font-semibold">{run.dateFrom} — {run.dateTo}</div>
      </div>
      <div className="grid gap-2">
        {directions.map(({ direction, summary }) => {
          const preview = directionPreview(run, direction);
          return (
            <div key={direction} className="rounded-md border border-border px-3 py-2">
              <div className="font-medium">{directionLabel(direction)}</div>
              <div className="mt-2 grid gap-2 sm:grid-cols-3">
                <ConfirmPreviewField label="TL" value={formatMoneyMinor(summaryNumber(summary, "successAmountMinor") ?? 0)} />
                <ConfirmPreviewField label="CRM" value={formatMoneyMinor(summaryNumber(summary, "crmAmountMinor") ?? 0)} />
                <ConfirmPreviewField
                  label="Применение"
                  value={`создать ${summaryNumber(preview, "createCount") ?? 0}, обновить ${summaryNumber(preview, "updateCount") ?? 0}, без изменений ${summaryNumber(preview, "unchangedCount") ?? 0}`}
                />
              </div>
            </div>
          );
        })}
      </div>
      <div className="rounded-md border border-border bg-white px-3 py-2">
        <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">Комментарий</div>
        <div className="mt-1 whitespace-pre-wrap">{comment || "Комментарий не указан"}</div>
      </div>
    </div>
  );
}

function ConfirmPreviewField({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-0.5 truncate font-semibold">{value}</div>
    </div>
  );
}

function TeamleadV2ItemsTable({
  direction,
  items,
  pagination,
  onPaginationChange,
  onCopy,
  isLoading,
  isFetching,
}: {
  direction: OrderDirection;
  items: TeamleadReconciliationItem[];
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  onCopy: CopyHandler;
  isLoading?: boolean;
  isFetching?: boolean;
}) {
  const columns = useTeamleadV2ItemColumns(onCopy, direction);
  const rows = useMemo(() => buildTeamleadItemRows(items), [items]);

  return (
    <DataTable
      columns={columns}
      data={rows}
      rowCount={rows.length}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle="Деталей по направлению нет"
      emptyDescription="Для выбранного направления не найдено проблем или preview-строк."
      pageSizeOptions={[15, 25, 50, 100]}
      getRowClassName={(row) => {
        if (row.severity === "blocker" || row.severity === "error") return "bg-red-50 text-red-950 hover:bg-red-100";
        if (row.severity === "warning") return "bg-amber-50 text-amber-950 hover:bg-amber-100";
        return undefined;
      }}
    />
  );
}

function useTeamleadV2ItemColumns(onCopy: CopyHandler, direction: OrderDirection) {
  return useMemo<ColumnDef<TeamleadReconciliationTableRow>[]>(
    () => [
      {
        accessorKey: "innerId",
        header: "Транзакция",
        cell: ({ row }) => <CopyableInnerId value={row.original.innerId} onCopy={onCopy} />,
      },
      {
        accessorKey: "trader",
        header: "Трейдер",
        cell: ({ row }) => <span className="font-medium">{row.original.trader}</span>,
      },
      {
        accessorKey: "requisite",
        header: direction === "outbound" ? "Получатель" : "Реквизит",
        cell: ({ row }) => <TeamleadReconciliationRequisiteCell value={row.original.requisite} onCopy={onCopy} />,
      },
      {
        accessorKey: "amountMinor",
        header: "Сумма CSV",
        cell: ({ row }) => (
          <div className="text-right font-semibold tabular-nums">
            {row.original.amountMinor === undefined ? "—" : formatMoneyMinor(row.original.amountMinor)}
          </div>
        ),
      },
      {
        accessorKey: "bankMethod",
        header: "Банк / метод",
        cell: ({ row }) => <span className="block min-w-[140px] whitespace-normal">{row.original.bankMethod}</span>,
      },
      {
        accessorKey: "statusLabel",
        header: "Статус сверки",
        cell: ({ row }) => <ReconciliationStatusHint row={row.original} />,
      },
    ],
    [direction, onCopy],
  );
}

function TeamleadReconciliationRequisiteCell({ value, onCopy }: { value: TeamleadReconciliationRequisiteCellData; onCopy: CopyHandler }) {
  const [open, setOpen] = useState(false);
  const isRecipientOnly = value.mode === "csv_recipient";
  const searchValue = value.rawValue && value.rawValue !== "—" ? value.rawValue : value.displayValue !== "—" ? value.displayValue : "";
  const fallbackCopyValue = teamleadRequisiteCopyValue(value);
  const searchFilters = useMemo(
    () => ({
      search: searchValue,
      bankCode: value.bankCode,
      page: 1,
      pageSize: 2,
    }),
    [searchValue, value.bankCode],
  );
  const requisiteQuery = useQuery({
    queryKey: queryKeys.teamlead.requisite(value.requisiteId),
    queryFn: () => api.requisites.get(value.requisiteId ?? 0),
    enabled: open && !isRecipientOnly && value.requisiteId !== undefined,
  });
  const requisiteSearchQuery = useQuery({
    queryKey: queryKeys.teamlead.requisites(searchFilters),
    queryFn: () => api.requisites.list(searchFilters),
    enabled: open && !isRecipientOnly && value.requisiteId === undefined && searchValue !== "",
  });
  const requisite = requisiteQuery.data ?? requisiteSearchQuery.data?.items[0];
  const isLoading = !isRecipientOnly && (requisiteQuery.isLoading || requisiteSearchQuery.isLoading);
  const isError = !isRecipientOnly && (requisiteQuery.isError || requisiteSearchQuery.isError);
  const canOpen = value.displayValue !== "—" && (isRecipientOnly || value.requisiteId !== undefined || searchValue !== "");

  if (!canOpen) {
    return <span className="block min-w-[180px] whitespace-normal text-muted-foreground">{value.displayValue}</span>;
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          className="h-auto max-w-[220px] justify-start px-0 py-0 text-left text-sm font-medium hover:bg-transparent hover:text-primary"
          title={isRecipientOnly ? "Показать данные получателя из CSV" : "Показать данные реквизита"}
          onClick={(event) => event.stopPropagation()}
        >
          <span className={cn("truncate", value.kind === "card" && "tabular-nums")}>{value.displayValue}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-80 p-2" onClick={(event) => event.stopPropagation()}>
        <DropdownMenuLabel className="px-1">{isRecipientOnly ? "Данные получателя из CSV" : "Данные реквизита"}</DropdownMenuLabel>
        {isRecipientOnly ? (
          <TeamleadRecipientDetailsRows value={value} onCopy={onCopy} />
        ) : isLoading ? (
          <div className="px-2 py-3 text-sm text-muted-foreground">Загружаем реквизит...</div>
        ) : isError ? (
          <div className="px-2 py-3 text-sm text-red-700">Не удалось загрузить реквизит</div>
        ) : requisite ? (
          <TeamleadRequisiteDetailsRows requisite={requisite} onCopy={onCopy} />
        ) : (
          <div className="space-y-3 px-2 py-3 text-sm">
            <div className="text-muted-foreground">Реквизит не найден</div>
            {fallbackCopyValue ? (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-8"
                onClick={(event) => {
                  event.stopPropagation();
                  onCopy(fallbackCopyValue, requisiteCopyLabel(value));
                }}
              >
                <Copy className="h-3.5 w-3.5" />
                Скопировать
              </Button>
            ) : null}
          </div>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function TeamleadRecipientDetailsRows({ value, onCopy }: { value: TeamleadReconciliationRequisiteCellData; onCopy: CopyHandler }) {
  const bankValue = value.bankName || value.bankCode || "—";
  const methodValue = uniqueStrings([value.methodName, value.methodType]).join(" / ") || "—";
  const copyValue = teamleadRequisiteCopyValue(value) ?? value.rawValue ?? value.displayValue;

  return (
    <div className="space-y-1">
      <TeamleadRequisiteMenuRow label="Получатель" value={value.recipientName || "—"} copyValue={value.recipientName} onCopy={onCopy} />
      <TeamleadRequisiteMenuRow label="Реквизит из CSV" value={value.displayValue} copyValue={copyValue} onCopy={onCopy} />
      <TeamleadRequisiteMenuRow label="Банк" value={bankValue} copyValue={bankValue === "—" ? undefined : bankValue} onCopy={onCopy} />
      <TeamleadRequisiteMenuRow label="Метод" value={methodValue} copyValue={methodValue === "—" ? undefined : methodValue} onCopy={onCopy} />
    </div>
  );
}

function TeamleadRequisiteDetailsRows({ requisite, onCopy }: { requisite: Requisite; onCopy: CopyHandler }) {
  const bankValue = requisite.bankName || requisite.bankCode || "—";

  return (
    <div className="space-y-1">
      <TeamleadRequisiteMenuRow label="Телефон" value={formatRussianPhone(requisite.phone)} copyValue={normalizeRussianPhone(requisite.phone)} onCopy={onCopy} />
      <TeamleadRequisiteMenuRow label="Банк" value={bankValue} copyValue={bankValue === "—" ? undefined : bankValue} onCopy={onCopy} />
      <TeamleadRequisiteMenuRow label="Карта" value={formatCardNumber(requisite.cardNumber)} copyValue={cardDigits(requisite.cardNumber ?? "") || requisite.cardNumber} onCopy={onCopy} />
      <TeamleadRequisiteMenuRow label="Держатель" value={requisite.holderName || "—"} copyValue={requisite.holderName} onCopy={onCopy} />
      <TeamleadRequisiteMenuRow label="Прокси" value={requisite.proxy || "—"} copyValue={requisite.proxy} onCopy={onCopy} />
    </div>
  );
}

function TeamleadRequisiteMenuRow({ label, value, copyValue, onCopy }: { label: string; value: string; copyValue?: string | null; onCopy: CopyHandler }) {
  return (
    <button
      type="button"
      className="flex w-full items-center justify-between gap-3 rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent"
      onClick={(event) => {
        event.stopPropagation();
        onCopy(copyValue, label);
      }}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate font-medium">{value}</span>
    </button>
  );
}

function teamleadRequisiteCopyValue(value: TeamleadReconciliationRequisiteCellData) {
  const source = value.rawValue && value.rawValue !== "—" ? value.rawValue : value.displayValue;
  const digits = cardDigits(source);
  return digits || undefined;
}

function requisiteCopyLabel(value: TeamleadReconciliationRequisiteCellData) {
  if (value.mode === "csv_recipient") return value.kind === "card" ? "Карта получателя" : "Получатель";
  if (value.kind === "card") return "Карта";
  if (value.kind === "phone") return "Телефон";
  return "Реквизит";
}

function copySuccessMessage(label: string) {
  if (label === "Карта") return "Карта скопирована";
  if (label === "Карта получателя") return "Карта получателя скопирована";
  return `${label} скопирован`;
}

function CopyableInnerId({ value, onCopy }: { value: string; onCopy: CopyHandler }) {
  const [copied, setCopied] = useState(false);
  if (!value || value === "—") return <span className="text-muted-foreground">—</span>;

  return (
    <button
      type="button"
      className="inline-flex max-w-[220px] items-center gap-2 rounded-md px-1 py-0.5 text-left font-mono text-xs text-primary hover:bg-primary/10"
      title={copied ? "Скопировано" : "Скопировать innerId"}
      onClick={(event) => {
        event.stopPropagation();
        onCopy(value, "innerId");
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1200);
      }}
    >
      <Copy className="h-3.5 w-3.5 shrink-0" />
      <span className="truncate">{copied ? "Скопировано" : value}</span>
    </button>
  );
}

function ReconciliationStatusHint({ row }: { row: TeamleadReconciliationTableRow }) {
  const stageTitle = uniqueStrings(row.stages.map(stageLabel)).join(", ");
  const triggerRef = useRef<HTMLSpanElement>(null);
  const [tooltip, setTooltip] = useState<{ left: number; top: number; placement: "above" | "below" } | null>(null);
  const showTooltip = () => {
    const rect = triggerRef.current?.getBoundingClientRect();
    if (!rect) return;
    const tooltipWidth = 320;
    const viewportWidth = window.innerWidth;
    const left = Math.min(Math.max(rect.left, 12), Math.max(12, viewportWidth - tooltipWidth - 12));
    const placement = rect.top < 180 ? "below" : "above";
    const top = placement === "below" ? rect.bottom + 8 : rect.top - 8;
    setTooltip({ left, top, placement });
  };

  return (
    <span className="relative inline-flex" onMouseEnter={showTooltip} onMouseLeave={() => setTooltip(null)}>
      <span
        ref={triggerRef}
        className={cn(
          "inline-flex items-center rounded-md border px-2 py-1 text-xs font-medium",
          row.severity === "info" && "border-slate-200 bg-slate-50 text-slate-700",
          row.severity === "warning" && "border-amber-200 bg-amber-50 text-amber-800",
          (row.severity === "error" || row.severity === "blocker") && "border-red-200 bg-red-50 text-red-700",
        )}
        aria-label={row.statusTitle}
      >
        {row.statusLabel}
      </span>
      {tooltip && typeof document !== "undefined"
        ? createPortal(
            <div
              className="pointer-events-none fixed z-[100] w-80 rounded-md border border-border bg-white p-3 text-xs text-foreground shadow-lg"
              style={{
                left: tooltip.left,
                top: tooltip.top,
                transform: tooltip.placement === "above" ? "translateY(-100%)" : undefined,
              }}
            >
              <span className="block font-semibold">{stageTitle}</span>
              <ul className="mt-2 space-y-1 text-muted-foreground">
                {row.reasons.map((reason) => (
                  <li key={reason} className="flex gap-2">
                    <span className="mt-1 h-1 w-1 shrink-0 rounded-full bg-current" />
                    <span>{reason}</span>
                  </li>
                ))}
              </ul>
            </div>,
            document.body,
          )
        : null}
    </span>
  );
}

function buildTeamleadItemRows(items: TeamleadReconciliationItem[]): TeamleadReconciliationTableRow[] {
  const groups = new Map<string, TeamleadReconciliationItem[]>();
  for (const item of items) {
    const csv = item.after ?? {};
    const crm = item.before ?? {};
    const innerId = item.externalInnerId ?? firstTableString(csv, crm, ["externalInnerId"]);
    const key = innerId ? `${item.direction}:${innerId}` : `item:${item.id}`;
    groups.set(key, [...(groups.get(key) ?? []), item]);
  }

  return Array.from(groups.entries()).map(([key, group]) => buildTeamleadItemGroupRow(key, group));
}

function buildTeamleadItemGroupRow(key: string, items: TeamleadReconciliationItem[]): TeamleadReconciliationTableRow {
  const csv = mergeTeamleadItemRecords(items.map((item) => item.after ?? {}));
  const crm = mergeTeamleadItemRecords(items.map((item) => item.before ?? {}));
  const worstItem = items.reduce((selected, item) => (severityRank(item.severity) > severityRank(selected.severity) ? item : selected), items[0]);
  const direction = worstItem.direction;
  const innerId = worstItem.externalInnerId ?? firstTableString(csv, crm, ["externalInnerId"]) ?? "—";
  const amountMinor = firstTableNumber(csv, ["amountMinor", "successAmountMinor", "transferAmountMinor", "paidAmountMinor"]);
  const methodName = firstTableString(csv, crm, ["methodName"]);
  const methodType = firstTableString(csv, crm, ["methodType"]);
  const bankCode = firstTableString(csv, crm, ["bankCode"]);
  const bankMethod = uniqueStrings([methodName, methodType, bankCode]).join(" / ") || "—";
  const requisiteId = firstTeamleadItemNumber(items, (item) => item.requisiteId) ?? firstTableNumber(csv, ["requisiteId"]) ?? firstTableNumber(crm, ["requisiteId"]);
  const requisite = buildTeamleadRequisiteCellData(csv, crm, requisiteId, direction);
  const reasons = uniqueStrings(items.map(issueComment));
  const stages = uniqueStrings(items.map((item) => item.stage));
  const statusTitle = reasons.join("\n");

  return {
    id: key,
    direction,
    stage: worstItem.stage,
    stages,
    severity: worstItem.severity,
    statusLabel: reconciliationStatusLabel(worstItem),
    statusTitle,
    innerId,
    trader: firstTableString(csv, crm, ["workerName", "traderLogin"]) ?? (worstItem.traderId ? `CRM трейдер #${worstItem.traderId}` : "—"),
    requisite,
    amountMinor,
    bankMethod,
    reasons,
    isBlocking: items.some((item) => item.isBlocking),
  };
}

function buildTeamleadRequisiteCellData(
  csv: Record<string, unknown>,
  crm: Record<string, unknown>,
  requisiteId?: number,
  direction: OrderDirection = "inbound",
): TeamleadReconciliationRequisiteCellData {
  const bankCode = firstTableString(csv, crm, ["bankCode"]);
  const bankName = firstTableString(csv, crm, ["bankName"]);
  const methodName = firstTableString(csv, crm, ["methodName"]);
  const methodType = firstTableString(csv, crm, ["methodType"]);
  const recipientName = firstTableString(csv, crm, ["recipientName", "holderName", "holder", "cardHolder", "receiverName", "receiver"]);
  const rawValue = firstTableString(csv, ["requisiteRaw", "requisite", "destinationRequisite"])
    ?? firstTableString(crm, ["requisiteRaw", "requisite", "destinationRequisite"]);
  const phoneValue = firstTableString(csv, crm, ["requisitePhone", "phone"]);
  const cardValue = firstLikelyCardValue([
    firstTableString(csv, ["requisiteRaw", "requisite", "destinationRequisite", "cardNumber"]),
    firstTableString(crm, ["requisiteRaw", "requisite", "destinationRequisite", "cardNumber"]),
  ]);

  if (cardValue) {
    return {
      displayValue: formatCardNumber(cardValue),
      rawValue: cardValue,
      kind: "card",
      mode: direction === "outbound" ? "csv_recipient" : "crm_requisite",
      requisiteId: direction === "outbound" ? undefined : requisiteId,
      bankCode,
      bankName,
      methodName,
      methodType,
      recipientName,
    };
  }

  const phoneCandidate = phoneValue ?? (isLikelyPhoneValue(rawValue) ? rawValue : undefined);
  if (phoneCandidate) {
    return {
      displayValue: formatRussianPhone(phoneCandidate),
      rawValue: phoneCandidate,
      kind: "phone",
      mode: direction === "outbound" ? "csv_recipient" : "crm_requisite",
      requisiteId: direction === "outbound" ? undefined : requisiteId,
      bankCode,
      bankName,
      methodName,
      methodType,
      recipientName,
    };
  }

  if (rawValue) {
    return {
      displayValue: rawValue,
      rawValue,
      kind: "raw",
      mode: direction === "outbound" ? "csv_recipient" : "crm_requisite",
      requisiteId: direction === "outbound" ? undefined : requisiteId,
      bankCode,
      bankName,
      methodName,
      methodType,
      recipientName,
    };
  }

  if (requisiteId) {
    return {
      displayValue: `CRM рек #${requisiteId}`,
      kind: "raw",
      mode: "crm_requisite",
      requisiteId,
      bankCode,
      bankName,
      methodName,
      methodType,
      recipientName,
    };
  }

  return {
    displayValue: "—",
    kind: "empty",
    mode: direction === "outbound" ? "csv_recipient" : "crm_requisite",
    bankCode,
    bankName,
    methodName,
    methodType,
    recipientName,
  };
}

function firstLikelyCardValue(values: Array<string | undefined>) {
  for (const value of values) {
    if (!value) continue;
    const digits = cardDigits(value);
    if (digits.length >= 13 && digits.length <= 19) return digits;
  }
  return undefined;
}

function isLikelyPhoneValue(value: string | undefined) {
  if (!value) return false;
  return /^\+?7\d{10}$|^8\d{10}$|^\d{10}$/.test(value.replace(/[\s().-]/g, ""));
}

function mergeTeamleadItemRecords(records: Record<string, unknown>[]) {
  const merged: Record<string, unknown> = {};
  for (const record of records) {
    for (const [key, value] of Object.entries(record)) {
      if (value === undefined || value === null || value === "") continue;
      if (merged[key] === undefined || merged[key] === null || merged[key] === "") {
        merged[key] = value;
      }
    }
  }
  return merged;
}

function severityRank(severity: TeamleadReconciliationItem["severity"]) {
  const ranks: Record<TeamleadReconciliationItem["severity"], number> = {
    info: 0,
    warning: 1,
    error: 2,
    blocker: 3,
  };
  return ranks[severity] ?? 0;
}

function reconciliationStatusLabel(item: TeamleadReconciliationItem) {
  if (item.isBlocking || item.severity === "blocker") return "Блокер";
  if (item.severity === "error") return "Ошибка";
  if (item.severity === "warning") return "Расхождение";
  return "Инфо";
}

function issueComment(item: TeamleadReconciliationItem) {
  const csv = item.after ?? {};
  const crm = item.before ?? {};
  const diff = amountDiffText(csv, crm);
  const labels: Record<string, string> = {
    unmatched_trader: "Трейдер из CSV не найден среди активных трейдеров CRM.",
    unmatched_requisite: "Реквизит из CSV не найден: банк, телефон и карта не совпали с реквизитами CRM.",
    bank_mismatch_requisite: "Реквизит найден по телефону/карте, но банк в CSV отличается от банка реквизита в CRM.",
    conflict_requisite: "Карта из CSV уже привязана к другому банку или телефону в CRM.",
    ambiguous_requisite: "CSV реквизит подходит к нескольким реквизитам CRM, нужен однозначный матчинг.",
    turnover_mismatch: `Оборот TL CSV не сходится с CRM за выбранный период.${diff ? ` ${diff}` : ""}`,
    missing_in_crm: "Транзакция есть в TL CSV, но не найдена в CRM.",
    missing_in_tl: "Транзакция есть в CRM, но отсутствует в TL CSV за выбранный период.",
    amount_changed: `Сумма транзакции отличается от TL CSV.${diff ? ` ${diff}` : ""}`,
    status_changed: "Статус транзакции отличается от TL CSV.",
    trader_changed: "Трейдер в CRM отличается от трейдера, указанного в TL CSV.",
    requisite_changed: "Реквизит в CRM отличается от реквизита, указанного в TL CSV.",
    date_changed: "Дата транзакции в CRM отличается от даты в TL CSV.",
    preview_summary: "Preview изменений рассчитан.",
  };
  return labels[item.issueType] ?? translateTeamleadMessage(item.message) ?? "Есть расхождение по строке сверки.";
}

function amountDiffText(csv: Record<string, unknown>, crm: Record<string, unknown>) {
  const tlAmount = firstTableNumber(csv, ["tlSuccessAmountMinor", "amountMinor", "successAmountMinor"]);
  const crmAmount = firstTableNumber(csv, ["crmAmountMinor"]) ?? firstTableNumber(crm, ["amountMinor", "successAmountMinor"]);
  if (tlAmount === undefined || crmAmount === undefined) return "";
  return `TL: ${formatMoneyMinor(tlAmount)}, CRM: ${formatMoneyMinor(crmAmount)}.`;
}

function translateTeamleadMessage(message?: string) {
  const translations: Record<string, string> = {
    "CSV workerName is not mapped to an active trader": "Трейдер из CSV не найден среди активных трейдеров CRM.",
    "CSV bank cannot be mapped to bank_code": "Банк из CSV не удалось сопоставить с bank_code.",
    "CSV bank, phone and card do not match any requisite": "Банк, телефон и карта из CSV не совпали ни с одним реквизитом CRM.",
    "CSV card matches CRM requisite with another bank": "Карта из CSV найдена в CRM, но у реквизита другой банк.",
    "CSV card matches several requisites in other banks": "Карта из CSV найдена у нескольких реквизитов в других банках.",
    "CSV phone matches CRM requisite with another bank": "Телефон из CSV найден в CRM, но у реквизита другой банк.",
    "CSV phone matches several requisites in other banks": "Телефон из CSV найден у нескольких реквизитов в других банках.",
    "CSV card points to another bank or phone": "Карта из CSV привязана к другому банку или телефону.",
    "CSV card matches several requisites": "Карта из CSV подходит к нескольким реквизитам CRM.",
    "CSV phone matches several requisite cards": "Телефон из CSV подходит к нескольким картам CRM.",
    "CSV requisite does not match CRM requisite identity": "Реквизит из CSV не совпал с идентичностью реквизита CRM.",
    "TL CSV success total differs from CRM turnover total": "Оборот TL CSV не сходится с CRM за выбранный период.",
    "TL CSV order is missing in CRM transactions": "Транзакция есть в TL CSV, но не найдена в CRM.",
    "CRM transaction is missing in TL CSV for selected period": "Транзакция есть в CRM, но отсутствует в TL CSV за выбранный период.",
    "CRM transaction differs from TL CSV": "Транзакция в CRM отличается от TL CSV.",
    "Preview changes calculated": "Preview изменений рассчитан.",
  };
  return message ? translations[message] ?? message : undefined;
}

function normalizeTeamleadPipeline(run: TeamleadReconciliationRun): TeamleadPipelineStepView[] {
  const values = Array.isArray(run.pipeline) ? run.pipeline : [];
  return values.flatMap((value, index) => {
    if (!value || typeof value !== "object") return [];
    const record = value as Record<string, unknown>;
    const stage = typeof record.stage === "string" ? record.stage : "";
    if (!stage) return [];
    const direction = isOrderDirection(record.direction) ? record.direction : undefined;
    return [{
      key: `${direction ?? "all"}:${stage}:${index}`,
      direction,
      stage,
      status: typeof record.status === "string" ? record.status : "matched",
      issuesCount: finiteNumber(record.issuesCount) ?? 0,
      facts: normalizePipelineFacts(record.facts),
    }];
  });
}

function buildTeamleadPipelineFlows(run: TeamleadReconciliationRun, stages: TeamleadPipelineStepView[]): TeamleadPipelineFlowView[] {
  const directedStages = stages.filter((stage) => stage.direction);
  if (!directedStages.length) {
    return stages.length ? [{ key: "all", label: "Пайплайн сверки", steps: stages }] : [];
  }

  const flows: TeamleadPipelineFlowView[] = [];
  if (run.inboundImportBatchId) {
    flows.push({
      key: "inbound",
      direction: "inbound",
      label: "Пайплайн входов",
      steps: directedStages.filter((stage) => stage.direction === "inbound"),
    });
  }
  if (run.outboundImportBatchId) {
    flows.push({
      key: "outbound",
      direction: "outbound",
      label: "Пайплайн выходов",
      steps: directedStages.filter((stage) => stage.direction === "outbound"),
    });
  }
  return flows.filter((flow) => flow.steps.length > 0);
}

function normalizePipelineFacts(value: unknown): TeamleadPipelineFactView[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((fact) => {
    if (!fact || typeof fact !== "object") return [];
    const record = fact as Record<string, unknown>;
    const label = typeof record.label === "string" ? record.label : "";
    if (!label) return [];
    return [{ label, value: record.value }];
  });
}

function pipelineStepStatus(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView) {
  if (step.stage === "preview") return "plan";
  return pipelineStepIsMismatch(run, step) ? "mismatch" : step.status;
}

function pipelineStepIssueCount(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView) {
  if (step.stage === "preview") return 0;
  if (step.issuesCount > 0) return step.issuesCount;
  return step.issuesCount;
}

function pipelineStepIsMismatch(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView) {
  if (step.stage === "preview") return false;
  return step.status === "mismatch" || pipelineStepIssueCount(run, step) > 0;
}

function pipelineStepCaption(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView, issuesCount: number) {
  if (step.stage === "preview") {
    const applyRowsCount = pipelineFactNumber(step, "К применению")
      ?? summaryNumber(step.direction ? directionPreview(run, step.direction) : undefined, "applyRowsCount")
      ?? summaryNumber(step.direction ? directionSummary(run, step.direction) : undefined, "applyRowsCount")
      ?? 0;
    return `${applyRowsCount} к применению`;
  }
  return pipelineStepIsMismatch(run, step) ? `${issuesCount} проблем` : "OK";
}

function pipelineFactNumber(step: TeamleadPipelineStepView, label: string) {
  const fact = step.facts.find((item) => item.label === label);
  return typeof fact?.value === "number" && Number.isFinite(fact.value) ? fact.value : undefined;
}

function stageDescription(stage: string, direction?: OrderDirection) {
  if (stage === "turnover_check" && direction === "outbound") {
    return "Сравниваем успешную сумму выплат из TL CSV с оборотом выплат, который трейдеры указали при закрытии смен.";
  }
  if (stage === "preview" && direction === "outbound") {
    return "Показываем, сколько строк выплатного CSV будет записано в базу при подтверждении. Получатели из CSV не создают и не обновляют CRM-реквизиты.";
  }

  const descriptions: Record<string, string> = {
    normalization: "Парсим CSV, приводим даты, суммы и статусы к внутреннему формату.",
    matching: "На этом этапе происходит сравнение и сопоставление транзакций из CSV и CRM.",
    turnover_check: "Сравниваем успешный оборот TL CSV с CRM-оборотом за выбранный период.",
    transaction_check: "Сверяем транзакции по innerId: наличие в CRM/TL CSV и отличия по сумме, статусу, трейдеру, реквизиту и дате.",
    preview: "Показываем план изменений после подтверждения сверки.",
    apply: "Фиксируем подтвержденную сверку в транзакциях и TL-статусах.",
  };
  return descriptions[stage] ?? "Этап сверки.";
}

function pipelineStepFacts(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView) {
  if (step.facts.length > 0) {
    const facts = step.facts.map((fact) => ({ label: fact.label, value: formatPipelineFactValue(fact) }));
    if (!facts.some((fact) => fact.label === "Отчет")) {
      return [pipelineFallbackReportFact(run, step), ...facts];
    }
    return facts;
  }

  const summary = step.direction ? directionSummary(run, step.direction) : run.inboundSummary;
  const preview = step.direction ? directionPreview(run, step.direction) : directionPreview(run, "inbound");
  if (step.stage === "turnover_check") {
    return [
      { label: "Сумма TL", value: summaryMoney(summary, "successAmountMinor") },
      { label: "Сумма CRM", value: summaryMoney(summary, "crmAmountMinor") },
      { label: "Разница", value: summaryMoney(summary, "diffAmountMinor") },
    ];
  }
  if (step.stage === "preview") {
    return [
      { label: "К применению", value: String(summaryNumber(preview, "applyRowsCount") ?? summaryNumber(summary, "applyRowsCount") ?? 0) },
      { label: "Создать / обновить / без изменений", value: previewTriplet(preview) },
      { label: "Блокеры", value: String(summaryNumber(preview, "blockedCount") ?? 0) },
    ];
  }
  return [
    { label: "Строк в периоде", value: String(summaryNumber(summary, "rowsInPeriod") ?? 0) },
    { label: "Успешных строк", value: String(summaryNumber(summary, "successCount") ?? 0) },
    { label: "Блокеры", value: String(summaryNumber(preview, "blockedCount") ?? 0) },
  ];
}

function pipelineFallbackReportFact(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView) {
  return { label: "Отчет", value: pipelineFallbackReportText(run, step) };
}

function pipelineFallbackReportText(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView) {
  const summary = step.direction ? directionSummary(run, step.direction) : undefined;
  const preview = step.direction ? directionPreview(run, step.direction) : undefined;
  if (step.stage === "matching") {
    const blockers = pipelineFactNumber(step, "Блокеров") ?? summaryNumber(preview, "blockedCount") ?? step.issuesCount;
    const rowsInPeriod = pipelineFactNumber(step, "Строк в периоде") ?? summaryNumber(summary, "rowsInPeriod") ?? 0;
    if (blockers <= 0) {
      return `В CSV ${rowsInPeriod} транзакций. Расхождений по матчингу нет.`;
    }
    return `В CSV ${rowsInPeriod} транзакций. Найдено ${blockers} расхождений.`;
  }
  if (step.stage === "transaction_check") {
    const rowsInPeriod = pipelineFactNumber(step, "Строк в периоде") ?? summaryNumber(summary, "rowsInPeriod") ?? 0;
    const issues = pipelineStepIssueCount(run, step);
    return issues > 0
      ? `В CSV ${rowsInPeriod} транзакций. Потранзакционная сверка нашла ${issues} расхождений.`
      : `В CSV ${rowsInPeriod} транзакций. Потранзакционная сверка прошла без расхождений.`;
  }
  if (step.stage === "preview") {
    const applyRowsCount = pipelineFactNumber(step, "К применению") ?? summaryNumber(preview, "applyRowsCount") ?? summaryNumber(summary, "applyRowsCount") ?? 0;
    const blockers = pipelineFactNumber(step, "Блокеры") ?? summaryNumber(preview, "blockedCount") ?? 0;
    if (blockers > 0) {
      return `Это план записи в базу, не ошибка этапа. Перед подтверждением осталось ${blockers} блокеров; после исправления будет применено ${applyRowsCount} строк.`;
    }
    return `Это план записи в базу, не расхождение. При подтверждении будет применено ${applyRowsCount} строк.`;
  }
  if (step.stage === "turnover_check") {
    const diff = pipelineFactNumber(step, "Разница") ?? summaryNumber(summary, "diffAmountMinor") ?? 0;
    return diff === 0
      ? "Обороты TL CSV и CRM за выбранный период сходятся."
      : "Обороты TL CSV и CRM за выбранный период не сходятся. Проверьте сумму TL, сумму CRM и разницу ниже.";
  }
  const rowsInPeriod = pipelineFactNumber(step, "Строк в периоде") ?? summaryNumber(summary, "rowsInPeriod") ?? 0;
  return `Проверено строк в периоде: ${rowsInPeriod}.`;
}

const pipelineTransactionPreviewFactLabels = new Set(["Создать", "Обновить", "Без изменений", "Создать / обновить / без изменений"]);

function buildPipelineStageReview(run: TeamleadReconciliationRun, step: TeamleadPipelineStepView, items: TeamleadReconciliationItem[]) {
  if (step.stage !== "matching" && step.stage !== "transaction_check") return null;
  const direction = step.direction;
  if (!direction) return null;

  const summary = directionSummary(run, direction);
  const rowsInPeriod = summaryNumber(summary, "rowsInPeriod") ?? pipelineFactNumber(step, "Строк в периоде") ?? 0;
  const stageItems = items.filter((item) => item.direction === direction && item.stage === step.stage && item.severity !== "info");
  const reasonCounts = aggregatePipelineReasonCounts(stageItems);
  const statusCounts = aggregatePipelineStatusCounts(stageItems);

  if (step.stage === "matching") {
    const blockers = stageItems.filter((item) => item.isBlocking || item.severity === "blocker").length;
    return {
      text: stageItems.length > 0
        ? `В CSV ${rowsInPeriod} транзакций. Найдено ${stageItems.length} расхождений, из них ${blockers} блокеров.`
        : `В CSV ${rowsInPeriod} транзакций. Расхождений по матчингу нет.`,
      statuses: statusCounts,
      reasons: reasonCounts,
    };
  }

  return {
    text: stageItems.length > 0
      ? `В CSV ${rowsInPeriod} транзакций. Потранзакционная сверка нашла ${stageItems.length} расхождений.`
      : `В CSV ${rowsInPeriod} транзакций. Потранзакционная сверка прошла без расхождений.`,
    statuses: statusCounts,
    reasons: reasonCounts,
  };
}

function aggregatePipelineReasonCounts(items: TeamleadReconciliationItem[]) {
  const counts = new Map<string, number>();
  for (const item of items) {
    const label = pipelineIssueReasonLabel(item);
    counts.set(label, (counts.get(label) ?? 0) + 1);
  }
  return Array.from(counts.entries())
    .map(([label, count]) => ({ label, count }))
    .sort((left, right) => right.count - left.count || left.label.localeCompare(right.label));
}

function aggregatePipelineStatusCounts(items: TeamleadReconciliationItem[]) {
  const byTransaction = new Map<string, TeamleadReconciliationItem>();
  for (const item of items) {
    const key = item.externalInnerId || `item:${item.id}`;
    const current = byTransaction.get(key);
    if (!current || severityRank(item.severity) > severityRank(current.severity)) {
      byTransaction.set(key, item);
    }
  }

  const counts = new Map<string, number>();
  for (const item of byTransaction.values()) {
    const label = reconciliationStatusLabel(item);
    counts.set(label, (counts.get(label) ?? 0) + 1);
  }
  return Array.from(counts.entries())
    .map(([label, count]) => ({ label, count }))
    .sort((left, right) => right.count - left.count || left.label.localeCompare(right.label));
}

function pipelineIssueReasonLabel(item: TeamleadReconciliationItem) {
  const labels: Record<string, string> = {
    unmatched_trader: "Трейдер из CSV не найден среди активных трейдеров CRM",
    unmatched_requisite: "Банк, телефон и карта из CSV не совпали ни с одним реквизитом CRM",
    bank_mismatch_requisite: "Банк в CSV отличается от банка реквизита в CRM",
    conflict_requisite: "Реквизит конфликтует с уже существующей связкой CRM",
    ambiguous_requisite: "CSV реквизит подходит к нескольким реквизитам CRM",
    turnover_mismatch: "Оборот TL CSV не сходится с CRM за выбранный период",
    missing_in_crm: "Транзакция есть в TL CSV, но не найдена в CRM",
    missing_in_tl: "Транзакция есть в CRM, но отсутствует в TL CSV",
    amount_changed: "Изменилась сумма",
    status_changed: "Изменился статус",
    trader_changed: "Изменился трейдер",
    requisite_changed: "Изменился реквизит",
    date_changed: "Изменилась дата",
  };
  return labels[item.issueType] ?? translateTeamleadMessage(item.message) ?? (item.issueType || "Другая причина");
}

function formatPipelineFactValue(fact: TeamleadPipelineFactView) {
  if (typeof fact.value === "number" && isMoneyPipelineFact(fact.label)) return formatMoneyMinor(fact.value);
  if (typeof fact.value === "number" && Number.isFinite(fact.value)) return String(fact.value);
  if (typeof fact.value === "string" && fact.value.trim() !== "") return fact.value;
  return "—";
}

function isPipelineReportFact(fact: { label: string; value: string }) {
  return fact.label === "Отчет" || fact.value.length > 48;
}

function isMoneyPipelineFact(label: string) {
  return /сумма|разница/i.test(label);
}

function directionSummary(run: TeamleadReconciliationRun, direction: OrderDirection) {
  return direction === "inbound" ? run.inboundSummary : run.outboundSummary;
}

function isOrderDirection(value: unknown): value is OrderDirection {
  return value === "inbound" || value === "outbound";
}

function finiteNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function summaryMoney(summary: Record<string, unknown> | undefined, key: string) {
  return formatMoneyMinor(summaryNumber(summary, key) ?? 0);
}

function previewTriplet(preview: Record<string, unknown> | undefined) {
  return `${summaryNumber(preview, "createCount") ?? 0} / ${summaryNumber(preview, "updateCount") ?? 0} / ${summaryNumber(preview, "unchangedCount") ?? 0}`;
}

function directionPreview(run: TeamleadReconciliationRun, direction: OrderDirection) {
  const preview = run.preview;
  if (!preview || typeof preview !== "object") return undefined;
  const value = preview[direction];
  return value && typeof value === "object" ? value as Record<string, unknown> : undefined;
}

function firstTableString(...args: [Record<string, unknown>, Record<string, unknown>, string[]] | [Record<string, unknown>, string[]]) {
  const values = args.length === 2 ? [args[0]] : [args[0], args[1]];
  const keys = args.length === 2 ? args[1] : args[2];
  for (const value of values) {
    for (const key of keys) {
      const candidate = value[key];
      if (typeof candidate === "string" && candidate.trim() !== "") return candidate;
      if (typeof candidate === "number" && Number.isFinite(candidate)) return String(candidate);
    }
  }
  return undefined;
}

function firstTableNumber(value: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const candidate = value[key];
    if (typeof candidate === "number" && Number.isFinite(candidate)) return candidate;
    if (typeof candidate === "string" && candidate.trim() !== "") {
      const parsed = Number(candidate);
      if (Number.isFinite(parsed)) return parsed;
    }
  }
  return undefined;
}

function firstTeamleadItemNumber(items: TeamleadReconciliationItem[], select: (item: TeamleadReconciliationItem) => number | undefined) {
  for (const item of items) {
    const value = select(item);
    if (typeof value === "number" && Number.isFinite(value)) return value;
  }
  return undefined;
}

function uniqueStrings(values: Array<string | undefined>) {
  const result: string[] = [];
  for (const value of values) {
    const normalized = value?.trim();
    if (!normalized || result.includes(normalized)) continue;
    result.push(normalized);
  }
  return result;
}

function summaryNumber(summary: Record<string, unknown> | undefined, key: string) {
  const value = summary?.[key];
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function stageLabel(stage: string) {
  const labels: Record<string, string> = {
    normalization: "Нормализация",
    matching: "Матчинг",
    turnover_check: "Проверка оборотов",
    transaction_check: "Потранзакционная сверка",
    preview: "План применения",
    apply: "Применение",
  };
  return labels[stage] ?? stage;
}

function TeamleadReconciliationStatusCard({
  inbound,
  outbound,
  issueCount,
  hasMismatch,
  isLoading,
}: {
  inbound?: ReconciliationSummary | null;
  outbound?: ReconciliationSummary | null;
  issueCount: number;
  hasMismatch: boolean;
  isLoading?: boolean;
}) {
  if (isLoading) return <EmptyState title="Загружаем сверку" />;
  if (!inbound && !outbound) {
    return (
      <Card>
        <CardContent className="p-4">
          <EmptyState
            title="Сверка не запускалась"
            description="Загрузите CSV входящих или выплат, затем нажмите «Начать сверку»."
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={hasMismatch || issueCount > 0 ? "border-amber-200 bg-amber-50" : "border-emerald-200 bg-emerald-50"}>
      <CardContent className="grid gap-4 p-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">Текущая сверка тимлида</span>
            {hasMismatch ? (
              <span className="inline-flex items-center gap-1 text-sm font-medium text-amber-900">
                <AlertTriangle className="h-4 w-4" />
                есть расхождения
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 text-sm font-medium text-emerald-800">
                <CheckCircle2 className="h-4 w-4" />
                без критичных расхождений
              </span>
            )}
          </div>
          <div className="text-sm text-muted-foreground">
            Импорт обновляет транзакции в базе по `innerId`. Сверка показывает отличия нового CSV от уже сохраненных trader-scope ордеров.
          </div>
        </div>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
          <StatusLine label="Входящие" summary={inbound} />
          <StatusLine label="Выплаты" summary={outbound} />
          <div className={issueCount ? "rounded-md border border-amber-200 bg-white/70 p-3" : "rounded-md border border-emerald-200 bg-white/70 p-3"}>
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm font-medium">Расхождения</span>
              <span className="text-sm font-semibold">{issueCount}</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function StatusLine({ label, summary }: { label: string; summary?: ReconciliationSummary | null }) {
  return (
    <div className={summary?.status === "mismatch" ? "rounded-md border border-amber-200 bg-white/70 p-3" : "rounded-md border border-emerald-200 bg-white/70 p-3"}>
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium">{label}</span>
        {summary ? <StatusBadge status={summary.status} /> : <span className="text-xs text-muted-foreground">нет запуска</span>}
      </div>
      <div className="mt-1 text-xs text-muted-foreground">{summary ? formatMoneyMinor(summary.diffMinor) : "CSV ещё не сверяли"}</div>
    </div>
  );
}

function TeamleadReconciliationResultsTabs({
  inboundSummary,
  outboundSummary,
  inboundItems,
  outboundItems,
  inboundLoading,
  outboundLoading,
}: {
  inboundSummary?: ReconciliationSummary | null;
  outboundSummary?: ReconciliationSummary | null;
  inboundItems: ReconciliationItem[];
  outboundItems: ReconciliationItem[];
  inboundLoading?: boolean;
  outboundLoading?: boolean;
}) {
  const [direction, setDirection] = useState<OrderDirection>("inbound");
  const isInbound = direction === "inbound";

  return (
    <div className="space-y-3">
      <TeamleadDirectionTabs
        value={direction}
        onChange={setDirection}
        inboundSummary={inboundSummary}
        outboundSummary={outboundSummary}
        inboundIssueCount={inboundItems.length}
        outboundIssueCount={outboundItems.length}
      />
      <TeamleadReconciliationResult
        title={isInbound ? "Входы" : "Выходы"}
        direction={direction}
        summary={isInbound ? inboundSummary : outboundSummary}
        items={isInbound ? inboundItems : outboundItems}
        isLoading={isInbound ? inboundLoading : outboundLoading}
      />
    </div>
  );
}

function TeamleadDirectionTabs({
  value,
  onChange,
  inboundSummary,
  outboundSummary,
  inboundIssueCount,
  outboundIssueCount,
}: {
  value: OrderDirection;
  onChange: (value: OrderDirection) => void;
  inboundSummary?: ReconciliationSummary | null;
  outboundSummary?: ReconciliationSummary | null;
  inboundIssueCount: number;
  outboundIssueCount: number;
}) {
  const tabs = [
    { value: "inbound" as const, label: "Входы", icon: ArrowDownLeft, summary: inboundSummary, count: inboundIssueCount },
    { value: "outbound" as const, label: "Выходы", icon: ArrowUpRight, summary: outboundSummary, count: outboundIssueCount },
  ];

  return (
    <div className="inline-flex rounded-lg border border-border bg-white p-1">
      {tabs.map((tab) => {
        const Icon = tab.icon;
        const isActive = value === tab.value;
        return (
          <Button
            key={tab.value}
            type="button"
            variant={isActive ? "default" : "ghost"}
            size="sm"
            onClick={() => onChange(tab.value)}
          >
            <Icon className="h-4 w-4" />
            {tab.label}
            {tab.summary ? (
              <span className={cn("ml-1 rounded-sm px-1.5 py-0.5 text-xs", isActive ? "bg-white/20" : "bg-slate-100 text-muted-foreground")}>
                {tab.summary.status === "mismatch" ? tab.count : statusShortLabel(tab.summary.status)}
              </span>
            ) : null}
          </Button>
        );
      })}
    </div>
  );
}

function TeamleadReconciliationResult({
  title,
  direction,
  summary,
  items,
  isLoading,
}: {
  title: string;
  direction: OrderDirection;
  summary?: ReconciliationSummary | null;
  items: ReconciliationItem[];
  isLoading?: boolean;
}) {
  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-4">
          <EmptyState title={`${title}: загружаем сверку`} />
        </CardContent>
      </Card>
    );
  }

  if (!summary) {
    return (
      <Card>
        <CardContent className="p-4">
          <EmptyState title={`${title}: сверка не запускалась`} description="Загрузите CSV и нажмите «Начать сверку»." />
        </CardContent>
      </Card>
    );
  }

  const isMismatch = summary.status === "mismatch";

  return (
    <Card className={isMismatch ? "border-red-200" : summary.status === "accepted_with_comment" ? "border-amber-200" : "border-emerald-200"}>
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <div className="font-semibold">{title}</div>
              <StatusBadge status={summary.status} />
            </div>
            <div className="text-sm text-muted-foreground">CSV тимлида против сохраненных ордеров CRM</div>
          </div>
          {isMismatch && summary.runId ? (
            <AcceptMismatchDialog scope="teamlead" direction={direction} runId={summary.runId} />
          ) : null}
        </div>
        <div className="grid gap-2 text-sm sm:grid-cols-3">
          <ReconciliationAmountBox label="CSV тимлида" value={summary.expectedMinor} />
          <ReconciliationAmountBox label="CRM" value={summary.actualMinor} />
          <ReconciliationAmountBox label="Расхождение" value={summary.diffMinor} warning={summary.status !== "matched"} />
        </div>
        {summary.comment ? <div className="rounded-md border border-border/70 p-3 text-sm">Комментарий: {summary.comment}</div> : null}
        {summary.status === "matched" ? (
          <div className="rounded-md border border-emerald-200 bg-emerald-50 p-4 text-sm font-medium text-emerald-900">
            Расхождений нет.
          </div>
        ) : (
          <div className="space-y-3">
            <div className="text-sm font-medium">Реквизиты с расхождением</div>
            <TeamleadMismatchTable
              items={items}
              isLoading={isLoading}
              resetKey={`${direction}-${summary.runId ?? summary.status}`}
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ReconciliationAmountBox({ label, value, warning }: { label: string; value: number; warning?: boolean }) {
  return (
    <div className={warning ? "min-w-0 rounded-md border border-amber-200 bg-amber-50 px-3 py-2" : "min-w-0 rounded-md border border-border/70 bg-slate-50 px-3 py-2"}>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 break-words text-sm font-semibold tabular-nums">{formatMoneyMinor(value)}</div>
    </div>
  );
}

function TeamleadMismatchTable({
  items,
  isLoading,
  resetKey,
}: {
  items: ReconciliationItem[];
  isLoading?: boolean;
  resetKey?: string | number;
}) {
  const columns = useTeamleadMismatchColumns();
  const rows = useMemo(() => buildMismatchRows(items), [items]);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });

  useEffect(() => {
    setPagination({ pageIndex: 0, pageSize: 8 });
  }, [resetKey]);

  return (
    <DataTable
      columns={columns}
      data={rows}
      rowCount={rows.length}
      pagination={pagination}
      onPaginationChange={setPagination}
      isLoading={isLoading}
      emptyTitle="Расхождений по реквизитам нет"
      emptyDescription="Итоговая сверка расходится, но подробных строк по реквизитам не найдено."
      pageSizeOptions={[8, 15, 25, 50]}
      getRowClassName={(row) => (row.diffMinor && row.diffMinor !== 0 ? "bg-red-50 text-red-950 hover:bg-red-100" : undefined)}
    />
  );
}

function useTeamleadMismatchColumns() {
  return useMemo<ColumnDef<TeamleadMismatchRow>[]>(
    () => [
      {
        accessorKey: "issueLabel",
        header: "Тип",
        cell: ({ row }) => (
          <div>
            <div className="font-medium">{row.original.issueLabel}</div>
            <div className="text-xs text-muted-foreground">{row.original.innerId}</div>
          </div>
        ),
      },
      {
        accessorKey: "requisite",
        header: "Реквизит",
        cell: ({ row }) => <span className="font-medium">{row.original.requisite}</span>,
      },
      {
        accessorKey: "trader",
        header: "Трейдер",
        cell: ({ row }) => row.original.trader,
      },
      {
        accessorKey: "csvAmountMinor",
        header: "CSV тимлида",
        cell: ({ row }) => <MismatchAmountCell amount={row.original.csvAmountMinor} count={row.original.csvCount} status={row.original.csvStatus} />,
      },
      {
        accessorKey: "crmAmountMinor",
        header: "CRM",
        cell: ({ row }) => <MismatchAmountCell amount={row.original.crmAmountMinor} count={row.original.crmCount} status={row.original.crmStatus} />,
      },
      {
        accessorKey: "diffMinor",
        header: "Расхождение",
        cell: ({ row }) => (
          <div className="text-right font-semibold tabular-nums">
            {row.original.diffMinor === undefined ? "—" : formatMoneyMinor(row.original.diffMinor)}
          </div>
        ),
      },
      {
        accessorKey: "createdAt",
        header: "Дата",
        cell: ({ row }) => row.original.createdAt,
      },
    ],
    [],
  );
}

function MismatchAmountCell({ amount, count, status }: { amount?: number; count?: number; status?: string }) {
  return (
    <div className="min-w-[120px] text-right">
      <div className="font-medium tabular-nums">{amount === undefined ? "—" : formatMoneyMinor(amount)}</div>
      <div className="text-xs text-muted-foreground">
        {count !== undefined ? `${count} шт.` : status || "—"}
      </div>
    </div>
  );
}

function TeamleadReconciliationHistoryCard({
  onOpen,
}: {
  onOpen: (row: TeamleadReconciliationHistoryRow) => void;
}) {
  const [direction, setDirection] = useState<OrderDirection>("inbound");

  return (
    <Card>
      <CardContent className="space-y-3 p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="font-semibold">История сверок</div>
            <div className="text-sm text-muted-foreground">Каждый запуск сверки сохраняется отдельно вместе с результатом и комментарием.</div>
          </div>
          <div className="inline-flex rounded-lg border border-border bg-white p-1">
            <Button
              type="button"
              variant={direction === "inbound" ? "default" : "ghost"}
              size="sm"
              onClick={() => setDirection("inbound")}
            >
              <ArrowDownLeft className="h-4 w-4" />
              Входы
            </Button>
            <Button
              type="button"
              variant={direction === "outbound" ? "default" : "ghost"}
              size="sm"
              onClick={() => setDirection("outbound")}
            >
              <ArrowUpRight className="h-4 w-4" />
              Выходы
            </Button>
          </div>
        </div>
        {direction === "inbound" ? <InboundReconciliationHistoryTable onOpen={onOpen} /> : null}
        {direction === "outbound" ? <OutboundReconciliationHistoryTable onOpen={onOpen} /> : null}
      </CardContent>
    </Card>
  );
}

function InboundReconciliationHistoryTable({ onOpen }: { onOpen: (row: TeamleadReconciliationHistoryRow) => void }) {
  const columns = useTeamleadHistoryColumns();
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const historyQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationHistory("inbound", paginationToQuery(pagination)),
    queryFn: () => api.orders.reconciliationHistory("teamlead", "inbound", paginationToQuery(pagination)),
    placeholderData: keepPreviousData,
  });
  const rows = useMemo(
    () => buildHistoryRows("inbound", historyQuery.data?.items ?? []),
    [historyQuery.data?.items],
  );

  return (
    <DataTable
      columns={columns}
      data={rows}
      rowCount={historyQuery.data?.total ?? 0}
      pagination={pagination}
      onPaginationChange={setPagination}
      serverSidePagination
      isLoading={historyQuery.isLoading}
      isFetching={historyQuery.isFetching}
      emptyTitle="Истории сверок по входам пока нет"
      emptyDescription="После запуска сверки входов запись появится здесь."
      pageSizeOptions={[8, 15, 25, 50]}
      onRowClick={onOpen}
      actions={[{ label: "Открыть", onSelect: onOpen }]}
      getRowClassName={(row) => (row.summary.status === "mismatch" ? "bg-red-50 text-red-950 hover:bg-red-100" : undefined)}
    />
  );
}

function OutboundReconciliationHistoryTable({ onOpen }: { onOpen: (row: TeamleadReconciliationHistoryRow) => void }) {
  const columns = useTeamleadHistoryColumns();
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const historyQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationHistory("outbound", paginationToQuery(pagination)),
    queryFn: () => api.orders.reconciliationHistory("teamlead", "outbound", paginationToQuery(pagination)),
    placeholderData: keepPreviousData,
  });
  const rows = useMemo(
    () => buildHistoryRows("outbound", historyQuery.data?.items ?? []),
    [historyQuery.data?.items],
  );

  return (
    <DataTable
      columns={columns}
      data={rows}
      rowCount={historyQuery.data?.total ?? 0}
      pagination={pagination}
      onPaginationChange={setPagination}
      serverSidePagination
      isLoading={historyQuery.isLoading}
      isFetching={historyQuery.isFetching}
      emptyTitle="Истории сверок по выплатам пока нет"
      emptyDescription="После запуска сверки выплат запись появится здесь."
      pageSizeOptions={[8, 15, 25, 50]}
      onRowClick={onOpen}
      actions={[{ label: "Открыть", onSelect: onOpen }]}
      getRowClassName={(row) => (row.summary.status === "mismatch" ? "bg-red-50 text-red-950 hover:bg-red-100" : undefined)}
    />
  );
}

function useTeamleadHistoryColumns() {
  return useMemo<ColumnDef<TeamleadReconciliationHistoryRow>[]>(
    () => [
      {
        accessorKey: "id",
        header: "Сверка",
        cell: ({ row }) => (
          <span className="inline-flex items-center gap-2 font-medium">
            <Eye className="h-4 w-4 text-muted-foreground" />
            #{row.original.id}
          </span>
        ),
      },
      {
        accessorKey: "createdAt",
        header: "Запуск",
        cell: ({ row }) => formatDateTime(row.original.summary.createdAt),
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.summary.status} />,
      },
      {
        accessorKey: "expectedMinor",
        header: "CSV",
        cell: ({ row }) => <div className="text-right tabular-nums">{formatMoneyMinor(row.original.summary.expectedMinor)}</div>,
      },
      {
        accessorKey: "actualMinor",
        header: "CRM",
        cell: ({ row }) => <div className="text-right tabular-nums">{formatMoneyMinor(row.original.summary.actualMinor)}</div>,
      },
      {
        accessorKey: "diffMinor",
        header: "Расхождение",
        cell: ({ row }) => <div className="text-right font-semibold tabular-nums">{formatMoneyMinor(row.original.summary.diffMinor)}</div>,
      },
      {
        accessorKey: "comment",
        header: "Комментарий",
        cell: ({ row }) => (
          <span className="block max-w-[260px] truncate text-muted-foreground" title={row.original.summary.comment ?? ""}>
            {row.original.summary.comment || "—"}
          </span>
        ),
      },
    ],
    [],
  );
}

function TeamleadReconciliationHistoryDialog({
  selected,
  onClose,
}: {
  selected: SelectedHistoryRun | null;
  onClose: () => void;
}) {
  const runID = selected?.summary.runId;
  const direction = selected?.direction ?? "inbound";
  const runQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationRun(direction, runID),
    queryFn: () => api.orders.reconciliationRun("teamlead", direction, runID ?? 0),
    enabled: Boolean(runID),
  });
  const itemsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationRunItems(direction, runID, { onlyMismatch: true, page: 1, pageSize: 200 }),
    queryFn: () => api.orders.reconciliationRunItems("teamlead", direction, runID ?? 0, { onlyMismatch: true, page: 1, pageSize: 200 }),
    enabled: Boolean(runID),
  });
  const summary = runQuery.data ?? selected?.summary;

  return (
    <Dialog open={Boolean(selected)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-[1200px] p-6">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">
            Сверка #{runID ?? ""}: {directionLabel(direction)}
          </DialogTitle>
          <DialogDescription>
            {summary?.createdAt ? `Запущена ${formatDateTime(summary.createdAt)}` : "Сохраненный результат сверки."}
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-[76vh] overflow-y-auto pr-2">
          {selected ? (
            <TeamleadReconciliationResult
              title={directionLabel(direction)}
              direction={direction}
              summary={summary}
              items={itemsQuery.data?.items ?? []}
              isLoading={runQuery.isLoading || itemsQuery.isLoading}
            />
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function TeamleadReconciliationUploadDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const queryClient = useQueryClient();
  const [inboundFile, setInboundFile] = useState<File | null>(null);
  const [outboundFile, setOutboundFile] = useState<File | null>(null);
  const [toast, setToast] = useState<ToastMessage | null>(null);
  const showErrorToast = (title: string, message: string) => setToast({ id: Date.now(), title, message });

  const inboundDashboardQuery = useQuery({
    queryKey: queryKeys.teamlead.dashboard("inbound", { reconciliationDialog: true }),
    queryFn: () => api.orders.dashboard("teamlead", "inbound"),
    enabled: open,
  });
  const outboundDashboardQuery = useQuery({
    queryKey: queryKeys.teamlead.dashboard("outbound", { reconciliationDialog: true }),
    queryFn: () => api.orders.dashboard("teamlead", "outbound"),
    enabled: open,
  });
  const inboundReconciliationQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliation("inbound"),
    queryFn: () => api.orders.reconciliation("teamlead", "inbound"),
    enabled: open,
  });
  const outboundReconciliationQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliation("outbound"),
    queryFn: () => api.orders.reconciliation("teamlead", "outbound"),
    enabled: open,
  });
  const inboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationItems("inbound", { uploadDialog: true, onlyMismatch: true, page: 1, pageSize: 200 }),
    queryFn: () => api.orders.reconciliationItems("teamlead", "inbound", { onlyMismatch: true, page: 1, pageSize: 200 }),
    enabled: open,
  });
  const outboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationItems("outbound", { uploadDialog: true, onlyMismatch: true, page: 1, pageSize: 200 }),
    queryFn: () => api.orders.reconciliationItems("teamlead", "outbound", { onlyMismatch: true, page: 1, pageSize: 200 }),
    enabled: open,
  });

  const latestInboundImport = latestSavedImport(inboundDashboardQuery.data?.recentImports);
  const latestOutboundImport = latestSavedImport(outboundDashboardQuery.data?.recentImports);
  const hasFilesToUpload = Boolean(inboundFile || outboundFile);
  const hasUploadedCSV = Boolean(latestInboundImport || latestOutboundImport);

  const invalidateTeamleadReconciliation = async () => {
    await queryClient.invalidateQueries({ queryKey: ["teamlead"] });
  };

  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!inboundFile && !outboundFile) {
        throw new Error("Выберите CSV входящих или выплат");
      }
      if (inboundFile) {
        await api.imports.upload({ file: inboundFile, scope: "teamlead", direction: "inbound" });
      }
      if (outboundFile) {
        await api.imports.upload({ file: outboundFile, scope: "teamlead", direction: "outbound" });
      }
    },
    onSuccess: async () => {
      setToast(null);
      setInboundFile(null);
      setOutboundFile(null);
      await invalidateTeamleadReconciliation();
    },
    onError: (error) => showErrorToast("Не удалось загрузить CSV", error instanceof Error ? error.message : "Не удалось загрузить CSV"),
  });

  const startMutation = useMutation({
    mutationFn: async () => {
      const directions: OrderDirection[] = [];
      if (latestInboundImport) directions.push("inbound");
      if (latestOutboundImport) directions.push("outbound");
      if (!directions.length) {
        throw new Error("Сначала загрузите CSV входящих или выплат");
      }
      for (const direction of directions) {
        await api.teamleadReports.startReconciliation(direction);
      }
    },
    onSuccess: async () => {
      setToast(null);
      await invalidateTeamleadReconciliation();
    },
    onError: (error) => showErrorToast("Не удалось запустить сверку", error instanceof Error ? error.message : "Не удалось запустить сверку"),
  });

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-[1200px] p-6">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold">Загрузить сверку</DialogTitle>
            <DialogDescription>Загрузите CSV входящих, выплат или оба файла, затем запустите сверку.</DialogDescription>
          </DialogHeader>

          <div className="max-h-[76vh] space-y-5 overflow-y-auto pr-2">
            <Card className="border-blue-200 bg-blue-50">
              <CardContent className="p-3 text-sm text-blue-950">
                CSV тимлида обновляет существующие ордера по innerId. Если через апелляцию изменился статус или сумма, сверка покажет отличие от сохраненного trader-scope.
              </CardContent>
            </Card>

            <section className="space-y-3">
              <h2 className="text-sm font-semibold">1. Загрузка CSV</h2>
              <div className="grid gap-4 md:grid-cols-2">
                <ReportFileDropzone
                  label="Входы CSV"
                  help="Файл входящих ордеров из внешней админки."
                  selectedFile={inboundFile}
                  savedImport={latestInboundImport}
                  isLoading={inboundDashboardQuery.isLoading}
                  onFileChange={setInboundFile}
                />
                <ReportFileDropzone
                  label="Выходы CSV"
                  help="Файл выплат/выходов из внешней админки."
                  selectedFile={outboundFile}
                  savedImport={latestOutboundImport}
                  isLoading={outboundDashboardQuery.isLoading}
                  onFileChange={setOutboundFile}
                />
              </div>
              <div className="flex flex-wrap gap-2">
                <Button type="button" disabled={!hasFilesToUpload || uploadMutation.isPending} onClick={() => uploadMutation.mutate()}>
                  {uploadMutation.isPending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
                  Загрузить CSV
                </Button>
                <Button type="button" disabled={!hasUploadedCSV || hasFilesToUpload || startMutation.isPending} onClick={() => startMutation.mutate()}>
                  {startMutation.isPending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <CheckCircle2 className="h-4 w-4" />}
                  Начать сверку
                </Button>
              </div>
              {hasFilesToUpload ? (
                <div className="text-sm text-muted-foreground">Сначала загрузите выбранные файлы, затем станет доступен запуск сверки.</div>
              ) : null}
            </section>

            <section className="space-y-3">
              <h2 className="text-sm font-semibold">2. Результат сверки</h2>
              <TeamleadReconciliationResultsTabs
                inboundSummary={inboundReconciliationQuery.data}
                outboundSummary={outboundReconciliationQuery.data}
                inboundItems={inboundItemsQuery.data?.items ?? []}
                outboundItems={outboundItemsQuery.data?.items ?? []}
                inboundLoading={inboundReconciliationQuery.isLoading || inboundItemsQuery.isLoading}
                outboundLoading={outboundReconciliationQuery.isLoading || outboundItemsQuery.isLoading}
              />
            </section>
          </div>
        </DialogContent>
      </Dialog>
      <ErrorToast toast={toast} onClose={() => setToast(null)} />
    </>
  );
}

function ErrorToast({ toast, onClose }: { toast: ToastMessage | null; onClose: () => void }) {
  useEffect(() => {
    if (!toast) return;
    const timeoutID = window.setTimeout(onClose, 10_000);
    return () => window.clearTimeout(timeoutID);
  }, [toast, onClose]);

  if (!toast) return null;

  return (
    <div
      role="alert"
      className="fixed right-5 top-5 z-[100] flex w-[min(420px,calc(100vw-40px))] items-start gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 shadow-lg"
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-600" />
      <div className="min-w-0 flex-1">
        <div className="font-semibold">{toast.title}</div>
        <div className="mt-1 break-words text-red-800">{toast.message}</div>
      </div>
      <button
        type="button"
        className="rounded-sm p-0.5 text-red-700 hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
        aria-label="Закрыть уведомление"
        onClick={onClose}
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

function ReportFileDropzone({
  label,
  help,
  selectedFile,
  savedImport,
  isLoading,
  onFileChange,
}: {
  label: string;
  help: string;
  selectedFile: File | null;
  savedImport?: OrderImportHistoryItem;
  isLoading?: boolean;
  onFileChange: (file: File | null) => void;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const fileName = selectedFile?.name ?? savedImport?.fileName;
  const fileMeta = selectedFile
    ? formatFileSize(selectedFile.size)
    : savedImport
      ? `Загружен ${formatDateTime(savedImport.appliedAt ?? savedImport.createdAt)} · строк: ${savedImport.rowsCount}`
      : "";
  const statusText = selectedFile
    ? "Будет загружен перед запуском сверки"
    : savedImport
      ? `Загружен ${formatDateTime(savedImport.appliedAt ?? savedImport.createdAt)} · строк: ${savedImport.rowsCount}`
      : "Файл еще не загружен";

  const handleFile = (file?: File) => {
    if (!file) return;
    onFileChange(file);
  };

  const clearSelectedFile = () => {
    onFileChange(null);
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  return (
    <div className="space-y-3">
      <input
        ref={inputRef}
        type="file"
        accept=".csv,text/csv"
        className="sr-only"
        onChange={(event) => handleFile(event.target.files?.[0])}
      />

      <div className="space-y-1">
        <div className="text-sm font-semibold">{label}</div>
        <div className="text-xs text-muted-foreground">{help}</div>
      </div>

      {fileName ? (
        <div className="flex min-h-[74px] items-center gap-3 rounded-lg border border-border bg-slate-50 px-4 py-3">
          <FileText className="h-6 w-6 shrink-0 text-red-500" />
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold text-primary" title={fileName}>
              {isLoading ? "Проверяем сохраненный файл" : fileName}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">{fileMeta || statusText}</div>
          </div>
          {selectedFile ? (
            <button
              type="button"
              className="shrink-0 rounded-md p-1 text-muted-foreground hover:bg-red-50 hover:text-red-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
              aria-label="Убрать выбранный файл"
              title="Убрать выбранный файл"
              onClick={clearSelectedFile}
            >
              <X className="h-5 w-5" />
            </button>
          ) : null}
        </div>
      ) : null}

      <button
        type="button"
        onDragOver={(event) => {
          event.preventDefault();
          setIsDragging(true);
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={(event) => {
          event.preventDefault();
          setIsDragging(false);
          handleFile(event.dataTransfer.files?.[0]);
        }}
        className={cn(
          "flex min-h-[132px] w-full appearance-none flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-white px-4 py-6 text-center shadow-none outline-none transition hover:border-primary hover:bg-primary/5 focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20",
          isDragging ? "border-primary bg-primary/10" : undefined,
        )}
        onClick={() => {
          inputRef.current?.click();
        }}
      >
        <Upload className="h-8 w-8 text-muted-foreground" />
        <span className="max-w-[320px] text-sm font-medium text-muted-foreground">
          Перетащите CSV сюда или нажмите для выбора
        </span>
      </button>
    </div>
  );
}

function formatFileSize(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 Б";
  if (bytes < 1024) return `${bytes} Б`;
  const kilobytes = bytes / 1024;
  if (kilobytes < 1024) return `${kilobytes.toFixed(kilobytes >= 10 ? 0 : 1)} КБ`;
  const megabytes = kilobytes / 1024;
  return `${megabytes.toFixed(megabytes >= 10 ? 0 : 1)} МБ`;
}

function buildMismatchRows(items: ReconciliationItem[]): TeamleadMismatchRow[] {
  return items.map((item) => {
    const teamleadValue = item.teamleadValue ?? {};
    const traderValue = item.traderValue ?? {};
    const csvAmountMinor = firstNumber(teamleadValue, ["successAmountMinor", "amountMinor", "transferAmountMinor", "paidAmountMinor"]);
    const crmAmountMinor = firstNumber(traderValue, ["successAmountMinor", "amountMinor", "transferAmountMinor", "paidAmountMinor"]);
    const csvCount = firstNumber(teamleadValue, ["successCount", "count"]);
    const crmCount = firstNumber(traderValue, ["successCount", "count"]);
    const diffMinor = csvAmountMinor === undefined && crmAmountMinor === undefined
      ? undefined
      : (crmAmountMinor ?? 0) - (csvAmountMinor ?? 0);

    return {
      id: String(item.id),
      issueType: item.issueType,
      issueLabel: issueTypeLabel(item.issueType),
      requisite: firstString(
        teamleadValue,
        traderValue,
        ["requisitePhone", "phone", "requisite", "destinationRequisite", "bankCode"],
      ) ?? "Итог",
      trader: firstString(teamleadValue, traderValue, ["workerName", "traderLogin", "traderId"]) ?? "—",
      innerId: item.externalInnerId ? `innerId: ${item.externalInnerId}` : "—",
      csvAmountMinor,
      crmAmountMinor,
      diffMinor,
      csvCount,
      crmCount,
      csvStatus: firstString(teamleadValue, {}, ["rawStatus", "normalizedStatus"]) ?? "—",
      crmStatus: firstString(traderValue, {}, ["rawStatus", "normalizedStatus"]) ?? "—",
      createdAt: firstDate(teamleadValue, traderValue, ["createdAtExternal"]) ?? formatDateTime(item.createdAt),
    };
  });
}

function firstNumber(value: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const raw = value[key];
    if (typeof raw === "number" && Number.isFinite(raw)) return raw;
  }

  return undefined;
}

function firstString(
  primary: Record<string, unknown>,
  secondary: Record<string, unknown>,
  keys: string[],
) {
  for (const source of [primary, secondary]) {
    for (const key of keys) {
      const raw = source[key];
      if (raw === undefined || raw === null || raw === "") continue;
      return String(raw);
    }
  }

  return undefined;
}

function firstDate(
  primary: Record<string, unknown>,
  secondary: Record<string, unknown>,
  keys: string[],
) {
  for (const source of [primary, secondary]) {
    for (const key of keys) {
      const raw = source[key];
      if (typeof raw === "string" && raw.trim()) return formatDateTime(raw);
    }
  }

  return undefined;
}

function issueTypeLabel(issueType: string) {
  const labels: Record<string, string> = {
    total_mismatch: "Итог не сходится",
    total_amount_mismatch: "Итог не сходится",
    worker_amount_mismatch: "Сумма по трейдеру",
    requisite_amount_mismatch: "Сумма по реквизиту",
    order_mismatch: "Ордер не сходится",
    amount_mismatch: "Сумма ордера",
    status_mismatch: "Статус ордера",
    worker_mismatch: "Трейдер ордера",
    missing_in_trader_import: "Нет в CRM",
    extra_in_trader_import: "Лишний в CRM",
    payout_not_fully_paid: "Выплата не закрыта",
    missing_manual_payout_order: "Нет ручной выплаты",
    extra_manual_payout_order: "Лишняя ручная выплата",
    manual_payout_not_fully_paid: "Ручная выплата не закрыта",
    source_requisite_outbound_mismatch: "Выход по реквизиту",
  };

  return labels[issueType] ?? issueType;
}

function buildHistoryRows(direction: OrderDirection, items: ReconciliationSummary[]): TeamleadReconciliationHistoryRow[] {
  return items.map((summary) => ({
    id: summary.runId ?? 0,
    direction,
    summary,
  }));
}

function directionLabel(direction: OrderDirection) {
  return direction === "inbound" ? "Входы" : "Выходы";
}

function statusShortLabel(status: ReconciliationSummary["status"]) {
  if (status === "matched") return "ok";
  if (status === "accepted_with_comment") return "принято";
  return "!";
}

function latestSavedImport(items?: OrderImportHistoryItem[]) {
  return items?.find((item) => item.status === "applied" || item.status === "reconciled") ?? items?.[0];
}
