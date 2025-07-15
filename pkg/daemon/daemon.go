package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/bilbo290/automagic/pkg/claude"
	"github.com/bilbo290/automagic/pkg/config"
	"github.com/bilbo290/automagic/pkg/enum"
	"github.com/bilbo290/automagic/pkg/provider"
	"github.com/bilbo290/automagic/pkg/session"
)

// Daemon manages the automated processing of issues from git providers
type Daemon struct {
	provider        provider.Provider
	config          *config.Config
	selectedProject string
	processManager  *claude.ProcessManager
	sessionStore    *session.SQLiteSessionStore
	resumeProcesses map[int]*exec.Cmd // Track resume processes by issue ID
	runType         enum.RunType
	lastCommentTime map[int]string // Track last processed comment timestamp by issue ID
}

// New creates a new daemon instance with normal run type
func New(providerInstance provider.Provider, config *config.Config, runType enum.RunType) *Daemon {
	sessionStore := initializeSessionStore()

	return &Daemon{
		provider:        providerInstance,
		config:          config,
		processManager:  claude.NewProcessManager(),
		sessionStore:    sessionStore,
		resumeProcesses: make(map[int]*exec.Cmd),
		runType:         runType,
		lastCommentTime: make(map[int]string),
	}
}

// initializeSessionStore sets up SQLite session storage
func initializeSessionStore() *session.SQLiteSessionStore {
	sqliteStore, err := session.NewSQLiteSessionStore("")
	if err != nil {
		fmt.Printf("Error: Failed to initialize SQLite session store: %v\n", err)
		fmt.Printf("SQLite session storage is required for daemon functionality.\n")
		os.Exit(1)
	}

	// Clean up old sessions
	cleanupSessions(sqliteStore)

	return sqliteStore
}

// cleanupSessions performs maintenance on the session store
func cleanupSessions(sqliteStore *session.SQLiteSessionStore) {
	// Clean up invalid sessions
	if err := sqliteStore.CleanupInvalidSessions(); err != nil {
		// Log warning but continue
	}

	// Clean up sessions older than 7 days
	if err := sqliteStore.CleanupOldSessions(7 * 24 * time.Hour); err != nil {
		// Log warning but continue
	}
}
