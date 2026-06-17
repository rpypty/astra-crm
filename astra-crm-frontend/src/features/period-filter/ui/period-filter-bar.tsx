import { CalendarRange, ChevronLeft, ChevronRight, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type { PeriodFilter } from "@/lib/period-filter";

type PeriodFilterBarProps = {
  value: PeriodFilter;
  onChange: (value: PeriodFilter) => void;
};

const weekdays = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];
const monthNames = [
  "Январь",
  "Февраль",
  "Март",
  "Апрель",
  "Май",
  "Июнь",
  "Июль",
  "Август",
  "Сентябрь",
  "Октябрь",
  "Ноябрь",
  "Декабрь",
];

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

function DatePickerField({
  value,
  placeholder,
  min,
  max,
  onChange,
}: {
  value?: string;
  placeholder: string;
  min?: string;
  max?: string;
  onChange: (value?: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const initialMonth = value ?? max ?? min ?? todayISO();
  const [{ year, month }, setVisibleMonth] = useState(() => parseISODate(initialMonth) ?? todayParts());

  useEffect(() => {
    if (value) {
      const parsed = parseISODate(value);
      if (parsed) {
        setVisibleMonth({ year: parsed.year, month: parsed.month });
      }
    }
  }, [value]);

  useEffect(() => {
    if (!open) return;

    const closeOnOutsideClick = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", closeOnOutsideClick);
    return () => document.removeEventListener("mousedown", closeOnOutsideClick);
  }, [open]);

  const days = useMemo(() => calendarDays(year, month), [year, month]);
  const selected = value;

  const commitDraft = () => {
    const normalized = normalizeDateInput(value ?? "");
    onChange(normalized);
  };

  const selectDate = (nextValue: string) => {
    if (!isWithinRange(nextValue, min, max)) {
      return;
    }
    onChange(nextValue);
    setOpen(false);
  };

  return (
    <div ref={rootRef} className="relative">
      <div className="flex gap-2">
        <Input
          type="text"
          inputMode="numeric"
          placeholder={placeholder}
          value={value ?? ""}
          onBlur={commitDraft}
          onChange={(event) => {
            const nextDraft = event.target.value;
            onChange(nextDraft || undefined);
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              commitDraft();
              event.currentTarget.blur();
            }
            if (event.key === "Escape") {
              setOpen(false);
            }
          }}
        />
        <Button type="button" variant="outline" size="icon" aria-label="Открыть календарь" onClick={() => setOpen((current) => !current)}>
          <CalendarRange className="h-4 w-4" />
        </Button>
      </div>

      {open ? (
        <div className="absolute left-0 top-11 z-50 w-72 rounded-md border border-border bg-popover p-3 text-popover-foreground shadow-md">
          <div className="mb-3 flex items-center justify-between gap-2">
            <Button type="button" variant="ghost" size="icon" aria-label="Предыдущий месяц" onClick={() => setVisibleMonth(addMonths(year, month, -1))}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <div className="text-sm font-semibold">
              {monthNames[month]} {year}
            </div>
            <Button type="button" variant="ghost" size="icon" aria-label="Следующий месяц" onClick={() => setVisibleMonth(addMonths(year, month, 1))}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>

          <div className="mb-1 grid grid-cols-7 gap-1 text-center text-xs font-medium text-muted-foreground">
            {weekdays.map((day) => (
              <div key={day} className="py-1">
                {day}
              </div>
            ))}
          </div>

          <div className="grid grid-cols-7 gap-1">
            {days.map((day, index) => {
              if (!day) {
                return <div key={`empty-${index}`} className="h-8" />;
              }

              const disabled = !isWithinRange(day.iso, min, max);
              const isSelected = selected === day.iso;
              const isToday = todayISO() === day.iso;

              return (
                <button
                  key={day.iso}
                  type="button"
                  disabled={disabled}
                  className={cn(
                    "flex h-8 items-center justify-center rounded-md text-sm transition-colors hover:bg-accent disabled:pointer-events-none disabled:opacity-35",
                    isSelected && "bg-primary text-primary-foreground hover:bg-primary",
                    !isSelected && isToday && "border border-primary/40",
                  )}
                  onClick={() => selectDate(day.iso)}
                >
                  {day.day}
                </button>
              );
            })}
          </div>

          <div className="mt-3 flex items-center justify-between gap-2 border-t border-border pt-3">
            <Button type="button" variant="ghost" size="sm" onClick={() => selectDate(todayISO())}>
              Сегодня
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                onChange(undefined);
                setOpen(false);
              }}
            >
              Очистить
            </Button>
          </div>
        </div>
      ) : null}
    </div>
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

function todayParts() {
  const now = new Date();
  return { year: now.getFullYear(), month: now.getMonth() };
}

function todayISO() {
  const now = new Date();
  return toISODate(now.getFullYear(), now.getMonth(), now.getDate());
}

function calendarDays(year: number, month: number) {
  const firstDay = new Date(year, month, 1);
  const startOffset = (firstDay.getDay() + 6) % 7;
  const totalDays = new Date(year, month + 1, 0).getDate();
  const days: Array<{ day: number; iso: string } | null> = [];

  for (let index = 0; index < startOffset; index += 1) {
    days.push(null);
  }
  for (let day = 1; day <= totalDays; day += 1) {
    days.push({ day, iso: toISODate(year, month, day) });
  }

  return days;
}

function toISODate(year: number, month: number, day: number) {
  return `${year}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}

function addMonths(year: number, month: number, offset: number) {
  const date = new Date(year, month + offset, 1);
  return { year: date.getFullYear(), month: date.getMonth() };
}

function isWithinRange(value: string, min?: string, max?: string) {
  if (min && value < min) {
    return false;
  }
  if (max && value > max) {
    return false;
  }
  return true;
}
