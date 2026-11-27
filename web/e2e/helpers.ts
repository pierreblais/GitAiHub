import { expect, type Page } from "@playwright/test";

export const USER = process.env.GITAIHUB_E2E_USER ?? "pierre";
export const PASSWORD = process.env.GITAIHUB_E2E_PASSWORD ?? "gitaihub-demo";

export async function signIn(page: Page) {
  await page.goto("/login");
  await page.getByLabel("Username").fill(USER);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: /sign in/i }).click();
  await expect(page).toHaveURL("/");
}
