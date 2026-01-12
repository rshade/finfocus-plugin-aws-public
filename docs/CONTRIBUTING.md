# Contributing to FinFocus AWS Public Plugin

Thank you for your interest in contributing to the FinFocus AWS Public Plugin!
This document provides guidelines and information for contributors.

## Ways to Contribute

### 🐛 Reporting Issues

- Use the [GitHub issue tracker](https://github.com/rshade/finfocus-plugin-aws-public/issues)
- Check existing issues to avoid duplicates
- Provide detailed reproduction steps and environment information

### 💡 Suggesting Features

- Open a [feature request](https://github.com/rshade/finfocus-plugin-aws-public/issues/new?template=feature_request.md)
- Describe the problem you're trying to solve
- Explain why this feature would be valuable

### 🛠️ Contributing Code

- Fork the repository
- Create a feature branch from `main`
- Make your changes following our coding standards
- Add tests for new functionality
- Submit a pull request

### 📚 Improving Documentation

- Documentation improvements are always welcome
- Check the [docs/](.) directory for existing documentation
- Follow markdown best practices and consistent formatting

## Development Setup

### Prerequisites

- Go 1.25.5 or later
- golangci-lint for linting
- goreleaser (optional, for releases)

### Getting Started

```bash
# Clone your fork
git clone https://github.com/your-username/finfocus-plugin-aws-public.git
cd finfocus-plugin-aws-public

# Install dependencies
go mod download

# Run tests
make test

# Run linter
make lint
```

### Building

```bash
# Build for development (fallback pricing)
make build

# Build for specific region (real pricing)
make build-region REGION=us-east-1
```

## Coding Standards

### Go Code

- Follow standard Go formatting (`gofmt`)
- Use `golangci-lint` for code quality checks
- Write comprehensive unit tests
- Document exported functions and types

### Commit Messages

Follow conventional commit format:

```text
type(scope): description

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### Pull Requests

- Use descriptive titles
- Reference related issues
- Keep changes focused and atomic
- Ensure CI checks pass
- Update documentation if needed

## Testing

### Running Tests

```bash
# Unit tests
make test

# Integration tests
go test -tags=integration ./internal/plugin/...

# Specific test
go test ./internal/plugin -run TestGetProjectedCost_EC2 -v
```

### Test Coverage

- Aim for high test coverage
- Test both success and error cases
- Use table-driven tests for multiple scenarios
- Mock external dependencies

## Documentation

### API Documentation

- Document all exported functions
- Include usage examples
- Explain parameters and return values
- Document error conditions

### User Documentation

- Keep README.md up to date
- Provide clear installation instructions
- Include troubleshooting guides
- Document breaking changes

## Release Process

This project uses automated releases:

1. **Development**: Make changes on feature branches
2. **Merge**: Merge to `main` with conventional commits
3. **Release**: release-please creates release PR automatically
4. **Publish**: goreleaser builds and publishes binaries

## Code of Conduct

This project follows a code of conduct to ensure a welcoming environment for all contributors.

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project (Apache 2.0).

## Getting Help

- 📖 [Documentation](README.md)
- 💬 [GitHub Discussions](https://github.com/rshade/finfocus-plugin-aws-public/discussions)
- 🐛 [Issue Tracker](https://github.com/rshade/finfocus-plugin-aws-public/issues)

Thank you for contributing to FinFocus! 🎉
