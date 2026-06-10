import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Upload } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { FormField } from "@/components/crm/form-field";
import { MoneyCell } from "@/components/crm/money-cell";
import { StatusBadge } from "@/components/crm/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { ImportResult, OrderDirection, ReconciliationSummary } from "@/lib/domain";
import { api } from "@/lib/api";
import { formatMoneyMinor } from "@/lib/utils";

const acceptSchema = z.object({
  comment: z.string().trim().min(1, "Комментарий обязателен для подтверждения расхождения"),
});

export function ImportCsvDialog({
  scopeLabel,
  scope,
  direction,
}: {
  scopeLabel: string;
  scope: "teamlead" | "trader";
  direction: OrderDirection;
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [isReimport, setIsReimport] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const uploadMutation = useMutation({
    mutationFn: api.imports.upload,
    onSuccess: async (nextResult) => {
      setResult(nextResult);
      await queryClient.invalidateQueries();
    },
    onError: (nextError) => setError(nextError instanceof Error ? nextError.message : "Не удалось импортировать CSV"),
  });

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button type="button" variant="outline">
            <Upload className="h-4 w-4" />
            Импорт CSV
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="text-base font-semibold">Импорт CSV</DialogTitle>
            <DialogDescription>{scopeLabel}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {isReimport ? (
              <Card className="border-amber-200 bg-amber-50">
                <CardContent className="p-3 text-sm text-amber-900">
                  Новый CSV заменит текущие активные данные в этой смене/периоде. История предыдущего импорта сохранится.
                </CardContent>
              </Card>
            ) : null}
            <FormField label="Файл CSV" help="CSV с разделителем | из внешней админки.">
              <Input
                type="file"
                accept=".csv,text/csv"
                onChange={(event) => setFile(event.target.files?.[0] ?? null)}
              />
            </FormField>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={isReimport}
                onChange={(event) => setIsReimport(event.target.checked)}
              />
              Это реимпорт активного scope
            </label>
            {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</div> : null}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Отмена
              </Button>
              <Button
                type="button"
                disabled={!file || uploadMutation.isPending}
                onClick={() => {
                  if (!file) return;
                  setError(null);
                  uploadMutation.mutate({ file, scope, direction });
                }}
              >
                Загрузить
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
      <ImportResultDialog result={result} onClose={() => result && setResult(null)} />
    </>
  );
}

export function ImportResultDialog({ result, onClose }: { result: ImportResult | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(result)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Результат импорта</DialogTitle>
          <DialogDescription>Итоги парсинга CSV и применения активного scope.</DialogDescription>
        </DialogHeader>
        {result ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <StatusBadge status={result.status} />
              <span className="text-sm text-muted-foreground">{result.importedRows} строк</span>
            </div>
            <div className="grid grid-cols-2 gap-3 text-sm">
              <SummaryCell label="Успешные" value={result.successCount} />
              <SummaryCell label="Неуспешные" value={result.failedCount} />
              <SummaryCell label="Дубликаты" value={result.duplicateCount} />
              <SummaryCell label="Diff" value={formatMoneyMinor(result.actualMinor - result.expectedMinor)} />
            </div>
            <div className="rounded-md border border-border p-3">
              <div className="grid grid-cols-3 gap-2 text-sm">
                <div>
                  <div className="text-xs text-muted-foreground">Ожидалось</div>
                  <MoneyCell valueMinor={result.expectedMinor} />
                </div>
                <div>
                  <div className="text-xs text-muted-foreground">Факт</div>
                  <MoneyCell valueMinor={result.actualMinor} />
                </div>
                <div>
                  <div className="text-xs text-muted-foreground">Разница</div>
                  <MoneyCell valueMinor={result.actualMinor - result.expectedMinor} />
                </div>
              </div>
            </div>
            {result.issues.length ? (
              <div className="space-y-2">
                <div className="text-sm font-medium">Ошибки строк</div>
                {result.issues.map((issue) => (
                  <div key={`${issue.row}-${issue.message}`} className="rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-800">
                    Строка {issue.row}: {issue.message}
                  </div>
                ))}
              </div>
            ) : null}
            <div className="flex justify-end">
              <Button type="button" onClick={onClose}>
                Закрыть
              </Button>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

export function MismatchAlert({ summary }: { summary: ReconciliationSummary }) {
  if (summary.status !== "mismatch") return null;

  return (
    <Card className="border-red-200 bg-red-50">
      <CardContent className="flex flex-wrap items-start justify-between gap-4 p-4 text-red-950">
        <div className="flex gap-3">
          <AlertTriangle className="mt-0.5 h-5 w-5 text-red-600" />
          <div>
            <div className="font-semibold">Есть расхождение сверки</div>
            <div className="text-sm text-red-800">
              Ожидалось {formatMoneyMinor(summary.expectedMinor)}, факт {formatMoneyMinor(summary.actualMinor)}.
            </div>
          </div>
        </div>
        <div className="text-right text-sm font-semibold">Diff: {formatMoneyMinor(summary.diffMinor)}</div>
      </CardContent>
    </Card>
  );
}

export function AcceptMismatchDialog({
  scope,
  direction,
  runId,
}: {
  scope: "teamlead" | "trader";
  direction: OrderDirection;
  runId: number;
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof acceptSchema>>({
    resolver: zodResolver(acceptSchema),
    defaultValues: { comment: "" },
  });
  const mutation = useMutation({
    mutationFn: api.imports.acceptMismatch,
    onSuccess: async () => {
      await queryClient.invalidateQueries();
      setOpen(false);
      form.reset();
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="destructive">
          Подтвердить расхождение
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Подтвердить расхождение</DialogTitle>
          <DialogDescription>Комментарий обязателен. После закрытия статус будет с расхождением.</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit((values) => mutation.mutate({ scope, direction, runId, comment: values.comment }))}>
          <FormField label="Комментарий" error={form.formState.errors.comment?.message}>
            <Textarea {...form.register("comment")} />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Отмена
            </Button>
            <Button type="submit" variant="destructive" disabled={mutation.isPending}>
              Подтвердить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function SummaryCell({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 font-semibold">{value}</div>
    </div>
  );
}
