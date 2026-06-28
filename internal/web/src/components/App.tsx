import { useCallback, useEffect, useState } from "react";
import { Navigate, Link, Route, Routes, useLocation } from "react-router-dom";
import { ApiError, api, type Session } from "../api/client.ts";
import { Login } from "../routes/Login.tsx";
import { Shares } from "../routes/Shares.tsx";

type Auth = { state: "loading" } | { state: "out" } | { state: "in"; session: Session };

export function App() {
  const [auth, setAuth] = useState<Auth>({ state: "loading" });

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
    return <div className="center muted">Loading…</div>;
  }

  return (
    <div className="app">
      <Header
        session={auth.state === "in" ? auth.session : null}
        onLogout={async () => {
          await api.logout();
          setAuth({ state: "out" });
        }}
      />
      <main className="container">
        <Routes>
          <Route
            path="/login"
            element={
              auth.state === "in" ? <Navigate to="/" replace /> : <Login onAuthed={refresh} />
            }
          />
          <Route
            path="/"
            element={
              <RequireAuth authed={auth.state === "in"}>
                <Shares />
              </RequireAuth>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}

function Header({ session, onLogout }: { session: Session | null; onLogout: () => void }) {
  return (
    <header className="header">
      <Link to="/" className="brand">
        sshbin
      </Link>
      {session && (
        <div className="user">
          <span className="muted">{session.email}</span>
          <button type="button" onClick={onLogout}>
            Log out
          </button>
        </div>
      )}
    </header>
  );
}

function RequireAuth({ authed, children }: { authed: boolean; children: React.ReactNode }) {
  const location = useLocation();
  if (!authed) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}
