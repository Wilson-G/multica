import { test, expect } from "@playwright/test";
import { loginAsDefault, sidebarLink } from "./helpers";

test.describe("Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsDefault(page);
  });

  test("sidebar navigation works", async ({ page }) => {
    await sidebarLink(page, "Inbox").click();
    await page.waitForURL("**/inbox");
    await expect(page).toHaveURL(/\/inbox/);

    await sidebarLink(page, "Agents").click();
    await page.waitForURL("**/agents");
    await expect(page).toHaveURL(/\/agents/);

    await sidebarLink(page, "Issues").click();
    await page.waitForURL("**/issues");
    await expect(page).toHaveURL(/\/issues/);
  });

  test("settings page loads via sidebar", async ({ page }) => {
    await sidebarLink(page, "Settings").click();
    await page.waitForURL("**/settings");

    await expect(page.getByText("Settings").first()).toBeVisible();
    await expect(page.getByRole("tab", { name: "General" })).toBeVisible();
    await expect(page.getByRole("tab", { name: "Members" })).toBeVisible();
  });

  test("agents page shows agent list", async ({ page }) => {
    await sidebarLink(page, "Agents").click();
    await page.waitForURL("**/agents");

    await expect(page.locator("text=Agents").first()).toBeVisible();
  });
});
