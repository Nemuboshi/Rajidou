# Rajidou CLI

This project is primarily a TypeScript automation reimplementation of the core ideas and workflows from the original Rajiko project:

- Rajiko: <https://github.com/jackyzy823/rajiko>

Most core logic in this repository (auth flow, playlist/chunklist handling, and AAC segment processing) is adapted from Rajiko and then integrated into a Playwright-based CLI automation tool.

Respect and thanks to the original Rajiko authors and contributors.

This is a minimal Rajiko CLI that:

1. Reads Radiko links from YAML.
2. Resolves to a detail page.
3. Fetches timeshift segments.


## Install

```bash
pnpm i
pnpm exec playwright install chromium
```

## Run

```bash
pnpm start -- --config links.yaml
```

See links.example.yaml for config.
