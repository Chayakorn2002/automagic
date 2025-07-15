package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bilbo290/automagic/pkg/config"
	"github.com/bilbo290/automagic/pkg/provider"
)

type MCPDebugInfo struct {
	MCPAvailable     bool     `json:"mcp_available"`
	AvailableTools   []string `json:"available_tools"`
	GitLabMCPWorking bool     `json:"gitlab_mcp_working"`
	ErrorMessages    []string `json:"error_messages"`
	TestResults      []string `json:"test_results"`
}

func TestMCPAvailability(cfg *config.Config) (*MCPDebugInfo, error) {
	debug := &MCPDebugInfo{
		ErrorMessages:  make([]string, 0),
		TestResults:    make([]string, 0),
		AvailableTools: make([]string, 0),
	}

	// Test 1: Check if Claude can access MCP tools
	debug.TestResults = append(debug.TestResults, "=== Test 1: MCP Tool Availability ===")

	mcpTestPrompt := `List all available tools and check if GitLab MCP tools work.

MCP_AVAILABLE: yes/no
AVAILABLE_TOOLS: tool1, tool2, tool3
GITLAB_MCP_WORKING: yes/no
ERROR_MESSAGES: any errors`

	mcpResult, err := runClaudeCommand(mcpTestPrompt, cfg)
	if err != nil {
		debug.ErrorMessages = append(debug.ErrorMessages, fmt.Sprintf("Failed to test MCP availability: %v", err))
		return debug, err
	}

	debug.TestResults = append(debug.TestResults, mcpResult)

	// Parse the response
	lines := strings.Split(mcpResult, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MCP_AVAILABLE:") {
			debug.MCPAvailable = strings.Contains(strings.ToLower(line), "yes")
		} else if strings.HasPrefix(line, "AVAILABLE_TOOLS:") {
			toolsStr := strings.TrimPrefix(line, "AVAILABLE_TOOLS:")
			toolsStr = strings.TrimSpace(toolsStr)
			if toolsStr != "" {
				tools := strings.Split(toolsStr, ",")
				for _, tool := range tools {
					debug.AvailableTools = append(debug.AvailableTools, strings.TrimSpace(tool))
				}
			}
		} else if strings.HasPrefix(line, "GITLAB_MCP_WORKING:") {
			debug.GitLabMCPWorking = strings.Contains(strings.ToLower(line), "yes")
		} else if strings.HasPrefix(line, "ERROR_MESSAGES:") {
			errorStr := strings.TrimPrefix(line, "ERROR_MESSAGES:")
			errorStr = strings.TrimSpace(errorStr)
			if errorStr != "" {
				debug.ErrorMessages = append(debug.ErrorMessages, errorStr)
			}
		}
	}

	return debug, nil
}

func TestProviderMCPIntegration(cfg *config.Config, providerInstance provider.Provider, projectPath string) error {
	providerType := "GitLab"
	if cfg.IsGitHub() {
		providerType = "GitHub"
	}

	fmt.Printf("=== Testing %s MCP Integration ===\n\n", providerType)

	// Test 1: Check MCP availability
	fmt.Printf("Step 1: Testing MCP availability...\n")
	mcpDebug, err := TestMCPAvailability(cfg)
	if err != nil {
		return fmt.Errorf("failed to test MCP availability: %v", err)
	}

	fmt.Printf("MCP Available: %v\n", mcpDebug.MCPAvailable)
	fmt.Printf("%s MCP Working: %v\n", providerType, mcpDebug.GitLabMCPWorking)
	fmt.Printf("Available Tools: %v\n", mcpDebug.AvailableTools)

	if len(mcpDebug.ErrorMessages) > 0 {
		fmt.Printf("Error Messages:\n")
		for _, msg := range mcpDebug.ErrorMessages {
			fmt.Printf("  - %s\n", msg)
		}
	}

	// Test 2: If MCP is available, test provider-specific operations
	if mcpDebug.MCPAvailable {
		fmt.Printf("\nStep 2: Testing %s MCP operations...\n", providerType)

		providerTestPrompt := fmt.Sprintf(`You have access to %s MCP tools. Please test the following operations:

1. Try to get project information for: %s
2. Try to list issues for the project
3. Try to get user information

Please use the actual %s MCP tools if available. If you encounter any errors, please describe them in detail.

Project path: %s
Provider URL: %s

Please respond with detailed information about:
- Which %s MCP tools worked
- Any errors encountered
- The actual data returned (if any)
`, providerType, projectPath, providerType, projectPath, cfg.ProviderURL, providerType)

		providerResult, err := runClaudeCommand(providerTestPrompt, cfg)
		if err != nil {
			fmt.Printf("Error testing %s MCP: %v\n", providerType, err)
		} else {
			fmt.Printf("%s MCP Test Results:\n%s\n", providerType, providerResult)
		}
	}

	// Test 3: Compare with direct API access
	fmt.Printf("\nStep 3: Comparing with direct API access...\n")

	// Get project info via direct API
	fmt.Printf("Direct API - Project info: ")
	projects, err := providerInstance.GetAccessibleProjects()
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	} else {
		fmt.Printf("Success - Found %d projects\n", len(projects))
		for _, project := range projects {
			if project.PathWithNamespace == projectPath {
				fmt.Printf("  Target project found: %s (ID: %d)\n", project.PathWithNamespace, project.ID)
				break
			}
		}
	}

	// Get issues via direct API
	fmt.Printf("Direct API - Issues: ")
	issues, err := providerInstance.GetProjectIssues(projectPath, []string{}, "opened")
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	} else {
		fmt.Printf("Success - Found %d open issues\n", len(issues))
	}

	fmt.Printf("\n=== MCP Debug Complete ===\n")
	return nil
}

func runClaudeCommand(prompt string, cfg *config.Config) (string, error) {
	// Set up environment first - this is crucial for MCP server initialization
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return "", fmt.Errorf("HOME environment variable not set")
	}

	// Use the user's shell to preserve environment
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
		fmt.Printf("Debug: Using default shell: %s\n", shell)
	} else {
		fmt.Printf("Debug: Using shell: %s\n", shell)
	}

	// Create a more focused command for MCP testing
	claudeCmd := fmt.Sprintf("%s %s -p %q", cfg.ClaudeCommand, cfg.ClaudeFlags, prompt)
	fmt.Printf("Debug: Running command: %s\n", claudeCmd)
	fmt.Printf("Debug: Working directory: %s\n", homeDir)

	cmd := exec.Command(shell, "-c", claudeCmd)

	// Set environment and working directory BEFORE anything else
	cmd.Env = os.Environ()
	cmd.Dir = homeDir
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting claude command: %v", err)
	}

	var result strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Debug: Raw output line: %s\n", line)

		// Try to parse as JSON first
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			// Not JSON, treat as regular text
			fmt.Printf("Debug: Non-JSON line: %s\n", line)
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Handle streaming JSON format
		fmt.Printf("Debug: JSON data: %+v\n", jsonData)
		if content, ok := jsonData["content"].(string); ok {
			fmt.Printf("Debug: Found content: %s\n", content)
			result.WriteString(content)
		} else if delta, ok := jsonData["delta"].(string); ok {
			fmt.Printf("Debug: Found delta: %s\n", delta)
			result.WriteString(delta)
		} else if resultField, ok := jsonData["result"].(string); ok {
			fmt.Printf("Debug: Found result: %s\n", resultField)
			result.WriteString(resultField)
		}
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("Debug: Command execution failed: %v\n", err)
		return "", fmt.Errorf("error executing claude command: %v", err)
	}

	finalResult := result.String()
	fmt.Printf("Debug: Final result length: %d\n", len(finalResult))
	fmt.Printf("Debug: Final result: %s\n", finalResult)

	return finalResult, nil
}

func CreateMCPDebugProcess(issueNumber int, cfg *config.Config, projectPath string) (*Process, error) {
	processID := fmt.Sprintf("mcp-debug-%d-%d", issueNumber, time.Now().Unix())

	providerType := cfg.ProviderType

	prompt := fmt.Sprintf(`# MCP Debug Session for Issue #%[1]d

## Debug Task
Please help debug %[2]s MCP integration for issue #%[1]d in project %[3]s.

## Debug Steps:
1. **Check MCP Availability**: 
   - List all available MCP tools
   - Specifically check for %[2]s MCP tools

2. **Test %[2]s MCP Functions**:
   - Try to connect to %[2]s: %[4]s
   - Try to get project info for: %[3]s
   - Try to get issue #%[1]d details
   - Try to list project issues

3. **Detailed Error Reporting**:
   - Report any connection errors
   - Report any authentication issues
   - Report any missing tools or permissions

4. **Fallback Options**:
   - If MCP doesn't work, suggest alternative approaches
   - Show how to use standard git/curl commands as fallback

## Configuration
- %[2]s URL: %[4]s
- Project: %[3]s
- Issue: #%[1]d
- Username: @%[5]s

Please provide detailed debugging information and suggest solutions.
`, issueNumber, providerType, projectPath, cfg.ProviderURL, cfg.ProviderUsername)

	return CreateProcess(
		issueNumber,
		processID,
		cfg.ClaudeCommand,
		cfg.ClaudeFlags,
		projectPath,
		cfg.ProviderUsername,
		prompt,
	)
}
