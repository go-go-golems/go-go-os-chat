---
Title: 'Intern Guide: Revised Organization for Mountable OS Chat Features and Generic Backend Modules'
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
Summary: Detailed intern-facing guide for the revised APP-04 organization, separating generic backend modules from mountable chat transport and optional chat profile features.
LastUpdated: 2026-03-06T08:41:00-05:00
WhatFor: Explain the revised APP-04 organization in which chat is packaged as a mountable feature that sits beside an app's main backend functionality instead of being baked into the generic backend module contract.
WhenToUse: Use this guide when designing go-go-os-chat package boundaries, reviewing inventory's backend-module shape, or onboarding an engineer who needs to understand which chat concerns belong in shared infrastructure versus app-owned code.
RelatedFiles:
    - Path: cmd/wesen-os-launcher/main.go
      Note: Current composition root that assembles inventory chat, profiles, and backend modules
    - Path: workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go
      Note: Current inventory route adapter that mixes transport and profile API mounting
    - Path: workspace-links/go-go-app-inventory/pkg/backendmodule/module.go
      Note: Current inventory module wrapper that combines chat routes with docs
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go
      Note: Current runtime composition algorithm and middleware-definition usage
    - Path: workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go
      Note: Current app-owned resolver for chat and websocket policy
    - Path: workspace-links/go-go-os-backend/pkg/backendhost/module.go
      Note: Generic backend module contract that should remain chat-agnostic
    - Path: workspace-links/go-go-os-backend/pkg/backendhost/routes.go
      Note: Namespaced app mount contract and legacy alias guard
    - Path: /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/server.go
      Note: Upstream statement that applications own /chat and /ws
    - Path: /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/router.go
      Note: Parsed values usage, static FS usage, and utility handler responsibilities
    - Path: /home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go
      Note: Meaning of profile API options such as middleware definitions and extension schemas
    - Path: workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/registerChatModules.ts
      Note: Frontend module registration boundary for custom chat widgets and SEM handlers
    - Path: workspace-links/go-go-os-frontend/packages/hypercard-runtime/src/hypercard/timeline/registerHypercardChatModules.ts
      Note: Example of app or feature-specific frontend chat extensions
---

# Intern Guide: Revised Organization for Mountable OS Chat Features and Generic Backend Modules

## Executive Summary

This document refines APP-04 after a closer review of the actual upstream `pinocchio` contract and the current inventory backend module shape.

The key conclusion is simple:

- `backendhost.AppBackendModule` should stay generic and chat-agnostic.
- Chat should be packaged as a feature that can be mounted under an app namespace such as `/api/apps/inventory`.
- Profile CRUD and schema routes should be an optional chat subfeature, not a required part of every backend module.
- App docs, reflection, and non-chat APIs should remain app-owned and composed beside chat, not pushed down into the chat server constructor.

In other words, the shared repository should not expose "one giant inventory-shaped backend module builder." It should expose a set of smaller chat-focused building blocks that an app module can compose in parallel with its own APIs. This is closer to how upstream `webchat` is already designed, and it matches the long-term goal of one OS chat platform that many apps can use without every app having to look like inventory.

## Reading Guide

If you are new to this system, read these sections in order:

1. `What Problem This Revision Solves`
2. `What the Current Code Actually Does`
3. `Short Answers to the Key Questions`
4. `Detailed Organization`
5. `Implementation Plan`

If you only need the short answer to the recent architecture questions, jump to `Short Answers to the Key Questions`.

## Problem Statement

The first APP-04 guide correctly identified that a lot of inventory's Go chat stack is generic in shape. After a deeper review, one refinement became necessary: the earlier framing still leaned a bit too much toward a chat-heavy backend module abstraction.

That is not quite right for the current codebase.

Upstream `pinocchio` explicitly describes a handler-first, app-owned transport model:

- the app owns `/chat` and `/ws`
- `pkg/webchat` provides the services and helper handlers behind those routes
- `Router` is useful for utility handlers such as UI and `/api/*`, but it is not the canonical owner of `/chat` and `/ws`

This is stated directly in:

- `pkg/doc/topics/webchat-framework-guide.md:21-36`
- `pkg/doc/topics/webchat-http-chat-setup.md:25-37`
- `pkg/webchat/server.go:20-21`
- `pkg/webchat/router.go:45-47`
- `pkg/webchat/router.go:216-218`

The current inventory backend module mixes several different concerns into a single options shape:

- chat transport mounting
- profile CRUD/schema endpoints
- profile write-audit metadata
- confirm routes
- app docs

You can see that mixed shape in:

- `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go:42-52`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go:27-37`

That shape makes it easy to assume that every backend module should know about profile registries, middleware definitions, extension schemas, and write actors. That assumption is wrong. Many modules will never support chat, and some chat-capable modules may support transport without offering profile editing.

The design problem, then, is not just "move code into `go-go-os-chat`." The deeper design problem is:

- how do we separate shared chat capability from the generic backend module contract
- how do we make chat mountable under an app path without making it the whole app
- how do we keep app-specific docs, reflection, tools, prompts, and UI extensions owned by the app

## What Problem This Revision Solves

This revision solves four concrete risks in the earlier organization:

1. It prevents `go-go-os-chat` from becoming an inventory-shaped mega-package.
2. It keeps `backendhost.AppBackendModule` small enough that non-chat modules remain simple.
3. It aligns the shared abstraction with upstream `webchat`, which already expects app-owned mounting.
4. It creates a clean path to the long-term goal: one shared OS chat platform served once from `wesen-os`, with multiple apps contributing context, tools, profiles, renderers, and UI add-ons.

## Short Answers to the Key Questions

This section directly answers the questions that prompted this follow-up document.

### What are `parsed *values.Values` used for?

`parsed` is not chat content. It is the Glazed/CLI/process configuration object.

Upstream `webchat.Router` decodes router/server settings out of it in `pkg/webchat/router.go:67-118` and `pkg/webchat/router.go:239-260`. Those values include:

- `addr`
- idle timeout
- eviction timing
- timeline SQLite or in-memory store settings
- turn snapshot store settings

Inventory's runtime composer also uses `parsed` to build baseline Geppetto step settings in `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go:79-88`.

So the mental model is:

- `parsed` = process configuration and engine baseline settings
- not conversation state
- not app docs
- not profile data

### What is `staticFS` used for?

`staticFS` is only for the embedded chat UI assets.

Upstream `Router.registerUIHandlers` uses it to serve:

- `/static/*`
- `/assets/*`
- `/`

See `pkg/webchat/router.go:302-339`.

If `staticFS` is `nil`, the UI handler is simply disabled. It does not affect the chat service, the websocket stream hub, or the timeline service.

So the mental model is:

- `staticFS` = optional embedded frontend bundle
- not required for backend chat transport

### What are `definitions` in `RuntimeComposer`?

In inventory's `RuntimeComposer`, `definitions` means middleware definitions. This is the typed catalog of middleware kinds that profiles can reference by name.

See:

- `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go:24-30`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go:151-213`

The composer uses those definitions to:

- validate that a named middleware exists
- resolve each middleware instance's config
- build a middleware chain from resolved instances

So `definitions` belongs to runtime composition and profile validation, not to a generic backend module contract.

### What are `ExtensionSchemas` in `ModuleOptions`?

`ExtensionSchemas` are schema documents for profile extensions exposed by the shared profile API.

This is defined upstream in `pkg/webchat/http/profile_api.go:89-103`.

The route `/api/chat/schemas/extensions` exposes those schemas through `RegisterProfileAPIHandlers(...)` in `pkg/webchat/http/profile_api.go:152-158`.

Upstream then merges three sources when listing extension schemas:

- explicit schemas
- middleware-derived extension schemas
- codec-derived extension schemas

See `pkg/webchat/http/profile_api.go:740-810`.

Important clarification:

- `ExtensionSchemas` do not mean "frontend chat window extensions"
- they mean typed backend profile-extension payloads used by the profile editor and validation flow

### What are `ProfileRegistry`, `DefaultRegistrySlug`, `WriteActor`, and `WriteSource` used for?

These all belong to the profile-management HTTP surface.

Upstream `ProfileAPIHandlerOptions` defines them in `pkg/webchat/http/profile_api.go:94-103`.

They are used for:

- looking up which registry to use by default
- validating and reading profiles
- creating/updating/deleting profiles
- recording audit metadata for writes

Concrete call sites:

- schema route mounting: `pkg/webchat/http/profile_api.go:143-158`
- create flow: `pkg/webchat/http/profile_api.go:211-283`
- patch/delete/default flows: `pkg/webchat/http/profile_api.go:352-492`
- current-profile cookie route: `pkg/webchat/http/profile_api.go:502-548`

`WriteActor` and `WriteSource` are specifically used to populate `gepprofiles.WriteOptions` in profile write operations at `pkg/webchat/http/profile_api.go:259-263`, `405-408`, and `474-477`.

So these fields should be thought of as:

- optional chat profile API settings
- not generic module metadata

### Why should `ReflectionProvider` and `DocStore` stay out of `oschatserver.BuildServer`?

Because the chat server should not be responsible for constructing the entire HTTP surface of an app.

Upstream explicitly says that:

- apps own `/chat` and `/ws`
- `Server` is a lifecycle wrapper around services and utility handlers

See `pkg/webchat/server.go:20-30` and `pkg/doc/topics/webchat-framework-guide.md:21-36`.

Inventory's current module wrapper proves that docs are already one layer above chat transport:

- the component mounts chat, profile, timeline, confirm, and UI routes in `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go:112-155`
- the module wrapper then mounts docs separately in `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go:74-81`

That is the correct direction:

- chat feature code should build and mount chat
- module wrapper code should decide whether to add docs, reflection, or other app APIs beside it

## What the Current Code Actually Does

### Upstream `webchat` already assumes app-owned transport

The cleanest evidence is the combination of the upstream docs and the code:

- `pkg/doc/topics/webchat-framework-guide.md:21-36`
- `pkg/doc/topics/webchat-http-chat-setup.md:27-37`
- `pkg/webchat/server.go:20-21`
- `pkg/webchat/router.go:45-47`

The current recommended baseline is:

1. Build `webchat.Server`.
2. Register middleware and tool factories.
3. Mount app-owned handlers:
   - `NewChatHTTPHandler(...)`
   - `NewWSHTTPHandler(...)`
   - `NewTimelineHTTPHandler(...)`
4. Optionally mount shared profile API handlers.
5. Optionally mount `srv.APIHandler()` and `srv.UIHandler()`.

That is already a feature-composition architecture.

### Inventory currently wraps that model in an inventory-shaped component

Inventory's backend component takes these fields in its `Options` struct:

- `Server`
- `RequestResolver`
- `ProfileRegistry`
- `DefaultRegistrySlug`
- `MiddlewareDefinitions`
- `ExtensionSchemas`
- `WriteActor`
- `WriteSource`
- `ConfirmMountPath`

See `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go:42-52`.

It then mounts:

- `/chat`
- `/ws`
- `/api/chat/*`
- `/api/timeline`
- `/api/`
- confirm
- `/`

See `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go:126-152`.

This is useful code, but it is not a generic app backend module abstraction. It is a composed chat feature plus a few adjacent routes.

### Inventory's module wrapper adds docs on top of that component

The wrapper in `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go:43-81` does two jobs:

1. create the inventory backend component
2. mount docs via `docmw.MountRoutes(...)`

This is exactly why docs and reflection should stay one layer above shared chat transport.

### `wesen-os` currently composes everything in `main.go`

The launcher composition root currently:

- builds inventory runtime composer
- seeds inventory profiles
- creates request resolver
- creates `webchat.Server`
- registers tools
- creates inventory backend module
- mounts the module under `/api/apps/inventory`

See `cmd/wesen-os-launcher/main.go:179-295` and `cmd/wesen-os-launcher/main.go:329-339`.

That composition root is the ideal place to benefit from the new packaging, because it is currently doing too much chat-specific assembly inline.

## Proposed Solution

The proposed solution is to make `go-go-os-chat` a shared chat capability library with small, explicit layers.

### Design Principle

Treat chat as a feature that can be mounted under an app namespace in parallel with the app's main functionality.

Do not treat chat as the definition of an app module.

### Proposed Layering

```text
backendhost.AppBackendModule
  |
  | owns app identity, lifecycle, and namespace mount
  v
app module wrapper
  |
  | composes app-specific routes + optional chat feature + optional docs/reflection
  v
chat feature
  |
  | composes transport, timeline, profile APIs, UI helpers, tool registration hooks
  v
webchat.Server + request resolver + runtime composer
```

### Layer 1: Generic backend host contract

`backendhost.AppBackendModule` should stay exactly what it already is in `workspace-links/go-go-os-backend/pkg/backendhost/module.go:19-27`:

- manifest
- mount routes
- lifecycle hooks
- health

This layer should not know about:

- profile registries
- middleware definitions
- extension schemas
- chat write-audit options

Those belong to optional features mounted inside a module.

### Layer 2: Shared chat transport feature

This is the smallest useful shared extraction.

Responsibilities:

- own the `webchat.Server`
- expose helpers to mount:
  - `/chat`
  - `/ws`
  - `/api/timeline`
  - `/api/`
  - `/`
- accept an app-owned request resolver
- accept tool registration and middleware registration hooks

Non-responsibilities:

- docs
- reflection
- app-specific non-chat APIs
- app-specific profile defaults

Suggested package shape:

```text
go-go-os-chat/
  pkg/chattransport/
    feature.go
    mount.go
    options.go
```

Suggested API sketch:

```go
type TransportFeatureOptions struct {
    Server          *webchat.Server
    RequestResolver webhttp.ConversationRequestResolver
    MountUI         bool
    MountCoreAPI    bool
    TimelineLogger  zerolog.Logger
}

type TransportFeature struct {
    server          *webchat.Server
    requestResolver webhttp.ConversationRequestResolver
    mountUI         bool
    mountCoreAPI    bool
    timelineLogger  zerolog.Logger
}

func NewTransportFeature(opts TransportFeatureOptions) (*TransportFeature, error)
func (f *TransportFeature) MountRoutes(mux *http.ServeMux) error
```

Mount behavior:

```text
/chat           -> NewChatHandler(server.ChatService(), resolver)
/ws             -> NewWSHandler(server.StreamHub(), resolver, upgrader)
/api/timeline   -> NewTimelineHandler(server.TimelineService(), logger)
/api/           -> optional server.APIHandler()
/               -> optional server.UIHandler()
```

### Layer 3: Optional shared profile API feature

The profile CRUD/schema HTTP surface is valuable, but it is optional.

Responsibilities:

- mount `/api/chat/profiles*`
- mount `/api/chat/profile` if enabled
- mount `/api/chat/schemas/middlewares`
- mount `/api/chat/schemas/extensions`

Inputs:

- `ProfileRegistry`
- `DefaultRegistrySlug`
- `MiddlewareDefinitions`
- `ExtensionCodecRegistry`
- `ExtensionSchemas`
- `WriteActor`
- `WriteSource`

Suggested package shape:

```text
go-go-os-chat/
  pkg/chatprofiles/
    feature.go
    options.go
```

Suggested API sketch:

```go
type ProfileAPIFeatureOptions struct {
    ProfileRegistry                gepprofiles.Registry
    DefaultRegistrySlug            gepprofiles.RegistrySlug
    EnableCurrentProfileCookieRoute bool
    MiddlewareDefinitions          middlewarecfg.DefinitionRegistry
    ExtensionCodecRegistry         gepprofiles.ExtensionCodecRegistry
    ExtensionSchemas               []webhttp.ExtensionSchemaDocument
    WriteActor                     string
    WriteSource                    string
}

func MountProfileAPIFeature(mux *http.ServeMux, opts ProfileAPIFeatureOptions) error
```

This package is mostly a thin, app-friendly wrapper around upstream `RegisterProfileAPIHandlers(...)`.

### Layer 4: Shared runtime composition helpers

This is where the generic algorithm from inventory's `RuntimeComposer` should move.

Responsibilities:

- start from baseline `parsed *values.Values`
- derive step settings
- apply profile `step_settings_patch`
- resolve middleware instances from middleware definitions
- build the Geppetto engine
- compute runtime fingerprint

App-owned inputs:

- default runtime key
- default prompt
- allowed tools
- default middleware uses
- middleware definitions
- middleware build deps

Suggested package shape:

```text
go-go-os-chat/
  pkg/chatruntime/
    composer.go
    options.go
    middleware_resolution.go
```

Important design rule:

- the algorithm moves
- inventory defaults do not

### Layer 5: App-owned chat package

Each app that supports chat should keep an app-owned package that provides its semantics.

For inventory, that means code like:

- inventory prompts and defaults
- inventory tool registration
- inventory profile definitions
- inventory-specific middleware definitions
- inventory-specific event sink wrappers

Suggested package shape inside inventory:

```text
go-go-app-inventory/
  pkg/chatapp/
    build.go
    profiles.go
    tools.go
    middleware.go
    events.go
```

The job of this package is not to reimplement transport. Its job is to bind inventory semantics into shared chat infrastructure.

### Layer 6: Frontend extension layer

Custom widgets and timeline renderers should remain a frontend extension concern.

Evidence:

- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/registerChatModules.ts:1-33`
- `workspace-links/go-go-os-frontend/packages/hypercard-runtime/src/hypercard/timeline/registerHypercardChatModules.ts:1-20`

This is where features such as:

- custom SEM handlers
- timeline renderers
- widget renderers
- chat-window add-ons

should plug in.

This is different from backend `ExtensionSchemas`, which are profile-extension schemas, not UI modules.

## Design Decisions

### Decision 1: Keep `backendhost.AppBackendModule` chat-agnostic

Rationale:

- it already models generic app modules correctly
- non-chat apps should not inherit chat-specific option clutter
- the host only needs identity, lifecycle, and route mounting

Evidence:

- `workspace-links/go-go-os-backend/pkg/backendhost/module.go:19-39`

### Decision 2: Package chat as a mountable feature beside app APIs

Rationale:

- upstream `webchat` is already app-owned and handler-first
- namespaced app mounting in `wesen-os` already expects module-owned submuxes
- this keeps chat parallel to the rest of the app instead of swallowing the app

Evidence:

- `pkg/webchat/server.go:20-30`
- `pkg/webchat/router.go:216-218`
- `workspace-links/go-go-os-backend/pkg/backendhost/routes.go:37-55`

### Decision 3: Make profile CRUD/schema APIs optional

Rationale:

- some chat-enabled apps may not want editable profiles
- the profile API has different data and policy concerns than transport
- the option fields are clearly profile-management-specific in upstream code

Evidence:

- `pkg/webchat/http/profile_api.go:94-103`
- `pkg/webchat/http/profile_api.go:143-158`
- `pkg/webchat/http/profile_api.go:211-492`

### Decision 4: Keep docs and reflection one layer above chat feature construction

Rationale:

- docs and reflection describe the app, not just the chat feature
- inventory already mounts docs separately after chat routes
- later "chat with app" should consume app docs, not make the chat feature own them

Evidence:

- `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go:74-81`
- `workspace-links/go-go-os-backend/pkg/backendhost/module.go:29-39`

### Decision 5: Keep HyperCard outside chat core until it is clearly OS-wide

Rationale:

- HyperCard currently looks like an optional structured-output convention
- extracting it into chat core too early would overfit the platform to inventory

If HyperCard becomes an OS-wide convention later, extract it as an optional shared extension package, not as the base chat feature.

## Detailed Organization

### Recommended repository boundaries

#### `go-go-os-chat`

This repository should own reusable Go chat infrastructure.

Suggested packages:

```text
pkg/chattransport
pkg/chatprofiles
pkg/chatruntime
pkg/chatmount
pkg/chatui
```

Suggested responsibilities:

- `chattransport`: transport feature and route mounting
- `chatprofiles`: optional profile CRUD/schema mount helpers
- `chatruntime`: generic runtime composer and middleware resolution
- `chatmount`: small helpers for composing transport + profile API + confirm
- `chatui`: optional wrappers for embedded UI/static FS handling if that grows

#### `go-go-app-inventory`

This repository should own inventory semantics.

Suggested responsibilities:

- tool registration
- inventory-specific profiles
- inventory middleware definitions
- inventory event sink wrapper
- inventory reflection/docs
- inventory-specific non-chat APIs

#### `wesen-os`

This repository should own system composition.

Suggested responsibilities:

- instantiate modules
- mount them under `/api/apps/<app-id>`
- choose required modules
- serve one shared assistant module in the long term

### Recommended app composition model

The module wrapper should compose routes in parallel.

```text
/api/apps/inventory/
  /chat
  /ws
  /api/timeline
  /api/chat/profiles*
  /api/chat/schemas/*
  /confirm
  /docs/*
  /reflection/*
  /inventory-specific-apis/*
```

The important point is that `/docs/*` and non-chat APIs are siblings of chat routes, not children of a chat subsystem that owns the entire app.

### Recommended constructor flow

The current temptation is to write something like:

```go
srv := oschatserver.BuildServer(
    parsed,
    docs,
    reflection,
    profiles,
    middlewareDefinitions,
    ...
)
```

That is the wrong long-term API because it makes the chat package responsible for the whole app.

The better flow is:

```go
chatServer := oschatruntime.BuildServer(ctx, parsed, staticFS, runtimeOpts)
transport := chattransport.NewTransportFeature(...)
profiles := chatprofiles.NewFeature(...)

module := inventorybackend.NewModule(inventorybackend.Options{
    TransportFeature: transport,
    ProfileFeature:   profiles,
    DocStore:         inventoryDocs,
    Reflection:       inventoryReflection,
    ConfirmMountPath: "/confirm",
})
```

Or, even more explicitly:

```go
func (m *InventoryModule) MountRoutes(mux *http.ServeMux) error {
    if err := m.transport.MountRoutes(mux); err != nil {
        return err
    }
    if err := m.profileAPI.MountRoutes(mux); err != nil {
        return err
    }
    m.confirm.Mount(mux, "/confirm")
    if err := docmw.MountRoutes(mux, m.docStore); err != nil {
        return err
    }
    mountInventoryHTTPAPIs(mux, m.inventoryService)
    return nil
}
```

That keeps ownership obvious.

## System Diagram

```text
                    +----------------------------------+
                    |          wesen-os main           |
                    |  composition root / module host  |
                    +----------------+-----------------+
                                     |
                                     v
                 +--------------------------------------------+
                 | inventory module (app-owned wrapper)       |
                 | manifest, lifecycle, docs, reflection      |
                 +----------------+---------------------------+
                                  |
              +-------------------+--------------------+
              |                                        |
              v                                        v
 +-----------------------------+          +-----------------------------+
 | chat feature (shared)       |          | app-specific APIs/docs      |
 | /chat /ws /api/timeline     |          | docs, reflection, confirm   |
 | optional /api/chat/*        |          | inventory service routes    |
 +--------------+--------------+          +-----------------------------+
                |
                v
 +---------------------------------------------------------------+
 | webchat.Server + request resolver + runtime composer          |
 | transport services, timeline, stream hub, UI/core API helpers |
 +-------------------------------+-------------------------------+
                                 |
                                 v
                  +----------------------------------+
                  | app-owned semantics              |
                  | profiles, tools, middleware defs |
                  | event sink wrappers              |
                  +----------------------------------+
```

## Backend Capability Model

The user's proposed list was directionally correct. The refined version below draws a firmer backend/frontend line.

### Backend capabilities of a chat-enabled app

- provide middleware definitions and middleware build dependencies
- provide profile definitions and a profile registry
- create an engine/runtime from profile selection
- provide tool registration
- optionally provide event sink wrappers that emit custom SEM/timeline entities
- optionally expose profile editor schemas and profile CRUD routes

### Frontend capabilities of a chat-enabled app

- register SEM handlers
- register custom timeline renderers and widgets
- add chat-window extensions such as starter suggestions or extra actions
- wrap `ChatConversationWindow` with app-specific window chrome or actions

### App-module capabilities that are related but separate

- docs
- reflection
- app business APIs
- app launch UX

This separation is important because it prevents backend `ExtensionSchemas` from being confused with frontend chat window extensions.

## API References

### Upstream transport and server APIs

- `webchat.NewServer(...)`: `pkg/webchat/server.go:28-42`
- `Router.ChatService()`: `pkg/webchat/router.go:220-221`
- `Router.StreamHub()`: `pkg/webchat/router.go:223-229`
- `Router.TimelineService()`: `pkg/webchat/router.go:231-237`
- `Router.APIHandler()`: `pkg/webchat/router.go:287-293`
- `Router.UIHandler()`: `pkg/webchat/router.go:295-300`

### Upstream profile API mounting

- `RegisterProfileAPIHandlers(...)`: `pkg/webchat/http/profile_api.go:137-159`
- `ProfileAPIHandlerOptions`: `pkg/webchat/http/profile_api.go:94-103`

### Inventory evidence for current composition

- runtime composer: `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go:69-149`
- request resolver: `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go:53-155`
- component route mount: `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go:112-155`
- module docs mount: `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go:74-81`
- launcher composition root: `cmd/wesen-os-launcher/main.go:179-295`

## Pseudocode for the Revised APP-04 End State

### Shared package side

```go
// go-go-os-chat/pkg/chattransport/feature.go
type Feature struct {
    server   *webchat.Server
    resolver webhttp.ConversationRequestResolver
    options  Options
}

func (f *Feature) MountRoutes(mux *http.ServeMux) error {
    mux.HandleFunc("/chat", webhttp.NewChatHandler(f.server.ChatService(), f.resolver))
    mux.HandleFunc("/chat/", webhttp.NewChatHandler(f.server.ChatService(), f.resolver))
    mux.HandleFunc("/ws", webhttp.NewWSHandler(f.server.StreamHub(), f.resolver, f.options.Upgrader))
    mux.HandleFunc("/api/timeline", webhttp.NewTimelineHandler(f.server.TimelineService(), f.options.TimelineLogger))
    mux.HandleFunc("/api/timeline/", webhttp.NewTimelineHandler(f.server.TimelineService(), f.options.TimelineLogger))
    if f.options.MountCoreAPI {
        mux.Handle("/api/", f.server.APIHandler())
    }
    if f.options.MountUI {
        mux.Handle("/", f.server.UIHandler())
    }
    return nil
}
```

### Inventory side

```go
// go-go-app-inventory/pkg/chatapp/build.go
func BuildInventoryChat(ctx context.Context, parsed *values.Values, store *inventorydb.Store) (*InventoryChat, error) {
    composer := oschatruntime.NewComposer(parsed, oschatruntime.Options{
        RuntimeKey:         "inventory",
        SystemPrompt:       "You are an inventory assistant. Be concise, accurate, and tool-first.",
        AllowedTools:       inventorytools.InventoryToolNames,
        MiddlewareDefs:     inventoryMiddlewareDefinitions(),
        DefaultMiddlewares: inventoryRuntimeMiddlewares(),
    })

    registry := inventoryProfileRegistry()
    resolver := oschatrequest.NewStrictResolver("inventory").WithProfileRegistry(registry, defaultRegistrySlug)
    server := oschatserver.New(ctx, parsed, inventoryStaticFS, composer, inventoryEventWrapper())
    registerInventoryTools(server, store)

    return &InventoryChat{
        Transport: oschattransport.NewFeature(...),
        Profiles: oschatprofiles.NewFeature(...),
    }, nil
}
```

### `wesen-os` side

```go
inventoryChat, err := inventorychat.BuildInventoryChat(ctx, parsed, inventoryStore)
if err != nil {
    return err
}

modules := []backendhost.AppBackendModule{
    inventorybackend.NewModule(inventorybackend.Options{
        ChatTransport: inventoryChat.Transport,
        ChatProfiles:  inventoryChat.Profiles,
        DocStore:      inventoryDocs,
        Reflection:    inventoryReflection,
        ConfirmMount:  confirmServer,
    }),
}
```

## Alternatives Considered

### Alternative 1: Keep chat-specific fields directly on every backend module options struct

Rejected because:

- it pollutes non-chat modules
- it confuses generic module identity with optional chat behavior
- it makes future module wrappers harder to reason about

### Alternative 2: Make a single `BuildServer(...)` constructor own chat, docs, reflection, and app APIs

Rejected because:

- it violates the upstream app-owned handler model
- it over-centralizes app composition
- it makes the shared package responsible for app semantics

### Alternative 3: Move HyperCard into chat core immediately

Rejected for now because:

- it is still not clear whether HyperCard is an inventory-specific extension or an OS-wide convention
- forcing it into core would overfit the shared abstraction

## Implementation Plan

This plan revises the APP-04 execution order without changing the ticket's overall scope.

### Phase 1: Extract shared chat transport and runtime packages into `go-go-os-chat`

Create shared packages for:

- transport mounting
- runtime composition
- request resolution
- optional profile API mounting

Do not move:

- inventory docs
- inventory reflection
- inventory prompts and profiles
- inventory tool registration
- inventory HyperCard-specific behavior unless extracted as an optional extension

Deliverables:

- new repo packages with unit tests
- no behavior change yet in `wesen-os`

### Phase 2: Refactor inventory to use shared chat feature composition

Refactor inventory's current backend component so it becomes a thin app wrapper around shared chat features plus docs/confirm/app APIs.

Success criteria:

- current inventory routes remain unchanged under `/api/apps/inventory`
- inventory profiles still work
- inventory tools still work
- inventory timeline and widgets still work

### Phase 3: Refactor `wesen-os` composition root

Replace the inline inventory chat assembly in `cmd/wesen-os-launcher/main.go:179-295` with calls into shared package constructors and app-owned chat binders.

Success criteria:

- less inline chat assembly in `main.go`
- clearer module list
- clearer ownership between platform and app

### Phase 4: Mount a shared assistant module

Still within APP-04 scope, but now easier because the abstraction is cleaner.

The assistant module should use the same shared transport/profile/runtime pieces, but with assistant-owned prompts, tools, and profile definitions.

That module should be another consumer of the shared chat platform, not a special case inside inventory.

## Concrete File-Level Guidance

### Files that should likely move or be split

- `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
  - split into shared chat feature pieces plus a thinner inventory wrapper
- `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
  - move generic algorithm to shared package
- `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
  - move generic resolver logic to shared package, keep inventory defaults in inventory binder

### Files that should likely stay app-owned

- `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
  - but simplified to compose features
- `workspace-links/go-go-app-inventory/pkg/backendmodule/reflection.go`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/docs_store.go`
- inventory tool registration packages
- inventory HyperCard middleware/event packages unless explicitly generalized

### Files that should be simplified in `wesen-os`

- `cmd/wesen-os-launcher/main.go`
  - reduce inline chat assembly
  - delegate app chat construction to shared and app-owned helper packages

## Testing and Validation Strategy

### Unit tests for shared packages

- transport feature mounts the expected routes
- profile API feature mounts nothing when registry is absent
- runtime composer builds equivalent runtime fingerprints before and after extraction
- resolver behavior matches current inventory semantics for:
  - missing `conv_id`
  - profile cookie selection
  - invalid profile slug
  - generated `conv_id`

### Integration tests

- inventory `/api/apps/inventory/chat`
- inventory `/api/apps/inventory/ws`
- inventory `/api/apps/inventory/api/timeline`
- inventory `/api/apps/inventory/api/chat/profiles`
- inventory docs still mounted

### Manual review checklist

- confirm app docs are still separate from chat feature assembly
- confirm frontend HyperCard modules still register correctly
- confirm no root-level `/chat` or `/ws` aliases were reintroduced

## Migration Risks

### Risk 1: Over-extracting inventory semantics

Mitigation:

- only move algorithms and route-mount helpers
- keep prompts, tools, profiles, docs, and HyperCard app-owned until there is clear cross-app need

### Risk 2: Regressing route ownership assumptions

Mitigation:

- anchor all transport decisions to the upstream handler-first docs
- preserve the `/api/apps/<app-id>` namespace model from `backendhost`

### Risk 3: Confusing profile extensions with frontend UI extensions

Mitigation:

- keep separate naming in docs and code
- reserve `ExtensionSchemas` for backend profile schema docs only
- use explicit terms like `ChatRuntimeModule` or `timeline renderer` for frontend add-ons

## Open Questions

- Should the long-term shared module id in `wesen-os` be `assistant` or `os-chat`?
- Which parts of HyperCard, if any, are truly OS-wide and deserve a shared extension package?
- Should confirm mounting stay in app wrappers or be one more optional chat-adjacent feature package?
- Does any app need transport-only chat without profile CRUD? If yes, prioritize that path in API design.

## References

- `cmd/wesen-os-launcher/main.go`
- `workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
- `workspace-links/go-go-app-inventory/pkg/backendmodule/module.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/runtime_composer.go`
- `workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver.go`
- `workspace-links/go-go-os-backend/pkg/backendhost/module.go`
- `workspace-links/go-go-os-backend/pkg/backendhost/routes.go`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/server.go`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/router.go`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/webchat/http/profile_api.go`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/doc/topics/webchat-framework-guide.md`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/pinocchio@v0.10.2/pkg/doc/topics/webchat-http-chat-setup.md`
- `workspace-links/go-go-os-frontend/packages/chat-runtime/src/chat/runtime/registerChatModules.ts`
- `workspace-links/go-go-os-frontend/packages/hypercard-runtime/src/hypercard/timeline/registerHypercardChatModules.ts`

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
