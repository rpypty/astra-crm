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
  AssignmentHistoryItem,
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
import { requisiteSchema, type RequisiteForm } from "./types";

export function RequisiteFormDialog({
  open,
  onOpenChange,
  requisite,
  editablePlan,
  banks,
  isSaving,
  error,
  onCreateAssignment,
  onEditAssignment,
  onCancelAssignment,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  requisite: Requisite | null;
  editablePlan?: RequisiteAssignmentWorkRow | null;
  banks: Bank[];
  isSaving: boolean;
  error?: string | null;
  onCreateAssignment?: () => void;
  onEditAssignment?: (plan: RequisiteAssignmentWorkRow) => void;
  onCancelAssignment?: (plan: RequisiteAssignmentWorkRow) => void;
  onSubmit: (values: RequisiteForm) => void;
}) {
  const formValues = useMemo<RequisiteForm>(
    () =>
      requisite
        ? {
            id: requisite.id,
            phone: formatRussianPhone(requisite.phone),
            bankCode: requisite.bankCode,
            proxy: requisite.proxy,
            employeeComment: requisite.employeeComment ?? "",
            status: requisite.status,
          }
        : {
            phone: "",
            bankCode: banks[0]?.code ?? "",
            proxy: "",
            employeeComment: "",
            status: "active",
          },
    [banks, requisite],
  );
  const form = useForm<RequisiteForm>({
    resolver: zodResolver(requisiteSchema),
    values: formValues,
  });
  const bankOptions = useMemo<SearchableSelectOption[]>(
    () => [
      { value: "", label: "Выберите банк" },
      ...banks.map((bank) => ({ value: bank.code, label: bank.name })),
    ],
    [banks],
  );
  const closeWithoutValidation = () => {
    form.reset(formValues);
    form.clearErrors();
    onOpenChange(false);
  };

  useEffect(() => {
    if (open) {
      form.reset(formValues);
      form.clearErrors();
    }
  }, [form, formValues, open]);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => (nextOpen ? onOpenChange(true) : closeWithoutValidation())}>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(560px,100vw)] translate-x-0 translate-y-0 rounded-none">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{requisite ? "Редактировать реквизит" : "Добавить реквизит"}</DialogTitle>
          <DialogDescription>ФИО и карта заполняются трейдером при первом взятии реквизита в работу.</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          <FormField label="Телефон" error={form.formState.errors.phone?.message}>
            <Input
              {...form.register("phone")}
              onBlur={(event) => form.setValue("phone", formatRussianPhone(event.target.value), { shouldValidate: true })}
            />
          </FormField>
          <FormField label="Банк" error={form.formState.errors.bankCode?.message}>
            <SearchableSelect
              value={form.watch("bankCode")}
              options={bankOptions}
              onValueChange={(value) => form.setValue("bankCode", value, { shouldDirty: true, shouldValidate: true })}
              placeholder="Выберите банк"
              searchPlaceholder="Найти банк"
            />
          </FormField>
          <FormField label="Proxy" error={form.formState.errors.proxy?.message}>
            <Input {...form.register("proxy")} />
          </FormField>
          <FormField label="Комментарий для сотрудника">
            <Textarea rows={3} {...form.register("employeeComment")} />
          </FormField>
          <FormField label="Статус">
            <Select {...form.register("status")}>
              <option value="active">Активен</option>
              <option value="archived">Архив</option>
            </Select>
          </FormField>
          {requisite ? (
            <RequisiteAssignmentSection
              requisite={requisite}
              editablePlan={editablePlan ?? null}
              onCreateAssignment={onCreateAssignment}
              onEditAssignment={onEditAssignment}
              onCancelAssignment={onCancelAssignment}
            />
          ) : null}
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
          {requisite ? <AssignmentHistoryDialog requisiteId={requisite.id} /> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onMouseDown={(event) => event.preventDefault()} onClick={closeWithoutValidation}>
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

function RequisiteAssignmentSection({
  requisite,
  editablePlan,
  onCreateAssignment,
  onEditAssignment,
  onCancelAssignment,
}: {
  requisite: Requisite;
  editablePlan: RequisiteAssignmentWorkRow | null;
  onCreateAssignment?: () => void;
  onEditAssignment?: (plan: RequisiteAssignmentWorkRow) => void;
  onCancelAssignment?: (plan: RequisiteAssignmentWorkRow) => void;
}) {
  return (
    <div className="rounded-md border border-border bg-muted/20 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <div className="text-sm font-medium">Назначение трейдера</div>
          {requisite.assignedTraderLogin ? (
            <div className="flex flex-wrap items-center gap-2">
              <UserCell login={requisite.assignedTraderLogin} />
              {requisite.assignmentStatus ? <StatusBadge status={requisite.assignmentStatus} /> : null}
            </div>
          ) : (
            <div className="text-sm text-muted-foreground">Трейдер не назначен</div>
          )}
        </div>
        {!requisite.assignedTraderLogin && requisite.status === "active" ? (
          <Button type="button" variant="outline" size="sm" onClick={onCreateAssignment}>
            Назначить трейдера
          </Button>
        ) : null}
      </div>
      {requisite.assignedTraderLogin ? (
        <div className="mt-3 grid gap-2 text-sm text-muted-foreground sm:grid-cols-3">
          <AssignmentMeta label="Дата" value={formatAssignmentDate(requisite.assignedForDate)} />
          <AssignmentMeta label="Лимит" value={formatMoneyMinor(requisite.targetTurnoverMinor)} />
          <AssignmentMeta label="Банк" value={requisite.bankName} />
        </div>
      ) : null}
      {editablePlan ? (
        <div className="mt-4 flex flex-wrap gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => onEditAssignment?.(editablePlan)}>
            Изменить назначение
          </Button>
          <Button type="button" variant="destructive" size="sm" onClick={() => onCancelAssignment?.(editablePlan)}>
            Снять назначение
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function AssignmentMeta({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs uppercase text-muted-foreground">{label}</div>
      <div className="truncate font-medium text-foreground">{value}</div>
    </div>
  );
}

function formatAssignmentDate(value: string | undefined) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit", year: "numeric" }).format(date);
}

export function RequisiteCommentDialog({
  requisite,
  isSaving,
  error,
  onClose,
  onSubmit,
}: {
  requisite: Requisite | null;
  isSaving: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (employeeComment: string) => void;
}) {
  const form = useForm<{ employeeComment: string }>({
    values: { employeeComment: requisite?.employeeComment ?? "" },
  });

  return (
    <Dialog open={Boolean(requisite)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Комментарий для сотрудника</DialogTitle>
          <DialogDescription>
            {requisite ? `${formatRussianPhone(requisite.phone)} · ${requisite.bankName}` : ""}
          </DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit((values) => onSubmit(values.employeeComment))}>
          <FormField label="Комментарий">
            <Textarea rows={4} autoFocus {...form.register("employeeComment")} />
          </FormField>
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>
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

function AssignmentHistoryDialog({ requisiteId }: { requisiteId: number }) {
  const historyQuery = useQuery({
    queryKey: ["teamlead", "requisites", requisiteId, "history"],
    queryFn: () => api.requisites.history(requisiteId),
    enabled: false,
  });
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button type="button" variant="outline" size="sm" onClick={() => void historyQuery.refetch()}>
          <History className="h-4 w-4" />
          История назначений
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">История назначений</DialogTitle>
        </DialogHeader>
        <AssignmentHistoryList items={historyQuery.data ?? []} isLoading={historyQuery.isLoading} />
      </DialogContent>
    </Dialog>
  );
}

export function AssignmentHistoryViewer({ requisite, onClose }: { requisite: Requisite | null; onClose: () => void }) {
  const historyQuery = useQuery({
    queryKey: ["teamlead", "requisites", requisite?.id, "history"],
    queryFn: () => api.requisites.history(requisite?.id ?? 0),
    enabled: Boolean(requisite),
  });

  return (
    <Dialog open={Boolean(requisite)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">История назначений</DialogTitle>
          <DialogDescription>{requisite?.phone}</DialogDescription>
        </DialogHeader>
        <AssignmentHistoryList items={historyQuery.data ?? []} isLoading={historyQuery.isLoading} />
      </DialogContent>
    </Dialog>
  );
}

function AssignmentHistoryList({
  items,
  isLoading,
}: {
  items: AssignmentHistoryItem[];
  isLoading: boolean;
}) {
  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ status: "active" }),
    queryFn: () => api.traders.list({ status: "active" }),
  });
  const tradersById = useMemo(
    () => new Map((tradersQuery.data ?? []).map((trader) => [trader.id, trader.login])),
    [tradersQuery.data],
  );

  return (
    <div className="space-y-2">
      {items.map((item) => (
        <AssignmentHistoryCard key={item.id} item={item} traderLogin={tradersById.get(item.traderId)} />
      ))}
      {!isLoading && items.length === 0 ? <EmptyState title="Истории пока нет" /> : null}
    </div>
  );
}

function AssignmentHistoryCard({
  item,
  traderLogin,
}: {
  item: AssignmentHistoryItem;
  traderLogin?: string;
}) {
  return (
    <Card>
      <CardContent className="space-y-3 p-3 text-sm">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div>
            <div className="font-medium">{item.wasReassign ? "Переназначение" : "Назначение"}</div>
            <div className="text-muted-foreground">
              <DateTimeCell value={item.changedAt} />
            </div>
          </div>
          <StatusBadge status={item.status} />
        </div>
        <div className="grid gap-2 sm:grid-cols-2">
          <AssignmentHistoryField label="Трейдер" value={traderLogin ?? `#${item.traderId}`} />
          <AssignmentHistoryField label="Дата работы" value={formatHistoryDate(item.assignedForDate)} />
          <AssignmentHistoryField label="Лимит" value={formatMoneyMinor(item.targetTurnoverMinor)} />
          <AssignmentHistoryField label="Изменил" value={`#${item.changedBy}`} />
          {item.startedAt ? <AssignmentHistoryField label="Взял в работу" value={formatDateTime(item.startedAt)} /> : null}
          {item.unassignedAt ? <AssignmentHistoryField label="Снят" value={formatDateTime(item.unassignedAt)} /> : null}
          {item.cancelledAt ? <AssignmentHistoryField label="Отменен" value={formatDateTime(item.cancelledAt)} /> : null}
          {item.completedAt ? <AssignmentHistoryField label="Завершен" value={formatDateTime(item.completedAt)} /> : null}
        </div>
        {item.comment ? (
          <div className="rounded-md bg-slate-50 px-2 py-1.5 text-muted-foreground">{item.comment}</div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function AssignmentHistoryField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs uppercase tracking-normal text-muted-foreground">{label}</div>
      <div className="font-medium">{value}</div>
    </div>
  );
}

function formatHistoryDate(value: string) {
  return new Intl.DateTimeFormat("ru-RU").format(new Date(value));
}

export function ConfirmActionDialog({
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
