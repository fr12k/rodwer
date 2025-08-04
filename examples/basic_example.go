package main

import (
	"fmt"
	"log"

	"github/fr12k/rodwer"
)

func main() {
	// Create browser with options
	browser, err := rodwer.NewBrowser(rodwer.BrowserOptions{
		Headless: true,
		Viewport: &rodwer.Viewport{
			Width:  1920,
			Height: 1080,
		},
	})
	if err != nil {
		log.Fatal("Failed to create browser:", err)
	}
	defer browser.Close()

	// Create a new page
	page, err := browser.NewPage()
	if err != nil {
		log.Fatal("Failed to create page:", err)
	}

	// Navigate to a website
	err = page.Goto("https://example.com")
	if err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	// Find an element and get its text
	element, err := page.Element("h1")
	if err != nil {
		log.Fatal("Failed to find element:", err)
	}

	text, err := element.Text()
	if err != nil {
		log.Fatal("Failed to get text:", err)
	}

	fmt.Printf("Found heading text: %s\n", text)

	// Take a screenshot
	screenshot, err := page.ScreenshotSimple()
	if err != nil {
		log.Fatal("Failed to take screenshot:", err)
	}

	fmt.Printf("Screenshot taken, size: %d bytes\n", len(screenshot))

	// Demonstrate element interaction
	fmt.Println("Rodwer browser testing framework example completed successfully!")
}
