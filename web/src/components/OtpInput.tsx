import { useRef, type ClipboardEvent, type KeyboardEvent } from "react";
import { Box, TextField } from "@mui/material";

interface OtpInputProps {
  value: string;
  onChange: (code: string) => void;
  // onComplete fires once the code fills every box (typed, pasted, or autofilled).
  onComplete?: (code: string) => void;
  length?: number;
  disabled?: boolean;
  autoFocus?: boolean;
}

// OtpInput renders one box per digit with auto-advance, backspace-to-previous,
// paste-to-fill, and auto-submit on completion — the ergonomics of the old
// sb-otp Lit component, rebuilt for the SPA.
export function OtpInput({
  value,
  onChange,
  onComplete,
  length = 6,
  disabled,
  autoFocus,
}: OtpInputProps) {
  const inputs = useRef<(HTMLInputElement | null)[]>([]);
  const chars = Array.from({ length }, (_, i) => value[i] ?? "");

  function focusBox(i: number) {
    inputs.current[Math.max(0, Math.min(i, length - 1))]?.focus();
  }

  // commit normalizes to digits, caps at length, propagates, and fires
  // onComplete when the code is full.
  function commit(next: string) {
    const code = next.replace(/\D/g, "").slice(0, length);
    onChange(code);
    if (code.length === length) onComplete?.(code);
  }

  function handleInput(idx: number, raw: string) {
    const digit = raw.replace(/\D/g, "").slice(-1);
    const arr = chars.slice();
    arr[idx] = digit;
    commit(arr.join(""));
    if (digit && idx < length - 1) focusBox(idx + 1);
  }

  function handleKeyDown(idx: number, e: KeyboardEvent<HTMLElement>) {
    if (e.key === "Backspace" && !chars[idx] && idx > 0) {
      e.preventDefault();
      const arr = chars.slice();
      arr[idx - 1] = "";
      commit(arr.join(""));
      focusBox(idx - 1);
    }
  }

  function handlePaste(idx: number, e: ClipboardEvent<HTMLElement>) {
    e.preventDefault();
    const text = e.clipboardData.getData("text").replace(/\D/g, "");
    if (!text) return;
    const arr = chars.slice();
    for (let i = 0; i < text.length && idx + i < length; i++) arr[idx + i] = text[i];
    commit(arr.join(""));
    focusBox(idx + text.length);
  }

  return (
    <Box sx={{ display: "flex", gap: 1, justifyContent: "center" }}>
      {chars.map((c, i) => (
        <TextField
          // Boxes are positional and never reordered, so the index key is stable.
          key={i}
          inputRef={(el: HTMLInputElement | null) => {
            inputs.current[i] = el;
          }}
          value={c}
          onChange={(e) => handleInput(i, e.target.value)}
          onKeyDown={(e) => handleKeyDown(i, e)}
          onPaste={(e) => handlePaste(i, e)}
          disabled={disabled}
          autoFocus={autoFocus && i === 0}
          slotProps={{
            htmlInput: {
              inputMode: "numeric",
              maxLength: 1,
              autoComplete: i === 0 ? "one-time-code" : "off",
              "aria-label": `Digit ${i + 1}`,
              style: { textAlign: "center", fontSize: "1.25rem" },
            },
          }}
          sx={{ width: 44 }}
        />
      ))}
    </Box>
  );
}
