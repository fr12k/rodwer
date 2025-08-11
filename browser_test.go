package rodwer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Use constants from constants.go
const (
	coverageDir = CoverageDir
	jsCoverage  = JSCoverageFile
	jsHTML      = JSCoverageHTML
	goCoverHTML = GoCoverageHTML
	goCoverRaw  = GoCoverageRaw
	screenshot1 = ScreenshotInitial
	screenshot2 = ScreenshotAfterClick
	indexHTML   = CoverageIndexHTML
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
	testServer, cleanup := NewTestServer()
	defer cleanup()

	// Get the test server URL
	testServerURL := testServer.URL + RoadmapPath

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
	time.Sleep(DOMContentLoadedDelay)

	// Take screenshot before interaction
	err = page.ScreenshotToFile(screenshot1)
	require.NoError(t, err)

	// Click the button and verify it changes
	btn, err := page.Element("#copy-all-btn")
	require.NoError(t, err)
	require.NoError(t, btn.Click())

	// Wait a bit for async JavaScript (setTimeout) to execute
	time.Sleep(AsyncJavaScriptDelay)

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

	// Use the new coverage reporter
	reporter := NewCoverageReporter()
	reporter.SetDebugMode(true)
	err = reporter.GenerateReport(coverageEntries, indexHTML)
	require.NoError(t, err)

	// Save raw coverage data for compatibility
	rawData := make([]*proto.ProfilerScriptCoverage, 0)
	for i, entry := range coverageEntries {
		scriptCov := &proto.ProfilerScriptCoverage{
			ScriptID: proto.RuntimeScriptID(fmt.Sprintf("script-%d", i)),
			URL:      entry.URL,
		}
		rawData = append(rawData, scriptCov)
	}
	b, _ := json.MarshalIndent(rawData, "", "  ")
	require.NoError(t, os.WriteFile(jsCoverage, b, 0644))
}

// Deprecated: moved to coverage_report.go
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

// Deprecated: moved to coverage_report.go
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

// Deprecated: moved to types.go or coverage_report.go

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
