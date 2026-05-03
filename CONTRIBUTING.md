# Contributing to go-task-offloader

Thank you for taking the time to contribute! This document explains how to get
set up, the workflow for submitting changes, and the standards we follow.

---

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)

---

## Getting Started

### Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.22+ | Build & test |
| Redis | 7+ | Integration target |
| Docker | 24+ | Container builds |
| golangci-lint | latest | Linting |

### Local setup

```bash
# 1. Fork & clone
git clone https://github.com/<your-fork>/go-task-offloader.git
cd go-task-offloader

# 2. Install dependencies
go mod tidy

# 3. Copy env config
cp .env.example .env

# 4. Run tests
make test

# 5. Build
make build
```

---

## Development Workflow

```
main  ←  feature/your-feature-name
      ←  fix/short-description
      ←  docs/update-readme
```

1. **Create a branch** from `main`:

   ```bash
   git checkout -b feature/streaming-json-handler
   ```

2. **Make your changes.** Keep commits small and focused.

3. **Run the full check suite** before pushing:

   ```bash
   make fmt vet lint test
   ```

4. **Push and open a PR** against `main`.

---

## Coding Standards

- Format with `gofmt` / `goimports` before committing (`make fmt`).
- All exported symbols must have a doc comment.
- Avoid global state; pass dependencies explicitly (constructor injection).
- New handlers must satisfy the `handlers.Handler` interface.
- Do not add runtime dependencies without prior discussion in an issue.

---

## Commit Messages

We follow **Conventional Commits** (`type(scope): description`):

| Type | When to use |
|---|---|
| `feat` | New feature or handler |
| `fix` | Bug fix |
| `perf` | Performance improvement |
| `refactor` | Code change with no behaviour change |
| `test` | Adding or fixing tests |
| `docs` | Documentation only |
| `chore` | Build tooling, CI, deps |

**Examples:**

```
feat(handlers): add PDF generation handler
fix(queue): handle Redis Nil error on graceful shutdown
perf(cruncher): use bufio writer in filterCSV
docs: update libvips activation steps in README
```

---

## Pull Request Process

1. Fill out the PR template completely.
2. Reference the related issue (`Closes #42`).
3. Ensure CI passes (lint + build + tests).
4. Request a review from a maintainer.
5. PRs are squash-merged into `main`.

---

## Reporting Bugs

Use the **Bug Report** issue template. Include:
- Go version (`go version`)
- Redis version
- Exact error message / stack trace
- Minimal reproduction steps

---

## Suggesting Features

Use the **Feature Request** issue template. Describe:
- The problem you're trying to solve
- Your proposed solution
- Alternatives you considered
