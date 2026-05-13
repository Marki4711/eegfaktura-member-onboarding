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
}

export const CONFIGURABLE_FIELDS: {
  application: ConfigurableField[];
  meteringPoint: ConfigurableField[];
} = {
  application: [
    { name: "phone",                   label: "Telefonnummer",                   defaultState: "optional" },
    { name: "birth_date",              label: "Geburtsdatum",                    defaultState: "optional" },
    { name: "membership_start_date",   label: "Aktiv am (Beitrittsdatum)",       defaultState: "hidden"   },
    { name: "persons_in_household",    label: "Anzahl Personen im Haushalt",     defaultState: "hidden"   },
    { name: "consumption_previous_year", label: "Verbrauch Vorjahr (kWh)",       defaultState: "hidden"   },
    { name: "consumption_forecast",    label: "Verbrauch Prognose (kWh)",        defaultState: "hidden"   },
    { name: "feed_in_forecast",        label: "Einspeisung Prognose (kWh)",      defaultState: "hidden"   },
    { name: "pv_power_kwp",            label: "PV-Leistung (kWp)",              defaultState: "hidden"   },
    { name: "heat_pump",               label: "Wärmepumpe vorhanden",            defaultState: "hidden"   },
    { name: "electric_vehicle",        label: "E-Auto vorhanden",               defaultState: "hidden"   },
    { name: "electric_hot_water",      label: "Warmwasser elektrisch (Boiler)",  defaultState: "hidden"   },
  ],
  meteringPoint: [
    { name: "transformer",        label: "Transformator", defaultState: "hidden" },
    { name: "installation_number", label: "Anlagen-Nr.",  defaultState: "hidden" },
    { name: "installation_name",  label: "Anlagenname",  defaultState: "hidden" },
  ],
};

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
}

export interface CreateApplicationRequest {
  rcNumber: string;
  memberType: MemberType;
  titel?: string;
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
  electricHotWater?: boolean | null;
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
  | "under_review"
  | "needs_info"
  | "approved"
  | "rejected"
  | "imported"
  | "import_failed";

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
  memberNumber?: number | null;
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
}

export interface AdminUpdateApplicationRequest {
  memberType?: MemberType;
  titel?: string;
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
  adminNote?: string;
  einzugsart?: string;
  bankName?: string;
  mandateReference?: string;
  mandateDate?: string;
  memberNumber?: number;
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
  lastname?: string;
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
  token?: string
): Promise<ApplicationListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);

  if (params.reference_number) qs.set("reference_number", params.reference_number);
  if (params.lastname) qs.set("lastname", params.lastname);
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
    token
  );
}

export function getApplicationDetail(id: string, token?: string): Promise<AdminApplicationDetail> {
  return adminRequest<AdminApplicationDetail>(`/api/admin/applications/${id}`, token);
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

// PROJ-27: Tariff selection sent with the import call. All fields optional;
// an empty body falls back to the legacy "no tariffs" behaviour.
export interface ImportRequestBody {
  tariffId?: string;
  meterTariffs?: Record<string, string>; // metering_point UUID -> tariff UUID
}

export function importApplication(
  id: string,
  body?: ImportRequestBody,
  token?: string,
): Promise<ImportResponse> {
  return adminRequest<ImportResponse>(
    `/api/admin/applications/${id}/import`,
    token,
    {
      method: "POST",
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    }
  );
}

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

export function fetchTariffs(rcNumber: string, token?: string): Promise<{ tariffs: Tariff[] }> {
  return adminRequest<{ tariffs: Tariff[] }>(
    `/api/admin/tariffs?rcNumber=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "GET" }
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

export function resendMemberConfirmation(id: string, token?: string): Promise<void> {
  return adminRequest<void>(`/api/admin/applications/${id}/resend-confirmation`, token, { method: "POST" });
}

export function deleteApplication(id: string, token?: string): Promise<void> {
  return adminRequest<void>(`/api/admin/applications/${id}`, token, { method: "DELETE" });
}

export function deleteDraftApplications(token?: string): Promise<{ deleted: number }> {
  return adminRequest<{ deleted: number }>("/api/admin/applications/drafts", token, { method: "DELETE" });
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
  eegId: string | null;
  eegName: string | null;
  eegStreet: string | null;
  eegStreetNumber: string | null;
  eegZip: string | null;
  eegCity: string | null;
  creditorId: string | null;
  sepaMandateEnabled: boolean;
  useCompanySEPAMandate: boolean;
  showCentralPolicy?: boolean;
  memberNumberStart?: number;
}

export function getEEGSettings(rcNumber: string, token?: string): Promise<EEGSettings> {
  return adminRequest<EEGSettings>(`/api/admin/settings/eeg?rc_number=${encodeURIComponent(rcNumber)}`, token);
}

export function saveEEGSettings(rcNumber: string, settings: Omit<EEGSettings, "rcNumber">, token?: string): Promise<void> {
  return adminRequest<void>(
    `/api/admin/settings/eeg?rc_number=${encodeURIComponent(rcNumber)}`,
    token,
    { method: "PUT", body: JSON.stringify(settings) }
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
