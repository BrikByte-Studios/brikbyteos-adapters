/*
Package sdk defines the canonical Adapter SDK v0 contract for BrikByteOS.

This package is the single authoritative boundary between:
  - adapter implementations
  - runtime orchestration
  - raw execution capture
  - normalization

Phase 1 goals:
  - one shared adapter interface for all built-in adapters
  - deterministic, stateless, local-first adapter behavior
  - no direct filesystem writes by adapters
  - no backend/API/dashboard dependency
  - structured failure-as-data semantics
  - one canonical execution input contract

Design rules:
  - metadata is the single source of truth for adapter identity
  - runtime owns persistence and artifact pathing
  - execution and normalization remain separate concerns
  - adapters receive the same execution context structure
*/
package sdk
