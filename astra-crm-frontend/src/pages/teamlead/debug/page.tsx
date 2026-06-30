import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, FileSpreadsheet, RefreshCw, Upload } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { MoneyCell } from "@/entities/order/ui/money-cell";
import { api } from "@/shared/api/api";
import { queryKeys } from "@/shared/api/query-keys";
import type { DebugFinAllImportJob, DebugFinAllImportResult } from "@/shared/model/domain";
import { cn } from "@/shared/lib/utils";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { FormField } from "@/shared/ui/form-field";
import { SearchableSelect, type SearchableSelectOption } from "@/shared/ui/searchable-select";

export function TeamleadDebugPage() {
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [dryRun, setDryRun] = useState(true);
  const [bankCode, setBankCode] = useState("");
  const [jobId, setJobId] = useState<number | null>(null);
  const [invalidatedJobId, setInvalidatedJobId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const banksQuery = useQuery({
    queryKey: queryKeys.banks,
    queryFn: api.banks.list,
  });
  const jobQuery = useQuery({
    queryKey: ["debug", "fin-all-import-job", jobId],
    queryFn: () => api.debug.finAllImportJob(jobId ?? 0),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "queued" || status === "running" ? 1500 : false;
    },
  });
  const bankOptions: SearchableSelectOption[] = [
    { value: "", label: "Выберите банк" },
    ...(banksQuery.data ?? []).map((bank) => ({ value: bank.code, label: bank.name })),
  ];

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
    if (!bankCode && banksQuery.data?.[0]) {
      setBankCode(banksQuery.data[0].code);
    }
  }, [bankCode, banksQuery.data]);
  useEffect(() => {
    if (!job || job.status !== "succeeded" || job.dryRun || invalidatedJobId === job.id) {
      return;
    }
    setInvalidatedJobId(job.id);
    void queryClient.invalidateQueries();
  }, [invalidatedJobId, job, queryClient]);

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-normal text-foreground">Импорт отчета Fin_ALL</h1>
          <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
            Загрузка Excel-отчета Fin_ALL для восстановления исторических смен, реквизитов и оборотов команды.
            Сначала запустите проверку без записи, затем применяйте импорт после просмотра результата.
          </p>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <FileSpreadsheet className="h-4 w-4 text-primary" />
            Параметры импорта
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
            <FinAllFileDropzone selectedFile={file} disabled={isProcessing} onFileChange={setFile} />

            <div className="space-y-4">
              <FormField
                label="Банк для созданных реквизитов"
                help="Выберите банк, который будет записан у реквизитов из Fin_ALL. Значение применяется ко всем строкам файла, где импорт создает реквизиты."
              >
                <SearchableSelect
                  value={bankCode}
                  onValueChange={setBankCode}
                  options={bankOptions}
                  placeholder={banksQuery.isLoading ? "Загрузка банков..." : "Выберите банк"}
                  searchPlaceholder="Поиск банка..."
                  emptyText="Банк не найден"
                  disabled={banksQuery.isLoading || isProcessing}
                />
              </FormField>

              <label
                className={cn(
                  "flex gap-3 rounded-md border border-border bg-background p-3 text-sm shadow-sm transition hover:border-primary/50 hover:bg-primary/5",
                  isProcessing ? "cursor-not-allowed opacity-60" : "cursor-pointer",
                )}
              >
                <input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 accent-primary"
                  checked={dryRun}
                  disabled={isProcessing}
                  onChange={(event) => setDryRun(event.target.checked)}
                />
                <span className="space-y-1">
                  <span className="block font-medium">Только проверить файл, ничего не записывать</span>
                  <span className="block text-xs leading-5 text-muted-foreground">
                    Оставьте включенным для безопасного предпросмотра. Снимите галочку, чтобы применить импорт и записать данные в CRM.
                  </span>
                </span>
              </label>
            </div>
          </div>

          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</div> : null}

          <div className="flex justify-end">
            <Button
              type="button"
              disabled={!file || !bankCode || isProcessing}
              onClick={() => {
                if (!file || !bankCode) return;
                mutation.mutate({ file, dryRun, bankCode });
              }}
            >
              {isProcessing ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
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

function FinAllFileDropzone({
  selectedFile,
  disabled,
  onFileChange,
}: {
  selectedFile: File | null;
  disabled?: boolean;
  onFileChange: (file: File | null) => void;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [isDragging, setIsDragging] = useState(false);

  const handleFile = (file?: File) => {
    if (!file || disabled) return;
    onFileChange(file);
  };

  return (
    <div className="space-y-2">
      <div>
        <div className="text-sm font-medium">Excel-файл Fin_ALL</div>
        <div className="text-xs text-muted-foreground">
          Загрузите книгу .xlsx с листом Fin_ALL. Поддерживается выбор файла кнопкой или перетаскивание в область ниже.
        </div>
      </div>
      <button
        type="button"
        disabled={disabled}
        onClick={() => inputRef.current?.click()}
        onDragOver={(event) => {
          event.preventDefault();
          if (!disabled) setIsDragging(true);
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={(event) => {
          event.preventDefault();
          setIsDragging(false);
          handleFile(event.dataTransfer.files?.[0]);
        }}
        className={cn(
          "flex min-h-[148px] w-full items-center justify-between gap-4 rounded-md border border-dashed border-border bg-white p-4 text-left transition hover:border-primary hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-60",
          isDragging ? "border-primary bg-primary/10" : undefined,
        )}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          className="sr-only"
          disabled={disabled}
          onChange={(event) => handleFile(event.target.files?.[0])}
        />
        <span className="min-w-0 space-y-1">
          <span className="flex items-center gap-2 text-sm font-semibold">
            <FileSpreadsheet className="h-4 w-4 shrink-0 text-primary" />
            <span className="min-w-0 truncate">{selectedFile?.name ?? "Перетащите Excel сюда"}</span>
          </span>
          <span className="block text-xs leading-5 text-muted-foreground">
            {selectedFile
              ? "Файл выбран и будет отправлен после запуска импорта."
              : "Нужен Excel .xlsx с листом Fin_ALL. Сначала можно выполнить проверку без записи в CRM."}
          </span>
        </span>
        <span className="inline-flex shrink-0 items-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground">
          <Upload className="h-4 w-4" />
          {selectedFile ? "Заменить" : "Выбрать"}
        </span>
      </button>
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
            {result.dryRun ? "проверка без записи" : "импорт применен"}
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
