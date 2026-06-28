import { useState } from "react";
import {
  Box,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  InputAdornment,
  Stack,
  TextField,
  Tooltip,
} from "@mui/material";
import CheckIcon from "@mui/icons-material/Check";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import type { Share } from "../api/client.ts";

export function ShareDialog({
  share,
  open,
  onClose,
}: {
  share: Share;
  open: boolean;
  onClose: () => void;
}) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(share.shareURL);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable; the field is selectable */
    }
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{share.fileName}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1, alignItems: "center" }}>
          <Box
            component="img"
            src={`/shares/${share.id}/qr`}
            alt="QR code"
            width={200}
            height={200}
            sx={{ borderRadius: 1, bgcolor: "#fff", p: 1 }}
          />
          <TextField
            fullWidth
            size="small"
            value={share.shareURL}
            slotProps={{
              input: {
                readOnly: true,
                endAdornment: (
                  <InputAdornment position="end">
                    <Tooltip title={copied ? "Copied" : "Copy"}>
                      <IconButton edge="end" onClick={() => void copy()} aria-label="copy link">
                        {copied ? (
                          <CheckIcon fontSize="small" />
                        ) : (
                          <ContentCopyIcon fontSize="small" />
                        )}
                      </IconButton>
                    </Tooltip>
                  </InputAdornment>
                ),
              },
            }}
          />
        </Stack>
      </DialogContent>
    </Dialog>
  );
}
