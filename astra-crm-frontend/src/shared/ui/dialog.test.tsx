import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";

describe("DialogContent", () => {
  it("does not bubble clicks from portalled content to React parents", async () => {
    const user = userEvent.setup();
    const parentClick = vi.fn();
    const contentClick = vi.fn();

    render(
      <div onClick={parentClick}>
        <Dialog open>
          <DialogContent>
            <DialogTitle>Диалог</DialogTitle>
            <DialogDescription>Проверочный диалог</DialogDescription>
            <button type="button" onClick={contentClick}>
              Сохранить
            </button>
          </DialogContent>
        </Dialog>
      </div>,
    );

    await user.click(screen.getByRole("button", { name: "Сохранить" }));

    expect(contentClick).toHaveBeenCalledTimes(1);
    expect(parentClick).not.toHaveBeenCalled();
  });
});
