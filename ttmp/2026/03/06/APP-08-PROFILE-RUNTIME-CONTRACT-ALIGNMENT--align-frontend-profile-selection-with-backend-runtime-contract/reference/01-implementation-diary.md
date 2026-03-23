---
Title: APP-08 Implementation Diary
Ticket: APP-08-PROFILE-RUNTIME-CONTRACT-ALIGNMENT
Status: active
Topics:
    - architecture
    - backend
    - chat
    - implementation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026/03/06/APP-08-PROFILE-RUNTIME-CONTRACT-ALIGNMENT--align-frontend-profile-selection-with-backend-runtime-contract/design-doc/01-profile-registry-and-runtime-key-contract-alignment-plan.md
      Note: Main ticket design note for the selector contract cutover
    - Path: cmd/wesen-os-launcher/main_integration_test.go
      Note: End-to-end launcher integration coverage for the cutover
ExternalSources: []
Summary: Chronological implementation diary for APP-08, covering the request-selector cutover to `profile`/`registry`, the resolved-runtime debug payload rename, repo-by-repo commits, and verification results.
LastUpdated: 2026-03-06T20:02:00-05:00
WhatFor: Use this diary to understand exactly how APP-08 was implemented, why the contract became a hard cutover instead of a compatibility migration, and how to verify the result.
WhenToUse: Use when reviewing APP-08 commits, replaying the implementation, or auditing the request/response selector contract across frontend and backend repos.
---

# APP-08 Implementation Diary

## 2026-03-06

This diary records the first execution slice for APP-08: align requested profile selection naming around canonical `profile` and `registry`, while preserving the existing notion that resolved runtime identity is still reported separately as `runtime_key` or `current_runtime_key`.

### Scope chosen for the first slice

I did not attempt the full naming cleanup described in the original ticket document in one pass.

Instead, I executed the lowest-risk slice that was already blocking correctness:

- make the frontend selection object consistently capable of carrying both `profile` and `registry`
- make request transport actually send `registry` again on HTTP and WebSocket paths
- teach the shared Go resolver to understand canonical `registry`
- preserve compatibility with legacy `registry_slug`
- preserve the existing lenient behavior where malformed or unknown registry selectors fall back to the default registry instead of failing the request

This keeps the requested-selection contract moving in the right direction without changing the separate resolved-runtime reporting contract.

### Initial audit findings

The code had already drifted from the assumptions written in the original APP-08 design note.

Observed state:

- frontend chat transport already uses `profile`, not `runtime_key`, for requested profile selection
- frontend `ChatProfileSelection` had regressed to only `{ profile?: string }`
- frontend profile APIs still support `registry` query parameters for profile CRUD/list routes
- chat submit/connect paths no longer propagated `registry`
- shared Go request resolver in `go-go-os-chat` only parsed `profile`, not `registry`
- current tests in the inventory app and shared resolver still referenced legacy `registry_slug` in a few places
- launcher integration tests still validated resolved runtime identity with `runtime_key` or `current_runtime_key`, which is correct and should remain distinct from requested profile selection

### Files changed

#### Shared resolver repo: `go-go-os-chat`

- `pkg/profilechat/request_resolver.go`
- `pkg/profilechat/request_resolver_test.go`

Changes:

- added request body support for both `registry` and legacy `registry_slug`
- added registry selection parsing from HTTP body and WS query string
- threaded `RegistrySlug` into `gepprofiles.ResolveInput`
- added compatibility fallback:
  - malformed registry selector -> ignore and use default registry
  - syntactically valid but missing registry -> retry with default registry
- updated tests so canonical `registry` is primary and `registry_slug` remains covered as a compatibility alias

Commit:

- `ac665ef` `Normalize registry selection in profile resolver`

#### Frontend repo: `go-go-os-frontend`

- `packages/chat-runtime/src/chat/runtime/profileTypes.ts`
- `packages/chat-runtime/src/chat/state/profileSlice.ts`
- `packages/chat-runtime/src/chat/state/selectors.ts`
- `packages/chat-runtime/src/chat/runtime/http.ts`
- `packages/chat-runtime/src/chat/ws/wsManager.ts`
- `packages/chat-runtime/src/chat/runtime/useConversation.ts`
- `packages/chat-runtime/src/chat/runtime/useProfiles.ts`
- `packages/chat-runtime/src/chat/runtime/useSetProfile.ts`
- associated tests in the same package

Changes:

- expanded `ChatProfileSelection` to `{ profile?: string; registry?: string }`
- taught Redux profile state to persist selected registry globally and per scope
- updated selectors to return both profile and registry
- updated HTTP prompt submission to include `registry` in the JSON payload
- updated WebSocket URL construction to include `registry` in the query string
- updated the selection-key logic so reconnects account for registry changes as well as profile changes
- updated tests to assert the new shape

Commit:

- `9aeae13` `Carry registry through chat profile selection`

#### Inventory app repo: `go-go-app-inventory`

- `apps/inventory/src/launcher/renderInventoryApp.tsx`
- `pkg/pinoweb/request_resolver_test.go`

Changes:

- updated launcher-local root state typing so scoped selection can carry registry if needed
- updated mirrored resolver tests to make canonical `registry` primary and keep legacy alias coverage

Commit:

- `cbd932c` `Update inventory tests for registry selection contract`

#### Launcher repo: `wesen-os`

- `cmd/wesen-os-launcher/main_integration_test.go`
- workspace link pointers for `go-go-os-frontend`
- workspace link pointers for `go-go-app-inventory`

Changes:

- updated the launcher integration test to use canonical `registry`
- added a separate compatibility test to ensure legacy `registry_slug` still works during migration
- advanced workspace-link SHAs after the frontend and inventory repo commits

Commits:

- `f6a30a7` `Cover canonical registry selector in launcher integration tests`
- `ab5a29c` `Advance workspace links for profile registry contract updates`

### Verification commands

I ran the following checks from the `wesen-os` workspace:

```bash
pnpm --dir workspace-links/go-go-os-frontend/packages/chat-runtime exec tsc -b --pretty false
gofmt -w /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/request_resolver.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/request_resolver_test.go cmd/wesen-os-launcher/main_integration_test.go workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver_test.go
go test /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat ./workspace-links/go-go-app-inventory/pkg/pinoweb ./cmd/wesen-os-launcher
```

Final result:

- TypeScript build passed for the chat runtime package
- targeted Go tests passed for:
  - `github.com/go-go-golems/go-go-os-chat/pkg/profilechat`
  - `github.com/go-go-golems/go-go-app-inventory/pkg/pinoweb`
  - `github.com/go-go-golems/wesen-os/cmd/wesen-os-launcher`

### Failure encountered during implementation

The first resolver patch introduced a regression against the existing compatibility contract.

Symptom:

- tests expecting unknown registry selectors to be ignored started failing with `registry not found`

Why it happened:

- canonical `registry` parsing succeeded for a syntactically valid but nonexistent registry like `missing`
- the resolver then passed that registry through to `ResolveEffectiveProfile`
- the profile registry returned `ErrRegistryNotFound`

Fix:

- kept the new canonical `registry` parsing
- added a fallback in `resolveEffectiveProfile(...)`
- if `ErrRegistryNotFound` is returned for a non-default requested registry, retry once with `defaultRegistrySlug`

This restored the old lenient behavior while preserving the new canonical selector path.

### Current state after this slice

What is now true:

- requested profile selection can carry `profile` and `registry` from frontend state to backend resolver
- canonical `registry` works on both HTTP and WS paths
- legacy `registry_slug` still works as a compatibility alias
- resolved runtime identity remains separate and unchanged

What is still not done:

- no cleanup yet for the overloaded meaning of `runtime_key` in some backend/debug payloads
- no current-profile or current-runtime payload normalization work yet
- no response schema or docs cleanup beyond this diary/changelog/tasks update
- no broader decision yet on whether the canonical response vocabulary should become `requested_profile` / `requested_registry` vs `runtime_key` / `current_runtime_key`

### Recommended next slice

The next APP-08 slice should be response-contract cleanup, not more request-path work.

Recommended order:

1. audit all response/debug payloads that emit `runtime_key`, `current_runtime_key`, and any lingering `registry_slug`
2. define a strict semantic split between:
   - requested selector fields
   - resolved/runtime-identity fields
3. add explicit compatibility adapters for legacy response names if needed
4. update the main APP-08 design doc once the response vocabulary decision is final

## 2026-03-06: Follow-up frontend slice

After the first slice, I audited the remaining request-path behavior and found a real functional gap:

- `/api/chat/profiles` already returns `registry`
- frontend `listProfiles(...)` decoding was dropping that field
- the profile picker only tracked a slug string
- this meant the new `{ profile, registry }` transport support was not enough to select cross-registry profiles from the UI

### Follow-up scope

I kept the second slice deliberately small and frontend-only:

- preserve `registry` on decoded `ChatProfileListItem`
- keep registry when refreshing profile lists
- make the profile dropdown round-trip `{ profile, registry }`
- keep this change limited to the dropdown path
- defer inventory menu command disambiguation for a later slice if multiple registries expose the same profile slug there

### Files changed in `go-go-os-frontend`

- `packages/chat-runtime/src/chat/runtime/profileTypes.ts`
- `packages/chat-runtime/src/chat/runtime/profileApi.ts`
- `packages/chat-runtime/src/chat/runtime/useProfiles.ts`
- `packages/chat-runtime/src/chat/runtime/useSetProfile.ts`
- `packages/chat-runtime/src/chat/components/profileSelectorState.ts`
- `packages/chat-runtime/src/chat/components/ChatProfileSelector.tsx`
- related tests:
  - `packages/chat-runtime/src/chat/components/profileSelectorState.test.ts`
  - `packages/chat-runtime/src/chat/runtime/useProfiles.test.ts`
  - `packages/chat-runtime/src/chat/runtime/profileApi.test.ts`

### What changed

- `ChatProfileListItem` now carries optional `registry`
- `profileApi` list decoding now keeps `registry` from `/api/chat/profiles`
- refresh reconciliation now matches selected profiles using both slug and registry when available
- picker option values now encode both profile and registry instead of only the slug
- `useSetProfile(...)` now accepts either a plain slug or an explicit `{ profile, registry }` object
- picker labels now include `[registry]` when available so the user can see cross-registry context

### Verification

I reran the chat runtime TypeScript build:

```bash
pnpm --dir workspace-links/go-go-os-frontend/packages/chat-runtime exec tsc -b --pretty false
```

Result:

- passed

### Commits for this slice

- frontend repo: `fbd78e9` `Preserve registry in profile picker selections`
- `wesen-os` workspace-link update: `0247374` `Advance frontend workspace link for registry-aware profile picker`

### Remaining gap after this follow-up

The inventory app’s menu-command profile switching path still keys selections by slug only.

That is acceptable as a short-term checkpoint because:

- the main transport/state path now supports registry-aware selection
- the dropdown picker can now preserve registry correctly
- the menu-command path can be addressed in a separate focused slice if duplicate slugs across registries become a real use case

## 2026-03-06: Breaking cutover slice

After the earlier compatibility-oriented slices, I revisited the contract decision with the explicit instruction that legacy compatibility was not needed. That changed the implementation goal materially.

The rest of APP-08 was executed as a hard cutover, not a migration shim.

### Cutover decision

Final requested-selector contract:

- request body/query/WS selectors accept only:
  - `profile`
  - `registry`

Final resolved-runtime reporting in JSON debug payloads:

- use `resolved_runtime_key`

Explicitly rejected legacy selector names:

- `runtime_key`
- `registry_slug`

Those names still exist in other parts of the system where they refer to internal runtime identity, persistence, or SEM protocol fields. APP-08 only removes them as public request selectors and renames the JSON debug payloads that were still ambiguous.

### Files changed for the hard cutover

#### Shared resolver repo: `go-go-os-chat`

- `pkg/profilechat/request_resolver.go`
- `pkg/profilechat/request_resolver_test.go`

What changed:

- request parsing now treats only `profile` and `registry` as valid selectors
- body/query `runtime_key` and `registry_slug` now fail fast with `400 unsupported legacy selector`
- invalid registry slugs now return `400` instead of being ignored
- unknown but syntactically valid registries now return `404` instead of falling back to default
- fallback retry behavior for unknown registries was removed

Commit:

- `ec3ea06` `Cut over profile resolver to profile and registry selectors`

#### Inventory app repo: `go-go-app-inventory`

- `pkg/pinoweb/request_resolver_test.go`

What changed:

- mirrored resolver tests were updated to the hard-cutover behavior
- legacy `runtime_key` / `registry_slug` coverage now asserts rejection rather than compatibility fallback

Commit:

- `f0b2f70` `Align inventory resolver tests with hard selector cutover`

#### Launcher repo: `wesen-os`

- `cmd/wesen-os-launcher/main_integration_test.go`

What changed:

- integration tests now assert the hard-cutover status codes:
  - unknown registry -> `404`
  - legacy selector names -> `400`
- invalid current-profile payload tests now use the canonical `{ profile, registry }` shape
- debug-route assertions now look for `resolved_runtime_key`

Commit:

- `d4044cc` `Finalize APP-08 selector cutover in launcher tests`

#### Web chat backend repo: `pinocchio`

- `pkg/webchat/http/api.go`
- `pkg/webchat/http/profile_api.go`
- `pkg/webchat/router_debug_routes.go`
- `pkg/webchat/router_debug_api_test.go`
- `cmd/web-chat/profile_policy.go`
- `cmd/web-chat/profile_policy_test.go`
- `cmd/web-chat/app_owned_chat_integration_test.go`

What changed:

- shared request body types now use canonical `profile` / `registry`
- old request fields remain only as explicit rejection points for clearer error messages
- app-owned resolver logic now rejects `runtime_key` and `registry_slug`
- `/api/chat/profile` now reads and writes `{ profile, registry }`
- the current-profile cookie now stores `registry/profile`
- debug JSON payloads were renamed from `current_runtime_key` / per-item `runtime_key` to `resolved_runtime_key`
- tests were updated for the new request and response shapes

Commit:

- `5452ac8` `Remove legacy profile selector aliases from web chat`

### Post-cutover audit finding

After the Go-side cutover passed, I did a broader string audit for `current_runtime_key` and found one remaining frontend consumer:

- the web debug UI adapter in `pinocchio/cmd/web-chat/web`

The backend had already switched to `resolved_runtime_key`, but the browser-side debug API adapter and MSW mocks still expected `current_runtime_key`. That would have left the debug UI stale even though backend tests were green.

Files changed:

- `cmd/web-chat/web/src/debug-ui/api/debugApi.ts`
- `cmd/web-chat/web/src/debug-ui/api/debugApi.test.ts`
- `cmd/web-chat/web/src/debug-ui/mocks/msw/createDebugHandlers.ts`

What changed:

- debug UI transport types now consume `resolved_runtime_key`
- the adapter mapping uses `resolved_runtime_key` when producing `ConversationSummary`
- the test fixture and MSW mock payloads were updated to the renamed field

Commit:

- `e93c449` `Align debug UI with resolved runtime key payloads`

### Verification for the cutover slice

I reran both targeted and package-local verification.

From the `wesen-os` workspace:

```bash
pnpm --dir workspace-links/go-go-os-frontend/packages/chat-runtime exec tsc -b --pretty false
gofmt -w /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/request_resolver.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat/request_resolver_test.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/pkg/webchat/http/api.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/pkg/webchat/http/profile_api.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/pkg/webchat/router_debug_routes.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/pkg/webchat/router_debug_api_test.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/cmd/web-chat/profile_policy.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/cmd/web-chat/profile_policy_test.go /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/cmd/web-chat/app_owned_chat_integration_test.go cmd/wesen-os-launcher/main_integration_test.go workspace-links/go-go-app-inventory/pkg/pinoweb/request_resolver_test.go
go test /home/manuel/workspaces/2026-03-02/os-openai-app-server/go-go-os-chat/pkg/profilechat ./workspace-links/go-go-app-inventory/pkg/pinoweb ./cmd/wesen-os-launcher /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/pkg/webchat /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/cmd/web-chat
```

Package-local debug UI verification:

```bash
npm --prefix /home/manuel/workspaces/2026-03-02/os-openai-app-server/pinocchio/cmd/web-chat/web run typecheck
npm exec vitest run src/debug-ui/api/debugApi.test.ts
```

Additional verification happened automatically during the `pinocchio` pre-commit hook for `5452ac8`:

- `go test ./...`
- web frontend `check` step
- `go build ./...`
- `golangci-lint run`
- `go vet`

Observed result:

- all targeted Go tests passed
- chat-runtime TypeScript build passed
- debug UI typecheck passed
- targeted vitest passed
- `pinocchio` pre-commit checks passed

### Final APP-08 state

What is complete now:

- requested selector vocabulary is consistently `profile` + `registry`
- legacy request selector names are rejected rather than normalized
- current-profile API payloads use `profile` + `registry`
- debug JSON payloads use `resolved_runtime_key`
- frontend transport and picker flows preserve `registry`
- the browser debug UI matches the backend rename

What APP-08 intentionally did not change:

- SEM/WebSocket protocol field names that already use `runtime_key` for resolved runtime identity
- turn persistence columns and timeline/event metadata that store resolved runtime identity
- unrelated profile CRUD payloads that already correctly expose `slug` as profile-document identity

At this point APP-08 is complete as a contract-cutover ticket rather than an open migration.
