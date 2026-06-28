import { useState } from "react";
import { ApiError, api } from "../api/client.ts";

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
    <div className="card narrow">
      {step === "email" ? (
        <form onSubmit={submitEmail}>
          <h1>Sign in</h1>
          <p className="muted">We&apos;ll email you a one-time code.</p>
          <input
            type="email"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoFocus
          />
          {error && <p className="error">{error}</p>}
          <button type="submit" disabled={busy}>
            {busy ? "Sending…" : "Send code"}
          </button>
        </form>
      ) : (
        <form onSubmit={submitCode}>
          <h1>Enter code</h1>
          <p className="muted">Sent to {masked}.</p>
          <input
            inputMode="numeric"
            placeholder="123456"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            required
            autoFocus
          />
          {error && <p className="error">{error}</p>}
          <button type="submit" disabled={busy}>
            {busy ? "Verifying…" : "Verify"}
          </button>
        </form>
      )}
    </div>
  );
}
