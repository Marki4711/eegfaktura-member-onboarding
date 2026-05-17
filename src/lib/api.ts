// Server-side (SSR/Node.js): use BACKEND_URL (runtime env var, set in Helm)
// Client-side (browser): use NEXT_PUBLIC_API_URL (baked at build) or "" for relative URLs via ingress
function getBaseUrl(): string {
  if (typeof window === "undefined") {
    return process.env.BACKEND_URL ?? "http://localhost:8080";
  }
  return process.env.NEXT_PUBLIC_API_URL ?? "";
}

const API_URL = getBaseUrl();

// ---------- response shapes ----------

export type FieldState = "hidden" | "optional" | "required" | "admin_only";
// Public registration form uses the simpler map (backend maps admin_only → hidden).
export type FieldConfig = Record<string, FieldState>;

// Admin field config uses the richer format with optional admin-provided default value.
export interface AdminFieldConfigEntry {
  state: FieldState;
  adminValue?: string;
}
export type AdminFieldConfig = Record<string, AdminFieldConfigEntry>;

export interface ConfigurableField {
  name: string;
  label: string;
  defaultState: FieldState;
  // PROJ-45: when set, rendered as an Info-Popover next to the label in the
  // admin field-config editor so admins see at a glance that a field only
  // takes effect under specific conditions (Zählpunkt-Typ, EV-Flag, …).
  visibilityHint?: string;
  // PROJ-45: small coloured badges shown next to the label in the admin
  // editor. Order matters — primary condition first (e.g. ["consumption","ev"]
  // means "needs CONSUMPTION-Zählpunkt and additionally E-Auto = Ja").
  visibilityTags?: VisibilityTag[];
}

// PROJ-45: visibility-condition tag taxonomy. Each value renders as a
// coloured badge in the admin field-config editor. Keep the set small —
// new tags require a label + colour mapping in admin-field-config-editor.
export type VisibilityTag = "consumption" | "production" | "pv" | "ev";

export const CONFIGURABLE_FIELDS: {
  application: ConfigurableField[];
  meteringPoint: ConfigurableField[];
} = {
  application: [
    { name: "phone",                   label: "Telefonnummer",                   defaultState: "optional" },
    { name: "birth_date",              label: "Geburtsdatum",                    defaultState: "optional" },
    { name: "membership_start_date",   label: "Aktiv am (Beitrittsdatum)",       defaultState: "hidden"   },
    { name: "persons_in_household",    label: "Anzahl Personen im Haushalt",     defaultState: "hidden",
      visibilityTags: ["consumption"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Verbraucher-Zählpunkt enthält." },
    { name: "consumption_previous_year", label: "Verbrauch Vorjahr (kWh)",       defaultState: "hidden",
      visibilityTags: ["consumption"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Verbraucher-Zählpunkt enthält." },
    { name: "consumption_forecast",    label: "Verbrauch Prognose (kWh)",        defaultState: "hidden",
      visibilityTags: ["consumption"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Verbraucher-Zählpunkt enthält." },
    { name: "feed_in_forecast",        label: "Einspeisung Prognose (kWh)",      defaultState: "hidden",
      visibilityTags: ["production"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Einspeise-Zählpunkt enthält." },
    { name: "pv_power_kwp",            label: "PV-Leistung (kWp)",              defaultState: "hidden",
      visibilityTags: ["production"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Einspeise-Zählpunkt enthält." },
    { name: "heat_pump",               label: "Wärmepumpe vorhanden",            defaultState: "hidden",
      visibilityTags: ["consumption"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Verbraucher-Zählpunkt enthält." },
    { name: "electric_vehicle",        label: "E-Auto vorhanden",               defaultState: "hidden",
      visibilityTags: ["consumption"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Verbraucher-Zählpunkt enthält." },
    { name: "electric_vehicle_count",  label: "Anzahl E-Fahrzeuge",             defaultState: "hidden",
      visibilityTags: ["consumption", "ev"],
      visibilityHint: "Wird nur angezeigt, wenn ein Verbraucher-Zählpunkt vorhanden ist UND E-Auto vorhanden mit Ja beantwortet wurde." },
    { name: "electric_vehicle_annual_km", label: "Jahres-Kilometer (E-Fahrzeuge)", defaultState: "hidden",
      visibilityTags: ["consumption", "ev"],
      visibilityHint: "Wird nur angezeigt, wenn ein Verbraucher-Zählpunkt vorhanden ist UND E-Auto vorhanden mit Ja beantwortet wurde." },
    { name: "electric_hot_water",      label: "Warmwasser elektrisch (Boiler)",  defaultState: "hidden",
      visibilityTags: ["consumption"],
      visibilityHint: "Wird nur angezeigt, wenn der Antrag mindestens einen Verbraucher-Zählpunkt enthält." },
    // PROJ-44: Netzbetreiber-Vollmacht (siehe NETWORK_OPERATOR_AUTH_TEXT
    // in registration-form.tsx für den verbindlichen Wortlaut).
    { name: "network_operator_authorization", label: "Netzbetreiber-Vollmacht erteilen", defaultState: "hidden" },
  ],
  meteringPoint: [
    { name: "transformer",        label: "Transformator", defaultState: "hidden" },
    { name: "installation_number", label: "Anlagen-Nr.",  defaultState: "hidden" },
    { name: "installation_name",  label: "Anlagenname",  defaultState: "hidden" },
    // PROJ-45: Batterie + Wechselrichter (nur bei generation_type='pv' aktiv).
    { name: "battery_size_kwh",      label: "Größe Batterie (kWh)",        defaultState: "hidden",
      visibilityTags: ["production", "pv"],
      visibilityHint: "Wird nur bei Einspeise-Zählpunkten mit Erzeugungsform PV angezeigt." },
    { name: "inverter_manufacturer", label: "Hersteller Wechselrichter",  defaultState: "hidden",
      visibilityTags: ["production", "pv"],
      visibilityHint: "Wird nur bei Einspeise-Zählpunkten mit Erzeugungsform PV angezeigt." },
  ],
};

// PROJ-45: Erzeugungsform pro PRODUCTION-Zählpunkt. Default 'pv' im Backend.
export const GENERATION_TYPES = [
  { value: "pv",      label: "PV (Photovoltaik)" },
  { value: "hydro",   label: "Wasser" },
  { value: "wind",    label: "Wind" },
  { value: "biomass", label: "Biomasse" },
] as const;
export type GenerationType = typeof GENERATION_TYPES[number]["value"];

export function resolveFieldState(config: FieldConfig | undefined, fieldName: string, defaultState: FieldState): FieldState {
  return config?.[fieldName] ?? defaultState;
}

export interface LegalDocumentItem {
  id: string;
  title: string;
  url: string;
  required: boolean;
  sortOrder: number;
  isCentralPolicy: boolean;
}

export interface ConsentInput {
  title: string;
  url: string;
  isCentralPolicy: boolean;
}

export interface DocumentConsentView {
  id: string;
  title: string;
  url: string;
  isCentralPolicy: boolean;
  consentedAt: string;
  // PROJ-36: `explicit` for an actively checked confirmation, `informational`
  // for a non-required document the member only acknowledged by submitting.
  consentType?: "explicit" | "informational";
}

export interface RegistrationConfig {
  rcNumber: string;
  title: string;
  active: boolean;
  fieldConfig?: FieldConfig;
  introText?: string | null;
  sepaMandateEnabled?: boolean;
  showCentralPolicy?: boolean;
  legalDocuments?: LegalDocumentItem[];
  // PROJ-37: cooperative-shares config. The two value fields are only
  // present when the feature is enabled for this EEG.
  cooperativeSharesEnabled?: boolean;
  cooperativeRequiredShares?: number;
  cooperativeShareAmountCents?: number;
}

export type MemberType =
  | "private"
  | "sole_proprietor"
  | "farmer"
  | "municipality"
  | "company"
  | "association";

export interface MeteringPointRequest {
  meteringPoint: string;
  direction: "CONSUMPTION" | "PRODUCTION";
  participationFactor?: number;
  transformer?: string;
  installationNumber?: string;
  installationName?: string;
  // PROJ-39: abweichende Adresse je Zählpunkt. Entweder alle vier
  // gesetzt oder alle vier weggelassen.
  addressStreet?: string;
  addressStreetNumber?: string;
  addressZip?: string;
  addressCity?: string;
  // PROJ-45: Erzeugungsform + Batterie. generationType ist Pflicht für
  // PRODUCTION (Backend defaultet auf 'pv'), NULL für CONSUMPTION.
  // batterySizeKwh + inverterManufacturer sind nur sinnvoll wenn
  // generationType='pv' — Backend cleart sonst.
  generationType?: GenerationType;
  batterySizeKwh?: number;
  inverterManufacturer?: string;
}

export interface CreateApplicationRequest {
  rcNumber: string;
  memberType: MemberType;
  titel?: string;
  titelNach?: string;
  firstname?: string;
  lastname?: string;
  birthDate?: string;
  companyName?: string;
  uidNumber?: string;
  registerNumber?: string;
  email: string;
  phone?: string;
  residentStreet: string;
  residentStreetNumber: string;
  residentZip: string;
  residentCity: string;
  privacyAccepted: boolean;
  privacyVersion: string;
  accuracyConfirmed: boolean;
  iban: string;
  accountHolder: string;
  bankName?: string;
  sepaMandateAccepted: boolean;
  meteringPoints: MeteringPointRequest[];
  // configurable application-level fields
  membershipStartDate?: string;
  personsInHousehold?: number;
  consumptionPreviousYear?: number;
  consumptionForecast?: number;
  feedInForecast?: number;
  pvPowerKwp?: number;
  heatPump?: boolean | null;
  electricVehicle?: boolean | null;
  // PROJ-42: nur sinnvoll wenn electricVehicle === true; Server cleart sonst.
  electricVehicleCount?: number;
  electricVehicleAnnualKm?: number;
  electricHotWater?: boolean | null;
  // PROJ-37: Anzahl gezeichneter Genossenschaftsanteile
  cooperativeSharesCount?: number;
  // PROJ-44: Netzbetreiber-Vollmacht. Backend setzt `_at` automatisch
  // beim ersten true. Frontend sendet das Flag nur, wenn die EEG das
  // Feld konfiguriert hat.
  networkOperatorAuthorization?: boolean;
  turnstileToken?: string;
}

export interface ApplicationResponse {
  id: string;
  referenceNumber: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface SubmitResponse {
  id: string;
  referenceNumber: string;
  status: string;
  submittedAt: string;
}

// ---------- error handling ----------

export interface ApiError {
  code: string;
  message: string;
  fields?: Record<string, string>;
}

export class ApiResponseError extends Error {
  public readonly apiError: ApiError;

  constructor(apiError: ApiError) {
    super(apiError.message);
    this.name = "ApiResponseError";
    this.apiError = apiError;
  }
}

// ---------- request helper ----------

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({
      code: "internal_error",
      message: "Ein unbekannter Fehler ist aufgetreten.",
    }));
    throw new ApiResponseError(body as ApiError);
  }

  return res.json().catch(() => {
    throw new ApiResponseError({
      code: "internal_error",
      message: "Die Antwort des Servers konnte nicht verarbeitet werden.",
    });
  }) as Promise<T>;
}

// ---------- admin request helper (adds Bearer token when present) ----------

// Cooldown for the 401 → signIn("keycloak") redirect. Stored in
// sessionStorage so it survives the Keycloak roundtrip — without that, an
// unauthorized response right after a deploy (new backend pod not yet ready,
// stale browser bundle, etc.) puts the user into an infinite redirect loop:
// 401 → signIn → Keycloak → back to app → 401 → signIn → …
//
// With the cooldown, a second 401 within the window throws an
// ApiResponseError that callers surface as a visible error banner; the user
// either reloads manually or waits for the upstream to recover.
const AUTH_EXPIRED_COOLDOWN_MS = 30_000;
const AUTH_EXPIRED_SS_KEY = "auth:lastSignInTrigger";

function shouldTriggerSignIn(): boolean {
  if (typeof window === "undefined") return false;
  try {
    const raw = window.sessionStorage.getItem(AUTH_EXPIRED_SS_KEY);
    if (!raw) return true;
    return Date.now() - Number(raw) > AUTH_EXPIRED_COOLDOWN_MS;
  } catch {
    // sessionStorage can be blocked (private mode + strict settings); fail
    // open so we keep the legacy "redirect to login on 401" behaviour.
    return true;
  }
}

function markSignInTriggered(): void {
  if (typeof window === "undefined") return;
  try {
    window.sessionStorage.setItem(AUTH_EXPIRED_SS_KEY, String(Date.now()));
  } catch {
    // ignore — see shouldTriggerSignIn()
  }
}

async function adminRequest<T>(
  path: string,
  token: string | undefined,
  options?: RequestInit
): Promise<T> {
  const defaultHeaders: Record<string, string> = { "Content-Type": "application/json" };
  if (token) defaultHeaders["Authorization"] = `Bearer ${token}`;
  // Spread options FIRST, then override headers with a merged map; otherwise
  // a caller passing `headers: { "Content-Type": ... }` would silently drop
  // the Authorization header set above and trip a 401 at the auth middleware.
  const callerHeaders = (options?.headers as Record<string, string> | undefined) ?? {};
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: { ...defaultHeaders, ...callerHeaders },
  });

  if (res.status === 401 && typeof window !== "undefined") {
    // Backend rejected the JWT (revoked session, clock skew, Keycloak
    // restart, new pod still loading JWKS after a deploy). SessionRefreshGuard
    // listens for "auth:expired" and triggers signIn("keycloak"). The
    // cooldown above prevents an infinite redirect loop when the first
    // signIn does not actually fix the 401 (deploy still in progress, etc.).
    if (shouldTriggerSignIn()) {
      markSignInTriggered();
      window.dispatchEvent(new Event("auth:expired"));
      throw new ApiResponseError({
        code: "unauthorized",
        message: "Sitzung abgelaufen — Sie werden zur Anmeldung weitergeleitet.",
      });
    }
    // Within cooldown: do NOT redirect again. Surface a clear error the UI
    // can render so the user understands they need to act (typically: wait
    // a moment and reload manually).
    throw new ApiResponseError({
      code: "unauthorized",
      message: "Anmeldung erforderlich, aber automatische Weiterleitung wurde unterdrückt (Loop-Schutz). Bitte Seite neu laden.",
    });
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({
      code: "internal_error",
      message: "Ein unbekannter Fehler ist aufgetreten.",
    }));
    throw new ApiResponseError(body as ApiError);
  }

  if (res.status === 204) return undefined as T;

  return res.json().catch(() => {
    throw new ApiResponseError({
      code: "internal_error",
      message: "Die Antwort des Servers konnte nicht verarbeitet werden.",
    });
  }) as Promise<T>;
}

// ---------- public API ----------

export function getRegistrationConfig(
  rcNumber: string
): Promise<RegistrationConfig> {
  return request<RegistrationConfig>(
    `/api/public/registration/${encodeURIComponent(rcNumber)}`
  );
}

export function createApplication(
  data: CreateApplicationRequest
): Promise<ApplicationResponse> {
  return request<ApplicationResponse>("/api/public/applications", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export function submitApplication(id: string, consents?: ConsentInput[]): Promise<SubmitResponse> {
  return request<SubmitResponse>(`/api/public/applications/${id}/submit`, {
    method: "POST",
    body: consents && consents.length > 0 ? JSON.stringify({ consents }) : undefined,
  });
}

// ---------- admin API types ----------

export type ApplicationStatus =
  | "draft"
  | "submitted"
  | "email_confirmed"
  | "under_review"
  | "needs_info"
  | "approved"
  | "rejected"
  | "imported"
  | "import_failed"
  // PROJ-46: post-import statuses
  | "awaiting_bank_confirmation"
  | "ready_for_activation"
  | "activated";

export interface ApplicationListItem {
  id: string;
  referenceNumber: string;
  rcNumber: string;
  status: ApplicationStatus;
  memberType: MemberType;
  firstname?: string | null;
  lastname?: string | null;
  companyName?: string | null;
  email: string;
  submittedAt: string | null;
}

export interface ApplicationListResponse {
  items: ApplicationListItem[];
  page: number;
  pageSize: number;
  total: number;
}

export interface MeteringPointDetail {
  id: string;
  meteringPoint: string;
  direction: "CONSUMPTION" | "PRODUCTION";
  participationFactor: number;
  transformer?: string | null;
  installationNumber?: string | null;
  installationName?: string | null;
  addressStreet?: string | null;
  addressStreetNumber?: string | null;
  addressZip?: string | null;
  addressCity?: string | null;
  // PROJ-45: Erzeugungsform + Batterie. generationType ist NULL bei
  // CONSUMPTION, sonst pv/hydro/wind/biomass.
  generationType?: GenerationType | null;
  batterySizeKwh?: number | null;
  inverterManufacturer?: string | null;
}

export interface StatusLogEntry {
  fromStatus: string | null;
  toStatus: string;
  changedByUserId: string | null;
  reason: string | null;
  createdAt: string;
}

export interface AdminApplicationDetail {
  id: string;
  referenceNumber: string;
  rcNumber: string;
  status: ApplicationStatus;
  startedAt: string | null;
  submittedAt: string | null;
  approvedAt: string | null;
  rejectedAt: string | null;
  importedAt: string | null;
  memberType: MemberType;
  titel?: string | null;
  titelNach?: string | null;
  firstname?: string | null;
  lastname?: string | null;
  birthDate: string | null;
  companyName?: string | null;
  uidNumber?: string | null;
  registerNumber?: string | null;
  email: string;
  phone: string | null;
  residentStreet: string;
  residentStreetNumber: string;
  residentZip: string;
  residentCity: string;
  privacyAccepted: boolean;
  privacyVersion: string | null;
  privacyAcceptedAt: string | null;
  accuracyConfirmed: boolean;
  iban: string | null;
  accountHolder: string | null;
  sepaMandateAccepted: boolean;
  sepaMandateAcceptedAt: string | null;
  adminNote: string | null;
  einzugsart: string;
  bankName?: string | null;
  mandateReference?: string | null;
  mandateDate?: string | null;
  memberNumber?: string | null;
  emailConfirmedAt?: string | null;
  emailConfirmationPending?: boolean;
  needsInfoReason: string | null;
  targetParticipantId: string | null;
  importStartedAt: string | null;
  importFinishedAt: string | null;
  importErrorMessage: string | null;
  createdAt: string;
  updatedAt: string;
  meteringPoints: MeteringPointDetail[];
  statusLog: StatusLogEntry[];
  consents?: DocumentConsentView[];
  // PROJ-34: true when status='approved' AND import_started_at set > 2 min ago
  // AND import_finished_at is null. The admin UI renders the unstuck banner
  // only when this is true. Computed server-side; do not derive on the client.
  importStuck?: boolean;
  // PROJ-37: Anzahl gezeichneter Anteile + die zugehörige EEG-Config
  // (joined backend-side). Bei deaktiviertem Feature sind die zwei
  // Config-Felder undefined.
  cooperativeSharesCount?: number;
  cooperativeSharesEnabled?: boolean;
  cooperativeRequiredShares?: number;
  cooperativeShareAmountCents?: number;
  // PROJ-44: Netzbetreiber-Vollmacht (per-EEG konfigurierbar). Default false
  // bei Bestandsanträgen — Admin-UI rendert das Feld nur wenn TRUE oder wenn
  // die EEG es als optional/required konfiguriert hat.
  networkOperatorAuthorization?: boolean;
  networkOperatorAuthorizationAt?: string | null;
}

// PROJ-34: payload for POST /api/admin/applications/{id}/mark-imported-manually
export interface MarkImportedManuallyRequest {
  targetParticipantId: string;
  memberNumber: string;
  reason?: string;
}

export function markImportedManually(
  id: string,
  body: MarkImportedManuallyRequest,
  token?: string,
): Promise<AdminApplicationDetail> {
  return adminRequest<AdminApplicationDetail>(
    `/api/admin/applications/${id}/mark-imported-manually`,
    token,
    { method: "POST", body: JSON.stringify(body) },
  );
}

// PROJ-34: payload for POST /api/admin/applications/{id}/clear-import-lock
export interface ClearImportLockRequest {
  reason: string;
}

export function clearImportLock(
  id: string,
  body: ClearImportLockRequest,
  token?: string,
): Promise<AdminApplicationDetail> {
  return adminRequest<AdminApplicationDetail>(
    `/api/admin/applications/${id}/clear-import-lock`,
    token,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export interface AdminUpdateApplicationRequest {
  memberType?: MemberType;
  titel?: string;
  titelNach?: string;
  firstname?: string;
  lastname?: string;
  birthDate?: string;
  companyName?: string;
  uidNumber?: string;
  // PROJ-37: Anteils-Anzahl admin-seitig korrigierbar
  cooperativeSharesCount?: number;
  registerNumber?: string;
  email: string;
  phone?: string;
  residentStreet: string;
  residentStreetNumber: string;
  residentZip: string;
  residentCity: string;
  adminNote?: string;
  einzugsart?: string;
  bankName?: string;
  mandateReference?: string;
  mandateDate?: string;
  meteringPoints: MeteringPointRequest[];
}

export interface AdminUpdateResponse {
  id: string;
  updatedAt: string;
}

export interface ChangeStatusRequest {
  toStatus: string;
  reason?: string;
}

export interface ChangeStatusResponse {
  id: string;
  status: string;
}

export type SortColumn = "referenceNumber" | "name" | "email" | "rcNumber" | "status" | "submittedAt";
export type SortOrder = "asc" | "desc";

export interface ListApplicationsParams {
  status?: string;
  reference_number?: string;
  // Partial-match search across firstname, lastname and company_name.
  // The admin list column is itself a coalesce of those three, so the
  // filter has to too — otherwise typing a firstname or a company's
  // name yields nothing.
  name?: string;
  email?: string;
  rc_number?: string;
  submitted_from?: string;
  submitted_to?: string;
  page?: number;
  page_size?: number;
  sort?: SortColumn;
  order?: SortOrder;
}

// ---------- admin API ----------

export function listApplications(
  params: ListApplicationsParams,
  token?: string,
  signal?: AbortSignal,
): Promise<ApplicationListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);

  if (params.reference_number) qs.set("reference_number", params.reference_number);
  if (params.name) qs.set("name", params.name);
  if (params.email) qs.set("email", params.email);
  if (params.rc_number) qs.set("rc_number", params.rc_number);
  if (params.submitted_from) qs.set("submitted_from", `${params.submitted_from}T00:00:00Z`);
  if (params.submitted_to) qs.set("submitted_to", `${params.submitted_to}T23:59:59Z`);
  if (params.page) qs.set("page", String(params.page));
  if (params.page_size) qs.set("page_size", String(params.page_size));
  if (params.sort) qs.set("sort", params.sort);
  if (params.order) qs.set("order", params.order);
  const query = qs.toString();
  return adminRequest<ApplicationListResponse>(
    `/api/admin/applications${query ? `?${query}` : ""}`,
    token,
    { signal },
  );
}

export function getApplicationDetail(
  id: string,
  token?: string,
  signal?: AbortSignal,
): Promise<AdminApplicationDetail> {
  return adminRequest<AdminApplicationDetail>(`/api/admin/applications/${id}`, token, { signal });
}

export function updateApplication(
  id: string,
  data: AdminUpdateApplicationRequest,
  token?: string
): Promise<AdminUpdateResponse> {
  return adminRequest<AdminUpdateResponse>(`/api/admin/applications/${id}`, token, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

// Dedicated endpoint to replace only the admin_note column without touching
// any other application field. Use this from the admin note editor instead of
// updateApplication so the save cannot accidentally reset metering-point
// participation factors or other attributes the editor doesn't render.
export function setAdminNote(
  id: string,
  note: string,
  token?: string,
): Promise<void> {
  return adminRequest<void>(`/api/admin/applications/${id}/admin-note`, token, {
    method: "PATCH",
    body: JSON.stringify({ note }),
  });
}

export function changeApplicationStatus(
  id: string,
  req: ChangeStatusRequest,
  token?: string
): Promise<ChangeStatusResponse> {
  return adminRequest<ChangeStatusResponse>(
    `/api/admin/applications/${id}/status`,
    token,
    {
      method: "POST",
      body: JSON.stringify(req),
    }
  );
}

export interface ImportResponse {
  success: boolean;
  applicationId: string;
  status: ApplicationStatus;
  targetParticipantId?: string;
  message?: string;
  // PROJ-27: set when participant creation succeeded but the follow-up
  // tariffId assignment on the participant failed. Import is still treated
  // as successful (meter tariffs are persisted); admin needs to fix the
  // member tariff manually in the eegFaktura core.
  memberTariffWarning?: string;
}

// Import-time payload. memberNumber is now required and string-typed because
// the core's participantNumber column is VARCHAR; legitimate values include
// "A005", "M-12" etc. as well as plain "42".
export interface ImportRequestBody {
  memberNumber: string;
  tariffId?: string;
  meterTariffs?: Record<string, string>; // metering_point UUID -> tariff UUID
}

export function importApplication(
  id: string,
  body: ImportRequestBody,
  token?: string,
): Promise<ImportResponse> {
  return adminRequest<ImportResponse>(
    `/api/admin/applications/${id}/import`,
    token,
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  );
}

// PROJ-46 Stage D: batch activation-check. Asks the backend to query the
// core for all `ready_for_activation`-Anträge of the admin's tenants and
// transition those whose core participant is now ACTIVE to `activated`.
export interface ActivationCheckResult {
  checked: number;
  activated: number;
  errors?: string[];
}

export function checkActivations(token?: string): Promise<ActivationCheckResult> {
  return adminRequest<ActivationCheckResult>(
    `/api/admin/applications/check-activation`,
    token,
    { method: "POST" },
  );
}

// Ask the backend for the next free member-number suggestion. The backend
// derives this from the core's existing participantNumber values, detecting
// the dominant pattern (prefix + padding) so "A001, A002" suggests "A003"
// and "1, 2, 3" suggests "4". String-typed because the core accepts
// alphanumeric values.
export function fetchNextMemberNumber(
  applicationId: string,
  token?: string,
  signal?: AbortSignal,
): Promise<{ next_member_number: string }> {
  return adminRequest<{ next_member_number: string }>(
    `/api/admin/applications/${applicationId}/next-member-number`,
    token,
    { method: "GET", signal },
  );
}

// fetchTariffs: signal optional so the import dialog can cancel a stale
// fetch when the user closes/reopens it on a different application.

// PROJ-27: Tariff catalogue entry as returned by GET /api/admin/tariffs.
// Subset of the core's GET /eeg/tariff response — only fields we need for
// the selection dialog.
export interface Tariff {
  id: string;
  type: "EEG" | "VZP" | "EZP" | "AKONTO";
  name: string;
  centPerKWh: number;
  discount: number;
  useVat: boolean;
  vatInPercent: number;
  inactiveSince?: string | null;
}

export function fetchTariffs(
  rcNumber: string,
  token?: string,
  signal?: AbortSignal,
): Promise<{ tariffs: Tariff[] }> {
  return adminRequest<{ tariffs: Tariff[] }>(
    `/api/admin/tariffs?rcNumber=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "GET", signal },
  );
}

// PROJ-30: reset an imported application back to approved so it can be
// re-imported after the participant was deleted in the eegFaktura core.
export function resetImportApplication(
  id: string,
  reason: string,
  token?: string,
): Promise<AdminApplicationDetail> {
  return adminRequest<AdminApplicationDetail>(
    `/api/admin/applications/${id}/reset-import`,
    token,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reason }),
    }
  );
}

export function syncEntrypoints(token?: string): Promise<void> {
  return adminRequest<void>("/api/admin/sync", token, { method: "POST" });
}

// PROJ-40: reassign an application to a different EEG during admin review.
// Admin must be authorized for both source and target (or be a superuser).
// The reference number is regenerated on the target's per-year counter.
export function reassignApplicationToEEG(
  id: string,
  targetRcNumber: string,
  reason: string,
  token?: string,
): Promise<AdminApplicationDetail> {
  return adminRequest<AdminApplicationDetail>(
    `/api/admin/applications/${id}/reassign-eeg`,
    token,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ targetRcNumber, reason }),
    }
  );
}

export function resendMemberConfirmation(id: string, token?: string): Promise<void> {
  return adminRequest<void>(`/api/admin/applications/${id}/resend-confirmation`, token, { method: "POST" });
}

export function deleteApplication(id: string, token?: string): Promise<void> {
  return adminRequest<void>(`/api/admin/applications/${id}`, token, { method: "DELETE" });
}

export function deleteDraftApplications(
  token?: string,
  rcNumber?: string,
): Promise<{ deleted: number }> {
  const qs = rcNumber ? `?rc_number=${encodeURIComponent(rcNumber)}` : "";
  return adminRequest<{ deleted: number }>(
    `/api/admin/applications/drafts${qs}`,
    token,
    { method: "DELETE" },
  );
}

export function getFieldConfig(rcNumber: string, token?: string): Promise<{ fieldConfig: AdminFieldConfig }> {
  return adminRequest<{ fieldConfig: AdminFieldConfig }>(`/api/admin/settings/fields?rc_number=${encodeURIComponent(rcNumber)}`, token);
}

export function saveFieldConfig(rcNumber: string, config: AdminFieldConfig, token?: string): Promise<void> {
  return adminRequest<void>(
    `/api/admin/settings/fields?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "PUT", body: JSON.stringify(config) }
  );
}

export interface EEGSettings {
  rcNumber: string;
  registrationActive?: boolean;
  // The eight fields below are Core-mastered (PROJ-32). Frontend
  // displays them read-only and never sends them in PUT /settings/eeg
  // (the backend ignores them if a legacy client still does).
  eegId: string | null;
  eegName: string | null;
  eegStreet: string | null;
  eegStreetNumber: string | null;
  eegZip: string | null;
  eegCity: string | null;
  creditorId: string | null;
  contactEmail?: string | null;
  lastSyncedFromCoreAt?: string | null;
  // PROJ-33: timestamp of the last successful logo fetch from the
  // eegFaktura-billing service. null until the first successful sync.
  eegLogoSyncedAt?: string | null;
  sepaMandateEnabled: boolean;
  useCompanySEPAMandate: boolean;
  showCentralPolicy?: boolean;
  memberNumberStart?: number;
  requireEmailConfirmation?: boolean;
  // PROJ-37: Genossenschaftsanteile-Konfig pro EEG.
  cooperativeSharesEnabled?: boolean;
  cooperativeRequiredShares?: number;
  cooperativeShareAmountCents?: number;
}

// Editable subset accepted by PUT /api/admin/settings/eeg. Everything
// else is either Core-mastered (synced fields) or written via a
// dedicated endpoint.
export interface EEGSettingsSavePayload {
  registrationActive?: boolean;
  sepaMandateEnabled: boolean;
  useCompanySEPAMandate: boolean;
  requireEmailConfirmation?: boolean;
  showCentralPolicy?: boolean;
  memberNumberStart?: number;
  // PROJ-37: Toggle + Pflichtanteils-Anzahl + Anteilswert in Cents.
  cooperativeSharesEnabled?: boolean;
  cooperativeRequiredShares?: number;
  cooperativeShareAmountCents?: number;
}

export interface ConfirmEmailResponse {
  eegName?: string;
  eegContactEmail?: string;
  alreadyConfirmed?: boolean;
}

export async function confirmEmail(token: string): Promise<ConfirmEmailResponse> {
  const res = await fetch(`${API_URL}/api/public/applications/confirm-email`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
  if (!res.ok) {
    let message = "Der Bestätigungs-Link ist ungültig oder abgelaufen.";
    try {
      const body = await res.json();
      if (body?.message) message = String(body.message);
    } catch {
      /* keep default */
    }
    throw new Error(message);
  }
  return res.json();
}

export function resendEmailConfirmation(applicationId: string, token?: string): Promise<void> {
  return adminRequest<void>(
    `/api/admin/applications/${encodeURIComponent(applicationId)}/resend-email-confirmation`,
    token,
    { method: "POST" }
  );
}

export function getEEGSettings(rcNumber: string, token?: string): Promise<EEGSettings> {
  return adminRequest<EEGSettings>(`/api/admin/settings/eeg?rc_number=${encodeURIComponent(rcNumber)}`, token);
}

export function saveEEGSettings(rcNumber: string, settings: EEGSettingsSavePayload, token?: string): Promise<void> {
  return adminRequest<void>(
    `/api/admin/settings/eeg?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "PUT", body: JSON.stringify(settings) }
  );
}

// PROJ-32: EEG master-data sync from the eegFaktura core.

export interface EEGSettingsFieldDiff {
  field: string;
  label: string;
  localValue: string;
  coreValue: string;
}

export interface EEGSettingsComparisonResponse {
  coreReachable: boolean;
  coreUnreachableError?: string;
  inSync: boolean;
  differingFields?: EEGSettingsFieldDiff[];
  lastSyncedAt?: string | null;
  // PROJ-33: set by POST /sync when the master-data sync succeeded but
  // the follow-up logo fetch did not (oversize, unsupported MIME, etc.).
  logoSyncWarning?: string;
  logoSyncedAt?: string | null;
}

// Fetches the EEG logo bytes via the admin API (which requires a Keycloak
// bearer token in the Authorization header — therefore `<img src>` can't
// load this URL directly, since the browser wouldn't attach the token).
// Returns an Object URL the caller can drop into an <img> tag, plus a
// dispose() callback to release the blob when the component unmounts.
// On 404 ("no logo synced yet") returns null without throwing.
export async function fetchEEGLogoBlob(
  rcNumber: string,
  token?: string,
): Promise<{ objectURL: string; dispose: () => void } | null> {
  const headers: Record<string, string> = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(
    `${API_URL}/api/admin/settings/eeg/logo?rc_number=${encodeURIComponent(rcNumber)}`,
    { headers },
  );
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`logo fetch failed: ${res.status}`);
  const blob = await res.blob();
  const objectURL = URL.createObjectURL(blob);
  return { objectURL, dispose: () => URL.revokeObjectURL(objectURL) };
}

export function compareEEGSettingsWithCore(rcNumber: string, token?: string): Promise<EEGSettingsComparisonResponse> {
  return adminRequest<EEGSettingsComparisonResponse>(
    `/api/admin/settings/eeg/core-comparison?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
  );
}

export function syncEEGSettingsFromCore(rcNumber: string, token?: string): Promise<EEGSettingsComparisonResponse> {
  return adminRequest<EEGSettingsComparisonResponse>(
    `/api/admin/settings/eeg/sync?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "POST" },
  );
}

export function getIntroText(rcNumber: string, token?: string): Promise<{ rcNumber: string; introText: string | null }> {
  return adminRequest(`/api/admin/settings/intro-text?rc_number=${encodeURIComponent(rcNumber)}`, token);
}

export function saveIntroText(rcNumber: string, introText: string | null, token?: string): Promise<void> {
  return adminRequest<void>(
    `/api/admin/settings/intro-text?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "PUT", body: JSON.stringify({ introText }) }
  );
}

export interface ApiKeyStatus {
  active: boolean;
  lastGeneratedAt: string | null;
}

export function getApiKeyStatus(rcNumber: string, token?: string): Promise<ApiKeyStatus> {
  return adminRequest<ApiKeyStatus>(`/api/admin/settings/api-key?rc_number=${encodeURIComponent(rcNumber)}`, token);
}

export function generateApiKey(rcNumber: string, token?: string): Promise<{ apiKey: string }> {
  return adminRequest<{ apiKey: string }>(
    `/api/admin/settings/api-key?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "POST" }
  );
}

export async function downloadApplicationExcel(id: string, token?: string): Promise<{ blob: Blob; filename: string }> {
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${API_URL}/api/admin/applications/${id}/export/excel`, { headers });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ code: "internal_error", message: "Download fehlgeschlagen." }));
    throw new ApiResponseError(body as ApiError);
  }
  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = disposition.match(/filename="([^"]+)"/);
  const filename = match ? match[1] : `${id}.xlsx`;
  return { blob, filename };
}

export async function downloadApprovalPDF(id: string, token?: string): Promise<{ blob: Blob; filename: string }> {
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${API_URL}/api/admin/applications/${id}/approval-pdf`, { headers });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ code: "internal_error", message: "Download fehlgeschlagen." }));
    throw new ApiResponseError(body as ApiError);
  }
  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = disposition.match(/filename="([^"]+)"/);
  const filename = match ? match[1] : `beitrittsbestaetigung-${id}.pdf`;
  return { blob, filename };
}

export function revokeApiKey(rcNumber: string, token?: string): Promise<void> {
  return adminRequest<void>(
    `/api/admin/settings/api-key?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "DELETE" }
  );
}

// ---------- legal documents admin API ----------

export interface CreateLegalDocumentRequest {
  title: string;
  url: string;
  required: boolean;
}

export function listLegalDocuments(rcNumber: string, token?: string): Promise<LegalDocumentItem[]> {
  return adminRequest<LegalDocumentItem[]>(
    `/api/admin/legal-documents?rc_number=${encodeURIComponent(rcNumber)}`,
    token
  );
}

export function createLegalDocument(rcNumber: string, req: CreateLegalDocumentRequest, token?: string): Promise<LegalDocumentItem> {
  return adminRequest<LegalDocumentItem>(
    `/api/admin/legal-documents?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "POST", body: JSON.stringify(req) }
  );
}

export function updateLegalDocument(id: string, req: CreateLegalDocumentRequest, token?: string): Promise<LegalDocumentItem> {
  return adminRequest<LegalDocumentItem>(
    `/api/admin/legal-documents/${id}`,
    token,
    { method: "PUT", body: JSON.stringify(req) }
  );
}

export function deleteLegalDocument(id: string, token?: string): Promise<void> {
  return adminRequest<void>(`/api/admin/legal-documents/${id}`, token, { method: "DELETE" });
}

export function reorderLegalDocuments(rcNumber: string, ids: string[], token?: string): Promise<void> {
  return adminRequest<void>(
    `/api/admin/legal-documents/reorder?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "PUT", body: JSON.stringify({ ids }) }
  );
}


// ---------- bulk actions admin API ----------

export type BulkAction = "approve" | "reject" | "under_review";

export interface BulkActionResponse {
  succeeded: string[];
  skipped: string[];
}

export function bulkAction(
  action: BulkAction,
  ids: string[],
  reason: string,
  token?: string
): Promise<BulkActionResponse> {
  return adminRequest<BulkActionResponse>(
    "/api/admin/applications/bulk-action",
    token,
    { method: "POST", body: JSON.stringify({ action, ids, reason }) }
  );
}
