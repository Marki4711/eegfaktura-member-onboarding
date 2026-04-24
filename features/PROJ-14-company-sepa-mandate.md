# PROJ-14: SEPA-Firmenlastschriftmandat für Unternehmen

## Status: In Review
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

## Dependencies
- Requires: PROJ-12 (SEPA-Lastschriftmandat PDF) — bestehende PDF-Infrastruktur wird erweitert
- Requires: PROJ-7 (Mitgliedstypen) — Erkennung von `company`, `association`
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Konfiguration nur für authentifizierte Admins

## Hintergrund

SEPA kennt zwei Mandatstypen:
- **SEPA-Lastschriftmandat (CORE):** für Privatpersonen und natürliche Personen
- **SEPA-Firmenlastschriftmandat (B2B):** ausschließlich für Konten von Unternehmen; weicht in Ermächtigungstext, Layout und Rückbuchungsregeln vom CORE-Mandat ab

Standardmäßig erhalten alle Mitglieder das CORE-Mandat. EEGs, die das B2B-Verfahren einsetzen, sollen Mitglieder vom Typ `company` und `association` stattdessen das Firmenlastschriftmandat erhalten.

## User Stories

- Als **EEG-Administrator** möchte ich festlegen können, ob Unternehmen und Verbände das SEPA-Firmenlastschriftmandat (B2B) erhalten statt des Standard-Mandats, damit das korrekte Mandat für das Einzugsverfahren meiner EEG versendet wird.
- Als **neues Mitglied** (Unternehmen/Verband) möchte ich das für meine Rechtsform passende SEPA-Mandat als PDF-Anhang erhalten, damit ich das richtige Dokument unterschreibe und zurückschicke.
- Als **EEG-Administrator** möchte ich, dass Privatmitglieder immer das CORE-Mandat erhalten, unabhängig von der B2B-Einstellung, damit kein falsches Mandat versendet wird.

## Acceptance Criteria

### Konfiguration im Admin-Backend
- [ ] Auf der Einstellungsseite der EEG gibt es im SEPA-Abschnitt einen neuen Toggle: **„Firmenlastschrift (B2B) für Unternehmen und Verbände verwenden"**
- [ ] Der Toggle ist per EEG steuerbar; Standard: **deaktiviert**
- [ ] Der Toggle ist nur sichtbar, wenn `sepa_mandate_enabled = true`
- [ ] Die Einstellung wird als `use_company_sepa_mandate BOOLEAN` in `member_onboarding.registration_entrypoint` gespeichert
- [ ] Änderungen sind sofort wirksam (kein Cache)

### PDF-Auswahl
- [ ] Wenn `use_company_sepa_mandate = false` (Standard): alle Mitglieder erhalten das CORE-Mandat (Verhalten wie bisher)
- [ ] Wenn `use_company_sepa_mandate = true`:
  - Mitglieder vom Typ `company` oder `association` → **SEPA-Firmenlastschriftmandat**
  - Alle anderen Typen (`private`, `farmer`, `municipality`) → weiterhin CORE-Mandat

### PDF-Inhalt (SEPA-Firmenlastschriftmandat)
- [ ] Das PDF enthält alle Pflichtbestandteile des SEPA-B2B-Mandats:
  - **Mandatsreferenz:** Leer-Zeile mit Hinweis „wird von [EEG-Name] ausgefüllt"
  - **Zahlungsempfänger (Abschnitt „ZAHLUNGSEMPFÄNGER"):**
    - Creditor CD (= Creditor-ID der EEG)
    - Name (= EEG-Name)
    - Anschrift (Straße + Hausnummer, PLZ + Ort)
  - **Ermächtigungstext:** standardisierter B2B-Text (Firmenlastschrift-Wortlaut, nicht CORE-Wortlaut):
    > „Ich ermächtige/Wir ermächtigen [EEG-Name], Zahlungen von meinem/unserem Konto mittels SEPA-Firmenlastschriften einzuziehen. Zugleich weise ich mein/weisen wir unser Kreditinstitut an, die von [EEG-Name] auf mein/unser Konto gezogenen SEPA-Firmenlastschriften einzulösen.
    > Hinweis: Dieses SEPA-Firmenlastschrift-Mandat dient nur dem Einzug von SEPA-Firmenlastschriften, die auf Konten von Unternehmen gezogen sind. Ich bin/Wir sind nicht berechtigt, nach der erfolgten Einlösung eine Erstattung des belasteten Betrages zu verlangen. Ich bin/Wir sind berechtigt, mein/unser Kreditinstitut bis zum Fälligkeitstag anzuweisen, SEPA-Firmenlastschriften nicht einzulösen."
  - **Zahlungsart:** Checkboxen „einmalig" und „wiederkehrend" — **„wiederkehrend" vorausgewählt** (✓)
  - **Zahlungspflichtiger (Abschnitt „ZAHLUNGSPFLICHTIGER"):**
    - Name (= Firmenname des Mitglieds aus `company_name`, Fallback auf `firstname + lastname`)
    - Anschrift (Straße + Hausnummer, PLZ + Ort — aus Antrag)
    - IBAN (aus Antrag)
    - BIC*-Feld (leer — zum Ausfüllen)
  - **Unterschriftsfeld:** Ort, Datum, Unterschrift (leer)
  - **BIC-Fußnote:** „* Seit 01.06.2016 kann die Angabe des BIC bei nationalen und grenzüberschreitenden Lastschriften entfallen."
- [ ] Das PDF ist auf Deutsch
- [ ] Dateiname: `sepa-firmenlastschriftmandat.pdf`

### Verhalten bei fehlenden Pflichtdaten
- [ ] Fehlen EEG-Stammdaten (Name, Adresse, Creditor-ID): kein PDF generiert, kein Fehler für das Mitglied (bestehende Regel aus PROJ-12 gilt auch hier)

### Regression
- [ ] CORE-Mandat-Verhalten für Privatmitglieder bleibt unverändert
- [ ] EEGs mit `use_company_sepa_mandate = false` senden weiterhin ausschließlich CORE-Mandate

## Edge Cases

- Mitglied mit Typ `company` und `use_company_sepa_mandate = false` → erhält CORE-Mandat (nicht B2B)
- Mitglied mit Typ `farmer` und `use_company_sepa_mandate = true` → erhält CORE-Mandat (Landwirte sind keine Unternehmen im B2B-Sinn)
- `municipality` mit `use_company_sepa_mandate = true` → erhält CORE-Mandat (Gemeinden sind keine Unternehmen)
- `company_name` ist NULL bei Typ `company` → Firmenname-Fallback auf `firstname + lastname`
- `use_company_sepa_mandate = true` aber `sepa_mandate_enabled = false` → kein PDF (Toggle für B2B gilt nur wenn SEPA generell aktiv)
