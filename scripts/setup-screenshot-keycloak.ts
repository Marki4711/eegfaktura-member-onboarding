/**
 * Configures the local screenshot-Keycloak (started via docker-compose.screenshots.yml).
 *
 * Run once after `docker compose ... up -d`. Creates:
 *   - Realm EEGFaktura
 *   - Public client eegfaktura-member-onboarding with redirect URI
 *     http://localhost:3000/api/auth/callback/keycloak
 *   - User screenshot-bot with attribute tenant=["RC123456"] and a freshly
 *     generated password
 *
 * Writes the credentials + URLs to .env.screenshots.local (gitignored), which
 * is consumed by generate-screenshots.ts for the auto-login.
 *
 * Idempotent: re-running rotates the password and updates the .env file in
 * place; existing realm/client/user are reused.
 */

import crypto from "crypto"
import fs from "fs"
import path from "path"

const KC_BASE = process.env.KC_BASE ?? "http://localhost:8180"
const ADMIN_USER = process.env.KC_ADMIN_USERNAME ?? "admin"
const ADMIN_PASS = process.env.KC_ADMIN_PASSWORD ?? "admin"

const REALM = "EEGFaktura"
const CLIENT_ID = "eegfaktura-member-onboarding"
const BOT_USERNAME = "screenshot-bot"
const REDIRECT_URI = "http://localhost:3000/api/auth/callback/keycloak"
const TENANT_RC = "RC123456"

const ENV_FILE = path.resolve(".env.screenshots.local")

function log(msg: string) {
  console.log(msg)
}

async function waitForKeycloak(timeoutMs = 90_000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    try {
      const r = await fetch(`${KC_BASE}/realms/master/.well-known/openid-configuration`)
      if (r.ok) return
    } catch {
      /* not ready */
    }
    await new Promise((r) => setTimeout(r, 1000))
  }
  throw new Error(`Keycloak at ${KC_BASE} did not become ready within ${timeoutMs}ms`)
}

async function getAdminToken(): Promise<string> {
  const r = await fetch(`${KC_BASE}/realms/master/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "password",
      client_id: "admin-cli",
      username: ADMIN_USER,
      password: ADMIN_PASS,
    }),
  })
  if (!r.ok) throw new Error(`admin token failed: ${r.status} ${await r.text()}`)
  const j = (await r.json()) as { access_token: string }
  return j.access_token
}

async function kcFetch(token: string, pathSuffix: string, init: RequestInit = {}) {
  const headers = {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
    ...((init.headers as Record<string, string>) ?? {}),
  }
  const r = await fetch(`${KC_BASE}/admin${pathSuffix}`, { ...init, headers })
  if (!r.ok && r.status !== 409) {
    const body = await r.text()
    throw new Error(`${init.method ?? "GET"} ${pathSuffix} → ${r.status}: ${body}`)
  }
  return r
}

async function ensureRealm(token: string) {
  const r = await fetch(`${KC_BASE}/admin/realms/${REALM}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (r.ok) {
    log(`  · realm ${REALM} exists`)
  } else {
    log(`  · creating realm ${REALM}`)
    await kcFetch(token, "/realms", {
      method: "POST",
      body: JSON.stringify({ realm: REALM, enabled: true }),
    })
  }

  // Keycloak 26 ships with a user-profile schema that disallows arbitrary
  // user attributes by default. We need `tenant` to be allowed before the
  // bot user can carry it. Easiest: switch the realm to ENABLED unmanaged
  // attributes (the pre-26 default).
  log("  · enabling unmanaged user attributes")
  const profileRes = await fetch(`${KC_BASE}/admin/realms/${REALM}/users/profile`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const profile = (await profileRes.json()) as Record<string, unknown>
  profile.unmanagedAttributePolicy = "ENABLED"
  await kcFetch(token, `/realms/${REALM}/users/profile`, {
    method: "PUT",
    body: JSON.stringify(profile),
  })
}

async function ensureClient(token: string): Promise<string> {
  const list = await fetch(
    `${KC_BASE}/admin/realms/${REALM}/clients?clientId=${encodeURIComponent(CLIENT_ID)}`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  const existing = (await list.json()) as Array<{ id: string }>
  if (existing.length > 0) {
    log(`  · client ${CLIENT_ID} exists`)
    return existing[0].id
  }
  log(`  · creating client ${CLIENT_ID}`)
  await kcFetch(token, `/realms/${REALM}/clients`, {
    method: "POST",
    body: JSON.stringify({
      clientId: CLIENT_ID,
      enabled: true,
      protocol: "openid-connect",
      publicClient: false,
      standardFlowEnabled: true,
      directAccessGrantsEnabled: true,
      redirectUris: [REDIRECT_URI],
      webOrigins: ["http://localhost:3000"],
      attributes: { "post.logout.redirect.uris": "http://localhost:3000/*" },
    }),
  })
  const list2 = await fetch(
    `${KC_BASE}/admin/realms/${REALM}/clients?clientId=${encodeURIComponent(CLIENT_ID)}`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  const arr = (await list2.json()) as Array<{ id: string }>
  return arr[0].id
}

async function getClientSecret(token: string, clientUuid: string): Promise<string> {
  const r = await fetch(`${KC_BASE}/admin/realms/${REALM}/clients/${clientUuid}/client-secret`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const j = (await r.json()) as { value: string }
  return j.value
}

async function ensureTenantMapper(token: string, clientUuid: string) {
  const r = await fetch(
    `${KC_BASE}/admin/realms/${REALM}/clients/${clientUuid}/protocol-mappers/models`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  const mappers = (await r.json()) as Array<{ name: string }>
  if (mappers.some((m) => m.name === "tenant")) {
    log("  · tenant mapper exists")
    return
  }
  log("  · creating tenant mapper")
  await kcFetch(token, `/realms/${REALM}/clients/${clientUuid}/protocol-mappers/models`, {
    method: "POST",
    body: JSON.stringify({
      name: "tenant",
      protocol: "openid-connect",
      protocolMapper: "oidc-usermodel-attribute-mapper",
      config: {
        "user.attribute": "tenant",
        "claim.name": "tenant",
        "jsonType.label": "String",
        multivalued: "true",
        "id.token.claim": "true",
        "access.token.claim": "true",
        "userinfo.token.claim": "true",
      },
    }),
  })
}

async function ensureBotUser(token: string, password: string): Promise<string> {
  const list = await fetch(
    `${KC_BASE}/admin/realms/${REALM}/users?username=${encodeURIComponent(BOT_USERNAME)}&exact=true`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  const existing = (await list.json()) as Array<{ id: string }>
  let userId: string
  if (existing.length > 0) {
    userId = existing[0].id
    log(`  · user ${BOT_USERNAME} exists (${userId})`)
    await kcFetch(token, `/realms/${REALM}/users/${userId}`, {
      method: "PUT",
      body: JSON.stringify({
        username: BOT_USERNAME,
        enabled: true,
        emailVerified: true,
        requiredActions: [],
        firstName: "Screenshot",
        lastName: "Bot",
        email: "screenshot-bot@example.local",
        attributes: { tenant: [TENANT_RC] },
      }),
    })
  } else {
    log(`  · creating user ${BOT_USERNAME}`)
    await kcFetch(token, `/realms/${REALM}/users`, {
      method: "POST",
      body: JSON.stringify({
        username: BOT_USERNAME,
        enabled: true,
        emailVerified: true,
        requiredActions: [],
        firstName: "Screenshot",
        lastName: "Bot",
        email: "screenshot-bot@example.local",
        attributes: { tenant: [TENANT_RC] },
      }),
    })
    const list2 = await fetch(
      `${KC_BASE}/admin/realms/${REALM}/users?username=${encodeURIComponent(BOT_USERNAME)}&exact=true`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    const arr = (await list2.json()) as Array<{ id: string }>
    userId = arr[0].id
  }

  log("  · rotating password")
  await kcFetch(token, `/realms/${REALM}/users/${userId}/reset-password`, {
    method: "PUT",
    body: JSON.stringify({ type: "password", value: password, temporary: false }),
  })
  return userId
}

function writeEnvFile(values: Record<string, string>) {
  const banner = [
    "# Auto-generated by scripts/setup-screenshot-keycloak.ts.",
    "# Consumed by generate-screenshots.ts for headless auto-login.",
    "# Re-running the setup overwrites this file with a fresh password.",
    "# DO NOT COMMIT — listed in .gitignore.",
    "",
  ].join("\n")
  const body = Object.entries(values)
    .map(([k, v]) => `${k}=${v}`)
    .join("\n")
  fs.writeFileSync(ENV_FILE, `${banner}${body}\n`)
}

async function main() {
  log(`Configuring screenshot Keycloak at ${KC_BASE}`)
  await waitForKeycloak()
  const token = await getAdminToken()

  await ensureRealm(token)
  const clientUuid = await ensureClient(token)
  await ensureTenantMapper(token, clientUuid)
  const clientSecret = await getClientSecret(token, clientUuid)

  const password = crypto.randomBytes(24).toString("base64url")
  await ensureBotUser(token, password)

  writeEnvFile({
    KC_BASE,
    KEYCLOAK_REALM: REALM,
    KEYCLOAK_ISSUER: `${KC_BASE}/realms/${REALM}`,
    KEYCLOAK_JWKS_URL: `${KC_BASE}/realms/${REALM}/protocol/openid-connect/certs`,
    KEYCLOAK_CLIENT_ID: CLIENT_ID,
    KEYCLOAK_CLIENT_SECRET: clientSecret,
    SCREENSHOT_BOT_USERNAME: BOT_USERNAME,
    SCREENSHOT_BOT_PASSWORD: password,
  })

  log(`\n✓ Wrote ${path.relative(process.cwd(), ENV_FILE)}`)
  log("  Next: start the backend+frontend pointed at this Keycloak, then run")
  log("        npm run screenshots")
}

main().catch((err) => {
  console.error("\n❌ Keycloak setup failed:", err.message ?? err)
  process.exit(1)
})
