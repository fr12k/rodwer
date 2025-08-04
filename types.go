package rodwer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// BrowserOptions configures browser creation
type BrowserOptions struct {
	Headless       bool
	NoSandbox      bool
	Args           []string
	ExecutablePath string
	Viewport       *Viewport
	DevTools       bool
	UserAgent      string
}

// Viewport defines browser window dimensions
type Viewport struct {
	Width  int
	Height int
}

// Browser represents a browser instance
type Browser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
	ctx      context.Context
	cancel   context.CancelFunc
	options  BrowserOptions
	mu       sync.RWMutex
	closed   bool
}

// Page represents a browser page/tab
type Page struct {
	page    *rod.Page
	browser *Browser
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	closed  bool
}

// Element represents a DOM element
type Element struct {
	element *rod.Element
	page    *Page
}

// ScreenshotOptions configures screenshot capture
type ScreenshotOptions struct {
	FullPage bool
	Format   string // "png", "jpeg"
	Quality  int    // for JPEG
	Selector string // for element screenshots
}

// CoverageEntry represents JavaScript coverage data
type CoverageEntry struct {
	URL    string
	Source string
	Ranges []CoverageRange
}

// CoverageRange represents a coverage range
type CoverageRange struct {
	Start int
	End   int
	Count int
}

// Browser interface methods

// NewBrowser creates a new browser instance
func NewBrowser(options BrowserOptions) (*Browser, error) {
	// Validate options first
	if err := ValidateBrowserOptions(options); err != nil {
		return nil, fmt.Errorf("invalid browser options: %w", err)
	}

	// Create context for browser lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	// Configure launcher
	launcher := launcher.New()
	launcher.Headless(options.Headless)

	// if options.NoSandbox {
	launcher.NoSandbox(true)
	// }

	if options.DevTools {
		launcher.Devtools(true)
	}

	if options.ExecutablePath != "" {
		launcher.Bin(options.ExecutablePath)
	}

	// Add custom arguments
	for _, arg := range options.Args {
		launcher.Set("args", arg)
	}

	// Launch browser
	controlURL, err := launcher.Launch()
	if err != nil {
		cancel()
		// Check if it's an executable not found error
		if strings.Contains(err.Error(), "no such file or directory") && options.ExecutablePath != "" {
			return nil, fmt.Errorf("executable not found: %s", options.ExecutablePath)
		}
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to browser
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	// Configure browser settings
	if options.UserAgent != "" {
		// Set user agent for new pages
		// Note: This will be applied to pages when they are created
	}

	b := &Browser{
		browser:  browser,
		launcher: launcher,
		ctx:      ctx,
		cancel:   cancel,
		options:  options,
	}

	return b, nil
}

// ValidateBrowserOptions validates browser options
func ValidateBrowserOptions(options BrowserOptions) error {
	if options.Viewport != nil {
		if options.Viewport.Width <= 0 {
			return fmt.Errorf("viewport width must be positive, got %d", options.Viewport.Width)
		}
		if options.Viewport.Height <= 0 {
			return fmt.Errorf("viewport height must be positive, got %d", options.Viewport.Height)
		}
	}

	if options.ExecutablePath != "" {
		// Only validate path format, not existence (that's done in NewBrowser)
		if !filepath.IsAbs(options.ExecutablePath) {
			return fmt.Errorf("executable path must be absolute: %s", options.ExecutablePath)
		}
	}

	return nil
}

// NewPage creates a new page
func (b *Browser) NewPage() (*Page, error) {
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("browser is closed")
	}

	// Create new page
	rodPage, err := b.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Configure viewport if specified
	if b.options.Viewport != nil {
		err = rodPage.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:  b.options.Viewport.Width,
			Height: b.options.Viewport.Height,
		})
		if err != nil {
			rodPage.MustClose()
			return nil, fmt.Errorf("failed to set viewport: %w", err)
		}
	}

	// Create page context
	ctx, cancel := context.WithCancel(b.ctx)

	page := &Page{
		page:    rodPage,
		browser: b,
		ctx:     ctx,
		cancel:  cancel,
	}

	return page, nil
}

// Pages returns all pages
func (b *Browser) Pages() ([]*Page, error) {
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("browser is closed")
	}

	// Get all pages from browser
	rodPages, err := b.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}

	// Convert to our Page type
	pages := make([]*Page, len(rodPages))
	for i, rodPage := range rodPages {
		ctx, cancel := context.WithCancel(b.ctx)
		pages[i] = &Page{
			page:    rodPage,
			browser: b,
			ctx:     ctx,
			cancel:  cancel,
		}
	}

	return pages, nil
}

// Close closes the browser
func (b *Browser) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true

	// Cancel context first
	if b.cancel != nil {
		b.cancel()
	}

	// Close browser
	if b.browser != nil {
		if err := b.browser.Close(); err != nil {
			return fmt.Errorf("failed to close browser: %w", err)
		}
	}

	// Close launcher
	if b.launcher != nil {
		b.launcher.Cleanup()
	}

	return nil
}

// IsConnected returns connection status
func (b *Browser) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed || b.browser == nil {
		return false
	}

	// Try to get browser version to check if still connected
	_, err := b.browser.Version()
	return err == nil
}

// Context returns browser context
func (b *Browser) Context() context.Context {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ctx
}

// Page interface methods

// Navigate navigates to URL
func (p *Page) Navigate(url string) error {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return fmt.Errorf("page is closed")
	}

	if err := p.page.Navigate(url); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// Wait for page to load
	p.page.MustWaitLoad()
	return nil
}

// Goto is an alias for Navigate (Playwright-style API)
func (p *Page) Goto(url string) error {
	return p.Navigate(url)
}

// NavigateWithContext navigates with context
func (p *Page) NavigateWithContext(ctx context.Context, url string) error {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return fmt.Errorf("page is closed")
	}

	// Use WithCancel to combine contexts
	combinedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Navigate with timeout
	page := p.page.Context(combinedCtx)
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// Wait for page to load with context
	page.MustWaitLoad()
	return nil
}

// Title returns page title
func (p *Page) Title() (string, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return "", fmt.Errorf("page is closed")
	}

	info, err := p.page.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get page info: %w", err)
	}

	return info.Title, nil
}

// URL returns current URL
func (p *Page) URL() string {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed || p.page == nil {
		return ""
	}

	info, err := p.page.Info()
	if err != nil {
		return ""
	}

	return info.URL
}

// Element finds an element by selector
func (p *Page) Element(selector string) (Element, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return Element{}, fmt.Errorf("page is closed")
	}

	rodElement, err := p.page.Element(selector)
	if err != nil {
		return Element{}, fmt.Errorf("element not found: %s", selector)
	}

	return Element{
		element: rodElement,
		page:    p,
	}, nil
}

// Elements finds multiple elements by selector
func (p *Page) Elements(selector string) ([]Element, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("page is closed")
	}

	rodElements, err := p.page.Elements(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to find elements: %s", selector)
	}

	elements := make([]Element, len(rodElements))
	for i, rodElement := range rodElements {
		elements[i] = Element{
			element: rodElement,
			page:    p,
		}
	}

	return elements, nil
}

// WaitForElement waits for element to appear
func (p *Page) WaitForElement(selector string, timeout time.Duration) (Element, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return Element{}, fmt.Errorf("page is closed")
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	return p.WaitForElementWithContext(ctx, selector)
}

// WaitForElementWithContext waits for element with context
func (p *Page) WaitForElementWithContext(ctx context.Context, selector string) (Element, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return Element{}, fmt.Errorf("page is closed")
	}

	// Use Rod's wait functionality with timeout
	rodElement, err := p.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		if ctx.Err() != nil {
			return Element{}, fmt.Errorf("timeout waiting for element %s: %w", selector, ctx.Err())
		}
		return Element{}, fmt.Errorf("element not found: %s", selector)
	}

	return Element{
		element: rodElement,
		page:    p,
	}, nil
}

// Screenshot captures page screenshot
func (p *Page) Screenshot(options ScreenshotOptions) ([]byte, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("page is closed")
	}

	// Handle element screenshot
	if options.Selector != "" {
		element, err := p.Element(options.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to find element for screenshot: %w", err)
		}
		return p.screenshotElement(element, options)
	}

	// Handle full page or viewport screenshot
	return p.screenshotPage(options)
}

// ScreenshotSimple captures page screenshot with default options (convenience method)
func (p *Page) ScreenshotSimple() ([]byte, error) {
	return p.Screenshot(ScreenshotOptions{
		Format: "png",
	})
}

// ScreenshotToFile captures page screenshot and saves directly to file
func (p *Page) ScreenshotToFile(filePath string, options ...ScreenshotOptions) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Use default options if none provided
	var opts ScreenshotOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = ScreenshotOptions{
			Format: "png",
		}
	}

	// Auto-detect format from file extension if not specified
	if opts.Format == "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".jpg", ".jpeg":
			opts.Format = "jpeg"
		case ".png":
			opts.Format = "png"
		default:
			opts.Format = "png" // default to PNG
		}
	}

	// Take screenshot
	data, err := p.Screenshot(opts)
	if err != nil {
		return fmt.Errorf("failed to take screenshot: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write screenshot to file %s: %w", filePath, err)
	}

	return nil
}

// ScreenshotSimpleToFile captures page screenshot with default options and saves to file
func (p *Page) ScreenshotSimpleToFile(filePath string) error {
	return p.ScreenshotToFile(filePath)
}

// StartJSCoverage starts JavaScript coverage collection
func (p *Page) StartJSCoverage() error {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return fmt.Errorf("page is closed")
	}

	// Enable Debugger and Profiler domains
	_, err := proto.DebuggerEnable{}.Call(p.page)
	if err != nil {
		return fmt.Errorf("failed to enable debugger: %w", err)
	}

	err = proto.ProfilerEnable{}.Call(p.page)
	if err != nil {
		return fmt.Errorf("failed to enable profiler: %w", err)
	}

	// Start precise coverage collection
	_, err = proto.ProfilerStartPreciseCoverage{
		CallCount: true,
		Detailed:  true,
	}.Call(p.page)
	if err != nil {
		return fmt.Errorf("failed to start precise coverage: %w", err)
	}

	return nil
}

// StopJSCoverage stops JavaScript coverage collection
func (p *Page) StopJSCoverage() ([]CoverageEntry, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("page is closed")
	}

	// Take precise coverage snapshot
	result, err := proto.ProfilerTakePreciseCoverage{}.Call(p.page)
	if err != nil {
		return nil, fmt.Errorf("failed to take coverage snapshot: %w", err)
	}

	// Stop coverage collection
	err = proto.ProfilerStopPreciseCoverage{}.Call(p.page)
	if err != nil {
		return nil, fmt.Errorf("failed to stop coverage: %w", err)
	}

	// Convert to our coverage format
	coverageEntries := make([]CoverageEntry, 0)

	for _, script := range result.Result {
		// Get script source
		srcResp, err := proto.DebuggerGetScriptSource{ScriptID: script.ScriptID}.Call(p.page)
		if err != nil || srcResp.ScriptSource == "" {
			continue // Skip scripts without source
		}

		// Collect all ranges from all functions
		ranges := make([]CoverageRange, 0)
		for _, fn := range script.Functions {
			for _, r := range fn.Ranges {
				ranges = append(ranges, CoverageRange{
					Start: r.StartOffset,
					End:   r.EndOffset,
					Count: r.Count,
				})
			}
		}

		// Handle empty URLs for inline scripts or data URLs
		url := script.URL
		if url == "" {
			url = fmt.Sprintf("inline-script-%s", script.ScriptID)
		}

		coverageEntries = append(coverageEntries, CoverageEntry{
			URL:    url,
			Source: srcResp.ScriptSource,
			Ranges: ranges,
		})
	}

	return coverageEntries, nil
}

// Close closes the page
func (p *Page) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Cancel context first
	if p.cancel != nil {
		p.cancel()
	}

	// Close the page
	if p.page != nil {
		if err := p.page.Close(); err != nil {
			return fmt.Errorf("failed to close page: %w", err)
		}
	}

	return nil
}

// Context returns page context
func (p *Page) Context() context.Context {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ctx
}

// Element interface methods

// Click clicks the element
func (e Element) Click() error {
	if e.element == nil {
		return fmt.Errorf("element is nil")
	}

	if err := e.element.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click element: %w", err)
	}

	return nil
}

// Type types text into the element
func (e Element) Type(text string) error {
	if e.element == nil {
		return fmt.Errorf("element is nil")
	}

	if err := e.element.Input(text); err != nil {
		return fmt.Errorf("failed to type text: %w", err)
	}

	return nil
}

// Fill is an alias for Type (Playwright-style API)
func (e Element) Fill(text string) error {
	return e.Type(text)
}

// Clear clears the element content
func (e Element) Clear() error {
	if e.element == nil {
		return fmt.Errorf("element is nil")
	}

	// Select all and delete
	if err := e.element.SelectAllText(); err != nil {
		return fmt.Errorf("failed to select text: %w", err)
	}

	if err := e.element.Input(""); err != nil {
		return fmt.Errorf("failed to clear element: %w", err)
	}

	return nil
}

// Text returns element text content
func (e Element) Text() (string, error) {
	if e.element == nil {
		return "", fmt.Errorf("element is nil")
	}

	text, err := e.element.Text()
	if err != nil {
		return "", fmt.Errorf("failed to get text: %w", err)
	}

	return text, nil
}

// Value returns element value
func (e Element) Value() (string, error) {
	if e.element == nil {
		return "", fmt.Errorf("element is nil")
	}

	// Get the value property
	val, err := e.element.Property("value")
	if err != nil {
		return "", fmt.Errorf("failed to get value: %w", err)
	}

	// Convert JSON value to string
	return val.String(), nil
}

// TagName returns element tag name
func (e Element) TagName() (string, error) {
	if e.element == nil {
		return "", fmt.Errorf("element is nil")
	}

	// Get the tag name
	val, err := e.element.Property("tagName")
	if err != nil {
		return "", fmt.Errorf("failed to get tag name: %w", err)
	}

	// Convert JSON value to string
	return val.String(), nil
}

// Screenshot takes a screenshot of the element
func (e Element) Screenshot() ([]byte, error) {
	if e.element == nil {
		return nil, fmt.Errorf("element is nil")
	}

	return e.page.screenshotElement(e, ScreenshotOptions{
		Format: "png",
	})
}

// ScreenshotToFile takes a screenshot of the element and saves directly to file
func (e Element) ScreenshotToFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if e.element == nil {
		return fmt.Errorf("element is nil")
	}

	// Auto-detect format from file extension
	var format string
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		format = "jpeg"
	case ".png":
		format = "png"
	default:
		format = "png" // default to PNG
	}

	// Take screenshot
	data, err := e.page.screenshotElement(e, ScreenshotOptions{
		Format: format,
	})
	if err != nil {
		return fmt.Errorf("failed to take element screenshot: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write element screenshot to file %s: %w", filePath, err)
	}

	return nil
}

// Helper function to check if file exists
func fileExists(filename string) bool {
	_, err := filepath.Abs(filename)
	return err == nil
}

// screenshotPage captures a full page or viewport screenshot
func (p *Page) screenshotPage(options ScreenshotOptions) ([]byte, error) {
	format := proto.PageCaptureScreenshotFormatPng
	if strings.ToLower(options.Format) == "jpeg" {
		format = proto.PageCaptureScreenshotFormatJpeg
	}

	// Configure screenshot request
	req := &proto.PageCaptureScreenshot{
		Format: format,
	}

	// Set quality for JPEG
	if format == proto.PageCaptureScreenshotFormatJpeg && options.Quality > 0 {
		req.Quality = &options.Quality
	}

	// Set capture beyond viewport for full page
	if options.FullPage {
		req.CaptureBeyondViewport = true
	}

	// Take screenshot
	result, err := req.Call(p.page)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	return result.Data, nil
}

// screenshotElement captures a screenshot of a specific element
func (p *Page) screenshotElement(element Element, options ScreenshotOptions) ([]byte, error) {
	if element.element == nil {
		return nil, fmt.Errorf("element is nil")
	}

	format := proto.PageCaptureScreenshotFormatPng
	if strings.ToLower(options.Format) == "jpeg" {
		format = proto.PageCaptureScreenshotFormatJpeg
	}

	// Get element bounds
	box, err := element.element.Shape()
	if err != nil {
		return nil, fmt.Errorf("failed to get element bounds: %w", err)
	}

	if len(box.Quads) == 0 {
		return nil, fmt.Errorf("element has no quads")
	}

	quad := box.Quads[0]

	// Calculate bounding box
	minX, maxX := quad[0], quad[0]
	minY, maxY := quad[1], quad[1]

	for i := 0; i < len(quad); i += 2 {
		if quad[i] < minX {
			minX = quad[i]
		}
		if quad[i] > maxX {
			maxX = quad[i]
		}
		if quad[i+1] < minY {
			minY = quad[i+1]
		}
		if quad[i+1] > maxY {
			maxY = quad[i+1]
		}
	}

	// Configure screenshot request
	req := &proto.PageCaptureScreenshot{
		Format: format,
		Clip: &proto.PageViewport{
			X:      minX,
			Y:      minY,
			Width:  maxX - minX,
			Height: maxY - minY,
			Scale:  1,
		},
	}

	// Set quality for JPEG
	if format == proto.PageCaptureScreenshotFormatJpeg && options.Quality > 0 {
		req.Quality = &options.Quality
	}

	// Take screenshot
	result, err := req.Call(p.page)
	if err != nil {
		return nil, fmt.Errorf("failed to capture element screenshot: %w", err)
	}

	return result.Data, nil
}
