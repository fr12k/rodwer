package rodwer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	coverageDir = "coverage"
	jsCoverage  = "coverage/js-coverage.json"
	jsHTML      = "coverage/js-coverage.html"
	goCoverHTML = "coverage/go-cover.html"
	goCoverRaw  = "coverage.txt"
	screenshot1 = "coverage/screenshot-page.png"
	screenshot2 = "coverage/screenshot-after-click.png"
	indexHTML   = "coverage/index.html"
)

// TDD Phase 1: Core Browser API Tests
// These tests define our desired API and will fail until we implement the framework

// BrowserTestSuite contains core browser functionality tests
type BrowserTestSuite struct {
	suite.Suite
}

func (s *BrowserTestSuite) TestBrowserCreationAndConnection() {
	tests := []struct {
		name    string
		options BrowserOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "create browser with default options",
			options: BrowserOptions{
				Headless: true,
			},
			wantErr: false,
		},
		{
			name: "create browser with custom launch options",
			options: BrowserOptions{
				Headless:  true,
				NoSandbox: true,
				Args:      []string{"--disable-web-security"},
			},
			wantErr: false,
		},
		{
			name: "fail on invalid executable path",
			options: BrowserOptions{
				Headless:       true,
				ExecutablePath: "/nonexistent/path/chrome",
			},
			wantErr: true,
			errMsg:  "executable not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			browser, err := NewBrowser(tt.options)
			if tt.wantErr {
				s.Error(err)
				if tt.errMsg != "" {
					s.Contains(err.Error(), tt.errMsg)
				}
				return
			}

			s.Require().NoError(err, "Failed to create browser")
			s.NotNil(browser, "Browser should not be nil")

			// Test browser is connected
			s.True(browser.IsConnected(), "Browser should be connected")

			// Test browser context
			ctx := browser.Context()
			s.NotNil(ctx, "Browser context should not be nil")

			// Clean up
			err = browser.Close()
			s.NoError(err, "Failed to close browser")
			s.False(browser.IsConnected(), "Browser should be disconnected after close")
		})
	}
}

func (s *BrowserTestSuite) TestPageCreationAndManagement() {
	browser, err := NewBrowser(BrowserOptions{Headless: true})
	s.Require().NoError(err)
	defer browser.Close()

	s.Run("create new page", func() {
		page, err := browser.NewPage()
		s.Require().NoError(err)
		s.NotNil(page)
		defer page.Close()

		// Test page properties
		s.NotEmpty(page.URL(), "Page should have a URL")
		s.NotNil(page.Context(), "Page should have a context")
	})

	s.Run("create multiple pages", func() {
		var pages []*Page
		for i := 0; i < 3; i++ {
			page, err := browser.NewPage()
			s.Require().NoError(err)
			pages = append(pages, page)
		}

		// Verify all pages exist
		allPages, err := browser.Pages()
		s.Require().NoError(err)
		s.GreaterOrEqual(len(allPages), 3, "Should have at least 3 pages")

		// Clean up
		for _, page := range pages {
			err := page.Close()
			s.NoError(err)
		}
	})

	s.Run("page navigation", func() {
		page, err := browser.NewPage()
		s.Require().NoError(err)
		defer page.Close()

		// Test navigation to data URL
		testHTML := `<html><head><title>Test Page</title></head><body><h1>Hello World</h1></body></html>`
		dataURL := "data:text/html," + testHTML

		err = page.Navigate(dataURL)
		s.Require().NoError(err)

		// Verify navigation
		title, err := page.Title()
		s.Require().NoError(err)
		s.Equal("Test Page", title)

		url := page.URL()
		s.Contains(url, "data:text/html")
	})
}

func (s *BrowserTestSuite) TestElementSelectionAndInteraction() {
	browser, err := NewBrowser(BrowserOptions{Headless: true})
	s.Require().NoError(err)
	defer browser.Close()

	page, err := browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	// Navigate to test page
	testHTML := `
	<html>
	<body>
		<div id="container">
			<h1 id="title" class="header">Test Title</h1>
			<input id="input" type="text" value="initial">
			<button id="btn" class="button">Click Me</button>
			<ul class="list">
				<li class="item">Item 1</li>
				<li class="item">Item 2</li>
			</ul>
		</div>
	</body>
	</html>`

	err = page.Navigate("data:text/html," + testHTML)
	s.Require().NoError(err)

	tests := []struct {
		name     string
		selector string
		action   func(*Page, string) error
		verify   func(*Page, string) error
	}{
		{
			name:     "select element by ID",
			selector: "#title",
			action: func(p *Page, sel string) error {
				el, err := p.Element(sel)
				if err != nil {
					return err
				}
				text, err := el.Text()
				if err != nil {
					return err
				}
				s.Equal("Test Title", text)
				return nil
			},
		},
		{
			name:     "select element by class",
			selector: ".header",
			action: func(p *Page, sel string) error {
				el, err := p.Element(sel)
				if err != nil {
					return err
				}
				tagName, err := el.TagName()
				if err != nil {
					return err
				}
				s.Equal("H1", strings.ToUpper(tagName))
				return nil
			},
		},
		{
			name:     "select multiple elements",
			selector: ".item",
			action: func(p *Page, sel string) error {
				elements, err := p.Elements(sel)
				if err != nil {
					return err
				}
				s.Len(elements, 2, "Should find 2 list items")
				return nil
			},
		},
		{
			name:     "interact with input element",
			selector: "#input",
			action: func(p *Page, sel string) error {
				el, err := p.Element(sel)
				if err != nil {
					return err
				}

				// Clear and type new value
				err = el.Clear()
				if err != nil {
					return err
				}

				err = el.Type("new value")
				if err != nil {
					return err
				}

				value, err := el.Value()
				if err != nil {
					return err
				}
				s.Equal("new value", value)
				return nil
			},
		},
		{
			name:     "click button element",
			selector: "#btn",
			action: func(p *Page, sel string) error {
				el, err := p.Element(sel)
				if err != nil {
					return err
				}

				return el.Click()
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.action(page, tt.selector)
			s.NoError(err)

			if tt.verify != nil {
				err := tt.verify(page, tt.selector)
				s.NoError(err)
			}
		})
	}
}

func (s *BrowserTestSuite) TestWaitingAndTimeouts() {
	browser, err := NewBrowser(BrowserOptions{Headless: true})
	s.Require().NoError(err)
	defer browser.Close()

	page, err := browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	// Test page with dynamic content
	testHTML := `
	<html>
	<body>
		<div id="initial">Initial Content</div>
		<script>
			setTimeout(function() {
				var div = document.createElement('div');
				div.id = 'delayed';
				div.textContent = 'Delayed Content';
				document.body.appendChild(div);
			}, 1000);
		</script>
	</body>
	</html>`

	err = page.Navigate("data:text/html," + testHTML)
	s.Require().NoError(err)

	s.Run("wait for element to appear", func() {
		// This should succeed within timeout
		el, err := page.WaitForElement("#delayed", 3*time.Second)
		s.Require().NoError(err)
		s.NotNil(el)

		text, err := el.Text()
		s.Require().NoError(err)
		s.Equal("Delayed Content", text)
	})

	s.Run("timeout when element doesn't appear", func() {
		// This should timeout
		_, err := page.WaitForElement("#nonexistent", 500*time.Millisecond)
		s.Error(err, "Should timeout waiting for non-existent element")
		s.Contains(err.Error(), "timeout", "Error should mention timeout")
	})

	s.Run("wait with context cancellation", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_, err := page.WaitForElementWithContext(ctx, "#another-nonexistent")
		s.Error(err, "Should be cancelled by context")
	})
}

func (s *BrowserTestSuite) TestScreenshotCapabilities() {
	browser, err := NewBrowser(BrowserOptions{Headless: true})
	s.Require().NoError(err)
	defer browser.Close()

	page, err := browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	testHTML := `
	<html>
	<head><style>
		body { font-family: Arial; margin: 20px; }
		.red-box { width: 100px; height: 100px; background: red; }
	</style></head>
	<body>
		<h1>Screenshot Test</h1>
		<div class="red-box"></div>
	</body>
	</html>`

	err = page.Navigate("data:text/html," + testHTML)
	s.Require().NoError(err)

	s.Run("full page screenshot", func() {
		data, err := page.Screenshot(ScreenshotOptions{
			FullPage: true,
			Format:   "png",
		})
		s.Require().NoError(err)
		s.NotEmpty(data, "Screenshot data should not be empty")
		s.Greater(len(data), 1000, "PNG screenshot should be substantial size")
	})

	s.Run("viewport screenshot", func() {
		data, err := page.Screenshot(ScreenshotOptions{
			FullPage: false,
			Format:   "jpeg",
			Quality:  90,
		})
		s.Require().NoError(err)
		s.NotEmpty(data, "Screenshot data should not be empty")
	})

	s.Run("element screenshot", func() {
		data, err := page.Screenshot(ScreenshotOptions{
			Selector: ".red-box",
			Format:   "png",
		})
		s.Require().NoError(err)
		s.NotEmpty(data, "Element screenshot should not be empty")
	})
}

// Run the browser test suite
func TestBrowserSuite(t *testing.T) {
	suite.Run(t, new(BrowserTestSuite))
}

func TestCoverageReport(t *testing.T) {
	require.NoError(t, os.MkdirAll(coverageDir, 0755))

	// Create embedded test server
	testServer := createTestServer()
	defer testServer.Close()

	// Get the test server URL
	testServerURL := testServer.URL + "/roadmap"

	// Create browser using Rodwer API
	browserOpts := BrowserOptions{
		Headless:  true,
		NoSandbox: true,
	}

	browser, err := NewBrowser(browserOpts)
	require.NoError(t, err)
	defer browser.Close()

	// Create new page
	page, err := browser.NewPage()
	require.NoError(t, err)
	defer page.Close()

	// Start JavaScript coverage collection
	require.NoError(t, page.StartJSCoverage())

	// Navigate to test page
	require.NoError(t, page.Navigate(testServerURL))

	// Give DOMContentLoaded event a moment to fire and execute calculateProgress()
	t.Logf("Allowing DOMContentLoaded event to execute...")
	time.Sleep(200 * time.Millisecond)

	// Take screenshot before interaction
	err = page.ScreenshotToFile(screenshot1)
	require.NoError(t, err)

	// Click the button and verify it changes
	btn, err := page.Element("#copy-all-btn")
	require.NoError(t, err)
	require.NoError(t, btn.Click())

	// Wait a bit for async JavaScript (setTimeout) to execute
	time.Sleep(200 * time.Millisecond)

	// Verify button text changed
	btnText, err := btn.Text()
	require.NoError(t, err)
	require.Contains(t, btnText, "Copied", "Button text should contain 'Copied' after click")

	// Take screenshot after interaction
	err = page.ScreenshotToFile(screenshot2)
	require.NoError(t, err)

	// Stop JavaScript coverage with async detection (using quick options to minimize timeout issues)
	coverageOptions := DefaultCoverageOptions()
	coverageOptions.EnableDebugLogs = true // Enable debug logging to see what's captured

	t.Logf("Collecting JavaScript coverage with enhanced async detection...")
	coverageEntries, err := page.StopJSCoverageWithWait(coverageOptions)
	require.NoError(t, err)

	t.Logf("Coverage collection complete: %d entries captured", len(coverageEntries))

	// Convert to old format for existing report generation
	result := convertToOldCoverageFormat(coverageEntries)

	// Save raw coverage data
	b, _ := json.MarshalIndent(result, "", "  ")
	require.NoError(t, os.WriteFile(jsCoverage, b, 0644))

	// Generate coverage report using existing function
	generateJSReportFromEntries(t, coverageEntries)

	jsPct := computeJavaScriptCoverageFromEntries(coverageEntries)
	goPct := computeGoCoveragePercent(t)

	generateCoverageIndex(goPct, jsPct)

	// OPTIONAL: generate go-cover.html via tool
	generateGoCoverHTML(t)
}

func must(_ any, err error) {
	if err != nil {
		panic(err)
	}
}

// convertToOldCoverageFormat converts new CoverageEntry to old format for compatibility
func convertToOldCoverageFormat(entries []CoverageEntry) []*proto.ProfilerScriptCoverage {
	var result []*proto.ProfilerScriptCoverage

	for i, entry := range entries {
		scriptCov := &proto.ProfilerScriptCoverage{
			ScriptID: proto.RuntimeScriptID(fmt.Sprintf("script-%d", i)),
			URL:      entry.URL,
		}

		// Convert ranges to ProfilerFunctionCoverage format
		if len(entry.Ranges) > 0 {
			functions := make([]*proto.ProfilerFunctionCoverage, 1)
			functions[0] = &proto.ProfilerFunctionCoverage{
				FunctionName: "",
				Ranges:       make([]*proto.ProfilerCoverageRange, 0),
			}

			for _, r := range entry.Ranges {
				functions[0].Ranges = append(functions[0].Ranges, &proto.ProfilerCoverageRange{
					StartOffset: r.Start,
					EndOffset:   r.End,
					Count:       r.Count,
				})
			}
			scriptCov.Functions = functions
		}

		result = append(result, scriptCov)
	}

	return result
}

// generateJSReportFromEntries generates report from new CoverageEntry format
func generateJSReportFromEntries(t *testing.T, entries []CoverageEntry) {
	// Convert to old format and use existing function
	oldFormat := convertToOldCoverageFormat(entries)

	// Create mapping from script index to source for the enhanced report
	indexToSource := make(map[int]string)
	for i, entry := range entries {
		indexToSource[i] = entry.Source
	}

	// Use the real Istanbul.js-style report generation with pre-collected sources
	generateJSReportWithPreCollectedSources(t, oldFormat, indexToSource)
}

// computeJavaScriptCoverageFromEntries computes coverage percentage from new format
func computeJavaScriptCoverageFromEntries(entries []CoverageEntry) float64 {
	totalBytes := 0
	coveredBytes := 0

	for _, entry := range entries {
		if entry.Source == "" {
			continue
		}

		totalBytes += len(entry.Source)

		// Calculate covered bytes from ranges
		covered := make([]bool, len(entry.Source))
		for _, r := range entry.Ranges {
			if r.Count > 0 && r.Start >= 0 && r.End <= len(entry.Source) {
				for i := r.Start; i < r.End; i++ {
					covered[i] = true
				}
			}
		}

		for _, c := range covered {
			if c {
				coveredBytes++
			}
		}
	}

	if totalBytes == 0 {
		return 0
	}

	return float64(coveredBytes) / float64(totalBytes) * 100
}

// SourceProvider is a function type that provides source code for a given script index and ScriptCoverage
type SourceProvider func(index int, script *proto.ProfilerScriptCoverage) (string, error)

// generateJSReportUnified generates Istanbul.js-style report with flexible source fetching
func generateJSReportUnified(raw []*proto.ProfilerScriptCoverage, sourceProvider SourceProvider, outputFunc func(string, ...interface{})) FilteringStats {
	// Use application coverage filtering options for HTML report generation
	filterOptions := getFilterOptions("application")

	entries := make([]FileEntry, 0, len(raw))
	var totalMetrics CoverageMetrics
	var filterStats FilteringStats

	filterStats.TotalScripts = len(raw)
	filterStats.FilterReasons = make(map[string]int)

	// Process each script individually to avoid losing scripts with same URL
	for i, r := range raw {
		// Get script source using the provided strategy
		scriptSource, err := sourceProvider(i, r)
		if err != nil || scriptSource == "" {
			filterStats.FilterReasons["source_unavailable"]++
			continue
		}

		// Apply filtering logic
		isApp, reason := isApplicationScript(r, scriptSource, filterOptions)
		filterStats.FilterReasons[reason]++

		if !isApp {
			continue // Skip this script
		}

		// Create unique URL identifier to distinguish scripts with same URL
		url := r.URL
		if url == "" {
			url = fmt.Sprintf("Script_%s", r.ScriptID)
		} else {
			// Add script ID to make each script entry unique
			url = fmt.Sprintf("%s#%s", url, r.ScriptID)
		}

		// Collect all ranges from all functions for this script
		var allRanges []*proto.ProfilerCoverageRange
		for _, function := range r.Functions {
			if function.Ranges != nil {
				allRanges = append(allRanges, function.Ranges...)
			}
		}

		lines := strings.Split(scriptSource, "\n")

		// Calculate metrics for this individual script
		metrics := calculateCoverageMetrics(scriptSource, allRanges, r.Functions)

		entry := FileEntry{
			ScriptID: r.ScriptID,
			URL:      url,
			Source:   scriptSource,
			Lines:    lines,
			Ranges:   allRanges,
			Metrics:  metrics,
		}

		entries = append(entries, entry)

		// Add to total metrics
		totalMetrics.Statements.Total += metrics.Statements.Total
		totalMetrics.Statements.Covered += metrics.Statements.Covered
		totalMetrics.Functions.Total += metrics.Functions.Total
		totalMetrics.Functions.Covered += metrics.Functions.Covered
		totalMetrics.Lines.Total += metrics.Lines.Total
		totalMetrics.Lines.Covered += metrics.Lines.Covered
	}

	// Calculate final filtering statistics
	filterStats.ApplicationScripts = len(entries)
	filterStats.FilteredOut = filterStats.TotalScripts - filterStats.ApplicationScripts

	// Calculate total percentages
	if totalMetrics.Statements.Total > 0 {
		totalMetrics.Statements.Pct = float64(totalMetrics.Statements.Covered) / float64(totalMetrics.Statements.Total) * 100
	}
	if totalMetrics.Functions.Total > 0 {
		totalMetrics.Functions.Pct = float64(totalMetrics.Functions.Covered) / float64(totalMetrics.Functions.Total) * 100
	}
	if totalMetrics.Lines.Total > 0 {
		totalMetrics.Lines.Pct = float64(totalMetrics.Lines.Covered) / float64(totalMetrics.Lines.Total) * 100
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].URL < entries[j].URL })

	html := generateIstanbulStyleHTML(entries, totalMetrics, filterStats)

	jsHTML := "coverage/js-coverage.html"
	_ = os.WriteFile(jsHTML, []byte(html), 0644)

	outputFunc("JavaScript coverage report written to %s", jsHTML)
	outputFunc("Coverage Summary - Statements: %.1f%%, Functions: %.1f%%, Lines: %.1f%%",
		totalMetrics.Statements.Pct, totalMetrics.Functions.Pct, totalMetrics.Lines.Pct)

	return filterStats
}

// generateJSReportWithPreCollectedSources generates Istanbul.js-style report with pre-collected source data
func generateJSReportWithPreCollectedSources(t *testing.T, raw []*proto.ProfilerScriptCoverage, indexToSource map[int]string) {
	sourceProvider := func(index int, script *proto.ProfilerScriptCoverage) (string, error) {
		source := indexToSource[index]
		if source == "" {
			return "", fmt.Errorf("source unavailable for index %d", index)
		}
		return source, nil
	}

	outputFunc := func(format string, args ...interface{}) {
		t.Logf(format, args...)
	}

	_ = generateJSReportUnified(raw, sourceProvider, outputFunc)
}

type OldCoverageEntry struct {
	ScriptID proto.RuntimeScriptID
	URL      string
	Source   string
	Ranges   []*proto.ProfilerCoverageRange
}

type CoverageMetrics struct {
	Statements CoverageStat `json:"statements"`
	Branches   CoverageStat `json:"branches"`
	Functions  CoverageStat `json:"functions"`
	Lines      CoverageStat `json:"lines"`
}

type CoverageStat struct {
	Total   int     `json:"total"`
	Covered int     `json:"covered"`
	Skipped int     `json:"skipped"`
	Pct     float64 `json:"pct"`
}

type CoverageFilterOptions struct {
	ExcludeEmptyURLs                bool     // Default: true - exclude scripts with empty URLs
	ExcludeDevTools                 bool     // Default: true - exclude automation framework scripts
	ExcludeBrowserExt               bool     // Default: true - exclude browser extension scripts
	ExcludeFrameworkTools           bool     // Default: true - exclude modern framework development tools
	ExcludeCDNLibraries             bool     // Default: true - exclude CDN-hosted libraries
	ExcludeMinifiedCode             bool     // Default: true - exclude minified/generated code
	ExcludeTestFrameworks           bool     // Default: true - exclude test framework code
	ExcludeHighDensityInlineScripts bool     // Default: true - exclude inline scripts with high statement density
	ExcludeInlineSystemScripts      bool     // Default: true - exclude browser-generated inline scripts
	MinScriptSize                   int      // Default: 30 - minimum script size in characters
	MaxStatementsPerLine            int      // Default: 50 - maximum statements per line before considering minified
	CustomExcludePatterns           []string // User-defined exclusion patterns
	CustomIncludePatterns           []string // Force include patterns (overrides exclusions)
}

type FilteringStats struct {
	TotalScripts         int
	ApplicationScripts   int
	FilteredOut          int
	FilterReasons        map[string]int
	ProcessingTimeMs     int64   // Total processing time in milliseconds
	AverageTimePerScript float64 // Average time per script in milliseconds
}

type FileEntry struct {
	ScriptID proto.RuntimeScriptID
	URL      string
	Source   string
	Lines    []string
	Ranges   []*proto.ProfilerCoverageRange
	Metrics  CoverageMetrics
}

// getFilterOptions returns CoverageFilterOptions based on the specified profile
// Supported profiles: "default", "development", "production", "application"
func getFilterOptions(profile string) CoverageFilterOptions {
	// Base options that are common across all profiles
	options := CoverageFilterOptions{
		ExcludeEmptyURLs:                true,
		ExcludeDevTools:                 true,
		ExcludeBrowserExt:               true,
		ExcludeFrameworkTools:           true,
		ExcludeCDNLibraries:             true,
		ExcludeMinifiedCode:             true,
		ExcludeTestFrameworks:           true,
		ExcludeHighDensityInlineScripts: true,
		ExcludeInlineSystemScripts:      true,
		MinScriptSize:                   30,
		MaxStatementsPerLine:            50,
		CustomExcludePatterns:           []string{},
		CustomIncludePatterns:           []string{},
	}

	// Apply profile-specific customizations
	switch profile {
	case "development":
		options.ExcludeFrameworkTools = false           // Include for debugging
		options.ExcludeMinifiedCode = false             // Include for debugging
		options.ExcludeTestFrameworks = false           // Include test code
		options.ExcludeHighDensityInlineScripts = false // Include for analysis
		options.MinScriptSize = 10                      // More permissive
		options.MaxStatementsPerLine = 100              // More permissive threshold
	case "production":
		options.MinScriptSize = 50       // Stricter minimum
		options.MaxStatementsPerLine = 5 // Ultra-strict threshold for production
	case "application":
		options.MinScriptSize = 15       // More permissive for small application scripts
		options.MaxStatementsPerLine = 5 // Keep strict threshold for minification detection
	case "default":
		// Use base options as-is
	}

	return options
}

// isApplicationScript determines if a script should be included in coverage reports
func isApplicationScript(scriptCoverage *proto.ProfilerScriptCoverage, source string, options CoverageFilterOptions) (bool, string) {
	// Check custom include patterns first (they override all exclusions)
	for _, pattern := range options.CustomIncludePatterns {
		if strings.Contains(strings.ToLower(scriptCoverage.URL), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(source), strings.ToLower(pattern)) {
			return true, "custom_include"
		}
	}

	// 1. Universal inline script blocking - exclude ALL inline-script-* patterns
	if strings.HasPrefix(scriptCoverage.URL, "inline-script-") {
		return false, "inline_script_blocked"
	}

	// 2. Exclude scripts with empty URLs (browser internals)
	if options.ExcludeEmptyURLs && scriptCoverage.URL == "" {
		return false, "empty_url"
	}

	// 3. Exclude browser extension scripts
	if options.ExcludeBrowserExt && (strings.Contains(scriptCoverage.URL, "chrome-extension://") ||
		strings.Contains(scriptCoverage.URL, "moz-extension://") ||
		strings.Contains(scriptCoverage.URL, "safari-extension://")) {
		return false, "browser_extension"
	}

	// 4. Exclude DevTools/automation framework specific function signatures
	if options.ExcludeDevTools {
		devToolsPatterns := []string{
			"functions.selectable", "functions.element", "f.toString",
			"__coverage__", "webdriver", "puppeteer", "playwright", "rod",
			"chromedriver", "seleniumwebdriver",
		}
		sourceLower := strings.ToLower(source)
		for _, pattern := range devToolsPatterns {
			if strings.Contains(sourceLower, strings.ToLower(pattern)) {
				return false, "devtools_framework"
			}
		}
	}

	// 5. Exclude very small scripts (likely browser internals)
	if len(strings.TrimSpace(source)) < options.MinScriptSize {
		return false, "too_small"
	}

	// 6. Exclude known browser internal patterns
	trimmedSource := strings.TrimSpace(source)
	browserInternalPatterns := []string{
		"console.clear()", "console.time()", "console.group()",
		"console.clear", "console.time", "console.group",
	}
	for _, pattern := range browserInternalPatterns {
		if trimmedSource == pattern || trimmedSource == pattern+";" {
			return false, "browser_internal"
		}
	}

	// 7. Exclude modern framework development tools
	if options.ExcludeFrameworkTools {
		frameworkToolPatterns := []string{
			// React DevTools and internals
			"__REACT_DEVTOOLS_GLOBAL_HOOK__", "react-devtools", "ReactDevTools",
			"__REACT_HOT_LOADER__", "react-hot-loader", "webpack-hot-middleware",
			// Vue DevTools
			"__VUE_DEVTOOLS_GLOBAL_HOOK__", "vue-devtools", "VueDevTools",
			"vue-hot-reload-api", "__VUE_HMR_RUNTIME__",
			// Angular DevTools
			"ng.probe", "ng.coreTokens", "getAllAngularRootElements",
			"@angular/core/bundles", "zone.js/bundles",
			// Build tool artifacts
			"webpack://", "webpackBootstrap", "__webpack_require__",
			"(function(module, exports, __webpack_require__)",
			"parcelRequire", "rollupPluginBabelHelpers",
			// Source map utilities
			"//# sourceMappingURL=", "//# sourceURL=",
		}
		sourceLower := strings.ToLower(source)
		urlLower := strings.ToLower(scriptCoverage.URL)
		for _, pattern := range frameworkToolPatterns {
			if strings.Contains(sourceLower, strings.ToLower(pattern)) ||
				strings.Contains(urlLower, strings.ToLower(pattern)) {
				return false, "framework_tools"
			}
		}
	}

	// 8. Exclude CDN-hosted libraries
	if options.ExcludeCDNLibraries {
		cdnPatterns := []string{
			"cdn.jsdelivr.net", "unpkg.com", "cdnjs.cloudflare.com",
			"ajax.googleapis.com", "code.jquery.com", "stackpath.bootstrapcdn.com",
			"maxcdn.bootstrapcdn.com", "use.fontawesome.com", "fonts.googleapis.com",
			"polyfill.io", "cdn.polyfill.io", "cloudflare.com/ajax/libs",
		}
		urlLower := strings.ToLower(scriptCoverage.URL)
		for _, pattern := range cdnPatterns {
			if strings.Contains(urlLower, pattern) {
				return false, "cdn_library"
			}
		}
	}

	// 9. Exclude minified/generated code detection
	if options.ExcludeMinifiedCode {
		// Check for minified indicators in URL
		urlLower := strings.ToLower(scriptCoverage.URL)
		if strings.Contains(urlLower, ".min.") || strings.Contains(urlLower, "-min.") ||
			strings.Contains(urlLower, "_min.") || strings.Contains(urlLower, "/min/") {
			return false, "minified_code"
		}

		// Check for generated code markers in source
		sourceLower := strings.ToLower(source)
		generatedCodeMarkers := []string{
			"this file was autogenerated", "do not edit", "auto-generated",
			"generated by webpack", "generated by rollup", "generated by parcel",
			"compiled by babel", "this is a generated file",
			"/* eslint-disable */", "/* tslint:disable */",
		}
		for _, marker := range generatedCodeMarkers {
			if strings.Contains(sourceLower, strings.ToLower(marker)) {
				return false, "generated_code"
			}
		}

		// Heuristic: very long lines (>200 chars) with no whitespace often indicate minification
		lines := strings.Split(source, "\n")
		for _, line := range lines[:min(5, len(lines))] { // Check first 5 lines
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 200 && !strings.Contains(trimmed, " ") && !strings.HasPrefix(trimmed, "//") {
				return false, "minified_heuristic"
			}
		}
	}

	// 10. Exclude test framework code
	if options.ExcludeTestFrameworks {
		testFrameworkPatterns := []string{
			// Jest
			"jest-runtime", "jest.fn()", "expect.extend", "__jest",
			"describe(", "test(", "it(", "expect(", "beforeEach(", "afterEach(",
			"jasmine.createSpy", "jasmine.clock", "jest/build/",
			// Mocha/Chai
			"mocha.setup", "chai.expect", "chai.assert", "should.js",
			// Jasmine
			"jasmine.getEnv()", "jasmine.DEFAULT_TIMEOUT_INTERVAL",
			// Cypress
			"cypress/", "cy.visit(", "cy.get(", "cy.click(",
			// Testing Library
			"@testing-library/", "render(", "screen.getBy",
			// QUnit
			"QUnit.test", "QUnit.module",
			// Karma
			"karma.conf", "__karma__",
		}
		sourceLower := strings.ToLower(source)
		urlLower := strings.ToLower(scriptCoverage.URL)
		for _, pattern := range testFrameworkPatterns {
			if strings.Contains(sourceLower, strings.ToLower(pattern)) ||
				strings.Contains(urlLower, strings.ToLower(pattern)) {
				return false, "test_framework"
			}
		}
	}

	// 11. Exclude high-density inline scripts (likely minified)
	if options.ExcludeHighDensityInlineScripts {
		// Check if this is an inline script
		if strings.HasPrefix(scriptCoverage.URL, "inline-script-") ||
			scriptCoverage.URL == "" ||
			strings.Contains(strings.ToLower(scriptCoverage.URL), "inline") {

			// Calculate statement density: total statements divided by number of lines
			lines := strings.Split(source, "\n")
			nonEmptyLines := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					nonEmptyLines++
				}
			}

			if nonEmptyLines > 0 {
				// Count statements using semicolons and common statement patterns
				statementCount := countJavaScriptStatements(source)
				statementsPerLine := float64(statementCount) / float64(nonEmptyLines)

				// If statements per line exceeds threshold, likely minified
				if statementsPerLine > float64(options.MaxStatementsPerLine) {
					return false, "high_density_inline"
				}
			}
		}
	}

	// 12. Exclude inline system scripts (browser-generated)
	if options.ExcludeInlineSystemScripts {
		// Check if this appears to be a browser-generated inline script
		if strings.HasPrefix(scriptCoverage.URL, "inline-script-") || scriptCoverage.URL == "" {
			// Look for system-generated content patterns
			systemPatterns := []string{
				// Browser console/devtools generated
				"console.log", "console.warn", "console.error",
				"window.chrome", "window.__REACT_DEVTOOLS",
				"window.__VUE_DEVTOOLS", "window.angular",
				// Performance monitoring
				"performance.mark", "performance.measure",
				"navigation.timing", "window.performance",
				// Browser automation detection
				"webdriver", "phantom", "selenium", "puppeteer",
				// Ad blockers and extensions
				"adblock", "ublock", "extension",
			}

			sourceLower := strings.ToLower(source)
			for _, pattern := range systemPatterns {
				if strings.Contains(sourceLower, strings.ToLower(pattern)) {
					return false, "inline_system_script"
				}
			}

			// Check for repetitive patterns (common in generated code)
			if isRepetitiveContent(source) {
				return false, "inline_system_script"
			}
		}
	}

	// 13. Check custom exclude patterns
	for _, pattern := range options.CustomExcludePatterns {
		if strings.Contains(strings.ToLower(scriptCoverage.URL), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(source), strings.ToLower(pattern)) {
			return false, "custom_exclude"
		}
	}

	return true, "application_script"
}

// min helper function for slice bounds checking
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// countJavaScriptStatements estimates the number of JavaScript statements in source code
func countJavaScriptStatements(source string) int {
	if source == "" {
		return 0
	}

	// Remove comments to avoid false positives
	source = removeJavaScriptComments(source)

	// Count semicolons (primary statement delimiter)
	semicolonCount := strings.Count(source, ";")

	// Count other statement patterns that might not end with semicolons
	statementPatterns := []string{
		"function ", "var ", "let ", "const ", "if ", "for ", "while ",
		"return ", "throw ", "try ", "catch ", "switch ", "case ",
		"break", "continue", "class ", "import ", "export ",
	}

	patternCount := 0
	sourceLower := strings.ToLower(source)
	for _, pattern := range statementPatterns {
		patternCount += strings.Count(sourceLower, pattern)
	}

	// Use the higher of the two counts as a rough estimate
	// Semicolons are usually more accurate for minified code
	if semicolonCount > patternCount {
		return semicolonCount
	}
	return patternCount
}

// removeJavaScriptComments removes single-line and multi-line comments from JavaScript source
func removeJavaScriptComments(source string) string {
	// Remove single-line comments
	lines := strings.Split(source, "\n")
	var cleanedLines []string

	for _, line := range lines {
		// Find // that's not inside a string
		inString := false
		escaped := false
		commentStart := -1

		for i, char := range line {
			if escaped {
				escaped = false
				continue
			}

			if char == '\\' && inString {
				escaped = true
				continue
			}

			if char == '"' || char == '\'' {
				inString = !inString
				continue
			}

			if !inString && i < len(line)-1 && line[i] == '/' && line[i+1] == '/' {
				commentStart = i
				break
			}
		}

		if commentStart >= 0 {
			line = line[:commentStart]
		}
		cleanedLines = append(cleanedLines, line)
	}

	result := strings.Join(cleanedLines, "\n")

	// Remove multi-line comments /* */
	for {
		start := strings.Index(result, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+2:], "*/")
		if end == -1 {
			// Unterminated comment, remove from start to end
			result = result[:start]
			break
		}
		result = result[:start] + result[start+2+end+2:]
	}

	return result
}

// isRepetitiveContent checks if content appears to be repetitive/generated
func isRepetitiveContent(source string) bool {
	if len(source) < 100 {
		return false // Too short to analyze
	}

	// Check for highly repetitive patterns
	lines := strings.Split(source, "\n")
	if len(lines) < 3 {
		return false
	}

	// Look for identical or very similar lines
	identicalLines := 0
	for i := 0; i < len(lines)-1; i++ {
		line1 := strings.TrimSpace(lines[i])
		line2 := strings.TrimSpace(lines[i+1])

		if line1 != "" && line1 == line2 {
			identicalLines++
		}
	}

	// If more than 30% of lines are identical to adjacent lines, likely repetitive
	repetitiveRatio := float64(identicalLines) / float64(len(lines))
	if repetitiveRatio > 0.6 { // More than 60% repetitive
		return true
	}

	// Check for repeated character sequences (common in generated code)
	for _, pattern := range []string{"...", "===", "!!!", "???", "000", "111"} {
		if strings.Count(source, pattern) > 10 {
			return true
		}
	}

	return false
}

// filterApplicationScriptsWithStats filters scripts and returns detailed statistics
func filterApplicationScriptsWithStats(scripts []*proto.ProfilerScriptCoverage, sources map[int]string, options CoverageFilterOptions) ([]int, FilteringStats) {
	startTime := time.Now()

	var applicationScripts []int
	stats := FilteringStats{
		TotalScripts:  len(scripts),
		FilterReasons: make(map[string]int),
	}

	for i, script := range scripts {
		source := sources[i]
		if source == "" {
			stats.FilterReasons["source_unavailable"]++
			continue
		}

		isApp, reason := isApplicationScript(script, source, options)
		stats.FilterReasons[reason]++

		if isApp {
			applicationScripts = append(applicationScripts, i)
		}
	}

	stats.ApplicationScripts = len(applicationScripts)
	stats.FilteredOut = stats.TotalScripts - stats.ApplicationScripts

	// Calculate timing metrics
	processingTime := time.Since(startTime)
	stats.ProcessingTimeMs = processingTime.Nanoseconds() / 1000000
	if stats.TotalScripts > 0 {
		stats.AverageTimePerScript = float64(stats.ProcessingTimeMs) / float64(stats.TotalScripts)
	}

	return applicationScripts, stats
}

func generateJSReport(t interface{ Fatal(args ...any) }, page *rod.Page, raw []*proto.ProfilerScriptCoverage) {
	client := page

	sourceProvider := func(index int, script *proto.ProfilerScriptCoverage) (string, error) {
		srcResp, err := proto.DebuggerGetScriptSource{ScriptID: script.ScriptID}.Call(client)
		if err != nil {
			return "", fmt.Errorf("failed to get script source for ScriptID %s: %w", script.ScriptID, err)
		}
		if srcResp.ScriptSource == "" {
			return "", fmt.Errorf("empty script source for ScriptID %s", script.ScriptID)
		}
		return srcResp.ScriptSource, nil
	}

	var filterStats FilteringStats
	outputFunc := func(format string, args ...interface{}) {
		if format == "JavaScript coverage report written to %s" {
			fmt.Printf("✅ Wrote enhanced JS coverage report (%d application scripts, %d filtered): %s\n",
				filterStats.ApplicationScripts, filterStats.FilteredOut, args[0])
		} else {
			// Skip the coverage summary as it's not needed in this output format
		}
	}

	filterStats = generateJSReportUnified(raw, sourceProvider, outputFunc)
}

func calculateCoverageMetrics(source string, ranges []*proto.ProfilerCoverageRange, functions []*proto.ProfilerFunctionCoverage) CoverageMetrics {
	sourceLen := len(source)
	lines := strings.Split(source, "\n")

	// Create coverage map
	coverage := make([]bool, sourceLen)
	for _, r := range ranges {
		if r.Count > 0 {
			for i := r.StartOffset; i < r.EndOffset && i < sourceLen; i++ {
				coverage[i] = true
			}
		}
	}

	// Calculate statements coverage (simplified as character-based)
	coveredChars := 0
	for _, covered := range coverage {
		if covered {
			coveredChars++
		}
	}

	// Calculate lines coverage
	linesCovered := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue // Skip empty lines and comments
		}

		// Check if any part of this line is covered
		lineStart := 0
		for j := 0; j < i; j++ {
			lineStart += len(lines[j]) + 1 // +1 for newline
		}
		lineEnd := lineStart + len(line)

		lineCovered := false
		for k := lineStart; k < lineEnd && k < len(coverage); k++ {
			if coverage[k] {
				lineCovered = true
				break
			}
		}
		if lineCovered {
			linesCovered++
		}
	}

	executableLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "//") {
			executableLines++
		}
	}

	// Functions coverage (count each function individually)
	functionsCovered := 0
	functionCount := len(functions)

	for _, fn := range functions {
		// Check if this function has any covered ranges
		hasCoverage := false
		for _, r := range fn.Ranges {
			if r.Count > 0 {
				hasCoverage = true
				break
			}
		}
		if hasCoverage {
			functionsCovered++
		}
	}

	return CoverageMetrics{
		Statements: CoverageStat{
			Total:   sourceLen,
			Covered: coveredChars,
			Pct:     calculatePct(coveredChars, sourceLen),
		},
		Functions: CoverageStat{
			Total:   functionCount,
			Covered: functionsCovered,
			Pct:     calculatePct(functionsCovered, functionCount),
		},
		Lines: CoverageStat{
			Total:   executableLines,
			Covered: linesCovered,
			Pct:     calculatePct(linesCovered, executableLines),
		},
	}
}

func calculatePct(covered, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}

// HTML templates as constants
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JavaScript Coverage Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism.min.css" rel="stylesheet">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-javascript.min.js"></script>
    <style>
        .coverage-high { background-color: #d4edda; }
        .coverage-medium { background-color: #fff3cd; }
        .coverage-low { background-color: #f8d7da; }
        .line-covered { background-color: #d4edda; }
        .line-uncovered { background-color: #f8d7da; }
        .line-number { background-color: #f8f9fa; border-right: 1px solid #dee2e6; }
    </style>
</head>
<body class="bg-gray-50 text-gray-900">
    <div class="container mx-auto px-4 py-8">
        <div class="mb-8">
            <h1 class="text-3xl font-bold text-gray-900 mb-2">JavaScript Coverage Report</h1>
            <p class="text-gray-600">Generated on {{.Timestamp}}</p>
            <div class="mt-3 flex flex-wrap gap-4 text-sm">
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                    📁 {{.FilterStats.ApplicationScripts}} Application Scripts
                </span>
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                    🚫 {{.FilterStats.FilteredOut}} Scripts Filtered
                </span>
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                    📊 {{.FilterStats.TotalScripts}} Total Scripts Analyzed
                </span>
            </div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">{{.SummaryCards}}</div>
        {{.FilteringStats}}
        <div class="bg-white rounded-lg shadow-md mb-8">
            <div class="px-6 py-4 border-b border-gray-200">
                <h2 class="text-xl font-semibold text-gray-900">File Coverage</h2>
            </div>
            <div class="overflow-x-auto">
                <table class="min-w-full">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Statements</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Functions</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Lines</th>
                        </tr>
                    </thead>
                    <tbody class="bg-white divide-y divide-gray-200">{{.FileTable}}</tbody>
                </table>
            </div>
        </div>
        {{.FileDetails}}
    </div>
    <script>
        function toggleFile(fileId) {
            const element = document.getElementById(fileId);
            element.classList.toggle('hidden');
        }
        Prism.highlightAll();
    </script>
</body>
</html>`

type htmlData struct {
	Timestamp      string
	FilterStats    FilteringStats
	SummaryCards   string
	FilteringStats string
	FileTable      string
	FileDetails    string
}

func generateIstanbulStyleHTML(entries []FileEntry, totalMetrics CoverageMetrics, filterStats FilteringStats) string {
	tmpl := template.Must(template.New("coverage").Parse(htmlTemplate))

	data := htmlData{
		Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		FilterStats:    filterStats,
		SummaryCards:   generateSummaryCards(totalMetrics),
		FilteringStats: generateFilteringStats(filterStats),
		FileTable:      generateFileTable(entries),
		FileDetails:    generateFileDetails(entries),
	}

	var buf strings.Builder
	tmpl.Execute(&buf, data)
	return buf.String()
}

const filteringStatsTemplate = `
<div class="bg-white rounded-lg shadow-md mb-8">
    <div class="px-6 py-4 border-b border-gray-200">
        <h2 class="text-xl font-semibold text-gray-900 flex items-center">
            🔍 Filtering Statistics
            <span class="ml-2 text-sm font-normal text-gray-500">
                (Processing time: {{printf "%.1f" .ProcessingTimeMs}}ms, avg: {{printf "%.2f" .AverageTimePerScript}}ms per script)
            </span>
        </h2>
    </div>
    <div class="p-6">
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">{{range .Reasons}}
            <div class="bg-gray-50 rounded-lg p-4">
                <div class="flex items-center justify-between mb-2">
                    <span class="text-sm font-medium text-gray-700">{{.Icon}} {{.Description}}</span>
                    <span class="text-lg font-bold text-gray-900">{{.Count}}</span>
                </div>
                <div class="text-xs text-gray-500 mb-2">{{printf "%.1f" .Percentage}}% of scripts</div>
                <div class="bg-gray-200 rounded-full h-2">
                    <div class="bg-blue-600 h-2 rounded-full" style="width: {{printf "%.1f" .Percentage}}%"></div>
                </div>
            </div>{{end}}
        </div>
    </div>
</div>`

type reasonData struct {
	Icon        string
	Description string
	Count       int
	Percentage  float64
}

type filteringData struct {
	ProcessingTimeMs     float64
	AverageTimePerScript float64
	Reasons              []reasonData
}

func generateFilteringStats(stats FilteringStats) string {
	if len(stats.FilterReasons) == 0 {
		return ""
	}

	var reasons []reasonData
	for reason, count := range stats.FilterReasons {
		icon, description := getFilterReasonDetails(reason)
		reasons = append(reasons, reasonData{
			Icon:        icon,
			Description: description,
			Count:       count,
			Percentage:  float64(count) / float64(stats.TotalScripts) * 100,
		})
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i].Count > reasons[j].Count })

	tmpl := template.Must(template.New("filtering").Parse(filteringStatsTemplate))
	data := filteringData{
		ProcessingTimeMs:     float64(stats.ProcessingTimeMs),
		AverageTimePerScript: stats.AverageTimePerScript,
		Reasons:              reasons,
	}

	var buf strings.Builder
	tmpl.Execute(&buf, data)
	return buf.String()
}

func getFilterReasonDetails(reason string) (string, string) {
	switch reason {
	case "application_script":
		return "✅", "Application Scripts"
	case "empty_url":
		return "🚫", "Empty URLs (Browser Internals)"
	case "browser_extension":
		return "🧩", "Browser Extensions"
	case "devtools_framework":
		return "🔧", "DevTools & Automation"
	case "framework_tools":
		return "⚛️", "Framework Development Tools"
	case "cdn_library":
		return "🌐", "CDN Libraries"
	case "minified_code":
		return "📦", "Minified Code"
	case "generated_code":
		return "🤖", "Auto-Generated Code"
	case "minified_heuristic":
		return "🔍", "Minified (Heuristic)"
	case "test_framework":
		return "🧪", "Test Frameworks"
	case "browser_internal":
		return "🔒", "Browser Internal Scripts"
	case "too_small":
		return "📏", "Scripts Too Small"
	case "source_unavailable":
		return "❌", "Source Unavailable"
	case "custom_exclude":
		return "⚙️", "Custom Exclusions"
	case "custom_include":
		return "✨", "Custom Inclusions"
	case "high_density_inline":
		return "📊", "High-Density Inline Scripts"
	case "inline_system_script":
		return "🔧", "Inline System Scripts"
	case "inline_script_blocked":
		return "🚫", "Inline Scripts (All Blocked)"
	default:
		return "❓", reason
	}
}

const summaryCardsTemplate = `{{range .}}
<div class="bg-white rounded-lg shadow-md p-6 {{.BgColor}}">
    <div class="flex items-center justify-between">
        <div>
            <p class="text-sm font-medium text-gray-600">{{.Icon}} {{.Title}}</p>
            <p class="text-2xl font-bold text-gray-900">{{printf "%.1f" .Pct}}%</p>
            <p class="text-xs text-gray-500">{{.Covered}}/{{.Total}} covered</p>
        </div>
        <div class="text-2xl">{{.Icon}}</div>
    </div>
    <div class="mt-4">
        <div class="bg-gray-200 rounded-full h-2">
            <div class="bg-blue-600 h-2 rounded-full" style="width: {{printf "%.1f" .Pct}}%"></div>
        </div>
    </div>
</div>{{end}}`

type cardData struct {
	Title   string
	Icon    string
	Pct     float64
	Covered int
	Total   int
	BgColor string
}

func generateSummaryCards(metrics CoverageMetrics) string {
	cards := []cardData{
		{"Statements", "📊", metrics.Statements.Pct, metrics.Statements.Covered, metrics.Statements.Total, getCoverageColor(metrics.Statements.Pct)},
		{"Functions", "⚡", metrics.Functions.Pct, metrics.Functions.Covered, metrics.Functions.Total, getCoverageColor(metrics.Functions.Pct)},
		{"Lines", "📝", metrics.Lines.Pct, metrics.Lines.Covered, metrics.Lines.Total, getCoverageColor(metrics.Lines.Pct)},
		{"Overall", "🎯", (metrics.Statements.Pct + metrics.Functions.Pct + metrics.Lines.Pct) / 3, 0, 0, getCoverageColor((metrics.Statements.Pct + metrics.Functions.Pct + metrics.Lines.Pct) / 3)},
	}

	tmpl := template.Must(template.New("cards").Parse(summaryCardsTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, cards)
	return buf.String()
}

const fileTableTemplate = `{{range .}}
<tr class="hover:bg-gray-50 cursor-pointer" onclick="toggleFile('file-{{.ScriptID}}')">
    <td class="px-6 py-4 text-sm text-blue-600 hover:text-blue-800">{{.FileName}}</td>
    <td class="px-6 py-4 text-sm text-gray-900">
        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.StmtBadgeColor}}">
            {{printf "%.1f" .Metrics.Statements.Pct}}% ({{.Metrics.Statements.Covered}}/{{.Metrics.Statements.Total}})
        </span>
    </td>
    <td class="px-6 py-4 text-sm text-gray-900">
        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.FuncBadgeColor}}">
            {{printf "%.1f" .Metrics.Functions.Pct}}% ({{.Metrics.Functions.Covered}}/{{.Metrics.Functions.Total}})
        </span>
    </td>
    <td class="px-6 py-4 text-sm text-gray-900">
        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.LinesBadgeColor}}">
            {{printf "%.1f" .Metrics.Lines.Pct}}% ({{.Metrics.Lines.Covered}}/{{.Metrics.Lines.Total}})
        </span>
    </td>
</tr>{{end}}`

const fileDetailsTemplate = `{{range .}}
<div id="file-{{.ScriptID}}" class="hidden bg-white rounded-lg shadow-md mb-6">
    <div class="px-6 py-4 border-b border-gray-200">
        <h3 class="text-lg font-semibold text-gray-900">{{.FileName}}</h3>
        <div class="mt-2 flex space-x-4 text-sm text-gray-600">
            <span>Statements: {{printf "%.1f" .Metrics.Statements.Pct}}%</span>
            <span>Functions: {{printf "%.1f" .Metrics.Functions.Pct}}%</span>
            <span>Lines: {{printf "%.1f" .Metrics.Lines.Pct}}%</span>
        </div>
    </div>
    <div class="p-0">
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <tbody>{{.SourceLines}}</tbody>
            </table>
        </div>
    </div>
</div>{{end}}`

type fileData struct {
	ScriptID        string
	FileName        string
	Metrics         CoverageMetrics
	StmtBadgeColor  string
	FuncBadgeColor  string
	LinesBadgeColor string
	SourceLines     string
}

func generateFileTable(entries []FileEntry) string {
	var files []fileData
	for _, entry := range entries {
		fileName := entry.URL
		if fileName == "" {
			fileName = fmt.Sprintf("Script %s", entry.ScriptID)
		}
		files = append(files, fileData{
			ScriptID:        string(entry.ScriptID),
			FileName:        fileName,
			Metrics:         entry.Metrics,
			StmtBadgeColor:  getCoverageBadgeColor(entry.Metrics.Statements.Pct),
			FuncBadgeColor:  getCoverageBadgeColor(entry.Metrics.Functions.Pct),
			LinesBadgeColor: getCoverageBadgeColor(entry.Metrics.Lines.Pct),
		})
	}

	tmpl := template.Must(template.New("fileTable").Parse(fileTableTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, files)
	return buf.String()
}

func generateFileDetails(entries []FileEntry) string {
	var files []fileData
	for _, entry := range entries {
		fileName := entry.URL
		if fileName == "" {
			fileName = fmt.Sprintf("Script %s", entry.ScriptID)
		}
		files = append(files, fileData{
			ScriptID:    string(entry.ScriptID),
			FileName:    fileName,
			Metrics:     entry.Metrics,
			SourceLines: generateSourceLines(entry),
		})
	}

	tmpl := template.Must(template.New("fileDetails").Parse(fileDetailsTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, files)
	return buf.String()
}

const sourceLineTemplate = `{{range .}}
<tr class="{{.LineClass}}">
    <td class="line-number px-4 py-1 text-right text-gray-500 select-none w-16">{{.LineNumber}}</td>
    <td class="px-4 py-1">
        <pre class="whitespace-pre-wrap font-mono text-xs"><code class="language-javascript">{{.EscapedLine}}</code></pre>
    </td>
</tr>{{end}}`

type lineData struct {
	LineNumber  int
	LineClass   string
	EscapedLine string
}

func generateSourceLines(entry FileEntry) string {
	sourceLen := len(entry.Source)
	coverage := make([]bool, sourceLen)
	for _, r := range entry.Ranges {
		if r.Count > 0 {
			for i := r.StartOffset; i < r.EndOffset && i < sourceLen; i++ {
				coverage[i] = true
			}
		}
	}

	var lines []lineData
	for lineNum, line := range entry.Lines {
		lineStart := 0
		for i := 0; i < lineNum; i++ {
			lineStart += len(entry.Lines[i]) + 1
		}
		lineEnd := lineStart + len(line)

		lineClass := ""
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			lineCovered := false
			for k := lineStart; k < lineEnd && k < len(coverage); k++ {
				if coverage[k] {
					lineCovered = true
					break
				}
			}
			if lineCovered {
				lineClass = "line-covered"
			} else {
				lineClass = "line-uncovered"
			}
		}

		lines = append(lines, lineData{
			LineNumber:  lineNum + 1,
			LineClass:   lineClass,
			EscapedLine: strings.Replace(strings.Replace(line, "<", "&lt;", -1), ">", "&gt;", -1),
		})
	}

	tmpl := template.Must(template.New("sourceLines").Parse(sourceLineTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, lines)
	return buf.String()
}

// Consolidated color helper functions
func getCoverageColor(pct float64) string {
	switch {
	case pct >= 80:
		return "coverage-high"
	case pct >= 60:
		return "coverage-medium"
	default:
		return "coverage-low"
	}
}

func getCoverageBadgeColor(pct float64) string {
	switch {
	case pct >= 80:
		return "bg-green-100 text-green-800"
	case pct >= 60:
		return "bg-yellow-100 text-yellow-800"
	default:
		return "bg-red-100 text-red-800"
	}
}

func generateCoverageIndex(goPct, jsPct float64) {
	content := fmt.Sprintf(`<!DOCTYPE html>
<html><head><title>Unified Coverage Report</title></head>
<body>
	<h1>Unified Coverage Report</h1>
	<h2>Coverage Summary</h2>
	<p>Go Coverage: %.1f%%</p>
	<p>JavaScript Coverage: %.1f%%</p>
	<ul>
		<li><a href="go-cover.html">✅ Go Coverage Report</a></li>
		<li><a href="js-coverage.html">✅ JavaScript Coverage Report</a></li>
		<li><a href="screenshot-page.png">🖼️ Screenshot - Initial</a></li>
		<li><a href="screenshot-after-click.png">🖼️ Screenshot - After Copy Click</a></li>
	</ul>
</body></html>`, goPct, jsPct)
	_ = os.WriteFile(indexHTML, []byte(content), 0644)
}

func computeJavaScriptCoverage(raw []*proto.ProfilerScriptCoverage) float64 {
	var total, covered int

	for _, script := range raw {
		for _, fn := range script.Functions {
			for _, r := range fn.Ranges {
				length := r.EndOffset - r.StartOffset
				if length <= 0 {
					continue
				}
				total += length
				if r.Count > 0 {
					covered += length
				}
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}

func computeGoCoveragePercent(t *testing.T) float64 {
	cmd := exec.Command("go", "tool", "cover", "-func=../"+goCoverRaw)
	out, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to compute go coverage: %v", err)
		return 0
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				pctStr := strings.TrimSuffix(parts[len(parts)-1], "%")
				if pct, err := strconv.ParseFloat(pctStr, 64); err == nil {
					return pct
				}
			}
		}
	}
	return 0
}

func generateGoCoverHTML(t *testing.T) {
	if _, err := os.Stat("../" + goCoverRaw); os.IsNotExist(err) {
		fmt.Println("⚠️  Skipping go-cover.html: go-coverage.out not found")
		return
	}
	cmd := exec.Command("go", "tool", "cover", "-html=../"+goCoverRaw, "-o", goCoverHTML)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate go cover html: %v\n%s", err, stderr.String())
	}
	fmt.Println("✅ Generated:", goCoverHTML)
}

// createTestServer creates an embedded HTTP server for testing
func createTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Serve the roadmap page with the required elements
	mux.HandleFunc("/roadmap", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Roadmap</title>
	<script>
		// JavaScript for coverage collection
		function copyToClipboard() {
			const btn = document.getElementById('copy-all-btn');
			btn.textContent = '✅ Copied';
			btn.style.backgroundColor = '#28a745';
			console.log('Button clicked for coverage test');
		}
		
		// Add some more JavaScript for coverage
		function calculateProgress() {
			const items = document.querySelectorAll('.progress-item');
			let completed = 0;
			items.forEach(item => {
				if (item.classList.contains('completed')) {
					completed++;
				}
			});
			return completed / items.length * 100;
		}
		
		// Initialize page
		document.addEventListener('DOMContentLoaded', function() {
			console.log('Page loaded');
			const progress = calculateProgress();
			console.log('Progress:', progress + '%');
		});
	</script>
	<style>
		body { font-family: Arial, sans-serif; padding: 20px; }
		.progress-item { margin: 10px 0; padding: 10px; border: 1px solid #ccc; }
		.completed { background-color: #d4edda; }
		#copy-all-btn { padding: 10px 20px; background: #007bff; color: white; border: none; cursor: pointer; }
	</style>
</head>
<body>
	<h1>Test Roadmap</h1>
	<div class="progress-item completed">✅ Framework Foundation</div>
	<div class="progress-item completed">✅ Browser Integration</div>
	<div class="progress-item">⏳ Advanced Features</div>
	<div class="progress-item">⏳ Documentation</div>
	
	<button id="copy-all-btn" onclick="copyToClipboard()">Copy All</button>
	
	<script>
		// More JavaScript for better coverage
		setTimeout(() => {
			console.log('Delayed execution for coverage');
		}, 100);
	</script>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	return httptest.NewServer(mux)
}
