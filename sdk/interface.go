package sdk

import "context"

// Adapter is the canonical BrikByteOS Adapter SDK v0 interface.
//
// Design intent:
//   - Metadata() is the single source of truth for identity, type, and order.
//   - CheckAvailability() reports whether the adapter can run locally.
//   - Version() detects tool version without changing runtime state.
//   - Run() performs execution only and returns structured execution truth.
//   - Normalize() transforms raw execution into canonical normalized JSON.
//
// Adapters must be:
//   - stateless
//   - deterministic
//   - local-first
//   - side-effect controlled
//
// Adapters must not:
//   - write files directly
//   - call external backend services
//   - depend on hidden global runtime state
type Adapter interface {
	// Metadata returns the canonical static description of this adapter.
	Metadata() AdapterMetadata

	// CheckAvailability determines whether this adapter can run in the current environment.
	CheckAvailability(ctx context.Context) Availability

	// Version returns a best-effort local tool version.
	// Returning "UNKNOWN" is acceptable when deterministic local detection is not possible.
	Version(ctx context.Context) (string, error)

	// Run performs adapter execution only.
	// It must not normalize, persist files, or mutate runtime-owned state.
	Run(ctx context.Context, req RunRequest) RunResult

	// Normalize transforms raw execution into schema-compliant normalized JSON.
	// This function must be deterministic and side-effect free.
	Normalize(ctx context.Context, input NormalizationInput) NormalizedResult
}
