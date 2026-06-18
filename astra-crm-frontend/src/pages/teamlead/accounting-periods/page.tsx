import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertTriangle, ArrowDownLeft, ArrowUpRight, CheckCircle2, Eye, FileText, RefreshCw, Upload, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { AcceptMismatchDialog } from "@/features/import-csv/ui/import-components";
import { StatusBadge } from "@/entities/status/ui/status-badge";
import { PageHeader } from "@/shared/ui/page-header";
import { EmptyState } from "@/shared/ui/empty-state";
import { DataTable } from "@/shared/ui/data-table";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import type { OrderDirection, OrderImportHistoryItem, ReconciliationItem, ReconciliationSummary } from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { queryKeys } from "@/shared/api/query-keys";
import { cn, formatDateTime, formatMoneyMinor } from "@/shared/lib/utils";

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
  directionLabel: string;
  summary: ReconciliationSummary;
};

type SelectedHistoryRun = {
  direction: OrderDirection;
  summary: ReconciliationSummary;
};

export function TeamleadPeriodsPage() {
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [selectedHistoryRun, setSelectedHistoryRun] = useState<SelectedHistoryRun | null>(null);
  const inboundReconciliationQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliation("inbound"),
    queryFn: () => api.orders.reconciliation("teamlead", "inbound"),
  });
  const outboundReconciliationQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliation("outbound"),
    queryFn: () => api.orders.reconciliation("teamlead", "outbound"),
  });
  const inboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationItems("inbound"),
    queryFn: () => api.orders.reconciliationItems("teamlead", "inbound"),
  });
  const outboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationItems("outbound"),
    queryFn: () => api.orders.reconciliationItems("teamlead", "outbound"),
  });
  const inboundHistoryQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationHistory("inbound"),
    queryFn: () => api.orders.reconciliationHistory("teamlead", "inbound"),
  });
  const outboundHistoryQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationHistory("outbound"),
    queryFn: () => api.orders.reconciliationHistory("teamlead", "outbound"),
  });

  const hasMismatch =
    inboundReconciliationQuery.data?.status === "mismatch" ||
    outboundReconciliationQuery.data?.status === "mismatch";
  const issueCount = (inboundItemsQuery.data?.length ?? 0) + (outboundItemsQuery.data?.length ?? 0);
  const historyRows = useMemo(
    () => buildHistoryRows(inboundHistoryQuery.data ?? [], outboundHistoryQuery.data ?? []),
    [inboundHistoryQuery.data, outboundHistoryQuery.data],
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title="Сверка"
        description="CSV тимлида обновляет существующие ордера по innerId и показывает изменения статусов, сумм и назначений."
        actions={
          <Button type="button" onClick={() => setUploadDialogOpen(true)}>
            <FileText className="h-4 w-4" />
            Загрузить сверку
          </Button>
        }
      />

      <TeamleadReconciliationStatusCard
        inbound={inboundReconciliationQuery.data}
        outbound={outboundReconciliationQuery.data}
        issueCount={issueCount}
        hasMismatch={hasMismatch}
        isLoading={inboundReconciliationQuery.isLoading || outboundReconciliationQuery.isLoading}
      />

      <TeamleadReconciliationResultsTabs
        inboundSummary={inboundReconciliationQuery.data}
        outboundSummary={outboundReconciliationQuery.data}
        inboundItems={inboundItemsQuery.data ?? []}
        outboundItems={outboundItemsQuery.data ?? []}
        inboundLoading={inboundReconciliationQuery.isLoading || inboundItemsQuery.isLoading}
        outboundLoading={outboundReconciliationQuery.isLoading || outboundItemsQuery.isLoading}
      />

      <TeamleadReconciliationHistoryCard
        rows={historyRows}
        isLoading={inboundHistoryQuery.isLoading || outboundHistoryQuery.isLoading}
        onOpen={(row) => setSelectedHistoryRun({ direction: row.direction, summary: row.summary })}
      />

      <TeamleadReconciliationUploadDialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen} />
      <TeamleadReconciliationHistoryDialog
        selected={selectedHistoryRun}
        onClose={() => setSelectedHistoryRun(null)}
      />
    </div>
  );
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
  rows,
  isLoading,
  onOpen,
}: {
  rows: TeamleadReconciliationHistoryRow[];
  isLoading?: boolean;
  onOpen: (row: TeamleadReconciliationHistoryRow) => void;
}) {
  const columns = useTeamleadHistoryColumns();
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });

  return (
    <Card>
      <CardContent className="space-y-3 p-4">
        <div>
          <div className="font-semibold">История сверок</div>
          <div className="text-sm text-muted-foreground">Каждый запуск сверки сохраняется отдельно вместе с результатом и комментарием.</div>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          rowCount={rows.length}
          pagination={pagination}
          onPaginationChange={setPagination}
          isLoading={isLoading}
          emptyTitle="Истории сверок пока нет"
          emptyDescription="После запуска сверки запись появится здесь."
          pageSizeOptions={[8, 15, 25, 50]}
          onRowClick={onOpen}
          actions={[{ label: "Открыть", onSelect: onOpen }]}
          getRowClassName={(row) => (row.summary.status === "mismatch" ? "bg-red-50 text-red-950 hover:bg-red-100" : undefined)}
        />
      </CardContent>
    </Card>
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
        accessorKey: "directionLabel",
        header: "Тип",
        cell: ({ row }) => row.original.directionLabel,
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
    queryKey: queryKeys.teamlead.reconciliationRunItems(direction, runID),
    queryFn: () => api.orders.reconciliationRunItems("teamlead", direction, runID ?? 0),
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
              items={itemsQuery.data ?? []}
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
    queryKey: queryKeys.teamlead.reconciliationItems("inbound"),
    queryFn: () => api.orders.reconciliationItems("teamlead", "inbound"),
    enabled: open,
  });
  const outboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.reconciliationItems("outbound"),
    queryFn: () => api.orders.reconciliationItems("teamlead", "outbound"),
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
                inboundItems={inboundItemsQuery.data ?? []}
                outboundItems={outboundItemsQuery.data ?? []}
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
    <div
      onClick={() => inputRef.current?.click()}
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
        "flex min-h-[142px] w-full items-center justify-between gap-4 rounded-md border border-dashed border-border bg-white p-4 text-left transition hover:border-primary hover:bg-primary/5",
        isDragging ? "border-primary bg-primary/10" : undefined,
      )}
    >
      <input
        ref={inputRef}
        type="file"
        accept=".csv,text/csv"
        className="sr-only"
        onChange={(event) => handleFile(event.target.files?.[0])}
      />
      <span className="min-w-0 space-y-1">
        <span className="block text-sm font-medium">{label}</span>
        <span className="block text-xs text-muted-foreground">{help}</span>
        <span className="flex min-w-0 items-center gap-2 pt-2 text-sm font-semibold">
          <FileText className="h-4 w-4 text-primary" />
          <span className="min-w-0 truncate">{isLoading ? "Проверяем сохраненный файл" : fileName || "Перетащите CSV сюда"}</span>
          {selectedFile ? (
            <button
              type="button"
              className="shrink-0 rounded-sm p-0.5 text-muted-foreground hover:bg-red-50 hover:text-red-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
              aria-label="Убрать выбранный файл"
              title="Убрать выбранный файл"
              onClick={(event) => {
                event.stopPropagation();
                clearSelectedFile();
              }}
            >
              <X className="h-4 w-4" />
            </button>
          ) : null}
        </span>
        <span className="block text-xs text-muted-foreground">
          {fileName ? statusText : "Можно выбрать файл кнопкой или перетащить его в эту область."}
        </span>
      </span>
      <button
        type="button"
        className="inline-flex shrink-0 items-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground"
        onClick={(event) => {
          event.stopPropagation();
          inputRef.current?.click();
        }}
      >
        <Upload className="h-4 w-4" />
        {savedImport || selectedFile ? "Заменить" : "Выбрать"}
      </button>
    </div>
  );
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

function buildHistoryRows(
  inbound: ReconciliationSummary[],
  outbound: ReconciliationSummary[],
): TeamleadReconciliationHistoryRow[] {
  return [
    ...inbound.map((summary) => ({
      id: summary.runId ?? 0,
      direction: "inbound" as const,
      directionLabel: directionLabel("inbound"),
      summary,
    })),
    ...outbound.map((summary) => ({
      id: summary.runId ?? 0,
      direction: "outbound" as const,
      directionLabel: directionLabel("outbound"),
      summary,
    })),
  ].sort((left, right) => {
    const rightTime = right.summary.createdAt ? new Date(right.summary.createdAt).getTime() : 0;
    const leftTime = left.summary.createdAt ? new Date(left.summary.createdAt).getTime() : 0;
    return rightTime - leftTime || right.id - left.id;
  });
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
