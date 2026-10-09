# Desktop shadcn controls

The compound components in `shadcn/` are downloaded official shadcn/ui
`new-york-v4` Radix sources, with the adaptations listed below. They are not
independently recreated lookalikes. Desktop-facing wrappers live beside this file.

## Provenance

- Repository: https://github.com/shadcn-ui/ui
- Pinned commit: `0132174664c07d41262fb51012d0cc782e458e6c`
- Source directory: https://github.com/shadcn-ui/ui/tree/0132174664c07d41262fb51012d0cc782e458e6c/apps/v4/registry/new-york-v4/ui
- Raw sourceURL template: `https://raw.githubusercontent.com/shadcn-ui/ui/0132174664c07d41262fb51012d0cc782e458e6c/apps/v4/registry/new-york-v4/ui/{component}.tsx`
- Downloaded components: `button`, `input`, `textarea`, `select`, `switch`,
  `checkbox`, `slider`, `progress`.
- Official setup: https://ui.shadcn.com/docs/installation/vite
- Official Select documentation: https://ui.shadcn.com/docs/components/radix/select
- MIT copyright and license: [shadcn/LICENSE.md](./shadcn/LICENSE.md).

## Deliberate source adaptations

- Resolve `cn` through the existing `@/lib/utils`; use individual
  `@radix-ui/react-*` packages rather than the aggregate `radix-ui` dependency.
- Forward refs explicitly for React 18 in Button, Input, Textarea, SelectTrigger
  and Slider. The downloaded sources otherwise use the newer ref-as-prop shape.
- Expose `thumbProps` on the source Slider so a scalar wrapper can place its
  accessible name, ID and disabled state on the actual slider thumb.
- Pass `value` and `max` to the Progress root and scale its indicator by `max`.
  The downloaded indicator assumes a percentage and does not pass its
  destructured value to the root.
- Keep official compound anatomy, Radix keyboard/focus behavior, portal,
  viewport, scroll buttons, item indicators and animation classes. Desktop
  wrappers choose compact dimensions and implement the controlled APIs below.

## Desktop APIs

All value APIs are controlled. There is no implicit selection of the first option.

| Export | Value contract | Other supported props |
| --- | --- | --- |
| `Select` | `value: string`, `onValueChange(string)`, `options: { value: string; label: ReactNode; disabled?: boolean }[]` | `disabled`, `id`, `className`, `placeholder`, `aria-label`, `aria-labelledby`, `aria-invalid`, `aria-describedby` |
| `Switch` | `checked: boolean`, `onCheckedChange(boolean)` | Button attributes including ARIA, `id`, `disabled`, `className`; no indeterminate value |
| `Checkbox` | `checked: boolean`, `onCheckedChange(boolean)` | Same button attributes; no indeterminate value |
| `Slider` | Scalar `value: number`, `onValueChange(number)` | `min=0`, `max=100`, `step=1`, `disabled`, `id`, `className`, `aria-label`, `aria-labelledby`, `aria-describedby`, `aria-valuetext` |
| `Progress` | `value: number`, `max: number`, required `aria-label` | HTML div attributes; bounds are shared by ARIA and the visual indicator |
| `Button` | Existing button props and ref | Variants `default`/`primary`, `secondary`, `ghost`, `danger`; sizes `default`/`md`, `sm`, `icon` |
| `Input`, `Textarea` | Existing native input/textarea props and refs | Exported together from `input.tsx` |

Select maps every option to its array index before passing values to Radix. An
empty business string is a real selectable option and cannot collide with a
sentinel or another business string such as `"0"`. A nonempty absent value stays
visible; an absent empty value displays the placeholder. Reordering options
recomputes the selected index without changing the business value.

Select, Switch and Checkbox respect native disabled fieldsets. Select closes its
portal when an ancestor is disabled. Slider observes ancestor fieldset changes
to remove thumb focus and block pointer, keyboard and callback writes. The first
legend exception follows HTML fieldset semantics.

Radix's hidden native inputs/select provide form semantics. Only the custom
shadcn controls are visible. Progress uses a styled Radix progressbar.

## Styling and integration

`main.tsx` imports `./shadcn.css` **before** `./style.css`. Components do not each
import that stylesheet. Existing Tabs, ScrollArea and Badge retain their behavior.

- Tailwind v4 runs through `@tailwindcss/vite`; both Vite and TypeScript resolve
  `@` to `src`. `components.json` targets `ui/shadcn` for future source additions.
- `shadcn.css` imports only Tailwind theme/utilities and `tw-animate-css`, never
  Preflight or a global layout reset. Scoped `@layer base` data-slot rules reset
  borders to `0 solid` and supply button defaults, allowing utility border widths
  to override them. Explicit popup typography survives the Desktop's global rules.
- Theme colors reuse Desktop tokens. Primary maps to `--accent`; the Tailwind
  accent color maps to `--surface-hover`. Existing `--accent` keeps its meaning.
- Dark mode follows `:root[data-theme="dark"]`. Popup text is explicitly 12px;
  form triggers are 32px with a 7px radius. Menu height is capped at 280px and the
  available viewport height, with Radix scroll controls. Reduced motion is honored.
- The Desktop's unlayered `.input` and `.button-*` styles retain layout ownership;
  their compact styling is maintained by the consumer/style integration owner.

## Focused verification

Run `npm test -- src/components/ui/controls.test.tsx` from the frontend directory.
Coverage includes portal placement, Arrow/Enter/Escape interaction, disabled
options, empty-string and sentinel-looking values, absent/reordered values,
ARIA labels, boolean callbacks, fieldset disabling, scalar Slider boundaries,
Progress ratios and retained Input/Textarea/Button refs.

The jsdom-only browser API polyfills are in `src/test/setup.ts`. These tests do
not claim browser layout or native-app visual validation; full frontend/build
and GUI verification belong to the integration owner.
