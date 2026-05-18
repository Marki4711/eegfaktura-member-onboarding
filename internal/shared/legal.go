package shared

// NetworkOperatorAuthText (PROJ-44) ist der rechtsverbindliche Wortlaut der
// Netzbetreiber-Vollmacht. Single Source of Truth — Frontend
// (src/lib/api.ts: NETWORK_OPERATOR_AUTH_TEXT) und PDF müssen denselben
// Wortlaut zeigen, damit der Konsens-Snapshot im Approval-PDF dem im
// Mitgliederformular geklickten Text 1:1 entspricht.
//
// Eine Änderung hier ist rechtlich relevant und muss mit dem
// EEG-Fachverband abgestimmt werden — bei jedem Edit beide Stellen
// (Frontend + Go-Konstante) synchron halten.
const NetworkOperatorAuthText = "Ich erteile der EEG für die Dauer der Mitgliedschaft zeitlich unbegrenzt " +
	"die Vollmacht, in meinem Namen sämtliche Schritte und Abstimmungen mit " +
	"dem zuständigen Netzbetreiber durchzuführen, die zur vollständigen " +
	"(De-)Aktivierung der angeführten Zählpunkte in der EEG notwendig sind. " +
	"Dies betrifft insbesondere auch die Nutzung des Online-Portals des Netzbetreibers."
