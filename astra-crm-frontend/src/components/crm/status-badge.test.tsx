import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusBadge } from "@/components/crm/status-badge";

describe("StatusBadge", () => {
  it("renders user-facing status label", () => {
    render(<StatusBadge status="closed_with_discrepancy" />);

    expect(screen.getByText("С расхождением")).toBeInTheDocument();
  });
});
