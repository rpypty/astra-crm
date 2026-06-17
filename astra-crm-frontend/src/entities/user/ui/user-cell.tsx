type UserCellProps = {
  login: string;
  secondary?: string;
};

export function UserCell({ login, secondary }: UserCellProps) {
  return (
    <div className="min-w-0">
      <div className="truncate text-sm font-medium">{login}</div>
      {secondary ? <div className="truncate text-xs text-muted-foreground">{secondary}</div> : null}
    </div>
  );
}
