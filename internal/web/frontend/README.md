# proxy dashboard

The admin console for proxy, following the lab34 brand guidelines: React 19 +
TypeScript on Vite, styled with Tailwind CSS v4 and [shadcn/ui](https://ui.shadcn.com)
components (Radix primitives + `class-variance-authority`).

## Stack

- **Fonts** — IBM Plex Sans (interface copy) and IBM Plex Mono (logotype, labels,
  tables, figures, code), self-hosted via `@fontsource` so the built binary works
  offline. Medium 500 is the heaviest weight in use — no bold anywhere.
- **Theme tokens** — `src/index.css` maps the lab34 palette (bone `#F3F2F2`,
  ink `#201F1D`, brass `#B68235`, grey `#9B9797`, carbon `#2D2B2B`) onto shadcn's
  CSS variables for both light (ink on bone) and dark (bone on ink/carbon) modes.
  Brass appears as stroke — rules, focus rings, small marks — never as a fill.
- **Components** — `src/components/ui/` holds the shadcn primitives
  (`components.json` is configured, so `npx shadcn@latest add <component>` works);
  `src/components/brand/` holds the lab34 wordmark lockup and «34» monogram.

## Development

```bash
npm ci
npm run dev     # Vite dev server on :5173, proxying /admin and /v1 to :8080
npm run lint
npm run build   # emits dist/, embedded into the Go binary via go:embed
```

From the repository root, `./development.sh` starts this dev server together with
the Go backend.
