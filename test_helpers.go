package rodwer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServer represents a test HTTP server for testing browser interactions
type TestServer struct {
	*httptest.Server
	mux *http.ServeMux
}

// NewTestServer creates a new test HTTP server with common endpoints
func NewTestServer() (*TestServer, func()) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Static HTML pages for testing
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Test Page</title>
			<meta charset="utf-8">
		</head>
		<body>
			<h1 id="title">Test Page</h1>
			<p class="content">This is a test page for browser automation.</p>
			<button id="test-btn" onclick="handleClick()">Click Me</button>
			<div id="result"></div>
			<script>
				function handleClick() {
					document.getElementById('result').textContent = 'Button clicked!';
				}
			</script>
		</body>
		</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Form page for testing interactions
	mux.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			name := r.FormValue("name")
			email := r.FormValue("email")
			html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<body>
				<h1>Form Submitted</h1>
				<p>Name: %s</p>
				<p>Email: %s</p>
			</body>
			</html>`, name, email)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
			return
		}

		html := `
		<!DOCTYPE html>
		<html>
		<body>
			<h1>Test Form</h1>
			<form method="POST" action="/form">
				<label for="name">Name:</label>
				<input type="text" id="name" name="name" required>
				
				<label for="email">Email:</label>
				<input type="email" id="email" name="email" required>
				
				<button type="submit" id="submit">Submit</button>
			</form>
		</body>
		</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Slow loading page for timeout testing
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Slow Page</h1></body></html>`))
	})

	// Dynamic content page for waiting tests
	mux.HandleFunc("/dynamic", func(w http.ResponseWriter, r *http.Request) {
		html := `
		<!DOCTYPE html>
		<html>
		<body>
			<h1>Dynamic Content</h1>
			<div id="initial">Initial content</div>
			<script>
				setTimeout(function() {
					var div = document.createElement('div');
					div.id = 'dynamic';
					div.textContent = 'Dynamic content loaded';
					document.body.appendChild(div);
				}, 1000);
			</script>
		</body>
		</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Delay endpoint for timeout testing (similar to httpbin.org/delay)
	mux.HandleFunc("/delay/", func(w http.ResponseWriter, r *http.Request) {
		// Extract delay seconds from URL path
		path := strings.TrimPrefix(r.URL.Path, "/delay/")
		seconds := 1 // default delay

		if path != "" {
			if parsed, err := time.ParseDuration(path + "s"); err == nil {
				seconds = int(parsed.Seconds())
			}
		}

		// Cap delay at 30 seconds for safety
		if seconds > 30 {
			seconds = 30
		}

		time.Sleep(time.Duration(seconds) * time.Second)

		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Delayed Response</title></head>
<body>
	<h1>Delayed Response</h1>
	<p>This response was delayed by %d seconds.</p>
	<p>Current time: %s</p>
</body>
</html>`, seconds, time.Now().Format("15:04:05"))

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	server := httptest.NewServer(mux)
	testServer := &TestServer{
		Server: server,
		mux:    mux,
	}

	cleanup := func() {
		server.Close()
	}

	return testServer, cleanup
}

// AddRoute adds a custom route to the test server
func (ts *TestServer) AddRoute(pattern string, handler http.HandlerFunc) {
	ts.mux.HandleFunc(pattern, handler)
}

// TestPageOptions configures test page creation
type TestPageOptions struct {
	HTML    string
	CSS     string
	JS      string
	Title   string
	Timeout time.Duration
}

// NewTestPage creates a test page with the specified options
func NewTestPage(options TestPageOptions) (*Page, func()) {
	if options.Title == "" {
		options.Title = "Test Page"
	}

	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}

	html := options.HTML
	if html == "" {
		html = fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>%s</title>
			<style>%s</style>
		</head>
		<body>
			<h1>Default Test Page</h1>
			<script>%s</script>
		</body>
		</html>`, options.Title, options.CSS, options.JS)
	}

	// This will fail until we implement the framework
	browser, err := NewBrowser(BrowserOptions{Headless: true})
	if err != nil {
		panic(fmt.Sprintf("Failed to create test browser: %v", err))
	}

	page, err := browser.NewPage()
	if err != nil {
		browser.Close()
		panic(fmt.Sprintf("Failed to create test page: %v", err))
	}

	dataURL := "data:text/html," + html
	err = page.Navigate(dataURL)
	if err != nil {
		page.Close()
		browser.Close()
		panic(fmt.Sprintf("Failed to navigate to test page: %v", err))
	}

	cleanup := func() {
		page.Close()
		browser.Close()
	}

	return page, cleanup
}

// NewTestBrowser creates a browser instance configured for testing
func NewTestBrowser() (*Browser, func(), error) {
	options := BrowserOptions{
		Headless:  true,
		NoSandbox: true, // Required for CI environments
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--disable-web-security",
			"--disable-features=VizDisplayCompositor",
			"--headless=new",
			"--remote-debugging-port=0",
			"--disable-background-timer-throttling",
			"--disable-renderer-backgrounding",
			"--disable-backgrounding-occluded-windows",
		},
	}

	browser, err := NewBrowser(options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create test browser: %w", err)
	}

	cleanup := func() {
		if browser != nil {
			browser.Close()
		}
	}

	return browser, cleanup, nil
}

// TestHelper provides common testing utilities
type TestHelper struct {
	t       *testing.T
	browser *Browser
	cleanup func()
}

// NewTestHelper creates a new test helper with a browser instance
func NewTestHelper(t *testing.T) *TestHelper {
	browser, cleanup, err := NewTestBrowser()
	require.NoError(t, err, "Failed to create test browser")

	return &TestHelper{
		t:       t,
		browser: browser,
		cleanup: cleanup,
	}
}

// Close cleans up the test helper resources
func (th *TestHelper) Close() {
	if th.cleanup != nil {
		th.cleanup()
	}
}

// NewPage creates a new page for testing
func (th *TestHelper) NewPage() *Page {
	page, err := th.browser.NewPage()
	require.NoError(th.t, err, "Failed to create new page")
	return page
}

// NavigateToHTML navigates to a page with the specified HTML content
func (th *TestHelper) NavigateToHTML(page *Page, html string) {
	dataURL := "data:text/html," + html
	err := page.Navigate(dataURL)
	require.NoError(th.t, err, "Failed to navigate to HTML content")
}

// WaitForElement waits for an element and returns it
func (th *TestHelper) WaitForElement(page *Page, selector string, timeout time.Duration) Element {
	element, err := page.WaitForElement(selector, timeout)
	require.NoError(th.t, err, "Failed to wait for element: %s", selector)
	return element
}

// AssertElementText verifies an element's text content
func (th *TestHelper) AssertElementText(element Element, expected string) {
	text, err := element.Text()
	require.NoError(th.t, err, "Failed to get element text")
	require.Equal(th.t, expected, text, "Element text mismatch")
}

// AssertElementValue verifies an input element's value
func (th *TestHelper) AssertElementValue(element Element, expected string) {
	value, err := element.Value()
	require.NoError(th.t, err, "Failed to get element value")
	require.Equal(th.t, expected, value, "Element value mismatch")
}

// ConcurrentTestRunner runs multiple test functions concurrently
type ConcurrentTestRunner struct {
	t      *testing.T
	wg     sync.WaitGroup
	mu     sync.Mutex
	errors []error
}

// NewConcurrentTestRunner creates a new concurrent test runner
func NewConcurrentTestRunner(t *testing.T) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		t:      t,
		errors: make([]error, 0),
	}
}

// Run executes a test function concurrently
func (ctr *ConcurrentTestRunner) Run(name string, testFunc func(*testing.T)) {
	ctr.wg.Add(1)
	go func() {
		defer ctr.wg.Done()

		// Create a sub-test
		ctr.t.Run(name, func(subT *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					ctr.mu.Lock()
					ctr.errors = append(ctr.errors, fmt.Errorf("panic in %s: %v", name, r))
					ctr.mu.Unlock()
				}
			}()

			testFunc(subT)
		})
	}()
}

// Wait waits for all concurrent tests to complete and reports any errors
func (ctr *ConcurrentTestRunner) Wait() {
	ctr.wg.Wait()

	ctr.mu.Lock()
	defer ctr.mu.Unlock()

	if len(ctr.errors) > 0 {
		for _, err := range ctr.errors {
			ctr.t.Errorf("Concurrent test error: %v", err)
		}
	}
}

// PerformanceTestRunner provides utilities for performance testing
type PerformanceTestRunner struct {
	t *testing.T
}

// NewPerformanceTestRunner creates a new performance test runner
func NewPerformanceTestRunner(t *testing.T) *PerformanceTestRunner {
	return &PerformanceTestRunner{t: t}
}

// TimeOperation measures the execution time of an operation
func (ptr *PerformanceTestRunner) TimeOperation(name string, operation func() error) time.Duration {
	start := time.Now()
	err := operation()
	duration := time.Since(start)

	require.NoError(ptr.t, err, "Operation %s failed", name)
	ptr.t.Logf("Operation %s took %v", name, duration)

	return duration
}

// AssertPerformance verifies that an operation completes within expected time
func (ptr *PerformanceTestRunner) AssertPerformance(name string, operation func() error, maxDuration time.Duration) {
	duration := ptr.TimeOperation(name, operation)
	require.LessOrEqual(ptr.t, duration, maxDuration,
		"Operation %s took %v, expected <= %v", name, duration, maxDuration)
}

// BenchmarkOperation runs a benchmark test for an operation
func (ptr *PerformanceTestRunner) BenchmarkOperation(name string, operation func() error, iterations int) {
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		duration := ptr.TimeOperation(fmt.Sprintf("%s-iteration-%d", name, i), operation)
		totalDuration += duration
	}

	avgDuration := totalDuration / time.Duration(iterations)
	ptr.t.Logf("Benchmark %s: %d iterations, avg %v, total %v",
		name, iterations, avgDuration, totalDuration)
}

// RetryHelper provides utilities for retrying operations
type RetryHelper struct {
	t           *testing.T
	maxAttempts int
	delay       time.Duration
}

// NewRetryHelper creates a new retry helper
func NewRetryHelper(t *testing.T, maxAttempts int, delay time.Duration) *RetryHelper {
	return &RetryHelper{
		t:           t,
		maxAttempts: maxAttempts,
		delay:       delay,
	}
}

// Retry executes an operation with retry logic
func (rh *RetryHelper) Retry(operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= rh.maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err
		rh.t.Logf("Attempt %d/%d failed: %v", attempt, rh.maxAttempts, err)

		if attempt < rh.maxAttempts {
			time.Sleep(rh.delay)
		}
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", rh.maxAttempts, lastErr)
}

// AssertRetry runs an operation with retries and requires success
func (rh *RetryHelper) AssertRetry(operation func() error) {
	err := rh.Retry(operation)
	require.NoError(rh.t, err, "Operation failed after retries")
}

// MockResponseServer creates a server that returns predefined responses
type MockResponseServer struct {
	*httptest.Server
	responses map[string]MockResponse
	mu        sync.RWMutex
}

// MockResponse defines a mock HTTP response
type MockResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       string
	Delay      time.Duration
}

// NewMockResponseServer creates a new mock response server
func NewMockResponseServer() (*MockResponseServer, func()) {
	mrs := &MockResponseServer{
		responses: make(map[string]MockResponse),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mrs.mu.RLock()
		response, exists := mrs.responses[r.URL.Path]
		mrs.mu.RUnlock()

		if !exists {
			http.NotFound(w, r)
			return
		}

		if response.Delay > 0 {
			time.Sleep(response.Delay)
		}

		for key, value := range response.Headers {
			w.Header().Set(key, value)
		}

		w.WriteHeader(response.StatusCode)
		w.Write([]byte(response.Body))
	})

	server := httptest.NewServer(handler)
	mrs.Server = server

	cleanup := func() {
		server.Close()
	}

	return mrs, cleanup
}

// SetResponse sets a mock response for a specific path
func (mrs *MockResponseServer) SetResponse(path string, response MockResponse) {
	mrs.mu.Lock()
	defer mrs.mu.Unlock()
	mrs.responses[path] = response
}

// ClearResponses clears all mock responses
func (mrs *MockResponseServer) ClearResponses() {
	mrs.mu.Lock()
	defer mrs.mu.Unlock()
	mrs.responses = make(map[string]MockResponse)
}

// ContextHelper provides utilities for context management in tests
type ContextHelper struct {
	t *testing.T
}

// NewContextHelper creates a new context helper
func NewContextHelper(t *testing.T) *ContextHelper {
	return &ContextHelper{t: t}
}

// WithTimeout creates a context with timeout
func (ch *ContextHelper) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// WithDeadline creates a context with deadline
func (ch *ContextHelper) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), deadline)
}

// WithCancel creates a cancellable context
func (ch *ContextHelper) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// AssertContextNotExpired verifies that a context hasn't expired
func (ch *ContextHelper) AssertContextNotExpired(ctx context.Context) {
	select {
	case <-ctx.Done():
		ch.t.Fatal("Context expired unexpectedly")
	default:
		// Context is still active
	}
}
