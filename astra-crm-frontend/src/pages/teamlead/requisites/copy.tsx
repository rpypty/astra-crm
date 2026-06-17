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

import type { CopyHandler } from "./types";
type RequisiteIdentity = {
  phone: string;
  cardNumber?: string;
  holderName?: string;
};

export function RequisitePhoneMenu({ item, onCopy }: { item: RequisiteIdentity; onCopy: CopyHandler }) {
  const formattedPhone = formatRussianPhone(item.phone);
  const phoneCopyValue = normalizeRussianPhone(item.phone);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          className="h-auto max-w-[190px] justify-start px-0 py-0 text-left text-sm font-medium hover:bg-transparent hover:text-primary"
          title="Показать данные реквизита"
          onClick={(event) => event.stopPropagation()}
        >
          <span className="truncate tabular-nums">{formattedPhone}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-80 p-2" onClick={(event) => event.stopPropagation()}>
        <DropdownMenuLabel className="px-1">Данные реквизита</DropdownMenuLabel>
        <div className="space-y-1">
          <CopyableMenuRow label="Телефон" value={formattedPhone} copyValue={phoneCopyValue} onCopy={onCopy} />
          <CopyableMenuRow
            label="Карта"
            value={formatCardNumber(item.cardNumber)}
            copyValue={item.cardNumber}
            onCopy={onCopy}
          />
          <CopyableMenuRow label="Держатель" value={item.holderName || "—"} copyValue={item.holderName} onCopy={onCopy} />
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function CopyableInlineValue({ label, value, onCopy }: { label: string; value?: string; onCopy: CopyHandler }) {
  return (
    <button
      type="button"
      className="max-w-[180px] truncate text-left text-sm text-muted-foreground hover:text-foreground"
      title={value || "—"}
      onClick={(event) => {
        event.stopPropagation();
        onCopy(value, label);
      }}
    >
      {value || "—"}
    </button>
  );
}

function CopyableMenuRow({
  label,
  value,
  copyValue,
  onCopy,
}: {
  label: string;
  value: string;
  copyValue?: string | null;
  onCopy: CopyHandler;
}) {
  return (
    <button
      type="button"
      className="flex w-full items-center justify-between gap-3 rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent"
      onClick={() => onCopy(copyValue, label)}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate font-medium">{value}</span>
    </button>
  );
}

export function CopyToast({ message }: { message: string | null }) {
  if (!message) return null;

  return (
    <div role="status" className="fixed bottom-5 right-5 z-50 rounded-md border border-border bg-slate-950 px-4 py-2 text-sm font-medium text-white shadow-lg">
      {message}
    </div>
  );
}
