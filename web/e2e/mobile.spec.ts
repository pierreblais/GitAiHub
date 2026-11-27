import { expect, test } from "@playwright/test";
import { signIn } from "./helpers";

test("repository page fits a phone screen", async ({ page }) => {
  await signIn(page);
  await page.goto("/repositories/orbit");
  await expect(page.getByRole("heading", { name: "README.md" })).toBeVisible();
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(overflow).toBeLessThanOrEqual(0);
});
