"use client";

import { useFieldArray, type UseFormReturn } from "react-hook-form";
import { PlusCircle, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
        <div key={field.id} className="flex gap-3 items-start">
          <div className="flex-1">
            <FormField
              control={form.control}
              name={`meteringPoints.${index}.meteringPoint`}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Zählpunkt {index + 1}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="AT0031000000000000000000990022105"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className="w-44">
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
                      <SelectItem value="CONSUMPTION">Verbrauch</SelectItem>
                      <SelectItem value="PRODUCTION">Einspeisung</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className="pt-8">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => remove(index)}
              disabled={fields.length === 1}
              aria-label={`Zählpunkt ${index + 1} entfernen`}
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
          onClick={() => append({ meteringPoint: "", direction: "CONSUMPTION" })}
        >
          <PlusCircle className="h-4 w-4 mr-2" />
          Zählpunkt hinzufügen
        </Button>
      )}
    </div>
  );
}
