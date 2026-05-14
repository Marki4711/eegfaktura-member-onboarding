// Package metrics exposes Prometheus counters and histograms for the
// service-level events that operators care about: how many applications
// arrive, how many imports succeed/fail, how many mails are sent vs
// dropped, and HTTP latency.
//
// All metrics are registered against the default registry, so the
// `go_*` and `process_*` collectors come for free. The /metrics handler
// is intended to be exposed on a separate port (see cmd/server/main.go)
// so it is reachable from inside the cluster (Prometheus pod) but NOT
// from the public ingress.
package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "eegfaktura_mo"

// Counter / histogram registry. Names follow the Prometheus convention
// `<namespace>_<subject>_<unit>_total` (counters) or `..._seconds`
// (histograms).
var (
	// ApplicationsSubmittedTotal — fires once when a draft transitions to
	// submitted. Indicates incoming public-form load. Sum over a window =
	// applications received.
	ApplicationsSubmittedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "applications_submitted_total",
		Help:      "Number of applications that have transitioned to submitted.",
	})

	// ImportsTotal — fires once per import attempt. Labels:
	//   result: success | failed
	// `failed` includes both core errors and bookkeeping errors.
	ImportsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "imports_total",
		Help:      "Number of import attempts to the eegFaktura core, by result.",
	}, []string{"result"})

	// MailSentTotal — fires once per outgoing mail. Labels:
	//   kind:   member_confirmation | eeg_notification | eeg_approval | resend
	//   result: success | failed
	MailSentTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "mail_sent_total",
		Help:      "Outgoing mails by template kind and SMTP result.",
	}, []string{"kind", "result"})

	// RateLimitHitsTotal — fires when the public-submit rate limiter blocks
	// a request. High values point at scraper traffic or a misconfigured
	// client.
	RateLimitHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "rate_limit_hits_total",
		Help:      "Public-submit rate-limit denials.",
	})

	// MemberNumberLookupTotal — covers the GET /next-member-number flow
	// (PROJ-X: assign member number at import time). Labels:
	//   result: success | core_error
	MemberNumberLookupTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "member_number_lookups_total",
		Help:      "next-member-number lookups against the core, by result.",
	}, []string{"result"})

	// HTTPRequestDurationSeconds — HTTP latency histogram. Labels:
	//   method
	//   status_class: 2xx | 3xx | 4xx | 5xx
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration, by method and status class.",
		Buckets:   prometheus.DefBuckets, // 0.005, 0.01, 0.025, … 10
	}, []string{"method", "status_class"})
)

// StatusClass maps a status code to "2xx" / "3xx" / "4xx" / "5xx" / "other"
// so the histogram cardinality stays small.
func StatusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "other"
	}
}

// statusClassFromString is here to keep callers concise where the status
// is already stringified for logging.
func statusClassFromString(s string) string {
	n, err := strconv.Atoi(s)
	if err != nil {
		return "other"
	}
	return StatusClass(n)
}

var _ = statusClassFromString // reserved for future callers
