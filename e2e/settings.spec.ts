import { test, expect } from "@playwright/test";
import {
  getDefaultE2EWorkspaceName,
  loginAsDefault,
  sidebarLink,
  workspaceSwitcherByName,
} from "./helpers";

test.describe("Settings", () => {
  test("updating workspace name reflects in sidebar immediately", async ({
    page,
  }) => {
    await loginAsDefault(page);

    await sidebarLink(page, "Settings").click();
    await page.waitForURL("**/settings");
    await page.getByRole("tab", { name: "General" }).click();

    const nameInput = page
      .locator('input[type="text"]')
      .first();
    await nameInput.clear();
    const newName = "Renamed WS " + Date.now();
    await nameInput.fill(newName);

    await page.locator("button", { hasText: "Save" }).click();

    await expect(page.getByText("Workspace settings saved").first()).toBeVisible({ timeout: 5000 });
    await expect(workspaceSwitcherByName(page, new RegExp(newName))).toBeVisible();

    await nameInput.clear();
    await nameInput.fill(getDefaultE2EWorkspaceName());
    await page.locator("button", { hasText: "Save" }).click();
    await expect(page.getByText("Workspace settings saved").first()).toBeVisible({ timeout: 5000 });
  });
});
