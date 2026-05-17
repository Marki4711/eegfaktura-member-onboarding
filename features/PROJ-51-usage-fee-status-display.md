# PROJ-51: Anzeige offener Nutzungsgebühren im Admin-UI

**Status:** On Hold
**Created:** 2026-05-17
**Last Updated:** 2026-05-17

## Hintergrund

Das Member-Onboarding-Tool wird Vereinen zur Nutzung bereitgestellt. Der Betreiber stellt Nutzungsgebühren in Rechnung. Es soll im Admin-Bereich klar erkennbar sein, ob die aktuelle Periode bezahlt wurde — als sanfter Hinweis, **ohne den Betrieb einzuschränken**.

## Scope (Minimalvariante)

### Datenmodell

Eine neue Spalte auf `member_onboarding.registration_entrypoint`:

- `paid_until` DATE NULL — Datum bis zu dem die Nutzungsgebühr beglichen ist.
  - `NULL` = noch kein Eintrag (z. B. neu erstellte EEG in der Übergangsphase)
  - `>= heute` = aktuell bezahlt, kein Hinweis nötig
  - `< heute` = überfällig → Banner im Admin-UI

### Backend

- `RegistrationEntrypoint`-Model + Repo um Spalte erweitern.
- Spalte in `GET /api/admin/settings/eeg` Response liefern (read-only für EEG-Admin).
- Neuer dedizierter Endpoint `PUT /api/admin/settings/eeg/paid-until` zum Setzen, **nur Superuser** (verhindert, dass EEG-Admin sich selbst auf „bezahlt" stempelt).

### Frontend

- Neue `UsageFeeBanner`-Komponente im Admin-Layout (oben, persistent über alle Admin-Seiten).
- Sichtbar wenn `paidUntil < today` oder `paidUntil === null` (Default: nach einer Übergangs-Karenz von z. B. 30 Tagen seit EEG-Erstellung).
- Visuell sanft: gelber/oranger Hinweis-Stil, keine roten Warnzeichen.
- Text generisch, z. B.: „Hinweis: Die Nutzungsgebühr für diesen Zeitraum ist offen. Der Betrieb läuft unverändert weiter — bitte begleichen Sie die Rechnung."
- **Kein Banner im Public-Form** (Mitglieder sollen nicht mitbekommen, dass ihre EEG eine Rechnung offen hat — Reputations-Schaden).
- Optional: konfigurierbarer Link auf eine Zahlungs-Anleitung (z. B. „Zahlungsdetails ansehen"), Inhalt via Superuser-Setting (`payment_info_url` o. ä.).

### Sichtbarkeits-Regeln

- Banner erscheint nur im **Admin-Bereich**, nie im Public-Form (`/register/{rc}`).
- Banner ist über alle Admin-Seiten hinweg konsistent sichtbar.
- Schließbar (X), aber State nur Session-lokal — beim nächsten Login wieder sichtbar.

## Out of Scope (bewusst)

- Keine Funktions-Einschränkung. Public-Form, Admin-Aktionen, Import, Mailer laufen unverändert weiter.
- Keine harten Status-Stufen (active/payment_due/overdue/suspended/terminated).
- Keine Auto-Mahn-Mails (manueller Versand durch Betreiber).
- Keine Audit-Logs der Banner-Anzeigen.
- Keine Pre-Notification-Mails vor Fälligkeit.
- Keine Selbstbedienung für die EEG zur Markierung „bezahlt".

## Offene Fragen (vor Umsetzung zu klären)

1. **Default-Wert für neue EEGs:** `NULL` mit X-Tage-Karenz, oder bei Anlage automatisch auf „heute + 1 Jahr" setzen (analog zur ersten Rechnung)?
2. **Karenzzeit bei `NULL`:** Wie lange darf das Feld leer bleiben, bevor der Banner erscheint? Empfehlung: 30–60 Tage nach EEG-Erstellung.
3. **Banner-Text:** soll der Text konfigurierbar sein (Superuser-Setting) oder fest verdrahtet?
4. **Zahlungs-Info:** soll im Banner ein Link auf Zahlungsinfo erscheinen? Wenn ja, woher der Inhalt — fester URL, eigener Settings-Block, oder externer Markdown-Editor?
5. **Audit:** soll der Wert `paid_until` versioniert/protokolliert werden (wer hat wann auf welchen Wert gesetzt), oder reicht das aktuelle Datum?
6. **Multi-Tenant-Sicht für Superuser:** soll es eine eigene Übersichtsseite „Welche EEGs sind überfällig?" geben, oder reicht das Setzen pro EEG?

## Acceptance Criteria (Skizze, vor Umsetzung verfeinern)

1. Migration legt `paid_until DATE NULL` an.
2. `GET /api/admin/settings/eeg` liefert das Feld.
3. `PUT /api/admin/settings/eeg/paid-until` ist superuser-only (403 für reguläre EEG-Admins).
4. Banner erscheint im Admin-UI bei überfälligem `paid_until`.
5. Public-Form rendert keinen Banner und kennt das Feld nicht.
6. Alle bestehenden Funktionen (Public-Form, Admin-Review, Import, Mail) laufen unabhängig vom Banner.

## Begründung „On Hold"

Die Überlegungen zum Geschäftsmodell, Tarif-Niveau und Abrechnungs-Mechanik (jährliche SEPA-Pauschale vs. anderes Modell, Tooling-Stack für Rechnungserstellung, Übergang zu größerem Volumen) sind noch nicht abgeschlossen. Eine Implementierung dieses Banner-Features ergibt erst Sinn, sobald geklärt ist, **wer wann zahlt und wie die Status-Pflege ablaufen soll** (manuell durch Betreiber, halbautomatisch via SEPA-Lauf-Result, oder anderes).

Bei Wiederaufnahme zuerst die offenen Fragen oben durchgehen, dann Spec-Status auf `Planned` setzen.
