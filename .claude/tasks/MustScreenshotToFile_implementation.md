# MustScreenshotToFile Implementation Plan

## Overview
Implement MustScreenshotToFile functions similar to Rod's MustScreenshot but specifically designed to save screenshots directly to files with robust error handling and path management.

## Background Research
- Rod's `MustScreenshot(toFile ...string) []byte` takes optional file path and returns bytes
- Current project has `Screenshot(options ScreenshotOptions) ([]byte, error)` that returns bytes
- Current usage: `page.MustScreenshot(screenshot1)` where screenshot1 = "coverage/screenshot-page.png"
- Need functions that save directly to file with proper error handling

## Implementation Plan

### Functions to Implement
1. `func (p *Page) MustScreenshotToFile(filePath string, options ...ScreenshotOptions) error`
2. `func (p *Page) MustScreenshotSimpleToFile(filePath string) error`
3. `func (e Element) MustScreenshotToFile(filePath string) error`

### Key Features
- **Automatic directory creation**: Creates parent directories if they don't exist
- **Format detection**: Auto-detects format from file extension if not specified
- **Error handling**: Proper validation of page state, file paths, and I/O operations
- **Flexible options**: Supports all existing ScreenshotOptions parameters

### Files to Modify
- `types.go` - Add new methods to Page and Element structs
- `framework_test.go` - Add comprehensive tests

## Progress Log
- [x] Created task directory and plan file
- [x] Implement Page.ScreenshotToFile method (renamed from MustScreenshotToFile)
- [x] Implement Page.ScreenshotSimpleToFile method
- [x] Implement Element.ScreenshotToFile method
- [x] Add comprehensive tests
- [x] Validate implementation with test runs

## Implementation Complete ✅

All ScreenshotToFile functions have been successfully implemented and tested:

### Functions Implemented:
1. `Page.ScreenshotToFile(filePath string, options ...ScreenshotOptions) error`
2. `Page.ScreenshotSimpleToFile(filePath string) error`
3. `Element.ScreenshotToFile(filePath string) error`

### Features:
- ✅ Automatic directory creation
- ✅ Format auto-detection from file extension
- ✅ Comprehensive error handling
- ✅ Support for PNG and JPEG formats
- ✅ Quality settings for JPEG
- ✅ Full page and element screenshots

### Test Results:
- ✅ All tests passing
- ✅ Code compiles without errors
- ✅ File creation and error handling validated

## Implementation Details

### Function Signatures
```go
// Page methods
func (p *Page) MustScreenshotToFile(filePath string, options ...ScreenshotOptions) error
func (p *Page) MustScreenshotSimpleToFile(filePath string) error

// Element method
func (e Element) MustScreenshotToFile(filePath string) error
```

### Error Handling Strategy
- Validate page/element is not closed
- Validate file path is not empty
- Create parent directories if needed
- Handle file I/O errors gracefully
- Auto-detect format from file extension if not specified in options

### Testing Strategy
- Test successful screenshot saves with various formats (PNG, JPEG)
- Test directory creation functionality
- Test error conditions (closed page, invalid paths)
- Test different screenshot options (full page, element, quality settings)