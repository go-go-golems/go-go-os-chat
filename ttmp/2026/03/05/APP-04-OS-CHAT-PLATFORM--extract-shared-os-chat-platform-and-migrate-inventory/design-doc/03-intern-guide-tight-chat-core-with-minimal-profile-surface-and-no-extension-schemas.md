---
Title: 'Intern Guide: Tight Chat Core with Minimal Profile Surface and No Extension Schemas'
Ticket: APP-04-OS-CHAT-PLATFORM
Status: active
Topics:
    - architecture
    - backend
    - chat
    - wesen-os
DocType: design-doc
Intent: long-term
Owners: []
ExternalSources: []
Summary: Detailed intern-facing guide for a tighter APP-04 target architecture: a minimal shared chat core, a small optional profile-list endpoint, and no extension-schema or profile-admin surface in the core contract.
LastUpdated: 2026-03-06T09:12:00-05:00
WhatFor: Define the leanest useful shared OS chat platform that matches current product needs and explicitly excludes extension schemas, profile CRUD, and registry complexity from the core design.
WhenToUse: Use this guide when implementing the first shared go-go-os-chat extraction, simplifying the current profile API surface, or reviewing whether a chat concern belongs in the shared core versus a later optional admin feature.
RelatedFiles:
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx
      Note: Current chat window only needs profile list and selected-profile state when the selector is enabled
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts
      Note: Live frontend code path that lists profiles and reads current-profile state
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts
      Note: Current profile-selection write path that can be simplified away from cookie-based persistence
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts
      Note: Current chat submit payload shape and the opportunity to send runtime_key explicitly
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts
      Note: Current websocket URL query shape and the opportunity to send runtime_key explicitly
    - Path: workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx
      Note: Inventory chat window hardcodes the default registry and only uses profiles for a dropdown/menu
    - Path: cmd/wesen-os-launcher/main.go
      Note: Current backend wiring only creates one registry slug and still passes extension schemas into the inventory module
    - Path: workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go
      Note: Current inventory backend component mounts the full profile/schema surface
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go
      Note: Current resolver expects runtime_key and cookie state rather than the frontend's profile field names
    - Path: /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go
      Note: Upstream full profile-admin API surface that should move out of the tight core
    - Path: /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/api.go
      Note: Canonical chat request shape with runtime_key and registry_slug fields
---

# Intern Guide: Tight Chat Core with Minimal Profile Surface and No Extension Schemas

## Executive Summary

This document is the third refinement of APP-04. It intentionally narrows the target architecture further than the previous two guides.

The main conclusion is:

- the shared `go-go-os-chat` core should be very small
- `ExtensionSchemas` should not be part of the core design at all
- the base profile feature should be reduced to a simple list endpoint for populating a dropdown
- all registry plumbing, profile CRUD, middleware schema APIs, extension schema APIs, and profile-write audit metadata should move out of the core design

This is not a statement that those richer features are bad. It is a statement that they are not needed to deliver the current product behavior.

Today the live UI uses profile data only to:

- populate a dropdown/menu
- remember which profile is selected for a conversation scope
- send that selection with chat traffic

It does not currently use:

- extension schema discovery
- middleware schema discovery
- profile create/update/delete
- set-default profile
- multi-registry selection in any meaningful way

Because of that, the right first shared platform is a tight chat core, not a full profile-management platform.

## Reading Guide

Read this guide in the following order:

1. `Current Usage Reality`
2. `Problem Statement`
3. `Proposed Tight Core`
4. `Minimal Profile Surface`
5. `Implementation Plan`

If you only want the answer to "do we need ExtensionSchemas and registries in the core?", jump to `Short Answer`.

## Current Usage Reality

The design should start from actual product usage, not from the widest API we can imagine.

### What the live frontend actually uses

When `ChatConversationWindow` enables the profile selector, it only wires three behaviors:

- list profiles
- read the currently selected profile state
- write a newly selected profile state

Evidence:

- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx:91-107`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts:84-126`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts:14-47`

`useProfiles` does two HTTP calls:

- `GET /api/chat/profiles`
- sometimes `GET /api/chat/profile`

See `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts:91-99` and the underlying API helpers in `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileApi.ts:237-245` and `:334-353`.

`useSetProfile` does one HTTP call:

- `POST /api/chat/profile`

See `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts:27-37` and `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileApi.ts:342-353`.

### What the live frontend does not use

The frontend package defines helpers for:

- `getProfile`
- `createProfile`
- `updateProfile`
- `deleteProfile`
- `setDefaultProfile`
- `listMiddlewareSchemas`
- `listExtensionSchemas`

See `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileApi.ts:248-369`.

However, repository search shows no live runtime call sites for those functions outside tests and a mock fixture. The real chat window path does not use them.

### What inventory currently does with profiles

Inventory hardcodes the registry to `"default"` in the actual chat window:

- `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx:947-955`

The backend also only builds one registry slug:

- `cmd/wesen-os-launcher/main.go:84`
- `cmd/wesen-os-launcher/main.go:185-230`

So the registry dimension is not currently buying meaningful product behavior. It is just extra plumbing.

### What `ExtensionSchemas` are currently doing

The only explicit extension schema currently passed into the inventory module is the starter-suggestions schema:

- `cmd/wesen-os-launcher/main.go:449-468`

That is useful only for a profile-editor or schema-driven authoring flow. It is not needed to run the current chat window or profile dropdown.

## Short Answer

No, the tight shared chat core does not need `ExtensionSchemas`.

It also does not need, in the base contract:

- profile CRUD routes
- middleware schema routes
- extension schema routes
- registry slugs
- write actor/source audit options

The tight core needs only:

- chat transport
- timeline hydration
- websocket attach
- a runtime-selection mechanism
- optionally a simple profile list endpoint for UI dropdowns

## Problem Statement

The current APP-04 design conversation progressively discovered that the shared backend surface is trying to solve two different problems at once:

1. run chat conversations
2. expose a rich profile-management system

Those are not the same problem.

Running chat requires:

- a conversation transport
- a runtime builder
- a request resolver
- timeline reads
- websocket streaming

Profile management adds a different family of concerns:

- editing
- schema introspection
- registry selection
- cookie persistence
- audit metadata

If we put both into the first shared package, we will over-design the foundation and make `go-go-os-chat` heavier than the current product requires.

There is also a correctness reason to simplify. The frontend submit and websocket code currently sends `profile` and `registry`:

- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts:34-48`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts:83-90`

But the inventory resolver actually reads:

- `runtime_key`
- `registry_slug`
- `chat_profile` cookie

See `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go:21-29` and `:168-189`.

That mismatch is a signal that the current profile protocol is more complicated than it should be.

## Proposed Solution

Build a tight shared chat core that exposes only the functionality needed for current app chat behavior.

### Tight core responsibilities

- build and run `webchat.Server`
- mount `/chat`
- mount `/ws`
- mount `/api/timeline`
- optionally mount core `/api/*` helpers
- optionally mount the static UI handler
- resolve runtime selection from the request

### Tight core non-responsibilities

- profile CRUD
- middleware schema APIs
- extension schema APIs
- registry selection
- cookie-managed current-profile persistence
- docs
- reflection
- app-specific non-chat APIs

### Tight core mental model

```text
shared chat core = "run a conversation against a selected runtime"
optional profile list = "give the UI a list of valid runtime presets"
everything else = later optional admin/editor features
```

## Proposed Tight Core

### Package shape

The first shared extraction should look like this:

```text
go-go-os-chat/
  pkg/chatcore/
    server.go
    transport.go
    resolver.go
    runtime.go
  pkg/profilelist/
    list.go
```

There should be no `extensionschemas` package and no requirement that a chat-enabled app exposes profile-admin endpoints.

### Core API sketch

```go
type CoreOptions struct {
    Server          *webchat.Server
    RequestResolver webhttp.ConversationRequestResolver
    MountCoreAPI    bool
    MountUI         bool
    TimelineLogger  zerolog.Logger
}

type Core struct {
    server   *webchat.Server
    resolver webhttp.ConversationRequestResolver
    options  CoreOptions
}

func NewCore(opts CoreOptions) (*Core, error)
func (c *Core) MountRoutes(mux *http.ServeMux) error
```

Mounted routes:

```text
POST /chat
GET  /ws?conv_id=<id>[&runtime_key=<profile-slug>]
GET  /api/timeline?conv_id=<id>
GET  /api/debug/*        optional via server.APIHandler()
GET  /                  optional via server.UIHandler()
```

### Minimal runtime selection contract

The runtime-selection contract should become explicit and simple:

- frontend stores selected profile/runtime key locally per conversation scope
- frontend sends `runtime_key` on `/chat`
- frontend sends `runtime_key` on `/ws`
- backend resolver reads `runtime_key`
- no registry is needed in the tight core

That matches the canonical upstream request shape better than the current frontend payload does:

- `pkg/webchat/http/api.go:20-29`

### Minimal profile list feature

The only profile-related HTTP feature in the core design should be an optional list endpoint.

Suggested route:

```text
GET /api/chat/profiles
```

Suggested response shape:

```json
[
  {
    "slug": "default",
    "display_name": "Default",
    "description": "Baseline assistant",
    "is_default": true
  },
  {
    "slug": "inventory",
    "display_name": "Inventory",
    "description": "Tool-first inventory assistant",
    "is_default": false
  }
]
```

No registry field is necessary in the base shape if the current system only exposes one registry.

No `extensions` field is necessary in the base shape if the current UI is only populating a dropdown.

### Profile list provider sketch

```go
type ProfileListItem struct {
    Slug        string `json:"slug"`
    DisplayName string `json:"display_name,omitempty"`
    Description string `json:"description,omitempty"`
    IsDefault   bool   `json:"is_default,omitempty"`
}

type ProfileListProvider interface {
    ListProfiles(ctx context.Context) ([]ProfileListItem, error)
}

func MountProfileListRoute(mux *http.ServeMux, provider ProfileListProvider) error
```

This is a much smaller and more honest abstraction than passing a full `gepprofiles.Registry`, default registry slug, middleware definitions, extension schemas, and write metadata into every chat-enabled module.

## Minimal Frontend Model

The frontend can also simplify materially.

### Current state

The chat runtime currently stores:

- `availableProfiles`
- `selectedProfile`
- `selectedRegistry`
- `selectedByScope`

See `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/state/selectors.ts:26-33`.

### Proposed state

For the tight core, the UI only needs:

- `availableProfiles`
- `selectedProfileByScope`
- `loading`
- `error`

Registry state can drop from the base model.

Suggested shape:

```ts
type ChatProfilesState = {
  availableProfiles: Array<{
    slug: string;
    display_name?: string;
    description?: string;
    is_default?: boolean;
  }>;
  selectedByScope: Record<string, string | null>;
  loading: boolean;
  error: string | null;
};
```

### Proposed client API

Replace the current live dependency on `getCurrentProfile()` and `setCurrentProfile()` with a simpler contract:

- `listProfiles(basePrefix)`
- local selection state
- send `runtime_key` with `/chat` and `/ws`

That means:

- `useProfiles` remains
- `useSetProfile` becomes local-state only
- `useConversation` sends `runtime_key`
- no cookie persistence is required in the core design

## System Diagram

```text
                 +-------------------------------------+
                 |          app module wrapper         |
                 | docs, reflection, confirm, app APIs |
                 +-----------------+-------------------+
                                   |
                    +--------------+--------------+
                    |                             |
                    v                             v
           +-------------------+        +----------------------+
           | shared chat core  |        | optional profile     |
           | /chat /ws         |        | list route only      |
           | /api/timeline     |        | /api/chat/profiles   |
           +---------+---------+        +----------------------+
                     |
                     v
         +--------------------------------------+
         | webchat.Server + resolver + runtime  |
         +----------------+---------------------+
                          |
                          v
         +--------------------------------------+
         | app-owned runtime presets/tools/etc. |
         +--------------------------------------+
```

## Design Decisions

### Decision 1: Remove `ExtensionSchemas` from the shared core entirely

Rationale:

- no live UI path uses them
- they are only useful for profile editors and schema-driven authoring
- they pull unnecessary abstraction weight into the first shared package

### Decision 2: Remove registry handling from the core profile surface

Rationale:

- the live app hardcodes `profileRegistry="default"` in `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx:953-955`
- the backend only creates one registry slug in `cmd/wesen-os-launcher/main.go:84` and `:185-230`
- the registry dimension is complexity without current product value

### Decision 3: Treat profile selection as runtime-key selection

Rationale:

- the chat transport ultimately needs a selected runtime
- current inventory resolver already thinks in terms of `runtime_key`
- using explicit `runtime_key` on requests is simpler and less error-prone than cookie-indirected selection

### Decision 4: Move profile-admin features to a later optional package

Rationale:

- CRUD and schema discovery are legitimate features
- they just do not belong in the first shared chat core

Suggested later package:

```text
go-go-os-chat/pkg/profileadmin
```

That later package can own:

- CRUD routes
- default-profile mutation
- middleware schema list
- extension schema list
- audit metadata

## API References

### Current live frontend usage

- profile listing and current-profile lookup:
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts:84-126`
- profile write:
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts:14-47`
- chat window wiring:
  - `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx:91-107`

### Current overexposed frontend API

- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileApi.ts:248-369`

### Current backend request shape

- upstream canonical fields:
  - `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/api.go:20-29`
- inventory resolver:
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go:21-29`
  - `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go:157-210`

### Current full profile-admin API surface

- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go:137-553`

## Pseudocode for the Tight Core

### Backend core

```go
type TightResolver struct {
    defaultRuntimeKey string
    runtimeProvider   RuntimeProvider
}

func (r *TightResolver) Resolve(req *http.Request) (webhttp.ResolvedConversationRequest, error) {
    switch req.Method {
    case http.MethodPost:
        body := decodeChatBody(req)
        runtimeKey := strings.TrimSpace(body.RuntimeKey)
        if runtimeKey == "" {
            runtimeKey = r.defaultRuntimeKey
        }
        convID := ensureConvID(body.ConvID)
        resolvedRuntime := r.runtimeProvider.Resolve(runtimeKey)
        return webhttp.ResolvedConversationRequest{
            ConvID:          convID,
            RuntimeKey:      runtimeKey,
            ResolvedRuntime: resolvedRuntime,
            Prompt:          body.Prompt,
        }, nil
    case http.MethodGet:
        convID := requireConvID(req)
        runtimeKey := queryOrDefault(req, "runtime_key", r.defaultRuntimeKey)
        resolvedRuntime := r.runtimeProvider.Resolve(runtimeKey)
        return webhttp.ResolvedConversationRequest{
            ConvID:          convID,
            RuntimeKey:      runtimeKey,
            ResolvedRuntime: resolvedRuntime,
        }, nil
    default:
        return errorMethodNotAllowed()
    }
}
```

### Frontend submit path

```ts
async function submitPrompt(prompt: string, convId: string, basePrefix: string, runtimeKey?: string) {
  await fetch(`${basePrefix}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      prompt,
      conv_id: convId,
      runtime_key: runtimeKey,
    }),
  });
}
```

### Frontend websocket path

```ts
function buildWsUrl(basePrefix: string, convId: string, runtimeKey?: string) {
  const params = new URLSearchParams({ conv_id: convId });
  if (runtimeKey) params.set('runtime_key', runtimeKey);
  return `${wsBase(basePrefix)}/ws?${params.toString()}`;
}
```

### Frontend profile selection

```ts
function useProfileSelector(scopeKey: string) {
  const profiles = useProfiles(basePrefix);
  const selected = useSelector((state) => state.chatProfiles.selectedByScope[scopeKey] ?? null);
  const setSelected = (slug: string | null) =>
    dispatch(chatProfilesSlice.actions.setSelectedProfile({ scopeKey, profile: slug }));
  return { profiles, selected, setSelected };
}
```

## Alternatives Considered

### Alternative 1: Keep the full current profile API surface in the shared core

Rejected because:

- it solves more than the current product requires
- it forces `ExtensionSchemas` and admin-oriented metadata into the base abstraction
- it keeps registry and schema complexity alive without a current consumer

### Alternative 2: Keep current-profile cookie endpoints in the tight core

Rejected for the preferred design because:

- local per-conversation selection plus explicit `runtime_key` is clearer
- cookie-based selection is awkward when multiple chat windows can have different active profiles
- it hides runtime selection behind a second mechanism

This route can still exist temporarily for migration if needed, but it should not define the new core abstraction.

### Alternative 3: Remove profile listing entirely

Rejected because:

- the current UI does need a profile list to populate the dropdown and menus
- exposing a small read-only list endpoint is a reasonable minimal feature

## Implementation Plan

### Phase 1: Define the tight shared core interfaces

Create shared packages for:

- core chat transport mount
- tight request resolver contract
- runtime builder/provider
- optional profile list route

Do not include:

- profile CRUD
- schema list routes
- extension schemas
- registry slugs
- write actor/source

### Phase 2: Simplify the frontend contract

Refactor the chat runtime so that:

- profile selection is local state by scope
- `useConversation` sends `runtime_key`
- `wsManager` sends `runtime_key`
- `useSetProfile` no longer depends on `/api/chat/profile`

This is the most important cleanup because it aligns the actual UI model with the backend runtime-selection model.

### Phase 3: Refactor inventory to use the tight core

Inventory should expose:

- shared chat core routes
- optional profile list route
- its own docs/reflection/confirm routes separately

It should stop exposing the richer profile-admin surface unless there is an explicit product need.

### Phase 4: Reserve richer profile/admin APIs for later

If later work needs:

- authoring profiles
- editing middleware configs
- inspecting extension schemas

then create a separate optional package and ticket for that work. Do not re-expand the core preemptively.

## Concrete File-Level Guidance

### Backend files to simplify

- `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
  - remove full `RegisterProfileAPIHandlers(...)` from the base shared shape
- `cmd/wesen-os-launcher/main.go`
  - stop passing `ExtensionSchemas`, `DefaultRegistrySlug`, `WriteActor`, and `WriteSource` into the base chat feature

### Frontend files to simplify

- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts`
  - send `runtime_key`, not `profile`/`registry`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts`
  - send `runtime_key`, not `profile`/`registry`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts`
  - make selection local-state only
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/profileApi.ts`
  - split list-only functionality from admin/editor helpers

## Testing and Validation Strategy

### Backend tests

- `/chat` honors explicit `runtime_key`
- `/ws` honors explicit `runtime_key`
- `/api/timeline` unchanged
- `/api/chat/profiles` returns a simple list
- no dependency on extension schemas in the base integration path

### Frontend tests

- profile dropdown populates from list route
- selecting a profile updates local scoped state
- `useConversation` sends `runtime_key`
- websocket reconnect changes when `runtime_key` changes

### Regression checks

- multiple chat windows can use different runtime selections without cookie collisions
- inventory menu/profile cycling still works
- no root `/chat` or `/ws` alias regression

## Open Questions

- Do we want to keep a temporary compatibility shim for `/api/chat/profile` during migration, or remove it immediately once the frontend is updated?
- Should the minimal profile list endpoint continue returning `is_default`, or should defaulting become purely a frontend concern?
- If a later app truly needs multiple registries, should that appear as a separate admin feature instead of re-expanding the core list API?

## References

- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useProfiles.ts`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/useSetProfile.ts`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/http.ts`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/ws/wsManager.ts`
- `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx`
- `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
- `cmd/wesen-os-launcher/main.go`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/api.go`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go`

## Problem Statement

<!-- Describe the problem this design addresses -->

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
