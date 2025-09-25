# GitHub Repository Setup Guide

## Branch Protection Rules

To ensure code quality and prevent regressions, set up the following branch protection rules:

### 1. Protect `main` branch

Go to **Settings** → **Branches** → **Add rule** for `main`:

**Required settings:**
- ✅ **Require a pull request before merging**
  - ✅ Require approvals: 1
  - ✅ Dismiss stale PR approvals when new commits are pushed
  - ✅ Require review from CODEOWNERS (now configured: @akurinnoy)
- ✅ **Require status checks to pass before merging**
  - ✅ Require branches to be up to date before merging
  - **Required status checks:**
    - `test`
- ✅ **Require conversation resolution before merging**
- ✅ **Restrict pushes that create files that exceed the file size limit**

**Optional settings:**
- ✅ **Require linear history** (optional, keeps git history clean)
- ✅ **Include administrators** (applies rules to repo admins too)


## Required Status Checks

The CI workflow provides these status checks:

### ✅ Required (Must Pass)
- **test** - Unit tests, build, cross-compile, and integration tests on Go 1.24

**The test job includes:**
- Unit tests (75+ tests) with race detection
- Test coverage validation (≥70%)
- Build verification
- Cross-compilation check
- Integration tests
- Security scanning (gosec)
- Code linting (golangci-lint)

### ⚠️ Optional (Can Fail)
- Security scanning and linting are included in the main test job but allowed to fail

## Workflow Files

- **`.github/workflows/test.yml`** - Main CI workflow (runs on push to main and PR to main)

## Setting Up Branch Protection

1. Go to your repository on GitHub
2. Click **Settings** tab
3. Click **Branches** in the left sidebar
4. Click **Add rule**
5. Enter `main` as the branch name pattern
6. Configure the settings as described above
7. Click **Create** to save the rule

## Testing the Setup

1. Create a feature branch: `git checkout -b test-pr-checks`
2. Make a small change (e.g., update README.md)
3. Push the branch: `git push -u origin test-pr-checks`
4. Create a Pull Request
5. Verify that the PR checks run automatically
6. Confirm that merge is blocked until checks pass

## Troubleshooting

### PR checks not running
- Ensure the workflow files are in `.github/workflows/`
- Check that the repository has Actions enabled
- Verify the branch name patterns in the workflow triggers

### Status checks not showing as required
- Make sure you've added the exact status check names to branch protection
- Status check names are case-sensitive
- The first run creates the status check names - you may need to update protection rules after the first PR

### Tests failing
- Check the Actions tab for detailed error logs
- Unit tests should be reliable; if they fail, there's likely a real issue
- Integration tests are marked as "can fail" temporarily
- Lint/security checks are informational and won't block PRs

## Recommended Setup

For a solo developer or small team:
```
Required status checks:
- pr-validation (1.21)
- pr-validation (1.22)

Required reviews: 0 (for solo) or 1 (for team)
Dismiss stale reviews: Yes
Require conversation resolution: Yes
```

For larger teams:
```
Required status checks:
- pr-validation (1.21) 
- pr-validation (1.22)

Required reviews: 1-2
Dismiss stale reviews: Yes
Require review from CODEOWNERS: Yes
Require conversation resolution: Yes
Include administrators: Yes
```
