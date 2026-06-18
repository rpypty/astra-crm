import { Button } from "@/shared/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { formatCardNumber, formatRussianPhone, normalizeRussianPhone } from "@/shared/lib/utils";

export type CopyHandler = (value: string | undefined | null, label: string) => void;

type RequisiteIdentity = {
  phone: string;
  proxy?: string;
  cardNumber?: string;
  holderName?: string;
};

export function RequisitePhoneMenu({ item, onCopy }: { item: RequisiteIdentity; onCopy: CopyHandler }) {
  const formattedPhone = formatRussianPhone(item.phone);
  const phoneCopyValue = normalizeRussianPhone(item.phone);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          className="h-auto max-w-[190px] justify-start px-0 py-0 text-left text-sm font-medium hover:bg-transparent hover:text-primary"
          title="Показать данные реквизита"
          onClick={(event) => event.stopPropagation()}
        >
          <span className="truncate tabular-nums">{formattedPhone}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-80 p-2" onClick={(event) => event.stopPropagation()}>
        <DropdownMenuLabel className="px-1">Данные реквизита</DropdownMenuLabel>
        <div className="space-y-1">
          <CopyableMenuRow label="Телефон" value={formattedPhone} copyValue={phoneCopyValue} onCopy={onCopy} />
          <CopyableMenuRow
            label="Карта"
            value={formatCardNumber(item.cardNumber)}
            copyValue={item.cardNumber}
            onCopy={onCopy}
          />
          <CopyableMenuRow label="Держатель" value={item.holderName || "—"} copyValue={item.holderName} onCopy={onCopy} />
          <CopyableMenuRow label="Прокси" value={item.proxy || "—"} copyValue={item.proxy} onCopy={onCopy} />
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function CopyableInlineValue({ label, value, onCopy }: { label: string; value?: string; onCopy: CopyHandler }) {
  return (
    <button
      type="button"
      className="max-w-[180px] truncate text-left text-sm text-muted-foreground hover:text-foreground"
      title={value || "—"}
      onClick={(event) => {
        event.stopPropagation();
        onCopy(value, label);
      }}
    >
      {value || "—"}
    </button>
  );
}

function CopyableMenuRow({
  label,
  value,
  copyValue,
  onCopy,
}: {
  label: string;
  value: string;
  copyValue?: string | null;
  onCopy: CopyHandler;
}) {
  return (
    <button
      type="button"
      className="flex w-full items-center justify-between gap-3 rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent"
      onClick={() => onCopy(copyValue, label)}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate font-medium">{value}</span>
    </button>
  );
}

export function CopyToast({ message }: { message: string | null }) {
  if (!message) return null;

  return (
    <div role="status" className="fixed bottom-5 right-5 z-50 rounded-md border border-border bg-slate-950 px-4 py-2 text-sm font-medium text-white shadow-lg">
      {message}
    </div>
  );
}
