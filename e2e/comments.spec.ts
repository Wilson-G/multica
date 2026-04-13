import { test, expect } from "@playwright/test";
import { createTestApi, loginAsDefault } from "./helpers";
import type { TestApiClient } from "./fixtures";

test.describe("Comments", () => {
  let api: TestApiClient;
  let issue: { id: string };

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    issue = await api.createIssue("E2E Comment Test " + Date.now());
    await loginAsDefault(page);
  });

  test.afterEach(async () => {
    await api.cleanup();
  });

  test("can add a comment on an issue", async ({ page }) => {
    await page.goto(`/issues/${issue.id}`);
    await page.waitForURL(/\/issues\/[\w-]+/);

    await expect(page.locator("text=Properties")).toBeVisible();

    const commentText = "E2E comment " + Date.now();
    const commentInput = page.locator('[contenteditable="true"]').last();
    await expect(commentInput).toBeVisible();
    await commentInput.fill(commentText);

    await page
      .locator("button")
      .filter({ has: page.locator("svg.lucide-arrow-up") })
      .last()
      .click();

    await expect(page.locator(`text=${commentText}`)).toBeVisible({
      timeout: 5000,
    });
  });

  test("comment submit button is disabled when empty", async ({ page }) => {
    await page.goto(`/issues/${issue.id}`);
    await page.waitForURL(/\/issues\/[\w-]+/);

    await expect(page.locator("text=Properties")).toBeVisible();

    const submitBtn = page
      .locator("button")
      .filter({ has: page.locator("svg.lucide-arrow-up") })
      .last();
    await expect(submitBtn).toBeDisabled();
  });
});
