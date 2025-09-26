class KubectxManager < Formula
  desc "Advanced Kubernetes context management tool"
  homepage "https://github.com/che-incubator/kubectx-manager"
  url "https://github.com/che-incubator/kubectx-manager.git", :tag => "v0.0.1"
  sha256 "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"  # Update when publishing
  version "0.0.1"
  head "https://github.com/che-incubator/kubectx-manager.git", :branch => "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  def caveats
    <<~EOS
      kubectx-manager has been installed!

      Configuration:
      Create a ~/.kubectx-manager_ignore file with context patterns to keep (whitelist).
      The file supports glob patterns (* and ?) for flexible matching.

      Usage examples:
        kubectx-manager --dry-run          # Preview what would be removed
        kubectx-manager --auth-check       # Remove contexts with invalid auth
        kubectx-manager --verbose          # Enable debug output
        kubectx-manager --quiet            # Suppress all output except errors
        kubectx-manager restore            # Restore from backup interactively

      Configuration file example (~/.kubectx-manager_ignore):
        production-*
        staging-important
        my-dev-context
        *-permanent

      The tool will:
      - Create automatic backups before making changes
      - Remove orphaned clusters and users
      - Run without prompts by default (use --interactive for confirmation)
      - Support both pattern matching and auth status filtering
      - Allow easy restoration from any backup
    EOS
  end

  test do
    system "#{bin}/kubectx-manager", "--help"
  end
end
