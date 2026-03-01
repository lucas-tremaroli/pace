package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/mcp"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server for AI assistant integration",
	Long: `Model Context Protocol (MCP) server for integration with AI assistants.

Use 'pace mcp install' to configure Claude Code to use Pace.
Use 'pace mcp uninstall' to remove the configuration.

The server is started automatically by Claude Code once installed.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If run without subcommand, start the server (for Claude to invoke)
		server, err := mcp.NewServer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to start MCP server: %v\n", err)
			os.Exit(1)
		}

		if err := server.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
	},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Configure Claude Code to use Pace MCP server",
	Long: `Adds Pace to Claude Code's MCP server configuration.

This command registers the Pace MCP server with Claude Code CLI,
making pace tools available in your Claude conversations.`,
	Run: func(cmd *cobra.Command, args []string) {
		runInstall()
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Pace from Claude Code's MCP server configuration",
	Run: func(cmd *cobra.Command, args []string) {
		runUninstall()
	},
}

func init() {
	McpCmd.AddCommand(installCmd)
	McpCmd.AddCommand(uninstallCmd)
}

// SetVersion sets the version for the MCP handler
func SetVersion(version string) {
	mcp.Version = version
}

func getPacePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	return execPath, nil
}

func findClaudeCLI() (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found in PATH - please install Claude Code first")
	}
	return path, nil
}

func runInstall() {
	claudePath, err := findClaudeCLI()
	if err != nil {
		output.Error(err)
		return
	}

	pacePath, err := getPacePath()
	if err != nil {
		output.Error(err)
		return
	}

	// Use claude mcp add to register the server
	cmd := exec.Command(claudePath, "mcp", "add", "pace", "--", pacePath, "mcp")
	if out, err := cmd.CombinedOutput(); err != nil {
		output.ErrorMsg(fmt.Sprintf("failed to register MCP server with Claude Code: %s", strings.TrimSpace(string(out))))
		return
	}

	tools := []string{
		"pace_context",
		"pace_task_list",
		"pace_task_create",
		"pace_task_update",
		"pace_task_delete",
		"pace_task_ready",
		"pace_task_dep_add",
		"pace_task_dep_remove",
		"pace_note_list",
		"pace_note_create",
		"pace_note_read",
		"pace_note_delete",
		"pace_task_log",
		"pace_task_close",
		"pace_task_logs",
	}

	output.Success("Pace MCP server installed", map[string]any{
		"command": pacePath,
		"tools":   tools,
	})
}

func runUninstall() {
	claudePath, err := findClaudeCLI()
	if err != nil {
		output.Error(err)
		return
	}

	cmd := exec.Command(claudePath, "mcp", "remove", "pace")
	out, cmdErr := cmd.CombinedOutput()

	if cmdErr != nil {
		// Check if it's just "not found" - that's okay
		if exitErr, ok := cmdErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			output.Success("Pace MCP server is not currently installed", nil)
			return
		}
		output.ErrorMsg(fmt.Sprintf("failed to remove MCP server from Claude Code: %s", strings.TrimSpace(string(out))))
		return
	}

	output.Success("Pace MCP server uninstalled", nil)
}
