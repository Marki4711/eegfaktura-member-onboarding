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
| PROJ-62 | Mitgliedstypen Kleinunternehmer + Unternehmen zusammenführen (sole_proprietor entfällt, company-Typ mit optionaler UID = Kleinunternehmerregelung) | Deployed | `features/PROJ-62-merge-sole-proprietor-into-company.md` | 2026-05-24 |
| PROJ-63 | USt-Pflicht-Checkbox bei Unternehmen + Verein (UI-Gate für UID-Eingabe, kein DB-Feld) | Deployed | `features/PROJ-63-vat-liability-checkbox.md` | 2026-05-24 |
| PROJ-64 | Faktura-Handover-Billing-Trigger (Excel-Bypass-Schließung — `application.faktura_handover_at` deckt /import UND /export/excel) | Deployed | `features/PROJ-64-faktura-handover-billing-trigger.md` | 2026-05-29 |
| PROJ-65 | Vorstands-Signaturblock im Beitrittsbestätigungs-PDF — Superseded durch PROJ-76 (größere Workflow-Lösung mit Mail-Routing-Wechsel) | Superseded | `features/PROJ-65-vorstands-signature-on-approval-pdf.md` | 2026-06-07 |
| PROJ-66 | Auto-Save für Settings-Editoren (Formular-Felder) + Tab-Switch-Schutz (alle drei Save-Button-Editoren) + Browser-Unload-Warnung | Deployed | `features/PROJ-66-settings-auto-save-and-tab-switch-guard.md` | 2026-05-30 |
| PROJ-67 | Standard-/Erweitert-Modus für Einstellungen — reduzierte Sicht für kleine EEGs (Toggle am Seitenkopf, Default Standard für neue EEGs, advanced für bestehende, mit Doku-Spiegelung) | Deployed | `features/PROJ-67-settings-standard-advanced-mode.md` | 2026-05-30 |
| PROJ-68 | admin_value-Default-Mechanismus entfernen (UI-Input, DB-Spalte, applyAdminValues-Funktion, Tests, Doku); admin_only-State bleibt als reine Public-Form-Hide-Markierung | Deployed | `features/PROJ-68-remove-admin-value-default-mechanism.md` | 2026-05-30 |
| PROJ-69 | Reconciliation-basierter Billing-Backstop — Login-Trigger gegen eegFaktura-Core, Strict-2-Keys-Match (IBAN+E-Mail), setzt faktura_handover_at + MNr-Backfill, global per Config-Flag, Cutoff ab Deploy | Deployed | `features/PROJ-69-billing-reconciliation-backstop.md` | 2026-05-31 |
| PROJ-70 | Stammdaten-Resync für aktivierte Anträge — zwei unabhängige Knöpfe im Antrags-Detail: „Stammdaten aus eegFaktura abgleichen" (Pull + Diff + Update, kein Status-Wechsel) und „SEPA-Mandat erneut senden" (PDF + Mail, hard-fail). Final-vereinfachte Variante nach 3 Iterationen. | Deployed | `features/PROJ-70-activated-stammdaten-resync.md` | 2026-06-01 |
| PROJ-71 | EEG-Customer-Onboarding (Architektur-Rewrite 2026-06-06) — Settings-Card-Submit aus `/admin/settings/` durch eingeloggten EEG-Admin, AGB/AVV-Click-Akzept mit Audit-Trail, AVV-PDF an Welcome-Mail bei Owner-Approve, Owner-Notification beim Submit, Superuser-BackOffice-Liste + Approve/Reject-Actions, Tenant-Status-Card mit Inline-Buchungsformular bei state=none. PROJ-74-Scope absorbiert. | Deployed | `features/PROJ-71-customer-onboarding-form-avv-mail.md` | 2026-06-06 |
| PROJ-72 | Member-Onboarding-Cockpit — Superuser-Übersicht aller EEGs unter `/admin/cockpit` mit Aktiv-Badge, Customer-Onboarding-State, Anträge-Pipeline (offen/erledigt) und Direkt-Links zu Anträgen & Einstellungen. Default-Sort nach Aktivität, alternativ nach Pipeline-Größe oder RC alphabetisch. Volltextsuche RC/Name. Live-Aggregation pro Aufruf. | Planned | `features/PROJ-72-member-onboarding-cockpit.md` | 2026-06-06 |
| PROJ-73 | Cleanup: verwaisten EEG-Toggle `use_company_sepa_mandate` entfernt — Domain-Logik seit PROJ-48 funktionslos; Toggle verwirrt Admins, die ihn umlegen und sehen, dass nichts passiert. Migration 000066 + Settings-UI-Aufräumung + Doku. | Deployed | `features/PROJ-73-cleanup-use-company-sepa-mandate.md` | 2026-06-06 |
| PROJ-74 | B2B-Mandat-Gate-Fix: `buildSEPAMandateData` durchlässt B2B-Anträge auch bei `SEPAMandateEnabled=false` (SEPA-Rulebook erlaubt keine Online-Zustimmung für Firmenlastschrift). Hart-Fail beim Import wenn Stammdaten fehlen. UI-Klarstellung an beiden SEPA-Toggles via Hint-Popover. PROJ-80 hat `SEPAMandateEnabled` später ganz entfernt — der Gate-Fix selbst (Hart-Fail bei fehlenden B2B-Stammdaten) bleibt aktiv. | Deployed | `features/PROJ-74-b2b-mandate-gate-fix.md` | 2026-06-06 |
| PROJ-75 | SEPA-Einwilligungs-Checkbox in Bankverbindungs-Card verschoben — direkt unter den Konto-Eingabefeldern statt im allgemeinen Einwilligungsblock; neuer Text mit EEG-Name + Creditor-ID aus PROJ-32-Sync. | Deployed | `features/PROJ-75-sepa-consent-checkbox-relocation.md` | 2026-06-06 |
| PROJ-76 | Vorstands-Genehmigungs-Workflow für Beitrittserklärung — per-EEG-Toggle, eigenes PDF „Beitrittserklärung" mit Vorstands-Signaturblock, Mail-Routing-Wechsel auf EEG-Kontakt statt Mitglied, On-Demand-Download im Admin-UI. Supersedes PROJ-65. | Deployed | `features/PROJ-76-board-approval-workflow.md` | 2026-06-07 |
| PROJ-77 | B2B-Mandat-Audit-Block — im Firmenlastschrift-PDF (`einzugsart=b2b`) ersetzt ein Audit-Trail-Text (§ 76 (3) EIWOG) den Datum/Unterschrift-Block. Neue Spalte `application.sepa_mandate_accepted_ip INET`. Bestandsanträge ohne IP fallen auf klassischen Unterschriftsblock zurück. Optionaler `submitterIp`-Body-Param für die externe API. PROJ-78 wrappt das hinter Per-EEG-Toggle (Default FALSE bis Rechtsklärung gem. § 76 (3) EIWOG) — Audit-Block-Code shipped, aber bis Opt-in stillgelegt. | Deployed | `features/PROJ-77-b2b-mandate-audit-block.md` | 2026-06-07 |
| PROJ-78 | Toggle „Elektronisches SEPA-Mandat" (B2B + CORE separat) — zwei unabhängige Per-EEG-Schalter (CORE und B2B) entscheiden, ob das jeweilige SEPA-Mandat-PDF den Audit-Trail-Block oder den klassischen Datum/Unterschrift-Block rendert. Audit-Variante für CORE neu; B2B-Audit (PROJ-77) wird hinter Toggle gestellt. Default beide FALSE (klassisch), bis Rechtsklärung zur formfreien Willenserklärung gem. § 76 (3) EIWOG durch ist. Render-Logik in shared Helper. | Deployed | `features/PROJ-78-electronic-sepa-mandate-toggle.md` | 2026-06-07 |
| PROJ-79 | B2B-Import als CORE in eegFaktura-Core — bei `einzugsart=b2b` wird der Antrag im Faktura-Core trotzdem mit SEPA-Typ CORE angelegt, um die Bank-Klärungs-Phase der B2B-Mandatsvereinbarung ohne Risiko fehlgeschlagener Erst-Lastschriften zu überbrücken. Aktivierungs-Mail an EEG-Kontakt (Auto + Vorstands-Modus) enthält gelben Hinweis-Block mit Aufforderung zur eigenständigen Bank-Klärung und manuellen Umstellung auf B2B im Core nach Bestätigung. Mapping-Änderung in `mapEinzugsart`, Shared Banner-Helper (single source statt PROJ-81-Doppelverdrahtung), hartkodierte globale Regel, kein Per-EEG-Toggle, keine DB-Schema-Änderung, Bestandsanträge unangetastet. Deployed v1.23.0-PROJ-79, wartet auf helm upgrade. | Deployed | `features/PROJ-79-b2b-import-as-core.md` | 2026-06-08 |
| PROJ-82 | Settings-Formular-Editor — UI-Staleness-Fix bei Tab-Wechsel — Auto-Save persistierte korrekt, aber der Parent-Cache (`fieldConfig` in `settings/page.tsx`) wurde nicht aktualisiert; bei Tab-Wechsel zurueck zum Formular-Tab kam der alte Stand. Fix via `onSaved`-Callback Editor → Parent (Variante B aus der Analyse), Parent-State synchron mit DB. Pure Frontend-Aenderung, keine API/DB/Helm-Aenderung. | Deployed | `features/PROJ-82-fieldconfig-editor-staleness-fix.md` | 2026-06-08 |
| PROJ-83 | Letzte EEG-Auswahl im Admin-Settings persistieren (localStorage). Owner-Schmerz: bei jedem Aufruf landet die Auswahl auf der ersten EEG; bei 10+ EEGs ist das jedes Mal ein Mehrklick. Helper `readLastUsedRc`/`writeLastUsedRc` mit Tenant-Scope-Validation (verlorene Berechtigungen verwerfen den Stale-Wert still). Storage-Inhalt nur RC-String, kein Token/PII (per Vitest-Sicherheitstest verifiziert). Kein Backend, keine DB, keine Helm-Aenderung. | Deployed | `features/PROJ-83-last-used-eeg-persistence.md` | 2026-06-08 |
| PROJ-84 | EEG-Stammdaten-Editor auf Auto-Save mit Client-seitiger Cross-Field-Validierung. Drei Backend-Regeln (PROJ-37 Genossenschaftsanteile, PROJ-80 SEPA-CORE-Audit-Coupling, PROJ-81 SEPA-Wahl-Whitelist) werden im Frontend gespiegelt; bei rotem Gate wird `autoSave.schedule()` uebersprungen und ein gelber Hint-Banner zeigt fehlende Pflichtfelder. Owner-approbierter Lead-In-Wortlaut + Bullet-Vorschlaege pro Buendel in der Spec. Drift-Schutz via Vitest-Permutationstest. Defense-in-Depth: Backend-Validation bleibt unveraendert. Spec-Reviewed nach /grill-me (4 Owner-Entscheidungen): Toggle-OFF blendet Sub-Felder aus (kein Disabled-Rendern), kein Feature-Flag, Last-Write-Wins (kein Optimistic-Lock). Tech-Design vollstaendig (9-Schritte-Implementierungs-Reihenfolge), Frontend-only, keine Backend/DB/Migration/Helm-Aenderung. QA APPROVED 21 ACs Pass + 2 N/A, 13/13 ECs OK, 0 Findings, 0 Regressionen. | Approved | `features/PROJ-84-eeg-settings-auto-save.md` | 2026-06-08 |
| PROJ-85 | ESLint v9 Flat-Config-Migration. `npm run lint` schlaegt fehl (next-lint-Wrapper-Bug + Legacy-Config inkompatibel mit ESLint v10). Reine Dev-Tooling-Migration auf `eslint.config.js`, optional Custom-Regel fuer `placeholder=`-Verbot (Memory-Regel `feedback_no_placeholders` als Lint-Pattern), CI-Schritt ergaenzen. Kein Production-Blocker — `tsc --noEmit` + `npm run build` sind die echten Korrektheits-Gates und laufen clean. Niedrige Prio. | Planned | `features/PROJ-85-eslint-v9-flat-config-migration.md` | 2026-06-08 |
| PROJ-80 | SEPA-Settings-Vereinfachung — Toggle `sepaMandateEnabled` raus, Online-Zustimmung-Checkbox + CORE-PDF-Generierung werden Pflicht-Konstanten, nur noch CORE-Audit-Toggle + Timing-Toggle als Konfig. Timing-Label neu „SEPA-Mandat erst beim Import senden (Mandatsreferenz = Mitgliedsnummer)". Migration mit Bestand-Backfill (`sepaMandateEnabled=false` → CORE-Audit=TRUE+Timing=TRUE wegen Coupling). EEG-Kopie des PDF bei Audit-Trail (Ablage-Pflicht). UI-Konsistenz: Kurz-Erklärung unter allen 3 SEPA-Toggles wie bei anderen Settings. Cross-Field-Validation: CORE-Audit ⇒ Timing. Tester-Bitte: Label „Kontoinhaber:in" → „Kontowortlaut" mit Hint-Popover. Deployed v1.21.0-PROJ-80, wartet auf helm upgrade. | Deployed | `features/PROJ-80-sepa-settings-simplification.md` | 2026-06-08 |
| PROJ-81 | SEPA-Einwilligung optional pro Mitgliedstyp — Per-EEG-Master-Toggle + konfigurierbare Mitgliedstyp-Liste (private/farmer/association/municipality, `company` ausgenommen). Bei aktivem Toggle wird die SEPA-Einwilligungs-Checkbox im Public-Form für die gewählten Mitgliedstypen optional; Bankdaten bleiben Pflicht (Owner-Korrektur). Checkbox nicht angekreuzt → `einzugsart=kein_sepa`, kein Mandat-PDF, gelber Hinweis-Banner in 3 EEG-Mails. Backend-Defense-in-Depth-Validation via shared Helper. Settings in SEPA-Sektion unter den PROJ-80-Toggles. Beifang-Fix Excel-Einzugsart-Label-Map. Deployed `v1.22.0-PROJ-81`, wartet auf helm upgrade. | Deployed | `features/PROJ-81-sepa-optional-per-membertype.md` | 2026-06-08 |

## Next Available ID: PROJ-86
