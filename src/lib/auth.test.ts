import { describe, it, expect } from "vitest";
import { isSuperuser, isTenantAdmin, hasAdminAccess } from "./auth";

describe("isSuperuser", () => {
  it("returns true when roles includes superuser", () => {
    expect(isSuperuser(["superuser"])).toBe(true);
    expect(isSuperuser(["user", "superuser", "admin"])).toBe(true);
  });

  it("returns false when roles does not include superuser", () => {
    expect(isSuperuser([])).toBe(false);
    expect(isSuperuser(["admin", "user"])).toBe(false);
  });
});

describe("isTenantAdmin", () => {
  it("returns true when tenant array is non-empty", () => {
    expect(isTenantAdmin(["RC101665"])).toBe(true);
    expect(isTenantAdmin(["RC101665", "RC101294"])).toBe(true);
  });

  it("returns false when tenant array is empty", () => {
    expect(isTenantAdmin([])).toBe(false);
  });
});

describe("hasAdminAccess", () => {
  it("grants access to superuser", () => {
    expect(hasAdminAccess(["superuser"], [])).toBe(true);
    expect(hasAdminAccess(["superuser"], ["RC101665"])).toBe(true);
  });

  it("grants access to tenant-admin", () => {
    expect(hasAdminAccess([], ["RC101665"])).toBe(true);
    expect(hasAdminAccess(["user"], ["RC101665", "RC101294"])).toBe(true);
  });

  it("denies access when neither superuser nor tenant-admin", () => {
    expect(hasAdminAccess([], [])).toBe(false);
    expect(hasAdminAccess(["user", "admin"], [])).toBe(false);
  });

  it("superuser role wins even if tenant is empty", () => {
    expect(hasAdminAccess(["superuser"], [])).toBe(true);
  });

  it("superuser role wins even if tenant is also present", () => {
    // superuser with tenant should still have access (and see ALL EEGs, not filtered)
    expect(hasAdminAccess(["superuser", "user"], ["RC101665"])).toBe(true);
  });
});
