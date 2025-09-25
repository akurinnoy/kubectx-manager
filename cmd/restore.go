// Package cmd provides command line interface commands for kubectx-manager.
// It includes commands for restoring kubernetes contexts from backup files.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/akurinnoy/kubectx-manager/internal/kubeconfig"
	"github.com/akurinnoy/kubectx-manager/internal/logger"
)

const (
	// BackupTimeFormat is the timestamp format used for backup file names
	BackupTimeFormat = "20060102-150405"

	// User choice constants
	choiceNone      = "none"
	choiceSelective = "selective"
	choiceFull      = "full"
	choiceCancel    = "cancel"
)

var (
	noBackup   bool
	keepBackup bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore kubeconfig from a backup",
	Long: `Restore your kubeconfig file from a previously created backup.
Lists available backups and allows you to select one to restore.
Intelligently handles backup creation to avoid redundant backups.`,
	RunE: runRestore,
}

func init() { //nolint:gochecknoinits // Cobra CLI flag setup requires init
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose (debug) output")
	restoreCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all output except errors")
	restoreCmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backup of current kubeconfig before restoring")
	restoreCmd.Flags().BoolVar(&keepBackup, "keep-backup", false, "Keep backup file after successful restore (default: delete)")
	restoreCmd.Flags().StringVarP(&kubeConfig, "kubeconfig", "k", "", "Path to kubeconfig file to restore")
}

func runRestore(_ *cobra.Command, _ []string) error {
	// Initialize logger
	log := logger.New(verbose, quiet)

	// Set default kubeconfig if not provided
	if kubeConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = os.Getenv("HOME")
			if homeDir == "" {
				homeDir = "/tmp"
			}
		}
		kubeConfig = filepath.Join(homeDir, ".kube", "config")
	}

	log.Debugf("Starting kubeconfig restore...")
	log.Debugf("Kubeconfig file: %s", kubeConfig)

	// Find available backups
	backups, err := findBackups(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to find backups: %w", err)
	}

	if len(backups) == 0 {
		log.Infof("No backups found for %s", kubeConfig)
		return nil
	}

	// Display available backups
	log.Infof("Available backups:")
	for i, backup := range backups {
		log.Infof("  %d. %s (%s)", i+1, backup.Name, backup.TimeStr)
	}

	// Get user selection
	selection, err := getUserSelection(len(backups))
	if err != nil {
		return err
	}

	if selection == 0 {
		log.Infof("Restore canceled")
		return nil
	}

	selectedBackup := backups[selection-1]
	log.Infof("Selected backup: %s", selectedBackup.Name)

	// Confirm restore
	if !confirmRestore(selectedBackup.Name, kubeConfig) {
		log.Infof("Restore canceled")
		return nil
	}

	// Smart backup handling
	if !noBackup {
		shouldCreateBackup, reason, conflicts := shouldCreateBackupBeforeRestore(kubeConfig, backups, selectedBackup, log)
		if shouldCreateBackup {
			log.Debugf("Creating backup: %s", reason)

			if len(conflicts) > 0 {
				// Create selective backup
				currentBackupPath, err := createSelectiveBackup(kubeConfig, conflicts, log)
				if err != nil {
					return fmt.Errorf("failed to create selective backup: %w", err)
				}
				log.Infof("Created selective backup of conflicting items: %s", currentBackupPath)
			} else {
				// Create full backup
				currentBackupPath, err := kubeconfig.CreateBackup(kubeConfig)
				if err != nil {
					return fmt.Errorf("failed to backup current kubeconfig: %w", err)
				}
				log.Infof("Created full backup of current kubeconfig: %s", currentBackupPath)
			}
		} else {
			log.Infof("Skipping backup: %s", reason)
		}
	} else {
		log.Infof("Skipping backup (--no-backup flag specified)")
	}

	// Restore from backup
	err = restoreFromBackup(selectedBackup.Path, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	log.Infof("Successfully restored kubeconfig from %s", selectedBackup.Name)

	// Clean up backup file after successful restore (unless --keep-backup flag is used)
	if !keepBackup {
		log.Debugf("Cleaning up backup file: %s", selectedBackup.Path)
		err = os.Remove(selectedBackup.Path)
		if err != nil {
			log.Warnf("Failed to remove backup file %s: %v", selectedBackup.Path, err)
			log.Warnf("You may want to manually remove it")
		} else {
			log.Infof("Removed backup file: %s", selectedBackup.Name)
		}
	} else {
		log.Infof("Backup file preserved: %s", selectedBackup.Name)
	}

	return nil
}

// Backup represents a kubeconfig backup file with metadata about when it was created.
// It contains the file path, display name, and timestamp information for restore operations.
type Backup struct {
	Name    string
	Path    string
	Time    time.Time
	TimeStr string
}

func findBackups(kubeconfigPath string) ([]Backup, error) {
	dir := filepath.Dir(kubeconfigPath)
	baseName := filepath.Base(kubeconfigPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backups []Backup
	prefix := baseName + ".backup."

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		backupPath := filepath.Join(dir, entry.Name())

		// Extract timestamp from filename
		timestampStr := strings.TrimPrefix(entry.Name(), prefix)
		timestamp, err := time.Parse(BackupTimeFormat, timestampStr)
		if err != nil {
			continue // Skip files that don't match our backup format
		}

		backup := Backup{
			Name:    entry.Name(),
			Path:    backupPath,
			Time:    timestamp,
			TimeStr: timestamp.Format("2006-01-02 15:04:05"),
		}
		backups = append(backups, backup)
	}

	// Sort backups by time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Time.After(backups[j].Time)
	})

	return backups, nil
}

func getUserSelection(maxOptions int) (int, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("Select backup to restore (1-%d, or 0 to cancel): ", maxOptions)
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}

		input = strings.TrimSpace(input)
		selection, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Please enter a valid number")
			continue
		}

		if selection == 0 {
			return 0, nil
		}

		if selection < 1 || selection > maxOptions {
			fmt.Printf("Please enter a number between 1 and %d (or 0 to cancel)\n", maxOptions)
			continue
		}

		return selection, nil
	}
}

func confirmRestore(backupName, kubeconfigPath string) bool {
	fmt.Printf("This will restore %s from backup %s.\n", kubeconfigPath, backupName)
	fmt.Printf("Are you sure you want to continue? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes"
}

func shouldCreateBackupBeforeRestore(kubeconfigPath string, _ []Backup, selectedBackup Backup, log *logger.Logger) (shouldBackup bool, reason string, conflicts []string) {
	// Load current kubeconfig
	currentConfig, err := kubeconfig.Load(kubeconfigPath)
	if err != nil {
		log.Debugf("Could not load current kubeconfig: %v", err)
		return true, "could not load current kubeconfig for analysis", nil
	}

	// Load backup kubeconfig
	backupConfig, err := kubeconfig.Load(selectedBackup.Path)
	if err != nil {
		log.Debugf("Could not load backup kubeconfig: %v", err)
		return true, "could not load backup kubeconfig for analysis", nil
	}

	// Analyze merge conflicts
	conflicts = analyzeRestoreConflicts(currentConfig, backupConfig, log)

	if len(conflicts) == 0 {
		return false, "no conflicts detected - backup contexts can be safely merged", nil
	}

	log.Debugf("Found %d potential conflicts: %v", len(conflicts), conflicts)

	// Ask user if they want selective backup or full backup
	choice := askUserAboutConflicts(conflicts)
	switch choice {
	case choiceNone:
		return false, "user chose to proceed without backup", nil
	case choiceSelective:
		return true, "user chose selective backup of conflicting contexts", conflicts
	case choiceFull:
		return true, "user chose full backup", nil
	default:
		return false, "restore canceled by user", nil
	}
}

func analyzeRestoreConflicts(current, backup *kubeconfig.Config, log *logger.Logger) []string {
	var conflicts []string

	// Check context conflicts
	for _, backupContext := range backup.Contexts {
		if currentContext := current.GetContext(backupContext.Name); currentContext != nil {
			// Context exists in both - check if they're different
			if !contextsEqual(currentContext, backupContext.Context) {
				conflicts = append(conflicts, fmt.Sprintf("context '%s' (different configuration)", backupContext.Name))
				log.Debugf("Context conflict: %s", backupContext.Name)
			}
		}
	}

	// Check cluster conflicts
	currentClusters := make(map[string]*kubeconfig.Cluster)
	for _, cluster := range current.Clusters {
		currentClusters[cluster.Name] = cluster.Cluster
	}

	for _, backupCluster := range backup.Clusters {
		if currentCluster, exists := currentClusters[backupCluster.Name]; exists {
			if !clustersEqual(currentCluster, backupCluster.Cluster) {
				conflicts = append(conflicts, fmt.Sprintf("cluster '%s' (different server/auth)", backupCluster.Name))
				log.Debugf("Cluster conflict: %s", backupCluster.Name)
			}
		}
	}

	// Check user conflicts
	currentUsers := make(map[string]*kubeconfig.User)
	for _, user := range current.Users {
		currentUsers[user.Name] = user.User
	}

	for _, backupUser := range backup.Users {
		if currentUser, exists := currentUsers[backupUser.Name]; exists {
			if !usersEqual(currentUser, backupUser.User) {
				conflicts = append(conflicts, fmt.Sprintf("user '%s' (different credentials)", backupUser.Name))
				log.Debugf("User conflict: %s", backupUser.Name)
			}
		}
	}

	return conflicts
}

func contextsEqual(a, b *kubeconfig.Context) bool {
	return a.Cluster == b.Cluster && a.User == b.User && a.Namespace == b.Namespace
}

func clustersEqual(a, b *kubeconfig.Cluster) bool {
	return a.Server == b.Server &&
		a.CertificateAuthorityData == b.CertificateAuthorityData &&
		a.CertificateAuthority == b.CertificateAuthority &&
		a.InsecureSkipTLSVerify == b.InsecureSkipTLSVerify
}

func usersEqual(a, b *kubeconfig.User) bool {
	return a.ClientCertificateData == b.ClientCertificateData &&
		a.ClientKeyData == b.ClientKeyData &&
		a.ClientCertificate == b.ClientCertificate &&
		a.ClientKey == b.ClientKey &&
		a.Token == b.Token &&
		a.Username == b.Username &&
		a.Password == b.Password
}

func askUserAboutConflicts(conflicts []string) string {
	fmt.Printf("⚠️  Restoring this backup would overwrite %d existing items:\n", len(conflicts))
	for _, conflict := range conflicts {
		fmt.Printf("  - %s\n", conflict)
	}
	fmt.Println()
	fmt.Println("Backup options:")
	fmt.Println("  1. No backup - proceed anyway (n)")
	fmt.Println("  2. Selective backup - backup only conflicting items (s)")
	fmt.Println("  3. Full backup - backup entire kubeconfig (f)")
	fmt.Println("  4. Cancel restore (c)")
	fmt.Printf("Choose (n/s/f/c): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return choiceCancel
	}
	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "n", "no":
		return choiceNone
	case "s", "selective":
		return choiceSelective
	case "f", "full":
		return choiceFull
	case "c", choiceCancel:
		return choiceCancel
	default:
		fmt.Printf("Invalid choice '%s', defaulting to cancel\n", response)
		return choiceCancel
	}
}

func createSelectiveBackup(kubeconfigPath string, conflicts []string, log *logger.Logger) (string, error) {
	// Load current kubeconfig
	currentConfig, err := kubeconfig.Load(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to load current kubeconfig: %w", err)
	}

	// Create a minimal config with only conflicting items
	selectiveConfig := &kubeconfig.Config{
		APIVersion: currentConfig.APIVersion,
		Kind:       currentConfig.Kind,
		Contexts:   []kubeconfig.NamedContext{},
		Clusters:   []kubeconfig.NamedCluster{},
		Users:      []kubeconfig.NamedUser{},
	}

	// Extract context names from conflicts
	conflictingContexts := make(map[string]bool)
	conflictingClusters := make(map[string]bool)
	conflictingUsers := make(map[string]bool)

	for _, conflict := range conflicts {
		if strings.Contains(conflict, "context '") {
			name := extractNameFromConflict(conflict, "context")
			conflictingContexts[name] = true

			// Also include related cluster and user
			if ctx := currentConfig.GetContext(name); ctx != nil {
				conflictingClusters[ctx.Cluster] = true
				conflictingUsers[ctx.User] = true
			}
		} else if strings.Contains(conflict, "cluster '") {
			name := extractNameFromConflict(conflict, "cluster")
			conflictingClusters[name] = true
		} else if strings.Contains(conflict, "user '") {
			name := extractNameFromConflict(conflict, "user")
			conflictingUsers[name] = true
		}
	}

	// Add conflicting contexts
	for _, namedContext := range currentConfig.Contexts {
		if conflictingContexts[namedContext.Name] {
			selectiveConfig.Contexts = append(selectiveConfig.Contexts, namedContext)
		}
	}

	// Add conflicting clusters
	for _, namedCluster := range currentConfig.Clusters {
		if conflictingClusters[namedCluster.Name] {
			selectiveConfig.Clusters = append(selectiveConfig.Clusters, namedCluster)
		}
	}

	// Add conflicting users
	for _, namedUser := range currentConfig.Users {
		if conflictingUsers[namedUser.Name] {
			selectiveConfig.Users = append(selectiveConfig.Users, namedUser)
		}
	}

	// Create backup filename
	timestamp := time.Now().Format(BackupTimeFormat)
	backupPath := kubeconfigPath + ".selective-backup." + timestamp

	// Save selective backup
	err = kubeconfig.Save(selectiveConfig, backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to save selective backup: %w", err)
	}

	log.Debugf("Created selective backup with %d contexts, %d clusters, %d users",
		len(selectiveConfig.Contexts), len(selectiveConfig.Clusters), len(selectiveConfig.Users))

	return backupPath, nil
}

func extractNameFromConflict(conflict, itemType string) string {
	// Extract name from conflict string like "context 'my-context' (different configuration)"
	start := strings.Index(conflict, itemType+" '")
	if start == -1 {
		return ""
	}
	start += len(itemType + " '")

	end := strings.Index(conflict[start:], "'")
	if end == -1 {
		return ""
	}

	return conflict[start : start+end]
}

func restoreFromBackup(backupPath, kubeconfigPath string) error {
	// Read backup file
	data, err := os.ReadFile(backupPath) //nolint:gosec // User-selected backup file path is intentional
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write to kubeconfig
	err = os.WriteFile(kubeconfigPath, data, 0600) //nolint:mnd // Use 0600 for security (kubeconfig contains credentials)
	if err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return nil
}
