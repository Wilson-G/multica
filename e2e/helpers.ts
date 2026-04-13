import { expect, test, type Page } from "@playwright/test";
import { TestApiClient } from "./fixtures";

const SIDEBAR_LINK_HREFS: Record<string, string> = {
  Inbox: "/inbox",
  "My Issues": "/my-issues",
  Issues: "/issues",
  Agents: "/agents",
  Settings: "/settings",
};

function currentE2EIdentity() {
  const parallelIndex = test.info().parallelIndex;
  const suffix = `${parallelIndex + 1}`;

  return {
    email: `e2e+worker-${suffix}@multica.ai`,
    name: `E2E User ${suffix}`,
    workspaceSlug: `e2e-workspace-${suffix}`,
    workspaceName: `E2E Workspace ${suffix}`,
  };
}

export function getDefaultE2EWorkspaceName() {
  return currentE2EIdentity().workspaceName;
}

/**
 * Log in as the default E2E user and ensure the workspace exists first.
 * Authenticates via API (send-code → DB read → verify-code), then injects
 * the token into localStorage so the browser session is authenticated.
 */
export async function loginAsDefault(page: Page) {
  const identity = currentE2EIdentity();
  const api = new TestApiClient();
  await api.login(identity.email, identity.name);
  const workspace = await api.ensureWorkspace(identity.workspaceName, identity.workspaceSlug);

  const token = api.getToken();
  await page.addInitScript(({ t, workspaceId }) => {
    window.localStorage.setItem("multica_token", t);
    window.localStorage.setItem("multica_workspace_id", workspaceId);
  }, { t: token, workspaceId: workspace.id });
  await page.goto("/issues");
  await page.waitForURL("**/issues", { timeout: 10000 });
  await expect(page.getByRole("button", { name: /New Issue/ })).toBeVisible({ timeout: 10000 });
}

/**
 * Create a TestApiClient logged in as the default E2E user.
 * Call api.cleanup() in afterEach to remove test data created during the test.
 */
export async function createTestApi(): Promise<TestApiClient> {
  const identity = currentE2EIdentity();
  const api = new TestApiClient();
  await api.login(identity.email, identity.name);
  await api.ensureWorkspace(identity.workspaceName, identity.workspaceSlug);
  return api;
}

export function workspaceSwitcher(page: Page) {
  return page
    .locator("button")
    .filter({ has: page.locator("svg.lucide-chevron-down") })
    .first();
}

export function workspaceSwitcherByName(page: Page, name: string | RegExp) {
  return page.getByRole("button", { name }).first();
}

export function sidebarLink(page: Page, name: string) {
  const sidebar = page.locator('[data-slot="sidebar"]').first();
  const href = SIDEBAR_LINK_HREFS[name];

  if (href) {
    return sidebar.locator(`a[href="${href}"]`).first();
  }

  return sidebar.getByRole("link", { name, exact: true });
}

export async function openWorkspaceMenu(page: Page) {
  await workspaceSwitcher(page).click();
  await expect(page.getByRole("menuitem", { name: "Log out" })).toBeVisible();
}
