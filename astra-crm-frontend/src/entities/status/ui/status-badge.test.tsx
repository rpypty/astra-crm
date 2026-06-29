import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusBadge } from "@/entities/status/ui/status-badge";

describe("StatusBadge", () => {
  it("renders user-facing status label", () => {
    render(<StatusBadge status="closed_with_discrepancy" />);

    expect(screen.getByText("С расхождением")).toBeInTheDocument();
  });

  it("renders teamlead reconciliation and severity labels", () => {
    render(
      <div>
        <StatusBadge status="confirmed_by_tl" />
        <StatusBadge status="updated_by_tl" />
        <StatusBadge status="tl_discrepancy" />
        <StatusBadge status="tl_accepted" />
        <StatusBadge status="blocker" />
      </div>,
    );

    expect(screen.getByText("Подтверждено TL")).toBeInTheDocument();
    expect(screen.getByText("Обновлено TL")).toBeInTheDocument();
    expect(screen.getByText("Расхождение с TL")).toBeInTheDocument();
    expect(screen.getByText("Принято TL")).toBeInTheDocument();
    expect(screen.getByText("Блокер")).toBeInTheDocument();
  });
});
