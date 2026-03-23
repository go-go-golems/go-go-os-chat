---
Title: Align Frontend Profile Selection with Backend Runtime Contract
Ticket: APP-08-PROFILE-RUNTIME-CONTRACT-ALIGNMENT
Status: active
Topics:
    - architecture
    - backend
    - chat
    - wesen-os
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Follow-on ticket for APP-04 that completed the selector contract cutover: requested selection now uses `profile`/`registry`, legacy request aliases are rejected, and debug JSON payloads report resolved runtime identity as `resolved_runtime_key`.
LastUpdated: 2026-03-06T20:03:00-05:00
WhatFor: Track and explain the completed contract cleanup after APP-04 so the shared OS chat platform has one clear runtime/profile selection vocabulary across frontend transport, backend request resolution, websocket handshakes, and profile APIs.
WhenToUse: Use this ticket when auditing or changing chat profile selection, runtime selection fields, request resolver behavior, websocket profile propagation, or compatibility between frontend profile UI state and backend runtime composition.
---

# Align Frontend Profile Selection with Backend Runtime Contract

## Overview

This ticket isolates one concrete follow-up from APP-04: the frontend currently thinks in terms of `profile` and `registry`, while parts of the backend and integration tests still speak in terms of `runtime_key` and `registry_slug`. Those names are not interchangeable in every context, and the overloading of `runtime_key` is especially confusing because it can mean either:

- the requested profile/runtime selector, such as `inventory` or `planner`
- the resolved runtime identity reported back by the backend, such as `planner@v1`

The goal of this ticket is to define and document one clean contract for:

- request field names
- websocket query parameters
- current-profile/current-runtime response payloads
- cutover rules for legacy selector names
- ownership boundaries between the shared chat platform from APP-04 and app-specific callers

This is a focused contract-alignment ticket, not the larger runtime-DSL or VM-boundary redesign work.

## Key Links

- [Design doc](./design-doc/01-profile-registry-and-runtime-key-contract-alignment-plan.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)
- Related parent platform ticket: `APP-04-OS-CHAT-PLATFORM`

## Status

Current status: **active**

Current state:

- request selector cutover implemented across frontend, shared resolver, inventory, pinocchio, and launcher tests
- debug JSON payloads renamed to `resolved_runtime_key`
- current-profile API cut over to `{ profile, registry }`
- implementation diary, tasks, and changelog updated with final verification

## Topics

- architecture
- backend
- chat
- wesen-os

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
