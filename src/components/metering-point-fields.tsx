"use client";

import { useFieldArray, type UseFormReturn } from "react-hook-form";
import { Info, PlusCircle, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
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
import { resolveFieldState, CONFIGURABLE_FIELDS, type FieldConfig } from "@/lib/api";
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
                        // 37 sichtbare Stellen (AT + 31 Ziffern + 4 Spaces) –
                        // mit text-xs + font-mono + tracking-tight passen sie
                        // sowohl am Handy als auch am Desktop in eine Zeile,
                        // ohne dass Richtung/Faktor in eine zweite Zeile rutschen.
                        className="text-xs font-mono tracking-tight"
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
                            <SelectValue placeholder="Richtung" />
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
                        <Input placeholder="ANL-12345" {...field} value={field.value ?? ""} />
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
                      <FormLabel>
                        Anlagenname{mpfs("installation_name") === "required" ? " *" : ""}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder="Hauptanlage" {...field} value={field.value ?? ""} />
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
          onClick={() => append({ meteringPoint: "", direction: "CONSUMPTION", participationFactor: 100 })}
        >
          <PlusCircle className="h-4 w-4 mr-2" />
          Zählpunkt hinzufügen
        </Button>
      )}
    </div>
  );
}
