import {
  flexRender,
  getCoreRowModel,
  type ColumnDef,
  type PaginationState,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table";
import { MoreHorizontal, Search } from "lucide-react";
import type { ReactNode } from "react";
import { EmptyState } from "@/components/crm/empty-state";
import { ErrorState } from "@/components/crm/error-state";
import { LoadingSkeleton } from "@/components/crm/loading-skeleton";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

export type DataTableAction<TData> = {
  label: string;
  onSelect: (row: TData) => void;
  destructive?: boolean;
};

type DataTableProps<TData> = {
  columns: ColumnDef<TData>[];
  data: TData[];
  rowCount?: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  sorting?: SortingState;
  onSortingChange?: (sorting: SortingState) => void;
  search?: string;
  onSearchChange?: (search: string) => void;
  toolbarFilters?: ReactNode;
  primaryAction?: ReactNode;
  actions?: DataTableAction<TData>[];
  isLoading?: boolean;
  error?: string | null;
  emptyTitle?: string;
  emptyDescription?: string;
};

export function DataTable<TData>({
  columns,
  data,
  rowCount = data.length,
  pagination,
  onPaginationChange,
  sorting = [],
  onSortingChange,
  search,
  onSearchChange,
  toolbarFilters,
  primaryAction,
  actions,
  isLoading,
  error,
  emptyTitle = "Нет данных",
  emptyDescription,
}: DataTableProps<TData>) {
  const tableColumns = actions?.length
    ? [
        ...columns,
        {
          id: "actions",
          header: "",
          cell: ({ row }) => <RowActions row={row.original} actions={actions} />,
          enableSorting: false,
        } satisfies ColumnDef<TData>,
      ]
    : columns;

  const table = useReactTable({
    data,
    columns: tableColumns,
    rowCount,
    state: { pagination, sorting },
    manualPagination: true,
    manualSorting: true,
    getCoreRowModel: getCoreRowModel(),
    onPaginationChange: (updater) => {
      const next = typeof updater === "function" ? updater(pagination) : updater;
      onPaginationChange(next);
    },
    onSortingChange: (updater) => {
      if (!onSortingChange) return;
      const next = typeof updater === "function" ? updater(sorting) : updater;
      onSortingChange(next);
    },
  });

  const pageCount = Math.max(1, Math.ceil(rowCount / pagination.pageSize));

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          {onSearchChange ? (
            <div className="relative w-full max-w-xs">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                value={search ?? ""}
                onChange={(event) => onSearchChange(event.target.value)}
                placeholder="Поиск"
                className="pl-8"
              />
            </div>
          ) : null}
          {toolbarFilters}
        </div>
        {primaryAction}
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        {error ? (
          <div className="p-4">
            <ErrorState message={error} />
          </div>
        ) : isLoading ? (
          <div className="p-4">
            <LoadingSkeleton rows={pagination.pageSize} />
          </div>
        ) : data.length === 0 ? (
          <div className="p-4">
            <EmptyState title={emptyTitle} description={emptyDescription} />
          </div>
        ) : (
          <table className="w-full border-collapse text-sm">
            <thead className="bg-slate-50 text-left text-xs uppercase tracking-normal text-muted-foreground">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <th
                      key={header.id}
                      className={cn(
                        "h-10 border-b border-border px-3 font-medium",
                        header.column.getCanSort() && "cursor-pointer select-none",
                      )}
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                      {{
                        asc: " ↑",
                        desc: " ↓",
                      }[header.column.getIsSorted() as string] ?? null}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="border-b border-border last:border-0 hover:bg-slate-50">
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="h-11 px-3 align-middle">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
        <div>
          Страница {pagination.pageIndex + 1} из {pageCount}
        </div>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={pagination.pageIndex === 0}
            onClick={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex - 1 })}
          >
            Назад
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={pagination.pageIndex + 1 >= pageCount}
            onClick={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex + 1 })}
          >
            Вперед
          </Button>
        </div>
      </div>
    </div>
  );
}

function RowActions<TData>({ row, actions }: { row: TData; actions: DataTableAction<TData>[] }) {
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="icon" aria-label="Действия">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {actions.map((action) => (
            <DropdownMenuItem
              key={action.label}
              className={action.destructive ? "text-red-700" : undefined}
              onClick={() => action.onSelect(row)}
            >
              {action.label}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
