import type { Order } from "@/shared/model/domain";

export function filterOrdersBySearch(orders: Order[], search?: string) {
  const normalizedSearch = search?.trim().toLowerCase();
  if (!normalizedSearch) {
    return orders;
  }

  return orders.filter((order) =>
    [order.id, order.trader, order.workerName, order.requisite, order.innerId].some((value) =>
      value.toLowerCase().includes(normalizedSearch),
    ),
  );
}
