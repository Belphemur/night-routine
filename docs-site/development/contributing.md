# Contributing

Thank you for your interest in contributing to Night Routine Scheduler! This guide will help you get started.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). By participating, you are expected to uphold this code.

## Ways to Contribute

- 🐛 Report bugs
- 💡 Suggest new features
- 📝 Improve documentation
- ✨ Submit code changes
- 🔍 Review pull requests
- ❓ Answer questions in issues/discussions

## Getting Started

### 1. Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/night-routine.git
cd night-routine

# Add upstream remote
git remote add upstream https://github.com/Belphemur/night-routine.git
```

### 2. Set Up Development Environment

Follow the [Local Development Guide](local.md) to set up your environment.

### 3. Create a Branch

```bash
# Fetch latest changes
git fetch upstream
git checkout main
git merge upstream/main

# Create a feature branch
git checkout -b feature/my-feature
```

**Branch naming conventions:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions/changes

## Reporting Bugs

### Before Reporting

1. Check if the bug has already been reported in [Issues](https://github.com/Belphemur/night-routine/issues)
2. Try to reproduce the bug with the latest version
3. Gather relevant information (logs, configuration, steps to reproduce)

### Bug Report Template

```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Configure with '...'
2. Click on '....'
3. See error

**Expected behavior**
What you expected to happen.

**Screenshots**
If applicable, add screenshots.

**Environment:**
- OS: [e.g., Ubuntu 22.04]
- Deployment: [Docker/Binary/Docker Compose]
- Version: [e.g., v1.2.0]

**Configuration:**
```toml
[Relevant config sections with sensitive data removed]
```

**Logs:**
```
[Relevant log output]
```

**Additional context**
Any other context about the problem.
```

## Suggesting Features

### Feature Request Template

```markdown
**Is your feature request related to a problem?**
A clear description of the problem. Ex. I'm always frustrated when [...]

**Describe the solution you'd like**
A clear description of what you want to happen.

**Describe alternatives you've considered**
Other solutions you've thought about.

**Additional context**
Any other context, screenshots, or examples.
```

## Code Contributions

### Development Workflow

1. **Create a branch** from `main`
2. **Make your changes**
3. **Write/update tests**
4. **Run linters and tests**
5. **Commit with conventional commits**
6. **Push to your fork**
7. **Open a pull request**

### Coding Standards

#### Go Style Guide

Follow [Effective Go](https://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**Key points:**
- Use `gofmt` for formatting
- Keep functions small and focused
- Write clear variable names
- Add comments for exported functions
- Handle errors explicitly

**Example:**
```go
// CalculateFairness determines which parent should be assigned
// based on historical assignments and configured availability.
// It returns the chosen parent and the decision reason.
func CalculateFairness(config Config, date time.Time) (string, Reason, error) {
    if err := validateConfig(config); err != nil {
        return "", "", fmt.Errorf("invalid config: %w", err)
    }
    
    // Implementation...
    return parent, reason, nil
}
```

#### Code Organization

```
internal/
├── package/              # Package directory
│   ├── package.go       # Main package logic
│   ├── package_test.go  # Tests
│   ├── types.go         # Type definitions (if many types)
│   └── README.md        # Package documentation (for complex packages)
```

#### Linting

Pass `golangci-lint` checks:

```bash
golangci-lint run
```

**Fix automatically when possible:**
```bash
golangci-lint run --fix
```

### Testing Requirements

#### Test Coverage

- All new code must include tests
- Maintain or improve overall coverage
- Aim for 70%+ coverage on new code

**Run tests:**
```bash
go test -v -cover ./...
```

#### Test Quality

- Use table-driven tests for multiple scenarios
- Test edge cases and error conditions
- Use meaningful test names
- Keep tests independent

**Example:**
```go
func TestParentAssignment(t *testing.T) {
    tests := []struct {
        name           string
        availability   Availability
        date           time.Time
        expectedParent string
        expectedReason Reason
    }{
        {
            name: "parent A unavailable on Monday",
            availability: Availability{
                ParentAUnavailable: []string{"Monday"},
            },
            date:           parseDate("2024-01-01"), // Monday
            expectedParent: "ParentB",
            expectedReason: Unavailability,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parent, reason, err := AssignParent(tt.availability, tt.date)
            require.NoError(t, err)
            assert.Equal(t, tt.expectedParent, parent)
            assert.Equal(t, tt.expectedReason, reason)
        })
    }
}
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/) specification.

**Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

**Examples:**

```bash
# Simple feature
feat: add monthly statistics export

# Bug fix with details
fix(fairness): correct consecutive limit calculation

Previously, the consecutive limit was not properly reset
when a parent became unavailable, leading to incorrect
assignments.

Fixes #123

# Breaking change
feat(config)!: change TOML structure for parent settings

BREAKING CHANGE: Parent configuration now uses nested
structure instead of flat fields. Update your config:

Before:
  parent_a = "Alice"
  
After:
  [parents]
  parent_a = "Alice"
```

### Pull Request Process

#### Before Submitting

- [ ] Tests pass locally
- [ ] Linters pass
- [ ] Code is formatted (`gofmt -s -w .`)
- [ ] Documentation updated (if needed)
- [ ] CHANGELOG.md updated (for notable changes)
- [ ] Commit messages follow conventions

#### Pull Request Template

```markdown
## Description
Brief description of what this PR does.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Related Issue
Fixes #(issue number)

## Changes Made
- Change 1
- Change 2
- Change 3

## Testing
Describe how you tested your changes.

## Screenshots (if applicable)
Add screenshots for UI changes.

## Checklist
- [ ] Tests pass locally
- [ ] Linters pass (`golangci-lint run`)
- [ ] Code formatted (`gofmt -s -l .`)
- [ ] Added tests for new features
- [ ] Updated documentation
- [ ] Updated CHANGELOG.md
- [ ] Followed commit message conventions
```

#### Review Process

1. **Automated checks** run (linting, tests, build)
2. **Maintainers review** code
3. **Address feedback** if requested
4. **CI passes** all checks
5. **Merge** by maintainer

**Timeline:**
- Initial response: Within 1-3 days
- Full review: Within 1 week
- Be patient and responsive to feedback

## Documentation Contributions

### Documentation Structure

Documentation is written in Markdown and built with MkDocs Material.

```
docs-site/
├── index.md
├── features.md
├── installation/
├── configuration/
├── user-guide/
├── architecture/
├── development/
└── api-reference.md
```

### Writing Documentation

**Guidelines:**
- Use clear, concise language
- Include code examples
- Add screenshots for UI features
- Use admonitions for important notes
- Link to related pages

**Example admonitions:**
```markdown
!!! note "Configuration Note"
    This setting requires a restart.

!!! warning "Breaking Change"
    This feature changes the API.

!!! tip "Pro Tip"
    Use this for better performance.

!!! danger "Security Warning"
    Never commit credentials.
```

### Building Documentation Locally

```bash
# Install MkDocs and dependencies
pip install mkdocs-material

# Serve documentation locally
mkdocs serve

# Open http://localhost:8000
```

## Release Process

Releases are automated via semantic-release and GitHub Actions.

### Semantic Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for new functionality (backward compatible)
- **PATCH** version for bug fixes (backward compatible)

### Release Workflow

1. Commits to `main` trigger semantic-release
2. Semantic-release analyzes commit messages
3. Version is determined automatically
4. CHANGELOG is updated
5. Git tag is created
6. GitHub Release is created
7. Binaries and Docker images are built
8. Documentation is published

### Commit Message Impact

| Commit Type | Release Type |
|------------|--------------|
| `fix:` | PATCH (0.0.x) |
| `feat:` | MINOR (0.x.0) |
| `BREAKING CHANGE:` | MAJOR (x.0.0) |
| `docs:`, `test:`, etc. | No release |

## Community

### Getting Help

- **Issues**: For bugs and feature requests
- **Discussions**: For questions and ideas
- **Discord/Slack**: (If available) For real-time chat

### Recognition

Contributors are recognized in:
- GitHub contributors list
- Release notes (for significant contributions)
- CHANGELOG.md

## License

By contributing, you agree that your contributions will be licensed under the AGPLv3 License.

## Questions?

If you have questions about contributing:
- Open an issue with the `question` label
- Ask in Discussions
- Reach out to maintainers

Thank you for contributing to Night Routine Scheduler! 🎉
