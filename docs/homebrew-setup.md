# Homebrew Setup and Release Workflow

This document explains how to set up and manage Homebrew distribution for kubectx-manager.

## Current Status

### Formula Location
- **File**: `Formula/kubectx-manager.rb`
- **Status**: ✅ Ready but disabled for initial releases
- **Version**: Correctly set to `v0.0.1`

### GoReleaser Configuration
- **File**: `.goreleaser.yml`
- **Homebrew Section**: 🔕 **Commented out** (disabled)
- **Reason**: Allows testing the release process before enabling automatic Homebrew updates

## Release Workflow

### Phase 1: Initial Release (v0.0.1)
**Recommended approach for first release:**

1. **Create GitHub Repository**
   ```bash
   # Create repo: github.com/akurinnoy/kubectx-manager
   git remote add origin https://github.com/akurinnoy/kubectx-manager.git
   git add .
   git commit -m "Initial kubectx-manager implementation"
   git push -u origin main
   ```

2. **Create First Release**
   ```bash
   git tag v0.0.1
   git push origin v0.0.1
   ```

3. **GoReleaser Builds Assets**
   - GitHub Actions automatically triggers
   - Builds binaries for all platforms
   - Creates GitHub release with assets
   - **Homebrew formula is NOT updated** (intentionally disabled)

4. **Manual Installation Available**
   Users can install manually from release assets:
   ```bash
   # Download appropriate binary from GitHub releases
   curl -L https://github.com/akurinnoy/kubectx-manager/releases/download/v0.0.1/kubectx-manager_0.0.1_darwin_arm64.tar.gz
   tar -xzf kubectx-manager_0.0.1_darwin_arm64.tar.gz
   sudo mv kubectx-manager /usr/local/bin/
   ```

### Phase 2: Enable Homebrew Distribution (Later)

**When ready to enable automatic Homebrew updates:**

#### Step 1: Update SHA256 Hash
Calculate the actual SHA256 hash from the release:

```bash
# Get SHA256 of the source tarball from v0.0.1 release
curl -sL https://github.com/akurinnoy/kubectx-manager/archive/v0.0.1.tar.gz | sha256sum

# Update Formula/kubectx-manager.rb with the real hash
# Replace: sha256 "0123456789abcdef..."
# With:    sha256 "actual_calculated_hash_here"
```

#### Step 2: Enable GoReleaser Homebrew Integration
Uncomment the `brews` section in `.goreleaser.yml`:

```yaml
# Homebrew tap configuration
brews:
  - # Repository to push to (same repo for simplicity)
    tap:
      owner: akurinnoy
      name: kubectx-manager
    
    # Formula name
    name: kubectx-manager
    
    # Formula description
    description: "Advanced Kubernetes context management tool"
    
    # Homepage
    homepage: "https://github.com/akurinnoy/kubectx-manager"
    
    # License
    license: "EPL-2.0"
    
    # Install section
    install: |
      bin.install "kubectx-manager"
    
    # Test section
    test: |
      system "#{bin}/kubectx-manager", "version"
```

#### Step 3: Test Homebrew Installation
After enabling, users can install via Homebrew:

```bash
# Add tap and install
brew tap akurinnoy/kubectx-manager
brew install kubectx-manager

# Or direct install
brew install akurinnoy/kubectx-manager/kubectx-manager
```

## File References

### Key Files
- **Formula**: `Formula/kubectx-manager.rb`
- **GoReleaser**: `.goreleaser.yml`
- **Release Workflow**: `.github/workflows/release.yml`

### Current Formula Content
```ruby
class KubectxManager < Formula
  desc "Advanced Kubernetes context management tool"
  homepage "https://github.com/akurinnoy/kubectx-manager"
  url "https://github.com/akurinnoy/kubectx-manager.git", :tag => "v0.0.1"
  sha256 "0123456789abcdef..."  # Update with real hash
  version "0.0.1"
  head "https://github.com/akurinnoy/kubectx-manager.git", :branch => "main"
  
  depends_on "go" => :build
  
  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end
  
  test do
    system "#{bin}/kubectx-manager", "--help"
  end
end
```

## Troubleshooting

### Common Issues

1. **SHA256 Mismatch**
   ```bash
   # Recalculate hash
   curl -sL https://github.com/akurinnoy/kubectx-manager/archive/v0.0.1.tar.gz | sha256sum
   ```

2. **Formula Not Found**
   ```bash
   # Ensure tap is added
   brew tap akurinnoy/kubectx-manager
   brew update
   ```

3. **Build Failures**
   ```bash
   # Check Go version and dependencies
   brew install go
   go mod tidy
   ```

## Best Practices

### Version Management
- Keep formula version in sync with Git tags
- Update SHA256 hash after each release
- Test formula locally before pushing:
  ```bash
  brew install --build-from-source ./Formula/kubectx-manager.rb
  ```

### Release Process
1. **Always test locally first**
2. **Use semantic versioning** (v0.0.1, v0.0.2, etc.)
3. **Update CHANGELOG.md** before releases
4. **Verify all platforms build** in GitHub Actions

## Future Enhancements

### Potential Improvements
- **Multiple Taps**: Consider separate tap repository for better organization
- **Bottle Support**: Pre-compiled binaries for faster installation
- **Cask Support**: If GUI version is developed
- **Homebrew Core**: Submit to main Homebrew repository when mature

---

**Status**: ✅ Ready for v0.0.1 release  
**Homebrew**: 🔕 Disabled (manual installation only)  
**Next Step**: Create GitHub repository and release v0.0.1
