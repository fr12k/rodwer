package rodwer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// FrameworkTestSuite demonstrates the complete API design for the rodwer framework
// Following TDD principles - these tests define our desired interface before implementation
type FrameworkTestSuite struct {
	suite.Suite
	browser   *Browser
	cleanupFn func()
}

func (s *FrameworkTestSuite) SetupSuite() {
	// This will fail initially (red phase) until we implement the framework
	browser, cleanup, err := NewTestBrowser()
	s.Require().NoError(err, "Failed to create test browser")
	s.browser = browser
	s.cleanupFn = cleanup
}

func (s *FrameworkTestSuite) TearDownSuite() {
	if s.cleanupFn != nil {
		s.cleanupFn()
	}
}

func (s *FrameworkTestSuite) TestBrowserCreation() {
	tests := []struct {
		name    string
		options BrowserOptions
		wantErr bool
	}{
		{
			name: "default browser creation",
			options: BrowserOptions{
				Headless: true,
			},
			wantErr: false,
		},
		{
			name: "browser with custom viewport",
			options: BrowserOptions{
				Headless: true,
				Viewport: &Viewport{
					Width:  1920,
					Height: 1080,
				},
			},
			wantErr: false,
		},
		{
			name: "browser with devtools enabled",
			options: BrowserOptions{
				Headless: false,
				DevTools: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			browser, err := NewBrowser(tt.options)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.NotNil(browser)

			// Clean up
			if browser != nil {
				s.NoError(browser.Close())
			}
		})
	}
}

func (s *FrameworkTestSuite) TestPageNavigation() {
	// Create internal test server for this test
	testServer, cleanup := NewTestServer()
	defer cleanup()

	page, err := s.browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	tests := []struct {
		name     string
		url      string
		wantErr  bool
		validate func(*testing.T, *Page)
	}{
		{
			name:    "navigate to valid URL",
			url:     testServer.URL,
			wantErr: false,
			validate: func(t *testing.T, p *Page) {
				title, err := p.Title()
				require.NoError(t, err)
				assert.NotEmpty(t, title)
			},
		},
		{
			name:    "navigate to invalid URL",
			url:     "invalid://url",
			wantErr: true,
		},
		{
			name:    "navigate with timeout",
			url:     testServer.URL + "/delay/5",
			wantErr: false,
			validate: func(t *testing.T, p *Page) {
				// This should work with proper timeout handling
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err := p.NavigateWithContext(ctx, testServer.URL+"/delay/1")
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := page.Navigate(tt.url)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)

			if tt.validate != nil {
				tt.validate(s.T(), page)
			}
		})
	}
}

func (s *FrameworkTestSuite) TestElementSelection() {
	page, err := s.browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	// Navigate to a test page (this will fail until we implement)
	err = page.Navigate("data:text/html,<html><body><h1 id='title'>Test Page</h1><button class='btn'>Click Me</button></body></html>")
	s.Require().NoError(err)

	tests := []struct {
		name     string
		selector string
		method   string
		wantErr  bool
		validate func(*testing.T, Element)
	}{
		{
			name:     "select by ID",
			selector: "#title",
			method:   "ID",
			wantErr:  false,
			validate: func(t *testing.T, el Element) {
				text, err := el.Text()
				require.NoError(t, err)
				assert.Equal(t, "Test Page", text)
			},
		},
		{
			name:     "select by class",
			selector: ".btn",
			method:   "Class",
			wantErr:  false,
			validate: func(t *testing.T, el Element) {
				text, err := el.Text()
				require.NoError(t, err)
				assert.Equal(t, "Click Me", text)
			},
		},
		// {
		// 	name:     "select non-existent element",
		// 	selector: "#nonexistent",
		// 	method:   "ID",
		// 	wantErr:  true,
		// },
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var element Element
			var err error

			switch tt.method {
			case "ID":
				element, err = page.Element(tt.selector)
			case "Class":
				element, err = page.Element(tt.selector)
			default:
				s.Fail("Unknown selection method")
				return
			}

			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.NotNil(element)

			if tt.validate != nil {
				tt.validate(s.T(), element)
			}
		})
	}
}

func (s *FrameworkTestSuite) TestElementInteraction() {
	page, err := s.browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	// Create a test page with interactive elements
	html := `
	<html>
	<body>
		<input id="text-input" type="text" placeholder="Enter text">
		<button id="submit-btn" onclick="this.textContent='Clicked!'">Submit</button>
		<div id="result"></div>
		<script>
			document.getElementById('submit-btn').addEventListener('click', function() {
				const input = document.getElementById('text-input');
				const result = document.getElementById('result');
				result.textContent = 'Input: ' + input.value;
			});
		</script>
	</body>
	</html>`

	err = page.Navigate("data:text/html," + html)
	s.Require().NoError(err)

	// Test input typing
	s.Run("type into input field", func() {
		input, err := page.Element("#text-input")
		s.Require().NoError(err)

		err = input.Type("Hello, World!")
		s.Require().NoError(err)

		value, err := input.Value()
		s.Require().NoError(err)
		s.Equal("Hello, World!", value)
	})

	// Test button clicking
	s.Run("click button", func() {
		button, err := page.Element("#submit-btn")
		s.Require().NoError(err)

		err = button.Click()
		s.Require().NoError(err)

		// Verify the click had an effect
		text, err := button.Text()
		s.Require().NoError(err)
		s.Equal("Clicked!", text)
	})

	// Test waiting for elements
	s.Run("wait for element to appear", func() {
		result, err := page.WaitForElement("#result", 5*time.Second)
		s.Require().NoError(err)

		text, err := result.Text()
		s.Require().NoError(err)
		s.Contains(text, "Input: Hello, World!")
	})
}

func (s *FrameworkTestSuite) TestScreenshotCapture() {
	page, err := s.browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	err = page.Navigate("data:text/html,<html><body><h1>Screenshot Test</h1></body></html>")
	s.Require().NoError(err)

	tests := []struct {
		name    string
		options ScreenshotOptions
		wantErr bool
	}{
		{
			name: "full page screenshot",
			options: ScreenshotOptions{
				FullPage: true,
				Format:   "png",
			},
			wantErr: false,
		},
		{
			name: "viewport screenshot",
			options: ScreenshotOptions{
				FullPage: false,
				Format:   "jpeg",
				Quality:  80,
			},
			wantErr: false,
		},
		{
			name: "element screenshot",
			options: ScreenshotOptions{
				Selector: "h1",
				Format:   "png",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			data, err := page.Screenshot(tt.options)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.NotEmpty(data, "Screenshot data should not be empty")
			s.Greater(len(data), 100, "Screenshot should contain meaningful data")
		})
	}
}

func (s *FrameworkTestSuite) TestScreenshotToFile() {
	page, err := s.browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	err = page.Navigate("data:text/html,<html><body><h1>ScreenshotToFile Test</h1><p id='test-element'>Test Element</p></body></html>")
	s.Require().NoError(err)

	// Test page screenshot to file with default options
	testDir := "test_screenshots"
	defer os.RemoveAll(testDir) // Clean up after test

	// Test ScreenshotSimpleToFile
	err = page.ScreenshotSimpleToFile(filepath.Join(testDir, "simple_test.png"))
	s.Require().NoError(err)
	s.FileExists(filepath.Join(testDir, "simple_test.png"))

	// Test ScreenshotToFile with custom options
	err = page.ScreenshotToFile(filepath.Join(testDir, "custom_test.jpg"), ScreenshotOptions{
		Format:  "jpeg",
		Quality: 80,
	})
	s.Require().NoError(err)
	s.FileExists(filepath.Join(testDir, "custom_test.jpg"))

	// Test element screenshot to file
	element, err := page.Element("#test-element")
	s.Require().NoError(err)
	err = element.ScreenshotToFile(filepath.Join(testDir, "element_test.png"))
	s.Require().NoError(err)
	s.FileExists(filepath.Join(testDir, "element_test.png"))

	// Test error cases
	err = page.ScreenshotToFile("", ScreenshotOptions{})
	s.Error(err, "Should error with empty file path")
}

func (s *FrameworkTestSuite) TestCoverageCollection() {
	page, err := s.browser.NewPage()
	s.Require().NoError(err)
	defer page.Close()

	// Enable coverage collection
	err = page.StartJSCoverage()
	s.Require().NoError(err)

	// Navigate to a page with JavaScript
	html := `
	<html>
	<body>
		<script>
			function testFunction() {
				return "test";
			}
			
			function unusedFunction() {
				return "unused";
			}
			
			// Call one function to create coverage data
			testFunction();
		</script>
	</body>
	</html>`

	err = page.Navigate("data:text/html," + html)
	s.Require().NoError(err)

	// Collect coverage data
	coverage, err := page.StopJSCoverage()
	s.Require().NoError(err)
	s.NotNil(coverage)
	s.Greater(len(coverage), 0, "Should have coverage data")

	// Verify coverage data structure
	for _, entry := range coverage {
		s.NotEmpty(entry.URL, "Coverage entry should have URL")
		s.NotEmpty(entry.Source, "Coverage entry should have source")
		s.NotNil(entry.Ranges, "Coverage entry should have ranges")
	}
}

func (s *FrameworkTestSuite) TestMultiplePages() {
	// Test creating and managing multiple pages
	var pages []*Page

	for i := 0; i < 3; i++ {
		page, err := s.browser.NewPage()
		s.Require().NoError(err)
		pages = append(pages, page)

		// Navigate each page to different content
		html := `<html><body><h1>Page ` + string(rune('A'+i)) + `</h1></body></html>`
		err = page.Navigate("data:text/html," + html)
		s.Require().NoError(err)
	}

	// Verify all pages are accessible
	s.Len(pages, 3)

	// Get browser pages and verify count
	allPages, err := s.browser.Pages()
	s.Require().NoError(err)
	s.GreaterOrEqual(len(allPages), 3, "Browser should track all created pages")

	// Clean up all pages
	for _, page := range pages {
		err := page.Close()
		s.NoError(err)
	}
}

// Run the framework test suite
func TestFrameworkSuite(t *testing.T) {
	suite.Run(t, new(FrameworkTestSuite))
}

// Table-driven tests for browser options validation
func TestBrowserOptionsValidation(t *testing.T) {
	t.Parallel() // Independent validation tests can run in parallel

	tests := []struct {
		name    string
		options BrowserOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid default options",
			options: BrowserOptions{
				Headless: true,
			},
			wantErr: false,
		},
		{
			name: "invalid viewport width",
			options: BrowserOptions{
				Headless: true,
				Viewport: &Viewport{
					Width:  -1,
					Height: 1080,
				},
			},
			wantErr: true,
			errMsg:  "viewport width must be positive",
		},
		{
			name: "invalid viewport height",
			options: BrowserOptions{
				Headless: true,
				Viewport: &Viewport{
					Width:  1920,
					Height: 0,
				},
			},
			wantErr: true,
			errMsg:  "viewport height must be positive",
		},
		{
			name: "invalid user agent",
			options: BrowserOptions{
				Headless:  true,
				UserAgent: "", // Empty user agent should be invalid
			},
			wantErr: false, // Empty is actually valid, will use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBrowserOptions(tt.options)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test helper functions and utilities
func TestTestHelpers(t *testing.T) {
	t.Parallel() // Helper tests are independent

	t.Run("test server creation", func(t *testing.T) {
		server, cleanup := NewTestServer()
		defer cleanup()

		assert.NotNil(t, server)
		assert.NotEmpty(t, server.URL)

		// Test that server is accessible
		resp, err := server.Client().Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("test page factory", func(t *testing.T) {
		page, cleanup := NewTestPage(TestPageOptions{
			HTML: "<html><body><h1>Test</h1></body></html>",
		})
		defer cleanup()

		assert.NotNil(t, page)

		title, err := page.Title()
		require.NoError(t, err)
		assert.NotEmpty(t, title)
	})
}

// Benchmark tests for performance validation
func BenchmarkBrowserCreation(b *testing.B) {
	options := BrowserOptions{Headless: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		browser, err := NewBrowser(options)
		if err != nil {
			b.Fatal(err)
		}
		browser.Close()
	}
}

func BenchmarkPageNavigation(b *testing.B) {
	browser, cleanup, err := NewTestBrowser()
	if err != nil {
		b.Fatal(err)
	}
	defer cleanup()

	page, err := browser.NewPage()
	if err != nil {
		b.Fatal(err)
	}
	defer page.Close()

	url := "data:text/html,<html><body><h1>Benchmark Test</h1></body></html>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := page.Navigate(url)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkElementSelection(b *testing.B) {
	browser, cleanup, err := NewTestBrowser()
	if err != nil {
		b.Fatal(err)
	}
	defer cleanup()

	page, err := browser.NewPage()
	if err != nil {
		b.Fatal(err)
	}
	defer page.Close()

	html := `<html><body><div id="test">Test Element</div></body></html>`
	err = page.Navigate("data:text/html," + html)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := page.Element("#test")
		if err != nil {
			b.Fatal(err)
		}
	}
}
