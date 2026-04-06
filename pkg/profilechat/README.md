# `pkg/profilechat`

`pkg/profilechat` is the shared request-resolution and runtime-composition layer for OS chat style applications.

It sits between:

- HTTP/WebSocket request handling
- profile-registry based runtime selection
- middleware/runtime assembly in `pinocchio`

## What this package owns

- strict request parsing for chat and websocket entrypoints
- profile and registry selection
- profile-runtime overlay resolution
- runtime fingerprint construction
- middleware resolution for composed runtimes
- conversation-context prompt augmentation

## Main entrypoints

- `NewStrictRequestResolver(runtimeKey string)`
  - parses incoming request data into a `pinocchio` `ResolvedConversationRequest`
  - resolves profile and registry selection when a profile registry is configured
  - turns known client mistakes into typed `RequestResolutionError` values

- `NewRuntimeComposer(...)`
  - converts parsed launcher/app settings plus profile runtime data into a composed runtime
  - resolves middleware definitions and builds the final engine through `pinocchio/pkg/inference/runtime`

## Mental model

Think of the flow in two stages:

1. request resolution

```text
HTTP/WS request
  -> select registry/profile
  -> resolve runtime plan
  -> compute runtime fingerprint
  -> emit ResolvedConversationRequest
```

2. runtime composition

```text
ResolvedConversationRequest
  -> merge base settings + resolved profile settings
  -> resolve middleware instances
  -> compose engine
  -> return ComposedRuntime
```

## Error-handling contract

The resolver intentionally distinguishes between client and server failures.

- invalid request fields
  - returned as `400`
- unknown registry/profile
  - returned as typed not-found errors
- invalid `pinocchio.runtime` extension payloads
  - returned as `400 invalid pinocchio runtime extension`
- registry access failures, runtime-plan assembly failures, and other internal problems
  - preserved as ordinary errors so callers can surface them as server failures

That distinction matters because runtime-plan resolution now performs more than simple extension decoding. It can fail because of registry access, inference-setting merge issues, or runtime overlay assembly, and those failures must not be mislabeled as bad client input.

## Key files

- `request_resolver.go`
  - request parsing and profile/runtime-plan resolution
- `runtime_composer.go`
  - engine composition and middleware resolution
- `request_resolver_test.go`
  - request-policy and resolver error-shape coverage
- `runtime_composer_test.go`
  - middleware/runtime composition coverage

## Typical usage

```go
resolver := NewStrictRequestResolver("inventory").
  WithProfileRegistry(profileRegistry, defaultRegistrySlug)

resolvedReq, err := resolver.Resolve(httpReq)
if err != nil {
  return err
}

composer := NewRuntimeComposer(parsed, opts, definitions, buildDeps, defaultMiddlewares)
runtime, err := composer.Compose(ctx, infruntime.ConversationRuntimeRequest{
  ConvID:                    resolvedReq.ConvID,
  ProfileKey:                resolvedReq.RuntimeKey,
  ProfileVersion:            resolvedReq.ProfileVersion,
  ResolvedProfileRuntime:    resolvedReq.ResolvedRuntime,
  ResolvedProfileFingerprint: resolvedReq.RuntimeFingerprint,
  ResolvedInferenceSettings: resolvedReq.ResolvedInferenceSettings,
})
if err != nil {
  return err
}
```

## Review checklist

- If request resolution changes, verify client-vs-server error mapping still makes sense.
- If runtime fingerprint inputs change, verify resolver and composer stay consistent.
- If middleware resolution changes, run `runtime_composer_test.go`.
- If profile-runtime overlay resolution changes, run `request_resolver_test.go` and confirm non-validation failures are not turned into `400` errors.
