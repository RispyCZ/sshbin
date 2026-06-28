import { useCallback, useEffect, useState } from "react";
import { ApiError, api, type Share } from "../api/client.ts";

export function Shares() {
  const [shares, setShares] = useState<Share[] | null>(null);
  const [error, setError] = useState("");

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

  async function remove(id: string) {
    if (!confirm("Delete this share?")) return;
    await api.deleteShare(id);
    await load();
  }

  if (error) return <p className="error">{error}</p>;
  if (!shares) return <p className="muted">Loading shares…</p>;
  if (shares.length === 0) return <p className="muted">No shares yet.</p>;

  return (
    <table className="shares">
      <thead>
        <tr>
          <th>File</th>
          <th>Visibility</th>
          <th>Expires</th>
          <th />
        </tr>
      </thead>
      <tbody>
        {shares.map((s) => (
          <tr key={s.id} className={s.expired ? "expired" : ""}>
            <td>
              <a href={s.shareURL}>{s.fileName}</a>
            </td>
            <td>{s.public ? "Public" : `Private (${s.allowedEmails.length})`}</td>
            <td>{s.expired ? "Expired" : (s.expiresAt ?? "Never")}</td>
            <td>
              <button type="button" onClick={() => void remove(s.id)}>
                Delete
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
