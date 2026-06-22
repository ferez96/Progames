package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

type SourceCode struct {
	ID          string    `db:"id"`
	Language    string    `db:"language"`
	StorageType string    `db:"storage_type"`
	StorageKey  string    `db:"storage_key"`
	Size        int64     `db:"size"`
	CreatedAt   time.Time `db:"created_at"`
}

type Submission struct {
	ID            int64          `db:"id"`
	UserID        int64          `db:"user_id"`
	SourceCodeID  string         `db:"source_code_id"`
	Status        string         `db:"status"`
	Msg           string         `db:"msg"`
	CompileOutput string         `db:"compile_output"`
	BinaryPath    sql.NullString `db:"binary_path"`
	CreatedAt     time.Time      `db:"created_at"`
}

func (s *Store) CreateSourceCode(id, storageKey string, size int64) error {
	_, err := s.DB.Exec(`INSERT INTO source_codes (id, language, storage_type, storage_key, size) VALUES (?, 'Go', 'local', ?, ?)`,
		id, storageKey, size)
	return err
}

func (s *Store) CreateSubmission(userID int64, sourceID string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO submissions (user_id, source_code_id, status) VALUES (?, ?, 'pending')`, userID, sourceID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateSubmissionBuild(id int64, status, msg, output, binaryPath string) error {
	_, err := s.DB.Exec(`UPDATE submissions SET status = ?, msg = ?, compile_output = ?, binary_path = ? WHERE id = ?`,
		status, msg, output, nullString(binaryPath), id)
	return err
}

func (s *Store) SubmissionByID(id int64) (Submission, error) {
	var sub Submission
	err := s.DB.Get(&sub, `SELECT * FROM submissions WHERE id = ?`, id)
	return sub, err
}

func (s *Store) SourcePath(sourceID string) string {
	base, _ := filepath.Abs(s.ArtifactDir)
	return filepath.Join(base, "sources", sourceID, "main.go")
}

func (s *Store) BinaryPath(submissionID int64) string {
	base, _ := filepath.Abs(s.ArtifactDir)
	name := "bot"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(base, "bins", fmt.Sprintf("%d", submissionID), name)
}
