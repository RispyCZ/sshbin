import { useCallback, useEffect, useState } from "react";
import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  IconButton,
  Link,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from "@mui/material";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import ShareIcon from "@mui/icons-material/Share";
import TuneIcon from "@mui/icons-material/Tune";
import { ApiError, api, errMessage, type Share } from "../api/client.ts";
import { useNotify } from "../components/NotifyProvider.tsx";
import { SetupDialog } from "../components/SetupDialog.tsx";
import { ShareDialog } from "../components/ShareDialog.tsx";
import { useConfirm } from "../components/useConfirm.tsx";

function formatExpiry(s: Share): string {
  if (s.expired) return "Expired";
  if (!s.expiresAt) return "Never";
  return new Date(s.expiresAt).toLocaleString();
}

export function Shares() {
  const notify = useNotify();
  const [shares, setShares] = useState<Share[] | null>(null);
  const [error, setError] = useState("");
  const [setupTarget, setSetupTarget] = useState<Share | null>(null);
  const [qrTarget, setQrTarget] = useState<Share | null>(null);
  const { confirm, dialog: confirmDialog } = useConfirm();

  const load = useCallback(async () => {
    setError("");
    try {
      setShares(await api.shares());
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load shares.");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function remove(s: Share) {
    const ok = await confirm({
      title: "Delete share",
      message: `Delete ${s.fileName}? This cannot be undone.`,
      confirmLabel: "Delete",
      danger: true,
    });
    if (!ok) return;
    try {
      await api.deleteShare(s.id);
      notify("Share deleted");
      await load();
    } catch (err) {
      notify(errMessage(err, "Could not delete share."), "error");
    }
  }

  if (error) return <Alert severity="error">{error}</Alert>;
  if (!shares) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 6 }}>
        <CircularProgress />
      </Box>
    );
  }
  if (shares.length === 0) {
    return <Typography color="text.secondary">No shares yet.</Typography>;
  }

  return (
    <>
      <Typography variant="h5" component="h1" sx={{ mb: 2 }}>
        My shares
      </Typography>
      <TableContainer component={Paper} variant="outlined">
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>File</TableCell>
              <TableCell>Visibility</TableCell>
              <TableCell>Expires</TableCell>
              <TableCell>Created</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {shares.map((s) => (
              <TableRow key={s.id} sx={{ opacity: s.expired ? 0.5 : 1 }}>
                <TableCell>
                  {s.configured ? (
                    <Link href={s.shareURL} underline="hover">
                      {s.fileName}
                    </Link>
                  ) : (
                    <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
                      <span>{s.fileName}</span>
                      <Chip size="small" variant="outlined" label="not set up" />
                    </Stack>
                  )}
                </TableCell>
                <TableCell>
                  {s.public ? (
                    <Chip size="small" color="success" label="Public" />
                  ) : (
                    <Chip
                      size="small"
                      variant="outlined"
                      label={`Private (${s.allowedEmails.length})`}
                    />
                  )}
                </TableCell>
                <TableCell>{formatExpiry(s)}</TableCell>
                <TableCell>{new Date(s.createdAt).toLocaleDateString()}</TableCell>
                <TableCell align="right">
                  {s.configured && (
                    <Tooltip title="Share link & QR">
                      <IconButton
                        size="small"
                        aria-label={`share ${s.fileName}`}
                        onClick={() => setQrTarget(s)}
                      >
                        <ShareIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  )}
                  <Tooltip title={s.configured ? "Edit" : "Set up"}>
                    <IconButton
                      size="small"
                      aria-label={`${s.configured ? "edit" : "set up"} ${s.fileName}`}
                      onClick={() => setSetupTarget(s)}
                    >
                      {s.configured ? <EditIcon fontSize="small" /> : <TuneIcon fontSize="small" />}
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Delete">
                    <IconButton
                      size="small"
                      color="error"
                      aria-label={`delete ${s.fileName}`}
                      onClick={() => void remove(s)}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {setupTarget && (
        <SetupDialog
          share={setupTarget}
          open
          onClose={() => setSetupTarget(null)}
          onSaved={() => {
            notify("Share saved");
            void load();
          }}
        />
      )}
      {qrTarget && <ShareDialog share={qrTarget} open onClose={() => setQrTarget(null)} />}
      {confirmDialog}
    </>
  );
}
