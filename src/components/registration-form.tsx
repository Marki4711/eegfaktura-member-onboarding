"use client";

import { useRef, useState } from "react";
import { Turnstile, type TurnstileInstance } from "@marsidev/react-turnstile";
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
import { IntroTextDisplay } from "./intro-text-display";
import { isValidIBAN } from "ibantools";
import {
  createApplication,
  submitApplication,
  ApiResponseError,
  resolveFieldState,
  CONFIGURABLE_FIELDS,
  type RegistrationConfig,
  type MemberType,
  type FieldConfig,
  type FieldState,
  type ConsentInput,
} from "@/lib/api";

// Hardcoded for MVP — matches backend default
const PRIVACY_VERSION = "2026-01";

const TURNSTILE_SITE_KEY = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY ?? "";

const MEMBER_TYPE_OPTIONS: { value: MemberType; label: string; hint: string }[] = [
  { value: "private",         label: "Privatperson",                    hint: "0 % USt." },
  { value: "sole_proprietor", label: "Kleinunternehmer",                hint: "0 % USt." },
  { value: "farmer",          label: "Pauschalierter Landwirt",         hint: "13 % USt." },
  { value: "municipality",    label: "Gemeinde / öffentl. Körperschaft", hint: "variabel" },
  { value: "company",         label: "Unternehmen",                     hint: "20 % USt." },
  { value: "association",     label: "Verein",                          hint: "variabel" },
];

// ---------- Zod schema ----------

const meteringPointSchema = z.object({
  meteringPoint: z
    .string()
    .transform((v) => v.replace(/\s/g, ""))
    .refine((v) => v.length >= 1, { message: "Zählpunkt ist erforderlich" })
    .refine((v) => v.length <= 33, { message: "Maximal 33 Zeichen" }),
  direction: z.enum(["CONSUMPTION", "PRODUCTION"]),
  participationFactor: z.number().int().min(1, "Mindestens 1%").max(100, "Maximal 100%"),
  transformer: z.string().trim().max(100).optional(),
  installationNumber: z.string().trim().max(50).optional(),
  installationName: z.string().trim().max(100).optional(),
});

const baseSchema = z.object({
  memberType: z.enum(["private", "sole_proprietor", "farmer", "municipality", "company", "association"] as const),
  titel: z.string().trim().max(50).optional(),
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
  sepaMandateAccepted: z.boolean(),
  meteringPoints: z
    .array(meteringPointSchema)
    .min(1, "Mindestens ein Zählpunkt ist erforderlich")
    .max(10, "Maximal 10 Zählpunkte erlaubt"),
  // configurable application-level fields
  membershipStartDate: z.string().optional(),
  personsInHousehold: z.number().int().min(0).optional(),
  consumptionPreviousYear: z.number().int().min(0).optional(),
  consumptionForecast: z.number().int().min(0).optional(),
  feedInForecast: z.number().int().min(0).optional(),
  pvPowerKwp: z.number().min(0).optional(),
  heatPump: z.boolean().nullable().optional(),
  electricVehicle: z.boolean().nullable().optional(),
  electricHotWater: z.boolean().nullable().optional(),
});

export type RegistrationFormValues = z.infer<typeof baseSchema>;

function buildFormSchema(fieldConfig: FieldConfig | undefined, sepaMandateEnabled: boolean) {
  const appFields = CONFIGURABLE_FIELDS.application;
  const resolve = (name: string): FieldState => {
    const f = appFields.find((x) => x.name === name);
    return resolveFieldState(fieldConfig, name, f?.defaultState ?? "hidden");
  };

  return baseSchema.superRefine((data, ctx) => {
    const isPerson = data.memberType === "private" || data.memberType === "farmer";

    // fixed required: person fields
    if (isPerson) {
      if (!data.firstname?.trim()) {
        ctx.addIssue({ code: "custom", path: ["firstname"], message: "Vorname ist erforderlich" });
      }
      if (!data.lastname?.trim()) {
        ctx.addIssue({ code: "custom", path: ["lastname"], message: "Nachname ist erforderlich" });
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

    // configurable required fields
    const requireText = (name: string, path: keyof RegistrationFormValues, label: string) => {
      if (resolve(name) === "required") {
        const v = data[path];
        if (!v && v !== 0) {
          ctx.addIssue({ code: "custom", path: [path], message: `${label} ist erforderlich` });
        }
      }
    };
    const requireNum = (name: string, path: keyof RegistrationFormValues, label: string) => {
      if (resolve(name) === "required") {
        if (data[path] === undefined || data[path] === null) {
          ctx.addIssue({ code: "custom", path: [path], message: `${label} ist erforderlich` });
        }
      }
    };

    requireText("phone", "phone", "Telefonnummer");
    requireText("birth_date", "birthDate", "Geburtsdatum");
    requireText("membership_start_date", "membershipStartDate", "Beitrittsdatum");
    requireNum("persons_in_household", "personsInHousehold", "Anzahl Personen im Haushalt");
    requireNum("consumption_previous_year", "consumptionPreviousYear", "Verbrauch Vorjahr");
    requireNum("consumption_forecast", "consumptionForecast", "Verbrauch Prognose");
    requireNum("feed_in_forecast", "feedInForecast", "Einspeisung Prognose");
    requireNum("pv_power_kwp", "pvPowerKwp", "PV-Leistung");
    requireNum("heat_pump", "heatPump", "Wärmepumpe vorhanden");
    requireNum("electric_vehicle", "electricVehicle", "E-Auto vorhanden");
    requireNum("electric_hot_water", "electricHotWater", "Warmwasser elektrisch");

    // SEPA mandate acceptance only required when not sent by email
    if (!sepaMandateEnabled && !data.sepaMandateAccepted) {
      ctx.addIssue({
        code: "custom",
        path: ["sepaMandateAccepted"],
        message: "Zustimmung zum SEPA-Lastschriftmandat ist erforderlich",
      });
    }
  });
}

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
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
  const turnstileRef = useRef<TurnstileInstance>(null);
  const [docConsents, setDocConsents] = useState<Record<string, boolean>>({});
  const [docConsentErrors, setDocConsentErrors] = useState<Record<string, string>>({});

  const fieldConfig = config.fieldConfig;
  const sepaMandateEnabled = config.sepaMandateEnabled ?? false;
  const showCentralPolicy = config.showCentralPolicy ?? true;
  const legalDocuments = config.legalDocuments ?? [];
  const centralPolicy = showCentralPolicy
    ? legalDocuments.find((d) => d.isCentralPolicy && d.url)
    : undefined;
  const eegSpecificDocs = legalDocuments.filter((d) => !d.isCentralPolicy);

  // returns the resolved FieldState for an application-level configurable field
  function fs(name: string): FieldState {
    const field = CONFIGURABLE_FIELDS.application.find((f) => f.name === name);
    return resolveFieldState(fieldConfig, name, field?.defaultState ?? "hidden");
  }

  const req = (name: string) => fs(name) === "required" ? " *" : "";

  const form = useForm<RegistrationFormValues>({
    resolver: zodResolver(buildFormSchema(fieldConfig, sepaMandateEnabled)),
    defaultValues: {
      memberType: "private",
      titel: "",
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
      privacyAccepted: !showCentralPolicy,
      accuracyConfirmed: false,
      sepaMandateAccepted: sepaMandateEnabled ? true : false,
      meteringPoints: [{ meteringPoint: "", direction: "CONSUMPTION", participationFactor: 100 }],
      membershipStartDate: "",
      personsInHousehold: undefined,
      consumptionPreviousYear: undefined,
      consumptionForecast: undefined,
      feedInForecast: undefined,
      pvPowerKwp: undefined,
      heatPump: undefined,
      electricVehicle: undefined,
      electricHotWater: undefined,
    },
  });

  const memberType = form.watch("memberType");
  const isPerson = memberType === "private" || memberType === "farmer";

  // extra configurable fields that default to "hidden"
  const extraFieldNames = [
    "membership_start_date", "persons_in_household", "consumption_previous_year",
    "consumption_forecast", "feed_in_forecast", "pv_power_kwp",
    "heat_pump", "electric_vehicle", "electric_hot_water",
  ];
  const hasExtraFields = extraFieldNames.some((n) => fs(n) !== "hidden");

  function onMemberTypeChange(value: MemberType) {
    form.setValue("memberType", value);
    if (value === "private" || value === "farmer") {
      form.setValue("companyName", "");
      form.setValue("uidNumber", "");
      form.setValue("registerNumber", "");
      form.clearErrors(["companyName", "uidNumber", "registerNumber"]);
    } else {
      form.setValue("titel", "");
      form.setValue("firstname", "");
      form.setValue("lastname", "");
      form.setValue("birthDate", "");
      form.clearErrors(["titel", "firstname", "lastname", "birthDate"]);
    }
  }

  function parseBoolSelect(v: string): boolean | null {
    if (v === "true") return true;
    if (v === "false") return false;
    return null;
  }

  function boolSelectValue(v: boolean | null | undefined): string {
    if (v === true) return "true";
    if (v === false) return "false";
    return "__none__";
  }

  async function onSubmit(values: RegistrationFormValues) {
    // Validate required EEG-specific doc consents
    const errors: Record<string, string> = {};
    for (const doc of eegSpecificDocs) {
      if (doc.required && !docConsents[doc.id]) {
        errors[doc.id] = "Zustimmung ist erforderlich";
      }
    }
    setDocConsentErrors(errors);
    if (Object.keys(errors).length > 0) return;

    setIsSubmitting(true);
    setApiError(null);

    const isPersonType = values.memberType === "private" || values.memberType === "farmer";

    // Build consents array
    const consents: ConsentInput[] = [];
    if (centralPolicy && values.privacyAccepted) {
      consents.push({ title: centralPolicy.title, url: centralPolicy.url, isCentralPolicy: true });
    }
    for (const doc of eegSpecificDocs) {
      if (docConsents[doc.id]) {
        consents.push({ title: doc.title, url: doc.url, isCentralPolicy: false });
      }
    }

    try {
      const app = await createApplication({
        rcNumber: config.rcNumber,
        memberType: values.memberType,
        titel: isPersonType ? values.titel || undefined : undefined,
        firstname: isPersonType ? values.firstname || undefined : undefined,
        lastname: isPersonType ? values.lastname || undefined : undefined,
        birthDate: isPersonType ? values.birthDate || undefined : undefined,
        companyName: !isPersonType ? values.companyName || undefined : undefined,
        uidNumber: values.uidNumber || undefined,
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
        membershipStartDate: values.membershipStartDate || undefined,
        personsInHousehold: values.personsInHousehold,
        consumptionPreviousYear: values.consumptionPreviousYear,
        consumptionForecast: values.consumptionForecast,
        feedInForecast: values.feedInForecast,
        pvPowerKwp: values.pvPowerKwp,
        heatPump: values.heatPump ?? null,
        electricVehicle: values.electricVehicle ?? null,
        electricHotWater: values.electricHotWater ?? null,
        meteringPoints: values.meteringPoints.map((mp) => ({
          meteringPoint: mp.meteringPoint,
          direction: mp.direction,
          participationFactor: mp.participationFactor,
          transformer: mp.transformer || undefined,
          installationNumber: mp.installationNumber || undefined,
          installationName: mp.installationName || undefined,
        })),
        turnstileToken: turnstileToken || undefined,
      });

      const submitted = await submitApplication(app.id, consents.length > 0 ? consents : undefined);

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
        } else if (code === "turnstile_failed" || code === "turnstile_missing") {
          setTurnstileToken(null);
          turnstileRef.current?.reset();
          setApiError("Sicherheitsprüfung fehlgeschlagen. Bitte lösen Sie das CAPTCHA erneut.");
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
        <div className="-mt-2">
          <IntroTextDisplay introText={config.introText} />
        </div>
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
                    name="titel"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Titel</FormLabel>
                        <FormControl>
                          <Input autoComplete="honorific-prefix" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="firstname"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Vorname *</FormLabel>
                        <FormControl>
                          <Input autoComplete="given-name" {...field} />
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
                          <Input autoComplete="family-name" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                {fs("birth_date") !== "hidden" && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="birthDate"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Geburtsdatum{req("birth_date")}</FormLabel>
                          <FormControl>
                            <Input type="date" autoComplete="bday" {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                )}
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
                        <Input autoComplete="organization" {...field} />
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
                            <Input {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}
                  {(memberType === "company" || memberType === "municipality") && (
                    <FormField
                      control={form.control}
                      name="uidNumber"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>UID-Nummer{memberType === "company" ? " *" : ""}</FormLabel>
                          <FormControl>
                            <Input {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}
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
                      <Input type="email" autoComplete="email" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {fs("phone") !== "hidden" && (
                <FormField
                  control={form.control}
                  name="phone"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Telefon{req("phone")}</FormLabel>
                      <FormControl>
                        <Input type="tel" autoComplete="tel" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </div>
          </CardContent>
        </Card>

        {/* Address */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Adresse (Rechnungsadresse)</CardTitle>
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
                        <Input autoComplete="address-line1" {...field} />
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
                      <Input {...field} />
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
                      <Input autoComplete="postal-code" {...field} />
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
                        <Input autoComplete="address-level2" {...field} />
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
                      <Input
                        {...field}
                        autoComplete="off"
                        onChange={(e) =>
                          field.onChange(e.target.value.toUpperCase())
                        }
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
                      <Input autoComplete="name" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
        </Card>

        {/* Extra configurable fields */}
        {hasExtraFields && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Weitere Angaben</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {fs("membership_start_date") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="membershipStartDate"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Aktiv am (Beitrittsdatum){req("membership_start_date")}</FormLabel>
                        <FormControl>
                          <Input type="date" {...field} />
                        </FormControl>
                        <p className="text-xs text-muted-foreground">
                          Datum, ab dem die Aktivierung der angegebenen Zählpunkte für die EEG erfolgen soll.
                          Nützlich wenn die Aktivierung nicht sofort, sondern zu einem fest definierten Zeitpunkt stattfinden soll.
                        </p>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                {fs("persons_in_household") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="personsInHousehold"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Personen im Haushalt{req("persons_in_household")}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}
                            value={field.value ?? ""}
                            onChange={(e) => field.onChange(isNaN(e.target.valueAsNumber) ? undefined : e.target.valueAsNumber)}
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
                {fs("consumption_previous_year") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="consumptionPreviousYear"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Verbrauch Vorjahr (kWh){req("consumption_previous_year")}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}

                            value={field.value ?? ""}
                            onChange={(e) => field.onChange(isNaN(e.target.valueAsNumber) ? undefined : e.target.valueAsNumber)}
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
                {fs("consumption_forecast") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="consumptionForecast"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Verbrauch Prognose (kWh){req("consumption_forecast")}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}

                            value={field.value ?? ""}
                            onChange={(e) => field.onChange(isNaN(e.target.valueAsNumber) ? undefined : e.target.valueAsNumber)}
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
                {fs("feed_in_forecast") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="feedInForecast"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Einspeisung Prognose (kWh){req("feed_in_forecast")}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}

                            value={field.value ?? ""}
                            onChange={(e) => field.onChange(isNaN(e.target.valueAsNumber) ? undefined : e.target.valueAsNumber)}
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
                {fs("pv_power_kwp") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="pvPowerKwp"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>PV-Leistung (kWp){req("pv_power_kwp")}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}
                            step={0.1}

                            value={field.value ?? ""}
                            onChange={(e) => field.onChange(isNaN(e.target.valueAsNumber) ? undefined : e.target.valueAsNumber)}
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
                {fs("heat_pump") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="heatPump"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Wärmepumpe vorhanden{req("heat_pump")}</FormLabel>
                        <Select
                          value={boolSelectValue(field.value)}
                          onValueChange={(v) => field.onChange(v === "__none__" ? null : parseBoolSelect(v))}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Bitte auswählen …" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="__none__">Keine Angabe</SelectItem>
                            <SelectItem value="true">Ja</SelectItem>
                            <SelectItem value="false">Nein</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                {fs("electric_vehicle") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="electricVehicle"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>E-Auto vorhanden{req("electric_vehicle")}</FormLabel>
                        <Select
                          value={boolSelectValue(field.value)}
                          onValueChange={(v) => field.onChange(v === "__none__" ? null : parseBoolSelect(v))}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Bitte auswählen …" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="__none__">Keine Angabe</SelectItem>
                            <SelectItem value="true">Ja</SelectItem>
                            <SelectItem value="false">Nein</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                {fs("electric_hot_water") !== "hidden" && (
                  <FormField
                    control={form.control}
                    name="electricHotWater"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Warmwasser elektrisch (Boiler){req("electric_hot_water")}</FormLabel>
                        <Select
                          value={boolSelectValue(field.value)}
                          onValueChange={(v) => field.onChange(v === "__none__" ? null : parseBoolSelect(v))}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Bitte auswählen …" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="__none__">Keine Angabe</SelectItem>
                            <SelectItem value="true">Ja</SelectItem>
                            <SelectItem value="false">Nein</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Metering points */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Zählpunkte</CardTitle>
          </CardHeader>
          <CardContent>
            <MeteringPointFields form={form} fieldConfig={fieldConfig} />
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
            {showCentralPolicy && (
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
                        {centralPolicy ? (
                          <>
                            Ich habe die{" "}
                            <a href={centralPolicy.url} target="_blank" rel="noopener noreferrer" className="underline hover:text-foreground">
                              {centralPolicy.title}
                            </a>{" "}
                            gelesen und stimme der Verarbeitung meiner Daten zu. *
                          </>
                        ) : (
                          <>
                            Ich habe die{" "}
                            <a href="/datenschutz" target="_blank" rel="noopener noreferrer" className="underline hover:text-foreground">
                              Datenschutzerklärung
                            </a>{" "}
                            gelesen und stimme der Verarbeitung meiner Daten zu. *
                          </>
                        )}
                      </FormLabel>
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />
            )}
            {eegSpecificDocs.map((doc) => (
              <div key={doc.id} className="flex flex-row items-start gap-3">
                <Checkbox
                  id={`doc-${doc.id}`}
                  checked={docConsents[doc.id] ?? false}
                  onCheckedChange={(checked) => {
                    setDocConsents((prev) => ({ ...prev, [doc.id]: checked === true }));
                    if (checked) {
                      setDocConsentErrors((prev) => { const n = { ...prev }; delete n[doc.id]; return n; });
                    }
                  }}
                />
                <div className="space-y-1 leading-none">
                  <label htmlFor={`doc-${doc.id}`} className="text-sm font-normal cursor-pointer">
                    Ich habe „
                    <a href={doc.url} target="_blank" rel="noopener noreferrer" className="underline hover:text-foreground">
                      {doc.title}
                    </a>
                    " gelesen und stimme zu.{doc.required ? " *" : ""}
                  </label>
                  {docConsentErrors[doc.id] && (
                    <p className="text-sm font-medium text-destructive">{docConsentErrors[doc.id]}</p>
                  )}
                </div>
              </div>
            ))}
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
            {!sepaMandateEnabled && (
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
            )}
          </CardContent>
        </Card>

        {TURNSTILE_SITE_KEY && (
          <Turnstile
            ref={turnstileRef}
            siteKey={TURNSTILE_SITE_KEY}
            onSuccess={(token) => setTurnstileToken(token)}
            onExpire={() => setTurnstileToken(null)}
            onError={() => setTurnstileToken(null)}
            options={{ theme: "auto" }}
          />
        )}

        <div>
          <Button
            type="submit"
            size="lg"
            disabled={isSubmitting || (!!TURNSTILE_SITE_KEY && !turnstileToken)}
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
