package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/jgfranco17/reposcout/pkg/auth"
	"github.com/jgfranco17/reposcout/pkg/logging"
	"github.com/sirupsen/logrus"
)

// Client manages a Copilot session scoped to a repository root.
type Client struct {
	workDir string
	client  *copilot.Client
	session *copilot.Session
	mu      sync.Mutex
	token   string

	onDelta   func(chunk string)
	onToolUse func(name string)
}

type AuthClient interface {
	LoadToken(host string) (string, error)
}

// New returns an Agent rooted at workDir (should be the repo root).
func New(ctx context.Context, workDir string, authClient AuthClient) (*Client, error) {
	logger := logging.FromContext(ctx)

	abs, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolving work dir: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("working directory not accessible: %w", err)
	}

	token, err := authClient.LoadToken(auth.DefaultGitHubHost)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"workDir": abs,
	}).Info("Initialized agent instance")
	return &Client{workDir: abs, token: token}, nil
}

// OnDelta registers a callback invoked with each streaming text chunk.
func (a *Client) OnDelta(fn func(chunk string)) {
	a.onDelta = fn
}

// OnToolUse registers a callback invoked when the agent calls a tool.
func (a *Client) OnToolUse(fn func(name string)) {
	a.onToolUse = fn
}

// Start connects to the Copilot CLI, creates a session, and registers codebase tools.
func (a *Client) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	client := copilot.NewClient(&copilot.ClientOptions{
		Cwd:         a.workDir,
		GitHubToken: a.token,
	})
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("starting copilot client: %w", err)
	}

	session, err := client.CreateSession(ctx, &copilot.SessionConfig{
		ClientName:            "reposcout",
		WorkingDirectory:      a.workDir,
		EnableConfigDiscovery: true,
		Streaming:             true,
		OnPermissionRequest:   copilot.PermissionHandler.ApproveAll,
		Tools:                 repoTools(a.workDir),
		OnEvent: func(event copilot.SessionEvent) {
			switch d := event.Data.(type) {
			case *copilot.AssistantMessageDeltaData:
				if a.onDelta != nil {
					a.onDelta(d.DeltaContent)
				}
			case *copilot.ToolExecutionStartData:
				if a.onToolUse != nil {
					a.onToolUse(d.ToolName)
				}
			}
		},
	})
	if err != nil {
		client.Stop()
		if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "Session was not created") {
			return fmt.Errorf("session creation failed - check your authentication credentials.\n\nOriginal error: %w", err)
		}
		return fmt.Errorf("creating session: %w", err)
	}

	a.client = client
	a.session = session
	return nil
}

// Ask sends prompt to the agent and blocks until the response is complete.
// Streaming chunks are delivered via the OnDelta callback while this blocks.
func (a *Client) Ask(ctx context.Context, prompt string) error {
	a.mu.Lock()
	session := a.session
	a.mu.Unlock()

	if session == nil {
		return fmt.Errorf("agent not started")
	}

	_, err := session.SendAndWait(ctx, copilot.MessageOptions{Prompt: prompt})
	return err
}

// Stop disconnects the session and shuts down the Copilot CLI process.
func (a *Client) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.session != nil {
		a.session.Disconnect()
		a.session = nil
	}
	if a.client != nil {
		a.client.Stop()
		a.client = nil
	}
}

// WorkDir returns the resolved repository root this agent is scoped to.
func (a *Client) WorkDir() string {
	return a.workDir
}
