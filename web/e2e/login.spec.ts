import { expect, test } from "@playwright/test";
import { installApiMock, makeShare } from "./api-mock.ts";

test("email → code → shares: a full sign-in lands on the shares page", async ({ page }) => {
  const state = await installApiMock(page, {
    shares: [makeShare({ fileName: "report.pdf" })],
  });

  await page.goto("/login");

  await page.getByLabel("Email").fill("user@example.com");
  await page.getByRole("button", { name: "Send code" }).click();

  // Code step shows the masked address the server returned.
  await expect(page.getByText(`Sent to ${state.maskedEmail}.`)).toBeVisible();

  // Typing the 6th digit auto-submits via OtpInput.onComplete.
  const digits = page.getByRole("textbox");
  for (let i = 0; i < 6; i++) await digits.nth(i).fill(String(i + 1));

  await expect(page).toHaveURL(/\/shares$/);
  await expect(page.getByRole("heading", { name: "My shares" })).toBeVisible();
  await expect(page.getByRole("link", { name: "report.pdf" })).toBeVisible();
});

test("a wrong code surfaces an error and keeps the user on the code step", async ({ page }) => {
  await installApiMock(page, { validCode: "999999" });

  await page.goto("/login");
  await page.getByLabel("Email").fill("user@example.com");
  await page.getByRole("button", { name: "Send code" }).click();

  const digits = page.getByRole("textbox");
  for (let i = 0; i < 6; i++) await digits.nth(i).fill("1");

  await expect(page.getByText("That code is not correct.")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Enter code" })).toBeVisible();
  await expect(page).toHaveURL(/\/login$/);
});

test("protected /shares redirects a signed-out visitor to login with a next param", async ({
  page,
}) => {
  await installApiMock(page);

  await page.goto("/shares");

  await expect(page).toHaveURL(/\/login\?next=%2Fshares$/);
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});
