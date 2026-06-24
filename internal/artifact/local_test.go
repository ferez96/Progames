package artifact_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"progames/internal/artifact"
	"progames/internal/testhelper"
)

func TestWriteRead(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	content := []byte("hello artifact")
	id, err := repo.Write(ctx, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	f, err := repo.Read(ctx, id)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	defer func() { _ = f.Content.Close() }()

	if f.ID != id {
		t.Errorf("id mismatch: got %q want %q", f.ID, id)
	}
	if f.Size != int64(len(content)) {
		t.Errorf("size mismatch: got %d want %d", f.Size, len(content))
	}
	got, err := io.ReadAll(f.Content)
	if err != nil {
		t.Fatalf("read content: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q want %q", got, content)
	}
}

func TestWriteStoresExecutableFile(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	id, err := repo.Write(ctx, bytes.NewReader([]byte("data")))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := os.Stat(repo.ResolvePath(id))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// 0o755 — binaries must be executable.
	if info.Mode().Perm()&0o755 != 0o755 {
		t.Errorf("expected perm 0o755, got %o", info.Mode().Perm())
	}
}

func TestExistsAfterWrite(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	id, err := repo.Write(ctx, bytes.NewReader([]byte("x")))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	ok, err := repo.Exists(ctx, id)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !ok {
		t.Error("expected file to exist after write")
	}
}

func TestExistsReturnsFalseForUnknownID(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	ok, err := repo.Exists(ctx, artifact.ID("no-such-id"))
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if ok {
		t.Error("expected false for unknown id")
	}
}

func TestDeleteRemovesFile(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	id, err := repo.Write(ctx, bytes.NewReader([]byte("to delete")))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	ok, err := repo.Exists(ctx, id)
	if err != nil {
		t.Fatalf("exists after delete: %v", err)
	}
	if ok {
		t.Error("expected file to be gone after delete")
	}
}

func TestDeleteIsIdempotent(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	id, err := repo.Write(ctx, bytes.NewReader([]byte("x")))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Errorf("second delete (idempotent) returned error: %v", err)
	}
}

func TestDeleteNonExistentIsIdempotent(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	if err := repo.Delete(ctx, artifact.ID("never-written")); err != nil {
		t.Errorf("delete of non-existent file returned error: %v", err)
	}
}

func TestResolvePathPointsToWrittenFile(t *testing.T) {
	t.Parallel()
	repo := testhelper.NewArtifactRepo(t)
	ctx := context.Background()

	id, err := repo.Write(ctx, bytes.NewReader([]byte("bin")))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	path := repo.ResolvePath(id)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("resolved path %q not accessible: %v", path, err)
	}
}
