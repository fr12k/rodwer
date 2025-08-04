# Rodwer

A Go-based browser automation and testing framework built on top of the [Rod](https://github.com/go-rod/rod) library. Rodwer provides a clean, type-safe API for web browser automation, JavaScript code coverage collection, and comprehensive testing utilities.

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24.1-blue.svg)](https://golang.org/)
[![Rod Version](https://img.shields.io/badge/rod-0.116.2-green.svg)](https://github.com/go-rod/rod)
[![Test Status](https://img.shields.io/badge/tests-passing-brightgreen.svg)](#testing)

## Features

- **Clean API**: Simplified browser automation with intuitive method names
- **Type Safety**: Full Go type safety with comprehensive error handling
- **JavaScript Coverage**: Built-in JavaScript code coverage collection using Chrome DevTools Protocol
- **Screenshot Capture**: Full page, viewport, and element-specific screenshots
- **Test Utilities**: Comprehensive testing helpers and utilities
- **Concurrent Testing**: Support for parallel test execution
- **Performance Benchmarking**: Built-in performance testing capabilities
- **Context Support**: Full context.Context integration for timeouts and cancellation

## Installation

```bash
go get github.com/fr12k/rodwer
```

## Quick Start

### Basic Browser Automation

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/fr12k/rodwer"
)

func main() {
    // Create a new browser instance
    browser, err := rodwer.NewBrowser(rodwer.BrowserOptions{
        Headless: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close()

    // Create a new page
    page, err := browser.NewPage()
    if err != nil {
        log.Fatal(err)
    }
    defer page.Close()

    // Navigate to a website
    err = page.Navigate("https://example.com")
    if err != nil {
        log.Fatal(err)
    }

    // Get page title
    title, err := page.Title()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Page title: %s\n", title)
}
```

### Element Interaction

```go
// Find and interact with elements
element, err := page.Element("#button")
if err != nil {
    log.Fatal(err)
}

// Click the element
err = element.Click()
if err != nil {
    log.Fatal(err)
}

// Type into input fields
input, err := page.Element("input[name='username']")
if err != nil {
    log.Fatal(err)
}

err = input.Type("myusername")
if err != nil {
    log.Fatal(err)
}
```

### Screenshot Capture

```go
// Take a full page screenshot
data, err := page.Screenshot(rodwer.ScreenshotOptions{
    FullPage: true,
    Format:   "png",
})
if err != nil {
    log.Fatal(err)
}

// Save screenshot directly to file
err = page.ScreenshotToFile("screenshot.png")
if err != nil {
    log.Fatal(err)
}

// Element screenshot
element, err := page.Element(".important-section")
if err != nil {
    log.Fatal(err)
}

err = element.ScreenshotToFile("element.png")
if err != nil {
    log.Fatal(err)
}
```

### JavaScript Coverage Collection

```go
// Start coverage collection
err = page.StartJSCoverage()
if err != nil {
    log.Fatal(err)
}

// Navigate and interact with the page
err = page.Navigate("https://example.com")
if err != nil {
    log.Fatal(err)
}

// Perform interactions that execute JavaScript
button, err := page.Element("#interactive-button")
if err != nil {
    log.Fatal(err)
}
err = button.Click()
if err != nil {
    log.Fatal(err)
}

// Collect coverage data
coverage, err := page.StopJSCoverage()
if err != nil {
    log.Fatal(err)
}

// Process coverage data
for _, entry := range coverage {
    fmt.Printf("Script: %s\n", entry.URL)
    fmt.Printf("Coverage ranges: %d\n", len(entry.Ranges))
}
```

## Advanced Usage

### Context Support

```go
// Navigate with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err = page.NavigateWithContext(ctx, "https://example.com")
if err != nil {
    log.Fatal(err)
}

// Wait for element with context
element, err := page.WaitForElementWithContext(ctx, "#dynamic-content")
if err != nil {
    log.Fatal(err)
}
```

### Multiple Pages

```go
// Create multiple pages
var pages []*rodwer.Page
for i := 0; i < 3; i++ {
    page, err := browser.NewPage()
    if err != nil {
        log.Fatal(err)
    }
    pages = append(pages, page)
}

// Get all pages
allPages, err := browser.Pages()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total pages: %d\n", len(allPages))
```

### Error Handling

```go
// Comprehensive error handling
element, err := page.Element("#might-not-exist")
if err != nil {
    if strings.Contains(err.Error(), "element not found") {
        // Handle missing element
        log.Println("Element not found, continuing...")
        return
    }
    // Handle other errors
    log.Fatal(err)
}
```

### Performance Testing

```go
func BenchmarkPageNavigation(b *testing.B) {
    browser, cleanup, err := rodwer.NewTestBrowser()
    if err != nil {
        b.Fatal(err)
    }
    defer cleanup()

    page, err := browser.NewPage()
    if err != nil {
        b.Fatal(err)
    }
    defer page.Close()

    url := "data:text/html,<html><body><h1>Benchmark</h1></body></html>"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        err := page.Navigate(url)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Project Structure

```
rodwer/
├── README.md                 # This file
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
├── types.go                  # Core types and implementations
├── browser_test.go           # Integration tests with coverage
├── framework_test.go         # Framework API tests
├── test_helpers.go           # Testing utilities
├── quick_test.go            # Fast validation tests
├── examples_test.go         # Usage examples
├── examples/                # Example code
│   ├── basic_example.go     # Basic usage example
│   ├── go.mod              # Example module
│   └── go.sum              # Example checksums
├── coverage/               # Generated coverage reports
│   ├── index.html         # Unified coverage report
│   ├── js-coverage.html   # JavaScript coverage
│   ├── go-cover.html      # Go coverage  
│   ├── js-coverage.json   # Raw JS coverage data
│   └── *.png             # Screenshots
├── CLAUDE.md              # Claude Code guidance
└── TDD_METHODOLOGY.md     # Development methodology
```

## Contributing

### Development Workflow

1. **Clone the repository**
```bash
git clone https://github.com/fr12k/rodwer.git
cd rodwer
```

2. **Run tests**
```bash
# Fast feedback loop
go test -short -v

# Full test suite
go test -v
```

3. **Generate coverage reports**
```bash
go test -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt -o coverage/go-cover.html
```

### Test-Driven Development

This project follows TDD principles:
- Tests define the desired API before implementation
- Red → Green → Refactor cycle
- Comprehensive test coverage for all features

### Code Quality

- Full Go type safety
- Comprehensive error handling
- Context support throughout
- Thread-safe operations
- Resource cleanup management

## Dependencies

- **Go 1.24.1+**: Modern Go version with latest features
- **Rod 0.116.2**: Browser automation library
- **Testify 1.10.0**: Testing utilities and assertions

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Acknowledgments

- Built on top of the excellent [Rod](https://github.com/go-rod/rod) library
- Uses [Testify](https://github.com/stretchr/testify) for comprehensive testing
- Inspired by modern browser automation tools like Playwright and Puppeteer

## Support

For questions, issues, or contributions:
- Open an issue on GitHub
- Submit a pull request
- Check the examples for usage patterns