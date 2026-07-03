import { useState } from "react";
import { describe, expect, it, vi } from "vite-plus/test";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { OtpInput } from "./OtpInput";

// Harness wires OtpInput to real state so typing/backspace behave as in the app,
// while exposing onChange/onComplete spies for assertions.
function Harness({
  onChange,
  onComplete,
  length,
}: {
  onChange?: (code: string) => void;
  onComplete?: (code: string) => void;
  length?: number;
}) {
  const [code, setCode] = useState("");
  return (
    <OtpInput
      value={code}
      length={length}
      onChange={(c) => {
        setCode(c);
        onChange?.(c);
      }}
      onComplete={onComplete}
    />
  );
}

function boxes(): HTMLInputElement[] {
  return screen
    .getAllByRole("textbox")
    .filter((el): el is HTMLInputElement => el instanceof HTMLInputElement);
}

describe("OtpInput", () => {
  it("renders one box per digit with numbered aria labels", () => {
    render(<Harness />);

    const inputs = boxes();
    expect(inputs).toHaveLength(6);
    expect(inputs[0]).toHaveAttribute("aria-label", "Digit 1");
    expect(inputs[5]).toHaveAttribute("aria-label", "Digit 6");
  });

  it("honors a custom length", () => {
    render(<Harness length={4} />);
    expect(boxes()).toHaveLength(4);
  });

  it("auto-advances focus to the next box as digits are typed", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    const inputs = boxes();

    await user.click(inputs[0]);
    await user.keyboard("12");

    expect(inputs[0]).toHaveValue("1");
    expect(inputs[1]).toHaveValue("2");
    expect(inputs[2]).toHaveFocus();
  });

  it("ignores non-digit input", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);

    await user.click(boxes()[0]);
    await user.keyboard("a");

    expect(boxes()[0]).toHaveValue("");
    expect(onChange).toHaveBeenLastCalledWith("");
  });

  it("fires onComplete once the final digit lands", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();
    render(<Harness onComplete={onComplete} />);

    await user.click(boxes()[0]);
    await user.keyboard("12345");
    expect(onComplete).not.toHaveBeenCalled();

    await user.keyboard("6");
    expect(onComplete).toHaveBeenCalledExactlyOnceWith("123456");
  });

  it("moves to the previous box and clears it on backspace when empty", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    const inputs = boxes();

    await user.click(inputs[0]);
    await user.keyboard("12");
    // Focus now on box 3 (empty). Backspace hops back and clears box 2.
    await user.keyboard("{Backspace}");

    expect(inputs[1]).toHaveFocus();
    expect(inputs[1]).toHaveValue("");
    expect(inputs[0]).toHaveValue("1");
  });

  it("fills boxes from a pasted code and fires onComplete", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();
    render(<Harness onComplete={onComplete} />);
    const inputs = boxes();

    await user.click(inputs[0]);
    await user.paste("246810");

    expect(inputs.map((i) => i.value)).toEqual(["2", "4", "6", "8", "1", "0"]);
    expect(onComplete).toHaveBeenCalledExactlyOnceWith("246810");
  });

  it("strips non-digits and caps length when pasting", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    const inputs = boxes();

    await user.click(inputs[0]);
    await user.paste("12-34-56-99");

    expect(inputs.map((i) => i.value)).toEqual(["1", "2", "3", "4", "5", "6"]);
  });

  it("disables every box when disabled", () => {
    render(<OtpInput value="" onChange={() => {}} disabled />);
    for (const input of boxes()) expect(input).toBeDisabled();
  });
});
