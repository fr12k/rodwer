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
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
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

	// Launch browser with --no-sandbox
	path := launcher.New().
		Headless(true).
		Leakless(false). // set to false to avoid SIGTRAP crash in CI
		NoSandbox(true). // this sets --no-sandbox
		MustLaunch()

	browser := rod.New().ControlURL(path).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(testServerURL)
	defer page.MustClose()

	// Enable Debugger and Profiler
	must(proto.DebuggerEnable{}.Call(page))
	must(nil, proto.ProfilerEnable{}.Call(page))
	must(proto.ProfilerStartPreciseCoverage{
		CallCount: true,
		Detailed:  true,
	}.Call(page))

	time.Sleep(1 * time.Second)

	// Screenshots
	page.MustScreenshot(screenshot1)

	// Click the button and verify it changes
	btn := page.MustElement("#copy-all-btn")
	btn.MustClick()

	// Wait for JavaScript to execute and verify the button text changed
	page.MustWaitStable()

	// Verify button text changed (more robust approach)
	btnText := btn.MustText()
	require.Contains(t, btnText, "Copied", "Button text should contain 'Copied' after click")

	page.MustScreenshot(screenshot2)

	// JS Coverage snapshot
	result, err := proto.ProfilerTakePreciseCoverage{}.Call(page)
	require.NoError(t, err)
	_ = proto.ProfilerStopPreciseCoverage{}.Call(page)

	b, _ := json.MarshalIndent(result.Result, "", "  ")
	require.NoError(t, os.WriteFile(jsCoverage, b, 0644))

	generateJSReport(t, page, result.Result)

	jsPct := computeJavaScriptCoverage(result.Result)
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

type OldCoverageEntry struct {
	ScriptID proto.RuntimeScriptID
	URL      string
	Source   string
	Ranges   []*proto.ProfilerCoverageRange
}

func generateJSReport(t interface{ Fatal(args ...any) }, page *rod.Page, raw []*proto.ProfilerScriptCoverage) {
	client := page

	entries := make([]OldCoverageEntry, 0, len(raw))
	for _, r := range raw {
		srcResp, err := proto.DebuggerGetScriptSource{ScriptID: r.ScriptID}.Call(client)
		if err != nil || srcResp.ScriptSource == "" {
			continue
		}
		allRanges := []*proto.ProfilerCoverageRange{}
		for _, fn := range r.Functions {
			allRanges = append(allRanges, fn.Ranges...)
		}
		entries = append(entries, OldCoverageEntry{
			ScriptID: r.ScriptID,
			URL:      r.URL,
			Source:   srcResp.ScriptSource,
			Ranges:   allRanges,
		})
	}

	// Generate HTML
	sb := &strings.Builder{}
	sb.WriteString(`<html><head><style>
    .hit { background: #cfc } .miss { background: #fcc }
    pre { white-space: pre-wrap; font-family: monospace; }
</style></head><body>`)
	sort.Slice(entries, func(i, j int) bool { return entries[i].URL < entries[j].URL })

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("<h2>%s</h2><pre>", e.URL))
		src := e.Source
		marks := make([]rune, len(src))
		for i := range marks {
			marks[i] = ' '
		}
		for _, r := range e.Ranges {
			flag := 'h'
			if r.Count == 0 {
				flag = 'm'
			}
			for i := r.StartOffset; i < r.EndOffset && i < len(marks); i++ {
				marks[i] = flag
			}
		}
		for i, ch := range src {
			switch marks[i] {
			case 'h':
				sb.WriteString("<span class=\"hit\">")
				sb.WriteRune(ch)
				sb.WriteString("</span>")
			case 'm':
				sb.WriteString("<span class=\"miss\">")
				sb.WriteRune(ch)
				sb.WriteString("</span>")
			default:
				sb.WriteRune(ch)
			}
		}
		sb.WriteString("</pre>")
	}

	_ = os.WriteFile(jsHTML, []byte(sb.String()), 0644)
	fmt.Println("✅ Wrote JS coverage:", jsHTML)
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
