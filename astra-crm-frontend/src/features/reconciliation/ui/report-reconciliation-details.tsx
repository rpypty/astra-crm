import type { ReconciliationItem, ReconciliationSummary } from "@/shared/model/domain";
import { formatMoneyMinor } from "@/shared/lib/utils";
import { StatusBadge } from "@/entities/status/ui/status-badge";

type ReportReconciliationDetailsProps = {
  title: string;
  summary?: ReconciliationSummary | null;
  items: ReconciliationItem[];
  isLoading?: boolean;
};

export function ReportReconciliationDetails({ title, summary, items, isLoading }: ReportReconciliationDetailsProps) {
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
    <section className={isProblem ? "rounded-md border border-amber-200 bg-amber-50 p-4" : "rounded-md border border-emerald-200 bg-emerald-50 p-4"}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="font-semibold">{title}</div>
          <div className="mt-1">
            <StatusBadge status={summary.status} />
          </div>
        </div>
        {summary.comment ? <div className="max-w-xl text-sm text-amber-950">Комментарий: {summary.comment}</div> : null}
      </div>

      <div className="mt-4 grid gap-2 text-sm sm:grid-cols-3">
        <AmountBox label="CSV" value={summary.expectedMinor} />
        <AmountBox label="CRM" value={summary.actualMinor} />
        <AmountBox label="Diff" value={summary.diffMinor} />
      </div>

      {items.length ? (
        <div className="mt-4 space-y-2">
          {items.map((item) => (
            <ReconciliationIssueItem key={item.id} item={item} />
          ))}
        </div>
      ) : isProblem ? (
        <TotalMismatchIssue summary={summary} />
      ) : null}
    </section>
  );
}

function AmountBox({ label, value }: { label: string; value: number }) {
  return (
    <div className="min-w-0 rounded-md border border-border/70 bg-white/70 p-3">
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
      {item.message ? <div className="mt-1 text-muted-foreground">{item.message}</div> : null}
      <div className="mt-2 grid gap-2 md:grid-cols-2">
        {item.teamleadValue ? <JsonValueBox label="CSV" value={item.teamleadValue} /> : null}
        {item.traderValue ? <JsonValueBox label="CRM" value={item.traderValue} /> : null}
      </div>
    </div>
  );
}

function TotalMismatchIssue({ summary }: { summary: ReconciliationSummary }) {
  return (
    <div className="mt-4 rounded-md border border-border/70 bg-white/80 p-3 text-sm">
      <div className="font-medium">Итоговая сумма не сходится</div>
      <div className="mt-2 grid gap-2 sm:grid-cols-3">
        <AmountBox label="CSV" value={summary.expectedMinor} />
        <AmountBox label="CRM" value={summary.actualMinor} />
        <AmountBox label="Diff" value={summary.diffMinor} />
      </div>
    </div>
  );
}

function JsonValueBox({ label, value }: { label: string; value: Record<string, unknown> }) {
  return (
    <div className="rounded-md bg-slate-50 p-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-1 break-words text-xs">{JSON.stringify(value)}</div>
    </div>
  );
}

function issueTypeLabel(issueType: string) {
  const labels: Record<string, string> = {
    payout_not_fully_paid: "Ручная выплата оплачена не полностью",
    missing_manual_payout_order: "Не найдена ручная выплата",
    manual_payout_not_fully_paid: "Ручная выплата оплачена не полностью",
    total_mismatch: "Итоговая сумма не сходится",
    total_amount_mismatch: "Итоговая сумма не сходится",
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
