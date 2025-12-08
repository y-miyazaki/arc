package exporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test GenerateHTML creates files.json and index.html under the account directory
func TestGenerateHTML_CreatesFilesAndIndex(t *testing.T) {
	base := t.TempDir()
	accountID := "123456789012"
	resourcesDir := filepath.Join(base, accountID, "resources")

	// ensure parent dirs
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatalf("failed to create resources dir: %v", err)
	}

	// create a couple of csv files (only these should appear in manifest)
	if err := os.WriteFile(filepath.Join(resourcesDir, "ec2.csv"), []byte("a,b,c\n"), 0o644); err != nil {
		t.Fatalf("failed to write csv file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourcesDir, "s3.csv"), []byte("x,y,z\n"), 0o644); err != nil {
		t.Fatalf("failed to write csv file: %v", err)
	}

	// call GenerateHTML with several categories including one missing
	outputFile := "files.json"
	categories := []string{"ec2", "s3", "rds"}

	err := GenerateHTML(base, accountID, outputFile, categories)
	if err != nil {
		t.Fatalf("GenerateHTML returned error: %v", err)
	}

	// validate manifest exists and contains only ec2 and s3
	manifestPath := filepath.Join(base, accountID, "files.json")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var entries []FileManifestEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		t.Fatalf("failed to unmarshal manifest json: %v", err)
	}

	// expect two entries for ec2 and s3
	assert.Len(t, entries, 2)
	assert.Equal(t, "resources/ec2.csv", entries[0].Path)
	assert.Equal(t, "ec2", entries[0].DisplayName)
	assert.Equal(t, "resources/s3.csv", entries[1].Path)
	assert.Equal(t, "s3", entries[1].DisplayName)

	// validate index.html exists and contains the output file placeholder
	indexPath := filepath.Join(base, accountID, "index.html")
	ib, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}

	s := string(ib)
	assert.Contains(t, s, "AWS Resources (123456789012)")
	assert.Contains(t, s, outputFile)
}

// If the account directory is a file (not a directory), GenerateHTML should return an error
func TestGenerateHTML_FailsWhenAccountPathIsFile(t *testing.T) {
	base := t.TempDir()
	accountID := "acct-as-file"

	// create a file at the path where a directory is expected
	acctPath := filepath.Join(base, accountID)
	if err := os.WriteFile(acctPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// call GenerateHTML (it should try to create files under base/accountID and fail)
	err := GenerateHTML(base, accountID, "index.html", []string{"x"})
	assert.Error(t, err)
}

// Empty categories should produce a manifest with zero entries and still create index.html
func TestGenerateHTML_EmptyCategories(t *testing.T) {
	base := t.TempDir()
	accountID := "no-cats"
	if err := os.MkdirAll(filepath.Join(base, accountID), 0o755); err != nil {
		t.Fatalf("failed to create account dir: %v", err)
	}

	err := GenerateHTML(base, accountID, "index.html", []string{})
	if err != nil {
		t.Fatalf("GenerateHTML returned error: %v", err)
	}

	// manifest should still exist
	manifestPath := filepath.Join(base, accountID, "files.json")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var entries []FileManifestEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		t.Fatalf("failed to unmarshal manifest json: %v", err)
	}
	assert.Len(t, entries, 0)

	indexPath := filepath.Join(base, accountID, "index.html")
	_, err = os.Stat(indexPath)
	assert.NoError(t, err)
}
