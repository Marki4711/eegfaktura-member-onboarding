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

// PROJ-45: "PV", "PV, Speicher 10,5 kWh (Fronius)" oder "" für CONSUMPTION.
const GEN_LABELS: Record<string, string> = {
  pv: "PV", hydro: "Wasser", wind: "Wind", biomass: "Biomasse",
};
function formatGeneration(mp: MeteringPointDetail): string | null {
  if (mp.direction !== "PRODUCTION" || !mp.generationType) return null;
  const label = GEN_LABELS[mp.generationType] ?? mp.generationType;
  const extras: string[] = [];
  if (mp.batterySizeKwh != null) {
    extras.push(`Speicher ${mp.batterySizeKwh.toString().replace(".", ",")} kWh`);
  }
  if (mp.inverterManufacturer) {
    if (extras.length > 0) extras[extras.length - 1] += ` (${mp.inverterManufacturer})`;
    else extras.push(`(${mp.inverterManufacturer})`);
  }
  return extras.length > 0 ? `${label}, ${extras.join(", ")}` : label;
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
          <TableHead>Erzeugung</TableHead>
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
