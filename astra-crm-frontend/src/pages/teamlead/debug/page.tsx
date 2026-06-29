import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Bug, FileSpreadsheet, Upload } from "lucide-react";
import { useEffect, useState } from "react";
import { MoneyCell } from "@/entities/order/ui/money-cell";
import { api } from "@/shared/api/api";
import type { DebugFinAllImportJob, DebugFinAllImportResult } from "@/shared/model/domain";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { FormField } from "@/shared/ui/form-field";
import { Input } from "@/shared/ui/input";

export function TeamleadDebugPage() {
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [dryRun, setDryRun] = useState(true);
  const [jobId, setJobId] = useState<number | null>(null);
  const [invalidatedJobId, setInvalidatedJobId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const jobQuery = useQuery({
    queryKey: ["debug", "fin-all-import-job", jobId],
    queryFn: () => api.debug.finAllImportJob(jobId ?? 0),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "queued" || status === "running" ? 1500 : false;
    },
  });

  const mutation = useMutation({
    mutationFn: api.debug.importFinAll,
    onSuccess: (job) => {
      setJobId(job.id);
      setError(null);
    },
    onError: (nextError) => {
      setError(nextError instanceof Error ? nextError.message : "Не удалось импортировать Excel");
    },
  });
  const job = jobQuery.data;
  const result = job?.result ?? null;
  const isProcessing = mutation.isPending || (jobId !== null && !job) || job?.status === "queued" || job?.status === "running";
  useEffect(() => {
    if (!job || job.status !== "succeeded" || job.dryRun || invalidatedJobId === job.id) {
      return;
    }
    setInvalidatedJobId(job.id);
    void queryClient.invalidateQueries();
  }, [invalidatedJobId, job, queryClient]);

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-normal text-foreground">Дебаг</h1>
          <p className="mt-1 text-sm text-muted-foreground">Служебные операции тимлида</p>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <Bug className="h-4 w-4" />
            Импорт Fin_ALL
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px]">
            <FormField label="Excel">
              <Input
                type="file"
                accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                onChange={(event) => setFile(event.target.files?.[0] ?? null)}
              />
            </FormField>
            <label className="flex h-10 items-center gap-2 self-end rounded-md border border-border px-3 text-sm">
              <input
                type="checkbox"
                checked={dryRun}
                onChange={(event) => setDryRun(event.target.checked)}
              />
              Dry run
            </label>
          </div>

          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</div> : null}

          <div className="flex justify-end">
            <Button
              type="button"
              disabled={!file || isProcessing}
              onClick={() => {
                if (!file) return;
                mutation.mutate({ file, dryRun });
              }}
            >
              <Upload className="h-4 w-4" />
              {dryRun ? "Проверить" : "Импортировать"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {job ? <DebugImportJob job={job} /> : null}
      {result ? <DebugImportResult result={result} /> : null}
    </div>
  );
}

function DebugImportJob({ job }: { job: DebugFinAllImportJob }) {
  const statusText = {
    queued: "В очереди",
    running: "Обрабатывается",
    succeeded: "Готово",
    failed: "Ошибка",
  }[job.status];
  return (
    <Card>
      <CardContent className="flex flex-wrap items-center justify-between gap-3 p-4">
        <div>
          <div className="text-sm font-semibold">Job #{job.id}: {statusText}</div>
          <div className="mt-1 text-xs text-muted-foreground">{job.fileName}</div>
        </div>
        {job.status === "failed" ? (
          <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
            {job.errorMessage ?? "Импорт завершился ошибкой"}
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function DebugImportResult({ result }: { result: DebugFinAllImportResult }) {
  const stats = [
    { label: "Строки", value: result.parsedRows },
    { label: "Круги", value: result.parsedCircles },
    { label: "Импортировано", value: result.importedCircles },
    { label: "Пропущено", value: result.skippedExistingCircles },
    { label: "Трейдеры", value: result.createdTraders },
    { label: "Реквизиты", value: result.createdRequisites },
    { label: "Смены", value: result.createdShifts },
    { label: "Блоки", value: result.blockedRequisites },
  ];

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <FileSpreadsheet className="h-4 w-4" />
          Результат
          <span className="rounded-md border border-border px-2 py-0.5 text-xs font-medium text-muted-foreground">
            {result.dryRun ? "dry run" : "applied"}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {stats.map((item) => (
            <div key={item.label} className="rounded-md border border-border p-3">
              <div className="text-xs text-muted-foreground">{item.label}</div>
              <div className="mt-1 text-lg font-semibold">{item.value}</div>
            </div>
          ))}
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          <MoneyStat label="Приход" value={result.inboundTurnoverMinor} />
          <MoneyStat label="Выплаты" value={result.outboundTurnoverMinor} />
          <MoneyStat label="Остатки" value={result.closingBalanceMinor} />
        </div>

        {result.warnings.length ? (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium text-amber-900">
              <AlertTriangle className="h-4 w-4" />
              Предупреждения
            </div>
            <div className="max-h-64 overflow-auto rounded-md border border-amber-200 bg-amber-50">
              {result.warnings.slice(0, 80).map((warning, index) => (
                <div key={`${warning.row}-${warning.circle ?? 0}-${index}`} className="border-b border-amber-200 px-3 py-2 text-sm text-amber-950 last:border-b-0">
                  Строка {warning.row}
                  {warning.circle ? `, круг ${warning.circle}` : ""}: {warning.message}
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function MoneyStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-base font-semibold">
        <MoneyCell valueMinor={value} />
      </div>
    </div>
  );
}
