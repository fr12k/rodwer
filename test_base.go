package rodwer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestSuiteBase provides common functionality for browser test suites
type TestSuiteBase struct {
	suite.Suite
	browser    *Browser
	testServer *TestServer
	cleanup    func()
}

// SetupTest initializes browser and test server for each test
func (s *TestSuiteBase) SetupTest() {
	// Create test server
	testServer, cleanup := NewTestServer()
	s.testServer = testServer

	// Create browser
	browser, err := NewBrowser(DefaultBrowserOptions())
	s.Require().NoError(err, "Failed to create browser")
	s.browser = browser

	// Combine cleanup functions
	s.cleanup = func() {
		if browser != nil {
			browser.Close()
		}
		cleanup()
	}
}

// TearDownTest cleans up resources after each test
func (s *TestSuiteBase) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// Browser returns the test browser instance
func (s *TestSuiteBase) Browser() *Browser {
	return s.browser
}

// TestServer returns the test server instance
func (s *TestSuiteBase) TestServer() *TestServer {
	return s.testServer
}

// NewPage creates a new page with common setup
func (s *TestSuiteBase) NewPage() *Page {
	page, err := s.browser.NewPage()
	s.Require().NoError(err, "Failed to create page")
	return page
}

// NavigateToHTML navigates to a page with the specified HTML
func (s *TestSuiteBase) NavigateToHTML(page *Page, html string) {
	dataURL := DataURLHTML(html)
	err := page.Navigate(dataURL)
	s.Require().NoError(err, "Failed to navigate to HTML")
}

// WaitForElement waits for an element and requires it to exist
func (s *TestSuiteBase) WaitForElement(page *Page, selector string, timeout time.Duration) Element {
	element, err := page.WaitForElement(selector, timeout)
	s.Require().NoError(err, "Element not found: %s", selector)
	return element
}

// AssertElementText verifies element text content
func (s *TestSuiteBase) AssertElementText(element Element, expected string) {
	text, err := element.Text()
	s.Require().NoError(err, "Failed to get element text")
	s.Equal(expected, text, "Element text mismatch")
}

// AssertElementValue verifies input element value
func (s *TestSuiteBase) AssertElementValue(element Element, expected string) {
	value, err := element.Value()
	s.Require().NoError(err, "Failed to get element value")
	s.Equal(expected, value, "Element value mismatch")
}

// DefaultBrowserOptions returns standard browser options for testing
func DefaultBrowserOptions() BrowserOptions {
	return BrowserOptions{
		Headless:  true,
		NoSandbox: true,
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--disable-web-security",
			"--disable-features=VizDisplayCompositor",
		},
	}
}

// QuickTestSuite is optimized for fast tests
type QuickTestSuite struct {
	TestSuiteBase
}

// SetupTest creates a minimal browser for quick tests
func (s *QuickTestSuite) SetupTest() {
	// Skip test server for quick tests unless needed
	browser, err := NewBrowser(DefaultBrowserOptions())
	s.Require().NoError(err, "Failed to create browser")
	s.browser = browser

	s.cleanup = func() {
		if browser != nil {
			browser.Close()
		}
	}
}

// CoverageTestSuite is specialized for coverage testing
type CoverageTestSuite struct {
	TestSuiteBase
	reporter *CoverageReporter
}

// SetupTest initializes browser, server and coverage reporter
func (s *CoverageTestSuite) SetupTest() {
	s.TestSuiteBase.SetupTest()
	s.reporter = NewCoverageReporter()
	s.reporter.SetDebugMode(true)
}

// Reporter returns the coverage reporter
func (s *CoverageTestSuite) Reporter() *CoverageReporter {
	return s.reporter
}

// TestBrowserAction represents a test action that can be performed
type TestBrowserAction func(t *testing.T, browser *Browser)

// TestPageAction represents a test action on a page
type TestPageAction func(t *testing.T, page *Page)

// RunBrowserAction executes a browser action with proper setup/teardown
func RunBrowserAction(t *testing.T, action TestBrowserAction) {
	browser, err := NewBrowser(DefaultBrowserOptions())
	require.NoError(t, err)
	defer browser.Close()

	action(t, browser)
}

// RunPageAction executes a page action with proper setup/teardown
func RunPageAction(t *testing.T, action TestPageAction) {
	RunBrowserAction(t, func(t *testing.T, browser *Browser) {
		page, err := browser.NewPage()
		require.NoError(t, err)
		defer page.Close()

		action(t, page)
	})
}

// RunHTMLPageAction executes a page action with predefined HTML
func RunHTMLPageAction(t *testing.T, html string, action TestPageAction) {
	RunPageAction(t, func(t *testing.T, page *Page) {
		err := page.Navigate(DataURLHTML(html))
		require.NoError(t, err)

		action(t, page)
	})
}

// TestCase represents a generic test case
type TestCase struct {
	Name    string
	Setup   func(*testing.T)
	Action  func(*testing.T) error
	Verify  func(*testing.T) error
	Cleanup func(*testing.T)
}

// RunTestCases executes a slice of test cases
func RunTestCases(t *testing.T, testCases []TestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Setup != nil {
				tc.Setup(t)
			}

			if tc.Cleanup != nil {
				defer tc.Cleanup(t)
			}

			if tc.Action != nil {
				err := tc.Action(t)
				require.NoError(t, err, "Action failed for test case: %s", tc.Name)
			}

			if tc.Verify != nil {
				err := tc.Verify(t)
				require.NoError(t, err, "Verification failed for test case: %s", tc.Name)
			}
		})
	}
}

// Utility functions for common test patterns

// WaitWithContext waits for a condition with context timeout
func WaitWithContext(ctx context.Context, condition func() bool, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// WaitForCondition waits for a condition with timeout
func WaitForCondition(timeout time.Duration, condition func() bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return WaitWithContext(ctx, condition, 50*time.Millisecond)
}

// RetryAction retries an action until it succeeds or max attempts reached
func RetryAction(maxAttempts int, delay time.Duration, action func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := action()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < maxAttempts {
			time.Sleep(delay)
		}
	}
	return lastErr
}
