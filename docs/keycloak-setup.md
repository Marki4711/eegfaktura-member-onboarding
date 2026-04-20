# Keycloak Setup

Realm: **EEGFaktura**  
Client: **eegfaktura-member-onboarding**

## Client-Konfiguration (Access Settings)

| Feld | Wert |
|------|------|
| Root URL | `https://member-onboarding-test.eegfaktura.at/` |
| Valid redirect URIs | `https://member-onboarding-test.eegfaktura.at/*` |
| Valid post logout redirect URIs | `https://member-onboarding-test.eegfaktura.at/*` |
| Web origins | `https://member-onboarding-test.eegfaktura.at` |

> **Wichtig:** `Valid post logout redirect URIs` muss den Wildcard `/*` enthalten, sonst schlägt der Logout mit „Ungültige Redirect URI" (HTTP 400) fehl.

## Benutzerrollen und Zugriffslogik

### Superuser
- Hat Realm Role `superuser`
- Sieht alle Anträge aller EEGs
- Kein `tenant`-Attribut erforderlich

### Tenant-Admin
- Hat **keine** Realm Role
- Hat User-Attribut `tenant` als JSON-Array von RC-Nummern, z.B. `["RC101665", "RC101294"]`
- Sieht nur Anträge seiner EEGs
- Das `tenant`-Attribut wird via Client Scope Mapper (User Attribute, Multivalued) in den JWT Access Token gemappt

### Kein Zugriff
- Benutzer ohne `superuser`-Rolle und ohne `tenant`-Attribut → 403 / Weiterleitung auf `/unauthorized`

## Client Scope Mapper für `tenant`

Der `tenant`-Claim muss explizit in den Access Token gemappt werden.

**Clients** → `eegfaktura-member-onboarding` → Tab **Client scopes** → Dedicated scope öffnen → **Configure a new mapper**:

| Feld | Wert |
|------|------|
| Mapper Type | User Attribute |
| Name | `tenant` |
| User Attribute | `tenant` |
| Token Claim Name | `tenant` |
| Claim JSON Type | `String` |
| Add to access token | **ON** |
| Multivalued | **OFF** |
| Aggregate attribute values | **OFF** |

> **Wichtig: Multivalued muss OFF sein.** Der Attributwert ist bereits ein JSON-Array-String (z.B. `["RC101665","RC101294"]`) — das ist das Format das andere Applikationen verwenden und nicht geändert werden kann. Mit Multivalued ON würde Keycloak den String nochmals in ein Array einwickeln.

Das Frontend parst den JSON-String in `auth.ts` automatisch zu einem `string[]`.

> **Realm roles** (`realm_access`) sind automatisch im Access Token — kein eigener Mapper nötig.

## Nginx Proxy Buffer

NextAuth setzt beim OAuth-Callback mehrere große Cookies (JWT mit Access-Token, ID-Token, Refresh-Token). nginx's Standard-Puffergröße reicht dafür nicht aus.

Konfiguration im Helm Chart (`values.yaml`):
```yaml
ingress:
  proxyBufferSize: "128k"
  proxyBuffersNumber: "4"
```

Ohne diese Einstellung: `upstream sent too big header` → HTTP 502 beim Login-Callback.
