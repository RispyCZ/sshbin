import { Box, type BoxProps } from "@mui/material";
import { useTheme } from "@mui/material/styles";
import wordmarkDark from "../assets/logo/sshbin-wordmark-transparent-dark.svg";
import wordmarkLight from "../assets/logo/sshbin-wordmark-transparent-light.svg";
import iconGreen from "../assets/logo/sshbin-icon-green.svg";

interface LogoProps extends Omit<BoxProps<"img">, "component" | "src" | "alt"> {
  variant?: "wordmark" | "icon";
  height?: number;
}

// Logo renders the sshbin brand mark. The wordmark follows the active theme
// (light/dark); the icon is the standalone green glyph.
export function Logo({ variant = "wordmark", height = 28, sx, ...rest }: LogoProps) {
  const { palette } = useTheme();
  const src =
    variant === "icon" ? iconGreen : palette.mode === "dark" ? wordmarkDark : wordmarkLight;

  return (
    <Box
      component="img"
      src={src}
      alt="sshbin"
      sx={[{ height, display: "block" }, ...(Array.isArray(sx) ? sx : [sx])]}
      {...rest}
    />
  );
}
