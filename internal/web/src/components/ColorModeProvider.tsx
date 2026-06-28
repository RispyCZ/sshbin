import { createContext, use, useMemo, useState, type ReactNode } from "react";
import { CssBaseline, ThemeProvider } from "@mui/material";
import { buildTheme, type ColorMode } from "../theme.ts";

const STORAGE_KEY = "theme";

interface ColorModeCtx {
  mode: ColorMode;
  toggle: () => void;
}

const ColorModeContext = createContext<ColorModeCtx>({ mode: "dark", toggle: () => {} });

export function useColorMode(): ColorModeCtx {
  return use(ColorModeContext);
}

// initialMode mirrors the old UI: use a persisted choice, else the OS preference.
function initialMode(): ColorMode {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function ColorModeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<ColorMode>(initialMode);

  const value = useMemo<ColorModeCtx>(
    () => ({
      mode,
      toggle: () =>
        setMode((m) => {
          const next: ColorMode = m === "dark" ? "light" : "dark";
          localStorage.setItem(STORAGE_KEY, next);
          return next;
        }),
    }),
    [mode],
  );

  const theme = useMemo(() => buildTheme(mode), [mode]);

  return (
    <ColorModeContext value={value}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ColorModeContext>
  );
}
