package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jgfranco17/reposcout/pkg/logging"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadToken(t *testing.T) {
	// Create a temporary netrc file
	tmpDir := t.TempDir()
	netrcPath := filepath.Join(tmpDir, ".netrc")
	netrcContent := `machine github.com
password ghp_test_token_123456

machine gitlab.com
password glpat_other_token

default
password default_token
`

	err := os.WriteFile(netrcPath, []byte(netrcContent), 0o600)
	require.NoError(t, err)

	// Temporarily override HOME
	t.Setenv("HOME", tmpDir)

	// Create test context with logger
	ctx := context.Background()
	logger := logrus.New()
	ctx = logging.AddToContext(ctx, logger)

	// Create credential client
	client, err := NewClient(ctx)
	require.NoError(t, err)

	tests := []struct {
		name    string
		host    string
		want    string
		wantErr bool
	}{
		{
			name:    "reads github.com token",
			host:    "github.com",
			want:    "ghp_test_token_123456",
			wantErr: false,
		},
		{
			name:    "reads gitlab.com token",
			host:    "gitlab.com",
			want:    "glpat_other_token",
			wantErr: false,
		},
		{
			name:    "returns error for unknown host",
			host:    "unknown.com",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.LoadToken(tt.host)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestLoadTokenNotFound(t *testing.T) {
	// Create test context with logger
	ctx := context.Background()
	logger := logrus.New()
	ctx = logging.AddToContext(ctx, logger)

	// Create a temporary directory without a netrc file
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create credential client
	client, err := NewClient(ctx)
	require.NoError(t, err)

	// Try to load a token from non-existent netrc
	_, err = client.LoadToken("github.com")
	assert.Error(t, err)
}

func TestNewClient(t *testing.T) {
	// Create test context with logger
	ctx := context.Background()
	logger := logrus.New()
	ctx = logging.AddToContext(ctx, logger)

	// Test that NewClient creates a client with the correct netrc path
	testHome := t.TempDir()
	t.Setenv("HOME", testHome)

	client, err := NewClient(ctx)
	require.NoError(t, err)

	expectedPath := filepath.Join(testHome, ".netrc")
	assert.Equal(t, expectedPath, client.netrcPath)
}
