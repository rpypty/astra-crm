import { formatDateTime } from "@/lib/utils";

type DateTimeCellProps = {
  value?: string | Date | null;
};

export function DateTimeCell({ value }: DateTimeCellProps) {
  return <span className="whitespace-nowrap text-sm text-muted-foreground">{formatDateTime(value)}</span>;
}
