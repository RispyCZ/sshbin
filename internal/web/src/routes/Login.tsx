import { useState } from "react";
import { Box, Button, Card, CardContent, Stack, TextField, Typography } from "@mui/material";
import LoginIcon from "@mui/icons-material/Login";
import SendIcon from "@mui/icons-material/Send";
import { api, errMessage } from "../api/client.ts";
import { Logo } from "../components/Logo.tsx";
import { useNotify } from "../components/NotifyProvider.tsx";

export function Login({ onAuthed }: { onAuthed: () => Promise<void> }) {
  const notify = useNotify();
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [masked, setMasked] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);

  async function submitEmail(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const { maskedEmail } = await api.login(email.trim());
      setMasked(maskedEmail);
      setStep("code");
    } catch (err) {
      notify(errMessage(err, "Could not send a code."), "error");
    } finally {
      setBusy(false);
    }
  }

  async function submitCode(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.verify(email.trim(), code.trim());
      notify("Signed in");
      await onAuthed();
    } catch (err) {
      notify(errMessage(err, "Could not verify."), "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Box sx={{ maxWidth: 380, mx: "auto", mt: 8 }}>
      <Card variant="outlined">
        <CardContent sx={{ p: 4 }}>
          <Logo height={30} sx={{ mb: 3 }} />
          {step === "email" ? (
            <Stack component="form" spacing={2} onSubmit={submitEmail}>
              <Typography variant="h5" component="h1">
                Sign in
              </Typography>
              <Typography variant="body2" color="text.secondary">
                We&apos;ll email you a one-time code.
              </Typography>
              <TextField
                label="Email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoFocus
                fullWidth
              />
              <Button type="submit" variant="contained" disabled={busy} startIcon={<SendIcon />}>
                {busy ? "Sending…" : "Send code"}
              </Button>
            </Stack>
          ) : (
            <Stack component="form" spacing={2} onSubmit={submitCode}>
              <Typography variant="h5" component="h1">
                Enter code
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Sent to {masked}.
              </Typography>
              <TextField
                label="Code"
                placeholder="123456"
                slotProps={{ htmlInput: { inputMode: "numeric" } }}
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
                autoFocus
                fullWidth
              />
              <Button type="submit" variant="contained" disabled={busy} startIcon={<LoginIcon />}>
                {busy ? "Verifying…" : "Verify"}
              </Button>
            </Stack>
          )}
        </CardContent>
      </Card>
    </Box>
  );
}
