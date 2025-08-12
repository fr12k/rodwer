package rodwer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
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

	// Roadmap page for coverage testing
	mux.HandleFunc("/roadmap", func(w http.ResponseWriter, r *http.Request) {
		html := RoadmapTestHTML()
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
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
