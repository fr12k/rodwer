# TDD Methodology for Rodwer Browser Testing Framework

## Overview

This document outlines the Test-Driven Development (TDD) methodology for the Rodwer browser testing framework. Rodwer is a Playwright-inspired browser automation framework built in Go using Rod as the underlying browser control library.

## TDD Philosophy

### Core Principles

1. **Red-Green-Refactor Cycle**: Write failing tests first, implement minimal code to pass, then refactor
2. **API-First Design**: Tests define the desired API before implementation exists
3. **Comprehensive Coverage**: Every public interface must have corresponding tests
4. **Quality Gates**: Tests serve as quality gates and regression prevention
5. **Documentation Through Tests**: Tests serve as living documentation of expected behavior

### Testing Pyramid Structure

```
    /\
   /  \  E2E Integration Tests (10%)
  /____\  
 /      \  Component Integration Tests (20%) 
/________\
          Unit Tests (70%)
```

## Project Structure and Test Organization

### File Organization

```
rodwer/
├── framework_test.go          # Comprehensive API demonstration tests
├── browser_test.go           # Core browser functionality tests  
├── test_helpers.go           # Test utilities and helpers
├── types.go                  # Type definitions (to be created)
├── browser.go                # Browser implementation (to be created)
├── page.go                   # Page implementation (to be created)
├── element.go                # Element implementation (to be created)
└── coverage/                 # Test coverage reports
```

### Test File Conventions

- `*_test.go`: Main test files containing test logic
- Test functions: `Test*` for standard tests, `Benchmark*` for performance tests
- Test suites: Use testify/suite for complex test scenarios
- Test helpers: Separate file with reusable test utilities

## API Design Through Tests

### Core Framework Components

The tests define these key interfaces:

#### 1. Browser Management
```go
type BrowserOptions struct {
    Headless       bool
    NoSandbox      bool
    Args          []string
    ExecutablePath string
    Viewport      *Viewport
    DevTools      bool
    UserAgent     string
}

type Browser interface {
    NewPage() (*Page, error)
    Pages() ([]*Page, error)
    Close() error
    IsConnected() bool
    Context() context.Context
}
```

#### 2. Page Navigation and Management
```go
type Page interface {
    Navigate(url string) error
    NavigateWithContext(ctx context.Context, url string) error
    Title() (string, error)
    URL() string
    Element(selector string) (Element, error)
    Elements(selector string) ([]Element, error)
    WaitForElement(selector string, timeout time.Duration) (Element, error)
    WaitForElementWithContext(ctx context.Context, selector string) (Element, error)
    Screenshot(options ScreenshotOptions) ([]byte, error)
    StartJSCoverage() error
    StopJSCoverage() ([]CoverageEntry, error)
    Close() error
    Context() context.Context
}
```

#### 3. Element Interaction
```go
type Element interface {
    Click() error
    Type(text string) error
    Clear() error
    Text() (string, error)
    Value() (string, error)
    TagName() (string, error)
}
```

#### 4. Screenshot and Coverage
```go
type ScreenshotOptions struct {
    FullPage bool
    Format   string // "png", "jpeg"
    Quality  int    // for JPEG
    Selector string // for element screenshots
}

type CoverageEntry struct {
    URL    string
    Source string
    Ranges []CoverageRange
}
```

## TDD Cycle Implementation

### Phase 1: RED - Write Failing Tests

#### Test Categories

1. **Framework API Tests** (`framework_test.go`)
   - Comprehensive test suite demonstrating complete API
   - Uses testify/suite for structured testing
   - Tests all major use cases and edge cases

2. **Core Browser Tests** (`browser_test.go`)
   - Browser creation and connection
   - Page management
   - Element selection and interaction
   - Waiting and timeouts
   - Screenshot capabilities

3. **Integration Tests**
   - End-to-end workflow testing
   - Real browser automation scenarios
   - Performance benchmarks

#### Test Patterns

1. **Table-Driven Tests**
```go
func TestBrowserCreation(t *testing.T) {
    tests := []struct {
        name    string
        options BrowserOptions
        wantErr bool
        errMsg  string
    }{
        // Test cases here
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

2. **Test Suites with Setup/Teardown**
```go
type BrowserTestSuite struct {
    suite.Suite
    browser *Browser
}

func (s *BrowserTestSuite) SetupSuite() {
    // Setup code
}

func (s *BrowserTestSuite) TearDownSuite() {
    // Cleanup code
}
```

3. **Benchmark Tests**
```go
func BenchmarkBrowserCreation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Benchmark code
    }
}
```

### Phase 2: GREEN - Minimal Implementation

After tests are written and failing, implement minimal code to make tests pass:

1. Create type definitions in `types.go`
2. Implement `Browser` struct and methods in `browser.go`
3. Implement `Page` struct and methods in `page.go`
4. Implement `Element` struct and methods in `element.go`
5. Add helper functions and utilities

### Phase 3: REFACTOR - Improve Code Quality

Once tests pass:

1. Refactor for better design patterns
2. Optimize performance
3. Improve error handling
4. Add documentation
5. Ensure consistent coding style

## Test Utilities and Helpers

### Test Helper Functions (`test_helpers.go`)

1. **Browser Creation**: `NewTestBrowser()` - Creates browser configured for testing
2. **Test Server**: `NewTestServer()` - HTTP server with common test endpoints
3. **Page Factory**: `NewTestPage()` - Creates pages with custom HTML content
4. **Test Helper**: `TestHelper` struct with common assertions and utilities
5. **Performance Testing**: `PerformanceTestRunner` for timing operations
6. **Concurrent Testing**: `ConcurrentTestRunner` for parallel test execution
7. **Retry Logic**: `RetryHelper` for flaky operation handling
8. **Mock Server**: `MockResponseServer` for controlled HTTP responses

### Example Usage

```go
func TestExample(t *testing.T) {
    // Use test helper for browser management
    helper := NewTestHelper(t)
    defer helper.Close()
    
    page := helper.NewPage()
    helper.NavigateToHTML(page, "<html><body><h1>Test</h1></body></html>")
    
    element := helper.WaitForElement(page, "h1", 5*time.Second)
    helper.AssertElementText(element, "Test")
}
```

## Quality Standards

### Test Coverage Requirements

- **Unit Tests**: ≥80% coverage for business logic
- **Integration Tests**: ≥70% coverage for critical workflows  
- **End-to-End Tests**: Cover all main user scenarios
- **Error Paths**: Test all error conditions and edge cases

### Test Quality Metrics

1. **Reliability**: Tests must be deterministic and not flaky
2. **Performance**: Unit tests <100ms, integration tests <5s
3. **Maintainability**: Tests should be easy to understand and modify
4. **Isolation**: Tests should not depend on external resources
5. **Clear Assertions**: Use descriptive error messages

### Testify Assertion Guidelines

```go
// Use require for critical assertions that should stop execution
require.NoError(t, err, "Critical operation failed")
require.NotNil(t, browser, "Browser must not be nil")

// Use assert for non-critical validations
assert.Equal(t, expected, actual, "Values should match")
assert.Contains(t, haystack, needle, "Should contain substring")

// Use meaningful error messages
assert.True(t, browser.IsConnected(), "Browser should be connected after creation")
```

## Testing Workflow

### Development Process

1. **Write Test First**: Define the desired API through tests
2. **Run Tests**: Verify they fail (RED phase)
3. **Implement Minimal Code**: Make tests pass (GREEN phase)  
4. **Refactor**: Improve code quality while keeping tests green
5. **Add More Tests**: Expand test coverage for edge cases
6. **Repeat**: Continue cycle for next feature

### Continuous Integration

1. **Pre-commit Hooks**: Run tests before commits
2. **CI Pipeline**: Run full test suite on every push
3. **Coverage Reports**: Generate and track test coverage
4. **Performance Benchmarks**: Monitor performance regressions
5. **Quality Gates**: Block merges if tests fail or coverage drops

### Test Execution Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.txt ./...

# Run specific test suite
go test -run TestBrowserSuite

# Run benchmarks
go test -bench=. -benchmem

# Generate coverage report
go tool cover -html=coverage.txt -o coverage/go-cover.html
```

## Best Practices

### Test Design

1. **Single Responsibility**: Each test should test one specific behavior
2. **Descriptive Names**: Test names should clearly describe what they test
3. **Arrange-Act-Assert**: Structure tests with clear setup, execution, and verification
4. **Independent Tests**: Tests should not depend on each other
5. **Data-Driven Tests**: Use table-driven tests for multiple scenarios

### Error Testing

1. **Test All Error Paths**: Every error condition should have a test
2. **Specific Error Validation**: Check error messages and types
3. **Timeout Testing**: Test timeout scenarios and cancellation
4. **Resource Cleanup**: Ensure proper cleanup even when tests fail

### Performance Testing

1. **Baseline Metrics**: Establish performance baselines
2. **Regression Detection**: Alert on performance degradation
3. **Resource Monitoring**: Monitor memory usage and resource leaks
4. **Scalability Testing**: Test with multiple browsers/pages

## Framework-Specific Considerations

### Browser Testing Challenges

1. **Timing Issues**: Use explicit waits instead of sleeps
2. **Cross-Platform**: Test on different operating systems
3. **Resource Management**: Properly close browsers and pages
4. **Headless vs GUI**: Test both modes when applicable
5. **Network Conditions**: Test under various network scenarios

### Rod Integration

1. **Rod Abstractions**: Hide Rod complexity behind clean APIs
2. **Error Wrapping**: Provide meaningful error messages
3. **Context Support**: Use contexts for cancellation and timeouts
4. **Resource Cleanup**: Ensure Rod resources are properly released

## Success Criteria

The TDD methodology is successful when:

1. **All Tests Pass**: Framework implementation satisfies test requirements
2. **High Coverage**: Achieve target coverage percentages
3. **Clean API**: Tests demonstrate intuitive and consistent API
4. **Performance**: Framework meets performance benchmarks
5. **Maintainability**: Code is easy to extend and modify
6. **Documentation**: Tests serve as comprehensive API documentation

## Next Steps

1. **Phase 1 Implementation**: Implement types and basic browser functionality
2. **Phase 2 Enhancement**: Add advanced features like coverage collection
3. **Phase 3 Optimization**: Performance improvements and edge case handling
4. **Phase 4 Polish**: Documentation, examples, and final testing

This TDD methodology ensures that the Rodwer framework is built with quality, reliability, and maintainability as core principles from the very beginning.