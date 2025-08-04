# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Plan & Review Workflow

### Before Starting Work

* Perform comprehensive project analysis if not already done
* Consult README.md for current project context and standards
* Write detailed implementation plans to `.claude/tasks/TASK_NAME.md`
* Include specific reasoning for architectural decisions and task breakdown
* Follow MVP principles - avoid over-engineering and focus on core requirements
* Present the plan for review and await approval before proceeding
* Use step-by-step reasoning for complex planning decisions

### During Implementation

* Reference README.md continuously for project-specific guidance
* Update the plan with progress and detailed change descriptions
* Document all modifications for knowledge transfer to other engineers
* Mark tasks as completed only when fully validated and tested
* Use chain-of-thought reasoning for complex problem-solving
* Maintain consistency with discovered project patterns and conventions

### Quality Gates

* Discover and use project-specific quality commands (e.g., lint, test, format)
* If no specific commands found, suggest appropriate tools for the technology stack
* Always validate changes against project's established quality standards

## Project Overview
Rodwer is a Go-based browser automation and testing project that uses the Rod library for web browser control and testing. The main functionality includes JavaScript and Go code coverage collection through automated browser tests.

## Technology Stack
- **Language**: Go 1.24.1
- **Testing Framework**: Rod (github.com/go-rod/rod) for browser automation
- **Test Assertions**: testify (github.com/stretchr/testify)

## Commands

### Fast Development Workflow
```bash
# Ultra-fast smoke tests (~6 seconds) - essential functionality only
go test -short -v -run="Quick"

# Fast test suite (~50 seconds) - skips network tests and slow concurrent operations  
go test -short -v

# Quick validation tests only
go test -short -v -run="Validation"
```

### Full Test Suite
```bash
# Complete test suite (~60-90 seconds) - includes all integration tests
go test -v

# Run specific test categories
go test -v -run="Framework"     # Core framework tests
go test -v -run="Coverage"      # Coverage collection tests  
go test -v -run="Examples"      # Integration examples
go test -v -run="Concurrent"    # Concurrent browser tests
```

### Coverage & Analysis
```bash
# Run tests with coverage
go test -coverprofile=coverage.txt ./...

# Generate HTML coverage report
go tool cover -html=coverage.txt -o coverage/go-cover.html

# Parallel execution (faster on multi-core systems)
go test -v -parallel=4
```

## Core Architecture
The project consists of a browser automation test suite that:
1. Launches a headless Chrome browser instance
2. Navigates to a test URL (localhost:8080)
3. Collects JavaScript code coverage using Chrome DevTools Protocol
4. Takes screenshots before and after user interactions
5. Generates both JavaScript and Go code coverage reports
6. Creates a unified coverage report with links to all coverage outputs

The test interacts with a web application at `http://localhost:8080/roadmap` and performs UI automation including clicking buttons and verifying state changes.

## Key Implementation Details
- Browser launches with `--no-sandbox` flag for CI compatibility
- Coverage collection uses Chrome's Profiler API for JavaScript coverage
- Test generates multiple coverage formats: JSON data, HTML reports, and screenshots
- All coverage outputs are stored in the `coverage/` directory