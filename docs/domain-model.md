# Domain Model
## eegfaktura Member Onboarding

## 1. Ziel

Das Datenmodell für `eegfaktura Member Onboarding` ist bewusst einfach gehalten und verwendet möglichst wenige Tabellen.

Es unterstützt:
- Selbstregistrierung neuer Mitglieder
- mehrere Zählpunkte pro Antrag
- Admin-Prüfung und Freigabe
- nachvollziehbare Statushistorie
- späteren Import nach eegFaktura

Nicht Teil des Modells:
- Dokumente
- Tarife
- Rollenpflege
- Kontoinformationen
- abweichende Zählpunktadressen
- JSON-Felder

## 2. Schema

Alle Tabellen liegen im PostgreSQL-Schema:

- `member_onboarding`

## 3. Tabellen

### 3.1 `member_onboarding.application`

Zentrale Haupttabelle für einen Onboarding-Antrag.

Enthält:
- Identifikation
- EEG-Zuordnung
- Status
- Person
- Kontakt
- Adresse
- Einwilligungen
- Admin-Notiz
- Importstatus

Felder:
- `id`
- `reference_number`
- `eeg_id`
- `registration_slug`
- `status`
- `started_at`
- `submitted_at`
- `approved_at`
- `rejected_at`
- `imported_at`
- `firstname`
- `lastname`
- `birth_date`
- `email`
- `phone`
- `resident_street`
- `resident_street_number`
- `resident_zip`
- `resident_city`
- `resident_country`
- `privacy_accepted`
- `privacy_version`
- `privacy_accepted_at`
- `accuracy_confirmed`
- `communication_consent`
- `reviewed_by_user_id`
- `admin_note`
- `needs_info_reason`
- `target_participant_id`
- `import_started_at`
- `import_finished_at`
- `import_error_message`
- `created_at`
- `updated_at`

### 3.2 `member_onboarding.metering_point`

Speichert die Zählpunkte eines Antrags.

Felder:
- `id`
- `application_id`
- `metering_point`
- `direction`
- `created_at`
- `updated_at`

Regeln:
- ein Antrag kann mehrere Zählpunkte haben
- `metering_point` ist innerhalb eines Antrags eindeutig
- alle Zählpunkte übernehmen im Onboarding dieselbe Adresse wie das Mitglied

### 3.3 `member_onboarding.status_log`

Historisiert Statuswechsel eines Antrags.

Felder:
- `id`
- `application_id`
- `from_status`
- `to_status`
- `changed_by_user_id`
- `reason`
- `created_at`

## 4. Statusmodell

Erlaubte Statuswerte:
- `draft`
- `submitted`
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported`
- `import_failed`

Erlaubte Übergänge:
- `draft -> submitted`
- `submitted -> under_review`
- `under_review -> needs_info`
- `under_review -> approved`
- `under_review -> rejected`
- `needs_info -> submitted`
- `approved -> imported`
- `approved -> import_failed`
- `import_failed -> approved`

## 5. Fachregeln

- Ein Antrag enthält genau ein Mitglied.
- Ein Antrag gehört genau zu einer EEG.
- Ein Antrag wird über einen festen Registrierungslink pro EEG gestartet.
- Ein Antrag kann mehrere Zählpunkte enthalten.
- Alle Zählpunkte haben im Onboarding dieselbe Adresse wie das Mitglied.
- Tarife, Rollen und Kontoinformationen werden erst nach dem Import in eegFaktura gepflegt.
- Nur Anträge im Status `approved` dürfen importiert werden.
