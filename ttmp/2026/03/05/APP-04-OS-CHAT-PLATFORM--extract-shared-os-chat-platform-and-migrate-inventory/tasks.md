# Tasks

## Done

- [x] Create APP-04 ticket workspace and primary docs
- [x] Investigate current inventory, wesen-os, chat-runtime, and backendhost architecture
- [x] Write intern guide for shared OS chat platform extraction, inventory migration, and assistant-module mount
- [x] Write revised organization guide that separates mountable chat features from generic backend modules
- [x] Write tight chat core guide that removes extension schemas and rich profile-admin concerns from the base design
- [x] Write investigation diary and relate key files
- [x] Validate APP-04 ticket with docmgr doctor
- [x] Upload APP-04 bundle to reMarkable
- [x] Upload refreshed APP-04 bundle with the revised organization guide to reMarkable
- [x] Upload refreshed APP-04 bundle with the tight chat core guide to reMarkable
- [x] Implement go-go-os-chat shared backend packages
- [x] Port inventory to shared packages with no behavior change
- [x] Mount shared assistant module in wesen-os
- [x] Validate implemented APP-04 code and upload the refreshed implementation bundle to reMarkable

## TODO

- [ ] Publish/version `go-go-os-chat` so consuming repos can replace the temporary local-workspace `v0.0.0` dependency
- [ ] Decide whether inventory should keep compatibility wrappers in `pkg/pinoweb` or switch all callers to direct shared-package imports
- [ ] Build APP-05 on top of the new shared assistant/app-chat backend shape
