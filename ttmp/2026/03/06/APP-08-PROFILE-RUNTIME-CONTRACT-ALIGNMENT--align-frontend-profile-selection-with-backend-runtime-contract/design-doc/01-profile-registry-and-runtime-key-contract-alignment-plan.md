---
Title: Profile Registry and Runtime Key Contract Alignment Plan
Ticket: APP-08-PROFILE-RUNTIME-CONTRACT-ALIGNMENT
Status: active
Topics:
    - architecture
    - backend
    - chat
    - wesen-os
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026/03/05/APP-04-OS-CHAT-PLATFORM--extract-shared-os-chat-platform-and-migrate-inventory/index.md
      Note: Parent platform ticket whose shared chat boundary this contract cleanup builds on
    - Path: cmd/wesen-os-launcher/main_integration_test.go
      Note: Integration tests show resolved current_runtime_key behavior and versioned runtime identity
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver_test.go
      Note: Backend resolver tests still exercise runtime_key and registry_slug aliases
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts
      Note: Frontend HTTP submit path currently sends profile and registry fields
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileTypes.ts
      Note: Frontend naming model for selected profile and registry
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts
      Note: Frontend websocket path currently propagates profile and registry query parameters
ExternalSources: []
Summary: Record and explain the APP-08 cutover that standardizes requested selectors on `profile`/`registry`, rejects legacy selector names, and separates requested profile selection from resolved runtime identity via `resolved_runtime_key`.
LastUpdated: 2026-03-06T13:46:02.92331469-05:00
WhatFor: Use this design doc to scope and implement the request/response contract cleanup around chat runtime selection after APP-04 extracted the shared OS chat platform.
WhenToUse: Use when changing frontend chat transport fields, websocket profile propagation, backend request resolution, or current-runtime/current-profile payload semantics.
---


# Profile Registry and Runtime Key Contract Alignment Plan

## Executive Summary

This ticket exists because the chat stack had two overlapping vocabularies for runtime selection:

- frontend/UI vocabulary: `profile`, `registry`
- backend/request-resolver vocabulary: `runtime_key`, `registry_slug`

That mismatch is already visible in code and tests, and it becomes more confusing because `runtime_key` is overloaded. Sometimes it means the requested profile selector such as `planner`; other times it means the resolved runtime identity such as `planner@v1`.

The implemented outcome is:

- define one canonical contract for selecting a profile/registry
- define a separate canonical contract for reporting resolved runtime identity
- reject legacy request selector names instead of preserving compatibility wrappers
- land the contract cleanup as a follow-on to APP-04 shared chat platform work

This is intentionally smaller than the VM or multi-DSL work. It is a transport and API contract cleanup ticket.

## Problem Statement

The codebase currently mixes these concepts:

1. requested profile selection
2. registry namespace selection
3. resolved runtime identity
4. versioned runtime fingerprinting

Concrete examples:

- frontend HTTP transport sends `profile` and `registry`
- frontend WS transport sends `profile` and `registry`
- backend resolver tests still exercise `runtime_key` and `registry_slug`
- backend integration tests inspect `current_runtime_key`
- runtime composer may return a versioned runtime key like `planner@v1`

This creates three practical problems.

### Problem A: Naming mismatch across the wire

Different layers use different field names for what is often the same selector.

### Problem B: `runtime_key` is ambiguous

`runtime_key` can mean either:

- "which profile/runtime should I use?"
- "which resolved runtime am I currently running?"

Those are not the same thing once versioning and runtime composition are involved.

### Problem C: APP-04 made the boundary more important

APP-04 extracted the shared OS chat platform. That was the right architectural move, but it also means the request/response contract now matters more because multiple apps and frontend packages will reuse it. Ambiguous naming that was survivable in a single inventory-shaped flow becomes a platform liability.

## Proposed Solution

Adopt a two-layer naming model.

### Layer 1: Requested Selection

This is what the client asks for.

Canonical fields:

- `profile`
- `registry`

Semantics:

- `profile`: requested profile slug
- `registry`: requested profile registry slug

These names already match frontend UI/state terminology and the profile APIs, so they are the better fit for user-facing transport.

### Layer 2: Resolved Runtime

This is what the backend actually composed for the conversation.

Canonical JSON debug field:

- `resolved_runtime_key`

Semantics:

- `resolved_runtime_key`: resolved runtime identity used by the engine, including version semantics if applicable

This separates "what the client asked for" from "what the server is actually running".

### Cutover Rule

After APP-08:

- backend accepts only `profile` and `registry` as public request selectors
- legacy `runtime_key` and `registry_slug` request fields return `400`
- invalid canonical registry slugs return `400`
- unknown registries return `404`
- debug JSON payloads use `resolved_runtime_key`
- internal persistence and SEM protocol fields that already represent resolved runtime identity may continue to use `runtime_key`

### Scope Boundary

This ticket should cover:

- frontend chat transport fields
- websocket query parameters
- backend request resolver input aliases
- current-profile/current-runtime response payload naming
- integration tests and documentation

This ticket should not cover:

- VM state/dispatch simplification
- multi-DSL runtime design
- broader profile registry administration APIs
- effect/runtime action platformization

## Design Decisions

### Decision 1: Prefer `profile` / `registry` for requested selection

Rationale:

- already matches frontend state and UI terminology
- better reflects what the user is selecting
- less overloaded than `runtime_key`

### Decision 2: Use `resolved_runtime_key` for resolved runtime identity in JSON debug payloads

Rationale:

- the composed runtime may be versioned (`planner@v1`)
- that is not equivalent to the requested profile slug
- `resolved_runtime_key` makes the semantics explicit and avoids overloading `current_*` terminology

### Decision 3: Treat `runtime_key` and `registry_slug` as rejected legacy request names, not compatibility inputs

Rationale:

- the user explicitly chose a hard cutover rather than compatibility wrappers
- silent fallback would keep the selector contract ambiguous
- explicit `400 unsupported legacy selector` responses make mistakes obvious and easy to fix

### Decision 4: Position this as a follow-on to APP-04

Rationale:

- APP-04 created the shared chat platform boundary
- this ticket cleans up one of the contract seams exposed by that work
- it should remain a separate ticket so the migration can be tested and reviewed on its own

## Alternatives Considered

### Alternative A: Keep both vocabularies indefinitely

Rejected because:

- ambiguity remains permanent
- docs and tests stay harder to reason about
- future apps inherit the confusion

### Alternative B: Standardize everything on `runtime_key` / `registry_slug`

Rejected because:

- `runtime_key` is the wrong name for a requested profile slug in the frontend UX layer
- `registry_slug` is more implementation-flavored than necessary for common UI code
- it does not solve the requested-versus-resolved ambiguity

### Alternative C: Remove all runtime-key reporting and only expose profile slugs

Rejected because:

- the backend really does have a resolved runtime identity
- versioned runtime keys are useful for debugging and correctness
- we should separate the concepts, not hide one of them

## Implementation Plan

### Phase 1: Audit and contract decision

- confirm every request/response surface carrying profile/runtime selection
- list all aliases and payload shapes
- confirm APP-04 ownership boundary for the shared chat platform pieces

### Phase 2: Execute selector cutover

- backend request resolver accepts:
  - `profile`
  - `registry`
- backend rejects:
  - `runtime_key`
  - `registry_slug`

### Phase 3: Update frontend transports

- HTTP submit path sends canonical `profile` / `registry`
- WS connection path sends canonical `profile` / `registry`
- current-profile selection state remains profile/registry-based

### Phase 4: Clean up response semantics

- document and standardize:
  - requested profile selection fields
  - resolved runtime identity fields
- ensure integration tests assert the right field for the right purpose

### Phase 5: Audit remaining consumers

- update browser-side debug consumers to `resolved_runtime_key`
- leave non-request uses of `runtime_key` alone where they already represent resolved runtime identity

## Final Decisions

- request selector vocabulary is `profile` and `registry`
- legacy request selector names are rejected
- current-profile payloads use `{ profile, registry }`
- debug JSON payloads use `resolved_runtime_key`
- SEM hello frames and persistence rows remain out of scope when they already represent resolved runtime identity rather than requested selector input

## References

- APP-04 shared platform ticket: `APP-04-OS-CHAT-PLATFORM`
- Frontend HTTP transport: `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts`
- Frontend WS transport: `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts`
- Frontend profile selection types: `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileTypes.ts`
- Backend request resolver tests: `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver_test.go`
- Launcher integration tests covering runtime key behavior: `cmd/wesen-os-launcher/main_integration_test.go`
