"use client";

import { useState, useEffect } from "react";
import { useFieldArray, type UseFormReturn } from "react-hook-form";
import { Info, PlusCircle, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { MaskedInput } from "@/components/ui/masked-input";
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { resolveFieldState, CONFIGURABLE_FIELDS, GENERATION_TYPES, type FieldConfig } from "@/lib/api";
import type { RegistrationFormValues } from "./registration-form";

interface MeteringPointFieldsProps {
  form: UseFormReturn<RegistrationFormValues>;
  fieldConfig?: FieldConfig;
  // PROJ-52: pro-Richtung Zählpunkt-Prefix aus der EEG-Config. NULL ⇒
  // kein Prefill, Mitglied tippt alles ab "AT" selbst. Wenn gesetzt,
  // wird die Mask beim Direction-Wechsel mit dem Prefix vorbelegt und
  // der onBlur-Auto-Pad füllt den Rest mit führenden Nullen auf 33 auf.
  prefixConsumption?: string | null;
  prefixProduction?: string | null;
}

// PROJ-52: imask definitions — `0` ist Default-Digit, `S` neue Klasse
// für die alphanumerischen Stellen 14–33 (E-Control-Spec).
const MP_MASK_DEFINITIONS = { S: /[A-Z0-9]/ } as const;

// buildMeteringPointMask (PROJ-52 Mini-Lücke „Mask-Lock"): baut die
// imask-Mask-String dynamisch je nach aktivem EEG-Prefix. Stellen
// innerhalb des Prefixes werden als literal-escapete Zeichen
// emittiert (`\X`), sodass der Mitglied sie weder überschreiben
// noch backspacen kann. Stellen außerhalb bleiben Placeholder:
//   - Position 1-2: AT (immer literal — Länder-Code)
//   - Position 3-13: `0` (Digit; Netzbetreibernummer + PLZ, 11 Stellen)
//   - Position 14-33: `S` ([A-Z0-9]; Zählpunkt-Kennung, 20 Stellen)
// Gruppierung 2-6-5-20: Spaces nach Position 2, 8 und 13.
//
// Ein leerer/NULL Prefix liefert die Default-Mask ohne Locks.
function buildMeteringPointMask(activePrefix: string | null): string {
  const out: string[] = [];
  for (let i = 1; i <= 33; i++) {
    if (activePrefix && i <= activePrefix.length) {
      // Inside configured prefix — escape so imask treats as literal,
      // unabhängig davon ob das Zeichen sonst Placeholder wäre (z. B. `S`).
      out.push("\\" + activePrefix[i - 1]);
    } else if (i === 1) {
      out.push("A");
    } else if (i === 2) {
      out.push("T");
    } else if (i <= 13) {
      out.push("0");
    } else {
      out.push("S");
    }
    if (i === 2 || i === 8 || i === 13) {
      out.push(" ");
    }
  }
  return out.join("");
}

// padToMeteringPointLength (PROJ-52): füllt zwischen Prefix und
// Mitglieds-Anteil mit führenden Nullen auf 33 Stellen auf. Lässt
// Eingaben, die bereits voll sind oder nicht mit AT/Prefix beginnen,
// unverändert.
function padToMeteringPointLength(raw: string, activePrefix: string | null): string {
  const stripped = raw.replace(/\s/g, "").toUpperCase();
  if (!stripped.startsWith("AT")) return stripped;
  if (stripped.length >= 33) return stripped.substring(0, 33);
  const locked = activePrefix && stripped.startsWith(activePrefix) ? activePrefix : "AT";
  const rest = stripped.substring(locked.length);
  const needed = 33 - locked.length - rest.length;
  if (needed <= 0) return stripped;
  return locked + "0".repeat(needed) + rest;
}

export function MeteringPointFields({
  form,
  fieldConfig,
  prefixConsumption,
  prefixProduction,
}: MeteringPointFieldsProps) {
  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "meteringPoints",
  });

  function mpfs(name: string) {
    const field = CONFIGURABLE_FIELDS.meteringPoint.find((f) => f.name === name);
    return resolveFieldState(fieldConfig, name, field?.defaultState ?? "hidden");
  }

  const showTransformer = mpfs("transformer") !== "hidden";
  const showInstallationNumber = mpfs("installation_number") !== "hidden";
  const showInstallationName = mpfs("installation_name") !== "hidden";
  const hasExtraMpFields = showTransformer || showInstallationNumber || showInstallationName;
  // PROJ-45: Batterie + Wechselrichter sind PV-only und PROJ-8-konfigurierbar.
  const showBatterySize = mpfs("battery_size_kwh") !== "hidden";
  const showInverter = mpfs("inverter_manufacturer") !== "hidden";
  // PROJ-49: Energie-Felder pro Zählpunkt.
  const showConsumptionPrev = mpfs("consumption_previous_year") !== "hidden";
  const showConsumptionFc   = mpfs("consumption_forecast") !== "hidden";
  const showFeedInForecast  = mpfs("feed_in_forecast") !== "hidden";
  const showPvPower         = mpfs("pv_power_kwp") !== "hidden";
  const showFeedInLimit     = mpfs("feed_in_limit_kw") !== "hidden";
  // PROJ-49 follow-up: „Speichersteuerung im Sinne der EEG vorstellbar?"
  const showBatteryControl  = mpfs("battery_control_acceptable") !== "hidden";

  return (
    <div className="space-y-4">
      {fields.map((field, index) => (
        <MeteringPointRow
          key={field.id}
          form={form}
          index={index}
          canRemove={fields.length > 1}
          onRemove={() => remove(index)}
          prefixConsumption={prefixConsumption ?? null}
          prefixProduction={prefixProduction ?? null}
          showTransformer={showTransformer}
          showInstallationNumber={showInstallationNumber}
          showInstallationName={showInstallationName}
          hasExtraMpFields={hasExtraMpFields}
          showFeedInForecast={showFeedInForecast}
          showPvPower={showPvPower}
          showFeedInLimit={showFeedInLimit}
          showBatterySize={showBatterySize}
          showInverter={showInverter}
          showBatteryControl={showBatteryControl}
          showConsumptionPrev={showConsumptionPrev}
          showConsumptionFc={showConsumptionFc}
          requiredOf={mpfs}
        />
      ))}

      {fields.length < 10 && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => append({ meteringPoint: "", direction: "CONSUMPTION", participationFactor: 100, generationType: "pv" })}
        >
          <PlusCircle className="h-4 w-4 mr-2" />
          Zählpunkt hinzufügen
        </Button>
      )}
    </div>
  );
}

// MeteringPointRow renders the layout of a single Zählpunkt:
//   Zeile 1: Richtung + Faktor + Lösch-Button
//   Zeile 2: Zählpunkt full-width (33-stellige Mask)
//   darunter: Abweichende Adresse, Erzeugung/Batterie/Verbrauch,
//             optionale Zusatz-Felder (Transformator, Anlagen-Nr., …)
//
// Eigener useEffect: bei Direction-Wechsel wird die Zählpunkt-Mask mit
// dem passenden Prefix vorbelegt (PROJ-52). Erst-Mount-Befüllung wird
// nicht überschrieben — der Effekt greift nur, wenn das Feld noch
// keinen Inhalt hat (Mitglied hat noch nichts eingegeben) ODER wenn
// die Direction sich tatsächlich ändert.
function MeteringPointRow({
  form,
  index,
  canRemove,
  onRemove,
  prefixConsumption,
  prefixProduction,
  showTransformer,
  showInstallationNumber,
  showInstallationName,
  hasExtraMpFields,
  showFeedInForecast,
  showPvPower,
  showFeedInLimit,
  showBatterySize,
  showInverter,
  showBatteryControl,
  showConsumptionPrev,
  showConsumptionFc,
  requiredOf,
}: {
  form: UseFormReturn<RegistrationFormValues>;
  index: number;
  canRemove: boolean;
  onRemove: () => void;
  prefixConsumption: string | null;
  prefixProduction: string | null;
  showTransformer: boolean;
  showInstallationNumber: boolean;
  showInstallationName: boolean;
  hasExtraMpFields: boolean;
  showFeedInForecast: boolean;
  showPvPower: boolean;
  showFeedInLimit: boolean;
  showBatterySize: boolean;
  showInverter: boolean;
  showBatteryControl: boolean;
  showConsumptionPrev: boolean;
  showConsumptionFc: boolean;
  requiredOf: (name: string) => string;
}) {
  const direction = form.watch(`meteringPoints.${index}.direction`);
  const activePrefix =
    direction === "PRODUCTION" ? prefixProduction : prefixConsumption;
  // PROJ-52 Mini-Lücke „Mask-Lock": Dynamische imask-Mask, in der die
  // Prefix-Stellen als literal-escapete Zeichen emittiert werden. Damit
  // kann das Mitglied weder den Prefix überschreiben noch backspacen —
  // die EEG-Vorgabe ist im Frontend hart, das Backend bleibt dahinter
  // als zweite Verteidigungslinie. Recomputed pro Direction-Wechsel,
  // weil sich der aktive Prefix dann ändert.
  const dynamicMask = buildMeteringPointMask(activePrefix);

  // PROJ-52: bei Direction-Wechsel die Zählpunkt-Mask mit dem passenden
  // Prefix vorbelegen — aber nur, wenn der bisherige Wert leer ist oder
  // mit dem ALTEN Prefix anfing (sonst würden manuell eingegebene
  // Zählpunkte überschrieben). Da wir die alte Direction hier nicht
  // tracken, ist das Heuristik: leerer Wert ⇒ prefill mit neuem Prefix.
  useEffect(() => {
    const current = form.getValues(`meteringPoints.${index}.meteringPoint`) ?? "";
    if (current === "" && activePrefix) {
      form.setValue(`meteringPoints.${index}.meteringPoint`, activePrefix, {
        shouldValidate: false,
      });
    }
    // Direction is the trigger; activePrefix mit drin damit Stale-Closure-
    // Warnungen ausbleiben. form.* ist stabil aus useForm.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [direction, activePrefix]);

  return (
    <div className="border border-border rounded-md p-3 space-y-3">
      {/* Zeile 1: Richtung + Faktor + Trash. Richtung muss VOR der
          Zählpunkt-Eingabe stehen, weil sie die Mask bestimmt. */}
      <div className="flex gap-2 items-end">
        <div className="w-40 shrink-0">
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.direction`}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Richtung</FormLabel>
                <Select
                  onValueChange={(v) => {
                    // PROJ-52: Direction-Wechsel cleart das Zählpunkt-Feld,
                    // damit der useEffect oben mit dem neuen Prefix neu
                    // prefillen kann (sonst bliebe der alte Wert mit dem
                    // alten Prefix stehen und das Mitglied müsste manuell
                    // korrigieren).
                    field.onChange(v);
                    form.setValue(`meteringPoints.${index}.meteringPoint`, "", {
                      shouldValidate: false,
                    });
                  }}
                  value={field.value}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value="CONSUMPTION">Verbraucher</SelectItem>
                    <SelectItem value="PRODUCTION">Erzeuger</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <div className="w-28">
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.participationFactor`}
            render={({ field }) => (
              <FormItem>
                <div className="flex items-center gap-1">
                  <FormLabel>Faktor</FormLabel>
                  <Popover>
                    <PopoverTrigger type="button" className="cursor-help">
                      <Info className="h-3.5 w-3.5 text-muted-foreground" />
                    </PopoverTrigger>
                    <PopoverContent className="max-w-60 text-sm">
                      Der Teilnahmefaktor gibt an, mit welchem prozentualen Anteil dieser Zählpunkt an der Energiegemeinschaft teilnimmt. Standardmäßig 100 %.
                    </PopoverContent>
                  </Popover>
                </div>
                <FormControl>
                  <div className="relative">
                    <Input
                      type="number"
                      min={1}
                      max={100}
                      className="pr-7"
                      value={field.value}
                      onChange={(e) =>
                        field.onChange(
                          isNaN(e.target.valueAsNumber) ? 100 : e.target.valueAsNumber,
                        )
                      }
                      onBlur={field.onBlur}
                      name={field.name}
                      ref={field.ref}
                    />
                    <span className="absolute right-2.5 top-1/2 -translate-y-1/2 text-sm text-muted-foreground pointer-events-none">
                      %
                    </span>
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={onRemove}
          disabled={!canRemove}
          aria-label={`Zählpunkt ${index + 1} entfernen`}
          className="shrink-0 mb-0.5 ml-auto"
        >
          <Trash2 className="h-4 w-4 text-destructive" />
        </Button>
      </div>

      {/* Zeile 2: Zählpunkt full-width (33-stellige Mask, dynamisch). */}
      <FormField
        control={form.control}
        name={`meteringPoints.${index}.meteringPoint`}
        render={({ field }) => (
          <FormItem>
            <div className="flex items-center gap-1">
              <FormLabel>Zählpunkt {index + 1}</FormLabel>
              <Popover>
                <PopoverTrigger type="button" className="cursor-help">
                  <Info className="h-3.5 w-3.5 text-muted-foreground" />
                </PopoverTrigger>
                <PopoverContent className="max-w-72 text-sm">
                  Die 33-stellige Zählpunktnummer (beginnt mit „AT") identifiziert Ihren Stromanschluss eindeutig. Sie finden sie auf jeder Stromrechnung sowie im Kundenportal Ihres Netzbetreibers. Wenn die EEG einen Prefix konfiguriert hat, ist dieser bereits vorbelegt — Sie tippen nur noch die restlichen Stellen.
                </PopoverContent>
              </Popover>
            </div>
            <FormControl>
              <MaskedInput
                // PROJ-52: dynamische Mask — Prefix-Stellen sind literal
                // gelockt (Mitglied kann sie nicht überschreiben), Reststellen
                // sind Placeholder (`0` für Stellen 3–13, `S` für 14–33).
                // Offizielle Gruppierung 2-6-5-20.
                mask={dynamicMask}
                definitions={MP_MASK_DEFINITIONS}
                lazy={false}
                prepareChar={(str: string) => str.toUpperCase()}
                value={field.value}
                onAccept={(value: string) => field.onChange(value)}
                onBlur={(e: React.FocusEvent<HTMLInputElement>) => {
                  // PROJ-52: Auto-Pad mit führenden Nullen zwischen Prefix
                  // (oder reinem "AT") und Mitglieds-Anteil. Greift nur,
                  // wenn Eingabe < 33 Stellen ist und mit AT beginnt.
                  const padded = padToMeteringPointLength(field.value ?? "", activePrefix);
                  if (padded !== (field.value ?? "").replace(/\s/g, "").toUpperCase()) {
                    field.onChange(padded);
                  }
                  field.onBlur();
                }}
                inputRef={field.ref}
                name={field.name}
                className="font-mono text-sm tabular-nums tracking-tight"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />

      <DeviatingAddressBlock form={form} index={index} />

      <GenerationBlock
        form={form}
        index={index}
        showFeedInForecast={showFeedInForecast}
        feedInForecastRequired={requiredOf("feed_in_forecast") === "required"}
        showPvPower={showPvPower}
        pvPowerRequired={requiredOf("pv_power_kwp") === "required"}
        showFeedInLimit={showFeedInLimit}
        feedInLimitRequired={requiredOf("feed_in_limit_kw") === "required"}
      />

      <BatteryBlock
        form={form}
        index={index}
        showBatterySize={showBatterySize}
        showInverter={showInverter}
        showControl={showBatteryControl}
        batteryRequired={requiredOf("battery_size_kwh") === "required"}
        inverterRequired={requiredOf("inverter_manufacturer") === "required"}
        controlRequired={requiredOf("battery_control_acceptable") === "required"}
      />

      <ConsumptionDetailsBlock
        form={form}
        index={index}
        showPrev={showConsumptionPrev}
        showFc={showConsumptionFc}
        prevRequired={requiredOf("consumption_previous_year") === "required"}
        fcRequired={requiredOf("consumption_forecast") === "required"}
      />

      {hasExtraMpFields && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-1">
          {showTransformer && (
            <FormField
              control={form.control}
              name={`meteringPoints.${index}.transformer`}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Transformator{requiredOf("transformer") === "required" ? " *" : ""}
                  </FormLabel>
                  <FormControl>
                    <Input {...field} value={field.value ?? ""} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
          {showInstallationNumber && (
            <FormField
              control={form.control}
              name={`meteringPoints.${index}.installationNumber`}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Anlagen-Nr.{requiredOf("installation_number") === "required" ? " *" : ""}
                  </FormLabel>
                  <FormControl>
                    <Input {...field} value={field.value ?? ""} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
          {showInstallationName && (
            <FormField
              control={form.control}
              name={`meteringPoints.${index}.installationName`}
              render={({ field }) => (
                <FormItem>
                  <div className="flex items-center gap-1">
                    <FormLabel>
                      Anlagenname{requiredOf("installation_name") === "required" ? " *" : ""}
                    </FormLabel>
                    <Popover>
                      <PopoverTrigger type="button" className="cursor-help">
                        <Info className="h-3.5 w-3.5 text-muted-foreground" />
                      </PopoverTrigger>
                      <PopoverContent className="max-w-60 text-sm">
                        Der Anlagenname ist eine Bezeichnung des Zählpunkts und wird auch auf der Rechnung ausgewiesen, z. B. Hauptanlage, Nebengebäude, Warmwasser, …
                      </PopoverContent>
                    </Popover>
                  </div>
                  <FormControl>
                    <Input {...field} value={field.value ?? ""} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
        </div>
      )}
    </div>
  );
}

// GenerationBlock renders the PROJ-45 fields per metering point:
//   - generation_type Select (only for PRODUCTION; Pflicht, Default 'pv')
//   - pv_power_kwp + feed_in_forecast inline (PROJ-49)
//   - feed_in_limit toggle + kW input (PROJ-49)
// Battery + Wechselrichter + Speichersteuerung leben im separaten
// BatteryBlock (Master-Checkbox).
function GenerationBlock({
  form,
  index,
  showFeedInForecast,
  feedInForecastRequired,
  showPvPower,
  pvPowerRequired,
  showFeedInLimit,
  feedInLimitRequired,
}: {
  form: UseFormReturn<RegistrationFormValues>;
  index: number;
  showFeedInForecast: boolean;
  feedInForecastRequired: boolean;
  showPvPower: boolean;
  pvPowerRequired: boolean;
  showFeedInLimit: boolean;
  feedInLimitRequired: boolean;
}) {
  const direction = form.watch(`meteringPoints.${index}.direction`);
  const generationType = form.watch(`meteringPoints.${index}.generationType`);
  const feedInLimitPresent = form.watch(`meteringPoints.${index}.feedInLimitPresent`);
  if (direction !== "PRODUCTION") return null;
  const isPv = (generationType ?? "pv") === "pv";
  return (
    <div className="pt-1 space-y-3">
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <FormField
          control={form.control}
          name={`meteringPoints.${index}.generationType`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Erzeugungsform *</FormLabel>
              <Select
                value={field.value ?? "pv"}
                onValueChange={(v) => field.onChange(v)}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {GENERATION_TYPES.map((g) => (
                    <SelectItem key={g.value} value={g.value}>{g.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
        {isPv && showPvPower && (
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.pvPowerKwp`}
            render={({ field }) => (
              <FormItem>
                <FormLabel>PV-Leistung (kWp){pvPowerRequired ? " *" : ""}</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    min={0}
                    step={0.1}
                    value={field.value ?? ""}
                    onChange={(e) => {
                      const v = e.target.valueAsNumber;
                      field.onChange(isNaN(v) ? undefined : v);
                    }}
                    onBlur={field.onBlur}
                    name={field.name}
                    ref={field.ref}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        )}
        {showFeedInForecast && (
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.feedInForecast`}
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Einspeisung Prognose (kWh){feedInForecastRequired ? " *" : ""}
                </FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    min={0}
                    value={field.value ?? ""}
                    onChange={(e) => {
                      const v = e.target.valueAsNumber;
                      field.onChange(isNaN(v) ? undefined : v);
                    }}
                    onBlur={field.onBlur}
                    name={field.name}
                    ref={field.ref}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        )}
        {/* Batterie + Wechselrichter + Speichersteuerung wandern in den
            BatteryBlock (Master-Checkbox) — siehe weiter unten. */}
      </div>
      {/* PROJ-49: Einspeiselimit — Mitglied gibt zuerst ja/nein an; bei ja
          erscheint das kW-Feld. Nur bei PV. */}
      {isPv && showFeedInLimit && (
        <div className="space-y-2">
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.feedInLimitPresent`}
            render={({ field }) => (
              <FormItem className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                  <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
                    <Checkbox
                      checked={field.value === true}
                      onCheckedChange={(v) => {
                        const next = v === true;
                        field.onChange(next);
                        if (!next) {
                          form.setValue(
                            `meteringPoints.${index}.feedInLimitKw`,
                            undefined,
                            { shouldValidate: true },
                          );
                        }
                      }}
                      aria-label="Einspeiselimit vorhanden"
                    />
                    <span>Einspeiselimit vorhanden</span>
                  </label>
                  <Popover>
                    <PopoverTrigger type="button" className="cursor-help">
                      <Info className="h-3.5 w-3.5 text-muted-foreground" />
                    </PopoverTrigger>
                    <PopoverContent className="max-w-72 text-sm">
                      Manche Netzanschlüsse sind leistungstechnisch beschränkt — es darf nur ein Teil der erzeugten PV-Leistung in das Netz eingespeist werden. Bei Ja den maximal zulässigen Wert in kW eintragen.
                    </PopoverContent>
                  </Popover>
                </div>
                <FormMessage />
              </FormItem>
            )}
          />
          {feedInLimitPresent === true && (
            <FormField
              control={form.control}
              name={`meteringPoints.${index}.feedInLimitKw`}
              render={({ field }) => (
                <FormItem className="max-w-xs">
                  <FormLabel>
                    Einspeiselimit (kW){feedInLimitRequired ? " *" : ""}
                  </FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={0}
                      step={0.1}
                      value={field.value ?? ""}
                      onChange={(e) => {
                        const v = e.target.valueAsNumber;
                        field.onChange(isNaN(v) ? undefined : v);
                      }}
                      onBlur={field.onBlur}
                      name={field.name}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
        </div>
      )}
    </div>
  );
}

// ConsumptionDetailsBlock renders the PROJ-49 per-MP energy inputs for
// CONSUMPTION rows (Verbrauch Vorjahr + Verbrauch Prognose). Hidden for
// PRODUCTION and when the EEG has both fields configured as `hidden`.
function ConsumptionDetailsBlock({
  form,
  index,
  showPrev,
  showFc,
  prevRequired,
  fcRequired,
}: {
  form: UseFormReturn<RegistrationFormValues>;
  index: number;
  showPrev: boolean;
  showFc: boolean;
  prevRequired: boolean;
  fcRequired: boolean;
}) {
  const direction = form.watch(`meteringPoints.${index}.direction`);
  if (direction !== "CONSUMPTION") return null;
  if (!showPrev && !showFc) return null;
  return (
    <div className="pt-1 grid grid-cols-1 sm:grid-cols-2 gap-3">
      {showPrev && (
        <FormField
          control={form.control}
          name={`meteringPoints.${index}.consumptionPreviousYear`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Verbrauch Vorjahr (kWh){prevRequired ? " *" : ""}</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  min={0}
                  value={field.value ?? ""}
                  onChange={(e) => {
                    const v = e.target.valueAsNumber;
                    field.onChange(isNaN(v) ? undefined : v);
                  }}
                  onBlur={field.onBlur}
                  name={field.name}
                  ref={field.ref}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
      {showFc && (
        <FormField
          control={form.control}
          name={`meteringPoints.${index}.consumptionForecast`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Verbrauch Prognose (kWh){fcRequired ? " *" : ""}</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  min={0}
                  value={field.value ?? ""}
                  onChange={(e) => {
                    const v = e.target.valueAsNumber;
                    field.onChange(isNaN(v) ? undefined : v);
                  }}
                  onBlur={field.onBlur}
                  name={field.name}
                  ref={field.ref}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
    </div>
  );
}

// DeviatingAddressBlock renders the "abweichende Adresse" checkbox + the
// four address inputs for a single metering point. Checkbox state is
// local-only (PROJ-39 — the spec says "Der Wert der checkbox muss nicht
// gespeichert werden"); on mount it's derived from whether any address
// field is already filled (which can happen when the user re-opens an
// existing draft). Unchecking clears all four fields so a fresh submit
// reverts to the member's primary address.
function DeviatingAddressBlock({
  form,
  index,
}: {
  form: UseFormReturn<RegistrationFormValues>;
  index: number;
}) {
  const street = form.watch(`meteringPoints.${index}.addressStreet`);
  const streetNumber = form.watch(`meteringPoints.${index}.addressStreetNumber`);
  const zip = form.watch(`meteringPoints.${index}.addressZip`);
  const city = form.watch(`meteringPoints.${index}.addressCity`);
  const anyFilled = !!(street || streetNumber || zip || city);
  const [enabled, setEnabled] = useState<boolean>(anyFilled);

  useEffect(() => {
    if (anyFilled && !enabled) setEnabled(true);
  }, [anyFilled, enabled]);

  function toggle(next: boolean) {
    setEnabled(next);
    if (!next) {
      form.setValue(`meteringPoints.${index}.addressStreet`, "", { shouldValidate: true });
      form.setValue(`meteringPoints.${index}.addressStreetNumber`, "", { shouldValidate: true });
      form.setValue(`meteringPoints.${index}.addressZip`, "", { shouldValidate: true });
      form.setValue(`meteringPoints.${index}.addressCity`, "", { shouldValidate: true });
    }
  }

  return (
    <div className="pt-1 space-y-2">
      <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
        <Checkbox
          checked={enabled}
          onCheckedChange={(v) => toggle(v === true)}
          aria-label="Abweichende Adresse für diesen Zählpunkt"
        />
        <span>Abweichende Adresse für diesen Zählpunkt</span>
      </label>
      {enabled && (
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.addressStreet`}
            render={({ field }) => (
              <FormItem className="sm:col-span-2">
                <FormLabel>Straße *</FormLabel>
                <FormControl>
                  <Input {...field} value={field.value ?? ""} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.addressStreetNumber`}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Hausnr. *</FormLabel>
                <FormControl>
                  <Input {...field} value={field.value ?? ""} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.addressZip`}
            render={({ field }) => (
              <FormItem>
                <FormLabel>PLZ *</FormLabel>
                <FormControl>
                  <Input {...field} value={field.value ?? ""} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`meteringPoints.${index}.addressCity`}
            render={({ field }) => (
              <FormItem className="sm:col-span-3">
                <FormLabel>Ort *</FormLabel>
                <FormControl>
                  <Input {...field} value={field.value ?? ""} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      )}
    </div>
  );
}

// BatteryBlock — Master-Checkbox „Batteriespeicher vorhanden" mit drei
// gruppierten Feldern darunter (Größe Batterie, Hersteller Wechselrichter,
// Speichersteuerung im Sinne der EEG vorstellbar?). Analog zum
// DeviatingAddressBlock-Pattern: Checkbox-State ist lokal, beim Mount aus
// dem Vorhandensein einer der drei Werte abgeleitet; beim Deaktivieren
// werden alle drei Felder gecleared. Nur bei PRODUCTION + PV sichtbar
// UND die EEG hat mindestens eines der drei Felder konfiguriert.
function BatteryBlock({
  form,
  index,
  showBatterySize,
  showInverter,
  showControl,
  batteryRequired,
  inverterRequired,
  controlRequired,
}: {
  form: UseFormReturn<RegistrationFormValues>;
  index: number;
  showBatterySize: boolean;
  showInverter: boolean;
  showControl: boolean;
  batteryRequired: boolean;
  inverterRequired: boolean;
  controlRequired: boolean;
}) {
  const direction = form.watch(`meteringPoints.${index}.direction`);
  const generationType = form.watch(`meteringPoints.${index}.generationType`);
  const batterySizeKwh = form.watch(`meteringPoints.${index}.batterySizeKwh`);
  const inverterManufacturer = form.watch(`meteringPoints.${index}.inverterManufacturer`);
  const batteryControlAcceptable = form.watch(`meteringPoints.${index}.batteryControlAcceptable`);
  const anyFilled =
    batterySizeKwh !== undefined ||
    !!(inverterManufacturer && inverterManufacturer.trim().length > 0) ||
    batteryControlAcceptable !== undefined;
  const [enabled, setEnabled] = useState<boolean>(anyFilled);

  useEffect(() => {
    if (anyFilled && !enabled) setEnabled(true);
  }, [anyFilled, enabled]);

  const isPv = direction === "PRODUCTION" && (generationType ?? "pv") === "pv";
  if (!isPv) return null;
  if (!showBatterySize && !showInverter && !showControl) return null;

  function toggle(next: boolean) {
    setEnabled(next);
    if (!next) {
      form.setValue(`meteringPoints.${index}.batterySizeKwh`, undefined, { shouldValidate: true });
      form.setValue(`meteringPoints.${index}.inverterManufacturer`, "", { shouldValidate: true });
      form.setValue(`meteringPoints.${index}.batteryControlAcceptable`, undefined, { shouldValidate: true });
    }
  }

  return (
    <div className="pt-1 space-y-2">
      <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
        <Checkbox
          checked={enabled}
          onCheckedChange={(v) => toggle(v === true)}
          aria-label="Batteriespeicher vorhanden"
        />
        <span>Batteriespeicher vorhanden</span>
      </label>
      {enabled && (
        <div className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {showBatterySize && (
              <FormField
                control={form.control}
                name={`meteringPoints.${index}.batterySizeKwh`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Größe Batterie (kWh){batteryRequired ? " *" : ""}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        step={0.1}
                        value={field.value ?? ""}
                        onChange={(e) => {
                          const v = e.target.valueAsNumber;
                          field.onChange(isNaN(v) ? undefined : v);
                        }}
                        onBlur={field.onBlur}
                        name={field.name}
                        ref={field.ref}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            {showInverter && (
              <FormField
                control={form.control}
                name={`meteringPoints.${index}.inverterManufacturer`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Hersteller Wechselrichter{inverterRequired ? " *" : ""}
                    </FormLabel>
                    <FormControl>
                      <Input {...field} value={field.value ?? ""} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </div>
          {showControl && (
            <FormField
              control={form.control}
              name={`meteringPoints.${index}.batteryControlAcceptable`}
              render={({ field }) => (
                <FormItem className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
                      <Checkbox
                        checked={field.value === true}
                        onCheckedChange={(v) => field.onChange(v === true)}
                        aria-label="Speichersteuerung im Sinne der EEG vorstellbar"
                      />
                      <span>
                        Speichersteuerung im Sinne der EEG vorstellbar?
                        {controlRequired ? " *" : ""}
                      </span>
                    </label>
                    <Popover>
                      <PopoverTrigger type="button" className="cursor-help">
                        <Info className="h-3.5 w-3.5 text-muted-foreground" />
                      </PopoverTrigger>
                      <PopoverContent className="max-w-72 text-sm">
                        Die EEG könnte Ihren Heimspeicher gemeinsam mit den Speichern anderer Mitglieder so steuern, dass die Erzeugung innerhalb der Gemeinschaft optimal genutzt wird. Das Häkchen ist nur Ihr Einverständnis im Sinne der EEG — eine konkrete Steuerung wird separat mit Ihnen abgestimmt.
                      </PopoverContent>
                    </Popover>
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
        </div>
      )}
    </div>
  );
}
