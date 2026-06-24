package artifact

import (
	"context"
	"io"
)

type ID string

type File struct {
	ID      ID
	Content io.ReadCloser
	Size    int64
}

// Crash-safe SAGA compensation via event store is deferred — tracked in the artifact-cleanup story.
type Repository interface {
	Write(ctx context.Context, r io.Reader) (ID, error)
	Read(ctx context.Context, id ID) (File, error)
	Delete(ctx context.Context, id ID) error
	Exists(ctx context.Context, id ID) (bool, error)
}

// Not implementable by remote backends without a local cache layer —
// use only where direct execution of the file is required (e.g. matchexec).
type PathResolver interface {
	ResolvePath(id ID) string
}
