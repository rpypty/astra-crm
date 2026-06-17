import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import type { ReactNode } from "react";
import { DataTable, type DataTableAction } from "@/shared/ui/data-table";
import type { Requisite, RequisiteAssignmentWorkRow } from "@/shared/model/domain";

type AllRequisitesTabProps = {
  columns: ColumnDef<Requisite>[];
  data: Requisite[];
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  search: string;
  onSearchChange: (search: string) => void;
  toolbarFilters: ReactNode;
  isLoading?: boolean;
  error?: string | null;
  onRowClick: (row: Requisite) => void;
  actions: DataTableAction<Requisite>[];
};

export function AllRequisitesTab({
  columns,
  data,
  pagination,
  onPaginationChange,
  search,
  onSearchChange,
  toolbarFilters,
  isLoading,
  error,
  onRowClick,
  actions,
}: AllRequisitesTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={data.length}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      search={search}
      onSearchChange={onSearchChange}
      toolbarFilters={toolbarFilters}
      isLoading={isLoading}
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
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  isLoading?: boolean;
  error?: string | null;
};

export function RequisiteActivityTab({
  columns,
  data,
  pagination,
  onPaginationChange,
  isLoading,
  error,
}: WorkRowsTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={data.length}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      isLoading={isLoading}
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
  pagination,
  onPaginationChange,
  isLoading,
  error,
  onRowClick,
}: PlanningTabProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      rowCount={data.length}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      isLoading={isLoading}
      error={error}
      emptyTitle="Планов пока нет"
      emptyDescription="Запланируйте дату, трейдера, реквизит и целевой оборот."
      onRowClick={onRowClick}
    />
  );
}
