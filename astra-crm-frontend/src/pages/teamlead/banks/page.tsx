import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { Pencil } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { api } from "@/shared/api/api";
import { queryKeys } from "@/shared/api/query-keys";
import type { Bank } from "@/shared/model/domain";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { DataTable } from "@/shared/ui/data-table";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { FormField } from "@/shared/ui/form-field";
import { Input } from "@/shared/ui/input";
import { PageHeader } from "@/shared/ui/page-header";

export function TeamleadBanksPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [editingBank, setEditingBank] = useState<Bank | null>(null);

  const banksQuery = useQuery({
    queryKey: queryKeys.banks,
    queryFn: api.banks.list,
  });

  const saveMutation = useMutation({
    mutationFn: api.banks.updateAlias,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.banks });
      setEditingBank(null);
    },
  });

  const filteredBanks = useMemo(() => {
    const query = search.trim().toLowerCase();
    const banks = banksQuery.data ?? [];
    if (!query) return banks;
    return banks.filter((bank) => {
      return (
        bank.name.toLowerCase().includes(query) ||
        bank.code.toLowerCase().includes(query) ||
        (bank.csvAlias ?? "").toLowerCase().includes(query)
      );
    });
  }, [banksQuery.data, search]);

  useEffect(() => {
    setPagination((current) => (current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }));
  }, [search]);

  const columns = useMemo<ColumnDef<Bank>[]>(
    () => [
      {
        accessorKey: "name",
        header: "Банк",
        cell: ({ row }) => (
          <div className="min-w-0">
            <div className="font-medium">{row.original.name}</div>
            <div className="text-xs text-muted-foreground">{row.original.code}</div>
          </div>
        ),
      },
      {
        accessorKey: "csvAlias",
        header: "Alias в CSV",
        cell: ({ row }) =>
          row.original.csvAlias ? (
            <span className="font-mono text-sm">{row.original.csvAlias}</span>
          ) : (
            <Badge>Не задан</Badge>
          ),
      },
    ],
    [],
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title="Банки"
        description="Alias банка используется при загрузке сверки, чтобы сопоставить название из CSV с банком CRM."
      />
      <DataTable
        columns={columns}
        data={filteredBanks}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        isLoading={banksQuery.isLoading}
        isFetching={banksQuery.isFetching}
        error={banksQuery.error instanceof Error ? banksQuery.error.message : null}
        emptyTitle="Банки не найдены"
        emptyDescription="Измените поиск или проверьте справочник банков."
        onRowClick={(row) => {
          saveMutation.reset();
          setEditingBank(row);
        }}
        actions={[
          {
            label: "Редактировать alias",
            onSelect: (row) => {
              saveMutation.reset();
              setEditingBank(row);
            },
          },
        ]}
      />
      <BankAliasDialog
        bank={editingBank}
        isSaving={saveMutation.isPending}
        error={saveMutation.error instanceof Error ? saveMutation.error.message : null}
        onClose={() => setEditingBank(null)}
        onSubmit={(csvAlias) => {
          if (!editingBank) return;
          saveMutation.mutate({ code: editingBank.code, csvAlias });
        }}
      />
    </div>
  );
}

function BankAliasDialog({
  bank,
  isSaving,
  error,
  onClose,
  onSubmit,
}: {
  bank: Bank | null;
  isSaving: boolean;
  error: string | null;
  onClose: () => void;
  onSubmit: (csvAlias: string) => void;
}) {
  const [csvAlias, setCsvAlias] = useState("");

  useEffect(() => {
    setCsvAlias(bank?.csvAlias ?? "");
  }, [bank]);

  return (
    <Dialog open={Boolean(bank)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="w-[min(460px,calc(100vw-32px))]">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Alias банка</DialogTitle>
          <DialogDescription>{bank ? bank.name : "Банк"}</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit(csvAlias);
          }}
        >
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
          <FormField
            label="Alias банка в CSV"
            help="Точное название банка из CSV-отчета. При сверке CRM будет считать его этим банком."
          >
            <Input value={csvAlias} onChange={(event) => setCsvAlias(event.target.value)} placeholder="Например: Ozon Bank" />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>
              Отмена
            </Button>
            <Button type="submit" disabled={isSaving}>
              <Pencil className="h-4 w-4" />
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
