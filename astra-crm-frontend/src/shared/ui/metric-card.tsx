import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";

type MetricCardProps = {
  label: string;
  value: string;
  warning?: boolean;
  layout?: "compact" | "header";
};

export function MetricCard({ label, value, warning, layout = "compact" }: MetricCardProps) {
  if (layout === "header") {
    return (
      <Card className={warning ? "border-amber-200 bg-amber-50" : undefined}>
        <CardHeader>
          <CardTitle className="text-xs uppercase text-muted-foreground">{label}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-semibold">{value}</div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={warning ? "border-amber-200 bg-amber-50" : undefined}>
      <CardContent className="p-4">
        <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
        <div className="mt-2 text-2xl font-semibold">{value}</div>
      </CardContent>
    </Card>
  );
}
