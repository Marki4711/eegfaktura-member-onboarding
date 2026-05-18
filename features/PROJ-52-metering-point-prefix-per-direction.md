# PROJ-52: Konfigurierbarer Zählpunkt-Prefix pro Richtung + Auto-Pad + Alphanumerik

**Status:** Deployed
**Created:** 2026-05-17
**Last Updated:** 2026-05-18

## Implementierungs-Notiz (2026-05-18)

Implementiert in vier Commits:

- `e84317f` Backend — Migration 000045, Repo-Erweiterung, Submit-Prefix-Match-Validation, Alphanumerik-Regex (`^AT\d{11}[A-Z0-9]{20}$`), Public-Config + Admin-Settings GET/PUT mit Patch-Semantik (`meteringPointPrefixesPresent`).
- `7bd8f78` Admin-UI — zwei Prefix-Inputs in der EEG-Settings-Seite mit Live-Vorschau und Auto-Normalisierung (Whitespace/Dots/Hyphens entfernt, uppercase). `PrefixPreview`-Helper zeigt „AT + N Stellen frei".
- `8771a80` Public-Form — Felder-Reihenfolge umgestellt (Richtung+Faktor in Zeile 1, Zählpunkt full-width in Zeile 2), dynamische Mask mit `S=[A-Z0-9]`-Definition, Prefix-Prefill bei Direction-Wechsel, `padToMeteringPointLength`-Helper für Auto-Pad onBlur, `MeteringPointRow`-Subkomponente extrahiert.
- (dieser Commit) Docs — api-spec + domain-model + INDEX-Status.

Bewusste Abweichungen von der Skizze:
- **Mask-Lock des Prefixes nicht implementiert.** imask kann literale Digits/Letters in der Mask-Definition nicht trivial behandeln. Stattdessen wird der Prefix beim Direction-Wechsel vorbelegt und der Backend validiert das Match beim Submit. UX-Vorteil: Mitglied sieht den Prefix, kann ihn aber theoretisch überschreiben — Backend fängt das ab.
- **Trennzeichen-Normalisierung** auch Hyphens (`-`) zusätzlich zu Whitespace + Dots — Tester benutzen alle drei austauschbar.
- `mandate_reference`-Format-Anker (Spec-Detailfrage 5) bleibt out-of-scope — wird im Approval-PDF unverändert als 33-stellige Rohform gerendert.

## Hintergrund

Die Eingabe der 33-stelligen Zählpunktbezeichnung ist für Mitglieder fehleranfällig und tippaufwendig. In der Praxis gehören die meisten Zählpunkte einer EEG zum selben Netzbetreiber, häufig sogar zum selben Postleitzahl-Bereich. Eine Konfiguration der ersten Stellen pro EEG würde dem Mitglied viel Tipparbeit ersparen und Falsch-Eingaben reduzieren.

Zusätzlich wurde bei der Recherche festgestellt:
- Die heutige UI-Gruppierung `2-6-5-12-8` ist willkürlich, nicht offiziell.
- Die offizielle Struktur nach E-Control / MeteringCode ist **`2-6-5-20`**: Ländercode + Netzbetreibernummer + Postleitzahl + Zählpunktnummer.
- Die letzten 20 Stellen sind **alphanumerisch** (im aktuellen Code nur Ziffern zugelassen — Diskrepanz zur Spec, in der österreichischen Praxis allerdings meist numerisch).

Die Mask wurde bereits vorab (separater Commit, 2026-05-17) auf `2-6-5-20` korrigiert. Dieses Spec deckt die übrigen Punkte ab.

## Scope

### 1. Pro Richtung konfigurierbarer Prefix

Datenmodell auf `member_onboarding.registration_entrypoint`:

```sql
ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN metering_point_prefix_consumption VARCHAR(33) NULL,
    ADD COLUMN metering_point_prefix_production  VARCHAR(33) NULL;
```

- Beide optional. NULL = heutiges Verhalten (nur `AT` als fixer Bestandteil).
- Validierung: muss mit `AT` beginnen, Länge 2–33, Stellen 3–13 nur Ziffern, Stellen 14+ alphanumerisch.
- Die sinnvolle Länge hängt vom Netzbetreiber ab — je nachdem wie viele Stellen bei dessen Zählpunkten konstant sind und ab welcher Stelle die individuelle Kennung beginnt. Der Admin wählt selbst aus, was für seinen Bereich passt.

### 2. Automatische Anwendung je nach Richtung

Frontend-Logik:
- Mitglied wählt die Zählpunkt-Richtung (CONSUMPTION / PRODUCTION).
- Die Mask des Zählpunkt-Eingabefelds wird live aus dem passenden Prefix gebaut.
- Richtungs-Wechsel cleart das Zählpunkt-Feld (sonst wären ungültige Eingaben möglich).

**UI-Reihenfolge im Zählpunkt-Block:**
- Heute: Zählpunktnummer | Richtung | Faktor (in einer Zeile)
- Künftig: **Richtung + Faktor** (zusammen, in einer Zeile) → **Zählpunktnummer** (mit dynamischer Mask) → restliche Felder

Begründung: Richtung muss vor der Zählpunkt-Eingabe bestimmt sein, weil sie die Mask bestimmt. Richtung und Faktor bleiben aber als visueller Block zusammen, nicht durch die Zählpunktnummer getrennt.

### 3. Auto-Pad mit führenden Nullen

`onBlur` des Zählpunkt-Inputs:
- Mitglieds-Anteil (alles nach dem konfigurierten Prefix) extrahieren
- Platzhalter (`_`, Spaces) entfernen
- Wenn weniger Ziffern als erwartet → links mit `0` auffüllen bis volle Länge
- Wert neu setzen

**Beispiel** (Prefix mit z. B. 27 Stellen → 6 freie Stellen):
- Mitglied tippt `123`, klickt weg
- Beim Blur wird zu `[Prefix]000123` ergänzt

Funktioniert auch ohne Prefix-Feature — bei reinem `AT`-Pattern wären 31 Stellen frei, `12345` würde zu `0000000000000000000000000012345`.

### 4. Alphanumerik im letzten Block (E-Control-Konformität)

Validation-Anpassungen:
- Backend (`internal/shared/requests.go`): Regex `^AT\d{31}$` → `^AT\d{11}[A-Z0-9]{20}$`
- Backend (`internal/application/application_service.go`): `meteringPointRegex` analog
- Frontend (zod-Schema in `registration-form.tsx`): analog
- Frontend (Mask in `metering-point-fields.tsx`): Stellen 14–33 als alphanumerische Platzhalter (`A`/`a` statt `0` in der imask-Notation, mit Uppercase-Transform)

**Migrations-Hinweis:** Bestandsdaten sind alle numerisch — keine Migration nötig. Die Validierung wird strenger nur in eine Richtung erweitert (mehr erlaubt, nichts wird ungültig).

### 5. Admin-UI (EEG-Settings)

Neuer Block **„Zählpunkt-Prefixes"** in der EEG-Einstellungen-Seite:
- Zwei Inputs nebeneinander: „Verbraucher-Prefix" und „Einspeisungs-Prefix"
- Beide optional, beide validiert (Format wie Datenmodell oben)
- Helper-Text: „Je mehr Stellen Sie hier festlegen, desto weniger müssen Mitglieder selbst eintippen. Die sinnvolle Länge hängt davon ab, ab welcher Stelle die Zählpunkte Ihres Netzbetreibers individuell werden."
- Vorschau: Mask-Darstellung wie sie das Mitglied im Formular sieht
- Save via existierendem `PUT /api/admin/settings/eeg`

### 6. Fallback-Verhalten

Drei Defaults nach Owner-Entscheidung 2026-05-17:

- **1a (strict):** Wenn Prefix konfiguriert, gibt es keinen Override für Sonderfälle. Mitglied mit Zählpunkt aus anderem Netzbereich muss sich an die EEG wenden.
- **2a (Fallback auf reines AT):** Wenn nur eine Richtung konfiguriert ist, fällt die andere Richtung auf das heutige `AT`-only-Pattern zurück. Einspeisung wird nicht ausgegraut.
- **3a (Bestand unangetastet):** Bestehende Anträge werden bei Prefix-Änderung nicht geprüft, keine Warnung im Admin.

## API-Erweiterungen

### `GET /api/public/registration/{rc_number}`

Response um die zwei Prefix-Felder erweitern:
```json
{
  "rcNumber": "RC123456",
  ...
  "meteringPointPrefixConsumption": "AT000600100012345678901234567",
  "meteringPointPrefixProduction": "AT000600100012345678901234567"
}
```

### `GET /api/admin/settings/eeg`

Analog erweitert um beide Felder (für die EEG-Admin-UI).

### `PUT /api/admin/settings/eeg`

Akzeptiert beide neuen Felder im Body.

## Out of Scope

- Override „anderer Netzbereich" für Sonderfälle (1b)
- Einspeisung im Dropdown ausgrauen, wenn kein Production-Prefix konfiguriert (2b)
- Warnung im Admin bei Bestandsanträgen mit Prefix-Mismatch (3b)
- Multi-Prefix pro Richtung (mehrere Netzbetreiber pro EEG, EEG-Mitglied wählt aus Dropdown)
- Helper-Auswahl mit bekannten österreichischen Netzbetreiber-Codes (~12 Einträge)
- Konfigurierbares Trennzeichen (Spaces vs. Punkte wie offiziell)
- Migration bestehender Zählpunkte beim Prefix-Setzen
- Internationale Erweiterung (DE/IT-Prefixes statt nur AT)

Alle können später als eigene Features ergänzt werden.

## Acceptance Criteria (Skizze, vor Umsetzung verfeinern)

1. Migration legt die zwei neuen Spalten an.
2. Admin-Settings-UI zeigt die zwei Prefix-Inputs mit Live-Validierung.
3. `PUT /api/admin/settings/eeg` speichert beide Felder.
4. Public-Form bekommt die Prefixes über `GET /api/public/registration/{rc}` und baut die Mask dynamisch je nach gewählter Richtung.
5. Reihenfolge im Zählpunkt-Block: Richtung+Faktor → Zählpunktnummer (mit dynamischer Mask) → restliche Felder.
6. Auto-Pad mit führenden Nullen greift im `onBlur` der Zählpunkt-Eingabe.
7. Backend prüft das Prefix-Match bei Submit (defense-in-depth).
8. Alphanumerische Zeichen im letzten 20-Stellen-Block werden akzeptiert (Validation + Mask).
9. Wenn nur eine Richtung konfiguriert ist, fällt die andere auf das `AT`-only-Verhalten zurück.
10. Bestandsanträge werden nicht geprüft, kein Backend-Crash bei Prefix-Mismatch.

## Offene Detailfragen (vor Umsetzung)

1. **Trennzeichen in der Admin-Eingabe:** soll der Admin Prefixes mit Spaces, Punkten oder ohne Trennzeichen eingeben dürfen? Vorschlag: alle Trennzeichen akzeptieren, Backend normalisiert auf reine 33-stellige Form.
2. **Default-Mask bei 0 Prefixes:** weiterhin `AT 000000 00000 [20-stellig]`, oder offizielle Punkt-Trennung `AT.000000.00000.[20-stellig]`? Eher Spaces beibehalten (Konsistenz mit Bestand).
3. **Vorschau im Admin:** kompakter Inline-Render der resultierenden Mask, oder eigener Preview-Bereich? Tendenz: inline reicht.
4. **Auto-Pad bei alphanumerischem Inhalt:** wenn die letzten 6 Stellen alphanumerisch sind, mit `0` oder mit Leerzeichen padden? Vorschlag: weiter `0`, weil 99% numerisch.
5. **`mandate_reference`-Format-Anker:** soll der eingespielte Wert bei der Ausgabe im PDF mit Spaces/Punkten gerendert werden, oder weiterhin als 33-stellige Rohform?

## Hinweis

Die Mask-Korrektur auf `2-6-5-20` ist separat vorab gemerged worden (im selben Tagesblock 2026-05-17), damit das Repo nicht in einem Zwischenzustand bleibt. Alle weiteren Änderungen dieses Specs sind Teil der eigentlichen PROJ-52-Implementierung.
