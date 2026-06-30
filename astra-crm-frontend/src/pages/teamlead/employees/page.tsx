import { zodResolver } from "@hookform/resolvers/zod";
import { keepPreviousData, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { CalendarDays, Copy, Eye, FileText, History, Pencil, Plus, RefreshCw, UserRound, X } from "lucide-react";
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
import { paginationToQuery } from "@/shared/lib/pagination";
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
    externalWorkerName: z.string().min(1, "Введите alias сотрудника из CSV"),
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
const GENERATED_TRADER_PASSWORD_LENGTH = 14;
const GENERATED_TRADER_PASSWORD_CHARS = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%";

export function TeamleadTradersPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [status, setStatus] = useState("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [editingTrader, setEditingTrader] = useState<Trader | null>(null);
  const [archiveTrader, setArchiveTrader] = useState<Trader | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [generatedPassword, setGeneratedPassword] = useState<string | null>(null);

  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ search: deferredSearch, status, ...paginationToQuery(pagination) }),
    queryFn: () => api.traders.list({ search: deferredSearch, status, ...paginationToQuery(pagination) }),
    placeholderData: keepPreviousData,
  });

  useEffect(() => {
    setPagination((current) => (current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }));
  }, [deferredSearch, status]);

  const saveMutation = useMutation({
    mutationFn: api.traders.save,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["teamlead", "traders"] });
      setFormOpen(false);
      setEditingTrader(null);
    },
  });

  const resetPasswordMutation = useMutation({
    mutationFn: api.traders.resetPassword,
    onSuccess: (response) => setGeneratedPassword(response.password),
  });

  const archiveMutation = useMutation({
    mutationFn: api.traders.archive,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["teamlead", "traders"] }),
  });

  const columns = useMemo<ColumnDef<Trader>[]>(
    () => [
      {
        accessorKey: "login",
        header: "Логин",
        cell: ({ row }) => <UserCell login={row.original.login} />,
      },
      {
        accessorKey: "salaryRateBps",
        header: "Ставка",
        cell: ({ row }) => <span className="tabular-nums">{bpsToPercent(row.original.salaryRateBps)}%</span>,
      },
      {
        accessorKey: "externalWorkerName",
        header: "Alias CSV",
      },
      {
        accessorKey: "assignedRequisitesCount",
        header: "Реквизиты",
      },
      {
        accessorKey: "currentShiftStatus",
        header: "Смена",
        cell: ({ row }) =>
          row.original.currentShiftStatus === "none" ? "—" : <StatusBadge status={row.original.currentShiftStatus} />,
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
    ],
    [],
  );

  const data = tradersQuery.data?.items ?? [];
  const openTraderForm = useCallback(
    (trader: Trader | null) => {
      saveMutation.reset();
      setEditingTrader(trader);
      setFormOpen(true);
    },
    [saveMutation],
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title="Сотрудники"
        description="Трейдеры, ставки, статусы и текущие смены."
        actions={
          <Button
            type="button"
            onClick={() => {
              openTraderForm(null);
            }}
          >
            <Plus className="h-4 w-4" />
            Добавить трейдера
          </Button>
        }
      />
      <DataTable
        columns={columns}
        data={data}
        rowCount={tradersQuery.data?.total ?? 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        serverSidePagination
        search={search}
        onSearchChange={setSearch}
        toolbarFilters={
          <Select className="w-40" value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="all">Все статусы</option>
            <option value="active">Активные</option>
            <option value="disabled">Отключенные</option>
          </Select>
        }
        isLoading={tradersQuery.isLoading}
        isFetching={tradersQuery.isFetching}
        error={tradersQuery.error instanceof Error ? tradersQuery.error.message : null}
        emptyTitle="Сотрудников пока нет"
        emptyDescription="Добавьте первого трейдера для работы со сменами."
        onRowClick={(row) => {
          openTraderForm(row);
        }}
        actions={[
          {
            label: "Редактировать",
            onSelect: (row) => {
              openTraderForm(row);
            },
          },
          {
            label: "Сбросить пароль",
            onSelect: (row) => {
              resetPasswordMutation.reset();
              resetPasswordMutation.mutate(row.id);
            },
          },
          { label: "Отключить", destructive: true, onSelect: (row) => setArchiveTrader(row) },
        ]}
      />
      <MutationErrorAlert error={resetPasswordMutation.error ?? archiveMutation.error} />
      <ConfirmActionDialog
        open={Boolean(archiveTrader)}
        onOpenChange={(open) => !open && setArchiveTrader(null)}
        title="Отключить трейдера?"
        description="Трейдер потеряет доступ к CRM. Действие будет записано в аудит."
        confirmText="Отключить"
        onConfirm={() => {
          if (archiveTrader) {
            archiveMutation.reset();
            archiveMutation.mutate(archiveTrader.id);
          }
          setArchiveTrader(null);
        }}
      />
      <TraderFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        trader={editingTrader}
        isSaving={saveMutation.isPending}
        error={saveMutation.error instanceof Error ? saveMutation.error.message : null}
        onSubmit={(values) =>
          saveMutation.mutate({
            id: values.id,
            login: editingTrader?.login ?? values.login,
            password: values.password,
            externalWorkerName: values.externalWorkerName,
            salaryRateBps: percentToBps(values.salaryPercent),
            status: values.status,
          })
        }
      />
      <GeneratedPasswordDialog password={generatedPassword} onClose={() => setGeneratedPassword(null)} />
    </div>
  );
}

function TraderFormDialog({
  open,
  onOpenChange,
  trader,
  isSaving,
  error,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  trader: Trader | null;
  isSaving: boolean;
  error: string | null;
  onSubmit: (values: TraderForm) => void;
}) {
  const form = useForm<TraderForm>({
    resolver: zodResolver(traderSchema),
    defaultValues: { login: "", password: "", externalWorkerName: "", salaryPercent: 0.5, status: "active" },
  });

  useEffect(() => {
    if (!open) return;

    form.reset(
      trader
        ? {
            id: trader.id,
            login: trader.login,
            password: "",
            externalWorkerName: trader.externalWorkerName,
            salaryPercent: bpsToPercent(trader.salaryRateBps),
            status: trader.status,
          }
        : {
            login: "",
            password: generateTraderPassword(),
            externalWorkerName: "",
            salaryPercent: 0.5,
            status: "active",
          },
    );
  }, [form, open, trader]);

  const regeneratePassword = useCallback(() => {
    form.setValue("password", generateTraderPassword(), { shouldDirty: true, shouldValidate: true });
  }, [form]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(520px,100vw)] translate-x-0 translate-y-0 rounded-none">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{trader ? "Редактировать трейдера" : "Добавить трейдера"}</DialogTitle>
          <DialogDescription>
            {trader
              ? "Логин и пароль на форме редактирования не меняются. Alias CSV можно обновить отдельно."
              : "Логин нужен для входа, alias CSV — для сопоставления строк отчета с сотрудником."}
          </DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
          <FormField label="Логин" error={form.formState.errors.login?.message}>
            <Input {...form.register("login")} readOnly={Boolean(trader)} className={trader ? "bg-muted" : undefined} />
          </FormField>
          <FormField
            label="Alias сотрудника в CSV"
            help="Значение из колонки workerName во внешнем отчете. CRM использует его, чтобы сопоставить CSV-строки с сотрудником."
            error={form.formState.errors.externalWorkerName?.message}
          >
            <Input {...form.register("externalWorkerName")} placeholder="Например: Bliss_OP2" />
          </FormField>
          {!trader ? (
            <FormField label="Пароль" error={form.formState.errors.password?.message}>
              <div className="flex gap-2">
                <Input type="text" autoComplete="new-password" {...form.register("password")} />
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label="Сгенерировать новый пароль"
                  title="Сгенерировать новый пароль"
                  onClick={regeneratePassword}
                >
                  <RefreshCw className="h-4 w-4" />
                </Button>
              </div>
            </FormField>
          ) : null}
          <FormField label="Ставка, %" error={form.formState.errors.salaryPercent?.message}>
            <Input type="number" step="0.01" {...form.register("salaryPercent")} />
          </FormField>
          <FormField label="Статус">
            <Select {...form.register("status")}>
              <option value="active">Активен</option>
              <option value="disabled">Отключен</option>
            </Select>
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" disabled={isSaving}>
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function MutationErrorAlert({ error }: { error: unknown }) {
  if (!error) return null;

  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
      {error instanceof Error ? error.message : "Не удалось выполнить действие"}
    </div>
  );
}

function generateTraderPassword() {
  const cryptoApi = globalThis.crypto;
  if (cryptoApi?.getRandomValues) {
    const values = new Uint32Array(GENERATED_TRADER_PASSWORD_LENGTH);
    cryptoApi.getRandomValues(values);
    return Array.from(values, (value) => GENERATED_TRADER_PASSWORD_CHARS[value % GENERATED_TRADER_PASSWORD_CHARS.length]).join("");
  }

  return Array.from({ length: GENERATED_TRADER_PASSWORD_LENGTH }, () => {
    return GENERATED_TRADER_PASSWORD_CHARS[Math.floor(Math.random() * GENERATED_TRADER_PASSWORD_CHARS.length)];
  }).join("");
}

function GeneratedPasswordDialog({ password, onClose }: { password: string | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(password)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Новый пароль</DialogTitle>
          <DialogDescription>Пароль показывается один раз. После закрытия он не будет доступен в интерфейсе.</DialogDescription>
        </DialogHeader>
        <div className="rounded-md border border-border bg-slate-50 p-3 font-mono text-sm">{password}</div>
        <Button type="button" onClick={() => void navigator.clipboard?.writeText(password ?? "")}>
          Скопировать
        </Button>
      </DialogContent>
    </Dialog>
  );
}

function ConfirmActionDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmText,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  confirmText: string;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm}>
            {confirmText}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
