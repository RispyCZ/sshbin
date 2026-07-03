import { expect, test } from "@playwright/test";
import { installApiMock, makeShare } from "./api-mock.ts";

test("lists shares with visibility chips for an authed user", async ({ page }) => {
  await installApiMock(page, {
    session: { email: "user@example.com" },
    shares: [
      makeShare({ id: "a", fileName: "public.zip", public: true }),
      makeShare({
        id: "b",
        fileName: "private.txt",
        public: false,
        allowedEmails: ["x@y.z", "a@b.c"],
      }),
    ],
  });

  await page.goto("/shares");

  await expect(page.getByRole("link", { name: "public.zip" })).toBeVisible();
  await expect(page.getByText("Public", { exact: true })).toBeVisible();
  await expect(page.getByText("Private (2)")).toBeVisible();
});

test("shows the empty state when there are no shares", async ({ page }) => {
  await installApiMock(page, { session: { email: "user@example.com" }, shares: [] });

  await page.goto("/shares");

  await expect(page.getByText("No shares yet.")).toBeVisible();
});

test("deleting a share confirms, calls the API, and removes the row", async ({ page }) => {
  const state = await installApiMock(page, {
    session: { email: "user@example.com" },
    shares: [makeShare({ id: "gone", fileName: "trash.log" })],
  });

  await page.goto("/shares");
  await expect(page.getByRole("link", { name: "trash.log" })).toBeVisible();

  await page.getByRole("button", { name: "delete trash.log" }).click();

  // useConfirm dialog: confirmLabel is "Delete".
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText("Delete trash.log? This cannot be undone.")).toBeVisible();
  await dialog.getByRole("button", { name: "Delete" }).click();

  await expect(page.getByText("No shares yet.")).toBeVisible();
  expect(state.deleted).toEqual(["gone"]);
});

test("shows an error alert when the shares request fails", async ({ page }) => {
  await installApiMock(page, { session: { email: "user@example.com" } });
  // Override the shares route to fail after the mock is installed.
  await page.route("**/api/shares", (route) =>
    route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({ error: "boom" }),
    }),
  );

  await page.goto("/shares");

  await expect(page.getByRole("alert")).toHaveText("boom");
});
