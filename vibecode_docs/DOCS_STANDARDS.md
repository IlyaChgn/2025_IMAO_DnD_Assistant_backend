# Documentation Standards

> This document defines naming conventions, document types, metadata requirements, and lifecycle statuses for the `vibecode_docs/` folder.

---

## Document Type Catalog

| Type | Purpose | Typical Audience | Filename Pattern |
|------|---------|------------------|------------------|
| **Feature Plan** | Comprehensive implementation plan with goals, scope, acceptance criteria, and PR breakdown | Developers, reviewers | `<feature>-implementation-plan.md` |
| **Implementation Notes** | Technical guidance: DB schemas, validation rules, error codes, SQL examples | Backend developers | `<feature>-implementation-notes.md` |
| **API Specification** | Formal API contract in OpenAPI/YAML format | Frontend, backend, QA | `<feature>-spec.yaml` |
| **Implementation Log** | PR-by-PR changelog with commits, file changes, status | Reviewers, maintainers | `<feature>_log.md` or `<feature>-log.md` |
| **Investigation Report** | Analysis of current state, gaps, proposed solutions | Tech leads, architects | `<Topic> - Investigation and Strategy Report.md` |
| **Rollout Plan** | Deployment steps, risks, testing strategy, rollback procedures | DevOps, backend leads | `<Topic> - Rollout Plan.md` |
| **API Change Backlog** | Tracks breaking API changes requiring frontend coordination | Frontend developers | `<Topic> changes backlog.md` |
| **Integration Guide** | Technical reference for integrating with a system/feature | Frontend, external consumers | `<Topic> Integration Guide.md` |

---

## Naming Conventions

### Folder structure

```
vibecode_docs/
├── <feature_name>/           # Lowercase, underscores
│   ├── <feature>-spec.yaml
│   ├── <feature>-implementation-plan.md
│   └── ...
├── <topic_name>/             # Lowercase, underscores
│   └── ...
├── DOCS_STANDARDS.md         # This file
└── README.md                 # Table of contents
```

### File naming rules

1. **Use lowercase with hyphens or underscores** for feature-scoped docs:
   - `maps-api-spec.yaml`
   - `maps-api-implementation-plan.md`

2. **Legacy naming (Title Case with spaces)** — for reference only, do not use for new files:
   - `User Model - Investigation and Strategy Report.md` (now: `user-model-investigation-and-strategy-report.md`)
   - `Auth Frontend Integration Guide.md` (now: `auth-frontend-integration-guide.md`)

3. **Suffix conventions:**
   - `-spec.yaml` — API specifications
   - `-implementation-plan.md` — Feature plans
   - `-implementation-notes.md` — Technical notes
   - `_log.md` or `-log.md` — Implementation logs

> **Note:** Some existing files use mixed conventions. This standard documents observed patterns; renaming is optional and should be coordinated.

### Preferred naming for NEW documents

For **new** documents, use **lowercase with hyphens** (kebab-case):
- `feature-name-implementation-plan.md`
- `feature-name-spec.yaml`

Title Case with spaces (`User Model - Rollout Plan.md`) is acceptable only for legacy/existing documents. Do not create new files in Title Case style.

---

## Metadata Block (Recommended)

Each document SHOULD include a metadata block at the top. Two formats are acceptable:

### Format A: Markdown quote block (current convention)

```markdown
> **Source of truth:** This document describes [scope].
> Verified against branch `<branch>` (HEAD `<commit>`).
> **Status:** [Draft | In Review | Approved | Implemented | Deprecated]
> **Last updated:** YYYY-MM-DD
```

### Format B: YAML front matter (alternative)

```yaml
---
title: Document Title
type: Feature Plan | Implementation Notes | API Specification | ...
status: Draft | In Review | Approved | Implemented | Deprecated
branch: feature/maps-api
last_updated: 2026-02-05
author: @username (optional)
---
```

### Required metadata fields

| Field | Description | Example |
|-------|-------------|---------|
| `status` | Document lifecycle status | `Implemented` |
| `type` | One of the types from the catalog | `Feature Plan` |
| `branch` | Git branch where changes were made (if applicable) | `feature/maps-api-align-spec` |
| `last_updated` | ISO date of last significant update | `2026-02-05` |
| `pr` or `commit` | **Required for `Implemented` status.** Link to PR or commit hash. | `#32` or `d792f26` |

---

## Status Lifecycle

| Status | Meaning | Next Status |
|--------|---------|-------------|
| **Draft** | Work in progress, not ready for review | In Review |
| **In Review** | Awaiting feedback/approval | Approved / Draft (if rejected) |
| **Approved** | Ready for implementation | Implemented |
| **Implemented** | Code changes merged, matches codebase. **Must include PR link or commit hash in metadata.** | Deprecated (if obsoleted) |
| **Deprecated** | No longer accurate or relevant; kept for history | — |

### Status badges in documents

Use status at the top of the document:

```markdown
> **Status:** Implemented
```

Or inline in tables (for logs):

| PR | Status |
|----|--------|
| PR-1 | Done |
| PR-2 | In Progress |

---

## Document Templates

### Feature Plan template

```markdown
# <Feature Name> Implementation Plan

> **Type:** Feature Plan
> **Status:** Draft
> **Branch:** feature/<feature-name>
> **Last updated:** YYYY-MM-DD
> **PR:** (add when Implemented, e.g., `#32` or commit `abc1234`)

## A. Goal
[One sentence summary]

## B. Target State
[Description of the end state]

## C. Scope
- In scope: ...
- Out of scope: ...

## D. Current State
[What exists today]

## E. Changes Required
[List of changes]

## F. Test Strategy
[How changes will be verified]

## G. Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## H. PR Plan
| PR | Description | Files |
|----|-------------|-------|
| PR-1 | ... | ... |

## I. Risks
[Known risks and mitigations]
```

### Implementation Log template

```markdown
# <Feature Name> Implementation Log

> **Type:** Implementation Log
> **Status:** In Progress
> **Branch:** feature/<feature-name>
> **Last updated:** YYYY-MM-DD

## PR-1: <Title>
**Commit:** `<hash>`
**Status:** Done | In Progress | Blocked

### Changes
- File 1: description
- File 2: description

### Verification
- [ ] Tests pass
- [ ] Manual check

---

## PR-2: <Title>
...
```

### Investigation Report template

```markdown
# <Topic> - Investigation and Strategy Report

> **Type:** Investigation Report
> **Status:** Approved
> **Last updated:** YYYY-MM-DD

## Executive Summary
[Brief overview of findings]

## Current State
[Analysis of existing implementation]

## Identified Gaps
[Problems found]

## Proposed Solution
[Recommended approach]

## Trade-offs
[Pros and cons of the approach]

## Next Steps
[Action items]
```

---

## Cross-References

When referencing other documents in the same folder:

```markdown
See [user-model-rollout-plan.md](user-model-rollout-plan.md) for deployment steps.
```

When referencing code:

```markdown
Source: `internal/pkg/auth/delivery/auth_handlers.go:35-83`
```

---

## Uncertainty Markers

When information is uncertain or requires verification:

- Use `**VERIFY:**` prefix for statements that need confirmation
- Use `(предположительно)` or `(presumably)` for educated guesses
- Use `[TODO]` for incomplete sections

Example:
```markdown
**VERIFY:** The backend may require `redirect_uri` in the token exchange request.
```

---

## Changelog

| Date | Change |
|------|--------|
| 2026-02-05 | Initial version created |
| 2026-02-05 | Added: preferred naming for new docs (kebab-case); PR/commit required for Implemented status |
