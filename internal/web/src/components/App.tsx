import { useCallback, useEffect, useState } from "react";
import { Navigate, Link as RouterLink, Route, Routes, useLocation } from "react-router-dom";
import { AppBar, Box, CircularProgress, Container, Stack, Toolbar } from "@mui/material";
import { ApiError, api, errMessage, type Session } from "../api/client.ts";
import { Login } from "../routes/Login.tsx";
import { Profile } from "../routes/Profile.tsx";
import { Setup } from "../routes/Setup.tsx";
import { Shares } from "../routes/Shares.tsx";
import { Logo } from "./Logo.tsx";
import { useNotify } from "./NotifyProvider.tsx";
import { ThemeToggle } from "./ThemeToggle.tsx";
import { UserMenu } from "./UserMenu.tsx";

type Auth = { state: "loading" } | { state: "out" } | { state: "in"; session: Session };

// safeNext returns the post-login redirect target, rejecting absolute or
// protocol-relative URLs to prevent open redirects (mirrors the server).
function safeNext(): string {
  const next = new URLSearchParams(window.location.search).get("next");
  return next && next.startsWith("/") && !next.startsWith("//") ? next : "/";
}

export function App() {
  const [auth, setAuth] = useState<Auth>({ state: "loading" });
  const notify = useNotify();

  const refresh = useCallback(async () => {
    try {
      const session = await api.session();
      setAuth({ state: "in", session });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setAuth({ state: "out" });
        return;
      }
      throw err;
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  if (auth.state === "loading") {
    return (
      <Box sx={{ display: "grid", placeItems: "center", minHeight: "100vh" }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ minHeight: "100vh" }}>
      <Header
        session={auth.state === "in" ? auth.session : null}
        onLogout={async () => {
          try {
            await api.logout();
            setAuth({ state: "out" });
            notify("Signed out");
          } catch (err) {
            notify(errMessage(err, "Could not sign out."), "error");
          }
        }}
      />
      <Container maxWidth="md" sx={{ py: 4 }}>
        <Routes>
          <Route
            path="/login"
            element={
              auth.state === "in" ? (
                <Navigate to={safeNext()} replace />
              ) : (
                <Login onAuthed={refresh} />
              )
            }
          />
          <Route path="/s/:id" element={<Setup />} />
          <Route
            path="/"
            element={
              <RequireAuth authed={auth.state === "in"}>
                <Shares />
              </RequireAuth>
            }
          />
          <Route
            path="/profile"
            element={
              <RequireAuth authed={auth.state === "in"}>
                <Profile onSignedOut={() => setAuth({ state: "out" })} />
              </RequireAuth>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Container>
    </Box>
  );
}

function Header({ session, onLogout }: { session: Session | null; onLogout: () => void }) {
  return (
    <AppBar
      position="static"
      color="transparent"
      elevation={0}
      sx={{ borderBottom: 1, borderColor: "divider" }}
    >
      <Toolbar>
        <Box component={RouterLink} to="/" sx={{ display: "flex", flexGrow: 1 }}>
          <Logo height={26} />
        </Box>
        <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
          <ThemeToggle />
          {session && <UserMenu email={session.email} onLogout={onLogout} />}
        </Stack>
      </Toolbar>
    </AppBar>
  );
}

function RequireAuth({ authed, children }: { authed: boolean; children: React.ReactNode }) {
  const location = useLocation();
  if (!authed) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}
