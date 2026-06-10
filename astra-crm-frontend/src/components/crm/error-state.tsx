import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

type ErrorStateProps = {
  title?: string;
  message: string;
  onRetry?: () => void;
};

export function ErrorState({ title = "Не удалось загрузить данные", message, onRetry }: ErrorStateProps) {
  return (
    <Card className="flex items-start gap-3 border-red-200 bg-red-50 p-4 text-red-950">
      <AlertTriangle className="mt-0.5 h-5 w-5 text-red-600" />
      <div className="min-w-0 flex-1 space-y-2">
        <div>
          <h2 className="text-sm font-semibold">{title}</h2>
          <p className="text-sm text-red-800">{message}</p>
        </div>
        {onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            Повторить
          </Button>
        ) : null}
      </div>
    </Card>
  );
}
