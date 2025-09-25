package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/akurinnoy/kubectx-manager/internal/config"
	"github.com/akurinnoy/kubectx-manager/internal/kubeconfig"
	"github.com/akurinnoy/kubectx-manager/internal/logger"
)

// Version information, set by build flags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var (
	dryRun      bool
	authCheck   bool
	verbose     bool
	quiet       bool
	configFile  string
	kubeConfig  string
	interactive bool
)

var rootCmd = &cobra.Command{
	Use:   "kubectx-manager",
	Short: "Advanced Kubernetes context management tool",
	Long: `kubectx-manager is a CLI tool that intelligently manages Kubernetes contexts in your kubeconfig file.
It features advanced pattern matching, authentication validation, cluster reachability checks, and comprehensive safety features including merge-aware backups.`,
	RunE: runCleanup,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/tmp"
		}
	}
	defaultConfig := filepath.Join(homeDir, ".kubectx-manager_ignore")
	defaultKubeConfig := filepath.Join(homeDir, ".kube", "config")

	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Show what would be removed without making changes")
	rootCmd.Flags().BoolVarP(&authCheck, "auth-check", "a", false, "Remove contexts with expired or unreachable authentication")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose (debug) output")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all output except errors")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Prompt for confirmation before removing contexts")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", defaultConfig, "Path to kubectx-manager configuration file")
	rootCmd.Flags().StringVarP(&kubeConfig, "kubeconfig", "k", defaultKubeConfig, "Path to kubeconfig file")

	// Add subcommands
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(versionCmd)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log := logger.New(verbose, quiet)

	log.Debug("Starting kubectx-manager...")
	log.Debug("Config file: %s", configFile)
	log.Debug("Kubeconfig file: %s", kubeConfig)

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	log.Debug("Loaded configuration with %d whitelist patterns", len(cfg.Whitelist))

	// Load kubeconfig
	kConfig, err := kubeconfig.Load(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	log.Debug("Loaded kubeconfig with %d contexts", len(kConfig.Contexts))

	// Create backup before modifications
	if !dryRun {
		backupPath, err := kubeconfig.CreateBackup(kubeConfig)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		log.Info("Created backup at: %s", backupPath)
	}

	// Find contexts to remove
	contextsToRemove := findContextsToRemove(kConfig, cfg, log)

	if len(contextsToRemove) == 0 {
		log.Info("No contexts to remove")
		return nil
	}

	// Display what will be removed
	log.Info("Contexts to remove:")
	for _, ctx := range contextsToRemove {
		log.Info("  - %s", ctx)
	}

	if dryRun {
		log.Info("Dry run mode - no changes made")
		return nil
	}

	// Confirm with user if interactive mode is enabled
	if interactive {
		if !confirmRemoval(contextsToRemove) {
			log.Info("Operation canceled by user")
			return nil
		}
	}

	// Remove contexts and cleanup orphaned entries
	err = kubeconfig.RemoveContexts(kConfig, contextsToRemove)
	if err != nil {
		return fmt.Errorf("failed to remove contexts: %w", err)
	}

	// Save modified kubeconfig
	err = kubeconfig.Save(kConfig, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to save kubeconfig: %w", err)
	}

	log.Info("Successfully removed %d contexts", len(contextsToRemove))
	return nil
}

func findContextsToRemove(kConfig *kubeconfig.Config, cfg *config.Config, log *logger.Logger) []string {
	var toRemove []string

	for _, contextName := range kConfig.GetContextNames() {
		// Check if context matches whitelist patterns
		if cfg.MatchesWhitelist(contextName) {
			log.Debug("Context '%s' matches whitelist, keeping", contextName)
			continue
		}

		// If auth-check is enabled, check authentication status
		if authCheck {
			if kubeconfig.IsAuthValid(kConfig, contextName) {
				log.Debug("Context '%s' has valid auth, keeping", contextName)
				continue
			}
			log.Debug("Context '%s' has invalid auth, marking for removal", contextName)
		}

		toRemove = append(toRemove, contextName)
	}

	return toRemove
}

func confirmRemoval(contexts []string) bool {
	fmt.Printf("Are you sure you want to remove %d context(s)? (y/N): ", len(contexts))
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false
	}
	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}
