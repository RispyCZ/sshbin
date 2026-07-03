import type { Page, Route } from "@playwright/test";

// Shape mirrors the client.ts Share interface; kept local so the E2E harness
// stays independent of the SPA source under test.
export interface MockShare {
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

export interface ApiMock {
  // session is the /api/session result; null answers 401 (signed out).
  session: { email: string } | null;
  shares: MockShare[];
  maskedEmail: string;
  // code the /api/verify endpoint accepts; any other value answers 401.
  validCode: string;
  // Records DELETE calls so specs can assert the backend was hit.
  deleted: string[];
}

export function makeShare(over: Partial<MockShare> = {}): MockShare {
  return {
    id: "s1",
    fileName: "report.pdf",
    configured: true,
    public: true,
    expired: false,
    expiresAt: null,
    allowedEmails: [],
    hasPassword: false,
    createdAt: "2026-06-01T12:00:00Z",
    shareURL: "https://ssh.example.com/s/s1",
    ...over,
  };
}

function json(route: Route, status: number, body: unknown) {
  return route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });
}

// installApiMock intercepts every /api/* request so the SPA runs against
// deterministic JSON without the Go backend. The returned ApiMock is mutable:
// tweak `state.session` or `state.shares` before/after navigation to drive a
// scenario, and read `state.deleted` to assert on writes.
export async function installApiMock(page: Page, initial: Partial<ApiMock> = {}): Promise<ApiMock> {
  const state: ApiMock = {
    session: null,
    shares: [],
    maskedEmail: "u•••@example.com",
    validCode: "123456",
    deleted: [],
    ...initial,
  };

  // Match on the pathname (not a "**/api/**" glob, which would also catch vite
  // and node_modules module URLs that happen to contain "/api/" and break the
  // dev module graph).
  await page.route(
    (url) => url.pathname.startsWith("/api/"),
    async (route) => {
      const req = route.request();
      const url = new URL(req.url());
      const path = url.pathname;
      const method = req.method();

      if (path === "/api/session") {
        return state.session
          ? json(route, 200, state.session)
          : json(route, 401, { error: "not signed in", code: "login_required" });
      }
      if (path === "/api/login" && method === "POST") {
        return json(route, 200, { maskedEmail: state.maskedEmail });
      }
      if (path === "/api/verify" && method === "POST") {
        const body = req.postDataJSON() as { email: string; code: string };
        if (body.code !== state.validCode) {
          return json(route, 401, { error: "That code is not correct." });
        }
        state.session = { email: body.email };
        return json(route, 200, state.session);
      }
      if (path === "/api/logout" && method === "POST") {
        state.session = null;
        return route.fulfill({ status: 204, body: "" });
      }
      if (path === "/api/shares" && method === "GET") {
        return json(route, 200, state.shares);
      }
      const del = /^\/api\/shares\/(.+)$/.exec(path);
      if (del && method === "DELETE") {
        state.deleted.push(del[1]);
        state.shares = state.shares.filter((s) => s.id !== del[1]);
        return route.fulfill({ status: 204, body: "" });
      }

      return json(route, 404, { error: "unhandled: " + method + " " + path });
    },
  );

  return state;
}
