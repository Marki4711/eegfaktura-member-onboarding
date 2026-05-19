-- PROJ-53: Flag wann die Beitrittsbestätigungs-Mail ans Mitglied versandt
-- wurde. NULL = noch nicht versandt. Wird beim Übergang nach `activated`
-- (sowohl regulär via ready_for_activation als auch via manueller
-- approved->activated-Skip) gesetzt — der Send-Pfad prüft das Flag und
-- sendet nicht doppelt.
--
-- Cut-off für Bestandsanträge: alle Anträge, die zum Migrations-Zeitpunkt
-- bereits in einem post-imported Status stehen, haben die alte
-- Beitrittsbestätigung beim Import-Pfad (PROJ-46 Stage B) bekommen. Damit
-- sie nicht doppelt eine Mail bekommen, setzen wir das Flag retrospektiv
-- auf NOW().

ALTER TABLE member_onboarding.application
    ADD COLUMN activation_notification_sent_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN member_onboarding.application.activation_notification_sent_at IS
    'Zeitpunkt des Versands der Beitrittsbestätigungs-Mail (PROJ-53). NULL = noch nicht versandt. Verhindert doppelten Versand.';

-- Bestandsanträge: Flag auf updated_at setzen, damit der neue Send-Pfad
-- bei einem späteren Wechsel nach activated nicht erneut sendet.
UPDATE member_onboarding.application
SET activation_notification_sent_at = COALESCE(updated_at, created_at, NOW())
WHERE status IN ('imported', 'ready_for_activation', 'awaiting_bank_confirmation', 'activated');
