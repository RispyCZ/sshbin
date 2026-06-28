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
  hasPassword: boolean;
  createdAt: string;
  shareURL: string;
}

export type Expiry = "1h" | "24h" | "168h" | "never";

export interface SetupInput {
  expires: Expiry;
  visibility: "public" | "private";
  emails: string[];
  password: string;
}

export interface Profile {
  email: string;
  defaultPublic: boolean;
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
  setupShare: (id: string, input: SetupInput) =>
    request<Share>(`/api/setup/${id}`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
  profile: () => request<Profile>("/api/profile"),
  saveProfile: (defaultPublic: boolean) =>
    request<void>("/api/profile", {
      method: "PUT",
      body: JSON.stringify({ defaultPublic }),
    }),
  deleteAllData: () => request<void>("/api/profile", { method: "DELETE" }),
};
