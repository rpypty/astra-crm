export type UiPagination = {
  pageIndex: number;
  pageSize: number;
};

export function paginationToQuery(pagination: UiPagination) {
  return {
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
  };
}
