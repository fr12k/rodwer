package rodwer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Quick smoke tests for ultra-fast feedback during development
// Target: <15 seconds total execution time
// Run with: go test -short -v -run="Quick"

func TestQuick(t *testing.T) {
	if !testing.Short() {
		t.Skip("Skipping quick tests - use -short flag for fast feedback")
	}

	t.Run("browser_basic_functionality", func(t *testing.T) {
		t.Parallel()

		// Create dedicated browser for this test
		browser, cleanup, err := NewTestBrowser()
		require.NoError(t, err, "Failed to create test browser")
		defer cleanup()

		// Test browser is connected and functional
		assert.True(t, browser.IsConnected(), "Browser should be connected")
		assert.NotNil(t, browser.Context(), "Browser context should not be nil")

		// Test page creation
		page, err := browser.NewPage()
		require.NoError(t, err)
		defer page.Close()

		assert.NotNil(t, page, "Page should not be nil")
	})

	t.Run("page_navigation_data_url", func(t *testing.T) {
		t.Parallel()

		browser, cleanup, err := NewTestBrowser()
		require.NoError(t, err)
		defer cleanup()

		page, err := browser.NewPage()
		require.NoError(t, err)
		defer page.Close()

		// Use data URL for instant navigation (no network)
		testHTML := `<html><body><h1 id="title">Quick Test</h1><button class="btn">Click</button></body></html>`
		err = page.Navigate("data:text/html," + testHTML)
		require.NoError(t, err)

		// Verify navigation worked
		title, err := page.Title()
		require.NoError(t, err)
		assert.NotEmpty(t, title)
	})

	t.Run("element_selection_basic", func(t *testing.T) {
		t.Parallel()

		browser, cleanup, err := NewTestBrowser()
		require.NoError(t, err)
		defer cleanup()

		page, err := browser.NewPage()
		require.NoError(t, err)
		defer page.Close()

		// Simple HTML for testing
		testHTML := `<html><body>
			<h1 id="heading">Test Heading</h1>
			<p class="content">Test content</p>
			<button data-testid="btn">Test Button</button>
		</body></html>`

		err = page.Navigate("data:text/html," + testHTML)
		require.NoError(t, err)

		// Test ID selector
		element, err := page.Element("#heading")
		require.NoError(t, err)
		text, err := element.Text()
		require.NoError(t, err)
		assert.Equal(t, "Test Heading", text)

		// Test class selector
		element, err = page.Element(".content")
		require.NoError(t, err)
		text, err = element.Text()
		require.NoError(t, err)
		assert.Equal(t, "Test content", text)

		// Test data-testid selector
		element, err = page.Element("[data-testid='btn']")
		require.NoError(t, err)
		text, err = element.Text()
		require.NoError(t, err)
		assert.Equal(t, "Test Button", text)
	})

	t.Run("element_interaction_basic", func(t *testing.T) {
		t.Parallel()

		browser, cleanup, err := NewTestBrowser()
		require.NoError(t, err)
		defer cleanup()

		page, err := browser.NewPage()
		require.NoError(t, err)
		defer page.Close()

		// Interactive HTML
		testHTML := `<html><body>
			<input id="input" type="text" placeholder="Enter text">
			<button id="btn" onclick="document.getElementById('result').textContent = 'Clicked'">Click Me</button>
			<div id="result"></div>
		</body></html>`

		err = page.Navigate("data:text/html," + testHTML)
		require.NoError(t, err)

		// Test input typing
		input, err := page.Element("#input")
		require.NoError(t, err)

		err = input.Type("test text")
		require.NoError(t, err)

		value, err := input.Value()
		require.NoError(t, err)
		assert.Equal(t, "test text", value)

		// Test button clicking
		button, err := page.Element("#btn")
		require.NoError(t, err)

		err = button.Click()
		require.NoError(t, err)

		// Verify result (with short timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		result, err := page.WaitForElementWithContext(ctx, "#result")
		require.NoError(t, err)

		text, err := result.Text()
		require.NoError(t, err)
		assert.Equal(t, "Clicked", text)
	})

	t.Run("screenshot_basic", func(t *testing.T) {
		t.Parallel()

		browser, cleanup, err := NewTestBrowser()
		require.NoError(t, err)
		defer cleanup()

		page, err := browser.NewPage()
		require.NoError(t, err)
		defer page.Close()

		// Simple page for screenshot
		testHTML := `<html><body style="background: #f0f0f0; padding: 20px;">
			<h1>Screenshot Test</h1>
			<p>This is a test page for screenshots</p>
		</body></html>`

		err = page.Navigate("data:text/html," + testHTML)
		require.NoError(t, err)

		// Take screenshot
		screenshot, err := page.ScreenshotSimple()
		require.NoError(t, err)

		assert.Greater(t, len(screenshot), 1000, "Screenshot should have reasonable size")
		assert.Less(t, len(screenshot), 1000000, "Screenshot should not be excessively large")
	})
}
