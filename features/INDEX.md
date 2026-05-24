# Feature Index

Central tracking for all features.

## Status Legend

- **Planned** - Requirements written, ready for development
- **Architected** - Technical design complete, ready for implementation
- **In Progress** - Currently being built
- **In Review** - QA testing in progress
- **Approved** - QA passed, ready for deployment
- **Deployed** - Live in production

## Features

| ID | Feature | Status | Spec | Created |
|----|---------|--------|------|---------|
| PROJ-1 | Public Registration | Deployed | `features/PROJ-1-public-registration.md` | 2026-04-18 |
| PROJ-2 | Admin Review | Deployed | `features/PROJ-2-admin-review.md` | 2026-04-19 |
| PROJ-3 | Admin Frontend UI | Deployed | `features/PROJ-3-admin-frontend-ui.md` | 2026-04-19 |
| PROJ-4 | Core Import | Deployed | `features/PROJ-4-core-import.md` | 2026-04-19 |
| PROJ-5 | Keycloak-secured Admin Area | Deployed | `features/PROJ-5-keycloak-admin-auth.md` | 2026-04-19 |
| PROJ-6 | E-Mail-Benachrichtigungen | Deployed | `features/PROJ-6-email-notifications.md` | 2026-04-19 |
| PROJ-7 | Mitgliedstypen | Deployed | `features/PROJ-7-member-types.md` | 2026-04-20 |
| PROJ-8 | Konfigurierbare Felder pro EEG | Deployed | `features/PROJ-8-configurable-fields.md` | 2026-04-21 |
| PROJ-9 | EEG-spezifische Rechtsdokumente | Deployed | `features/PROJ-9-legal-documents.md` | 2026-04-21 |
| PROJ-10 | Admin Notifications | On Hold | `features/PROJ-10-admin-notifications.md` | 2026-04-22 |
| PROJ-11 | Konfigurierbarer Einleitungstext | Deployed | `features/PROJ-11-registration-intro-text.md` | 2026-04-23 |
| PROJ-12 | SEPA-Lastschriftmandat PDF | Deployed | `features/PROJ-12-sepa-mandate-pdf.md` | 2026-04-23 |
| PROJ-13 | Externe Registrierungs-API | Deployed | `features/PROJ-13-external-registration-api.md` | 2026-04-24 |
| PROJ-14 | SEPA-Firmenlastschriftmandat | Deployed | `features/PROJ-14-company-sepa-mandate.md` | 2026-04-24 |
| PROJ-15 | Konfigurierbare Felder Erweiterungen | Deployed | `features/PROJ-15-configurable-fields-extensions.md` | 2026-04-24 |
| PROJ-16 | Cloudflare Turnstile Spam-Schutz | Deployed | `features/PROJ-16-turnstile-spam-protection.md` | 2026-04-24 |
| PROJ-17 | Excel-Export für eegFaktura-Import | Deployed | `features/PROJ-17-excel-export.md` | 2026-04-25 |
| PROJ-18 | Datenschutzerklärung & Central Policy Toggle | Deployed | `features/PROJ-18-datenschutz-policy-toggle.md` | 2026-04-25 |
| PROJ-19 | Manuelle Aktivierung der Registrierung | Deployed | `features/PROJ-19-registration-activation.md` | 2026-04-25 |
| PROJ-20 | Vollständige Antragsdaten in EEG-Einreichungsbenachrichtigung | Deployed | `features/PROJ-20-submission-notification-extended.md` | 2026-04-26 |
| PROJ-21 | Genehmigungs-Benachrichtigung mit Beitrittsbestätigung PDF | Deployed | `features/PROJ-21-approval-notification-pdf.md` | 2026-04-26 |
| PROJ-22 | Tailwind CSS v3 → v4 Upgrade | On Hold | `features/PROJ-22-tailwindcss-v4-upgrade.md` | 2026-04-26 |
| PROJ-23 | Stammdaten-Import aus eegFaktura-Excel | On Hold | `features/PROJ-23-stammdaten-import.md` | 2026-04-26 |
| PROJ-24 | OpenAPI/Swagger Dokumentation | Deployed | `features/PROJ-24-openapi-documentation.md` | 2026-04-29 |
| PROJ-25 | Bulk-Aktionen im Admin | Deployed | `features/PROJ-25-bulk-actions.md` | 2026-04-29 |
| PROJ-26 | Eigener Mailserver pro EEG | On Hold | `features/PROJ-26-per-eeg-smtp-override.md` | 2026-05-08 |
| PROJ-27 | Tarif-Auswahl beim Import | Deployed | `features/PROJ-27-tariff-selection-on-import.md` | 2026-05-09 |
| PROJ-28 | Trennung Privat / Kleinunternehmer | Deployed | `features/PROJ-28-split-private-and-kleinunternehmer.md` | 2026-05-12 |
| PROJ-29 | IBAN-Eingabe mit visueller Gruppierung | Deployed | `features/PROJ-29-iban-visual-grouping.md` | 2026-05-12 |
| PROJ-30 | Reset eines importierten Antrags auf approved | Deployed | `features/PROJ-30-reset-imported-to-approved.md` | 2026-05-12 |
| PROJ-31 | E-Mail-Adresse-Bestätigung (Anti-Abuse) | Deployed | `features/PROJ-31-email-confirmation.md` | 2026-05-14 |
| PROJ-32 | EEG-Stammdaten aus Core (Phase 1 – ohne Logo) | Deployed | `features/PROJ-32-eeg-master-data-from-core.md` | 2026-05-14 |
| PROJ-33 | EEG-Logo aus Core (Phase 2 von PROJ-32) | Deployed | `features/PROJ-33-eeg-logo-from-core.md` | 2026-05-14 |
| PROJ-34 | Robuste Import-Recovery (Orphan-Fallback + Pre-Check + Unstuck-GUI) | Deployed | `features/PROJ-34-import-recovery.md` | 2026-05-14 |
| PROJ-35 | Per-EEG-Referenznummern (`<RC>-<Jahr>-<NNNN>`) | Deployed | `features/PROJ-35-per-eeg-reference-numbers.md` | 2026-05-14 |
| PROJ-36 | Optionale Rechtsdokumente als Info-Dokumente | Deployed | `features/PROJ-36-optional-legal-documents-as-info.md` | 2026-05-14 |
| PROJ-37 | Genossenschaftsanteile (per-EEG-Konfiguration + Antragsfeld) | Deployed | `features/PROJ-37-cooperative-shares.md` | 2026-05-15 |
| PROJ-38 | Status-Modell-Hygiene & Audit-Fixes | Deployed | `features/PROJ-38-status-hygiene.md` | 2026-05-16 |
| PROJ-39 | Titel-Nach + Bankname im Public-Form + abweichende Adresse je Zählpunkt | Deployed | `features/PROJ-39-extra-fields.md` | 2026-05-17 |
| PROJ-41 | Status-Change-Mails an Mitglied (Ablehnung + Info-Anfrage) | Deployed | `features/PROJ-41-status-change-mails.md` | 2026-05-17 |
| PROJ-40 | EEG-Umzuordnung eines Antrags im Admin-Review | Deployed | `features/PROJ-40-eeg-reassign.md` | 2026-05-17 |
| PROJ-42 | E-Fahrzeug-Detailerfassung (Anzahl + Jahres-km) | Deployed | `features/PROJ-42-ev-details.md` | 2026-05-17 |
| PROJ-44 | Netzbetreiber-Vollmacht (per-EEG konfigurierbar) | Deployed | `features/PROJ-44-network-operator-authorization.md` | 2026-05-17 |
| PROJ-45 | Erzeugungsform + Batterie-Felder + typabhängige Sichtbarkeit | Deployed | `features/PROJ-45-generation-type-and-conditional-fields.md` | 2026-05-17 |
| PROJ-46 | Stati für Import-Nachbereitung (B2B-Bank-Bestätigung + Aktivierung) | Deployed | `features/PROJ-46-post-import-statuses.md` | 2026-05-17 |
| PROJ-47 | B2B-SEPA-Firmenlastschrift-Mandat mit Mandatsreferenz beim Import | Deployed | `features/PROJ-47-b2b-sepa-mandate-at-import.md` | 2026-05-17 |
| PROJ-48 | SEPA-Default-Core + konfigurierbares Mandat-Timing + B2B-Hinweis | Deployed | `features/PROJ-48-sepa-default-core-and-import-timing.md` | 2026-05-17 |
| PROJ-49 | Energie-Felder pro Zählpunkt (Refactoring + Einspeiselimit) | Deployed | `features/PROJ-49-energy-fields-on-metering-point.md` | 2026-05-17 |
| PROJ-50 | Zugang Online-Portal Netzbetreiber + bedingte Anleitungs-Mail | On Hold | `features/PROJ-50-network-operator-portal-access.md` | 2026-05-17 |
| PROJ-51 | Anzeige offener Nutzungsgebühren im Admin-UI | On Hold | `features/PROJ-51-usage-fee-status-display.md` | 2026-05-17 |
| PROJ-52 | Konfigurierbarer Zählpunkt-Prefix pro Richtung + Auto-Pad + Alphanumerik | Deployed | `features/PROJ-52-metering-point-prefix-per-direction.md` | 2026-05-17 |
| PROJ-53 | Aktivierungs-Modus pro EEG + Beitrittsbestätigung erst bei `activated` | Deployed | `features/PROJ-53-activation-mode-and-deferred-approval-mail.md` | 2026-05-19 |
| PROJ-54 | Aufteilung in öffentliches Schaufenster + privates Hauptrepo | Deployed | `features/PROJ-54-public-private-repo-split.md` | 2026-05-20 |
| PROJ-55 | Nachmelden von Zählpunkten anhand der Mitgliedsnummer | Planned | `features/PROJ-55-add-metering-points-by-member-number.md` | 2026-05-21 |
| PROJ-56 | Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF | Deployed | `features/PROJ-56-network-operator-info-pdf.md` | 2026-05-21 |
| PROJ-57 | Ansprechperson für Org-Mitgliedstypen | Deployed | `features/PROJ-57-contact-person.md` | 2026-05-21 |
| PROJ-58 | Abweichende Rechnungs-E-Mail für Org-Mitgliedstypen | Deployed | `features/PROJ-58-billing-email.md` | 2026-05-21 |
| PROJ-59 | BgA / Hoheitsbereich-Vermerk im Anlagennamen bei Gemeinden | Deployed | `features/PROJ-59-municipal-business-type-per-meter.md` | 2026-05-23 |
| PROJ-60 | Datenweiterleitung an externe Systeme — Async-Plugin-Framework mit Job-Queue + In-App-Worker; Excel/CSV-Plugin als erste Implementierung; Bulk-Action aus Antragsliste oder Single aus Detail; Phase 2 = weitere Plugins (Zoho/HubSpot/…) | Deployed | `features/PROJ-60-external-system-sync.md` | 2026-05-23 |
| PROJ-61 | Konfigurations-Export & -Import pro EEG (4 Sub-Typen: EEG-Settings, Field-Config, Legal-Documents, Data-Export-Configs; JSON-Datei + Diff-Preview; Tenant-Admin) | Deployed | `features/PROJ-61-config-export-import.md` | 2026-05-24 |
| PROJ-62 | Mitgliedstypen Kleinunternehmer + Unternehmen zusammenführen (sole_proprietor entfällt, company-Typ mit optionaler UID = Kleinunternehmerregelung) | Approved | `features/PROJ-62-merge-sole-proprietor-into-company.md` | 2026-05-24 |
| PROJ-63 | USt-Pflicht-Checkbox bei Unternehmen + Verein (UI-Gate für UID-Eingabe, kein DB-Feld) | In Progress | `features/PROJ-63-vat-liability-checkbox.md` | 2026-05-24 |

## Next Available ID: PROJ-64
