import { useState } from "react";
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  FormLabel,
  Radio,
  RadioGroup,
  Stack,
  TextField,
} from "@mui/material";
import SaveIcon from "@mui/icons-material/Save";
import { ApiError, api, type Expiry, type Share } from "../api/client.ts";

export function SetupDialog({
  share,
  open,
  onClose,
  onSaved,
}: {
  share: Share;
  open: boolean;
  onClose: () => void;
  onSaved: (updated: Share) => void;
}) {
  // The stored expiry timestamp can't be mapped back to its preset, so default
  // to "never" when there is none and let a save re-apply a preset otherwise.
  const [expires, setExpires] = useState<Expiry>("never");
  const [visibility, setVisibility] = useState<"public" | "private">(
    share.public ? "public" : "private",
  );
  const [emails, setEmails] = useState(share.allowedEmails.join("\n"));
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function save() {
    setBusy(true);
    setError("");
    try {
      const updated = await api.setupShare(share.id, {
        expires,
        visibility,
        emails: emails
          .split("\n")
          .map((e) => e.trim())
          .filter(Boolean),
        password,
      });
      onSaved(updated);
      onClose();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not save settings.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {share.configured ? "Edit share" : "Set up share"} — {share.fileName}
      </DialogTitle>
      <DialogContent>
        <Stack spacing={3} sx={{ pt: 1 }}>
          <FormControl>
            <FormLabel>Expiry</FormLabel>
            <RadioGroup value={expires} onChange={(e) => setExpires(e.target.value as Expiry)}>
              <FormControlLabel value="1h" control={<Radio />} label="1 hour" />
              <FormControlLabel value="24h" control={<Radio />} label="24 hours" />
              <FormControlLabel value="168h" control={<Radio />} label="7 days" />
              <FormControlLabel value="never" control={<Radio />} label="Never" />
            </RadioGroup>
          </FormControl>

          <FormControl>
            <FormLabel>Visibility</FormLabel>
            <RadioGroup
              value={visibility}
              onChange={(e) => setVisibility(e.target.value as "public" | "private")}
            >
              <FormControlLabel
                value="public"
                control={<Radio />}
                label="Public — anyone with the link"
              />
              <FormControlLabel
                value="private"
                control={<Radio />}
                label="Private — only people I list"
              />
            </RadioGroup>
            {visibility === "private" && (
              <TextField
                label="Allowed emails (one per line)"
                multiline
                rows={3}
                value={emails}
                onChange={(e) => setEmails(e.target.value)}
                placeholder="alice@example.com"
                fullWidth
                sx={{ mt: 1 }}
              />
            )}
          </FormControl>

          <TextField
            type="password"
            label="Extra password (optional)"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={share.hasPassword ? "leave blank to keep current" : "no password"}
            autoComplete="new-password"
            fullWidth
          />

          {error && <Alert severity="error">{error}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          startIcon={<SaveIcon />}
          onClick={() => void save()}
          disabled={busy}
        >
          {busy ? "Saving…" : "Save settings"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
