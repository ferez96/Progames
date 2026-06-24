package artifact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	// 0o755 — binaries must be executable; applying the same mode to source files is harmless.
	filePerm os.FileMode = 0o755
	// 0o755 — the process may resolve paths under a different uid, so group/other need traversal.
	dirPerm os.FileMode = 0o755
)

type LocalRepository struct {
	dir string
}

func NewLocalRepository(artifactDir string) *LocalRepository {
	return &LocalRepository{dir: filepath.Join(artifactDir, "files")}
}

func (r *LocalRepository) Write(ctx context.Context, content io.Reader) (ID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate artifact id: %w", err)
	}
	aid := ID(id.String())
	path := r.pathFor(aid)
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, content); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return aid, nil
}

func (r *LocalRepository) Read(ctx context.Context, id ID) (File, error) {
	path := r.pathFor(id)
	f, err := os.Open(path)
	if err != nil {
		return File{}, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return File{}, err
	}
	return File{ID: id, Content: f, Size: info.Size()}, nil
}

func (r *LocalRepository) Delete(ctx context.Context, id ID) error {
	// Idempotent — SAGA compensation must not fail when the file was never written.
	err := os.Remove(r.pathFor(id))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (r *LocalRepository) Exists(ctx context.Context, id ID) (bool, error) {
	_, err := os.Stat(r.pathFor(id))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *LocalRepository) ResolvePath(id ID) string {
	return r.pathFor(id)
}

func (r *LocalRepository) pathFor(id ID) string {
	return filepath.Join(r.dir, string(id))
}
