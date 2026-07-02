import { createContext, use, useCallback, useMemo, useState, type ReactNode } from "react";
import { Alert, Snackbar, type AlertColor } from "@mui/material";

type Notify = (message: string, severity?: AlertColor) => void;

const NotifyContext = createContext<Notify>(() => {});

export function useNotify(): Notify {
  return use(NotifyContext);
}

interface Toast {
  key: number;
  message: string;
  severity: AlertColor;
}

export function NotifyProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<Toast | null>(null);
  const [open, setOpen] = useState(false);

  // notify swaps the toast content and (re)opens. The key forces a fresh
  // Snackbar so rapid successive calls restart the auto-hide timer cleanly.
  const notify = useCallback<Notify>((message, severity = "success") => {
    setToast({ key: Date.now(), message, severity });
    setOpen(true);
  }, []);

  return (
    <NotifyContext value={useMemo(() => notify, [notify])}>
      {children}
      {toast && (
        <Snackbar
          key={toast.key}
          open={open}
          autoHideDuration={toast.severity === "error" ? 5000 : 3000}
          onClose={(_, reason) => {
            if (reason === "clickaway") return;
            setOpen(false);
          }}
          anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
        >
          <Alert severity={toast.severity} variant="filled" onClose={() => setOpen(false)}>
            {toast.message}
          </Alert>
        </Snackbar>
      )}
    </NotifyContext>
  );
}
