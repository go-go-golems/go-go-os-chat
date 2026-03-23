---
Title: Investigation diary
Ticket: APP-04-OS-CHAT-PLATFORM
Status: active
Topics:
    - architecture
    - backend
    - chat
    - wesen-os
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os-chat/pkg/chatservice/component.go
      Note: |-
        Shared mountable chat transport package created during implementation
        Shared mountable chat transport implementation extracted during APP-04
    - Path: ../../../../../../../go-go-os-chat/pkg/profilechat/runtime_composer.go
      Note: |-
        Shared profile-aware runtime composer extracted from inventory
        Shared profile-aware runtime composer extracted during APP-04
    - Path: cmd/wesen-os-launcher/main.go
      Note: Primary evidence file for current backend assembly
    - Path: pkg/assistantbackendmodule/module.go
      Note: |-
        Assistant backend module mounted in wesen-os using the shared chat packages
        Assistant backend module mounted in wesen-os during APP-04
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go
      Note: Primary evidence file for request contract analysis
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go
      Note: Primary evidence file for runtime composition analysis
ExternalSources: []
Summary: Chronological diary of the research, implementation, and migration work for APP-04.
LastUpdated: 2026-03-06T10:42:08-05:00
WhatFor: Record how the APP-04 research ticket was created, how the shared chat packages were extracted and validated, how inventory was ported, and how the assistant backend module was mounted in wesen-os.
WhenToUse: Use this diary to review the investigation process, reproduce the research steps, or continue the ticket later.
---



# Diary

## Goal

Capture the research and documentation work used to create APP-04 and APP-05, with APP-04 receiving the detailed implementation guide.

## Step 1: Create the Shared OS Chat Platform Tickets and Write the APP-04 Guide

The goal of this step was to split the work into two clean tickets: APP-04 for the shared platform extraction/migration/mount work, and APP-05 for the later app-bootstrap/context work. The implementation emphasis for this turn was APP-04, so the work centered on evidence gathering across `wesen-os`, `go-go-app-inventory`, `go-go-os-frontend`, and `go-go-os-backend`.

The main challenge was avoiding hand-wavy recommendations. The guide needed to make a precise distinction between generic platform code and inventory-owned app semantics, because that distinction is the entire point of the ticket.

### Prompt Context

**User prompt (verbatim):** "ok, create a detailed new ticket for 1 - 3, then another one for 4. Create a detailed implementation guide for either of them, and t asks for the 1-3. ticket. (we'll go over 4. later).

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create two new `docmgr` tickets, write the detailed intern-facing design/implementation guide into the first ticket covering steps 1-3, keep the second ticket scoped for step 4, and upload the APP-04 deliverables to reMarkable.

**Inferred user intent:** Establish a rigorous, trackable architecture plan for turning inventory chat into a proper OS chat platform before implementing generic app-chat bootstrap.

**Commit (code):** N/A — documentation/ticket work only

### What I did

- Confirmed the `docmgr` root and existing ticket inventory with:
  - `docmgr status --summary-only`
  - `docmgr ticket list`
- Created two new tickets:
  - `docmgr ticket create-ticket --ticket APP-04-OS-CHAT-PLATFORM --title "Extract Shared OS Chat Platform and Migrate Inventory" --topics architecture,backend,chat,wesen-os`
  - `docmgr ticket create-ticket --ticket APP-05-GENERIC-APP-CHAT-BOOTSTRAP --title "Generic App Chat Bootstrap and Context Injection" --topics backend,chat,documentation,wesen-os`
- Added the APP-04 design doc and diary, plus a scope note doc for APP-05:
  - `docmgr doc add --ticket APP-04-OS-CHAT-PLATFORM --doc-type design-doc --title "Intern Guide: Shared OS Chat Platform Extraction, Inventory Migration, and Wesen-OS Mounting"`
  - `docmgr doc add --ticket APP-04-OS-CHAT-PLATFORM --doc-type reference --title "Investigation diary"`
  - `docmgr doc add --ticket APP-05-GENERIC-APP-CHAT-BOOTSTRAP --doc-type design-doc --title "Scope note: generic app chat bootstrap and context injection"`
- Gathered file-backed evidence from:
  - `package.json`
  - `pnpm-workspace.yaml`
  - `go.work`
  - `apps/os-launcher/src/App.tsx`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/...`
  - `workspace-links/go-go-app-inventory/...`
  - `workspace-links/go-go-os-backend/...`
  - `cmd/wesen-os-launcher/main.go`
- Wrote the APP-04 design document as the primary technical deliverable.
- Wrote the APP-05 scope note so the bootstrap work is tracked separately and does not get mixed into APP-04.

### Why

- APP-04 and APP-05 are related but not the same kind of work.
- APP-04 is a platform extraction/migration ticket.
- APP-05 is a feature/bootstrap ticket that depends on APP-04.
- Keeping them separate reduces implementation risk and makes review cleaner.

### What worked

- The codebase already had a strong evidence trail for the key architectural points.
- `ChatConversationWindow` and the launcher host context made it clear the frontend routing model is already generic.
- `cmd/wesen-os-launcher/main.go` made the current backend assembly responsibilities explicit.
- `backendhost` made the target mount model explicit.

### What didn't work

- `docmgr ticket list | rg '^APP-'` returned no useful filtered output in the first attempt, so I switched to reading the unfiltered `docmgr ticket list` output directly.
- Large `nl -ba` dumps of `request_resolver.go` and related files were too verbose to consume in one shot; I switched to targeted `rg -n` plus smaller `sed -n` windows.

### What I learned

- The frontend runtime is already sufficiently generic that the first high-value extraction is clearly on the Go side.
- The right long-term framing is "OS chat platform" rather than "inventory chat made reusable."
- The backendhost route contract already gives the right long-term shape: `/api/apps/<app-id>`.

### What was tricky to build

The sharp edge in this task was deciding what actually counts as "generic." Several inventory packages are generic in algorithm but inventory-specific in defaults. The fix was to separate "generic algorithm" from "app-owned defaults":

- route mounting logic is generic
- request normalization is generic
- runtime composition is generic
- inventory prompts/tools/docs/reflection are not generic
- HyperCard is likely an optional extension, not chat core

Without that distinction, the new repo would either be too thin to matter or too polluted with inventory semantics.

### What warrants a second pair of eyes

- The package layout recommendation for `go-go-os-chat`
- The decision to keep HyperCard outside core chat for now
- The recommendation to defer moving `@hypercard/chat-runtime`
- The proposed `assistant` app id versus an alternative such as `os-chat`

### What should be done in the future

- Implement APP-04 in phases.
- After APP-04 is stable, work APP-05 to add generic app-chat bootstrap/context injection.

### Code review instructions

- Start with the APP-04 design doc:
  - `design-doc/01-intern-guide-shared-os-chat-platform-extraction-inventory-migration-and-wesen-os-mounting.md`
- Then verify the key evidence files:
  - `cmd/wesen-os-launcher/main.go`
  - `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
  - `workspace-links/go-go-os-backend/pkg/backendhost/routes.go`
- Validation for the documentation step:
  - run `docmgr doctor --ticket APP-04-OS-CHAT-PLATFORM --stale-after 30`

### Technical details

Key commands run:

```bash
docmgr status --summary-only
docmgr ticket list
docmgr ticket create-ticket --ticket APP-04-OS-CHAT-PLATFORM --title "Extract Shared OS Chat Platform and Migrate Inventory" --topics architecture,backend,chat,wesen-os
docmgr ticket create-ticket --ticket APP-05-GENERIC-APP-CHAT-BOOTSTRAP --title "Generic App Chat Bootstrap and Context Injection" --topics backend,chat,documentation,wesen-os
docmgr doc add --ticket APP-04-OS-CHAT-PLATFORM --doc-type design-doc --title "Intern Guide: Shared OS Chat Platform Extraction, Inventory Migration, and Wesen-OS Mounting"
docmgr doc add --ticket APP-04-OS-CHAT-PLATFORM --doc-type reference --title "Investigation diary"
docmgr doc add --ticket APP-05-GENERIC-APP-CHAT-BOOTSTRAP --doc-type design-doc --title "Scope note: generic app chat bootstrap and context injection"
rg -n "resolveApiBase|resolveWsBase|chat-runtime|ChatConversationWindow" ...
nl -ba cmd/wesen-os-launcher/main.go | sed -n '160,320p'
nl -ba workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go | sed -n '1,220p'
nl -ba workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go | sed -n '1,320p'
nl -ba workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go | sed -n '1,380p'
```

Primary outputs created:

- `design-doc/01-intern-guide-shared-os-chat-platform-extraction-inventory-migration-and-wesen-os-mounting.md`
- `reference/01-investigation-diary.md`
- `../../APP-05-GENERIC-APP-CHAT-BOOTSTRAP--generic-app-chat-bootstrap-and-context-injection/design-doc/01-scope-note-generic-app-chat-bootstrap-and-context-injection.md`

## Step 2: Write the Revised Organization Guide That Splits Chat Features from Generic Backend Modules

This step followed a narrower design discussion after the initial APP-04 guide. The goal was to tighten the architecture around the actual upstream `webchat` model and make the ticket explicit that chat should be packaged as a mountable feature, not as the defining shape of every backend module.

The main value of this step was clarity. The earlier guide already pointed in the right direction, but this addendum makes the package boundary much harder to misunderstand for a new engineer: generic backend modules stay small, chat transport and profile APIs are optional features, and docs/reflection remain app-owned.

### Prompt Context

**User prompt (verbatim):** "yes, detailed document for a new organization, and upload to remarkable once done. Add to the ticket."

**Assistant interpretation:** Add a second, detailed APP-04 design document that formalizes the revised architecture discussed in chat, then validate and republish the ticket bundle to reMarkable.

**Inferred user intent:** Lock in the refined organization before implementation starts so the repo split and API design do not drift toward an inventory-shaped abstraction.

**Commit (code):** N/A — documentation/ticket work only

### What I did

- Re-read the local skill instructions for `docmgr`, `ticket-research-docmgr-remarkable`, `diary`, and `remarkable-upload`.
- Re-opened the APP-04 workspace and current guide/diary.
- Gathered additional line-anchored evidence from:
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/router.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/server.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/doc/topics/webchat-framework-guide.md`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/doc/topics/webchat-http-chat-setup.md`
  - `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
  - `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
  - `workspace-links/go-go-os-backend/pkg/backendhost/module.go`
  - `workspace-links/go-go-os-backend/pkg/backendhost/routes.go`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/registerChatModules.ts`
  - `workspace-links/go-go-os-frontend/packages/hypercard-runtime/src/hypercard/timeline/registerHypercardChatModules.ts`
- Created a new APP-04 design doc:
  - `design-doc/02-intern-guide-revised-organization-for-mountable-os-chat-features-and-generic-backend-modules.md`
- Wrote a detailed intern-facing explanation covering:
  - what `parsed *values.Values` and `staticFS` really do
  - what middleware definitions are in `RuntimeComposer`
  - what `ExtensionSchemas`, `ProfileRegistry`, `DefaultRegistrySlug`, `WriteActor`, and `WriteSource` are actually for
  - why docs/reflection should stay above chat feature construction
  - the revised package layout for `go-go-os-chat`
  - API sketches, diagrams, migration phases, and testing strategy

### Why

- The original APP-04 guide described the right destination but still left room for an overly large "chat backend module" abstraction.
- The follow-up questions exposed the real boundary more precisely: chat transport and profile APIs are optional features, while app module identity, docs, and reflection are separate concerns.

### What worked

- The upstream `pinocchio` docs were explicit enough to support the revised architecture without guesswork.
- Inventory's current split between `backendcomponent` and `backendmodule` made it easy to show that docs are already one layer above chat transport.
- The frontend module registration code provided a clean example of where custom widgets and SEM renderers should live.

### What didn't work

- The newly created design-doc skeleton started with an empty `RelatedFiles: []` field; after the first patch it coexisted with the populated `RelatedFiles` block, so I corrected the duplicate frontmatter entry.
- The first reading-guide draft referenced section names that did not exactly match the final headings, so I corrected those references before validation.

### What I learned

- The strongest argument for the revised split is not aesthetic; it is that upstream `webchat` already assumes app-owned transport and utility-handler composition.
- The profile API options that looked like generic module config are actually narrowly tied to profile CRUD, schema discovery, and audit metadata.
- The correct abstraction is "mountable chat capability" rather than "chat-shaped backend module."

### What was tricky to build

The tricky part here was avoiding a misleading kind of generalization. Inventory's current component is generic enough that it is tempting to elevate it wholesale into the new shared package. But that would preserve the wrong conceptual boundary: it would still make profile editing, confirm mounting, and UI serving feel mandatory for any chat-enabled module. The fix was to separate concerns by responsibility instead of by today's file layout:

- backend host contract
- chat transport feature
- optional profile API feature
- app-owned wrapper
- frontend extension layer

That framing is much closer to the real ownership model already present in the code.

### What warrants a second pair of eyes

- The exact package names inside `go-go-os-chat`
- Whether confirm mounting belongs in an app wrapper or a chat-adjacent optional feature package
- Whether the assistant module id should be `assistant` or `os-chat`
- Which parts of HyperCard should remain inventory-specific versus becoming optional shared extensions

### What should be done in the future

- Validate the revised package split with the actual first extraction PR.
- Keep APP-05 focused on bootstrap/context injection and do not mix it back into APP-04.
- If a second chat-enabled app appears before extraction is complete, use it as a forcing function to test whether the new shared API is truly app-agnostic.

### Code review instructions

- Start with the new design doc:
  - `design-doc/02-intern-guide-revised-organization-for-mountable-os-chat-features-and-generic-backend-modules.md`
- Then verify the key upstream evidence:
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/server.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/router.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/doc/topics/webchat-framework-guide.md`
- Finally compare that against the inventory wrapper:
  - `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
  - `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
- Validation commands:
  - `docmgr doctor --ticket APP-04-OS-CHAT-PLATFORM --stale-after 30`
  - `remarquee upload bundle --dry-run ...`

### Technical details

Key commands run:

```bash
docmgr doc add --ticket APP-04-OS-CHAT-PLATFORM --doc-type design-doc --title "Intern Guide: Revised Organization for Mountable OS Chat Features and Generic Backend Modules"
nl -ba /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/router.go | sed -n '1,360p'
nl -ba /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/server.go | sed -n '1,220p'
nl -ba /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go | sed -n '1,980p'
nl -ba /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/doc/topics/webchat-framework-guide.md | sed -n '1,260p'
nl -ba /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/doc/topics/webchat-http-chat-setup.md | sed -n '1,220p'
nl -ba workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go | sed -n '1,240p'
nl -ba workspace-links/go-go-app-inventory/pkg/backendmodule/module.go | sed -n '1,220p'
nl -ba workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go | sed -n '1,320p'
nl -ba workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go | sed -n '1,340p'
nl -ba workspace-links/go-go-os-backend/pkg/backendhost/module.go | sed -n '1,220p'
nl -ba workspace-links/go-go-os-backend/pkg/backendhost/routes.go | sed -n '1,220p'
nl -ba workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/registerChatModules.ts | sed -n '1,220p'
nl -ba workspace-links/go-go-os-frontend/packages/hypercard-runtime/src/hypercard/timeline/registerHypercardChatModules.ts | sed -n '1,220p'
```

Primary outputs created:

- `design-doc/02-intern-guide-revised-organization-for-mountable-os-chat-features-and-generic-backend-modules.md`
- `reference/01-investigation-diary.md` (updated)

## Step 3: Write the Tight Chat Core Guide That Removes Extension Schemas from the Core Design

This step narrowed the design further after reviewing what the live chat runtime actually uses. The goal was to stop designing around a broad profile-management platform and instead define the leanest useful shared chat core for the current product: transport, runtime selection, timeline, and an optional read-only profile list for dropdowns.

The main design shift in this step was to treat `ExtensionSchemas` and the rest of the rich profile-admin surface as explicitly out of scope for the first shared core. That makes the platform smaller, more honest, and better aligned with the real UI behavior.

### Prompt Context

**User prompt (verbatim):** "Ok, make a third updated design that defines a tight chat core, leaving out extension schemas entirely. upload to remarkable when done"

**Assistant interpretation:** Add a third APP-04 design document that defines a minimal shared chat core, explicitly excludes `ExtensionSchemas`, and refresh the ticket bundle on reMarkable.

**Inferred user intent:** Constrain the first extraction so implementation effort goes into the minimum useful platform rather than optional profile-admin and schema-authoring features.

**Commit (code):** N/A — documentation/ticket work only

### What I did

- Re-checked the live frontend call sites for profile APIs and chat submission behavior:
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts`
- Verified that the live runtime path uses:
  - profile list
  - current-profile get/set
  - but not profile CRUD or schema discovery
- Verified that inventory hardcodes the `default` registry in the UI and only constructs one registry in the backend:
  - `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx`
  - `cmd/wesen-os-launcher/main.go`
- Verified that `ExtensionSchemas` are only explicitly used for starter suggestions in:
  - `cmd/wesen-os-launcher/main.go`
- Verified that the backend request resolver expects `runtime_key` and cookie state, while the frontend still sends `profile` and `registry`:
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/api.go`
- Created a new third design doc:
  - `design-doc/03-intern-guide-tight-chat-core-with-minimal-profile-surface-and-no-extension-schemas.md`
- Wrote a detailed guide that proposes:
  - a small shared chat core
  - an optional read-only profile list route
  - no `ExtensionSchemas`
  - no profile CRUD
  - no registry slugs in the base contract
  - sending `runtime_key` explicitly from the frontend

### Why

- The second design guide still left room for a profile API layer that is broader than the current product needs.
- The real UI requirement is much smaller: list profiles for a dropdown and send the selected runtime/profile with the conversation.
- Removing `ExtensionSchemas` from the core now reduces implementation complexity and future confusion.

### What worked

- The frontend hook code made the live API usage very easy to verify.
- The inventory launcher proved that registry support is effectively fixed to `default` today.
- The request-shape mismatch between frontend and backend gave a concrete reason to simplify the protocol rather than a merely stylistic reason.

### What didn't work

- The generated doc skeleton again started with an empty `RelatedFiles: []` field, so I replaced it with the populated file list while writing the final guide.
- The current codebase still contains a large amount of profile-admin scaffolding in `profileApi.ts`, which makes the actual live usage smaller than the apparent API surface. I had to distinguish "defined helper" from "used runtime path" carefully.

### What I learned

- The strongest simplification is not just deleting `ExtensionSchemas`; it is removing the assumption that shared chat core and rich profile management are the same subsystem.
- The current product can plausibly run with just:
  - `/chat`
  - `/ws`
  - `/api/timeline`
  - `/api/chat/profiles`
- If the frontend sends `runtime_key` explicitly, even `/api/chat/profile` can likely leave the core design.

### What was tricky to build

The tricky part here was deciding whether "minimal profile surface" should still include the current-profile cookie route. The current UI does use that route today, but the cleaner design is actually to remove it from the core and send `runtime_key` explicitly from the client. That means the guide had to distinguish between the preferred target model and a possible migration compatibility step.

### What warrants a second pair of eyes

- Whether to keep a temporary compatibility route for `/api/chat/profile`
- Whether the minimal list endpoint should still return `is_default`
- Whether any app really needs multi-registry support soon enough to justify keeping registries in the base contract

### What should be done in the future

- Align the frontend transport with `runtime_key` explicitly.
- Split the existing `profileApi.ts` into list-only and admin/editor concerns.
- If profile authoring becomes a real product feature later, create a separate optional `profile-admin` package instead of expanding the core.

### Code review instructions

- Start with the third design doc:
  - `design-doc/03-intern-guide-tight-chat-core-with-minimal-profile-surface-and-no-extension-schemas.md`
- Then verify the live frontend call paths:
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts`
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts`
- Then compare that against the current backend/profile surface:
  - `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go`
- Validation commands:
  - `docmgr doctor --ticket APP-04-OS-CHAT-PLATFORM --stale-after 30`
  - `remarquee upload bundle --dry-run ...`

### Technical details

Key commands run:

```bash
docmgr doc add --ticket APP-04-OS-CHAT-PLATFORM --doc-type design-doc --title "Intern Guide: Tight Chat Core with Minimal Profile Surface and No Extension Schemas"
rg -n "api/chat/schemas/extensions|api/chat/schemas/middlewares|api/chat/profiles|api/chat/profile|useProfiles|useSetProfile|chatProfiles" workspace-links/go-go-os-frontend workspace-links/go-go-app-inventory -S
rg -n "RegisterProfileAPIHandlers|ExtensionSchemas|MiddlewareDefinitions|DefaultRegistrySlug|WriteActor|WriteSource" . -S
nl -ba workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts | sed -n '1,240p'
nl -ba workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts | sed -n '1,220p'
nl -ba workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileApi.ts | sed -n '220,390p'
nl -ba workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useConversation.ts | sed -n '1,260p'
nl -ba workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts | sed -n '1,120p'
nl -ba workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts | sed -n '1,220p'
nl -ba /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/api.go | sed -n '1,120p'
```

Primary outputs created:

- `design-doc/03-intern-guide-tight-chat-core-with-minimal-profile-surface-and-no-extension-schemas.md`
- `reference/01-investigation-diary.md` (updated)

## Step 4: Extract the Shared Backend Packages into go-go-os-chat

This step turned the design work into a real shared repo. The goal was to extract the smallest reusable backend slice first so inventory could migrate onto real code instead of continuing to target a document-only architecture.

The key design decision was to keep the first repo backend-only and narrow. I did not move docs, reflection, UI embedding, or HyperCard-specific logic into `go-go-os-chat`. The new repo only contains the mountable route layer and the profile-aware resolver/composer that were already effectively generic.

### Prompt Context

**User prompt (verbatim):** "perfect, now let's go back to the APP-04 ticket for the os chat platform, and implement it and port over the inventory app to use the shared framework."

**Assistant interpretation:** Start implementing the APP-04 design now: create the shared backend repo, extract the generic chat code into it, and then move inventory onto the shared code.

**Inferred user intent:** Convert APP-04 from architecture planning into working code, while keeping the implementation clean enough that inventory becomes just one consumer of a shared OS chat platform.

**Commit (code):** `48a629b` — `feat: extract shared os chat backend packages`

### What I did

- Created a new sibling repo at `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat` and initialized it as its own Git repository.
- Added the new repo to both workspace files so `wesen-os` and the larger root workspace can resolve it locally:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go.work`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/go.work`
- Added two backend-only packages:
  - `pkg/profilechat`
  - `pkg/chatservice`
- Extracted the generic inventory request resolver into:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/request_resolver.go`
- Extracted the generic runtime composer into:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/runtime_composer.go`
- Added a mountable route component with optional profile API support and no static UI mount:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/chatservice/component.go`
- Copied and adapted the resolver/composer tests into the new repo:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/request_resolver_test.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/runtime_composer_test.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/chatservice/component_test.go`
- Ran:
  - `go mod tidy`
  - `go test ./... -count=1`

### Why

- Inventory already contained the reusable logic; the risk was continuing to duplicate it by implementing the assistant backend first.
- Extracting the shared package before touching inventory gave a clean dependency direction:
  - `go-go-os-chat` becomes the platform
  - inventory becomes a consumer
  - `wesen-os` can mount a generic assistant directly on top of the same package

### What worked

- The extracted code was already generic enough that the first shared repo passed tests with only package/import changes.
- The shared route component was able to drop static UI entirely without impacting the route contract inventory actually uses.
- The initial package split lined up well with the tight-core design:
  - `profilechat` for runtime/profile resolution
  - `chatservice` for transport/profile/timeline mounting

### What didn't work

- I accidentally ran `gofmt -w go.mod`, which failed with:

```text
go.mod:1:1: expected 'package', found module
```

- Running `go test ./... -count=1` from the new repo failed at first because the root workspace `go.work` did not include `go-go-os-chat` yet:

```text
pattern ./...: directory prefix . does not contain modules listed in go.work or their selected dependencies
```

- I fixed that by adding `./go-go-os-chat` to the root workspace file and `../go-go-os-chat` to `wesen-os/go.work`.

### What I learned

- The extracted backend can stay very small and still be useful.
- The right first shared package is not a giant “OS chat platform”; it is a narrow backend seam that other repos can actually adopt immediately.
- Removing static UI from the shared layer was a clean cut, not a compromise.

### What was tricky to build

The main sharp edge was Go workspace resolution. The new repo was local-only and unpublished, so workspace wiring mattered immediately. The tests passed once the workspace files included the repo, but the later `go mod tidy` work in consuming modules exposed the next issue: unpublished workspace-only modules need explicit handling if the consuming repo expects standalone module hygiene.

### What warrants a second pair of eyes

- Whether `pkg/chatservice` should stay named that way long term or be renamed later
- Whether `chatservice.Component` should keep optional profile mounting in the same type or split further later
- Whether a future `profile-admin` package should live in this repo or stay elsewhere

### What should be done in the future

- Publish/version `go-go-os-chat` so consuming repos can replace the temporary local-workspace dependency story with normal module versions.
- Decide whether to add README/project plumbing to the new repo once the API shape stabilizes.

### Code review instructions

- Start with the new repo:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/chatservice/component.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/request_resolver.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/runtime_composer.go`
- Then verify tests:
  - `go test ./... -count=1` in `/home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat`

### Technical details

Key commands run:

```bash
mkdir -p /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/chatservice /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat
git -C /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat init
gofmt -w pkg/chatservice/component.go pkg/chatservice/component_test.go pkg/profilechat/request_resolver.go pkg/profilechat/request_resolver_test.go pkg/profilechat/runtime_composer.go pkg/profilechat/runtime_composer_test.go
go mod tidy
go test ./... -count=1
```

## Step 5: Port Inventory to the Shared Backend Packages and Remove ExtensionSchemas from the Live Path

This step migrated inventory onto the new shared repo without changing its user-facing route shape. The main change was architectural: inventory stopped owning generic resolver/composer/route-mount logic and became an app-specific shell around the shared backend packages.

I also removed `ExtensionSchemas` from the live inventory path while doing the migration. That matched the tighter design and the actual frontend behavior, and it removed one of the main pieces of dead complexity the earlier ticket discussion had identified.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue the APP-04 implementation by switching inventory over to the new shared backend repo and trimming unnecessary profile/schema surface on the way.

**Inferred user intent:** Reduce duplication and cruft immediately, not just create a new repo that inventory still ignores.

**Commit (code):** `68906df` — `refactor: use shared os chat backend packages`

### What I did

- Replaced inventory’s old generic request resolver implementation with a thin alias wrapper:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
- Replaced inventory’s old generic runtime composer implementation with a thin wrapper that only injects inventory defaults and middleware definitions:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
- Reworked inventory’s backend component into an app-specific wrapper around the shared route component:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
- Removed `ExtensionSchemas` from inventory module options and construction:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/cmd/wesen-os-launcher/main.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/cmd/wesen-os-launcher/main_integration_test.go`
- Added a direct module dependency placeholder in:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/go.mod`
- Validated inventory after the port with:
  - `go test ./pkg/... -count=1` in `go-go-app-inventory`

### Why

- A shared repo is not useful until the first consumer actually stops owning duplicate logic.
- Inventory’s backend still needed to own:
  - manifest/capabilities
  - docs/reflection
  - HyperCard middleware/event logic
- But it no longer needed to own:
  - generic request resolution
  - generic runtime composition algorithm
  - generic mountable chat/timeline/profile transport

### What worked

- The inventory tests passed after a much smaller code diff than the original implementation had implied.
- The thin-wrapper approach preserved inventory-owned defaults while deleting a large amount of duplicated generic logic.
- Removing `ExtensionSchemas` from the live path did not affect the current backend contract that the frontend actually uses.

### What didn't work

- `go mod tidy` in both `go-go-app-inventory` and `wesen-os` tried to resolve the unpublished local module from the network instead of treating it as a released dependency:

```text
go: finding module for package github.com/go-go-golems/go-go-os-chat/pkg/profilechat
go: finding module for package github.com/go-go-golems/go-go-os-chat/pkg/chatservice
```

- I did not keep pushing on `go mod tidy` after that, because the underlying issue is release/versioning, not a compile problem. I added explicit `require github.com/go-go-golems/go-go-os-chat v0.0.0` entries instead so the local workspace build remains explicit.

### What I learned

- Inventory can become a clean app wrapper without a disruptive rewrite.
- The `ExtensionSchemas` removal is practical now, not just theoretical.
- The main remaining long-term hygiene issue is versioning the new repo, not code structure.

### What was tricky to build

The tricky part was keeping the migration honest. It would have been easy to “port” inventory by leaving the generic code in place and only changing a few import sites. Instead, I replaced the generic logic in inventory with wrappers that are obviously just inventory-specific defaults layered on top of `go-go-os-chat`. That makes future drift easier to spot in code review.

### What warrants a second pair of eyes

- The `v0.0.0` placeholder dependency approach until the new repo is published
- Whether any hidden frontend/debug tooling still expected the old embedded UI route under inventory
- Whether the inventory backend component should advertise the same capabilities now that the shared service no longer mounts UI

### What should be done in the future

- Publish `go-go-os-chat` and replace the placeholder dependency.
- Decide whether inventory should keep the compatibility wrappers indefinitely or switch callers to the shared package directly.

### Code review instructions

- Compare old-versus-new ownership in:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
- Validation:
  - `go test ./pkg/... -count=1` in `go-go-app-inventory`

### Technical details

Key commands run:

```bash
gofmt -w cmd/wesen-os-launcher/main.go cmd/wesen-os-launcher/main_integration_test.go workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go workspace-links/go-go-app-inventory/pkg/backendmodule/module.go workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go
go test ./pkg/... -count=1
go mod tidy
```

## Step 6: Mount the Shared Assistant Backend Module in wesen-os

This step completed the APP-04 implementation by putting the extracted packages to use directly from `wesen-os`, not only through inventory. The result is a real backendhost app module named `assistant` that exposes namespaced chat/profile/timeline routes on the same shared platform inventory now uses.

That matters because it proves the extraction is not just “inventory code moved around.” `wesen-os` now has one generic assistant backend mounted once, alongside the app-specific inventory backend.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Finish the APP-04 implementation by mounting a shared assistant backend module in `wesen-os` and validating that the launcher serves it.

**Inferred user intent:** End the ticket with a concrete OS-level chat platform outcome, not just a new library and one migrated consumer.

**Commit (code):** `856109f` — `feat: mount shared assistant chat backend`

### What I did

- Added a new backendhost module package:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/pkg/assistantbackendmodule/module.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/pkg/assistantbackendmodule/module_test.go`
- Mounted the assistant backend in the launcher by:
  - creating a shared `profilechat.RuntimeComposer`
  - creating a one-profile in-memory registry with the `assistant` profile slug
  - creating a shared `profilechat.StrictRequestResolver`
  - creating a `webchat.Server` with no static UI FS
  - adding the new assistant module to the backend module registry
- Updated the integration test helper to mount the assistant module in the in-test server as well:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/cmd/wesen-os-launcher/main_integration_test.go`
- Added integration coverage for:
  - assistant presence in `/api/os/apps`
  - assistant profile list at `/api/apps/assistant/api/chat/profiles`
- Added a direct module dependency placeholder in:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/go.mod`
- Validated with:
  - `go test ./pkg/assistantbackendmodule ./cmd/wesen-os-launcher/... -count=1`

### Why

- The shared repo needed a second concrete consumer besides inventory to justify its existence.
- The assistant module is the natural generic OS-level consumer.
- Mounting it once in `wesen-os` matches the long-term platform direction from the design docs.

### What worked

- The assistant module could stay very small because the shared packages now own the hard parts.
- The launcher integration tests passed after adding the assistant module and its manifest/profile endpoint assertions.
- The namespaced route model stayed consistent:
  - `/api/apps/assistant/chat`
  - `/api/apps/assistant/ws`
  - `/api/apps/assistant/api/timeline`
  - `/api/apps/assistant/api/chat/profiles`

### What didn't work

- Nothing failed in a substantive way after the earlier workspace and module-resolution fixes. The assistant slice mostly confirmed the extraction was sound.

### What I learned

- The extracted shared packages are already good enough to support a second backend module with minimal new code.
- The best proof of the architecture was not another design doc; it was the assistant module fitting cleanly into `backendhost` next to inventory.

### What was tricky to build

The subtle part was deciding what “assistant module” should mean at this stage. I kept it intentionally backend-only: no new launcher UI app, no docs, no custom widgets, and no special tools. That keeps APP-04 focused on platform extraction and mount shape, while leaving APP-05 and later frontend work free to decide how this generic assistant should be presented in the UI.

### What warrants a second pair of eyes

- Whether the app id should stay `assistant` or be renamed later
- Whether the assistant module should expose reflection/doc endpoints in a later step
- Whether the single built-in `assistant` profile is enough for now or whether default/analyst/planner-style variants are warranted later

### What should be done in the future

- Add a frontend launcher surface for the assistant module if/when needed.
- Build APP-05 on top of this shared backend by adding app/bootstrap context injection.

### Code review instructions

- Start with the new assistant module:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/pkg/assistantbackendmodule/module.go`
- Then verify the launcher assembly:
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/cmd/wesen-os-launcher/main.go`
  - `/home/manuel/workspaces/2026-03-02/os-openai-app-server/wesen-os/cmd/wesen-os-launcher/main_integration_test.go`
- Validation:
  - `go test ./pkg/assistantbackendmodule ./cmd/wesen-os-launcher/... -count=1`

### Technical details

Key commands run:

```bash
gofmt -w cmd/wesen-os-launcher/main.go cmd/wesen-os-launcher/main_integration_test.go pkg/assistantbackendmodule/module.go pkg/assistantbackendmodule/module_test.go
go test ./pkg/assistantbackendmodule ./cmd/wesen-os-launcher/... -count=1
```
