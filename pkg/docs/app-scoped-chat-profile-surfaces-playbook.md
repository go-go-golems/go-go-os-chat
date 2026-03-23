# App-Scoped Chat Profile Surfaces Playbook

This playbook explains how to give one chat application its own visible profile surface while still allowing shared generic profiles to exist underneath as fallback material.

The pattern is intended for applications that use:

- `go-go-os-chat/pkg/chatservice`
- `go-go-os-chat/pkg/profilechat`
- `pinocchio/pkg/webchat/http` profile APIs

## Why This Exists

The shared `go-go-os-chat` layer mounts generic endpoints:

- `/chat`
- `/ws`
- `/api/chat/profiles`
- `/api/chat/profiles/:slug`

Those endpoints only know about the `geppetto/pkg/engineprofiles.Registry` injected into them. If multiple apps reuse the same raw registry, they will expose the same visible profile list and accept the same explicit profile selections.

That is usually wrong for product behavior.

Examples:

- inventory wants `default`, `inventory`, `analyst`, and maybe `assistant`
- assistant may want only `assistant`
- a future document chat may want document-specific profiles and middleware

So the generic layer should stay generic, but each app should inject its own registry surface.

## Core Rule

An app-specific chat surface should own all of these together:

- its visible registry slug
- its default profile slug
- its allowlisted visible profile slugs
- its built-in profile YAML
- its injected `Registry` for both `/chat` and `/api/chat/profiles`

Do not make `/api/chat/profiles` app-specific while leaving `/chat` bound to a broader shared registry. They must agree.

## The Important Distinction

There are two different things:

1. The visible app profile surface
2. The generic fallback registry pool

The visible app profile surface is what the selector sees and what explicit `/chat` selection is allowed to choose from.

The generic fallback pool is where shared profiles may still live for:

- stack inheritance
- shared engine defaults
- other internal composition needs

These are not the same thing.

## What “Stack On Top Of The General Registry” Actually Means

Saying “the app profiles stack on top of the general registry” can mean two different things:

1. The app-visible root profiles come from the app surface, and those root profiles may reference shared generic profiles in their `stack`.
2. The user can directly select any profile from the shared generic registry.

Only the first meaning is desired.

If an app-owned profile YAML includes stack refs like:

```yaml
stack:
  - registry_slug: shared
    profile_slug: default
```

then the app profile can inherit from the shared registry, as long as the injected aggregate store contains both registries.

But if the app surface allowlist does not include the shared profile slug, the user still cannot directly select that shared profile at `/chat` or see it in `/api/chat/profiles`.

That is the intended architecture.

## Recommended Composition Model

For each app:

```text
app built-in registry
  + shared fallback registries
  = app-specific aggregate registry surface
```

Then expose only the app built-in registry as the visible registry.

Pseudocode:

```go
type AppProfileSurfaceConfig struct {
    AppID              string
    VisibleRegistry    *engineprofiles.EngineProfileRegistry
    DefaultProfileSlug engineprofiles.EngineProfileSlug
    VisibleProfiles    []engineprofiles.EngineProfileSlug
    FallbackRegistry   engineprofiles.Registry
}
```

Behavior:

- copy fallback registries into an aggregate store
- upsert the app visible registry into that same store
- mark the app visible registry as the default registry for this app surface
- reject explicit registry slugs other than the app visible registry
- reject explicit profile slugs outside the app visible allowlist
- resolve the selected app-visible root profile against the aggregate store

This lets stack expansion see both:

- the app-visible built-ins
- the shared fallback registries

while keeping the user-facing surface curated.

## Injection Points

The app-specific registry surface must be injected into both:

- `profilechat.NewStrictRequestResolver(...).WithProfileRegistry(...)`
- `chatservice.ProfileAPIOptions{ Registry: ..., DefaultRegistrySlug: ... }`

If you inject the app surface only into one of those, the system drifts.

## Built-In Profile Source Rules

Each app should ship built-in profile YAML in its own package.

Recommended shape:

- `pkg/<app>/builtin_profiles.go`
- `pkg/<app>/profiles/profiles.yaml`

Load with:

```go
registry, err := gepprofiles.DecodeEngineProfileYAMLSingleRegistry(data)
```

Why:

- keeps app-owned runtime policy in real profile documents
- avoids hiding shipped profiles in launcher testdata
- makes the profile surface explicit and reviewable

## Validation Checklist

For every app using this pattern, add tests that prove:

1. `/api/chat/profiles` returns only the intended visible set
2. `/chat` accepts visible profiles
3. `/chat` rejects hidden or foreign profiles
4. the app default profile is correct when no explicit selection is sent
5. another app’s profile list does not leak into this app

## Current Known Adopters

At the time of writing, the backend chat surfaces in `wesen-os` that use this pattern are:

- inventory
- assistant

Those are mounted from:

- `wesen-os/workspace-links/go-go-app-inventory/pkg/backendcomponent/component.go`
- `wesen-os/pkg/assistantbackendmodule/module.go`

Other modules like `sqlite`, `gepa`, and `arc-agi` do not currently mount the shared `go-go-os-chat` chat/profile surface.

There are other frontend consumers of `@hypercard/chat-runtime`, but they are not separate backend chat services yet. If they grow real backend `/chat` and `/api/chat/profiles` endpoints, they should follow this playbook.

## When To Use This Playbook

Use this whenever a new app needs any of the following:

- its own profile selector contents
- app-owned middleware/tool policy
- app-owned default profile behavior
- separation from other chat apps that share the same server process
