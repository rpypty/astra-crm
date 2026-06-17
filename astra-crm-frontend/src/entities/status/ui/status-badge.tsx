import { Badge } from "@/shared/ui/badge";

type StatusBadgeProps = {
  status: string;
};

const statusMap: Record<string, { label: string; variant: "neutral" | "success" | "warning" | "failed" | "info" | "processing" }> = {
  hand_success: { label: "Успех", variant: "success" },
  corrected: { label: "Исправлен", variant: "success" },
  failed: { label: "Неуспех", variant: "failed" },
  auto_decline: { label: "Неуспех", variant: "failed" },
  cancelled: { label: "Отменен", variant: "neutral" },
  unknown: { label: "Неизвестно", variant: "warning" },
  matched: { label: "Сошлось", variant: "success" },
  mismatch: { label: "Расхождение", variant: "failed" },
  accepted_with_comment: { label: "Подтверждено", variant: "warning" },
  open: { label: "Открыта", variant: "processing" },
  closed: { label: "Закрыта", variant: "success" },
  closed_with_discrepancy: { label: "С расхождением", variant: "warning" },
  active: { label: "Активен", variant: "success" },
  disabled: { label: "Отключен", variant: "neutral" },
  archived: { label: "Архив", variant: "neutral" },
  blocked: { label: "В блоке", variant: "failed" },
  planned: { label: "Запланирован", variant: "info" },
  assigned: { label: "Назначен", variant: "info" },
  in_work: { label: "В работе", variant: "processing" },
  worked: { label: "Отработан", variant: "success" },
  correction: { label: "Коррекция", variant: "warning" },
  expired: { label: "Просрочен", variant: "warning" },
  released: { label: "Сдан", variant: "success" },
  paid: { label: "Оплачена", variant: "success" },
  applied: { label: "Загружен", variant: "info" },
  reconciled: { label: "Сверен", variant: "success" },
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const statusMeta = statusMap[status] ?? { label: status, variant: "neutral" as const };
  return <Badge variant={statusMeta.variant}>{statusMeta.label}</Badge>;
}
