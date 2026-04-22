# Keycloak Setup

Realm: **EEGFaktura**  
Client: **eegfaktura-member-onboarding**

## Client Configuration (Access Settings)

| Field | Value |
|------|------|
| Root URL | `https://member-onboarding-test.eegfaktura.at/` |
| Valid redirect URIs | `https://member-onboarding-test.eegfaktura.at/*` |
| Valid post logout redirect URIs | `https://member-onboarding-test.eegfaktura.at/*` |
| Web origins | `https://member-onboarding-test.eegfaktura.at` |

> **Important:** `Valid post logout redirect URIs` must contain the wildcard `/*`, otherwise logout fails with "Invalid Redirect URI" (HTTP 400).

## User Roles and Access Logic

### Superuser
- Has Realm Role `superuser`
- Can see all applications of all EEGs
- No `tenant` attribute required

### Tenant Admin
- Has **no** Realm Role
- Has user attribute `tenant` as a JSON array of RC numbers, e.g. `["RC101665", "RC101294"]`
- Can only see applications of their own EEGs
- The `tenant` attribute is mapped into the JWT Access Token via a Client Scope Mapper (User Attribute, Multivalued)

### No Access
- Users without the `superuser` role and without a `tenant` attribute → 403 / redirect to `/unauthorized`

## Client Scope Mapper for `tenant`

The `tenant` claim must be explicitly mapped into the Access Token.

**Clients** → `eegfaktura-member-onboarding` → Tab **Client scopes** → Open dedicated scope → **Configure a new mapper**:

| Field | Value |
|------|------|
| Mapper Type | User Attribute |
| Name | `tenant` |
| User Attribute | `tenant` |
| Token Claim Name | `tenant` |
| Claim JSON Type | `String` |
| Add to access token | **ON** |
| Multivalued | **OFF** |
| Aggregate attribute values | **OFF** |

> **Important: Multivalued must be OFF.** The attribute value is already a JSON array string (e.g. `["RC101665","RC101294"]`) — this is the format used by other applications and cannot be changed. With Multivalued ON, Keycloak would wrap the string into another array.

The frontend parses the JSON string in `auth.ts` automatically into a `string[]`.

> **Realm roles** (`realm_access`) are automatically included in the Access Token — no separate mapper needed.

## Nginx Proxy Buffer

NextAuth sets several large cookies during the OAuth callback (JWT with Access Token, ID Token, Refresh Token). nginx's default buffer size is not sufficient for this.

Configuration in the Helm Chart (`values.yaml`):
```yaml
ingress:
  proxyBufferSize: "128k"
  proxyBuffersNumber: "4"
```

Without this configuration: `upstream sent too big header` → HTTP 502 during login callback.
