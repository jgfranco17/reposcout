package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jgfranco17/reposcout/pkg/agent"
	"github.com/jgfranco17/reposcout/pkg/auth"
	"github.com/jgfranco17/reposcout/pkg/logging"
)

func main() {
	logger := logging.New(os.Stderr, logging.GetLogLevel())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx = logging.AddToContext(ctx, logger)

	workDir, err := findRepoRoot()
	if err != nil {
		logger.WithError(err).Fatal("Failed to find repository root")
	}

	authClient, err := auth.NewClient(ctx)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize auth client")
	}

	a, err := agent.New(ctx, workDir, authClient)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize agent")
	}

	a.OnDelta(func(chunk string) {
		fmt.Print(chunk)
	})

	a.OnToolUse(func(name string) {
		logger.Infof("[tool: %s]", name)
	})

	if err := a.Start(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to start agent")
	}
	defer a.Stop()

	printBanner(a.WorkDir())
	repl(ctx, a)
}

// repl runs the interactive prompt loop until EOF or context cancellation.
func repl(ctx context.Context, a *agent.Client) {
	logger := logging.FromContext(ctx)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n\033[1;36myou>\033[0m ")
		if !scanner.Scan() {
			break
		}
		prompt := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if prompt == "" {
			continue
		}
		if prompt == "/exit" || prompt == "/quit" {
			break
		}

		fmt.Print("\n\033[1;32mreposcout>\033[0m\n")

		if err := a.Ask(ctx, prompt); err != nil {
			if ctx.Err() != nil {
				break
			}
			logger.WithError(err).Error("Failed to send prompt to agent")
		}
		if ctx.Err() != nil {
			logger.Warn("Context cancelled, exiting")
			break
		}
		fmt.Println() // newline after streamed response
	}

	if err := scanner.Err(); err != nil {
		logger.WithError(err).Error("Input error")
	}
	logger.Info("Closing reposcout")
}

// findRepoRoot walks up from the current directory to locate a .git folder.
// Falls back to the current directory if no git root is found.
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root — use cwd as fallback.
			return cwd, nil
		}
		dir = parent
	}
}

func printBanner(workDir string) {
	fmt.Printf("\033[1;35m")
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║         reposcout  🔭                ║")
	fmt.Println("║  AI codebase explorer (Copilot SDK)  ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Printf("\033[0m")
	fmt.Printf("repo root: %s\n", workDir)
	fmt.Printf("type your question, /exit to quit\n")
}
