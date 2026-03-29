package playwright

/*
Package playwright defines the canonical Phase 1 Playwright adapter execution specification.

This package intentionally separates:
  - static metadata
  - execution contract
  - deterministic artifact path rules
  - command-shape helpers

from:
  - raw command execution
  - parser implementation
  - normalization logic

Phase 1 execution rules:
  1. Resolve Playwright in deterministic order:
     - local node_modules/.bin/playwright
     - npx playwright
     - global playwright (only if allowed by runtime policy)
  2. Use one canonical structured reporter: json
  3. Treat the structured report artifact as the primary parser input
  4. Treat stdout/stderr as raw evidence only
  5. Require browsers to be pre-installed; no hidden install behavior
  6. Use deterministic artifact file names under .bb/
  7. Keep HTML/traces/screenshots/videos out of the canonical parser flow in Phase 1
*/