import { formatMoneyMinor } from "@/lib/utils";

type MoneyCellProps = {
  valueMinor: number;
  currency?: string;
};

export function MoneyCell({ valueMinor, currency }: MoneyCellProps) {
  return <span className="block text-right tabular-nums">{formatMoneyMinor(valueMinor, currency)}</span>;
}
