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
        </TableRow>
      </TableHeader>
      <TableBody>
        {meteringPoints.map((mp) => (
          <TableRow key={mp.id}>
            <TableCell className="font-mono text-sm">{mp.meteringPoint}</TableCell>
            <TableCell>
              {mp.direction === "CONSUMPTION" ? "Bezug" : "Einspeisung"}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
