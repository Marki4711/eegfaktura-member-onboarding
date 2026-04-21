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
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { RegistrationFormValues } from "./registration-form";

interface MeteringPointFieldsProps {
  form: UseFormReturn<RegistrationFormValues>;
}

export function MeteringPointFields({ form }: MeteringPointFieldsProps) {
  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "meteringPoints",
  });

  return (
    <div className="space-y-4">
      {fields.map((field, index) => (
        <div key={field.id} className="flex flex-col sm:flex-row gap-3 sm:items-start">
          <div className="flex-1 min-w-0">
            <FormField
              control={form.control}
              name={`meteringPoints.${index}.meteringPoint`}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Zählpunkt {index + 1}</FormLabel>
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
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className="flex gap-2 items-end">
            <div className="flex-1 sm:w-44">
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
                    <FormLabel className="flex items-center gap-1">
                      Faktor
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger type="button" className="cursor-help">
                            <Info className="h-3.5 w-3.5 text-muted-foreground" />
                          </TooltipTrigger>
                          <TooltipContent className="max-w-60">
                            Der Teilnahmefaktor gibt an, mit welchem prozentualen Anteil dieser Zählpunkt an der Energiegemeinschaft teilnimmt. Standardmäßig 100 %.
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormLabel>
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
