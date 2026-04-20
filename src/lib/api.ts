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

export interface RegistrationConfig {
  rcNumber: string;
  eegId: string;
  title: string;
  active: boolean;
}

export interface MeteringPointRequest {
  meteringPoint: string;
  direction: "CONSUMPTION" | "PRODUCTION";
}

export interface CreateApplicationRequest {
  rcNumber: string;
  firstname: string;
  lastname: string;
  birthDate?: string;
  email: string;
  phone?: string;
  residentStreet: string;
  residentStreetNumber: string;
  residentZip: string;
  residentCity: string;
  residentCountry: string;
  privacyAccepted: boolean;
  privacyVersion: string;
  accuracyConfirmed: boolean;
  iban: string;
  accountHolder: string;
  sepaMandateAccepted: boolean;
  meteringPoints: MeteringPointRequest[];
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

export function submitApplication(id: string): Promise<SubmitResponse> {
  return request<SubmitResponse>(`/api/public/applications/${id}/submit`, {
    method: "POST",
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
  eegId: string;
  status: ApplicationStatus;
  firstname: string;
  lastname: string;
  email: string;
  submittedAt: string | null;
  meteringPoints: string[];
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
  eegId: string;
  rcNumber: string;
  status: ApplicationStatus;
  startedAt: string | null;
  submittedAt: string | null;
  approvedAt: string | null;
  rejectedAt: string | null;
  importedAt: string | null;
  firstname: string;
  lastname: string;
  birthDate: string | null;
  email: string;
  phone: string | null;
  residentStreet: string;
  residentStreetNumber: string;
  residentZip: string;
  residentCity: string;
  residentCountry: string;
  privacyAccepted: boolean;
  privacyVersion: string | null;
  privacyAcceptedAt: string | null;
  accuracyConfirmed: boolean;
  iban: string | null;
  accountHolder: string | null;
  sepaMandateAccepted: boolean;
  sepaMandateAcceptedAt: string | null;
  adminNote: string | null;
  needsInfoReason: string | null;
  targetParticipantId: string | null;
  importStartedAt: string | null;
  importFinishedAt: string | null;
  importErrorMessage: string | null;
  createdAt: string;
  updatedAt: string;
  meteringPoints: MeteringPointDetail[];
  statusLog: StatusLogEntry[];
}

export interface AdminUpdateApplicationRequest {
  firstname: string;
  lastname: string;
  birthDate?: string;
  email: string;
  phone?: string;
  residentStreet: string;
  residentStreetNumber: string;
  residentZip: string;
  residentCity: string;
  residentCountry: string;
  adminNote?: string;
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

export interface ListApplicationsParams {
  status?: string;
  eeg_id?: string;
  reference_number?: string;
  lastname?: string;
  email?: string;
  metering_point?: string;
  submitted_from?: string;
  submitted_to?: string;
  page?: number;
  page_size?: number;
}

// ---------- admin API ----------

export function listApplications(
  params: ListApplicationsParams
): Promise<ApplicationListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.eeg_id) qs.set("eeg_id", params.eeg_id);
  if (params.reference_number) qs.set("reference_number", params.reference_number);
  if (params.lastname) qs.set("lastname", params.lastname);
  if (params.email) qs.set("email", params.email);
  if (params.metering_point) qs.set("metering_point", params.metering_point);
  if (params.submitted_from) qs.set("submitted_from", `${params.submitted_from}T00:00:00Z`);
  if (params.submitted_to) qs.set("submitted_to", `${params.submitted_to}T23:59:59Z`);
  if (params.page) qs.set("page", String(params.page));
  if (params.page_size) qs.set("page_size", String(params.page_size));
  const query = qs.toString();
  return request<ApplicationListResponse>(
    `/api/admin/applications${query ? `?${query}` : ""}`
  );
}

export function getApplicationDetail(id: string): Promise<AdminApplicationDetail> {
  return request<AdminApplicationDetail>(`/api/admin/applications/${id}`);
}

export function updateApplication(
  id: string,
  data: AdminUpdateApplicationRequest
): Promise<AdminUpdateResponse> {
  return request<AdminUpdateResponse>(`/api/admin/applications/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export function changeApplicationStatus(
  id: string,
  req: ChangeStatusRequest
): Promise<ChangeStatusResponse> {
  return request<ChangeStatusResponse>(
    `/api/admin/applications/${id}/status`,
    {
      method: "POST",
      body: JSON.stringify(req),
    }
  );
}
