import { createTheme, type Theme } from "@mui/material/styles";

export type ColorMode = "light" | "dark";

// Colors follow architecture/colors.md. The spec defines tokens in oklch for
// perceptual uniformity; MUI's color manipulator only understands hex/rgb/hsl,
// so each token is the sRGB (gamut-clamped) rendering of its oklch source.
const palettes = {
  dark: {
    primary: { main: "#00ba1a", contrastText: "#001400" }, // green-400 / green-900
    error: { main: "#ff8172" }, // coral-300
    background: { default: "#010601", paper: "#050f05" }, // dark-surface base / raised
    text: { primary: "#e9fbe9", secondary: "#788d78" }, // green-50 / muted
    divider: "#253525", // dark-border
  },
  light: {
    primary: { main: "#00a000", contrastText: "#e9fbe9" }, // green-500 / green-50
    error: { main: "#f92725" }, // coral-500
    background: { default: "#f8fbf8", paper: "#eff6ef" }, // surface base / raised
    text: { primary: "#001400", secondary: "#005b00" }, // green-900 / green-700
    divider: "#c9dac8", // surface-border
  },
} as const;

export function buildTheme(mode: ColorMode): Theme {
  return createTheme({
    palette: { mode, ...palettes[mode] },
    shape: { borderRadius: 8 },
    typography: { fontFamily: '"Noto Sans", system-ui, sans-serif' },
  });
}
