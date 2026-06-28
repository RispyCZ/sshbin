export interface Session {
  email: string;
}

export interface Share {
  id: string;
  fileName: string;
  configured: boolean;
  public: boolean;
  expired: boolean;
  expiresAt: string | null;
  allowedEmails: string[];
  shareURL: string;
}

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new ApiError(res.status, body?.error ?? res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  session: () => request<Session>("/api/session"),
  login: (email: string) =>
    request<{ maskedEmail: string }>("/api/login", {
      method: "POST",
      body: JSON.stringify({ email }),
    }),
  verify: (email: string, code: string) =>
    request<Session>("/api/verify", {
      method: "POST",
      body: JSON.stringify({ email, code }),
    }),
  logout: () => request<void>("/api/logout", { method: "POST" }),
  shares: () => request<Share[]>("/api/shares"),
  deleteShare: (id: string) => request<void>(`/api/shares/${id}`, { method: "DELETE" }),
};
