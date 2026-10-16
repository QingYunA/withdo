# Android first implementation contract

The accepted plan is `m7-mobile.md`. This shared contract fixes the initial Android integration surface; implementation status and remaining M7 work must be reported separately.

- Public TypeScript types and the UI store interface are in `mobile/src/model.ts`. Use the exact camelCase JSON field names there.
- Entity types are `project`, `task`, `report`. IDs are stable UUID strings. Reports are append-only. Deletion is represented by `data: null` and a server tombstone, never an absent pull entry.
- `GET /healthz`: readiness, no credentials in its response.
- Authenticated routes require `Authorization: Bearer <configured server token>`. This is a private development/self-hosted transport; OIDC and public multiuser hosting remain separate work.
- `POST /v1/sync/push` body `{deviceId, mutations: Mutation[]}`. Response `{results: PushResult[]}`. Each mutation applies atomically; ID + payload hash is persisted for idempotency; the same ID with different content is rejected. Stale `baseVersion` returns a conflict with the current canonical record. Process independent mutations even if one conflicts.
- `GET /v1/sync/pull?cursor=<nonnegative sequence>&limit=<bounded count>&epoch=<optional known epoch>` returns `PullResponse`. Epoch mismatch or impossible cursor is an explicit resnapshot response (HTTP 409 with a JSON error code).
- `GET /v1/sync/snapshot` returns `SnapshotResponse` atomically at one cursor, including tombstones. All responses use JSON and bounded request bodies. No CORS requirement for native clients.
- The client persists its last confirmed server state separately from pending local edits, saves edits/outbox in one SQLite transaction, and replays pending intent after inbound changes. Never upload raw local integer IDs or erase pending edits during snapshot replacement.
- Mobile state is empty on first start. Demo data is allowed only in explicit automated test fixtures; do not show fake progress or seeded tasks as user data.
- First Android deliverable covers task/project/subtask editing, Mine/Agents/project/calendar views, local notifications, durable offline state and optional authenticated server sync. Desktop replication, production identity, iOS validation and store release remain distinct M7 milestones and cannot be claimed complete implicitly.
