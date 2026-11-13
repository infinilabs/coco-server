---
title: "Record Path Contract"
weight: 60
---

# Record Path Contract (`document.*` freeze)

When you write a record field path into a pipeline configuration — a processor's field argument, a `when`/`if` condition, a `required_fields` assertion — that path becomes part of Coco's public contract. This page freezes those paths. Treat every path listed here as a stable API: what you build against it today keeps working across minor releases.

## Frozen Paths

| Path | Meaning | Produced by |
|------|---------|-------------|
| `document.*` | The full `core.Document` tree projection — `title`, `content`, `lang`, `tags`, `summary`, `source.id`, `content_hash`, `status`, and every other document field | `coco_document` codec (record namespace `document`) |
| `document.document_chunk` | Cold subtree: opaque raw bytes inside the chain (contract-wise treat as a black box; only the embedding processors read/write it) | `coco_document` codec |
| `attachment.*` | The `core.Attachment` tree projection | `coco_attachment` codec (record namespace `attachment`) |
| `metadata.labels._trace_id` | Record trace ID — correlate a record across pipeline stages and flight records | Entry points (consumer, `lane_consumer`, connectors) |
| `metadata.labels.datasource`, `metadata.labels.connector_id`, `metadata.labels.lang`, `metadata.labels.type` | Routing metadata labels | `coco_document` codec |
| `tags` | Failure-tag landing field — `on_failure: tag` merges its tag here before re-encoding | `for_each` / `lane_consumer` hosts |

## Freeze Rules

1. **Additive evolution.** New `Document` fields automatically become visible as `document.<json_field_name>` (whole-tree projection) — new paths appear without a release note, and no existing path changes meaning.
2. **Hot-path names are immutable.** Field names in the table above and every `metadata.labels` key, once released, are never renamed.
3. **Breaking changes are major-version events.** Removing or renaming anything under `document.*` (or `attachment.*`) only happens in a major version, signaled by a `coco_document` codec version bump — never in a minor or patch release.

## What This Means in Practice

- **Pipeline configs you write today keep compiling and running.** Conditions like `when: { equals: { document.lang: "en" } }` and field references like `document.content` are contract-frozen.
- **Unknown or misspelled paths surface early.** Dry-run (`POST /pipeline/dry-run`) validates `required_fields` assertions against the sample, and compile-time contract checks reject processors placed in the wrong record namespace (a `document`-namespace processor inside an `attachment` chain fails to compile).
- **Cross-product compatibility.** The same paths are honored by Coco's processing chains and any pipeline built on the shared framework engine — the `document.*` namespace is the interchange contract, not a Coco-internal detail.

## Scope Notes

- Paths outside the frozen set — transient chain-local fields, internal underscores-prefixed keys, and namespace codec subtrees other than `document`/`attachment` — are **not** contract; do not build on them.
- The `document.document_chunk` subtree is contract-opaque: it survives round-trips byte-stably, but its internal structure may change without notice.
