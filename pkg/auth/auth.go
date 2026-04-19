package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/jgfranco17/reposcout/pkg/logging"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultGitHubHost is the default GitHub API host
	DefaultGitHubHost string = "github.com"

	netrcFileName string = ".netrc"
)

type CredentialClient struct {
	netrcPath string
}

func NewClient(ctx context.Context) (*CredentialClient, error) {
	logger := logging.FromContext(ctx)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine user home directory: %w", err)
	}
	if homeDir == "" {
		return nil, fmt.Errorf("user home directory is empty")
	}

	netrcPath := filepath.Join(homeDir, netrcFileName)
	logger.WithFields(logrus.Fields{
		"path": netrcPath,
	}).Info("Using netrc file")
	return &CredentialClient{netrcPath: netrcPath}, nil
}

// LoadToken reads a GitHub token from the netrc file.
// It searches for the entry matching the given host (default: github.com).
// Returns the token value or an error if not found.
func (c *CredentialClient) LoadToken(host string) (string, error) {
	if host == "" {
		host = DefaultGitHubHost
	}

	netrcFile, err := netrc.ParseFile(c.netrcPath)
	if err != nil {
		return "", fmt.Errorf("cannot parse netrc file: %w", err)
	}

	machine := netrcFile.FindMachine(host)
	if machine == nil || machine.Name != host {
		return "", fmt.Errorf("no credentials found in netrc for host %q", host)
	}

	if machine.Password == "" {
		return "", fmt.Errorf("no password found in netrc for host %q", host)
	}

	return machine.Password, nil
}
