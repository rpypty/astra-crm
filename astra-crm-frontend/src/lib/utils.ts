import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatMoneyMinor(value: number, currency = "RUB") {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency,
    maximumFractionDigits: 2,
  }).format(value / 100);
}

export function formatDateTime(value: string | Date | null | undefined) {
  if (!value) return "—";

  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function parseMoneyToMinor(value: string) {
  const normalized = value.replace(/\s/g, "").replace(",", ".");
  const numberValue = Number(normalized);
  if (!Number.isFinite(numberValue)) return Number.NaN;
  return Math.round(numberValue * 100);
}

export function bpsToPercent(value: number) {
  return value / 100;
}

export function percentToBps(value: number) {
  return Math.round(value * 100);
}
