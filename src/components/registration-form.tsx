"use client";

import { useRef, useState } from "react";
import { Turnstile, type TurnstileInstance } from "@marsidev/react-turnstile";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { AlertCircle, CheckCircle2, Info } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { MaskedInput } from "@/components/ui/masked-input";
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
import { IBAN_DYNAMIC_MASK, IBAN_DEFINITIONS } from "@/lib/iban-mask";
import {
  createApplication,
  submitApplication,
  ApiResponseError,
  resolveFieldState,
  CONFIGURABLE_FIELDS,
  NETWORK_OPERATOR_AUTH_TEXT,
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
  { value: "private",         label: "Privatperson",                    hint: "" },
  { value: "sole_proprietor", label: "Kleinunternehmer",                hint: "0 % USt." },
  { value: "farmer",          label: "Pauschalierter Landwirt",         hint: "13 % USt." },
  { value: "municipality",    label: "Gemeinde / öffentliche Körperschaft", hint: "variabel" },
  { value: "company",         label: "Unternehmen",                     hint: "20 % USt." },
  { value: "association",     label: "Verein",                          hint: "variabel" },
];

// ---------- Zod schema ----------

const meteringPointSchema = z
  .object({
    meteringPoint: z
      .string()
      .transform((v) => v.replace(/\s/g, "").toUpperCase())
      .refine((v) => v.length >= 1, { message: "Zählpunkt ist erforderlich" })
      // PROJ-52: Stellen 3–13 müssen Ziffern bleiben (Netzbetreibernummer + PLZ
      // nach E-Control-Spec), die letzten 20 Stellen sind alphanumerisch.
      // Bestandsdaten bleiben gültig (Ziffern ⊂ [A-Z0-9]).
      .refine((v) => /^AT\d{11}[A-Z0-9]{20}$/.test(v), {
        message: "Zählpunkt muss mit AT + 11 Ziffern beginnen, gefolgt von 20 alphanumerischen Stellen (33 Zeichen gesamt)",
      }),
    direction: z.enum(["CONSUMPTION", "PRODUCTION"]),
    participationFactor: z.number().int().min(1, "Mindestens 1%").max(100, "Maximal 100%"),
    transformer: z.string().trim().max(100).optional(),
    installationNumber: z.string().trim().max(50).optional(),
    installationName: z.string().trim().max(100).optional(),
    // PROJ-39: abweichende Adresse je Zählpunkt. UI-Checkbox-State wird
    // nicht persistiert — der Server leitet ihn beim Reload aus dem
    // Befülltsein der vier Felder ab. Hier optional auf Schema-Ebene;
    // superRefine im baseSchema erzwingt das All-or-Nothing.
    addressStreet: z.string().trim().max(255).optional(),
    addressStreetNumber: z.string().trim().max(50).optional(),
    addressZip: z.string().trim().max(20).optional(),
    addressCity: z.string().trim().max(255).optional(),
    // PROJ-45: Erzeugungsform + Batterie. generationType ist nur für
    // PRODUCTION relevant — UI rendert das Feld auch nur dann.
    generationType: z.enum(["pv", "hydro", "wind", "biomass"]).optional(),
    batterySizeKwh: z.number().min(0).optional(),
    inverterManufacturer: z.string().trim().max(100).optional(),
    // PROJ-49: Energie-Felder pro Zählpunkt. Sichtbarkeit per direction +
    // generationType — UI rendert nur die jeweils sinnvollen Felder, Server
    // cleart unzutreffende Werte serverseitig.
    consumptionPreviousYear: z.number().int().min(0).optional(),
    consumptionForecast: z.number().int().min(0).optional(),
    feedInForecast: z.number().int().min(0).optional(),
    pvPowerKwp: z.number().min(0).optional(),
    inverterPowerKw: z.number().min(0).optional(),
    feedInLimitPresent: z.boolean().optional(),
    feedInLimitKw: z.number().min(0).optional(),
    // PROJ-49 follow-up: „Speichersteuerung im Sinne der EEG vorstellbar?"
    // Nur sinnvoll bei PV + vorhandenem Speicher (Server cleart sonst).
    batteryControlAcceptable: z.boolean().optional(),
  })
  .superRefine((mp, ctx) => {
    const fields = [mp.addressStreet, mp.addressStreetNumber, mp.addressZip, mp.addressCity];
    const filled = fields.filter((v) => v && v.trim().length > 0).length;
    if (filled > 0 && filled < 4) {
      const names = ["addressStreet", "addressStreetNumber", "addressZip", "addressCity"] as const;
      names.forEach((name, i) => {
        if (!fields[i] || fields[i]!.trim().length === 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [name],
            message: "Bei abweichender Adresse sind alle Adressfelder Pflicht",
          });
        }
      });
    }
  });

const baseSchema = z.object({
  memberType: z.enum(["private", "sole_proprietor", "farmer", "municipality", "company", "association"] as const),
  titel: z.string().trim().max(50).optional(),
  titelNach: z.string().trim().max(50).optional(),
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
    // Strip alles außer [A-Z0-9]: entfernt sowohl Mask-Spaces als auch
    // die iMask-Platzhalter (`_`), die mit lazy=false in unbefüllten
    // Slots im value mitgeliefert werden.
    .transform((v) => v.replace(/[^A-Za-z0-9]/g, "").toUpperCase())
    .refine((v) => isValidIBAN(v), { message: "Ungültige IBAN" }),
  accountHolder: z.string().trim().min(1, "Kontoinhaber:in ist erforderlich").max(255),
  bankName: z.string().trim().max(255).optional(),
  privacyAccepted: z.boolean().refine((v) => v === true, {
    message: "Datenschutzerklärung muss akzeptiert werden",
  }),
  accuracyConfirmed: z.boolean().refine((v) => v === true, {
    message: "Richtigkeit der Angaben muss bestätigt werden",
  }),
  sepaMandateAccepted: z.boolean(),
  // PROJ-44: required-Validierung via buildFormSchema, abhängig vom field_config
  networkOperatorAuthorization: z.boolean().optional(),
  meteringPoints: z
    .array(meteringPointSchema)
    .min(1, "Mindestens ein Zählpunkt ist erforderlich")
    .max(10, "Maximal 10 Zählpunkte erlaubt"),
  // configurable application-level fields. PROJ-49: consumption / feed_in /
  // pv_power leben jetzt pro Zählpunkt — siehe meteringPointSchema.
  membershipStartDate: z.string().optional(),
  personsInHousehold: z.number().int().min(0).optional(),
  heatPump: z.boolean().nullable().optional(),
  electricVehicle: z.boolean().nullable().optional(),
  // PROJ-42: nur sinnvoll wenn electricVehicle = true. Server cleart sonst.
  electricVehicleCount: z.number().int().min(1).optional(),
  electricVehicleAnnualKm: z.number().int().min(0).optional(),
  electricHotWater: z.boolean().nullable().optional(),
  // PROJ-37: Genossenschaftsanteile (nur Pflicht wenn EEG es aktiviert hat
  // — Validierung gegen den configurierten Pflichtwert wird in
  // buildFormSchema via superRefine ergänzt).
  cooperativeSharesCount: z.number().int().min(1).optional(),
});

export type RegistrationFormValues = z.infer<typeof baseSchema>;

function buildFormSchema(
  fieldConfig: FieldConfig | undefined,
  sepaMandateEnabled: boolean,
  cooperativeSharesEnabled: boolean,
  cooperativeRequiredShares: number,
) {
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
        : data.memberType === "sole_proprietor" ? "Firmenbezeichnung"
        : "Firmenname";
      if (!data.companyName?.trim()) {
        ctx.addIssue({ code: "custom", path: ["companyName"], message: `${orgLabel} ist erforderlich` });
      }
      if (data.memberType === "company") {
        if (!data.uidNumber?.trim()) {
          ctx.addIssue({ code: "custom", path: ["uidNumber"], message: "UID-Nummer ist erforderlich" });
        }
        // Firmenbuchnummer ist optional — manche Firmen (z. B. nicht
        // firmenbuchpflichtige Einzelunternehmer) haben keine.
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
    // Geburtsdatum-Feld wird im UI nur für isPerson-Typen gerendert;
    // für Organisationen (sole_proprietor, company, municipality, association)
    // existiert kein Eingabefeld — die Validierung muss daher ebenfalls
    // auf isPerson begrenzt sein, sonst schlägt der Submit unsichtbar fehl.
    if (isPerson) {
      requireText("birth_date", "birthDate", "Geburtsdatum");
    }
    requireText("bank_name", "bankName", "Bankname");
    requireText("membership_start_date", "membershipStartDate", "Beitrittsdatum");
    // Konsum-bezogene Felder werden im UI nur gezeigt, wenn mindestens ein
    // CONSUMPTION-Zählpunkt vorhanden ist (siehe shouldShow + consumptionFields
    // weiter unten). Die Validierung muss dieselbe Bedingung tragen, sonst
    // wäre der Submit für eine reine PRODUCTION-Anlage stumm blockiert.
    const hasConsumption = (data.meteringPoints ?? []).some(
      (m) => m?.direction === "CONSUMPTION",
    );
    if (hasConsumption) {
      requireNum("persons_in_household", "personsInHousehold", "Anzahl Personen im Haushalt");
      // PROJ-49: consumption_*, feed_in_forecast, pv_power_kwp werden jetzt
      // pro Zählpunkt validiert (Backend) — keine app-level Required-Checks mehr.
      requireNum("heat_pump", "heatPump", "Wärmepumpe vorhanden");
      requireNum("electric_vehicle", "electricVehicle", "E-Auto vorhanden");
      requireNum("electric_hot_water", "electricHotWater", "Warmwasser elektrisch");
    }

    // PROJ-44: Netzbetreiber-Vollmacht. Wenn required, muss das Häkchen
    // explizit gesetzt sein (false zählt nicht als Erteilung).
    if (resolve("network_operator_authorization") === "required" && !data.networkOperatorAuthorization) {
      ctx.addIssue({
        code: "custom",
        path: ["networkOperatorAuthorization"],
        message: "Netzbetreiber-Vollmacht muss erteilt werden",
      });
    }

    // PROJ-37: Genossenschaftsanteile required when the EEG has enabled
    // it. Count must be at least cooperativeRequiredShares; voluntary
    // higher is fine.
    if (cooperativeSharesEnabled) {
      if (data.cooperativeSharesCount === undefined || data.cooperativeSharesCount === null) {
        ctx.addIssue({
          code: "custom",
          path: ["cooperativeSharesCount"],
          message: "Anzahl der Anteile ist erforderlich",
        });
      } else if (data.cooperativeSharesCount < cooperativeRequiredShares) {
        ctx.addIssue({
          code: "custom",
          path: ["cooperativeSharesCount"],
          message: `Mindestens ${cooperativeRequiredShares} Pflichtanteil(e) müssen gezeichnet werden`,
        });
      }
    }

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
  // Cache the application id between createApplication-success and a later
  // retry of submitApplication so a transient failure on submit does not
  // create a second draft when the user clicks "Einreichen" again.
  // Invalidated when the user changes any form value (snapshot mismatch) or
  // when the backend reports the draft is gone (404).
  const pendingApplicationIdRef = useRef<string | null>(null);
  const lastSubmittedSnapshotRef = useRef<string | null>(null);

  const fieldConfig = config.fieldConfig;
  const sepaMandateEnabled = config.sepaMandateEnabled ?? false;
  const sepaMandateAtImport = config.sepaMandateAtImport ?? false;
  const showCentralPolicy = config.showCentralPolicy ?? true;
  const legalDocuments = config.legalDocuments ?? [];
  const centralPolicy = showCentralPolicy
    ? legalDocuments.find((d) => d.isCentralPolicy && d.url)
    : undefined;
  const eegSpecificDocs = legalDocuments.filter((d) => !d.isCentralPolicy);
  // PROJ-36: split EEG-specific docs into "required" (member must tick a
  // checkbox to confirm) and "informational" (link is shown but no
  // checkbox — server records an `informational` consent at submit time
  // from the legal_document table). The old "optional checkbox" mode is
  // gone — users were confused whether a non-required tick mattered.
  const requiredEegDocs = eegSpecificDocs.filter((d) => d.required);
  const informationalEegDocs = eegSpecificDocs.filter((d) => !d.required);

  // returns the resolved FieldState for an application-level configurable field
  function fs(name: string): FieldState {
    const field = CONFIGURABLE_FIELDS.application.find((f) => f.name === name);
    return resolveFieldState(fieldConfig, name, field?.defaultState ?? "hidden");
  }

  const req = (name: string) => fs(name) === "required" ? " *" : "";

  const form = useForm<RegistrationFormValues>({
    resolver: zodResolver(buildFormSchema(
      fieldConfig,
      sepaMandateEnabled,
      config.cooperativeSharesEnabled ?? false,
      config.cooperativeRequiredShares ?? 1,
    )),
    defaultValues: {
      memberType: "private",
      titel: "",
      titelNach: "",
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
      bankName: "",
      privacyAccepted: !showCentralPolicy,
      accuracyConfirmed: false,
      sepaMandateAccepted: sepaMandateEnabled ? true : false,
      meteringPoints: [{ meteringPoint: "", direction: "CONSUMPTION", participationFactor: 100, generationType: "pv" }],
      membershipStartDate: "",
      personsInHousehold: undefined,
      heatPump: undefined,
      electricVehicle: undefined,
      electricVehicleCount: undefined,
      electricVehicleAnnualKm: undefined,
      electricHotWater: undefined,
      // PROJ-37: pre-fill with required-shares so the input starts at min.
      // If the EEG hasn't enabled the feature, this value is silently
      // ignored on submit (backend ignores when entrypoint disabled).
      cooperativeSharesCount: config.cooperativeSharesEnabled
        ? (config.cooperativeRequiredShares ?? 1)
        : undefined,
      // PROJ-44: ungesetzt — Mitglied muss aktiv das Häkchen setzen.
      networkOperatorAuthorization: false,
    },
  });

  const memberType = form.watch("memberType");
  const isPerson = memberType === "private" || memberType === "farmer";

  // PROJ-45: typabhängige Sichtbarkeit der App-level Felder. Wir leiten
  // hasConsumption live aus den eingegebenen Zählpunkten ab. PROJ-49: die
  // PRODUCTION-only Felder (PV-Leistung, Einspeisung Prognose) sind jetzt
  // pro Zählpunkt — hasProduction wird hier nicht mehr benötigt.
  const watchedMps = form.watch("meteringPoints");
  const hasConsumption = (watchedMps ?? []).some((m) => m?.direction === "CONSUMPTION");

  // Mapping: Feld → benötigter Zählpunkttyp.
  const consumptionFields = new Set([
    "persons_in_household",
    "heat_pump", "electric_vehicle", "electric_hot_water",
  ]);
  function shouldShow(name: string): boolean {
    if (fs(name) === "hidden") return false;
    if (consumptionFields.has(name) && !hasConsumption) return false;
    return true;
  }

  // extra configurable fields that default to "hidden"
  const extraFieldNames = [
    "membership_start_date", "persons_in_household",
    "heat_pump", "electric_vehicle", "electric_hot_water",
  ];
  const hasExtraFields = extraFieldNames.some((n) => shouldShow(n));

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
    // PROJ-36: only required documents have checkboxes. Non-required ones
    // get an informational consent written server-side at submit time —
    // they are not validated here.
    const errors: Record<string, string> = {};
    for (const doc of requiredEegDocs) {
      if (!docConsents[doc.id]) {
        errors[doc.id] = "Zustimmung ist erforderlich";
      }
    }
    setDocConsentErrors(errors);
    if (Object.keys(errors).length > 0) return;

    setIsSubmitting(true);
    setApiError(null);

    const isPersonType = values.memberType === "private" || values.memberType === "farmer";

    // Build consents array. Frontend only sends explicit (required) ticks;
    // backend writes informational entries for non-required docs.
    const consents: ConsentInput[] = [];
    if (centralPolicy && values.privacyAccepted) {
      consents.push({ title: centralPolicy.title, url: centralPolicy.url, isCentralPolicy: true });
    }
    for (const doc of requiredEegDocs) {
      if (docConsents[doc.id]) {
        consents.push({ title: doc.title, url: doc.url, isCentralPolicy: false });
      }
    }

    // Snapshot the form values so we can detect whether a retry comes after
    // the user edited something (then we must re-create the draft) or is a
    // pure re-submit (then we reuse the existing application id).
    const valuesSnapshot = JSON.stringify(values);
    const canReuseDraft =
      pendingApplicationIdRef.current !== null &&
      lastSubmittedSnapshotRef.current === valuesSnapshot;

    try {
      let applicationId: string;
      if (canReuseDraft) {
        applicationId = pendingApplicationIdRef.current!;
      } else {
        const app = await createApplication({
          rcNumber: config.rcNumber,
          memberType: values.memberType,
          titel: isPersonType ? values.titel || undefined : undefined,
          titelNach: isPersonType ? values.titelNach || undefined : undefined,
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
          bankName: values.bankName || undefined,
          sepaMandateAccepted: values.sepaMandateAccepted,
          membershipStartDate: values.membershipStartDate || undefined,
          personsInHousehold: values.personsInHousehold,
          heatPump: values.heatPump ?? null,
          electricVehicle: values.electricVehicle ?? null,
          electricVehicleCount: values.electricVehicle ? values.electricVehicleCount : undefined,
          electricVehicleAnnualKm: values.electricVehicle ? values.electricVehicleAnnualKm : undefined,
          electricHotWater: values.electricHotWater ?? null,
          cooperativeSharesCount: values.cooperativeSharesCount,
          networkOperatorAuthorization: values.networkOperatorAuthorization || undefined,
          meteringPoints: values.meteringPoints.map((mp) => {
            const isProduction = mp.direction === "PRODUCTION";
            const isPv = isProduction && (mp.generationType ?? "pv") === "pv";
            const isConsumption = mp.direction === "CONSUMPTION";
            return {
              meteringPoint: mp.meteringPoint,
              direction: mp.direction,
              participationFactor: mp.participationFactor,
              transformer: mp.transformer || undefined,
              installationNumber: mp.installationNumber || undefined,
              installationName: mp.installationName || undefined,
              addressStreet: mp.addressStreet || undefined,
              addressStreetNumber: mp.addressStreetNumber || undefined,
              addressZip: mp.addressZip || undefined,
              addressCity: mp.addressCity || undefined,
              // PROJ-45: server normalisiert nochmal (CONSUMPTION ⇒ nil),
              // aber wir senden bewusst nur was relevant ist.
              generationType: isProduction ? (mp.generationType ?? "pv") : undefined,
              batterySizeKwh: isPv ? mp.batterySizeKwh : undefined,
              inverterManufacturer: isPv ? (mp.inverterManufacturer || undefined) : undefined,
              batteryControlAcceptable: isPv ? mp.batteryControlAcceptable : undefined,
              // PROJ-49: Energie-Felder pro Zählpunkt.
              consumptionPreviousYear: isConsumption ? mp.consumptionPreviousYear : undefined,
              consumptionForecast: isConsumption ? mp.consumptionForecast : undefined,
              feedInForecast: isProduction ? mp.feedInForecast : undefined,
              pvPowerKwp: isPv ? mp.pvPowerKwp : undefined,
              inverterPowerKw: isPv ? mp.inverterPowerKw : undefined,
              feedInLimitPresent: isPv ? mp.feedInLimitPresent : undefined,
              feedInLimitKw: isPv && mp.feedInLimitPresent ? mp.feedInLimitKw : undefined,
            };
          }),
          turnstileToken: turnstileToken || undefined,
        });
        applicationId = app.id;
        pendingApplicationIdRef.current = app.id;
        lastSubmittedSnapshotRef.current = valuesSnapshot;
      }

      const submitted = await submitApplication(applicationId, consents.length > 0 ? consents : undefined);

      // Terminal success — release the cached id; the success view will
      // unmount the form anyway, but be explicit.
      pendingApplicationIdRef.current = null;
      lastSubmittedSnapshotRef.current = null;

      setSuccess({
        referenceNumber: submitted.referenceNumber,
        submittedAt: submitted.submittedAt,
      });
    } catch (err) {
      // If the cached draft is gone (cron sweep, manual delete), reset so the
      // next click re-creates it instead of looping on a stale id.
      if (err instanceof ApiResponseError && err.apiError.code === "not_found") {
        pendingApplicationIdRef.current = null;
        lastSubmittedSnapshotRef.current = null;
      }
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
          setApiError("Sicherheitsprüfung fehlgeschlagen. Bitte löse das CAPTCHA erneut.");
        } else {
          setApiError(message || "Ein Fehler ist aufgetreten.");
        }
      } else {
        setApiError(
          "Ein unerwarteter Fehler ist aufgetreten. Bitte versuche es erneut."
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
            {config.requireEmailConfirmation
              ? "Bitte prüfe jetzt dein E-Mail-Postfach und bestätige deine E-Mail-Adresse über den zugesandten Link."
              : "Dein Antrag wurde übermittelt und wird nun von unserem Team geprüft."}
          </p>
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium">Deine Antragsnummer:</span>
            <Badge variant="secondary" className="font-mono text-sm">
              {success.referenceNumber}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Bitte notiere diese Nummer für eventuelle Rückfragen.
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
                          {opt.hint && (
                            <span className="ml-2 text-xs text-muted-foreground">({opt.hint})</span>
                          )}
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
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="titelNach"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Titel nach</FormLabel>
                        <FormControl>
                          <Input autoComplete="honorific-suffix" {...field} />
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
                            {memberType === "association" ? "Vereinsnummer *" : "Firmenbuchnummer"}
                          </FormLabel>
                          <FormControl>
                            <Input {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}
                  {(memberType === "company" || memberType === "municipality" || memberType === "association") && (
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

        {/* PROJ-37: Genossenschaftsanteile — wird gerendert, sobald das EEG-
            Feature aktiviert ist. Der Anteilswert (cooperativeShareAmountCents)
            kann fehlen, wenn der Admin den Toggle umgelegt aber den Wert noch
            nicht eingetragen hat — Backend lehnt den Submit dann ohne Wert
            mit 400 ab; UI fällt nicht still aus dem Block raus (siehe
            Sichtbarkeits-Bug-Fix). Der Anteilswert wird hier nur zur
            Gesamtbetrags-Anzeige verwendet; ohne ihn rendern wir den Block
            trotzdem mit einem Hinweis. */}
        {config.cooperativeSharesEnabled && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Genossenschaftsanteile</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Pflichtanteil je Standort: <strong>{config.cooperativeRequiredShares ?? 1}</strong>{" "}
                {(config.cooperativeRequiredShares ?? 1) === 1 ? "Anteil" : "Anteile"}
              </p>

              <FormField
                control={form.control}
                name="cooperativeSharesCount"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Anzahl Anteile gesamt *</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        inputMode="numeric"
                        min={config.cooperativeRequiredShares ?? 1}
                        value={field.value ?? ""}
                        onChange={(e) => {
                          const v = e.target.value;
                          field.onChange(v === "" ? undefined : parseInt(v, 10));
                        }}
                      />
                    </FormControl>
                    <p className="text-xs text-muted-foreground">
                      min. {config.cooperativeRequiredShares ?? 1} (Pflichtanteile),
                      freiwillig mehr möglich
                    </p>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {config.cooperativeShareAmountCents != null ? (
                (() => {
                  const amount = config.cooperativeShareAmountCents ?? 0;
                  const count = form.watch("cooperativeSharesCount") ?? 0;
                  const formatEur = (cents: number) =>
                    new Intl.NumberFormat("de-AT", {
                      style: "currency",
                      currency: "EUR",
                    }).format(cents / 100);
                  return (
                    <div className="text-sm space-y-1 border-t pt-3">
                      <div className="flex justify-between">
                        <span>Genossenschaftsanteilswert:</span>
                        <span>{formatEur(amount)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span>Gezeichnete Anteile:</span>
                        <span>× {count}</span>
                      </div>
                      <div className="flex justify-between font-semibold border-t pt-2 mt-1">
                        <span>Gesamtbetrag:</span>
                        <span>{formatEur(amount * count)}</span>
                      </div>
                    </div>
                  );
                })()
              ) : (
                <p className="text-xs text-muted-foreground border-t pt-3">
                  Der Anteilswert wird dir von deiner Energiegemeinschaft
                  separat mitgeteilt.
                </p>
              )}
            </CardContent>
          </Card>
        )}

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
                        // PROJ-29: dynamic country-aware IBAN mask. The first
                        // two typed letters select the country-specific mask
                        // (correct length + correct digit/letter positions).
                        // Until then a generic 34-char alphanumeric fallback
                        // applies. `isValidIBAN` remains the final authority.
                        {...IBAN_DYNAMIC_MASK}
                        definitions={IBAN_DEFINITIONS}
                        lazy={false}
                        prepareChar={(str: string) => str.toUpperCase()}
                        value={field.value}
                        onAccept={(value: string) => field.onChange(value)}
                        onBlur={field.onBlur}
                        inputRef={field.ref}
                        name={field.name}
                        autoComplete="off"
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
              {fs("bank_name") !== "hidden" && (
                <FormField
                  control={form.control}
                  name="bankName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Bankname{req("bank_name")}</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </div>
          </CardContent>
        </Card>

        {/* Metering points — first so the typabhängige Sichtbarkeit
            der "Weitere Angaben"-Felder (PROJ-45) erst nach der
            Zählpunkt-Eingabe greift (Verbraucher- vs. Einspeise-
            Felder werden dynamisch ein-/ausgeblendet). */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Zählpunkte</CardTitle>
          </CardHeader>
          <CardContent>
            <MeteringPointFields
              form={form}
              fieldConfig={fieldConfig}
              prefixConsumption={config.meteringPointPrefixConsumption ?? null}
              prefixProduction={config.meteringPointPrefixProduction ?? null}
            />
            {form.formState.errors.meteringPoints?.message && (
              <p className="text-sm font-medium text-destructive mt-3">
                {form.formState.errors.meteringPoints.message}
              </p>
            )}
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
                        <div className="flex items-center gap-1">
                          <FormLabel>Aktiv am (Beitrittsdatum){req("membership_start_date")}</FormLabel>
                          <Popover>
                            <PopoverTrigger type="button" className="cursor-help">
                              <Info className="h-3.5 w-3.5 text-muted-foreground" />
                            </PopoverTrigger>
                            <PopoverContent className="max-w-72 text-sm">
                              Datum, ab dem die Aktivierung der angegebenen Zählpunkte für die EEG erfolgen soll.
                              Nützlich wenn die Aktivierung nicht sofort, sondern zu einem fest definierten Zeitpunkt stattfinden soll.
                            </PopoverContent>
                          </Popover>
                        </div>
                        <FormControl>
                          <Input type="date" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                {shouldShow("persons_in_household") && (
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
                {/* PROJ-49: consumption_*, feed_in_forecast, pv_power_kwp
                    sind jetzt pro Zählpunkt (siehe MeteringPointFields). */}
                {shouldShow("heat_pump") && (
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
                {shouldShow("electric_vehicle") && (
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
                {/* PROJ-42: EV-Details nur wenn EEG diese Felder aktiviert hat
                    UND der Bewerber "Ja" beim E-Auto angekreuzt hat. */}
                {fs("electric_vehicle_count") !== "hidden" && form.watch("electricVehicle") === true && (
                  <FormField
                    control={form.control}
                    name="electricVehicleCount"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Anzahl E-Fahrzeuge{req("electric_vehicle_count")}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={1}
                            value={field.value ?? ""}
                            onChange={(e) => field.onChange(e.target.value === "" ? undefined : parseInt(e.target.value, 10))}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                {fs("electric_vehicle_annual_km") !== "hidden" && form.watch("electricVehicle") === true && (
                  <FormField
                    control={form.control}
                    name="electricVehicleAnnualKm"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Jahres-Kilometer (E-Fahrzeuge){req("electric_vehicle_annual_km")}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}
                            value={field.value ?? ""}
                            onChange={(e) => field.onChange(e.target.value === "" ? undefined : parseInt(e.target.value, 10))}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                {shouldShow("electric_hot_water") && (
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
            {/* PROJ-36: required EEG-specific document consents stay grouped
                with the other required checkboxes (privacy, accuracy, sepa).
                Informational documents are moved to a separate block below. */}
            {requiredEegDocs.map((doc) => (
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
                    " gelesen und stimme zu. *
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
            {sepaMandateEnabled && sepaMandateAtImport && (
              <p className="text-sm text-muted-foreground pl-1">
                Das SEPA-Lastschriftmandat erhältst du nach der Freigabe deines
                Antrags per E-Mail — mit eingetragener Mandatsreferenz (deiner
                Mitgliedsnummer) zur Unterschrift.
              </p>
            )}
            {fs("network_operator_authorization") !== "hidden" && (
              <FormField
                control={form.control}
                name="networkOperatorAuthorization"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-start gap-3 space-y-0">
                    <FormControl>
                      <Checkbox
                        checked={field.value ?? false}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                      <FormLabel className="font-normal cursor-pointer">
                        {NETWORK_OPERATOR_AUTH_TEXT}
                        {fs("network_operator_authorization") === "required" && " *"}
                      </FormLabel>
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />
            )}
            {/* PROJ-36: informational documents are visually separated from
                the required-confirmation checkboxes so the user clearly
                sees these are kein „weiteres Häkchen zum Übersehen". */}
            {informationalEegDocs.length > 0 && (
              <div className="pt-2 mt-2 border-t space-y-2">
                <p className="text-sm font-medium">Zur Information</p>
                <p className="text-xs text-muted-foreground">
                  Die folgenden Dokumente werden dir zur Information bereitgestellt.
                  Mit Absenden des Antrags bestätigst du, sie zur Kenntnis genommen zu haben:
                </p>
                <ul className="list-disc pl-5 text-sm space-y-1">
                  {informationalEegDocs.map((doc) => (
                    <li key={doc.id}>
                      <a
                        href={doc.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="underline hover:text-foreground"
                      >
                        {doc.title}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
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
