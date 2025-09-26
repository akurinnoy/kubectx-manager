# kubectx-manager

A smart CLI tool for managing Kubernetes contexts with pattern-based scoping. Features advanced pattern matching, authentication validation, cluster reachability checks, and comprehensive safety features including merge-aware backups.

## Features

✅ **Smart Context Filtering**

- Whitelist contexts using glob patterns in `~/.kubectx-manager_ignore`
- Remove contexts with expired/invalid authentication (`--auth-check`)
- Pattern matching with `*` (any characters) and `?` (single character)

✅ **Safety First**

- Automatic backups before any modifications
- Dry-run mode to preview changes (`--dry-run`)
- Optional interactive confirmation (`--interactive` for extra safety)
- Comprehensive error handling and validation

✅ **Clean & Thorough**

- Removes orphaned cluster and user entries
- Updates current-context if removed
- Multiple output modes: default, verbose, quiet

✅ **Zero Dependencies**

- No `kubectl` or other external tools required
- Pure Go implementation
- Single binary installation

✅ **Backup & Restore**

- Automatic timestamped backups before modifications
- Easy restore from any backup with `kubectx-manager restore`
- Interactive backup selection with confirmation

## Installation

### Quick Install (Recommended)

**One-line install script** (downloads latest release):

```bash
curl -fsSL https://raw.githubusercontent.com/che-incubator/kubectx-manager/main/install.sh | bash
```

Or download and inspect the script first:

```bash
curl -fsSL https://raw.githubusercontent.com/che-incubator/kubectx-manager/main/install.sh -o install.sh
chmod +x install.sh
./install.sh --help  # View options
./install.sh         # Run installation
```

The script will:

- 🔍 Auto-detect your platform (Linux/macOS, amd64/arm64)
- 📥 Download the latest release from GitHub
- 📦 Install to `/usr/local/bin` (or `~/bin` if no sudo permissions)
- ✅ Verify the installation

### Manual Installation from Source

```bash
# Clone and build
git clone https://github.com/che-incubator/kubectx-manager.git
cd kubectx-manager
go build -o kubectx-manager

# Install to user's bin directory
mkdir -p $HOME/bin
cp kubectx-manager $HOME/bin/

# Add to PATH (if not already added)
echo 'export PATH="$HOME/bin:$PATH"' >> $HOME/.zshrc
source $HOME/.zshrc

# Verify installation
kubectx-manager --help
```

### System-wide Installation

```bash
# Clone and build
git clone https://github.com/che-incubator/kubectx-manager.git
cd kubectx-manager
go build -o kubectx-manager

# Install system-wide (requires sudo)
sudo cp kubectx-manager /usr/local/bin/

# Verify installation
kubectx-manager --help
```

### Pre-built Binary Installation

1. **Download binary from [GitHub Releases](https://github.com/che-incubator/kubectx-manager/releases/latest)**
2. **Extract and install**:

   ```bash
   # Download appropriate archive for your platform from the link above
   tar -xzf kubectx-manager_*_*.tar.gz  # (or unzip for Windows)
   sudo mv kubectx-manager /usr/local/bin/
   kubectx-manager --help  # Verify installation
   ```

## Quick Start

1. **Create configuration file** (`~/.kubectx-manager_ignore`):

   ```dotfile
   # kubectx-manager ignore file (contexts to keep)
   # Contexts to keep (supports glob patterns)
   production-*
   staging-important
   my-dev-cluster
   *-permanent
   ```

2. **Preview what would be removed**:

   ```bash
   kubectx-manager --dry-run --verbose
   ```

3. **Clean up contexts**:

   ```bash
   kubectx-manager
   ```

## Usage Examples

### Basic Usage

```bash
# Remove all contexts except those in whitelist
kubectx-manager

# Preview changes without making them
kubectx-manager --dry-run

# Remove contexts with invalid authentication
kubectx-manager --auth-check

# Combine pattern matching and auth checking
kubectx-manager --auth-check --dry-run
```

### Output Control

```bash
# Verbose output with debug information
kubectx-manager --verbose

# Quiet mode (errors only)
kubectx-manager --quiet

# Enable interactive confirmation (optional)
kubectx-manager --interactive
```

### Custom Configuration

```bash
# Use custom config file
kubectx-manager --config /path/to/my-config

# Use custom kubeconfig file
kubectx-manager --kubeconfig /path/to/kubeconfig
```

## Configuration File Format

The `~/.kubectx-manager_ignore` file contains patterns for contexts to **keep** (whitelist). Each line represents a pattern:

```bash
# Comments start with #
production-*          # Keep all production contexts
staging-cluster       # Keep specific context
*-important           # Keep any context ending with "-important"  
my-dev-?-context      # Keep contexts like "my-dev-1-context"

# Empty lines are ignored
development-team-*
```

### Pattern Matching

- `*` - Matches any number of characters
- `?` - Matches exactly one character
- Patterns are case-sensitive
- Full context name must match (anchored matching)

## Command-Line Options

| Flag | Short | Description |
|------|-------|-------------|
| `--dry-run` | `-d` | Show what would be removed without making changes |
| `--auth-check` | `-a` | Remove contexts with expired/unreachable authentication |
| `--verbose` | `-v` | Enable verbose (debug) output |
| `--quiet` | `-q` | Suppress all output except errors |
| `--interactive` | `-i` | Prompt for confirmation before removing contexts |
| `--config` | `-c` | Path to configuration file (default: `~/.kubectx-manager_ignore`) |
| `--kubeconfig` | `-k` | Path to kubeconfig file (default: `~/.kube/config`) |

### Restore Command Options

| Flag | Description |
|------|-------------|
| `--no-backup` | Skip creating backup of current kubeconfig before restoring |
| `--keep-backup` | Keep backup file after successful restore (default: delete) |
| `--kubeconfig` `-k` | Path to kubeconfig file to restore |
| `--verbose` `-v` | Enable verbose (debug) output |
| `--quiet` `-q` | Suppress all output except errors |

### Backup Types

kubectx-manager creates different types of backups based on the situation:

| Backup Type | Filename Pattern | When Created | Contents |
|-------------|------------------|--------------|----------|
| **Standard Backup** | `config.backup.YYYYMMDD-HHMMSS` | During cleanup operations | Complete kubeconfig file |
| **Selective Backup** | `config.selective-backup.YYYYMMDD-HHMMSS` | When restoring with conflicts | Only conflicting contexts/clusters/users |
| **Manual Backup** | `config.backup.YYYYMMDD-HHMMSS` | User chooses full backup | Complete kubeconfig file |

## How It Works

1. **Load Configuration**: Reads whitelist patterns from `~/.kubectx-manager_ignore`
2. **Parse Kubeconfig**: Loads and validates your kubeconfig file
3. **Create Backup**: Automatically backs up kubeconfig before changes
4. **Filter Contexts**: Applies whitelist patterns and optional auth checking
5. **Clean Up**: Removes contexts and orphaned cluster/user entries (with optional confirmation)
6. **Save Changes**: Writes cleaned kubeconfig back to disk

## Safety Features

### Automatic Backups

Every modification creates a timestamped backup, providing safety without requiring confirmation prompts:

```bash
~/.kube/config.backup.20231124-143022
```

Since backups are automatic, kubectx-manager runs without prompts by default. Use `--interactive` if you want confirmation before changes.

### Dry Run Mode

Preview changes before applying:

```bash
$ kubectx-manager --dry-run --verbose
[DEBUG] Starting kubectx-manager...
[DEBUG] Loaded configuration with 3 whitelist patterns
[DEBUG] Loaded kubeconfig with 12 contexts
Contexts to remove:
  - old-cluster-context
  - expired-dev-context
  - unused-test-context
Dry run mode - no changes made
```

### Authentication Checking

The `--auth-check` flag identifies contexts with:

- Missing or invalid certificates
- Expired tokens
- Unreachable authentication commands
- Missing authentication providers

## Advanced Usage

### Combining Filters

```bash
# Remove contexts that don't match whitelist AND have invalid auth
kubectx-manager --auth-check

# This is an OR operation: remove if (not in whitelist) OR (invalid auth)
```

### Scripting

```bash
#!/bin/bash
# Automated cleanup script (no prompts by default)
kubectx-manager --quiet --auth-check
if [ $? -eq 0 ]; then
    echo "Cleanup completed successfully"
else
    echo "Cleanup failed" >&2
    exit 1
fi
```

### Multiple Kubeconfig Files

```bash
# Clean different kubeconfig files
kubectx-manager --kubeconfig ~/.kube/config-dev
kubectx-manager --kubeconfig ~/.kube/config-staging
kubectx-manager --kubeconfig ~/.kube/config-prod
```

## Backup & Restore

### Creating Backups

Backups are created automatically before any modifications:

```bash
kubectx-manager --dry-run  # No backup needed (no changes)
kubectx-manager            # Creates backup before cleaning
```

### Restoring from Backup

Use the restore command to recover from a backup:

```bash
# List and restore from available backups (smart backup handling)
kubectx-manager restore

# Skip backup creation entirely
kubectx-manager restore --no-backup

# Restore specific kubeconfig file
kubectx-manager restore --kubeconfig ~/.kube/config-dev

# Keep backup file after restore (don't delete it)
kubectx-manager restore --keep-backup

# Restore without any backup creation and keep original backup
kubectx-manager restore --no-backup --keep-backup
```

#### **Merge-Aware Backup Logic**

The restore command intelligently analyzes conflicts to avoid unnecessary backups:

- ✅ **Conflict detection** - Analyzes if backup contexts would overwrite existing ones
- ✅ **Smart skipping** - No backup needed when contexts can be safely merged
- ✅ **Selective backups** - Backs up only conflicting items instead of entire kubeconfig
- ✅ **User choice** - Options for no backup, selective backup, or full backup
- ✅ **Respects `--no-backup` flag** for complete control

The restore process:

1. **Lists available backups** (sorted by date, newest first)
2. **Interactive selection** - choose which backup to restore
3. **Conflict analysis** - checks if backup contexts would overwrite existing ones
4. **Smart backup decision** - no backup, selective backup, or full backup
5. **Confirmation prompt** - confirms the restore operation
6. **Restores the file** - replaces current kubeconfig with backup

### Example Restore Sessions

#### **No Conflicts (Safe Merge)**

```bash
$ kubectx-manager restore --verbose
[DEBUG] Starting kubeconfig restore...
Available backups:
  1. config.backup.20231124-143022 (2023-11-24 14:30:22)
Select backup: 1
Are you sure you want to continue? (y/N): y
[DEBUG] Found 0 potential conflicts: []
Skipping backup: no conflicts detected - backup contexts can be safely merged
Successfully restored kubeconfig from config.backup.20231124-143022
Removed backup file: config.backup.20231124-143022
```

#### **Conflicts Detected (User Choice)**

```bash
$ kubectx-manager restore
Available backups:
  1. config.backup.20231124-143022 (2023-11-24 14:30:22)
Select backup: 1
Are you sure you want to continue? (y/N): y
⚠️  Restoring this backup would overwrite 2 existing items:
  - context 'production-cluster' (different configuration)
  - user 'admin-user' (different credentials)

Backup options:
  1. No backup - proceed anyway (n)
  2. Selective backup - backup only conflicting items (s)
  3. Full backup - backup entire kubeconfig (f)
  4. Cancel restore (c)
Choose (n/s/f/c): s
Created selective backup of conflicting items: /Users/user/.kube/config.selective-backup.20231124-144501
Successfully restored kubeconfig from config.backup.20231124-143022
Removed backup file: config.backup.20231124-143022
```

#### **Power User (No Backup)**

```bash
$ kubectx-manager restore --no-backup
Available backups:
  1. config.backup.20231124-143022 (2023-11-24 14:30:22)
Select backup: 1
Are you sure you want to continue? (y/N): y
Skipping backup (--no-backup flag specified)
Successfully restored kubeconfig from config.backup.20231124-143022
Removed backup file: config.backup.20231124-143022
```

## Troubleshooting

### Common Issues

#### "Failed to load kubeconfig"

- Verify the kubeconfig file exists and is readable
- Check YAML syntax with `kubectl config view`

#### "No contexts to remove"

- All contexts match your whitelist patterns
- Use `--verbose` to see which patterns are matching

#### "Failed to create backup"

- Ensure write permissions to kubeconfig directory
- Check available disk space

### Verbose Debugging

```bash
kubectx-manager --dry-run --verbose
```

This shows:

- Configuration file location and patterns loaded
- Number of contexts found
- Pattern matching decisions for each context
- Authentication status (if `--auth-check` enabled)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

EPL-2.0 (Eclipse Public License 2.0) - see LICENSE file for details.

## Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/che-incubator/kubectx-manager/issues)
- 📖 **Documentation**: This README and `kubectx-manager --help`
