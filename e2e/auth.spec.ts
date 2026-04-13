import { test, expect } from "@playwright/test";
import { loginAsDefault, openWorkspaceMenu, workspaceSwitcher } from "./helpers";

test.describe("Authentication", () => {
  test("login page renders correctly", async ({ page }) => {
    await page.goto("/login");

    await expect(page.getByText("Sign in to Multica")).toBeVisible();
    await expect(page.getByRole("textbox", { name: "Email" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Continue" })).toBeVisible();
  });

  test("login and redirect to /issues", async ({ page }) => {
    await loginAsDefault(page);

    await expect(page).toHaveURL(/\/issues/);
    await expect(workspaceSwitcher(page)).toBeVisible();
    await expect(page.getByRole("button", { name: /New Issue/ })).toBeVisible();
  });

  test("unauthenticated user is redirected to landing page", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => {
      localStorage.removeItem("multica_token");
      localStorage.removeItem("multica_workspace_id");
    });

    await page.goto("/issues");
    await expect(page).toHaveURL(/\/$/);
  });

  test("logout redirects to landing page", async ({ page }) => {
    await loginAsDefault(page);

    await openWorkspaceMenu(page);
    await page.getByRole("menuitem", { name: "Log out" }).click();
    await expect(page).toHaveURL(/\/$/);
  });
});
