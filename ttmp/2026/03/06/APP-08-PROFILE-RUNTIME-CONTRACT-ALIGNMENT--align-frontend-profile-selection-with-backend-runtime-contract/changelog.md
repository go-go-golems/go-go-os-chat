# Changelog

## 2026-03-06

- Initial workspace created
- Added focused follow-up scope for the frontend/backend profile selection contract mismatch discovered while tracing runtime selection behavior. This ticket is explicitly positioned as a follow-on to APP-04 shared OS chat platform work rather than part of the VM/runtime-DSL redesign track.
- Executed the first implementation slice:
  - frontend `ChatProfileSelection` now carries both `profile` and `registry`
  - HTTP and WS chat transport now propagate canonical `registry`
  - shared Go request resolver now accepts canonical `registry` and still supports legacy `registry_slug`
  - malformed or unknown registry selectors still fall back to the default registry for compatibility
  - targeted TypeScript build and Go tests passed
- Executed a second frontend-only slice:
  - profile list decoding now preserves `registry`
  - the profile dropdown now round-trips `{ profile, registry }` instead of dropping registry on selection
  - chat-runtime TypeScript build passed again
- Executed the breaking contract cutover:
  - request selectors now accept only `profile` and `registry`
  - legacy request selectors `runtime_key` and `registry_slug` now return `400`
  - invalid registry slugs now return `400`
  - unknown registries now return `404` instead of silently falling back
  - `/api/chat/profile` now uses `{ profile, registry }`
  - debug JSON payloads now use `resolved_runtime_key`
  - targeted Go tests passed across `go-go-os-chat`, `go-go-app-inventory`, `pinocchio`, and `wesen-os`
- Closed the final frontend consumer gap after audit:
  - the web debug UI adapter and mocks now consume `resolved_runtime_key`
  - frontend debug UI typecheck and targeted vitest passed
