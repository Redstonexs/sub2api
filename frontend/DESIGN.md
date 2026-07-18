# Sub2API Frontend Design System

## 1. Atmosphere & Identity

Sub2API is a warm, editorial control surface: dense enough for operational work,
but never cold or blue-black. Its signature is warm tonal depth. Cream, clay,
and charcoal surfaces are separated with restrained warm shifts, low-contrast
borders, and low-alpha brown shadows rather than cool navy fills.

## 2. Color

### Palette

| Role | Tailwind token | Light | Dark | Usage |
| --- | --- | --- | --- | --- |
| Canvas | gray-50 | #FAF9F5 | #1A1814 | App background |
| Surface | white | #FFFFFF | #2A2722 | Cards, menus, inputs |
| Surface/subtle | gray-100 | #F3F1EA | #211F1A | Table headers, quiet panels |
| Surface/elevated | accent-50 | #F8F6F0 | #39352E | Popovers and selected layers |
| Text/primary | gray-900 | #1A1815 | #F8F6F0 | Headings and values |
| Text/secondary | gray-600 | #605A4E | #DAD5C9 | Labels and supporting copy |
| Border | gray-200 | #E7E3D8 | #39352E | Cards, tables, inputs |
| Accent | primary-600 | #B05C40 | #B05C40 | Primary actions and active states |
| Accent/hover | primary-700 | #8F4A34 | #8F4A34 | Pressed and hover states |
| Success | success-600 | #5D9A51 | #8ED381 | Positive status |
| Warning | warning-600 | #BE8927 | #F9BE5E | Caution status |
| Danger | danger-600 | #BE5448 | #DD877B | Destructive status |
| Info | info-600 | #376FC0 | #7EA4DD | Informational status |
| Overlay black | black | #000000 | #000000 | Transient scrims only |

gray, slate, zinc, neutral, and stone all resolve to the same warm gray ramp.
red/rose, orange/yellow, green, cyan, blue, and purple utility families resolve
to documented brick, terracotta, fern, aqua, denim, and plum ramps, so ordinary
surface utilities cannot reintroduce the upstream palette.

### Rules

- Shared components use Tailwind utility tokens rather than raw RGB surface
  values, so the configured warm palette stays authoritative.
- Use semantic aliases (success, warning, danger, info) for new status UI; use
  the family mappings only when retaining an existing utility contract.
- Pure black is reserved for translucent modal and image-preview scrims; it is
  not a card, table, tab, or page-surface color.
- Payment-provider brand colors are a deliberate exception and remain confined
  to their branded payment controls.

## 3. Typography

| Level | Size | Weight | Line height | Usage |
| --- | --- | --- | --- | --- |
| Display | 30px | 700 | 1.2 | Public/setup headings |
| Page title | 24px | 700 | 1.25 | Admin and user page titles |
| Section title | 20px | 600 | 1.4 | Card and settings sections |
| Body | 16px | 400 | 1.5 | Default form and prose text |
| Table/label | 14px | 400-500 | 1.4 | Tables, filters, controls |
| Caption | 12px | 500 | 1.4 | Metadata and helper text |

- Primary: Inter, Inter Variable, system and CJK sans-serif fallbacks.
- Display: Fraunces, Fraunces Variable, Georgia and CJK serif fallbacks.
- Mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas.

## 4. Spacing & Layout

- Base unit: 4px.
- Common steps: 4, 8, 12, 16, 20, 24, 32, 40, 48, and 64px.
- Maximum content width: 1280px (max-w-7xl).
- Breakpoints: 640, 768, 1024, 1280, and 1536px.
- Admin table pages use TablePageLayout: action/filter/pagination regions do
  not scroll; .table-wrapper is the named table scroll owner.

## 5. Components

### App shell and cards

- **Structure:** app canvas, sticky header, content area, optional sidebar/card.
- **States:** default, hover, focus-visible, disabled, loading, and empty.
- **Surface:** warm canvas with white/dark-charcoal cards and warm borders.
- **Accessibility:** visible primary focus ring; semantic buttons and labels.

### Data table

- **Structure:** TablePageLayout -> card -> .table-wrapper -> semantic table.
- **States:** header, row default, hover, selected, loading, and empty.
- **Spacing:** header and cell padding are 20px horizontally and 16px vertically.
- **Layout:** table wrapper owns both axes of overflow; sticky columns use the
  same warm surface token as their row or header. Shared DataTable switches to
  cards below 1536px so fixed edge columns cannot hide the middle fields.
- **Accessibility:** readable header contrast, keyboard-reachable row actions,
  and no color-only status signal.

### Filters, buttons, inputs, and tabs

- **Structure:** wrapping action/filter clusters, shared .btn and .input
  primitives, and tab navigation.
- **States:** default, hover, active, focus-visible, disabled, loading, error.
- **Motion:** 150-300ms color/transform transitions only.

## 6. Motion & Interaction

| Type | Duration | Easing | Usage |
| --- | --- | --- | --- |
| Micro | 150ms | ease-out | Table-row and icon hover |
| Standard | 200ms | ease-out | Button, tab, input feedback |
| Enter/exit | 200-300ms | ease-out | Dialogs, menus, toasts |

- Only transform, opacity, color, and shadow transitions are used.
- Focus-visible state remains visible; reduced-motion media rules disable
  non-essential dialog transitions.

## 7. Depth & Surface

The system uses a mixed strategy: warm tonal shifts and 1px warm borders define
structure, while cards and menus may use a low-alpha brown shadow. No cool
navy/Slate surface is permitted in dark mode. Dark table headers, sticky
columns, and settings navigation use the warm charcoal ramp.

## 8. Accessibility Constraints & Accepted Debt

- WCAG 2.2 AA target: at least 4.5:1 for body text and 3:1 for large text.
- Every interactive control needs keyboard access and a visible focus state.
- CJK labels must remain legible without clipped glyphs or orphaned short
  phrases at supported widths.
- The only documented color exception is the payment-provider brand palette,
  which is scoped to the provider-specific payment controls.
