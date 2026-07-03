import { expect, test } from "@playwright/test";
import { installApiMock } from "./api-mock.ts";

test("landing page renders the hero and the injected ssh host in the scp command", async ({
  page,
}) => {
  await installApiMock(page);
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "Drop a file. Share a link." })).toBeVisible();
  // index.html injects meta[name="sshbin-host"]=ssh.example.com; Landing reads it.
  await expect(page.getByText("scp my-log-file.log ssh.example.com:")).toBeVisible();
});

test("Sign in button routes to the login form", async ({ page }) => {
  await installApiMock(page);
  await page.goto("/");

  await page.getByRole("link", { name: "Sign in" }).click();

  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});
