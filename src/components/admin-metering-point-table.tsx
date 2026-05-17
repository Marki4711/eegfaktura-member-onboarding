import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { MeteringPointDetail } from "@/lib/api";

interface Props {
  meteringPoints: MeteringPointDetail[];
}

function formatAddress(mp: MeteringPointDetail): string | null {
  if (!mp.addressStreet) return null;
  const street = [mp.addressStreet, mp.addressStreetNumber].filter(Boolean).join(" ");
  const cityLine = [mp.addressZip, mp.addressCity].filter(Boolean).join(" ");
  return [street, cityLine].filter(Boolean).join(", ");
}

// Detail-Zeile pro Zählpunkt — analog zu FormatGenerationLine im Backend
// (internal/mail/service.go), damit Admin-UI und Mail/PDF dasselbe rendern.
const GEN_LABELS: Record<string, string> = {
  pv: "PV", hydro: "Wasser", wind: "Wind", biomass: "Biomasse",
};
function fmtNum(v: number): string {
  return v.toString().replace(".", ",");
}
function formatGeneration(mp: MeteringPointDetail): string | null {
  // CONSUMPTION: Verbrauchsdaten
  if (mp.direction === "CONSUMPTION") {
    const parts: string[] = [];
    if (mp.consumptionPreviousYear != null) {
      parts.push(`Verbrauch Vorjahr ${mp.consumptionPreviousYear} kWh`);
    }
    if (mp.consumptionForecast != null) {
      parts.push(`Prognose ${mp.consumptionForecast} kWh`);
    }
    return parts.length > 0 ? parts.join(", ") : null;
  }
  if (mp.direction !== "PRODUCTION" || !mp.generationType) return null;

  // PRODUCTION
  const label = GEN_LABELS[mp.generationType] ?? mp.generationType;
  const parts: string[] = [];
  if (mp.pvPowerKwp != null && mp.generationType === "pv") {
    parts.push(`${label} ${fmtNum(mp.pvPowerKwp)} kWp`);
  } else {
    parts.push(label);
  }
  if (mp.feedInForecast != null) {
    parts.push(`Prognose ${mp.feedInForecast} kWh/J`);
  }
  if (mp.batterySizeKwh != null) {
    let entry = `Speicher ${fmtNum(mp.batterySizeKwh)} kWh`;
    if (mp.inverterManufacturer) entry += ` (${mp.inverterManufacturer})`;
    parts.push(entry);
  } else if (mp.inverterManufacturer) {
    parts.push(`Wechselrichter ${mp.inverterManufacturer}`);
  }
  if (mp.feedInLimitPresent) {
    parts.push(
      mp.feedInLimitKw != null
        ? `Einspeiselimit ${fmtNum(mp.feedInLimitKw)} kW`
        : "Einspeiselimit vorhanden",
    );
  }
  if (mp.batteryControlAcceptable != null) {
    parts.push(`Speichersteuerung im Sinne der EEG: ${mp.batteryControlAcceptable ? "Ja" : "Nein"}`);
  }
  return parts.join(", ");
}

export function AdminMeteringPointTable({ meteringPoints }: Props) {
  if (meteringPoints.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4">
        Keine Zählpunkte vorhanden.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Zählpunktnummer</TableHead>
          <TableHead>Richtung</TableHead>
          <TableHead className="text-right">Teilnahmefaktor</TableHead>
          <TableHead>Adresse (falls abweichend)</TableHead>
          <TableHead>Details</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {meteringPoints.map((mp) => {
          const addr = formatAddress(mp);
          const gen = formatGeneration(mp);
          return (
            <TableRow key={mp.id}>
              <TableCell className="font-mono text-sm">{mp.meteringPoint}</TableCell>
              <TableCell>
                {mp.direction === "CONSUMPTION" ? "Bezug" : "Einspeisung"}
              </TableCell>
              <TableCell className="text-right">{mp.participationFactor} %</TableCell>
              <TableCell className={addr ? "" : "text-muted-foreground"}>
                {addr ?? "—"}
              </TableCell>
              <TableCell className={gen ? "" : "text-muted-foreground"}>
                {gen ?? "—"}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
