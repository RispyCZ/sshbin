import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";
import { ApiError, api, errMessage } from "./client";

// jsonResponse builds a fetch Response with a JSON body and given status.
function jsonResponse(body: unknown, status = 200): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

const fetchMock = vi.fn<typeof fetch>();

beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  fetchMock.mockReset();
  vi.unstubAllGlobals();
});

describe("request", () => {
  it("sends JSON content-type and same-origin credentials by default", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ email: "a@b.c" }));

    await api.session();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/session");
    expect(init?.credentials).toBe("same-origin");
    const headers = new Headers(init?.headers);
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("parses and returns the JSON body on success", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ email: "user@example.com" }));

    const session = await api.session();

    expect(session).toEqual({ email: "user@example.com" });
  });

  it("returns undefined for 204 responses without parsing a body", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));

    await expect(api.logout()).resolves.toBeUndefined();
  });

  it("throws ApiError carrying status, server message, and code", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: "nope", code: "forbidden" }, 403));

    const err = await api.session().catch((e: unknown) => e);

    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(403);
    expect((err as ApiError).message).toBe("nope");
    expect((err as ApiError).code).toBe("forbidden");
  });

  it("falls back to statusText when the error body has no message", async () => {
    fetchMock.mockResolvedValue(
      new Response("not json", {
        status: 500,
        statusText: "Internal Server Error",
      }),
    );

    const err = (await api.session().catch((e: unknown) => e)) as ApiError;

    expect(err).toBeInstanceOf(ApiError);
    expect(err.message).toBe("Internal Server Error");
    expect(err.code).toBeUndefined();
  });
});

describe("api endpoints", () => {
  it("login posts the email and returns the masked address", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ maskedEmail: "u***@b.c" }));

    const res = await api.login("user@example.com");

    expect(res).toEqual({ maskedEmail: "u***@b.c" });
    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/login");
    expect(init?.method).toBe("POST");
    expect(init?.body).toBe(JSON.stringify({ email: "user@example.com" }));
  });

  it("verify posts email and code", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ email: "user@example.com" }));

    await api.verify("user@example.com", "123456");

    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/verify");
    expect(init?.method).toBe("POST");
    expect(init?.body).toBe(JSON.stringify({ email: "user@example.com", code: "123456" }));
  });

  it("deleteShare targets the share id with DELETE", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));

    await api.deleteShare("abc123");

    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/shares/abc123");
    expect(init?.method).toBe("DELETE");
  });

  it("setupShare posts the setup input to the share id", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ id: "abc123" }));
    const input = {
      expires: "24h" as const,
      visibility: "private" as const,
      emails: ["x@y.z"],
      password: "pw",
    };

    await api.setupShare("abc123", input);

    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/setup/abc123");
    expect(init?.method).toBe("POST");
    expect(init?.body).toBe(JSON.stringify(input));
  });

  it("saveProfile puts the defaultPublic flag", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));

    await api.saveProfile(true);

    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/profile");
    expect(init?.method).toBe("PUT");
    expect(init?.body).toBe(JSON.stringify({ defaultPublic: true }));
  });

  it("unlockShare posts the password to the public share endpoint", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ unlocked: true, downloadURL: "/d/abc" }));

    const res = await api.unlockShare("abc", "secret");

    expect(res).toEqual({ unlocked: true, downloadURL: "/d/abc" });
    const [path, init] = fetchMock.mock.calls[0];
    expect(path).toBe("/api/s/abc");
    expect(init?.method).toBe("POST");
    expect(init?.body).toBe(JSON.stringify({ password: "secret" }));
  });
});

describe("errMessage", () => {
  it("returns the ApiError message when given an ApiError", () => {
    expect(errMessage(new ApiError(400, "bad input"), "fallback")).toBe("bad input");
  });

  it("returns the fallback for non-ApiError values", () => {
    expect(errMessage(new Error("boom"), "fallback")).toBe("fallback");
    expect(errMessage("boom", "fallback")).toBe("fallback");
    expect(errMessage(null, "fallback")).toBe("fallback");
  });
});
