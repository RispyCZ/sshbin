import { useCallback, useEffect, useState } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import DownloadIcon from "@mui/icons-material/Download";
import HomeIcon from "@mui/icons-material/Home";
import LockOpenIcon from "@mui/icons-material/LockOpen";
import { ApiError, api, type ShareView } from "../api/client.ts";

interface Denied {
  message: string;
  loginRequired: boolean;
}

export function Setup() {
  const { id = "" } = useParams();
  const [view, setView] = useState<ShareView | null>(null);
  const [denied, setDenied] = useState<Denied | null>(null);
  const [loading, setLoading] = useState(true);
  const [password, setPassword] = useState("");
  const [unlockError, setUnlockError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setDenied(null);
    try {
      setView(await api.shareView(id));
    } catch (err) {
      if (err instanceof ApiError) {
        setDenied({ message: err.message, loginRequired: err.code === "login_required" });
      } else {
        setDenied({ message: "Something went wrong.", loginRequired: false });
      }
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  async function unlock(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setUnlockError("");
    try {
      const res = await api.unlockShare(id, password);
      setView((v) => (v ? { ...v, unlocked: res.unlocked, downloadURL: res.downloadURL } : v));
    } catch (err) {
      setUnlockError(err instanceof ApiError ? err.message : "Could not unlock.");
    } finally {
      setBusy(false);
    }
  }

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 6 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (denied) {
    return (
      <Box sx={{ maxWidth: 420, mx: "auto", mt: 6, textAlign: "center" }}>
        <Alert severity={denied.loginRequired ? "info" : "error"} sx={{ mb: 2 }}>
          {denied.message}
        </Alert>
        {denied.loginRequired ? (
          <Button variant="contained" component={RouterLink} to={`/login?next=/s/${id}`}>
            Sign in
          </Button>
        ) : (
          <Button variant="outlined" component={RouterLink} to="/" startIcon={<HomeIcon />}>
            Back home
          </Button>
        )}
      </Box>
    );
  }

  if (!view) return null;

  const locked = view.requiresPassword && !view.unlocked;

  return (
    <Box sx={{ maxWidth: 420, mx: "auto", mt: 6 }}>
      <Card variant="outlined">
        <CardContent sx={{ p: 4 }}>
          <Stack spacing={2}>
            <Typography variant="h5" component="h1">
              {locked ? "Password required" : "Shared file"}
            </Typography>
            <Typography color="text.secondary" sx={{ wordBreak: "break-all" }}>
              {view.fileName}
            </Typography>
            {view.expiresAt && (
              <Typography variant="body2" color="text.secondary">
                Expires {new Date(view.expiresAt).toLocaleString()}
              </Typography>
            )}
            {locked ? (
              <Stack component="form" spacing={2} onSubmit={unlock}>
                <TextField
                  type="password"
                  label="Password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoFocus
                  fullWidth
                  autoComplete="off"
                />
                {unlockError && <Alert severity="error">{unlockError}</Alert>}
                <Button
                  type="submit"
                  variant="contained"
                  startIcon={<LockOpenIcon />}
                  disabled={busy}
                >
                  {busy ? "Unlocking…" : "Unlock"}
                </Button>
              </Stack>
            ) : (
              <Button
                variant="contained"
                startIcon={<DownloadIcon />}
                href={view.downloadURL}
                download
              >
                Download
              </Button>
            )}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}
