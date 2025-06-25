# Contributing to go-tee-lib

Thank you for your interest in contributing to go-tee-lib! We welcome contributions from everyone.

## Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/go-tee-lib.git
   cd go-tee-lib
   ```
3. **Install Go** (version 1.21 or later)
4. **Install development tools**:
   ```bash
   go install golang.org/x/tools/cmd/goimports@latest
   go install honnef.co/go/tools/cmd/staticcheck@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

## Making Changes

1. **Create a new branch** for your feature or bugfix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards:
   - Write clear, readable code
   - Add tests for new functionality
   - Update documentation as needed
   - Follow Go conventions and idioms

3. **Run tests** to ensure everything works:
   ```bash
   go test -v -race ./...
   ```

4. **Run linting** to check code quality:
   ```bash
   golangci-lint run
   ```

5. **Format your code**:
   ```bash
   gofmt -s -w .
   goimports -w .
   ```

## Testing

- **Write tests** for all new functionality
- **Ensure high test coverage** - aim for 90%+
- **Test edge cases** and error conditions
- **Use table-driven tests** where appropriate
- **Test with race detector**: `go test -race ./...`

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

## Code Style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` and `goimports` to format code
- Write clear, descriptive commit messages
- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for exported functions and types

## Submitting Changes

1. **Push your changes** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a Pull Request** on GitHub with:
   - Clear description of changes
   - Reference to any related issues
   - Screenshots or examples if applicable

3. **Ensure CI passes** - all GitHub Actions workflows must pass

4. **Respond to feedback** from maintainers promptly

## Commit Message Format

Use clear, descriptive commit messages:

```
type(scope): brief description

Longer description if needed

Fixes #123
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

## Release Process

Releases are automated through GitHub Actions:

1. **Version bump** in appropriate files
2. **Create and push a tag**: `git tag v1.x.x && git push origin v1.x.x`
3. **GitHub Actions** will automatically create a release

## Getting Help

- **Open an issue** for bugs or feature requests
- **Start a discussion** for questions or ideas
- **Check existing issues** before creating new ones

## Code of Conduct

Be respectful and inclusive. We want everyone to feel welcome contributing to this project.

Thank you for contributing! 🎉
