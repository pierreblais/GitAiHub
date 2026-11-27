import { expect, test } from "@playwright/test";
import { signIn } from "./helpers";

// Walks the demo data the seeder creates (see api/cmd/gitaihub-seed).

test.beforeEach(async ({ page }) => {
  await signIn(page);
});

test("dashboard lists the demo repositories and agent activity", async ({ page }) => {
  for (const name of ["orbit", "nebula-web", "agent-playbooks"]) {
    await expect(page.getByTestId("top-repositories").getByRole("link", { name: `gitai/${name}` })).toBeVisible();
  }
  await expect(page.getByText("Recent agent activity")).toBeVisible();
  await expect(page.getByText("Open reviews")).toBeVisible();
});

test("code, blame and history of a file", async ({ page }) => {
  await page.goto("/repositories/orbit");
  await expect(page.getByRole("heading", { name: "README.md" })).toBeVisible();
  await page.getByRole("link", { name: "limiter", exact: true }).click();
  await page.getByRole("link", { name: "limiter.go", exact: true }).click();
  await expect(page.getByText("package limiter").first()).toBeVisible();
  await page.getByRole("link", { name: "Blame" }).click();
  await expect(page.getByText("Lines by agent")).toBeVisible();
});

test("commit shows its landing and diff", async ({ page }) => {
  await page.goto("/repositories/orbit/commits");
  await page.locator('a[href*="/commit/"]').first().click();
  await expect(page.getByText(/landed on/)).toBeVisible();
  await expect(page.getByText(/changed file/)).toBeVisible();
});

test("auto-revert and conflict resolution are visible in history", async ({ page }) => {
  await page.goto("/repositories/orbit/commits");
  await expect(page.getByText(/^Revert "/).first()).toBeVisible();
  await page.goto("/agents/builder");
  await expect(page.getByText("Conflicts resolved")).toBeVisible();
});

test("review shows rules, verdicts and accepts a comment", async ({ page }) => {
  await page.goto("/repositories/orbit/reviews/1");
  await expect(page.getByText("Requirements")).toBeVisible();
  await expect(page.getByText("Changed files, by class")).toBeVisible();
  const note = `e2e comment ${Date.now()}`;
  await page.getByPlaceholder("Leave a comment").fill(note);
  await page.getByRole("button", { name: "Comment", exact: true }).click();
  await expect(page.getByText(note)).toBeVisible();
});

test("search finds code", async ({ page }) => {
  await page.goto("/repositories/orbit/search?q=Limiter");
  await expect(page.getByText("limiter/limiter.go").first()).toBeVisible();
});

test("unknown repository is a 404 page", async ({ page }) => {
  const resp = await page.goto("/repositories/does-not-exist");
  expect(resp?.status()).toBe(404);
});

test("global navigation drawer and repository About panel", async ({ page }) => {
  await page.getByRole("button", { name: "Open global navigation menu" }).click();
  const drawer = page.getByRole("complementary", { name: "Global navigation" });
  await expect(drawer.getByRole("link", { name: "gitai/orbit" })).toBeVisible();
  await drawer.getByRole("link", { name: "gitai/orbit" }).click();
  await expect(page.getByRole("heading", { name: "About" })).toBeVisible();
  await expect(page.getByRole("link", { name: /\d+ Commits?/ })).toBeVisible();
});

test("all reviews across repositories", async ({ page }) => {
  await page.goto("/reviews");
  await expect(page.getByTestId("review-row").first()).toBeVisible();
});
