package rodwer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicExample demonstrates the basic usage of the rodwer framework
func TestBasicExample(t *testing.T) {
	t.Parallel() // Allow parallel execution with other tests

	// Create internal test server
	testServer, cleanup := NewTestServer()
	defer cleanup()

	// Create browser with options (like Playwright)
	browser, err := NewBrowser(BrowserOptions{
		Headless: true,
		Viewport: &Viewport{
			Width:  1920,
			Height: 1080,
		},
	})
	require.NoError(t, err)
	defer browser.Close()

	// Create a new page
	page, err := browser.NewPage()
	require.NoError(t, err)

	// Navigate to internal test server
	err = page.Goto(testServer.URL)
	require.NoError(t, err)

	// Find an element and get its text
	element, err := page.Element("h1")
	require.NoError(t, err)

	text, err := element.Text()
	require.NoError(t, err)
	assert.Contains(t, text, "Test Page")

	// Take a screenshot
	screenshot, err := page.ScreenshotSimple()
	require.NoError(t, err)
	assert.Greater(t, len(screenshot), 1000) // Should be a reasonable size

	t.Logf("Successfully completed basic example: found heading '%s', screenshot size: %d bytes", text, len(screenshot))
}

// TestAdvancedExample demonstrates more advanced features
func TestAdvancedExample(t *testing.T) {
	t.Parallel() // Allow parallel execution with other tests

	// Create internal test server
	testServer, cleanup := NewTestServer()
	defer cleanup()

	browser, err := NewBrowser(BrowserOptions{
		Headless: true,
		Viewport: &Viewport{Width: 1280, Height: 720},
	})
	require.NoError(t, err)
	defer browser.Close()

	page, err := browser.NewPage()
	require.NoError(t, err)

	// Navigate to internal test server
	err = page.Goto(testServer.URL)
	require.NoError(t, err)

	// Find multiple elements
	h1Element, err := page.Element("h1")
	require.NoError(t, err)

	text, err := h1Element.Text()
	require.NoError(t, err)
	assert.NotEmpty(t, text)

	// Take element screenshot
	elementScreenshot, err := h1Element.Screenshot()
	require.NoError(t, err)
	assert.Greater(t, len(elementScreenshot), 100)

	t.Logf("Advanced example completed: element text '%s', element screenshot size: %d bytes", text, len(elementScreenshot))
}

// TestConcurrentBrowsers demonstrates multiple browser instances
func TestConcurrentBrowsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow concurrent test in short mode")
	}
	const numBrowsers = 3

	// Create internal test server (shared by all browsers)
	testServer, cleanup := NewTestServer()
	defer cleanup()

	type result struct {
		id    int
		title string
		err   error
	}

	results := make(chan result, numBrowsers)

	// Launch multiple browsers concurrently
	for i := 0; i < numBrowsers; i++ {
		go func(id int) {
			browser, err := NewBrowser(BrowserOptions{
				Headless: true,
				Viewport: &Viewport{Width: 800, Height: 600},
			})
			if err != nil {
				results <- result{id: id, err: err}
				return
			}
			defer browser.Close()

			page, err := browser.NewPage()
			if err != nil {
				results <- result{id: id, err: err}
				return
			}

			err = page.Goto(testServer.URL)
			if err != nil {
				results <- result{id: id, err: err}
				return
			}

			element, err := page.Element("h1")
			if err != nil {
				results <- result{id: id, err: err}
				return
			}

			title, err := element.Text()
			results <- result{id: id, title: title, err: err}
		}(i)
	}

	// Collect results with overall timeout
	completed := 0
	successful := 0
	timeout := time.After(45 * time.Second) // Total timeout for all browsers

	for completed < numBrowsers {
		select {
		case res := <-results:
			completed++
			if res.err != nil {
				t.Logf("Browser %d failed: %v", res.id, res.err)
				// Don't fail immediately, let other browsers complete
				continue
			}
			successful++
			assert.NotEmpty(t, res.title, "Browser %d got empty title", res.id)
			t.Logf("Browser %d completed successfully: title '%s'", res.id, res.title)
		case <-timeout:
			t.Logf("Timeout waiting for concurrent browsers. Completed: %d/%d, Successful: %d", completed, numBrowsers, successful)
			break
		}
	}

	// Require at least 2 out of 3 browsers to succeed (to handle network issues)
	if successful < 2 {
		t.Fatalf("Too few browsers completed successfully: %d/%d (completed: %d)", successful, numBrowsers, completed)
	}

	t.Logf("Concurrent browser test passed: %d/%d browsers successful", successful, numBrowsers)
}

// TestFormInteraction demonstrates form handling
func TestFormInteraction(t *testing.T) {
	// Create internal test server
	testServer, cleanup := NewTestServer()
	defer cleanup()

	browser, err := NewBrowser(BrowserOptions{
		Headless: true,
		Viewport: &Viewport{Width: 1024, Height: 768},
	})
	require.NoError(t, err)
	defer browser.Close()

	page, err := browser.NewPage()
	require.NoError(t, err)

	// Navigate to internal test server form page
	err = page.Goto(testServer.URL + "/form")
	require.NoError(t, err)

	// Fill form fields
	nameField, err := page.Element("input[name='name']")
	require.NoError(t, err)

	err = nameField.Fill("John Doe")
	require.NoError(t, err)

	value, err := nameField.Value()
	require.NoError(t, err)
	assert.Equal(t, "John Doe", value)

	// Fill email field
	emailField, err := page.Element("input[name='email']")
	require.NoError(t, err)

	err = emailField.Fill("john.doe@example.com")
	require.NoError(t, err)

	emailValue, err := emailField.Value()
	require.NoError(t, err)
	assert.Equal(t, "john.doe@example.com", emailValue)

	t.Log("Form interaction test completed successfully")
}
