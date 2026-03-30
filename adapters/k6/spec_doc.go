package k6

/*
Package k6 defines the canonical Phase 1 k6 adapter execution specification.

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
  1. Resolve k6 using the documented binary strategy.
  2. Use one canonical machine-readable summary-export artifact: JSON.
  3. Treat the summary-export JSON file as the primary parser input.
  4. Treat stdout/stderr as raw evidence only.
  5. Support one local script per adapter run.
  6. Forbid remote/cloud/distributed execution in Phase 1.
  7. Use deterministic artifact file names under .bb/.
  8. Treat threshold evaluation as part of the structured summary source-of-truth,
     not as an ambiguous stdout-only interpretation.
*/