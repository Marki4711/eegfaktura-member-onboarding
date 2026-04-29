# PROJ-24: OpenAPI/Swagger Dokumentation

## Status: Planned
**Created:** 2026-04-29
**Last Updated:** 2026-04-29

## Dependencies
- PROJ-1 (Public Registration API)
- PROJ-2 (Admin Review API)
- PROJ-13 (External Registration API)

## User Stories
- Als externer Entwickler möchte ich die External API über eine interaktive Dokumentation erkunden, damit ich die Integration ohne Rückfragen implementieren kann.
- Als EEG-Admin möchte ich die verfügbaren API-Endpunkte und ihre Parameter nachschlagen können, damit ich die Schnittstelle korrekt verwende.
- Als Entwickler möchte ich Requests direkt aus der Swagger UI absetzen können, damit ich die API ohne extra Tools testen kann.
- Als Maintainer möchte ich, dass die Dokumentation automatisch aus dem Code generiert wird, damit sie nicht manuell gepflegt werden muss.

## Acceptance Criteria
- [ ] Swagger UI ist unter `/api/docs` erreichbar (öffentlich, kein Auth erforderlich)
- [ ] Alle Public-Endpunkte (`/api/public/`) sind vollständig dokumentiert (Parameter, Request Body, Responses)
- [ ] Alle External-API-Endpunkte (`/api/external/`) sind dokumentiert inkl. API-Key-Auth-Hinweis
- [ ] Alle Admin-Endpunkte (`/api/admin/`) sind dokumentiert mit Hinweis auf Keycloak-Bearer-Auth
- [ ] Jeder Endpunkt hat eine kurze Beschreibung, alle Parameter sind benannt und typisiert
- [ ] Response-Schemas sind für alle HTTP-Status-Codes (200, 400, 401, 403, 404, 422, 500) dokumentiert
- [ ] Die Spec wird beim Build automatisch aus Go-Annotations generiert (`swag generate`)
- [ ] `swag generate` ist in den Build-Prozess / Makefile integriert
- [ ] Die generierte `docs/swagger.json` ist im Repository committed und bleibt in Sync mit dem Code

## Edge Cases
- Admin-Endpunkte sind in der Spec dokumentiert, aber die Swagger UI sendet keine echten Auth-Tokens — Nutzer müssen den Bearer-Token manuell eintragen
- `swag generate` muss vor `go build` laufen; CI schlägt fehl wenn Spec veraltet ist
- Turnstile-geschützte Endpunkte können nicht direkt aus der Swagger UI getestet werden — das ist im Endpunkt zu vermerken

## Technical Requirements
- Bibliothek: `swaggo/swag` (Code-Annotation-basiert) + `swaggo/http-swagger` für die UI
- Swagger UI Route: `GET /api/docs/*` → öffentlich, kein Auth-Middleware
- Generiertes Artefakt: `docs/swagger.json` (committed), `docs/docs.go` (committed)
- Go-Build: `swag init` muss vor `go build` ausgeführt werden
- Security: Swagger UI gibt keine echten Credentials preis; Spec enthält keine Beispiel-Tokens

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
