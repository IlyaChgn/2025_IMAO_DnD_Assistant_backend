# vibecode_docs — Documentation Index

> This folder contains technical documentation for features, API specifications, rollout plans, and implementation logs created during development.

---

## Quick Navigation

| I need to... | Go to |
|--------------|-------|
| Understand documentation standards | [DOCS_STANDARDS.md](DOCS_STANDARDS.md) |
| Understand Creature Model for automation | [creature-model-automation/](#creature-model-automation) |
| Design/implement Item System | [item-system/](#item-system) |
| Implement Maps API | [maps-api/](#maps-api) |
| Understand User Model refactor | [user-model-refactor-and-oauth/](#user-model-refactor-and-oauth) |
| Integrate frontend with auth | [auth-frontend-integration-guide.md](user-model-refactor-and-oauth/auth-frontend-integration-guide.md) |
| See API field changes for frontend | [frontend-api-field-changes-backlog.md](user-model-refactor-and-oauth/frontend-api-field-changes-backlog.md) |

---

## Directory Structure

```
vibecode_docs/
├── DOCS_STANDARDS.md           # Documentation standards and templates
├── README.md                   # This file (table of contents)
│
├── creature-model-automation/  # Creature model evolution for automation
│   ├── creature-model-evolution.md
│   └── migration-rules.md
│
├── item-system/                # Item System feature documentation
│   └── item-system-design-plan.md
│
├── maps-api/                   # Maps API feature documentation
│   ├── maps-api-spec.yaml
│   ├── maps-api-backend-implementation-plan.md
│   └── maps-api-implementation-notes.md
│
└── user-model-refactor-and-oauth/   # User model refactor + OAuth
    ├── user-model-investigation-and-strategy-report.md
    ├── user-model-rollout-plan.md
    ├── user-entity-refactor-log.md
    ├── frontend-api-field-changes-backlog.md
    ├── auth-frontend-integration-guide.md
    └── migrations-application-investigation.md
```

---

## Document Index

### creature-model-automation/

Documentation for evolving the Creature model to support combat automation — structured actions, runtime state, spellcasting.

| Document | Type | Status | Description |
|----------|------|--------|-------------|
| [creature-model-evolution.md](creature-model-automation/creature-model-evolution.md) | Investigation Report | Implemented | Complete evolution guide: Level 1 (Vision/Movement), Level 2 (StructuredActions), Level 3 (RuntimeState), Level 4 (Spellcasting). Includes before/after schemas, examples, migration strategy. |
| [migration-rules.md](creature-model-automation/migration-rules.md) | Implementation Notes | Draft | Detailed migration rules for converting legacy Speed→Movement, Senses→Vision, llm_parsed_attack→StructuredActions. Includes mapping tables, edge cases, validation checklist. |

**Related branch:** `feature/maps-api-align-spec`

---

### item-system/

Documentation for the Item System feature — inventory management, equipment, modifiers, and consumables.

| Document | Type | Status | Description |
|----------|------|--------|-------------|
| [item-system-design-plan.md](item-system/item-system-design-plan.md) | Feature Plan | Draft | Comprehensive design plan: data models (ItemDefinition/ItemInstance), modifiers system, API contracts, content pipeline, 3-iteration roadmap, backlog |

**Related branch:** TBD (feature/item-system)

---

### maps-api/

Documentation for the Maps API feature — CRUD operations for map storage with JSONB data.

| Document | Type | Status | Description |
|----------|------|--------|-------------|
| [maps-api-spec.yaml](maps-api/maps-api-spec.yaml) | API Specification | Implemented | OpenAPI 3.0.3 specification for Maps API endpoints (create, get, update, delete, list) |
| [maps-api-backend-implementation-plan.md](maps-api/maps-api-backend-implementation-plan.md) | Feature Plan | Implemented | Comprehensive plan with goals, target model, PR breakdown (PR-1 through PR-6), acceptance criteria |
| [maps-api-implementation-notes.md](maps-api/maps-api-implementation-notes.md) | Implementation Notes | Implemented | Technical guidance: PostgreSQL schema with JSONB, validation rules, error codes, SQL examples |

**Related branch:** `feature/maps-api-align-spec`

---

### user-model-refactor-and-oauth/

Documentation for the User entity refactor and multi-provider OAuth implementation (VK, Google, Yandex).

| Document | Type | Status | Description |
|----------|------|--------|-------------|
| [user-model-investigation-and-strategy-report.md](user-model-refactor-and-oauth/user-model-investigation-and-strategy-report.md) | Investigation Report | Implemented | Analysis of VK-only limitations, multi-provider blockers, proposed `user_identity` table pattern |
| [user-model-rollout-plan.md](user-model-refactor-and-oauth/user-model-rollout-plan.md) | Rollout Plan | Implemented | 10-PR deployment plan with target data model, API changes, testing strategy, rollback procedures |
| [user-entity-refactor-log.md](user-model-refactor-and-oauth/user-entity-refactor-log.md) | Implementation Log | Implemented | PR-by-PR changelog (PR1+2 through PR9) with commits, file changes, verification steps |
| [frontend-api-field-changes-backlog.md](user-model-refactor-and-oauth/frontend-api-field-changes-backlog.md) | API Change Backlog | Implemented | Breaking changes tracker: `name`→`displayName`, `avatar`→`avatarUrl`, route changes, new endpoints |
| [auth-frontend-integration-guide.md](user-model-refactor-and-oauth/auth-frontend-integration-guide.md) | Integration Guide | Implemented | Comprehensive frontend integration guide: OAuth flows (VK/Google/Yandex), session handling, error model, curl examples |
| [migrations-application-investigation.md](user-model-refactor-and-oauth/migrations-application-investigation.md) | Technical Investigation | Implemented | Deep dive into `golang-migrate` behavior: version tracking, dirty state recovery, deploy procedures |

**Related branch:** `main` (merged via PR #32)

---

## Document Types Reference

| Type | Purpose |
|------|---------|
| **Feature Plan** | Goals, scope, PR breakdown, acceptance criteria |
| **Implementation Notes** | DB schemas, validation rules, error codes |
| **API Specification** | OpenAPI/YAML formal contracts |
| **Implementation Log** | PR-by-PR changelog with commits |
| **Investigation Report** | Analysis of current state, proposed solutions |
| **Rollout Plan** | Deployment steps, risks, testing strategy |
| **API Change Backlog** | Breaking API changes for frontend coordination |
| **Integration Guide** | Technical reference for system integration |
| **Technical Investigation** | Deep analysis of specific systems/tools |

See [DOCS_STANDARDS.md](DOCS_STANDARDS.md) for templates and metadata requirements.

---

## Status Legend

| Status | Meaning |
|--------|---------|
| **Draft** | Work in progress |
| **In Review** | Awaiting feedback |
| **Approved** | Ready for implementation |
| **Implemented** | Code merged, matches codebase |
| **Deprecated** | Obsolete, kept for history |

---

## Notes

- All documents in this folder are **internal technical documentation** — not user-facing
- Documents marked as "Implemented" were verified against the codebase at the time of completion
- For the latest API contracts, always cross-check with the actual code in `internal/` and `db/migrations/`

---

## Changelog

| Date | Change |
|------|--------|
| 2026-02-05 | Initial README and DOCS_STANDARDS created |
| 2026-02-05 | Fixed maps-api-backend-implementation-plan.md link/filename mismatch |
| 2026-02-05 | Normalized all file/folder names to kebab-case |
| 2026-02-06 | Added creature-model-automation section with creature-model-evolution.md |
