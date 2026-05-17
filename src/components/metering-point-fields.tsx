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
}

export function MeteringPointFields({ form, fieldConfig }: MeteringPointFieldsProps) {
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
        <div key={field.id} className="border border-border rounded-md p-3 space-y-3">
          <div className="flex flex-col sm:flex-row gap-3 sm:items-end">
            <div className="w-full sm:flex-1 sm:min-w-0">
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
                          Die 33-stellige Zählpunktnummer (beginnt mit „AT") identifiziert Ihren Stromanschluss eindeutig. Sie finden sie auf jeder Stromrechnung sowie im Kundenportal Ihres Netzbetreibers.
                        </PopoverContent>
                      </Popover>
                    </div>
                    <FormControl>
                      <MaskedInput
                        mask="AT 000000 00000 000000000000 00000000"
                        lazy={false}
                        prepareChar={(str: string) => str.toUpperCase()}
                        value={field.value}
                        onAccept={(value: string) => field.onChange(value)}
                        onBlur={field.onBlur}
                        inputRef={field.ref}
                        name={field.name}
                        // 37 sichtbare Stellen (AT + 31 Ziffern + 4 Spaces) müssen
                        // in eine Zeile passen, ohne Richtung/Faktor zu verdrängen.
                        // Default-Sans ist ~25% schmaler als font-mono; mit
                        // tabular-nums bleiben die Ziffern sauber ausgerichtet.
                        // 2026-05-15 Tester-Feedback (Apple-Geräte): Safari/macOS
                        // rendert San-Francisco-Glyphen ~10% breiter als die
                        // Windows/Android-Defaults — letzte 4-5 Zeichen wurden
                        // abgeschnitten. Daher:
                        //   - text-[11px] statt text-xs (12px → 11px = -8.3%)
                        //   - tracking-[-0.06em] statt tracking-tighter
                        //     (Tailwind-Default -0.05em → -0.06em, etwas enger)
                        //   - px-2 reduziert das Input-Padding gegenüber dem
                        //     Default (px-3) für zusätzliche Innenbreite.
                        className="text-[11px] tabular-nums tracking-[-0.06em] px-2"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="flex gap-2 items-end">
              <div className="w-36 shrink-0">
                <FormField
                  control={form.control}
                  name={`meteringPoints.${index}.direction`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Richtung</FormLabel>
                      <Select
                        onValueChange={field.onChange}
                        defaultValue={field.value}
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

              <div className="w-24">
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
                            onChange={(e) => field.onChange(isNaN(e.target.valueAsNumber) ? 100 : e.target.valueAsNumber)}
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
                onClick={() => remove(index)}
                disabled={fields.length === 1}
                aria-label={`Zählpunkt ${index + 1} entfernen`}
                className="shrink-0 mb-0.5"
              >
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </div>
          </div>

          <DeviatingAddressBlock form={form} index={index} />

          <GenerationBlock
            form={form}
            index={index}
            showFeedInForecast={showFeedInForecast}
            feedInForecastRequired={mpfs("feed_in_forecast") === "required"}
            showPvPower={showPvPower}
            pvPowerRequired={mpfs("pv_power_kwp") === "required"}
            showFeedInLimit={showFeedInLimit}
            feedInLimitRequired={mpfs("feed_in_limit_kw") === "required"}
          />

          <BatteryBlock
            form={form}
            index={index}
            showBatterySize={showBatterySize}
            showInverter={showInverter}
            showControl={showBatteryControl}
            batteryRequired={mpfs("battery_size_kwh") === "required"}
            inverterRequired={mpfs("inverter_manufacturer") === "required"}
            controlRequired={mpfs("battery_control_acceptable") === "required"}
          />

          <ConsumptionDetailsBlock
            form={form}
            index={index}
            showPrev={showConsumptionPrev}
            showFc={showConsumptionFc}
            prevRequired={mpfs("consumption_previous_year") === "required"}
            fcRequired={mpfs("consumption_forecast") === "required"}
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
                        Transformator{mpfs("transformer") === "required" ? " *" : ""}
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
                        Anlagen-Nr.{mpfs("installation_number") === "required" ? " *" : ""}
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
                          Anlagenname{mpfs("installation_name") === "required" ? " *" : ""}
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
