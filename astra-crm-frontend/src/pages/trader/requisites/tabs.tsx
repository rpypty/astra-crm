import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { DataTable } from "@/shared/ui/data-table";
import type { ShiftRequisite } from "@/shared/model/domain";

type RequisitesTabProps = {
  columns: ColumnDef<ShiftRequisite>[];
  data: ShiftRequisite[];
  rowCount: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  isLoading?: boolean;
  isFetching?: boolean;
  error?: string | null;
};

type CurrentRequisitesTabProps = RequisitesTabProps & {
  onRowClick: (row: ShiftRequisite) => void;
};

export function CurrentRequisitesTab({
  columns,
  data,
  rowCount,
  pagination,
  onPaginationChange,
  isLoading,
  isFetching,
  error,
  onRowClick,
}: CurrentRequisitesTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={rowCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      serverSidePagination
      isLoading={isLoading}
      isFetching={isFetching}
      error={error}
      emptyTitle="Нет текущих реквизитов"
      onRowClick={onRowClick}
    />
  );
}

export function FutureRequisitesTab({
  columns,
  data,
  rowCount,
  pagination,
  onPaginationChange,
  isLoading,
  isFetching,
  error,
}: RequisitesTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={rowCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      serverSidePagination
      isLoading={isLoading}
      isFetching={isFetching}
      error={error}
      emptyTitle="Будущих реквизитов нет"
      emptyDescription="Здесь появятся назначения на будущие даты."
    />
  );
}

export function HistoricalRequisitesTab({
  columns,
  data,
  rowCount,
  pagination,
  onPaginationChange,
  isLoading,
  isFetching,
  error,
}: RequisitesTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={rowCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      serverSidePagination
      isLoading={isLoading}
      isFetching={isFetching}
      error={error}
      emptyTitle="Истории реквизитов нет"
      emptyDescription="Закрытые и заблокированные реквизиты появятся после отработки."
    />
  );
}
