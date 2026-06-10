type RequisiteCellProps = {
  phone: string;
  method: string;
  proxy?: string;
};

export function RequisiteCell({ phone, method, proxy }: RequisiteCellProps) {
  return (
    <div className="min-w-0">
      <div className="truncate text-sm font-medium">{phone}</div>
      <div className="truncate text-xs text-muted-foreground">
        {method}
        {proxy ? ` · ${proxy}` : ""}
      </div>
    </div>
  );
}
