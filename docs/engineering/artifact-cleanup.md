# Artifact Cleanup

Satisfies SPECS §16.4: stale filesystem artifacts must be marked or recorded for later cleanup.

## Mechanism

All submission artifacts (source files, compiled binaries) are written through
`artifact.LocalRepository` (`internal/artifact/`). The repository stores files
under `<ArtifactDir>/files/<uuid>` and returns an opaque `artifact.ID`.

The DB is the manifest:
- `source_codes.storage_key` — every source file that should exist
- `submissions.binary_path` — every binary that should exist (non-null, status `compiled`)

### Forward cleanup (new submissions)

`submission.Service.Submit` compensates in-process on failure:

| Fails at | Compensation |
|---|---|
| `CreateSourceCode` | `repo.Delete(sourceID)` |
| `UpdateSubmissionBuild` (compiled path) | `repo.Delete(artifactID)` |
| Build failure | none — source is tracked, no binary was written |

### Retroactive cleanup (startup scan)

A startup scan can find orphans by querying all known IDs from the DB and
calling `repo.Exists` for each. Missing files are logged for operator action.
This covers artifacts left by process crashes between `repo.Write` and the
subsequent DB write.

Implementation of the startup scan is deferred — see the artifact-cleanup story.

### Deferred: crash-safe SAGA via event store

Full crash-safe compensation requires durable events per SAGA step. The
existing event store is match-scoped (SPECS §14.7) and cannot be reused here
without a schema change. This is tracked as a follow-up story.
