import { useCallback, useEffect, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  IconButton,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import { ApiError, api, errMessage, type SSHKey } from "../api/client.ts";
import { useNotify } from "../components/NotifyProvider.tsx";
import { useConfirm } from "../components/useConfirm.tsx";

export function SSHKeys() {
  const notify = useNotify();
  const { confirm, dialog: confirmDialog } = useConfirm();
  const [keys, setKeys] = useState<SSHKey[] | null>(null);
  const [error, setError] = useState("");
  const [title, setTitle] = useState("");
  const [key, setKey] = useState("");
  const [adding, setAdding] = useState(false);

  const load = useCallback(async () => {
    setError("");
    try {
      setKeys(await api.keys());
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load keys.");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function add() {
    if (!key.trim()) return;
    setAdding(true);
    try {
      await api.addKey(title.trim(), key.trim());
      notify("Key added");
      setTitle("");
      setKey("");
      await load();
    } catch (err) {
      notify(errMessage(err, "Could not add key."), "error");
    } finally {
      setAdding(false);
    }
  }

  async function remove(k: SSHKey) {
    const ok = await confirm({
      title: "Delete SSH key",
      message: `Delete ${k.title || k.fingerprint}? Uploads with this key will no longer be attributed to you.`,
      confirmLabel: "Delete",
      danger: true,
    });
    if (!ok) return;
    try {
      await api.deleteKey(k.id);
      notify("Key deleted");
      await load();
    } catch (err) {
      notify(errMessage(err, "Could not delete key."), "error");
    }
  }

  return (
    <Stack spacing={3} sx={{ maxWidth: 640 }}>
      <Typography variant="h5" component="h1">
        SSH keys
      </Typography>
      <Typography variant="body2" color="text.secondary">
        Add your public keys to upload over SFTP as yourself. Uploads authenticated with a
        registered key appear in your shares automatically.
      </Typography>

      <Card variant="outlined">
        <CardContent>
          <Stack spacing={2}>
            <Typography variant="h6">Add a key</Typography>
            <TextField
              label="Title"
              placeholder="Work laptop"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              size="small"
              fullWidth
            />
            <TextField
              label="Public key"
              placeholder="ssh-ed25519 AAAA... user@host"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              multiline
              minRows={2}
              fullWidth
            />
            <Box>
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                disabled={adding || !key.trim()}
                onClick={() => void add()}
              >
                Add key
              </Button>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      {error ? (
        <Alert severity="error">{error}</Alert>
      ) : !keys ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 6 }}>
          <CircularProgress />
        </Box>
      ) : keys.length === 0 ? (
        <Typography color="text.secondary">No keys yet.</Typography>
      ) : (
        <Stack spacing={1.5}>
          {keys.map((k) => (
            <Card key={k.id} variant="outlined">
              <CardContent
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  gap: 2,
                  "&:last-child": { pb: 2 },
                }}
              >
                <Box sx={{ minWidth: 0 }}>
                  <Typography noWrap>{k.title || "(untitled)"}</Typography>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{ fontFamily: "monospace", wordBreak: "break-all" }}
                  >
                    {k.fingerprint}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Added {new Date(k.createdAt).toLocaleDateString()}
                    {k.lastUsedAt
                      ? ` · Last used ${new Date(k.lastUsedAt).toLocaleDateString()}`
                      : " · Never used"}
                  </Typography>
                </Box>
                <Tooltip title="Delete">
                  <IconButton
                    color="error"
                    aria-label={`delete ${k.title || k.fingerprint}`}
                    onClick={() => void remove(k)}
                  >
                    <DeleteIcon />
                  </IconButton>
                </Tooltip>
              </CardContent>
            </Card>
          ))}
        </Stack>
      )}

      {confirmDialog}
    </Stack>
  );
}
