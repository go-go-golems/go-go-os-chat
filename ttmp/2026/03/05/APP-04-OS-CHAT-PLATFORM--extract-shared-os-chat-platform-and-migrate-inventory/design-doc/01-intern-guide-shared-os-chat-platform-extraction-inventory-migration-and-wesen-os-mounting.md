---
Title: 'Intern Guide: Shared OS Chat Platform Extraction, Inventory Migration, and Wesen-OS Mounting'
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
RelatedFiles:
    - Path: cmd/wesen-os-launcher/main.go
      Note: Current inventory chat backend assembly and future assistant-module mount site
    - Path: workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go
      Note: Genericizable route adapter around webchat.Server
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_events.go
      Note: Inventory-specific optional extension boundary
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go
      Note: Genericizable strict request resolver for chat and websocket routes
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go
      Note: Genericizable runtime composition algorithm
    - Path: workspace-links/go-go-os-backend/pkg/backendhost/routes.go
      Note: Namespaced backend module contract and legacy alias guard
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx
      Note: Evidence that frontend chat window is already generic
ExternalSources: []
Summary: Detailed intern guide for extracting shared Go chat backend infrastructure into go-go-os-chat, migrating inventory to consume it without behavior change, and mounting a reusable assistant backend module in wesen-os.
LastUpdated: 2026-03-05T21:03:42.493593778-05:00
WhatFor: Explain the current chat architecture, separate generic platform code from inventory-owned semantics, and provide a phased implementation plan for steps 1-3 of the OS chat platform roadmap.
WhenToUse: Use this guide when implementing APP-04, onboarding a new engineer to the chat stack, or reviewing whether code belongs in inventory, wesen-os, or the future go-go-os-chat repo.
---


# Intern Guide: Shared OS Chat Platform Extraction, Inventory Migration, and Wesen-OS Mounting

## Executive Summary

This ticket groups three platform steps that belong together:

1. Extract shared Go backend chat infrastructure into a new `go-go-os-chat` repository.
2. Port the inventory app to consume that shared layer without changing behavior or public routes.
3. Mount one shared assistant backend module in `wesen-os`.

These three steps are the platform prerequisite for later work such as generic "chat with app" bootstrapping. Today, the frontend chat window and Redux/runtime package are already mostly shared, but the Go-side backend plumbing is split across the inventory repository and the `wesen-os` launcher composition root. The system therefore behaves like a platform, but the code is still organized like a feature-specific implementation.

The long-term goal is not "inventory chat made reusable." The long-term goal is one OS-level chat platform that many entry points can use: inventory chat, generic assistant chat, workspace chat, and later "chat with app" or "chat with selection." Inventory should remain an important consumer of that platform, but not its owner.

The recommended design is:

- create `go-go-os-chat` as a Go-first repository for shared backend chat plumbing
- keep app meaning inside app repositories: tools, prompts, docs, and app-specific extensions
- keep the TypeScript `@hypercard/chat-runtime` package where it is for now
- mount a shared backend module in `wesen-os` under an app id such as `assistant`
- leave the future app-specific bootstrap/context injection work to a separate ticket (`APP-05`)

## Reading Guide

If you are new to this codebase, read these sections in order:

1. `Problem Statement and Scope`
2. `Current-State Architecture`
3. `Gap Analysis`
4. `Proposed Solution`
5. `Implementation Plan`

If you only need the answer to "what should move where?", jump to `What Moves, What Stays, and Why`.

## Problem Statement and Scope

The current chat system already contains clear platform-like pieces, but those pieces are spread across repositories in a way that makes generic OS chat work harder than it should be.

Observed current state:

- The frontend already consumes a shared chat runtime package. `ChatConversationWindow` accepts a generic `basePrefix` and `convId`, which means the main chat window itself is already backend-agnostic (`workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx:47-59`, `:85-107`).
- Inventory already consumes that shared package in its app store and launcher wiring (`workspace-links/go-go-app-inventory/apps/inventory/src/app/store.ts:1-21`, `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx:1-18`, `:887-997`).
- The Go backend side is not similarly centralized. `wesen-os` assembles the inventory chat runtime directly in the launcher `main.go`, including the runtime composer, profile registry, request resolver, `webchat.Server`, tool registration, and backend module instantiation (`cmd/wesen-os-launcher/main.go:179-249`, `:281-295`).
- The inventory repo owns several pieces that are generic in shape:
  - the namespaced backend route adapter (`workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go:42-155`)
  - the strict request resolver for `/chat` and `/ws` (`workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go:15-277`)
  - the runtime composition algorithm that applies profiles, tools, middleware, and step settings (`workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go:18-149`)
- The backend host already expects namespaced app modules mounted under `/api/apps/<app-id>` and explicitly forbids legacy aliases like `/chat`, `/ws`, and `/api/timeline` (`workspace-links/go-go-os-backend/pkg/backendhost/routes.go:12-16`, `:37-66`).

This means the platform boundary is already visible in the code, but it is not codified as a reusable package boundary.

### Why This Ticket Exists

Without this extraction:

- every app-specific chat backend will copy inventory's Go plumbing
- `wesen-os` will keep accumulating per-app chat assembly logic
- a shared assistant backend cannot become a stable OS primitive
- later app-chat bootstrap work will be forced to depend on inventory-shaped backend code

### Scope

In scope:

- extracting shared Go-side chat backend infrastructure into `go-go-os-chat`
- keeping inventory behavior unchanged while switching it to the shared layer
- mounting a new shared assistant backend module in `wesen-os`
- defining the code ownership boundary between platform and apps

Out of scope:

- generic app bootstrap/context injection for "chat with app"
- changing inventory’s UX
- moving the TypeScript `chat-runtime` package to a new repo right now
- rewriting inventory-specific HyperCard behavior unless it clearly proves to be an OS-wide extension

## Current-State Architecture

This section is intentionally evidence-heavy. The goal is to explain what exists today before proposing any moves.

### 1. Repository and Workspace Boundaries

At the frontend workspace level, `wesen-os` stitches together multiple linked repos. The root `package.json` and `pnpm-workspace.yaml` include app and package workspaces from `workspace-links/*` (`package.json:4-8`, `pnpm-workspace.yaml:1-4`). On the Go side, `go.work` includes the local `wesen-os` module plus linked app/backend repos such as `go-go-app-inventory` and `go-go-os-backend` (`go.work:3-10`).

That tells you two things:

- the current system is intentionally multi-repo
- shared code can live in a separate repository as long as it is wired into the workspace and `go.work`

The inventory repository also documents its current ownership boundary explicitly. Its README says it owns both the inventory backend/domain runtime and the inventory frontend app package, while shared desktop/platform APIs belong elsewhere (`workspace-links/go-go-app-inventory/README.md:3-18`, `:44-52`). That is a strong hint that the generic chat backend layer should not remain inventory-owned forever.

### 2. The Frontend Chat Runtime Is Already Mostly Generic

The TypeScript package `@hypercard/chat-runtime` is already a shared package, not an inventory package (`workspace-links/go-go-os-frontend/packages/chat-runtime/package.json:2-37`). Inventory imports reducers such as `chatProfilesReducer`, `chatSessionReducer`, `chatWindowReducer`, and `timelineReducer` directly from it (`workspace-links/go-go-app-inventory/apps/inventory/src/app/store.ts:1-21`).

The core chat window component is also backend-agnostic:

- it takes `convId`
- it takes `basePrefix`
- it optionally enables profile selection
- it internally uses `useConversation`, `useProfiles`, and `useSetProfile` against that base prefix

Evidence: `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx:47-59`, `:85-107`.

The launcher shell already provides app-specific base paths generically through `resolveApiBase(appId)` and `resolveWsBase(appId)` (`apps/os-launcher/src/App.tsx:22-35`, `workspace-links/go-go-os-frontend/packages/desktop-os/src/contracts/launcherHostContext.ts:3-9`).

#### Important Conclusion

The first bottleneck is not the frontend package location. The immediate bottleneck is the Go backend plumbing.

### 3. Inventory Uses Generic Frontend Runtime but App-Specific Launcher Glue

Inventory’s launcher module resolves the inventory backend namespace through the shared host context (`workspace-links/go-go-app-inventory/apps/inventory/src/launcher/module.tsx:31-60`). It then renders `InventoryLauncherAppWindow`, which dispatches to several inventory-specific windows, including the chat window (`workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx:999-1038`).

The inventory chat window itself is only a wrapper around `ChatConversationWindow`. It adds:

- inventory window title
- conversation-scoped profile selection
- event viewer / timeline buttons
- a debug toggle
- some inventory-specific context actions

Evidence: `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx:887-997`.

This is the right kind of app-specific code. It is launcher glue and app UX, not shared backend infrastructure.

### 4. `wesen-os` Currently Assembles the Inventory Chat Backend Directly

The most important current-state file is `cmd/wesen-os-launcher/main.go`.

Today it does all of the following for inventory chat:

- creates a runtime composer with inventory defaults (`:179-183`)
- registers inventory HyperCard extensions (`:184`)
- creates an in-memory profile registry with `default`, `inventory`, `analyst`, and `planner` profiles (`:185-225`)
- creates a strict request resolver bound to runtime key `inventory` (`:231-234`)
- creates a shared `webchat.Server` with runtime composer and inventory event sink wrapper (`:236-243`)
- registers inventory tools on the server (`:247-249`)
- wraps all of that in the inventory backend module and adds it to the backend module list (`:281-295`)

This is good evidence of what the platform actually needs, because all of these steps will still exist after extraction. They just should not all live in `main.go` and should not all be inventory-owned.

### 5. Inventory Backend Module Adapter Is Mostly Generic

The inventory backend component is a thin adapter around generic pinocchio/webchat services (`workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go:42-155`).

It mounts:

- `POST /chat`
- `GET /ws`
- profile API routes via `webhttp.RegisterProfileAPIHandlers`
- `GET /api/timeline`
- `m.server.APIHandler()` under `/api/`
- a plz-confirm mount
- the chat UI handler under `/`

That is not inventory domain logic. It is a generic app-backend route adapter with inventory labels baked into names, defaults, and error messages.

### 6. Request Resolution Is Generic in Shape

`StrictRequestResolver` handles both `POST /chat` and `GET /ws` (`workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go:53-72`).

For websocket requests it:

- requires `conv_id`
- optionally resolves the effective profile
- derives runtime key, resolved runtime, and profile version (`:74-107`)

For chat requests it:

- decodes `prompt`, `text`, `conv_id`, `runtime_key`, `registry_slug`, `request_overrides`, and `idempotency_key` (`:21-29`, `:110-154`)
- generates a `conv_id` if missing (`:123-126`)
- optionally resolves profile selection from request body, query string, or cookie (`:157-190`)
- resolves the effective runtime against the profile registry (`:192-210`)

Nothing in that algorithm is inherently inventory-specific except the fallback runtime key default.

### 7. Runtime Composition Is Generic in Shape

The runtime composer takes parsed CLI values plus runtime defaults and then applies profile runtime overrides, middleware resolution, tool allowlists, and runtime fingerprinting (`workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go:18-149`).

The generic parts are:

- clone baseline step settings from parsed values (`:79-89`)
- override them with profile `step_settings_patch` (`:84-88`)
- choose a runtime key (`:91-97`)
- choose the effective system prompt (`:99-105`)
- choose the effective allowed tool list (`:106-109`)
- normalize and resolve middleware definitions (`:111-122`, `:151-213`)
- build the engine from settings and middlewares (`:124-148`)

The non-generic parts are the inventory defaults baked into `NewRuntimeComposer`:

- inventory middleware definition registry (`:32-43`, `workspace-links/go-go-app-inventory/pkg/pinoweb/middleware_definitions.go:57-77`)
- inventory fallback runtime key/system prompt (`workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go:95-105`)

The algorithm should move. The defaults should not.

### 8. HyperCard Middleware and Event Plumbing Are Inventory-Specific Today

Inventory currently owns a substantial extension layer:

- middleware definitions for artifact policy, suggestions policy, and artifact generation (`workspace-links/go-go-app-inventory/pkg/pinoweb/middleware_definitions.go:57-180`)
- middleware that injects system blocks and enforces structured widget/card output (`workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_middleware.go:14-123`)
- event factory, SEM, and timeline registration for HyperCard artifacts (`workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_events.go:18-125`, `:167-408`)
- an event sink wrapper that extracts structured artifacts from assistant output (`workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_extractors.go:459-479`)

This is the main area where you must not over-extract too early.

Two possible truths can exist:

- HyperCard is inventory-only behavior.
- HyperCard is becoming an OS-wide artifact convention.

Until that is decided, treat HyperCard as an optional extension, not as chat core.

### 9. Backend Module Reflection and Docs Are Already App-Native

Inventory’s backend module exposes reflection and docs from inside the app repository (`workspace-links/go-go-app-inventory/pkg/backendmodule/module.go:43-61`, `:74-105`, `workspace-links/go-go-app-inventory/pkg/backendmodule/reflection.go:14-100`, `workspace-links/go-go-app-inventory/pkg/backendmodule/docs_store.go:9-14`).

This is the correct ownership model:

- app docs stay with the app
- app reflection stays with the app
- generic transport/runtime plumbing moves out

### 10. The Backend Host Already Provides the OS-Level Mounting Contract

`go-go-os-backend` already defines the exact contract that the shared assistant module should implement:

- `AppBackendModule` interface (`workspace-links/go-go-os-backend/pkg/backendhost/module.go:19-27`)
- optional reflection/docs interfaces (`:29-39`)
- app id validation and registry management (`workspace-links/go-go-os-backend/pkg/backendhost/registry.go:14-37`, `workspace-links/go-go-os-backend/pkg/backendhost/routes.go:18-35`)
- namespaced route mounting under `/api/apps/<app-id>` (`workspace-links/go-go-os-backend/pkg/backendhost/routes.go:37-55`)
- explicit rejection of legacy root aliases (`:12-16`, `:58-66`)
- `/api/os/apps` manifest/reflection discovery (`workspace-links/go-go-os-backend/pkg/backendhost/manifest_endpoint.go:38-123`)

This means the shared assistant module should be just another backendhost module, not a special-case server.

## Gap Analysis

The requested end state requires these gaps to be closed.

### Gap 1: Shared Go Chat Infrastructure Lives in Inventory-Specific Packages

Generic infrastructure currently lives in:

- `workspace-links/go-go-app-inventory/pkg/backendcomponent`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`

That makes inventory the accidental owner of OS-wide backend chat primitives.

### Gap 2: `wesen-os` Composition Knows Too Much About Inventory Internals

The composition root directly assembles runtime composer, profile registry, request resolver, webchat server, and tools for inventory (`cmd/wesen-os-launcher/main.go:179-249`). Some amount of composition will always belong in `wesen-os`, but the low-level backend plumbing should become library code instead of one-off assembly.

### Gap 3: No Generic Assistant Module Exists Yet

There is currently no mounted backend module whose job is "shared OS assistant chat." Inventory has a chat backend. The OS does not yet have a reusable assistant backend.

### Gap 4: Extraction Pressure Should Not Leak App Semantics into Platform Code

If the new repo blindly copies inventory, it will be wrong. Platform code should not own:

- inventory tool factories
- inventory system prompts
- inventory docs
- inventory reflection text
- inventory-only HyperCard policies unless they are intentionally promoted into optional shared extensions

### Gap 5: Moving the TypeScript Chat Package Now Would Add Versioning Friction

`@hypercard/chat-runtime` currently depends directly on `@hypercard/engine` as a workspace dependency (`workspace-links/go-go-os-frontend/packages/chat-runtime/package.json:18-37`). Multiple frontend packages and apps already consume it. Moving it into a separate repo before the Go extraction is stable would add packaging and version coordination without solving the main backend issue.

## Proposed Solution

The proposed solution is to create a new repository, `go-go-os-chat`, which becomes the shared Go-side chat platform layer above pinocchio and below app repos.

### Proposed Layering

```text
+---------------------------------------------------------------+
| Apps and app repos                                            |
| inventory / sqlite / future apps                              |
| own tools, prompts, docs, reflection, app-specific UX         |
+-------------------------------+-------------------------------+
                                |
                                v
+---------------------------------------------------------------+
| go-go-os-chat                                                 |
| shared backend module adapter                                 |
| shared request resolver                                       |
| shared runtime composer                                       |
| shared server assembly helpers                                |
| optional extension interfaces / optional shared extensions     |
+-------------------------------+-------------------------------+
                                |
                                v
+---------------------------------------------------------------+
| pinocchio / geppetto / backendhost                            |
| webchat.Server, timeline APIs, profile APIs, middleware cfg   |
+-------------------------------+-------------------------------+
                                |
                                v
+---------------------------------------------------------------+
| wesen-os composition root                                     |
| mounts app modules under /api/apps/<app-id>                   |
+---------------------------------------------------------------+
```

### Proposed Long-Term Runtime Shape

```text
Frontend window
    |
    | basePrefix=/api/apps/inventory or /api/apps/assistant
    v
ChatConversationWindow
    |
    v
backendhost namespaced app module
    |
    +--> shared route adapter
    +--> shared request resolver
    +--> shared runtime composer
    +--> pinocchio webchat server
             |
             +--> app-provided tools
             +--> app-provided docs/reflection
             +--> optional app-provided extensions
```

### Naming Decision

For the mounted shared module, prefer `assistant` as the app id.

Reasoning:

- it describes the OS concept, not the model provider
- it fits the backendhost contract cleanly as `/api/apps/assistant/...`
- it avoids over-coupling platform identity to `openai`

The repository name can still be `go-go-os-chat`. Repository names and mounted app ids do not need to match exactly.

## What Moves, What Stays, and Why

This is the most important ownership table in the document.

| Concern | Move to `go-go-os-chat`? | Why |
| --- | --- | --- |
| namespaced route adapter around `webchat.Server` | yes | generic app-backend plumbing |
| strict request resolver | yes | generic request normalization/profile resolution |
| runtime composition algorithm | yes | generic profile/middleware/tool/runtime assembly |
| server assembly helpers | yes | remove duplicate launcher composition code |
| inventory tool factories | no | domain semantics |
| inventory docs store | no | app-owned docs |
| inventory reflection doc | no | app-owned API/discovery text |
| inventory launcher window wiring | no | app UX |
| inventory HyperCard extension code | maybe later; not core now | optional extension, not platform core |
| TypeScript `@hypercard/chat-runtime` repo move | not in this ticket | backend extraction is the immediate blocker |

## Proposed Package Layout

This is one reasonable package layout for the new repo.

```text
go-go-os-chat/
  pkg/backendmodule/
    module.go
    reflection.go
  pkg/backendcomponent/
    component.go
  pkg/requestresolver/
    strict.go
  pkg/runtimecomposer/
    composer.go
  pkg/serverbuilder/
    builder.go
  pkg/extensions/
    interfaces.go
  pkg/modules/assistant/
    module.go
```

If you want fewer packages, `serverbuilder` and `extensions` can be folded into `pkg/backendmodule` or `pkg/runtimecomposer`. Do not optimize for elegance too early. Optimize for clear ownership boundaries.

## Proposed API Sketches

These are design sketches, not existing APIs.

### 1. Generic Backend Module Adapter

```go
type ModuleOptions struct {
    Manifest              backendhost.AppBackendManifest
    Server                *webchat.Server
    RequestResolver       webhttp.ConversationRequestResolver
    ProfileRegistry       gepprofiles.Registry
    DefaultRegistrySlug   gepprofiles.RegistrySlug
    MiddlewareDefinitions middlewarecfg.DefinitionRegistry
    ExtensionSchemas      []webhttp.ExtensionSchemaDocument
    WriteActor            string
    WriteSource           string
    ConfirmMountPath      string
    DocStore              *docmw.DocStore
    ReflectionProvider    func(context.Context) (*backendhost.ModuleReflectionDocument, error)
}

func NewModule(opts ModuleOptions) backendhost.AppBackendModule
```

Purpose:

- generalize `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
- allow inventory and assistant to share the same route mounting logic
- keep docs/reflection injectable instead of hard-coded

### 2. Generic Strict Request Resolver

```go
type StrictRequestResolver struct {
    defaultRuntimeKey   string
    profileRegistry     gepprofiles.Registry
    defaultRegistrySlug gepprofiles.RegistrySlug
}

func NewStrictRequestResolver(defaultRuntimeKey string) *StrictRequestResolver
func (r *StrictRequestResolver) WithProfileRegistry(reg gepprofiles.Registry, slug gepprofiles.RegistrySlug) *StrictRequestResolver
func (r *StrictRequestResolver) Resolve(req *http.Request) (webhttp.ResolvedConversationRequest, error)
```

Notes:

- keep the existing wire contract for `conv_id`, `runtime_key`, and `idempotency_key`
- preserve inventory compatibility in phase 2
- rename inventory-specific comments/errors/defaults

### 3. Generic Runtime Composer

```go
type RuntimeComposerOptions struct {
    DefaultRuntimeKey   string
    DefaultSystemPrompt string
    DefaultAllowedTools []string
    Definitions         middlewarecfg.DefinitionRegistry
    DefaultMiddlewares  []gepprofiles.MiddlewareUse
}

func NewRuntimeComposer(parsed *values.Values, opts RuntimeComposerOptions) *RuntimeComposer
func (c *RuntimeComposer) MiddlewareDefinitions() middlewarecfg.DefinitionRegistry
func (c *RuntimeComposer) Compose(ctx context.Context, req infruntime.ConversationRuntimeRequest) (infruntime.ComposedRuntime, error)
```

Important design rule:

- make the algorithm generic
- inject defaults from the app or assistant module

Do not keep inventory strings like `"You are an inventory assistant."` in the generic package.

### 4. Shared Server Assembly Helper

```go
type ServerBuildOptions struct {
    Parsed           *values.Values
    StaticFS         fs.FS
    RuntimeComposer  infruntime.RuntimeComposer
    EventSinkWrapper webchat.EventSinkWrapper
    ToolFactories    map[string]tool.Factory
    DebugRoutes      bool
}

func BuildServer(ctx context.Context, opts ServerBuildOptions) (*webchat.Server, error)
```

The goal is not to hide pinocchio. The goal is to centralize repetitive assembly and default validation.

### 5. Optional Extension Contract

This is only needed if HyperCard or similar behavior becomes reusable across multiple apps.

```go
type ChatExtension interface {
    Register()
    MiddlewareDefinitions() middlewarecfg.DefinitionRegistry
    DefaultMiddlewares() []gepprofiles.MiddlewareUse
    EventSinkWrapper(context.Context) webchat.EventSinkWrapper
    ExtensionSchemas() []webhttp.ExtensionSchemaDocument
}
```

If only inventory uses HyperCard, do not force this abstraction yet. Keep the interface small or skip it entirely until a second consumer exists.

## Proposed Module Topology After APP-04

After steps 1-3, the topology should look like this:

```text
/api/apps/inventory/*
    -> inventory wrapper module
    -> shared go-go-os-chat backend plumbing
    -> inventory tools/docs/reflection/extensions

/api/apps/assistant/*
    -> assistant module
    -> shared go-go-os-chat backend plumbing
    -> generic assistant profiles/docs/reflection
```

Important: inventory should continue to work during and after APP-04. The shared assistant module is additive in this ticket. It does not replace inventory chat yet.

## Pseudocode: End-to-End Wiring

### Current Shape

```go
composer := pinoweb.NewRuntimeComposer(parsed, inventoryDefaults)
profiles := newInMemoryProfileService(...)
resolver := pinoweb.NewStrictRequestResolver("inventory").WithProfileRegistry(profiles, registrySlug)
srv := webchat.NewServer(..., webchat.WithRuntimeComposer(composer), webchat.WithEventSinkWrapper(pinoweb.NewInventoryEventSinkWrapper(ctx)))
registerInventoryTools(srv, store)
modules = append(modules, inventorybackendmodule.NewModule(...))
```

### Target Shape for Inventory

```go
inventoryComposer := oschatruntime.NewRuntimeComposer(parsed, oschatruntime.RuntimeComposerOptions{
    DefaultRuntimeKey:   "inventory",
    DefaultSystemPrompt: "You are an inventory assistant. Be concise, accurate, and tool-first.",
    DefaultAllowedTools: inventorytools.InventoryToolNames,
    Definitions:         inventoryHypercardDefinitions,
    DefaultMiddlewares:  inventoryDefaultMiddlewares,
})

inventoryResolver := oschatresolver.NewStrictRequestResolver("inventory").
    WithProfileRegistry(inventoryProfiles, registrySlug)

inventoryServer := oschatserver.BuildServer(ctx, oschatserver.ServerBuildOptions{
    Parsed:           parsed,
    StaticFS:         inventoryStaticFS,
    RuntimeComposer:  inventoryComposer,
    EventSinkWrapper: inventoryEventSinkWrapper,
    ToolFactories:    inventorytools.InventoryToolFactories(store),
    DebugRoutes:      debugRoutesEnabled,
})

inventoryModule := inventorybackendmodule.NewModule(inventorybackendmodule.Options{
    SharedModuleOptions: oschatmodule.ModuleOptions{
        Manifest: backendhost.AppBackendManifest{
            AppID: "inventory",
            Name: "Inventory",
            Description: "Inventory chat runtime, profiles, timeline, and confirm APIs",
            Required: true,
            Capabilities: []string{"chat", "ws", "timeline", "profiles", "confirm"},
        },
        Server:                inventoryServer,
        RequestResolver:       inventoryResolver,
        ProfileRegistry:       inventoryProfiles,
        DefaultRegistrySlug:   registrySlug,
        MiddlewareDefinitions: inventoryComposer.MiddlewareDefinitions(),
        ExtensionSchemas:      inventoryExtensionSchemas(),
        DocStore:              inventoryDocStore,
        ReflectionProvider:    inventoryReflection,
    },
})
```

### Target Shape for Shared Assistant Module

```go
assistantComposer := oschatruntime.NewRuntimeComposer(parsed, oschatruntime.RuntimeComposerOptions{
    DefaultRuntimeKey:   "assistant",
    DefaultSystemPrompt: "You are the OS assistant. Be concise, accurate, and tool-aware.",
})

assistantResolver := oschatresolver.NewStrictRequestResolver("assistant").
    WithProfileRegistry(assistantProfiles, registrySlug)

assistantServer := oschatserver.BuildServer(ctx, oschatserver.ServerBuildOptions{
    Parsed:          parsed,
    StaticFS:        assistantStaticFS,
    RuntimeComposer: assistantComposer,
    ToolFactories:   assistantToolFactories,
    DebugRoutes:     debugRoutesEnabled,
})

assistantModule := assistantmodule.NewModule(oschatmodule.ModuleOptions{
    Manifest: backendhost.AppBackendManifest{
        AppID: "assistant",
        Name: "Assistant",
        Description: "Shared OS assistant chat backend",
        Required: false,
        Capabilities: []string{"chat", "ws", "timeline", "profiles", "confirm"},
    },
    Server:              assistantServer,
    RequestResolver:     assistantResolver,
    ProfileRegistry:     assistantProfiles,
    DefaultRegistrySlug: registrySlug,
    DocStore:            assistantDocStore,
    ReflectionProvider:  assistantReflection,
})
```

## Design Decisions

### Decision 1: Create a Go-First Shared Repo Before Moving Frontend Chat Runtime

Reason:

- the frontend chat runtime is already shared
- the current pain is in Go backend ownership and launcher assembly
- the TypeScript package has workspace coupling to `@hypercard/engine` and multiple existing consumers

### Decision 2: Keep Inventory Routes Stable During Migration

Inventory should continue to serve from `/api/apps/inventory/...` while it is being ported to the shared backend layer. That preserves existing launcher wiring and avoids mixing extraction work with behavior changes.

### Decision 3: Mount the Shared Assistant Module as a Separate App Backend

The assistant backend should appear as another backendhost module, likely `/api/apps/assistant/...`, rather than a special root-level server. This follows the backendhost route contract and existing launcher host conventions.

### Decision 4: Keep Docs, Reflection, Tools, and Prompts with the App

These are app semantics. If they move into the platform repo, the platform repo will become a disguised inventory repo.

### Decision 5: Treat HyperCard as Optional Extension, Not Core Chat

Do not make HyperCard the default shape of generic chat. Make it optional and app-provided unless a second strong consumer proves it is an OS-wide primitive.

## Alternatives Considered

### Alternative A: Leave Everything in Inventory and Copy It for New Apps

Rejected because:

- it duplicates platform code
- it makes app-chat work harder
- it keeps `wesen-os` composition app-specific

### Alternative B: Move Everything, Including Inventory Tools and Docs, Into `go-go-os-chat`

Rejected because:

- it destroys app ownership boundaries
- it turns the platform repo into an inventory-shaped monolith
- it makes future apps harder to onboard cleanly

### Alternative C: Move the TypeScript `chat-runtime` Package First

Rejected for this ticket because:

- it does not solve the current backend duplication/ownership problem
- it adds publishing/version coordination work
- it is safer after the Go-side platform API stabilizes

### Alternative D: Replace Inventory with Assistant Immediately

Rejected because:

- step 2 explicitly asks for no behavior change
- changing inventory route identity and user-facing behavior at the same time as extraction increases risk
- APP-05 still needs a later bootstrap/context layer before "chat with app" can target the assistant module cleanly

## Implementation Plan

This section is written for an intern. Follow the phases in order. Do not merge several phases into one giant refactor.

### Phase 1: Create `go-go-os-chat` with Generic Packages

Goal:

- stand up the new shared repo
- port generic backend plumbing into it
- avoid any behavior change in inventory yet

Suggested file moves/adaptations:

1. Use `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go` as the source for a generic backend component.
2. Use `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go` as the source for a generic strict request resolver.
3. Use `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go` as the source for a generic runtime composer.
4. Keep `middleware_definitions.go`, `hypercard_middleware.go`, `hypercard_events.go`, and `hypercard_extractors.go` in inventory initially.

Concrete implementation notes:

- rename inventory-specific type names such as `InventoryBackendComponent` to generic names
- remove inventory string defaults from generic packages
- inject manifest/docs/reflection/profile/middleware/tool defaults from callers
- add unit tests for the new resolver and composer before integrating inventory

### Phase 2: Port Inventory to Consume Shared Packages Without Behavior Change

Goal:

- inventory still serves from `/api/apps/inventory`
- inventory still renders the same frontend windows
- inventory still exposes the same profiles, timeline behavior, docs, and reflection

Suggested approach:

1. Replace direct use of inventory-owned generic packages with imports from `go-go-os-chat`.
2. Keep an inventory-owned wrapper module if that makes docs/reflection ownership clearer.
3. Keep inventory HyperCard code app-owned unless another app already needs it.
4. Preserve request/response wire formats and route layout.

Files you will likely touch:

- `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/reflection.go`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/docs_store.go`
- `cmd/wesen-os-launcher/main.go`

Success criteria:

- inventory chat still opens and streams normally
- profile APIs still work
- timeline APIs still work
- inventory docs/reflection still resolve
- frontend code does not need route changes

### Phase 3: Mount a Shared Assistant Module in `wesen-os`

Goal:

- introduce a reusable OS assistant backend module
- keep it additive to inventory, not a replacement yet

Implementation outline:

1. Create assistant profiles and assistant defaults.
2. Build an assistant `webchat.Server` with the shared server builder and shared runtime composer.
3. Create a backendhost module with app id `assistant`.
4. Add it to the module registry in `cmd/wesen-os-launcher/main.go`.
5. Verify it appears in `/api/os/apps`.

Recommended initial assistant capabilities:

- generic chat
- websocket streaming
- timeline
- profile selection
- confirm endpoints
- docs/reflection

Not in this phase:

- app-doc bootstrap
- module-specific context injection
- component-browser integration

## File-by-File Guidance

This section answers the question "what file should I open first?"

### Start Here

1. `cmd/wesen-os-launcher/main.go`
   - This is where current inventory chat assembly lives.
2. `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
   - This is the clearest example of generic route adapter logic.
3. `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
   - This shows the request contract for `/chat` and `/ws`.
4. `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
   - This shows the runtime assembly algorithm.

### Keep These App-Owned Unless Proven Otherwise

1. `workspace-links/go-go-app-inventory/pkg/pinoweb/middleware_definitions.go`
2. `workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_middleware.go`
3. `workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_events.go`
4. `workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_extractors.go`
5. `workspace-links/go-go-app-inventory/pkg/backendmodule/reflection.go`
6. `workspace-links/go-go-app-inventory/pkg/backendmodule/docs_store.go`

### Frontend Files That Should Not Need Major Changes in APP-04

1. `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/module.tsx`
2. `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx`
3. `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx`

If you find yourself rewriting these heavily during APP-04, stop and ask whether you are accidentally doing APP-05 work.

## Test and Validation Strategy

Validation must happen at three levels.

### 1. Unit Tests in `go-go-os-chat`

Add tests for:

- request resolver behavior for `POST /chat`
- request resolver behavior for `GET /ws`
- missing/invalid `conv_id`
- profile selection precedence
- runtime composer profile override handling
- middleware resolution failure cases

### 2. Inventory Regression Validation

After inventory is ported to the shared packages:

- open inventory chat from the launcher
- send a turn
- confirm websocket streaming still works
- confirm the event viewer and timeline windows still work
- confirm profile selection still works
- confirm inventory docs/reflection routes still exist

### 3. `wesen-os` Platform Validation

Once the assistant module is mounted:

- confirm `/api/os/apps` lists `assistant`
- confirm `/api/os/apps/assistant/reflection` works
- confirm `/api/apps/assistant/chat` works
- confirm `/api/apps/assistant/ws?conv_id=<id>` connects
- confirm no forbidden root aliases are introduced

Relevant guardrail:

- `workspace-links/go-go-os-backend/pkg/backendhost/routes.go:12-16`, `:58-66`

## Open Questions

1. Should the shared module app id be `assistant` or `os-chat`?
2. Is HyperCard becoming an OS-wide extension or remaining inventory-specific?
3. Should inventory keep an app-owned wrapper module around the generic backend module for docs/reflection clarity, or should `wesen-os` assemble inventory directly from shared components plus app-owned providers?
4. When the Go-side extraction stabilizes, should `@hypercard/chat-runtime` later move into the same repo, or remain in `go-go-os-frontend` with a clean package boundary?

## References

Key evidence files:

- `package.json`
- `pnpm-workspace.yaml`
- `go.work`
- `apps/os-launcher/src/App.tsx`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/package.json`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/components/ChatConversationWindow.tsx`
- `workspace-links/go-go-app-inventory/README.md`
- `workspace-links/go-go-app-inventory/apps/inventory/src/app/store.ts`
- `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/module.tsx`
- `workspace-links/go-go-app-inventory/apps/inventory/src/launcher/renderInventoryApp.tsx`
- `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/reflection.go`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/docs_store.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/middleware_definitions.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_middleware.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_events.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/hypercard_extractors.go`
- `workspace-links/go-go-os-backend/pkg/backendhost/module.go`
- `workspace-links/go-go-os-backend/pkg/backendhost/registry.go`
- `workspace-links/go-go-os-backend/pkg/backendhost/routes.go`
- `workspace-links/go-go-os-backend/pkg/backendhost/manifest_endpoint.go`
- `cmd/wesen-os-launcher/main.go`
