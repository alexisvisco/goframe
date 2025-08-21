# Pre-commit Hook for GoFrame

This document explains the pre-commit hook setup for the GoFrame project.

## What's Currently Set Up

A pre-commit hook has been installed at `.git/hooks/pre-commit` that automatically runs `go work sync` before each commit.

## What the Hook Does

1. **Checks for Go workspace** - Ensures `go.work` file exists
2. **Runs `go work sync`** - Synchronizes all module dependencies
3. **Auto-adds modified files** - If `go work sync` changes any `go.mod` or `go.sum` files, they're automatically added to the commit

## How It Works

Every time you run `git commit`, the hook will:

```
🔄 Running pre-commit checks...
📦 Synchronizing workspace modules...
📝 go work sync modified go.mod files. Adding them to the commit...
✅ All pre-commit checks passed!
```

## Enhanced Hook (Optional)

For more comprehensive checks, you can replace the current hook with an enhanced version:

```bash
#!/bin/sh
#
# GoFrame Enhanced Pre-commit Hook
# This hook runs comprehensive checks before each commit

echo "🔄 Running enhanced pre-commit checks..."

# Check if we're in a Go workspace
if [ ! -f "go.work" ]; then
    echo "❌ Error: go.work file not found. This hook requires a Go workspace."
    exit 1
fi

# Run go work sync to ensure all modules are synchronized
echo "📦 Synchronizing workspace modules..."
go work sync
if [ $? -ne 0 ]; then
    echo "❌ Error: go work sync failed"
    exit 1
fi

# Check if go work sync made any changes
if ! git diff --quiet go.mod */go.mod */*/go.mod 2>/dev/null; then
    echo "📝 go work sync modified go.mod files. Adding them to the commit..."
    git add go.mod */go.mod */*/go.mod 2>/dev/null || true
    git add go.sum */go.sum */*/go.sum 2>/dev/null || true
fi

# Run go mod tidy on all modules
echo "🧹 Running go mod tidy on all modules..."
for dir in $(find . -name "go.mod" -exec dirname {} \;); do
    echo "  Tidying $dir"
    (cd "$dir" && go mod tidy)
    if [ $? -ne 0 ]; then
        echo "❌ Error: go mod tidy failed in $dir"
        exit 1
    fi
done

# Check code formatting
echo "🎨 Checking code formatting..."
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    echo "❌ Error: The following files are not properly formatted:"
    echo "$unformatted"
    echo "Run 'gofmt -w .' to fix formatting issues."
    exit 1
fi

# Run go vet
echo "🔍 Running go vet..."
go vet ./...
if [ $? -ne 0 ]; then
    echo "❌ Error: go vet found issues"
    exit 1
fi

# Run tests
echo "🧪 Running tests..."
go test ./...
if [ $? -ne 0 ]; then
    echo "❌ Error: Tests failed"
    exit 1
fi

echo "✅ All enhanced pre-commit checks passed!"
```

To use the enhanced version:

```bash
# Save the above script to .git/hooks/pre-commit and make it executable
chmod +x .git/hooks/pre-commit
```

## Managing the Hook

### Skip the hook temporarily
```bash
git commit --no-verify -m "your message"
```

### Test the hook manually
```bash
.git/hooks/pre-commit
```

### Disable the hook
```bash
mv .git/hooks/pre-commit .git/hooks/pre-commit.disabled
```

### Re-enable the hook
```bash
mv .git/hooks/pre-commit.disabled .git/hooks/pre-commit
```

## What Each Check Does

- **`go work sync`** - Ensures all modules have consistent dependencies
- **`go mod tidy`** (enhanced only) - Cleans up unused dependencies
- **`gofmt`** (enhanced only) - Checks code formatting
- **`go vet`** (enhanced only) - Static analysis for common errors
- **`go test`** (enhanced only) - Runs all tests

## Benefits

1. **Prevents broken builds** - Dependencies are always synchronized
2. **Maintains consistency** - All team members have the same dependency versions
3. **Automatic cleanup** - No need to remember to run `go work sync`
4. **Quality assurance** - Enhanced version ensures code quality standards

## Troubleshooting

### Hook fails with "go.work file not found"
Make sure you're committing from the root directory of the project.

### Hook is slow
The enhanced version runs tests and other checks. Use the basic version for faster commits, or skip occasionally with `--no-verify`.

### Want to customize
Edit `.git/hooks/pre-commit` directly to add or remove checks as needed.