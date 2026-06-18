import type { ReconciliationItem, ReconciliationSummary } from "@/shared/model/domain";
import { formatDateTime, formatMoneyMinor } from "@/shared/lib/utils";

type ReportReconciliationDetailsProps = {
  title: string;
  summary?: ReconciliationSummary | null;
  items?: ReconciliationItem[];
  isLoading?: boolean;
  csvLabel?: string;
  crmLabel?: string;
  diffLabel?: string;
};

export function ReportReconciliationDetails({
  title,
  summary,
  items = [],
  isLoading,
  csvLabel = "CSV",
  crmLabel = "CRM",
  diffLabel = "Diff",
}: ReportReconciliationDetailsProps) {
  if (isLoading) {
    return (
      <div className="rounded-md border border-border p-4 text-sm text-muted-foreground">
        Загружаем сверку: {title.toLowerCase()}
      </div>
    );
  }

  if (!summary) {
    return (
      <div className="rounded-md border border-border p-4 text-sm text-muted-foreground">
        {title}: сверка не запускалась.
      </div>
    );
  }

  const isProblem = summary.status === "mismatch" || summary.status === "accepted_with_comment";

  return (
    <section className="rounded-md border border-border bg-white p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold">{title}</div>
        </div>
        {summary.comment ? <div className="max-w-xl text-xs text-muted-foreground">Комментарий: {summary.comment}</div> : null}
      </div>

      <div className="mt-3 grid gap-2 text-sm sm:grid-cols-3">
        <AmountBox label={csvLabel} value={summary.expectedMinor} />
        <AmountBox label={crmLabel} value={summary.actualMinor} />
        <AmountBox label={diffLabel} value={summary.diffMinor} warning={isProblem} />
      </div>

      {items.length ? (
        <div className="mt-4 space-y-2">
          {items.map((item) => (
            <ReconciliationIssueItem key={item.id} item={item} />
          ))}
        </div>
      ) : null}
    </section>
  );
}

function AmountBox({ label, value, warning }: { label: string; value: number; warning?: boolean }) {
  return (
    <div className={warning ? "min-w-0 rounded-md border border-amber-200 bg-amber-50 px-3 py-2" : "min-w-0 rounded-md border border-border/70 bg-slate-50 px-3 py-2"}>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 break-words text-sm font-semibold tabular-nums">{formatMoneyMinor(value)}</div>
    </div>
  );
}

function ReconciliationIssueItem({ item }: { item: ReconciliationItem }) {
  return (
    <div className="rounded-md border border-border/70 bg-white/80 p-3 text-sm">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="font-medium">{issueTypeLabel(item.issueType)}</span>
        {item.externalInnerId ? <span className="text-xs text-muted-foreground">innerId: {item.externalInnerId}</span> : null}
      </div>
      <div className="mt-1 text-xs text-muted-foreground">{issueTypeDescription(item.issueType)}</div>
      <div className="mt-2 grid gap-2 md:grid-cols-2">
        {item.teamleadValue ? <ValueBox label="CSV тимлида" value={item.teamleadValue} /> : null}
        {item.traderValue ? <ValueBox label="CRM / трейдеры" value={item.traderValue} /> : null}
      </div>
    </div>
  );
}

function ValueBox({ label, value }: { label: string; value: Record<string, unknown> }) {
  const rows = valueRows(value);

  return (
    <div className="rounded-md bg-slate-50 p-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-2 grid gap-1">
        {rows.map((row) => (
          <div key={row.label} className="flex min-w-0 items-center justify-between gap-3 text-xs">
            <span className="shrink-0 text-muted-foreground">{row.label}</span>
            <span className="min-w-0 truncate text-right font-medium tabular-nums" title={row.value}>
              {row.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function valueRows(value: Record<string, unknown>) {
  const rows = [
    valueRow(value, "workerName", "Трейдер"),
    valueRow(value, "traderLogin", "Логин"),
    valueRow(value, "traderId", "ID трейдера"),
    valueRow(value, "shiftRequisiteId", "Рек в смене"),
    valueRow(value, "requisitePhone", "Реквизит"),
    valueRow(value, "phone", "Реквизит"),
    valueRow(value, "destinationRequisite", "Получатель"),
    valueRow(value, "bankName", "Банк"),
    valueRow(value, "destinationBank", "Банк"),
    valueRow(value, "amountMinor", "Сумма", "money"),
    valueRow(value, "closedOutboundTurnoverMinor", "Закрытый выход", "money"),
    valueRow(value, "transferAmountMinor", "Переводы", "money"),
    valueRow(value, "diffAmountMinor", "Расхождение", "money"),
    valueRow(value, "successAmountMinor", "Успешно", "money"),
    valueRow(value, "paidAmountMinor", "Оплачено", "money"),
    valueRow(value, "remainingAmountMinor", "Осталось", "money"),
    valueRow(value, "successCount", "Кол-во"),
    valueRow(value, "rawStatus", "Статус CSV"),
    valueRow(value, "normalizedStatus", "Статус"),
    valueRow(value, "createdAtExternal", "Дата", "date"),
    valueRow(value, "manualPayoutOrderId", "Выплата"),
  ].filter(Boolean) as Array<{ label: string; value: string }>;

  return rows.length ? rows : [{ label: "Данные", value: "Нет отображаемых полей" }];
}

function valueRow(value: Record<string, unknown>, key: string, label: string, format?: "money" | "date") {
  const raw = value[key];
  if (raw === undefined || raw === null || raw === "") return null;

  if (format === "money" && typeof raw === "number") {
    return { label, value: formatMoneyMinor(raw) };
  }
  if (format === "date" && (typeof raw === "string" || raw instanceof Date)) {
    return { label, value: formatDateTime(raw) };
  }

  return { label, value: String(raw) };
}

function issueTypeLabel(issueType: string) {
  const labels: Record<string, string> = {
    payout_not_fully_paid: "Ручная выплата оплачена не полностью",
    missing_manual_payout_order: "Не найдена ручная выплата",
    extra_manual_payout_order: "Лишняя ручная выплата",
    manual_payout_not_fully_paid: "Ручная выплата оплачена не полностью",
    source_requisite_outbound_mismatch: "Оборот выплат по реквизиту не сходится",
    total_mismatch: "Итоговая сумма не сходится",
    total_amount_mismatch: "Итоговая сумма не сходится",
    worker_amount_mismatch: "Сумма по трейдеру не сходится",
    requisite_amount_mismatch: "Сумма по реквизиту не сходится",
    order_mismatch: "Ордер не сходится",
    amount_mismatch: "Сумма ордера не сходится",
    status_mismatch: "Статус ордера не сходится",
    worker_mismatch: "Трейдер ордера не сходится",
    missing_in_trader_import: "Нет в импорте трейдера",
    extra_in_trader_import: "Лишний ордер в импорте трейдера",
  };

  return labels[issueType] ?? issueType;
}

function issueTypeDescription(issueType: string) {
  const descriptions: Record<string, string> = {
    payout_not_fully_paid: "Ручная выплата есть, но сумма переводов не закрывает ее полностью.",
    missing_manual_payout_order: "В CSV есть выплата, но в CRM нет ручной выплаты с такой же суммой.",
    extra_manual_payout_order: "В CRM есть ручная выплата, но в CSV нет успешной выплаты с такой же суммой.",
    manual_payout_not_fully_paid: "Ручная выплата не полностью закрыта переводами.",
    source_requisite_outbound_mismatch: "Исходящий оборот, указанный при закрытии реквизита, отличается от суммы частичных переводов из этого реквизита.",
    total_mismatch: "Итог по CSV и CRM отличается.",
    total_amount_mismatch: "Итог по CSV и CRM отличается.",
    worker_amount_mismatch: "Сумма или количество успешных операций по трейдеру отличается.",
    requisite_amount_mismatch: "Сумма или количество по реквизиту отличается.",
    order_mismatch: "Один и тот же ордер отличается между CSV и CRM.",
    amount_mismatch: "Один и тот же innerId имеет разную сумму.",
    status_mismatch: "Один и тот же innerId имеет разный статус.",
    worker_mismatch: "Один и тот же innerId относится к разным трейдерам.",
    missing_in_trader_import: "Ордер есть в CSV тимлида, но его нет в активных импортах трейдеров.",
    extra_in_trader_import: "Ордер есть в импортах трейдеров, но отсутствует в CSV тимлида.",
  };

  return descriptions[issueType] ?? "Требует проверки.";
}
