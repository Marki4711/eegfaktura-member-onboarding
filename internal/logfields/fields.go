// Package logfields centralises the slog field-keys used across the
// application. Free-form strings like "rc" vs "rc_number", "user_id" vs
// "admin_user_id" cause silent drift that breaks log-shipper filter rules
// in production. New code should import these constants instead of typing
// literals; existing call-sites can migrate opportunistically.
//
// Conventions:
//   - snake_case (consistent with K8s/Postgres column names)
//   - singular nouns where possible
//   - resource-ID fields end in "_id" (e.g. application_id, job_id)
//   - PII-derived fields use a "_domain"/"_hash"/"_count" suffix so it's
//     obvious at-a-glance that they're not the raw value (which would
//     violate .claude/rules/security.md "no IBAN/email/phone/name in logs")
package logfields

// Resource IDs
const (
	ApplicationID = "application_id"
	JobID         = "job_id"
	ConfigID      = "config_id"
	RCNumber      = "rc_number"
	EEGID         = "eeg_id"
	TenantID      = "tenant_id"
	RequestID     = "request_id"
	AdminUserID   = "admin_user_id"
	MemberType    = "member_type"
	Status        = "status"
	PluginType    = "plugin_type"
)

// Audit-trail markers. `Classification` carries a fixed vocabulary that
// log-shippers can route on (e.g. "sensitive-export" → DSGVO-archive).
const (
	Classification = "classification"

	ClassSensitiveExport = "sensitive-export" // data-export with IBAN/birth_date columns
	ClassPIIRead         = "pii-read"         // admin opens an application detail
	ClassPIIExport       = "pii-export"       // admin downloads single-application Excel/PDF
)

// Sanitised PII proxies — never log the raw value.
const (
	ToDomain     = "to_domain"      // @-suffix of recipient email
	EmailDomain  = "email_domain"   // generic email-domain field
	BirthYear    = "birth_year"     // year-only proxy for birth_date
	IBANCountry  = "iban_country"   // 2-letter prefix of IBAN
)

// Operation timing + counts (helpful for grep/aggregation).
const (
	DurationMs       = "duration_ms"
	ApplicationCount = "application_count"
	Processed        = "processed"
	Total            = "total"
	RetryCount       = "retry_count"
)
