---
Title: Extract Shared OS Chat Platform and Migrate Inventory
Ticket: APP-04-OS-CHAT-PLATFORM
Status: active
Topics:
    - architecture
    - backend
    - chat
    - wesen-os
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/chatservice/component.go
      Note: Shared chat transport package created by the ticket implementation
    - Path: /home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/pkg/assistantbackendmodule/module.go
      Note: Assistant backend module now consumes the shared platform
    - Path: /home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/cmd/wesen-os-launcher/main.go
      Note: Primary evidence for current inventory chat backend assembly
    - Path: ttmp/2026/03/05/APP-04-OS-CHAT-PLATFORM--extract-shared-os-chat-platform-and-migrate-inventory/design-doc/01-intern-guide-shared-os-chat-platform-extraction-inventory-migration-and-wesen-os-mounting.md
      Note: Primary architecture and implementation guide
    - Path: ttmp/2026/03/05/APP-04-OS-CHAT-PLATFORM--extract-shared-os-chat-platform-and-migrate-inventory/design-doc/02-intern-guide-revised-organization-for-mountable-os-chat-features-and-generic-backend-modules.md
      Note: Revised organization guide that separates mountable chat features from generic backend modules
    - Path: ttmp/2026/03/05/APP-04-OS-CHAT-PLATFORM--extract-shared-os-chat-platform-and-migrate-inventory/design-doc/03-intern-guide-tight-chat-core-with-minimal-profile-surface-and-no-extension-schemas.md
      Note: Tight-core guide that removes extension schemas and rich profile-admin concerns from the base design
    - Path: ttmp/2026/03/05/APP-04-OS-CHAT-PLATFORM--extract-shared-os-chat-platform-and-migrate-inventory/reference/01-investigation-diary.md
      Note: Chronological research log
ExternalSources: []
Summary: Track the extraction of shared Go chat backend infrastructure into a reusable OS chat platform, including the implemented shared repo, the inventory migration, the mounted assistant backend module, and the tighter architecture that removes extension schemas from the base design.
LastUpdated: 2026-03-06T10:42:08-05:00
WhatFor: Track the platform work that turned inventory-shaped Go chat infrastructure into shared OS chat backend packages, migrated inventory to the shared backend, mounted the assistant backend module in wesen-os, and documented the remaining follow-up work.
WhenToUse: Use this ticket when implementing or reviewing the shared Go chat platform extraction, the revised feature split, the tight chat core proposal, inventory migration, or the assistant-module mount.
---



# Extract Shared OS Chat Platform and Migrate Inventory

## Overview

This ticket covers the platform work that must exist before generic app-chat can be implemented cleanly. The target is to move reusable Go chat backend infrastructure out of `go-go-app-inventory`, migrate inventory to consume that shared layer without behavior change, and mount one reusable assistant backend in `wesen-os`.

The primary deliverables are:

- `design-doc/01-intern-guide-shared-os-chat-platform-extraction-inventory-migration-and-wesen-os-mounting.md`
- `design-doc/02-intern-guide-revised-organization-for-mountable-os-chat-features-and-generic-backend-modules.md`
- `design-doc/03-intern-guide-tight-chat-core-with-minimal-profile-surface-and-no-extension-schemas.md`

## Key Links

- [Design doc](./design-doc/01-intern-guide-shared-os-chat-platform-extraction-inventory-migration-and-wesen-os-mounting.md)
- [Revised organization guide](./design-doc/02-intern-guide-revised-organization-for-mountable-os-chat-features-and-generic-backend-modules.md)
- [Tight chat core guide](./design-doc/03-intern-guide-tight-chat-core-with-minimal-profile-surface-and-no-extension-schemas.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**

Current state:
- ticket created
- detailed design guide written
- revised organization guide written
- tight chat core guide written
- investigation diary written
- go-go-os-chat backend repo implemented
- inventory migrated to shared backend packages
- assistant backend module mounted in wesen-os
- implementation validated and refreshed implementation bundle uploaded

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
