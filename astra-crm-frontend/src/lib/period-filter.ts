import { useEffect, useState } from "react";

export type PeriodFilter = {
  dateFrom?: string;
  dateTo?: string;
};

const emptyPeriodFilter: PeriodFilter = {};

export function usePersistentPeriodFilter(storageKey: string) {
  const [period, setPeriod] = useState<PeriodFilter>(() => {
    if (typeof window === "undefined") {
      return emptyPeriodFilter;
    }

    try {
      const raw = window.localStorage.getItem(storageKey);
      if (!raw) {
        return emptyPeriodFilter;
      }
      const parsed = JSON.parse(raw) as PeriodFilter;
      return sanitizePeriodFilter(parsed);
    } catch {
      return emptyPeriodFilter;
    }
  });

  useEffect(() => {
    const sanitized = sanitizePeriodFilter(period);
    const hasValue = Boolean(sanitized.dateFrom || sanitized.dateTo);

    try {
      if (!hasValue) {
        window.localStorage.removeItem(storageKey);
        return;
      }

      window.localStorage.setItem(storageKey, JSON.stringify(sanitized));
    } catch {
      // Local persistence is best-effort; the in-memory filter still works.
    }
  }, [period, storageKey]);

  return [period, setPeriod] as const;
}

export function sanitizePeriodFilter(value: PeriodFilter): PeriodFilter {
  return {
    dateFrom: isISODate(value.dateFrom) ? value.dateFrom : undefined,
    dateTo: isISODate(value.dateTo) ? value.dateTo : undefined,
  };
}

function isISODate(value: unknown): value is string {
  return typeof value === "string" && /^\d{4}-\d{2}-\d{2}$/.test(value);
}
