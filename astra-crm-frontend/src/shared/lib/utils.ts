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

export function phoneDigits(value: string) {
  const digits = value.replace(/\D/g, "");
  if (digits.length === 11 && digits.startsWith("8")) return `7${digits.slice(1)}`;
  if (digits.length === 10) return `7${digits}`;
  return digits;
}

export function isValidRussianPhone(value: string) {
  return /^7\d{10}$/.test(phoneDigits(value));
}

export function normalizeRussianPhone(value: string) {
  const digits = phoneDigits(value);
  return /^7\d{10}$/.test(digits) ? digits : value.trim();
}

export function formatRussianPhone(value: string) {
  const digits = phoneDigits(value);
  if (!/^7\d{10}$/.test(digits)) return value || "—";
  return `+7 (${digits.slice(1, 4)}) ${digits.slice(4, 7)}-${digits.slice(7, 9)}-${digits.slice(9, 11)}`;
}

export function cardDigits(value: string) {
  return value.replace(/\D/g, "");
}

export function normalizeCardNumber(value: string) {
  const digits = cardDigits(value);
  return digits || value.trim();
}

export function formatCardNumber(value: string | null | undefined) {
  if (!value) return "—";
  const digits = cardDigits(value);
  if (!digits) return value;
  return digits.replace(/(.{4})/g, "$1 ").trim();
}
