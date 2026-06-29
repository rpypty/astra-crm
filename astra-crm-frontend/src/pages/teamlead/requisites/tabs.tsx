import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import type { ReactNode } from "react";
import { DataTable, type DataTableAction } from "@/shared/ui/data-table";
import type { Requisite, RequisiteAssignmentWorkRow } from "@/shared/model/domain";

type AllRequisitesTabProps = {
  columns: ColumnDef<Requisite>[];
  data: Requisite[];
  rowCount: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  search: string;
  onSearchChange: (search: string) => void;
  toolbarFilters: ReactNode;
  isLoading?: boolean;
  isFetching?: boolean;
  error?: string | null;
  onRowClick: (row: Requisite) => void;
  actions: DataTableAction<Requisite>[];
};

export function AllRequisitesTab({
  columns,
  data,
  rowCount,
  pagination,
  onPaginationChange,
  search,
  onSearchChange,
  toolbarFilters,
  isLoading,
  isFetching,
  error,
  onRowClick,
  actions,
}: AllRequisitesTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={rowCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      serverSidePagination
      search={search}
      onSearchChange={onSearchChange}
      toolbarFilters={toolbarFilters}
      isLoading={isLoading}
      isFetching={isFetching}
      error={error}
      emptyTitle="Реквизитов пока нет"
      emptyDescription="Добавьте первый реквизит, чтобы назначить его трейдеру."
      onRowClick={onRowClick}
      actions={actions}
    />
  );
}

type WorkRowsTabProps = {
  columns: ColumnDef<RequisiteAssignmentWorkRow>[];
  data: RequisiteAssignmentWorkRow[];
  rowCount: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  search?: string;
  onSearchChange?: (search: string) => void;
  toolbarFilters?: ReactNode;
  isLoading?: boolean;
  isFetching?: boolean;
  error?: string | null;
};

export function RequisiteActivityTab({
  columns,
  data,
  rowCount,
  pagination,
  onPaginationChange,
  search,
  onSearchChange,
  toolbarFilters,
  isLoading,
  isFetching,
  error,
}: WorkRowsTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={rowCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      serverSidePagination
      search={search}
      onSearchChange={onSearchChange}
      toolbarFilters={toolbarFilters}
      isLoading={isLoading}
      isFetching={isFetching}
      error={error}
      emptyTitle="Активности пока нет"
      emptyDescription="Здесь появятся фактические взятия в работу, закрытия, блоки и зафиксированные обороты."
    />
  );
}

type PlanningTabProps = WorkRowsTabProps & {
  onRowClick: (row: RequisiteAssignmentWorkRow) => void;
};

export function RequisitePlanningTab({
  columns,
  data,
  rowCount,
  pagination,
  onPaginationChange,
  isLoading,
  isFetching,
  error,
  onRowClick,
}: PlanningTabProps) {
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
      emptyTitle="Планов пока нет"
      emptyDescription="Запланируйте дату, трейдера, реквизит и лимит."
      onRowClick={onRowClick}
    />
  );
}
