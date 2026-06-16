# asyou — Developer Tunnel Platform (Phase 1)

This repository contains the initial Phase 1 artifacts for the `asyou` open-source developer tunnel project.

What’s included in Phase 1:
- `docs/PHASE1.md` — Phase 1 design document
- `api/openapi.yaml` — OpenAPI v3 spec (partial)
- `migrations/0001_init.sql` — initial SQLite schema
- `server/internal/model/models.go` — Go data model structs

Next steps:
- Implement `server` HTTP handlers based on `api/openapi.yaml`
- Add migrations runner (`golang-migrate`/`goose`) and DB initialization
- Implement authentication (JWT) and API key logic

License:
- Components derived from `fatedier/frp` must remain under Apache-2.0.
- New components are intended to be MIT-licensed; update `LICENSE` files accordingly.

