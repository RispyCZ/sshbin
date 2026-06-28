import { useState } from "react";
import { Alert, Box, Button, Card, CardContent, Stack, TextField, Typography } from "@mui/material";
import LoginIcon from "@mui/icons-material/Login";
import SendIcon from "@mui/icons-material/Send";
import { ApiError, api } from "../api/client.ts";
import { Logo } from "../components/Logo.tsx";

export function Login({ onAuthed }: { onAuthed: () => Promise<void> }) {
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [masked, setMasked] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submitEmail(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const { maskedEmail } = await api.login(email.trim());
      setMasked(maskedEmail);
      setStep("code");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not send a code.");
    } finally {
      setBusy(false);
    }
  }

  async function submitCode(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.verify(email.trim(), code.trim());
      await onAuthed();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not verify.");
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
              {error && <Alert severity="error">{error}</Alert>}
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
              {error && <Alert severity="error">{error}</Alert>}
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
