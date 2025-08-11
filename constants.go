package rodwer

import "time"

// Test and Coverage Constants
const (
	// Directory and file paths
	CoverageDir       = "coverage"
	JSCoverageFile    = "coverage/js-coverage.json"
	JSCoverageHTML    = "coverage/js-coverage.html"
	GoCoverageHTML    = "coverage/go-cover.html"
	GoCoverageRaw     = "coverage.txt"
	CoverageIndexHTML = "coverage/index.html"

	// Screenshot file paths
	ScreenshotInitial    = "coverage/screenshot-page.png"
	ScreenshotAfterClick = "coverage/screenshot-after-click.png"
	TestScreenshotDir    = "test_screenshots"

	// Default image format for screenshots
	DefaultScreenshotFormat = "png"
)

// Default timeouts and delays
const (
	// Test execution timeouts
	DefaultTestTimeout = 30 * time.Second
	QuickTestTimeout   = 5 * time.Second
	ElementWaitTimeout = 3 * time.Second
	PageLoadTimeout    = 10 * time.Second

	// Coverage collection timeouts
	DefaultCoverageTimeout = 5 * time.Second
	QuickCoverageTimeout   = 1 * time.Second
	AsyncWaitTimeout       = 2 * time.Second
	StabilityWaitTimeout   = 1 * time.Second

	// Polling and retry intervals
	ElementPollInterval   = 50 * time.Millisecond
	StabilityPollInterval = 50 * time.Millisecond
	RetryDelay            = 100 * time.Millisecond

	// Test execution delays
	DOMContentLoadedDelay = 200 * time.Millisecond
	AsyncJavaScriptDelay  = 200 * time.Millisecond
	MinimumWaitTime       = 50 * time.Millisecond
)

// Browser configuration constants
const (
	// Default viewport dimensions
	DefaultViewportWidth  = 1920
	DefaultViewportHeight = 1080
	TestViewportWidth     = 1280
	TestViewportHeight    = 720
	QuickTestWidth        = 800
	QuickTestHeight       = 600

	// Coverage filtering thresholds
	MinScriptSize            = 30
	MaxStatementsPerLine     = 50
	ApplicationScriptMinSize = 15
	ProductionMinSize        = 50

	// Performance thresholds
	MaxScreenshotSize     = 1000000 // 1MB
	MinScreenshotSize     = 1000    // 1KB
	MaxConcurrentBrowsers = 3

	// Test constants
	MaxRetryAttempts = 3
	StabilityChecks  = 3
)

// Test server configuration
const (
	TestServerDelay    = 2 * time.Second
	MaxTestServerDelay = 30 * time.Second
	HealthCheckPath    = "/health"
	FormPath           = "/form"
	SlowPath           = "/slow"
	DynamicPath        = "/dynamic"
	DelayPathPrefix    = "/delay/"
	RoadmapPath        = "/roadmap"
)

// Error messages and patterns
const (
	BrowserClosedError      = "browser is closed"
	PageClosedError         = "page is closed"
	ElementNilError         = "element is nil"
	EmptyFilePathError      = "file path cannot be empty"
	ExecutableNotFoundError = "executable not found"
	TimeoutWaitingError     = "timeout waiting for element"
	NavigationFailedError   = "failed to navigate"
	ScreenshotFailedError   = "failed to take screenshot"
	CoverageCollectionError = "failed to collect coverage"
)

// Browser launch arguments for different environments
var (
	// Standard Chrome arguments for headless testing
	DefaultChromeArgs = []string{
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
	}

	// CI-specific arguments (more restrictive)
	CIChromeArgs = append(DefaultChromeArgs,
		"--no-zygote",
		"--single-process",
		"--disable-background-networking",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-sync",
		"--disable-translate",
		"--hide-scrollbars",
		"--metrics-recording-only",
		"--mute-audio",
		"--no-first-run",
		"--safebrowsing-disable-auto-update",
	)

	// Development arguments (more permissive)
	DevChromeArgs = []string{
		"--no-sandbox",
		"--disable-web-security",
		"--disable-features=VizDisplayCompositor",
	}
)

// Coverage filtering profiles
var (
	DefaultFilterProfile     = "default"
	DevelopmentFilterProfile = "development"
	ProductionFilterProfile  = "production"
	ApplicationFilterProfile = "application"
)

// Common selectors used in tests
var (
	CommonSelectors = struct {
		Title        string
		Heading      string
		Button       string
		Input        string
		Form         string
		List         string
		ListItem     string
		Content      string
		Result       string
		SubmitButton string
		TestButton   string
		CopyButton   string
	}{
		Title:        "#title",
		Heading:      "h1",
		Button:       "button",
		Input:        "input",
		Form:         "form",
		List:         "ul",
		ListItem:     "li",
		Content:      ".content",
		Result:       "#result",
		SubmitButton: "#submit",
		TestButton:   "#test-btn",
		CopyButton:   "#copy-all-btn",
	}
)

// File extensions and formats
var (
	ImageFormats = struct {
		PNG  string
		JPEG string
		JPG  string
	}{
		PNG:  "png",
		JPEG: "jpeg",
		JPG:  "jpg",
	}

	FileExtensions = struct {
		PNG  string
		JPEG string
		JPG  string
		HTML string
		JSON string
		TXT  string
	}{
		PNG:  ".png",
		JPEG: ".jpeg",
		JPG:  ".jpg",
		HTML: ".html",
		JSON: ".json",
		TXT:  ".txt",
	}
)

// Quality settings for different output formats
var (
	ImageQuality = struct {
		High   int
		Medium int
		Low    int
	}{
		High:   90,
		Medium: 75,
		Low:    50,
	}
)
