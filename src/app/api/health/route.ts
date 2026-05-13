// Lightweight health endpoint for K8s probes. Returns 200 unconditionally
// once the Next.js server is up — does NOT render the public landing page,
// does NOT call the backend. This decouples the frontend pod's lifecycle
// from backend availability: a backend outage no longer cascades into a
// frontend liveness restart loop.
//
// Pointed to from helm/member-onboarding/templates/frontend.yaml for both
// liveness and readiness.

export const dynamic = "force-dynamic";

export function GET() {
  return new Response(JSON.stringify({ status: "alive" }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}
