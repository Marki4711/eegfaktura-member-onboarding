"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { AlertCircle, CheckCircle2 } from "lucide-react";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MeteringPointFields } from "./metering-point-fields";
import { MaskedInput } from "@/components/ui/masked-input";
import { isValidIBAN } from "ibantools";
import {
  createApplication,
  submitApplication,
  ApiResponseError,
  type RegistrationConfig,
  type MemberType,
} from "@/lib/api";

// Hardcoded for MVP — matches backend default
const PRIVACY_VERSION = "2026-01";

const MEMBER_TYPE_OPTIONS: { value: MemberType; label: string; hint: string }[] = [
  { value: "private",      label: "Privatperson / Kleinunternehmer", hint: "0 % USt." },
  { value: "farmer",       label: "Pauschalierter Landwirt",   hint: "13 % USt." },
  { value: "municipality", label: "Gemeinde / öffentl. Körperschaft", hint: "variabel" },
  { value: "company",      label: "Unternehmen",               hint: "20 % USt." },
  { value: "association",  label: "Verein",                    hint: "variabel" },
];

// ---------- Zod schema ----------

const meteringPointSchema = z.object({
  meteringPoint: z
    .string()
    .transform((v) => v.replace(/\s/g, ""))
    .refine((v) => v.length >= 1, { message: "Zählpunkt ist erforderlich" })
    .refine((v) => v.length <= 33, { message: "Maximal 33 Zeichen" }),
  direction: z.enum(["CONSUMPTION", "PRODUCTION"]),
});

const formSchema = z
  .object({
    memberType: z.enum(["private", "farmer", "municipality", "company", "association"] as const),
    firstname: z.string().trim().max(255).optional(),
    lastname: z.string().trim().max(255).optional(),
    birthDate: z.string().optional(),
    companyName: z.string().trim().max(255).optional(),
    uidNumber: z.string().trim().max(50).optional(),
    registerNumber: z.string().trim().max(50).optional(),
    email: z.string().trim().email("Ungültige E-Mail-Adresse"),
    phone: z.string().trim().optional(),
    residentStreet: z.string().trim().min(1, "Straße ist erforderlich").max(255),
    residentStreetNumber: z.string().trim().min(1, "Hausnummer ist erforderlich").max(50),
    residentZip: z.string().trim().min(1, "PLZ ist erforderlich").max(20),
    residentCity: z.string().trim().min(1, "Ort ist erforderlich").max(255),
    iban: z
      .string()
      .min(1, "IBAN ist erforderlich")
      .transform((v) => v.replace(/\s/g, "").toUpperCase())
      .refine((v) => isValidIBAN(v), { message: "Ungültige IBAN" }),
    accountHolder: z.string().trim().min(1, "Kontoinhaber:in ist erforderlich").max(255),
    privacyAccepted: z.boolean().refine((v) => v === true, {
      message: "Datenschutzerklärung muss akzeptiert werden",
    }),
    accuracyConfirmed: z.boolean().refine((v) => v === true, {
      message: "Richtigkeit der Angaben muss bestätigt werden",
    }),
    sepaMandateAccepted: z.boolean().refine((v) => v === true, {
      message: "Zustimmung zum SEPA-Lastschriftmandat ist erforderlich",
    }),
    meteringPoints: z
      .array(meteringPointSchema)
      .min(1, "Mindestens ein Zählpunkt ist erforderlich")
      .max(10, "Maximal 10 Zählpunkte erlaubt"),
  })
  .superRefine((data, ctx) => {
    const isPerson = data.memberType === "private" || data.memberType === "farmer";
    if (isPerson) {
      if (!data.firstname?.trim()) {
        ctx.addIssue({ code: "custom", path: ["firstname"], message: "Vorname ist erforderlich" });
      }
      if (!data.lastname?.trim()) {
        ctx.addIssue({ code: "custom", path: ["lastname"], message: "Nachname ist erforderlich" });
      }
      if (!data.birthDate) {
        ctx.addIssue({ code: "custom", path: ["birthDate"], message: "Geburtsdatum ist erforderlich" });
      }
    } else {
      const orgLabel = data.memberType === "municipality" ? "Organisationsname"
        : data.memberType === "association" ? "Vereinsname"
        : "Firmenname";
      if (!data.companyName?.trim()) {
        ctx.addIssue({ code: "custom", path: ["companyName"], message: `${orgLabel} ist erforderlich` });
      }
      if (data.memberType === "company") {
        if (!data.uidNumber?.trim()) {
          ctx.addIssue({ code: "custom", path: ["uidNumber"], message: "UID-Nummer ist erforderlich" });
        }
        if (!data.registerNumber?.trim()) {
          ctx.addIssue({ code: "custom", path: ["registerNumber"], message: "Firmenbuchnummer ist erforderlich" });
        }
      }
      if (data.memberType === "association") {
        if (!data.registerNumber?.trim()) {
          ctx.addIssue({ code: "custom", path: ["registerNumber"], message: "Vereinsnummer ist erforderlich" });
        }
      }
    }
  });

export type RegistrationFormValues = z.infer<typeof formSchema>;

// ---------- component ----------

interface SuccessState {
  referenceNumber: string;
  submittedAt: string;
}

interface RegistrationFormProps {
  config: RegistrationConfig;
}

export function RegistrationForm({ config }: RegistrationFormProps) {
  const [success, setSuccess] = useState<SuccessState | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<RegistrationFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      memberType: "private",
      firstname: "",
      lastname: "",
      birthDate: "",
      companyName: "",
      uidNumber: "",
      registerNumber: "",
      email: "",
      phone: "",
      residentStreet: "",
      residentStreetNumber: "",
      residentZip: "",
      residentCity: "",
      iban: "",
      accountHolder: "",
      privacyAccepted: false,
      accuracyConfirmed: false,
      sepaMandateAccepted: false,
      meteringPoints: [{ meteringPoint: "", direction: "CONSUMPTION" }],
    },
  });

  const memberType = form.watch("memberType");
  const isPerson = memberType === "private" || memberType === "farmer";

  function onMemberTypeChange(value: MemberType) {
    form.setValue("memberType", value);
    // Clear fields from the other group so stale data is not submitted
    if (value === "private" || value === "farmer") {
      form.setValue("companyName", "");
      form.setValue("uidNumber", "");
      form.setValue("registerNumber", "");
      form.clearErrors(["companyName", "uidNumber", "registerNumber"]);
    } else {
      form.setValue("firstname", "");
      form.setValue("lastname", "");
      form.setValue("birthDate", "");
      form.clearErrors(["firstname", "lastname", "birthDate"]);
    }
  }

  async function onSubmit(values: RegistrationFormValues) {
    setIsSubmitting(true);
    setApiError(null);

    const isPersonType = values.memberType === "private" || values.memberType === "farmer";

    try {
      const app = await createApplication({
        rcNumber: config.rcNumber,
        memberType: values.memberType,
        firstname: isPersonType ? values.firstname || undefined : undefined,
        lastname: isPersonType ? values.lastname || undefined : undefined,
        birthDate: isPersonType ? values.birthDate || undefined : undefined,
        companyName: !isPersonType ? values.companyName || undefined : undefined,
        uidNumber: !isPersonType ? values.uidNumber || undefined : undefined,
        registerNumber: !isPersonType ? values.registerNumber || undefined : undefined,
        email: values.email,
        phone: values.phone || undefined,
        residentStreet: values.residentStreet,
        residentStreetNumber: values.residentStreetNumber,
        residentZip: values.residentZip,
        residentCity: values.residentCity,
        privacyAccepted: values.privacyAccepted,
        privacyVersion: PRIVACY_VERSION,
        accuracyConfirmed: values.accuracyConfirmed,
        iban: values.iban,
        accountHolder: values.accountHolder,
        sepaMandateAccepted: values.sepaMandateAccepted,
        meteringPoints: values.meteringPoints,
      });

      const submitted = await submitApplication(app.id);

      setSuccess({
        referenceNumber: submitted.referenceNumber,
        submittedAt: submitted.submittedAt,
      });
    } catch (err) {
      if (err instanceof ApiResponseError) {
        const { code, message, fields } = err.apiError;

        if (fields) {
          const knownFields = Object.keys(form.getValues()) as Array<
            keyof RegistrationFormValues
          >;
          const unmapped: string[] = [];

          for (const [key, msg] of Object.entries(fields)) {
            if (knownFields.includes(key as keyof RegistrationFormValues)) {
              form.setError(key as keyof RegistrationFormValues, {
                type: "server",
                message: msg,
              });
            } else {
              unmapped.push(msg);
            }
          }

          if (unmapped.length > 0) {
            setApiError(unmapped.join(" "));
          }
        } else if (code === "not_found") {
          setApiError("Die RC-Nummer wurde nicht gefunden.");
        } else if (code === "gone") {
          setApiError("Die Registrierung ist nicht mehr aktiv.");
        } else {
          setApiError(message || "Ein Fehler ist aufgetreten.");
        }
      } else {
        setApiError(
          "Ein unerwarteter Fehler ist aufgetreten. Bitte versuchen Sie es erneut."
        );
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  // ---------- success state ----------

  if (success) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <CheckCircle2 className="h-7 w-7 text-green-600 shrink-0" />
            <CardTitle>Antrag erfolgreich eingereicht</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-muted-foreground">
            Ihr Antrag wurde übermittelt und wird nun von unserem Team geprüft.
          </p>
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium">Ihre Antragsnummer:</span>
            <Badge variant="secondary" className="font-mono text-sm">
              {success.referenceNumber}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Bitte notieren Sie diese Nummer für eventuelle Rückfragen.
          </p>
        </CardContent>
      </Card>
    );
  }

  // ---------- form ----------

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6" noValidate>
        {apiError && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Fehler</AlertTitle>
            <AlertDescription>{apiError}</AlertDescription>
          </Alert>
        )}

        {/* Member type */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Mitgliedstyp</CardTitle>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="memberType"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Mitgliedstyp *</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={(v) => onMemberTypeChange(v as MemberType)}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Bitte auswählen …" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {MEMBER_TYPE_OPTIONS.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>
                          {opt.label}
                          <span className="ml-2 text-xs text-muted-foreground">({opt.hint})</span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        {/* Member / organisation data */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              {isPerson ? "Persönliche Daten" : "Organisationsdaten"}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {isPerson ? (
              <>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="firstname"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Vorname *</FormLabel>
                        <FormControl>
                          <Input placeholder="Max" autoComplete="given-name" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="lastname"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Nachname *</FormLabel>
                        <FormControl>
                          <Input placeholder="Mustermann" autoComplete="family-name" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="birthDate"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Geburtsdatum *</FormLabel>
                        <FormControl>
                          <Input type="date" autoComplete="bday" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </>
            ) : (
              <>
                <FormField
                  control={form.control}
                  name="companyName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {memberType === "municipality"
                          ? "Organisationsname *"
                          : memberType === "association"
                          ? "Vereinsname *"
                          : "Firmenname *"}
                      </FormLabel>
                      <FormControl>
                        <Input
                          placeholder={
                            memberType === "municipality"
                              ? "Gemeinde Musterort"
                              : memberType === "association"
                              ? "Musterverein"
                              : "Muster GmbH"
                          }
                          autoComplete="organization"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {(memberType === "company" || memberType === "association") && (
                    <FormField
                      control={form.control}
                      name="registerNumber"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {memberType === "association" ? "Vereinsnummer *" : "Firmenbuchnummer *"}
                          </FormLabel>
                          <FormControl>
                            <Input
                              placeholder={memberType === "association" ? "ZVR 123456789" : "FN 123456 a"}
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}
                  <FormField
                    control={form.control}
                    name="uidNumber"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>
                          UID-Nummer{memberType === "company" ? " *" : ""}
                        </FormLabel>
                        <FormControl>
                          <Input placeholder="ATU12345678" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </>
            )}

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>E-Mail *</FormLabel>
                    <FormControl>
                      <Input
                        type="email"
                        placeholder="max@example.at"
                        autoComplete="email"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="phone"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Telefon</FormLabel>
                    <FormControl>
                      <Input
                        type="tel"
                        placeholder="0664 / 1234567"
                        autoComplete="tel"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
        </Card>

        {/* Address */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Adresse</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2">
                <FormField
                  control={form.control}
                  name="residentStreet"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Straße *</FormLabel>
                      <FormControl>
                        <Input placeholder="Musterstraße" autoComplete="address-line1" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="residentStreetNumber"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Nr. *</FormLabel>
                    <FormControl>
                      <Input placeholder="1a" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="grid grid-cols-3 gap-4">
              <FormField
                control={form.control}
                name="residentZip"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>PLZ *</FormLabel>
                    <FormControl>
                      <Input placeholder="4020" autoComplete="postal-code" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="col-span-2">
                <FormField
                  control={form.control}
                  name="residentCity"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Ort *</FormLabel>
                      <FormControl>
                        <Input placeholder="Linz" autoComplete="address-level2" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Bank account / SEPA */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Bankverbindung</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="iban"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>IBAN *</FormLabel>
                    <FormControl>
                      <MaskedInput
                        mask="aa00 0000 0000 0000 0000"
                        lazy={false}
                        prepareChar={(str: string) => str.toUpperCase()}
                        placeholder="AT12 3456 7890 1234 5678"
                        autoComplete="off"
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
              <FormField
                control={form.control}
                name="accountHolder"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Kontoinhaber:in *</FormLabel>
                    <FormControl>
                      <Input placeholder="Max Mustermann" autoComplete="name" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
        </Card>

        {/* Metering points */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Zählpunkte</CardTitle>
          </CardHeader>
          <CardContent>
            <MeteringPointFields form={form} />
            {form.formState.errors.meteringPoints?.message && (
              <p className="text-sm font-medium text-destructive mt-3">
                {form.formState.errors.meteringPoints.message}
              </p>
            )}
          </CardContent>
        </Card>

        {/* Consent */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Einwilligungen</CardTitle>
          </CardHeader>
          <CardContent className="space-y-5">
            <FormField
              control={form.control}
              name="privacyAccepted"
              render={({ field }) => (
                <FormItem className="flex flex-row items-start gap-3 space-y-0">
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <div className="space-y-1 leading-none">
                    <FormLabel className="font-normal cursor-pointer">
                      Ich habe die Datenschutzerklärung gelesen und stimme der
                      Verarbeitung meiner Daten zu. *
                    </FormLabel>
                    <FormMessage />
                  </div>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="accuracyConfirmed"
              render={({ field }) => (
                <FormItem className="flex flex-row items-start gap-3 space-y-0">
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <div className="space-y-1 leading-none">
                    <FormLabel className="font-normal cursor-pointer">
                      Ich bestätige die Richtigkeit meiner Angaben. *
                    </FormLabel>
                    <FormMessage />
                  </div>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="sepaMandateAccepted"
              render={({ field }) => (
                <FormItem className="flex flex-row items-start gap-3 space-y-0">
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <div className="space-y-1 leading-none">
                    <FormLabel className="font-normal cursor-pointer">
                      Ich erteile der Energiegemeinschaft ein SEPA-Lastschriftmandat
                      und stimme dem Einzug fälliger Rechnungsbeträge von meinem
                      angegebenen Konto zu. *
                    </FormLabel>
                    <FormMessage />
                  </div>
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <div>
          <Button
            type="submit"
            size="lg"
            disabled={isSubmitting}
            className="w-full sm:w-auto"
          >
            {isSubmitting ? "Antrag wird eingereicht …" : "Antrag einreichen"}
          </Button>
          <p className="text-xs text-muted-foreground mt-2">
            * Pflichtfelder
          </p>
        </div>
      </form>
    </Form>
  );
}
