import { CalendarRange, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { DatePickerField } from "@/shared/ui/date-picker-field";
import type { PeriodFilter } from "@/shared/lib/period-filter";

type PeriodFilterBarProps = {
  value: PeriodFilter;
  onChange: (value: PeriodFilter) => void;
};

export function PeriodFilterBar({ value, onChange }: PeriodFilterBarProps) {
  const hasValue = Boolean(value.dateFrom || value.dateTo);
  const today = todayISO();
  const isTodaySelected = value.dateFrom === today && value.dateTo === today;
  const [draft, setDraft] = useState({
    dateFrom: value.dateFrom ?? "",
    dateTo: value.dateTo ?? "",
  });

  useEffect(() => {
    setDraft({
      dateFrom: value.dateFrom ?? "",
      dateTo: value.dateTo ?? "",
    });
  }, [value.dateFrom, value.dateTo]);

  const normalizedDraft = {
    dateFrom: normalizeDateInput(draft.dateFrom),
    dateTo: normalizeDateInput(draft.dateTo),
  };
  const hasDraftValue = Boolean(draft.dateFrom || draft.dateTo);
  const canApply =
    normalizedDraft.dateFrom !== value.dateFrom ||
    normalizedDraft.dateTo !== value.dateTo ||
    (hasValue && !hasDraftValue);

  const applyDraft = () => {
    onChange(normalizedDraft);
  };

  const selectToday = () => {
    setDraft({ dateFrom: today, dateTo: today });
    onChange({ dateFrom: today, dateTo: today });
  };

  return (
    <Card>
      <CardContent className="flex flex-col gap-3 p-4 md:flex-row md:items-end md:justify-between">
        <div className="flex min-w-0 flex-1 flex-col gap-3 md:flex-row md:items-end">
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex items-center gap-2 text-sm font-medium">
              <CalendarRange className="h-4 w-4 text-muted-foreground" />
              Период
            </div>
            <Button
              type="button"
              variant={isTodaySelected ? "default" : "outline"}
              size="sm"
              onClick={selectToday}
            >
              Сегодня
            </Button>
          </div>
          <div className="min-w-0 flex-1 md:max-w-52">
            <DatePickerField
              value={draft.dateFrom}
              placeholder="С YYYY-MM-DD"
              max={normalizedDraft.dateTo}
              onChange={(dateFrom) => {
                setDraft((current) => ({ ...current, dateFrom: dateFrom ?? "" }));
              }}
            />
          </div>
          <div className="min-w-0 flex-1 md:max-w-52">
            <DatePickerField
              value={draft.dateTo}
              placeholder="По YYYY-MM-DD"
              min={normalizedDraft.dateFrom}
              onChange={(dateTo) => {
                setDraft((current) => ({ ...current, dateTo: dateTo ?? "" }));
              }}
            />
          </div>
        </div>
        <Button type="button" disabled={!canApply} onClick={applyDraft}>
          Применить
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={!hasValue && !hasDraftValue}
          onClick={() => {
            setDraft({ dateFrom: "", dateTo: "" });
            onChange({});
          }}
        >
          <X className="h-4 w-4" />
          Сбросить
        </Button>
      </CardContent>
    </Card>
  );
}

function normalizeDateInput(value: string) {
  const trimmed = value.trim();
  return parseISODate(trimmed) ? trimmed : undefined;
}

function parseISODate(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) {
    return undefined;
  }

  const year = Number(match[1]);
  const month = Number(match[2]) - 1;
  const day = Number(match[3]);
  const date = new Date(year, month, day);
  if (date.getFullYear() !== year || date.getMonth() !== month || date.getDate() !== day) {
    return undefined;
  }

  return { year, month, day };
}

function todayISO() {
  const now = new Date();
  return toISODate(now.getFullYear(), now.getMonth(), now.getDate());
}

function toISODate(year: number, month: number, day: number) {
  return `${year}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}
