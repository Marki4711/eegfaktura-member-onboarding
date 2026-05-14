# Third-Party Notices

This file lists third-party open-source components used by **eegfaktura Member Onboarding** and the licenses they are distributed under. It is intended to satisfy the attribution requirements of the listed licenses.

The full text of each referenced license can be found in the respective package's source repository.

Last reviewed: 2026-05-14.

---

## Go backend (direct dependencies)

| Module | License | Project |
|---|---|---|
| github.com/MicahParks/keyfunc/v3 | Apache-2.0 | https://github.com/MicahParks/keyfunc |
| github.com/go-chi/chi/v5 | MIT | https://github.com/go-chi/chi |
| github.com/go-pdf/fpdf | MIT | https://github.com/go-pdf/fpdf |
| github.com/go-playground/validator/v10 | MIT | https://github.com/go-playground/validator |
| github.com/golang-jwt/jwt/v5 | MIT | https://github.com/golang-jwt/jwt |
| github.com/golang-migrate/migrate/v4 | MIT | https://github.com/golang-migrate/migrate |
| github.com/google/uuid | BSD-3-Clause | https://github.com/google/uuid |
| github.com/lib/pq | MIT-style (permissive) | https://github.com/lib/pq |
| github.com/microcosm-cc/bluemonday | BSD-3-Clause | https://github.com/microcosm-cc/bluemonday |
| github.com/swaggo/http-swagger/v2 | MIT | https://github.com/swaggo/http-swagger |
| github.com/swaggo/swag | MIT | https://github.com/swaggo/swag |
| github.com/wneessen/go-mail | MIT | https://github.com/wneessen/go-mail |
| github.com/xuri/excelize/v2 | BSD-2-Clause | https://github.com/qax-os/excelize |
| golang.org/x/text | BSD-3-Clause | https://pkg.go.dev/golang.org/x/text |

## Go backend (notable indirect / runtime dependencies)

| Module | License | Notes |
|---|---|---|
| github.com/prometheus/client_golang | Apache-2.0 | Metrics exposition |
| github.com/gorilla/css | BSD-3-Clause | Pulled by bluemonday |
| github.com/gabriel-vasile/mimetype | MIT | Pulled by go-mail |
| go.yaml.in/yaml/v2, gopkg.in/yaml.v2 | Apache-2.0 + MIT | Pulled by swag |

No GPL-, AGPL-, or SSPL-licensed Go modules are part of the dependency tree.

---

## Node / Next.js frontend (direct dependencies)

| Package | License | Project |
|---|---|---|
| next | MIT | https://github.com/vercel/next.js |
| next-auth | ISC | https://github.com/nextauthjs/next-auth |
| react, react-dom | MIT | https://github.com/facebook/react |
| @hookform/resolvers, react-hook-form | MIT | https://github.com/react-hook-form/react-hook-form |
| @marsidev/react-turnstile | MIT | https://github.com/marsidev/react-turnstile |
| @radix-ui/* (Radix UI primitives) | MIT | https://github.com/radix-ui/primitives |
| @tiptap/react, @tiptap/starter-kit, @tiptap/extension-link | MIT | https://github.com/ueberdosis/tiptap |
| @types/dompurify | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| class-variance-authority | Apache-2.0 | https://github.com/joe-bell/cva |
| clsx | MIT | https://github.com/lukeed/clsx |
| cmdk | MIT | https://github.com/pacocoursey/cmdk |
| dompurify | Apache-2.0 OR MPL-2.0 (chooseable) | https://github.com/cure53/DOMPurify |
| ibantools | MIT | https://github.com/Simplify/ibantools |
| lucide-react | ISC | https://github.com/lucide-icons/lucide |
| next-themes | MIT | https://github.com/pacocoursey/next-themes |
| react-imask | MIT | https://github.com/uNmAnNeR/imaskjs |
| sonner | MIT | https://github.com/emilkowalski/sonner |
| tailwind-merge | MIT | https://github.com/dcastil/tailwind-merge |
| zod | MIT | https://github.com/colinhacks/zod |

## Node / Next.js frontend (notable transitive dependencies)

| Package | License | Notes |
|---|---|---|
| **sharp** (libvips bindings) | Apache-2.0 (sharp itself); **LGPL-3.0-or-later** for the bundled libvips native binaries | Pulled by Next.js for built-in image optimisation. We do not modify libvips; we ship the unmodified prebuilt native addon as published by the `sharp` project. The source for libvips is available at https://github.com/libvips/libvips and the source for sharp at https://github.com/lovell/sharp. |
| undici | MIT | HTTP client used by Next.js |
| tailwindcss | MIT | Build-time only |

### LGPL §6 source-offer (sharp / libvips)

The frontend container image distributes the `sharp` Node addon, which links dynamically against `libvips` shared libraries. `libvips` is licensed under LGPL-3.0-or-later.

In accordance with section 6 of the LGPL, the corresponding source code for the version of `libvips` shipped with this Software is available from the upstream project at:

* https://github.com/libvips/libvips

We do not modify the upstream library. Customers who would like the source for the exact version bundled with their build of the Software may request it from the copyright holder of the Software.

---

## Build-time / development dependencies

These are not shipped with the production deployment and are listed only for completeness:

| Package | License | Use |
|---|---|---|
| @playwright/test | Apache-2.0 | E2E tests |
| @testing-library/* | MIT | Unit tests |
| @vitejs/plugin-react, vitest | MIT | Unit tests |
| eslint, eslint-config-next | MIT | Linting |
| postcss, autoprefixer | MIT | Build |
| typescript | Apache-2.0 | Build |
| lightningcss | MPL-2.0 | Build tool, never shipped |
| caniuse-lite | CC-BY-4.0 | Browser-target data (build-time) |
| axe-core | MPL-2.0 | Accessibility checks (dev only) |

MPL-2.0 build-time tools are not redistributed and therefore do not trigger any source-disclosure obligation.

---

## UI components copied from external sources

The components under `src/components/ui/` are based on [shadcn/ui](https://github.com/shadcn-ui/ui) (MIT). Shadcn explicitly publishes its components for copy-in use; no attribution is legally required, but it is included here for transparency.

---

## Embedded assets

* `src/app/icon.svg` — original work, no third-party content.
* PDF/email templates under `internal/mail/templates/` and `internal/pdf/` — original work.
* The PostScript core-14 fonts (Helvetica, Times, Courier) used by `go-pdf/fpdf` are font references shipped as part of the PDF format itself and require no separate license.

No third-party images, fonts, screenshots, or sample data are bundled in the repository.

---

## Reporting issues

If you believe this notice is incomplete or incorrect, please contact the copyright holder.
