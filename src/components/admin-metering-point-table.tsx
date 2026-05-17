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
        </TableRow>
      </TableHeader>
      <TableBody>
        {meteringPoints.map((mp) => {
          const addr = formatAddress(mp);
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
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
