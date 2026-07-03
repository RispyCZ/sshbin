import { createTheme, type Theme } from "@mui/material/styles";

export type ColorMode = "light" | "dark";

// Colors follow architecture/colors.md. The spec defines tokens in oklch for
// perceptual uniformity; MUI's color manipulator only understands hex/rgb/hsl,
// so each token is the sRGB (gamut-clamped) rendering of its oklch source.
const green = { 50: "#e9fbe9", 400: "#00ba1a", 500: "#00a000", 600: "#007d00", 700: "#005b00" };
const amber = { 300: "#ffc100", 900: "#542a00" };
const teal = { 300: "#00cccf", 900: "#001d1f" };

// Amber (warning) and teal (info) are light accents, so filled surfaces carry
// dark text; green (success) and coral (error) mirror the primary treatment.
const warning = { main: amber[300], contrastText: amber[900] };
const info = { main: teal[300], contrastText: teal[900] };
const success = { main: green[600], contrastText: green[50] };

const palettes = {
  dark: {
    primary: { main: green[400], contrastText: "#001400" }, // green-400 / green-900
    error: { main: "#ff8172", contrastText: "#520000" }, // coral-300 / coral-900
    success,
    warning,
    info,
    background: { default: "#010601", paper: "#050f05" }, // dark-surface base / raised
    text: { primary: green[50], secondary: "#788d78" }, // green-50 / muted
    divider: "#253525", // dark-border
  },
  light: {
    primary: { main: green[500], contrastText: green[50] }, // green-500 / green-50
    error: { main: "#f92725", contrastText: green[50] }, // coral-500
    success,
    warning,
    info,
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
    components: {
      // Contained buttons put text on the accent, so they use the deeper
      // green-600 (light text passes WCAG AA); green-400 stays the accent for
      // links, icons, and outlined buttons where the green is the foreground.
      MuiButton: {
        styleOverrides: {
          contained: ({ ownerState }) =>
            ownerState.color === "primary"
              ? {
                  backgroundColor: green[600],
                  color: green[50],
                  "&:hover": { backgroundColor: green[700] },
                }
              : {},
        },
      },
    },
  });
}
