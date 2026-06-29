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
import { planSchema, type PlanForm } from "./types";

export function PlanRequisiteDialog({
  open,
  onOpenChange,
  plan,
  initialRequisiteId,
  requisites,
  traders,
  isSaving,
  error,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  plan: RequisiteAssignmentWorkRow | null;
  initialRequisiteId?: number | null;
  requisites: Requisite[];
  traders: Trader[];
  isSaving: boolean;
  error?: string | null;
  onSubmit: (values: PlanForm) => void;
}) {
  const initialRequisite = initialRequisiteId
    ? requisites.find((requisite) => requisite.id === initialRequisiteId)
    : undefined;
  const formValues = useMemo<PlanForm>(
    () =>
      plan
        ? {
            assignmentId: plan.assignmentId,
            requisiteId: String(plan.requisiteId),
            traderId: String(plan.traderId),
            assignedForDate: toDateInputValue(plan.assignedForDate),
            targetTurnover: moneyMinorToInput(plan.targetTurnoverMinor),
            comment: plan.comment ?? "",
          }
        : {
            requisiteId: initialRequisite ? String(initialRequisite.id) : requisites[0]?.id ? String(requisites[0].id) : "",
            traderId: traders[0]?.id ? String(traders[0].id) : "",
            assignedForDate: tomorrowDateInputValue(),
            targetTurnover: "",
            comment: "",
          },
    [initialRequisite, plan, requisites, traders],
  );
  const form = useForm<PlanForm>({
    resolver: zodResolver(planSchema),
    values: formValues,
  });
  const requisiteOptions = useMemo<SearchableSelectOption[]>(
    () => [
      { value: "", label: "Выберите реквизит" },
      ...requisites.map((requisite) => ({
        value: String(requisite.id),
        label: `${formatRussianPhone(requisite.phone)} · ${requisite.bankName}`,
        searchText: `${requisite.proxy} ${requisite.holderName ?? ""} ${requisite.cardNumber ?? ""}`,
      })),
    ],
    [requisites],
  );
  const traderOptions = useMemo<SearchableSelectOption[]>(
    () => [
      { value: "", label: "Выберите трейдера" },
      ...traders.map((trader) => ({
        value: String(trader.id),
        label: trader.login,
        searchText: trader.externalWorkerName,
      })),
    ],
    [traders],
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
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">
            {plan ? "Редактировать план" : "Запланировать реквизит"}
          </DialogTitle>
          <DialogDescription>Назначение реквизита на дату с лимитом для трейдера.</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          <FormField label="Дата" error={form.formState.errors.assignedForDate?.message}>
            <Input type="date" {...form.register("assignedForDate")} />
          </FormField>
          <FormField label="Реквизит" error={form.formState.errors.requisiteId?.message}>
            <SearchableSelect
              value={form.watch("requisiteId")}
              options={requisiteOptions}
              onValueChange={(value) => form.setValue("requisiteId", value, { shouldDirty: true, shouldValidate: true })}
              placeholder="Выберите реквизит"
              searchPlaceholder="Найти реквизит"
            />
          </FormField>
          <FormField label="Трейдер" error={form.formState.errors.traderId?.message}>
            <SearchableSelect
              value={form.watch("traderId")}
              options={traderOptions}
              onValueChange={(value) => form.setValue("traderId", value, { shouldDirty: true, shouldValidate: true })}
              placeholder="Выберите трейдера"
              searchPlaceholder="Найти трейдера"
            />
          </FormField>
          <FormField label="Лимит" error={form.formState.errors.targetTurnover?.message}>
            <Input inputMode="decimal" placeholder="500000" {...form.register("targetTurnover")} />
          </FormField>
          <FormField label="Комментарий">
            <Textarea rows={3} {...form.register("comment")} />
          </FormField>
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
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

export function PlanEventsDialog({ plan, onClose }: { plan: RequisiteAssignmentWorkRow | null; onClose: () => void }) {
  const eventsQuery = useQuery({
    queryKey: queryKeys.teamlead.requisitePlanEvents(plan?.assignmentId),
    queryFn: () => api.requisites.planEvents(plan?.assignmentId ?? 0, { page: 1, pageSize: 100 }),
    enabled: Boolean(plan),
  });

  return (
    <Dialog open={Boolean(plan)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">История изменений плана</DialogTitle>
          <DialogDescription>
            {plan ? `${formatRussianPhone(plan.phone)} · ${plan.bankName} · ${formatDateOnly(plan.assignedForDate)}` : ""}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          {(eventsQuery.data?.items ?? []).map((event) => (
            <AssignmentEventItem key={event.id} event={event} />
          ))}
          {!eventsQuery.isLoading && eventsQuery.data?.items.length === 0 ? <EmptyState title="Истории пока нет" /> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

export async function invalidateRequisiteWorkQueries(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["teamlead", "requisites"] }),
    queryClient.invalidateQueries({ queryKey: queryKeys.teamlead.requisitePlans() }),
    queryClient.invalidateQueries({ queryKey: queryKeys.teamlead.requisiteActivity() }),
    queryClient.invalidateQueries({ queryKey: ["trader", "requisites"] }),
  ]);
}

export function assignmentProgressLabel(row: RequisiteAssignmentWorkRow) {
  if (row.targetTurnoverMinor <= 0) return "—";
  const percent = Math.round((row.inboundTurnoverMinor / row.targetTurnoverMinor) * 100);
  return `${percent}%`;
}

export function isPlanCancellable(row: RequisiteAssignmentWorkRow) {
  return row.status === "planned" || row.status === "assigned";
}

function AssignmentEventItem({ event }: { event: RequisiteAssignmentEvent }) {
  return (
    <Card>
      <CardContent className="space-y-1 p-3 text-sm">
        <div className="flex items-center justify-between gap-3">
          <span className="font-medium">{assignmentEventLabel(event.action)}</span>
          <DateTimeCell value={event.createdAt} />
        </div>
        {event.comment ? <div className="text-muted-foreground">{event.comment}</div> : null}
      </CardContent>
    </Card>
  );
}

export function assignmentEventLabel(action: string) {
  switch (action) {
    case "created":
      return "Создан";
    case "updated":
      return "Изменен";
    case "cancelled":
      return "Отменен";
    default:
      return action;
  }
}

function toDateInputValue(value: string) {
  return value.slice(0, 10);
}

function tomorrowDateInputValue() {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return date.toISOString().slice(0, 10);
}

export function formatDateOnly(value: string | null | undefined) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit", year: "numeric" }).format(date);
}

function moneyMinorToInput(value: number) {
  if (!value) return "";
  return String(value / 100);
}
