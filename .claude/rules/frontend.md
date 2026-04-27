---
paths:
  - "src/components/**"
  - "src/app/**/page.tsx"
  - "src/app/**/layout.tsx"
  - "src/hooks/**"
---

# Frontend Development Rules

## shadcn/ui First (MANDATORY)
- Before creating ANY UI component, check if shadcn/ui has it: `ls src/components/ui/`
- NEVER create custom implementations of: Button, Input, Select, Checkbox, Switch, Dialog, Modal, Alert, Toast, Table, Tabs, Card, Badge, Dropdown, Popover, Tooltip, Navigation, Sidebar, Breadcrumb
- If a shadcn component is missing, install it: `npx shadcn@latest add <name> --yes`
- Custom components are ONLY for business-specific compositions that internally use shadcn primitives

## Import Pattern
```tsx
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
```

## Component Standards
- Use Tailwind CSS exclusively (no inline styles, no CSS modules)
- All components must be responsive (mobile 375px, tablet 768px, desktop 1440px)
- Implement loading states, error states, and empty states
- Use semantic HTML and ARIA labels for accessibility
- Keep components small and focused
- Use TypeScript interfaces for all props

## Hint / Tooltip Pattern (MANDATORY)
When a field needs explanatory text, use a Tooltip with an `Info` icon next to the label — never a `<p>` block below the input:
```tsx
import { Info } from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

<Label className="flex items-center gap-1">
  Feldname
  <TooltipProvider>
    <Tooltip>
      <TooltipTrigger type="button" className="cursor-help">
        <Info className="h-3.5 w-3.5 text-muted-foreground" />
      </TooltipTrigger>
      <TooltipContent className="max-w-60">
        Erklärungstext hier.
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>
</Label>
```
Reference implementation: `src/components/metering-point-fields.tsx` (Teilnahmefaktor field).

## Auth (NextAuth + Keycloak)
- Admin pages are protected via the `(admin)` layout — do not add global middleware
- Use `useSession()` to get the access token for admin API calls
- `session.accessToken` is the Keycloak JWT — pass it as `Authorization: Bearer` to the backend
