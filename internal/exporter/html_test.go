package exporter

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		accountID      string
		accountDisplay string
		outputFile     string
		categories     []string
		setup          func(t *testing.T, base, accountID string)
		wantErr        bool
		wantEntries    []FileManifestEntry
		wantIndex      []string
	}{
		{
			name:           "creates manifest and index from existing csv files",
			accountID:      "123456789012",
			accountDisplay: "123456789012",
			outputFile:     "files.json",
			categories:     []string{"ec2", "s3", "rds"},
			setup: func(t *testing.T, base, accountID string) {
				t.Helper()
				resourcesDir := filepath.Join(base, accountID, "resources")
				require.NoError(t, os.MkdirAll(resourcesDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(resourcesDir, "ec2.csv"), []byte("a,b,c\n"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(resourcesDir, "s3.csv"), []byte("x,y,z\n"), 0o644))
			},
			wantEntries: []FileManifestEntry{
				{Path: "resources/ec2.csv", DisplayName: "ec2"},
				{Path: "resources/s3.csv", DisplayName: "s3"},
			},
			wantIndex: []string{"AWS Resources (123456789012)", "files.json"},
		},
		{
			name:           "account path as file returns error",
			accountID:      "acct-as-file",
			accountDisplay: "acct-as-file",
			outputFile:     "index.html",
			categories:     []string{"x"},
			setup: func(t *testing.T, base, accountID string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(base, accountID), []byte("not a directory"), 0o644))
			},
			wantErr: true,
		},
		{
			name:           "empty categories still writes manifest and index",
			accountID:      "no-cats",
			accountDisplay: "no-cats",
			outputFile:     "index.html",
			categories:     []string{},
			setup: func(t *testing.T, base, accountID string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(base, accountID), 0o755))
			},
			wantEntries: nil,
			wantIndex:   []string{},
		},
		{
			name:           "custom account display is rendered",
			accountID:      "123456789012",
			accountDisplay: "my-account(123456789012)",
			outputFile:     "index.html",
			categories:     []string{},
			setup: func(t *testing.T, base, accountID string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(base, accountID), 0o755))
			},
			wantEntries: nil,
			wantIndex:   []string{"AWS Resources (my-account(123456789012))"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			tt.setup(t, base, tt.accountID)

			err := GenerateHTML(base, tt.accountID, tt.accountDisplay, tt.outputFile, tt.categories)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			b, readErr := os.ReadFile(filepath.Join(base, tt.accountID, "files.json"))
			require.NoError(t, readErr)
			var entries []FileManifestEntry
			require.NoError(t, json.Unmarshal(b, &entries))
			assert.Equal(t, tt.wantEntries, entries)

			indexPath := filepath.Join(base, tt.accountID, "index.html")
			_, statErr := os.Stat(indexPath)
			require.NoError(t, statErr)
			ib, indexErr := os.ReadFile(indexPath)
			require.NoError(t, indexErr)
			for _, fragment := range tt.wantIndex {
				assert.Contains(t, string(ib), fragment)
			}
		})
	}
}

func TestCreateResourcesZip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, resourcesDir string)
		wantFiles []string
	}{
		{
			name:      "missing resources dir creates empty zip",
			setup:     func(t *testing.T, _ string) { t.Helper() },
			wantFiles: []string{},
		},
		{
			name: "includes only csv files",
			setup: func(t *testing.T, resourcesDir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(resourcesDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(resourcesDir, "ec2.csv"), []byte("a,b\n"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(resourcesDir, "readme.txt"), []byte("ignore"), 0o644))
			},
			wantFiles: []string{"ec2.csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			resourcesDir := filepath.Join(base, "resources")
			tt.setup(t, resourcesDir)

			zipPath := filepath.Join(base, "resources.zip")
			require.NoError(t, createResourcesZip(zipPath, resourcesDir))

			zr, err := zip.OpenReader(zipPath)
			require.NoError(t, err)
			defer func() {
				_ = zr.Close()
			}()

			got := make([]string, 0, len(zr.File))
			for _, f := range zr.File {
				got = append(got, f.Name)
			}
			assert.Equal(t, tt.wantFiles, got)
		})
	}
}

func TestGenerateIndexHTML_FallbackToAccountIDWhenDisplayEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		accountID      string
		accountDisplay string
		wantContains   string
	}{
		{name: "empty display falls back to account id", accountID: "123456789012", accountDisplay: "", wantContains: "AWS Resources (123456789012)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			indexPath := filepath.Join(t.TempDir(), "index.html")
			require.NoError(t, generateIndexHTML(indexPath, tt.accountID, tt.accountDisplay, "all.csv"))
			b, err := os.ReadFile(indexPath)
			require.NoError(t, err)
			assert.Contains(t, string(b), tt.wantContains)
		})
	}
}

// Walk permission failure needs a chmod'd directory; keep dedicated setup (TBL-05).
func TestCreateResourcesZip_ReturnsErrorWhenWalkFails(t *testing.T) {
	base := t.TempDir()
	resourcesDir := filepath.Join(base, "resources")
	blockedDir := filepath.Join(resourcesDir, "blocked")
	if err := os.MkdirAll(blockedDir, 0o755); err != nil {
		t.Fatalf("failed to create blocked dir: %v", err)
	}
	if err := os.Chmod(blockedDir, 0o000); err != nil {
		t.Fatalf("failed to chmod blocked dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(blockedDir, 0o755)
	}()

	zipPath := filepath.Join(base, "resources.zip")
	err := createResourcesZip(zipPath, resourcesDir)
	assert.Error(t, err)
}
