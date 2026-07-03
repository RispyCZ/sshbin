import { useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Box, Button, IconButton, Paper, Stack, Tooltip, Typography } from "@mui/material";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import LoginIcon from "@mui/icons-material/Login";
import { useNotify } from "../components/NotifyProvider.tsx";

// sshHost reads the SSH endpoint the server injected into the SPA shell, so the
// example scp command targets the real deployment. Falls back to the browser
// host for local/dev where the meta tag may be empty.
function sshHost(): string {
  const meta = document.querySelector<HTMLMetaElement>('meta[name="sshbin-host"]');
  return meta?.content?.trim() || window.location.hostname;
}

export function Landing() {
  const steps = [
    {
      title: "Upload over SSH",
      body: null,
      command: `scp my-log-file.log ${sshHost()}:`,
    },
    {
      title: "Configure",
      body: "Open the setup link printed in your terminal to set expiry, a password, or access policy.",
      command: null,
    },
    {
      title: "Share",
      body: "Send the link or QR code to anyone.",
      command: null,
    },
  ];

  return (
    <Stack spacing={6} sx={{ py: { xs: 2, sm: 6 } }}>
      <Stack spacing={2} sx={{ alignItems: "flex-start" }}>
        <Typography variant="h3" component="h1" sx={{ fontWeight: 700 }}>
          Drop a file. Share a link.
        </Typography>
        <Typography variant="h6" component="p" color="text.secondary" sx={{ fontWeight: 400 }}>
          Upload straight from any server with the tools you already have — no account, no client to
          install.
        </Typography>
        <Button
          component={RouterLink}
          to="/login"
          variant="contained"
          size="large"
          startIcon={<LoginIcon />}
          sx={{ mt: 1 }}
        >
          Sign in
        </Button>
      </Stack>

      <Stack spacing={2} component="ol" sx={{ listStyle: "none", p: 0, m: 0 }}>
        {steps.map((step, i) => (
          <Paper key={step.title} variant="outlined" component="li" sx={{ p: 2 }}>
            <Stack direction="row" spacing={2} sx={{ alignItems: "flex-start" }}>
              <StepNumber n={i + 1} />
              <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                <Typography variant="h6" component="h3">
                  {step.title}
                </Typography>
                {step.body && (
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                    {step.body}
                  </Typography>
                )}
                {step.command && <Command text={step.command} />}
              </Box>
            </Stack>
          </Paper>
        ))}
      </Stack>
    </Stack>
  );
}

function StepNumber({ n }: { n: number }) {
  return (
    <Box
      aria-hidden
      sx={{
        flexShrink: 0,
        width: 32,
        height: 32,
        borderRadius: "50%",
        display: "grid",
        placeItems: "center",
        bgcolor: "primary.main",
        color: "primary.contrastText",
        fontWeight: 700,
      }}
    >
      {n}
    </Box>
  );
}

function Command({ text }: { text: string }) {
  const notify = useNotify();
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      notify("Could not copy to clipboard.", "error");
    }
  };

  return (
    <Stack
      direction="row"
      spacing={1}
      sx={{
        mt: 1,
        alignItems: "center",
        bgcolor: "action.hover",
        borderRadius: 1,
        pl: 1.5,
        pr: 0.5,
        py: 0.5,
      }}
    >
      <Box
        component="code"
        sx={{ flexGrow: 1, fontFamily: "monospace", fontSize: "0.875rem", overflowX: "auto" }}
      >
        {text}
      </Box>
      <Tooltip title={copied ? "Copied" : "Copy"}>
        <IconButton size="small" aria-label="copy command" onClick={() => void copy()}>
          <ContentCopyIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Stack>
  );
}
