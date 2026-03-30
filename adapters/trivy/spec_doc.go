package trivy

/*
Package trivy defines the canonical Phase 1 Trivy adapter execution specification.

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
  1. Resolve Trivy using the documented binary strategy.
  2. Use one canonical machine-readable JSON report artifact.
  3. Treat the JSON report file as the primary parser input.
  4. Treat stdout/stderr as raw evidence only.
  5. Support filesystem targets only in Phase 1.
  6. Preserve findings in raw JSON without making them execution failures by default.
  7. Avoid destructive severity filtering at execution time.
  8. Use deterministic artifact file names under .bb/.
  9. Keep offline/air-gapped DB optimization behavior out of scope for Phase 1.
*/