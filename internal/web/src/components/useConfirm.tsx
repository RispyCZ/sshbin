import { useCallback, useState } from "react";
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from "@mui/material";

interface ConfirmOptions {
  title?: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}

type Pending = ConfirmOptions & { resolve: (ok: boolean) => void };

// useConfirm returns a promise-based confirm() to replace window.confirm, plus a
// `dialog` element the caller renders. Resolves true on confirm, false otherwise.
export function useConfirm() {
  const [pending, setPending] = useState<Pending | null>(null);

  const confirm = useCallback(
    (opts: ConfirmOptions) => new Promise<boolean>((resolve) => setPending({ ...opts, resolve })),
    [],
  );

  function settle(ok: boolean) {
    pending?.resolve(ok);
    setPending(null);
  }

  const dialog = (
    <Dialog open={pending !== null} onClose={() => settle(false)} maxWidth="xs" fullWidth>
      {pending && (
        <>
          <DialogTitle>{pending.title ?? "Are you sure?"}</DialogTitle>
          <DialogContent>
            <DialogContentText>{pending.message}</DialogContentText>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => settle(false)}>{pending.cancelLabel ?? "Cancel"}</Button>
            <Button
              variant="contained"
              color={pending.danger ? "error" : "primary"}
              onClick={() => settle(true)}
              autoFocus
            >
              {pending.confirmLabel ?? "Confirm"}
            </Button>
          </DialogActions>
        </>
      )}
    </Dialog>
  );

  return { confirm, dialog };
}
