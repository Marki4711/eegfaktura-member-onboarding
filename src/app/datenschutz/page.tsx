import type { Metadata } from "next";
import { PublicHeader } from "@/components/public-header";

export const metadata: Metadata = {
  title: "Datenschutzerklärung – eegFaktura Mitglieder-Onboarding",
};

export default function DatenschutzPage() {
  return (
    <div className="min-h-screen flex flex-col">
      <PublicHeader />
      <main className="flex-1 py-10 px-4">
        <div className="max-w-3xl mx-auto space-y-8 text-sm text-foreground leading-relaxed">
          <div>
            <h1 className="text-2xl font-bold tracking-tight mb-2">Datenschutzerklärung</h1>
            <p className="text-muted-foreground text-xs">Stand: April 2026</p>
          </div>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">1. Verantwortlicher</h2>
            <p>
              Verantwortlicher im Sinne der Datenschutz-Grundverordnung (DSGVO) und des
              österreichischen Datenschutzgesetzes (DSG) ist die jeweilige
              Energiegemeinschaft (EEG), über deren Registrierungslink Sie auf dieses
              Formular gelangt sind. Die genauen Kontaktdaten entnehmen Sie bitte dem
              Anschreiben oder der Website Ihrer Energiegemeinschaft.
            </p>
            <p>
              Technischer Betreiber dieser Plattform ist:
            </p>
            <address className="not-italic pl-4 border-l-2 border-border text-muted-foreground space-y-0.5">
              <p className="font-medium text-foreground">eegFaktura</p>
              <p>c/o Softwareentwicklung eegFaktura</p>
              <p>E-Mail: <a href="mailto:office@eegfaktura.at" className="underline hover:text-foreground">office@eegfaktura.at</a></p>
            </address>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">2. Welche Daten werden verarbeitet?</h2>
            <p>Im Rahmen der Mitglieder-Registrierung verarbeiten wir folgende personenbezogene Daten:</p>
            <ul className="list-disc pl-5 space-y-1">
              <li>Mitgliedstyp (Privatperson, Unternehmen, Gemeinde, Verein, Landwirt)</li>
              <li>Vor- und Nachname oder Firmen-/Organisationsname</li>
              <li>Geburtsdatum (sofern abgefragt)</li>
              <li>Firmenbuchnummer, Vereinsnummer oder UID-Nummer (sofern zutreffend)</li>
              <li>Anschrift (Straße, Hausnummer, PLZ, Ort)</li>
              <li>E-Mail-Adresse und Telefonnummer (sofern abgefragt)</li>
              <li>IBAN und Name des/der Kontoinhabers/in für den SEPA-Lastschrifteinzug</li>
              <li>Zählpunktnummern, Verbrauchsrichtung und Beteiligungsfaktor</li>
              <li>Optionale technische Angaben (PV-Leistung, Verbrauchsprognose, Haushaltsgröße u. a.), sofern von der Energiegemeinschaft abgefragt</li>
              <li>Zeitstempel der Antragstellung sowie technische Protokolldaten (IP-Adresse, Zeitpunkt)</li>
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">3. Zweck und Rechtsgrundlage der Verarbeitung</h2>
            <p>Ihre Daten werden ausschließlich für folgende Zwecke verarbeitet:</p>
            <ul className="list-disc pl-5 space-y-2">
              <li>
                <span className="font-medium">Mitglieder-Aufnahme in die Energiegemeinschaft</span> —
                Prüfung, Verwaltung und Genehmigung Ihres Beitrittsantrags
                (Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO – Vertragsanbahnung)
              </li>
              <li>
                <span className="font-medium">SEPA-Lastschriftmandat</span> —
                Einzug von Rechnungsbeträgen auf Basis Ihrer Einwilligung
                (Rechtsgrundlage: Art. 6 Abs. 1 lit. a DSGVO – Einwilligung)
              </li>
              <li>
                <span className="font-medium">Kommunikation</span> —
                Benachrichtigung über den Bearbeitungsstand Ihres Antrags per E-Mail
                (Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO)
              </li>
              <li>
                <span className="font-medium">Spam-Schutz</span> —
                Verwendung von Cloudflare Turnstile zur Missbrauchsverhinderung,
                sofern aktiviert (Rechtsgrundlage: Art. 6 Abs. 1 lit. f DSGVO –
                berechtigtes Interesse)
              </li>
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">4. Empfänger Ihrer Daten</h2>
            <p>
              Ihre Daten werden nach Genehmigung Ihres Antrags in das Mitgliederverwaltungssystem
              <em> eegFaktura</em> der jeweiligen Energiegemeinschaft übertragen. Eine
              Weitergabe an Dritte außerhalb der Energiegemeinschaft erfolgt nicht, sofern
              keine gesetzliche Verpflichtung besteht.
            </p>
            <p>
              Technische Dienstleister (Hosting, Datenbankbetrieb) verarbeiten Daten
              ausschließlich im Auftrag und nach Weisung gemäß Art. 28 DSGVO.
            </p>
            <p>
              Bei Nutzung von Cloudflare Turnstile werden Daten (z. B. IP-Adresse,
              Browser-Informationen) an Cloudflare Inc., San Francisco, USA übermittelt.
              Cloudflare ist unter dem EU-U.S. Data Privacy Framework zertifiziert.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">5. Speicherdauer</h2>
            <p>
              Nicht genehmigte oder abgelehnte Anträge werden nach Abschluss des
              Prüfverfahrens gelöscht, spätestens jedoch nach 12 Monaten ab
              Antragstellung.
            </p>
            <p>
              Nach Aufnahme in die Energiegemeinschaft werden Ihre Stammdaten für die
              Dauer der Mitgliedschaft sowie die gesetzlich vorgeschriebenen
              Aufbewahrungsfristen (z. B. 7 Jahre für buchhalterische Unterlagen) gespeichert.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">6. Ihre Rechte</h2>
            <p>Sie haben folgende Rechte bezüglich Ihrer personenbezogenen Daten:</p>
            <ul className="list-disc pl-5 space-y-1">
              <li><span className="font-medium">Auskunft</span> (Art. 15 DSGVO) – Sie können Auskunft über die von uns gespeicherten Daten verlangen.</li>
              <li><span className="font-medium">Berichtigung</span> (Art. 16 DSGVO) – Sie können die Korrektur unrichtiger Daten fordern.</li>
              <li><span className="font-medium">Löschung</span> (Art. 17 DSGVO) – Sie können die Löschung Ihrer Daten verlangen, sofern keine gesetzlichen Aufbewahrungspflichten entgegenstehen.</li>
              <li><span className="font-medium">Einschränkung der Verarbeitung</span> (Art. 18 DSGVO)</li>
              <li><span className="font-medium">Datenübertragbarkeit</span> (Art. 20 DSGVO)</li>
              <li><span className="font-medium">Widerspruch</span> (Art. 21 DSGVO) – gegen die Verarbeitung auf Basis berechtigter Interessen.</li>
              <li>
                <span className="font-medium">Widerruf einer Einwilligung</span> –
                Sie können eine erteilte Einwilligung jederzeit mit Wirkung für die Zukunft
                widerrufen. Die Rechtmäßigkeit der bis zum Widerruf erfolgten Verarbeitung
                bleibt unberührt.
              </li>
            </ul>
            <p>
              Zur Ausübung Ihrer Rechte wenden Sie sich bitte an die in Abschnitt 1
              genannte Energiegemeinschaft oder an{" "}
              <a href="mailto:office@eegfaktura.at" className="underline hover:text-foreground">
                office@eegfaktura.at
              </a>.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">7. Beschwerderecht</h2>
            <p>
              Sie haben das Recht, bei der österreichischen Datenschutzbehörde Beschwerde
              einzulegen:
            </p>
            <address className="not-italic pl-4 border-l-2 border-border text-muted-foreground space-y-0.5">
              <p className="font-medium text-foreground">Österreichische Datenschutzbehörde</p>
              <p>Barichgasse 40–42, 1030 Wien</p>
              <p>
                Web:{" "}
                <a
                  href="https://www.dsb.gv.at"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline hover:text-foreground"
                >
                  www.dsb.gv.at
                </a>
              </p>
            </address>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold">8. Technische Sicherheit</h2>
            <p>
              Die Übertragung Ihrer Daten erfolgt ausschließlich über verschlüsselte
              HTTPS-Verbindungen. Die Speicherung erfolgt in einer gesicherten
              PostgreSQL-Datenbank in einem europäischen Rechenzentrum.
            </p>
          </section>

          <p className="text-xs text-muted-foreground pt-4 border-t border-border">
            Diese Datenschutzerklärung bezieht sich auf das eegFaktura Mitglieder-Onboarding-System.
            Die jeweilige Energiegemeinschaft kann ergänzende Datenschutzhinweise bereitstellen.
          </p>
        </div>
      </main>
      <footer className="py-4 px-4 border-t border-border text-center text-xs text-muted-foreground">
        © {new Date().getFullYear()} eegFaktura — Energiegemeinschaften einfach verwalten
      </footer>
    </div>
  );
}
