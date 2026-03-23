# Tasks

## TODO

- [x] Confirm exact APP-04 ownership boundary for chat profile/runtime contract normalization
- [x] Audit all frontend request and websocket selectors that currently send `profile` and `registry`
- [x] Audit all backend request resolver, current-profile, and current-runtime payload fields that still use `runtime_key` and `registry_slug`
- [x] Decide canonical vocabulary for requested profile selection versus resolved runtime identity
- [x] Define compatibility policy for old and new field names during migration
- [x] Add end-to-end tests covering HTTP, WS, current-profile, and versioned runtime reporting
- [x] Update frontend chat-runtime transport and selectors to use the canonical contract
- [x] Update backend resolver/response adapters to normalize and document the contract
- [x] Refresh APP-04 and related docs once the final naming decision is implemented

## In Progress

- [x] Normalize backend response payload vocabulary for requested profile selection versus resolved runtime identity
