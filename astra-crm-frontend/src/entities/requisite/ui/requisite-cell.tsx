import { formatRussianPhone, normalizeRussianPhone, phoneDigits } from "@/shared/lib/utils";

type RequisiteCellProps = {
  phone: string;
  method: string;
  proxy?: string;
};

export function RequisiteCell({ phone, method, proxy }: RequisiteCellProps) {
  const formattedPhone = formatRussianPhone(phone);
  const copyValue = phoneDigits(normalizeRussianPhone(phone));
  const canCopy = /^7\d{10}$/.test(copyValue);

  return (
    <div className="min-w-0">
      <button
        type="button"
        className="block max-w-full truncate text-left text-sm font-medium hover:text-primary"
        title={canCopy ? "Скопировать номер" : undefined}
        onClick={(event) => {
          event.stopPropagation();
          if (canCopy) void navigator.clipboard?.writeText(copyValue);
        }}
      >
        {formattedPhone}
      </button>
      <div className="truncate text-xs text-muted-foreground">
        {method}
        {proxy ? ` · ${proxy}` : ""}
      </div>
    </div>
  );
}
