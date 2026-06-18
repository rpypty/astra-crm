import type { ReactNode } from "react";
import { Info } from "lucide-react";
import { Label } from "@/shared/ui/label";

type FormFieldProps = {
  label: ReactNode;
  htmlFor?: string;
  error?: string;
  help?: string;
  labelInfo?: string;
  children: ReactNode;
};

export function FormField({ label, htmlFor, error, help, labelInfo, children }: FormFieldProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <Label htmlFor={htmlFor}>{label}</Label>
        {labelInfo ? (
          <span
            aria-label={labelInfo}
            title={labelInfo}
            className="inline-flex h-4 w-4 items-center justify-center rounded-full text-muted-foreground hover:text-foreground"
          >
            <Info className="h-3.5 w-3.5" />
          </span>
        ) : null}
      </div>
      {children}
      {help ? <p className="text-xs text-muted-foreground">{help}</p> : null}
      {error ? <p className="text-xs font-medium text-red-600">{error}</p> : null}
    </div>
  );
}
