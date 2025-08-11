package rodwer

import "fmt"

// TestHTMLTemplates provides reusable HTML templates for testing

// BasicTestHTML returns a basic HTML page for testing
func BasicTestHTML(title, content string) string {
	if title == "" {
		title = "Test Page"
	}
	if content == "" {
		content = "This is a test page."
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>%s</title>
	<meta charset="utf-8">
</head>
<body>
	<h1 id="title">%s</h1>
	<p class="content">%s</p>
</body>
</html>`, title, title, content)
}

// InteractiveTestHTML returns an HTML page with interactive elements
func InteractiveTestHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
	<title>Interactive Test Page</title>
	<meta charset="utf-8">
</head>
<body>
	<h1 id="title">Interactive Test Page</h1>
	<input id="text-input" type="text" placeholder="Enter text">
	<button id="submit-btn" onclick="handleClick()">Submit</button>
	<div id="result"></div>
	<script>
		function handleClick() {
			const input = document.getElementById('text-input');
			const result = document.getElementById('result');
			result.textContent = 'Input: ' + input.value;
			document.getElementById('submit-btn').textContent = 'Clicked!';
		}
	</script>
</body>
</html>`
}

// FormTestHTML returns an HTML page with a form
func FormTestHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
	<title>Form Test Page</title>
	<meta charset="utf-8">
</head>
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
}

// DynamicContentHTML returns an HTML page with dynamically loading content
func DynamicContentHTML(delayMs int) string {
	if delayMs <= 0 {
		delayMs = 1000
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Dynamic Content Test</title>
	<meta charset="utf-8">
</head>
<body>
	<h1>Dynamic Content</h1>
	<div id="initial">Initial content</div>
	<script>
		setTimeout(function() {
			var div = document.createElement('div');
			div.id = 'dynamic';
			div.textContent = 'Dynamic content loaded';
			document.body.appendChild(div);
		}, %d);
	</script>
</body>
</html>`, delayMs)
}

// CoverageTestHTML returns an HTML page for JavaScript coverage testing
func CoverageTestHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
	<title>Coverage Test Page</title>
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
</head>
<body>
	<h1>Coverage Test</h1>
</body>
</html>`
}

// RoadmapTestHTML returns the roadmap page used in coverage tests
func RoadmapTestHTML() string {
	return `<!DOCTYPE html>
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
}

// DataURLHTML creates a data URL from HTML content
func DataURLHTML(html string) string {
	return "data:text/html," + html
}

// SimpleElementHTML returns HTML with various element types for testing
func SimpleElementHTML() string {
	return `<html><body>
		<h1 id="heading">Test Heading</h1>
		<p class="content">Test content</p>
		<button data-testid="btn">Test Button</button>
		<input id="input" type="text" value="initial">
		<ul class="list">
			<li class="item">Item 1</li>
			<li class="item">Item 2</li>
		</ul>
	</body></html>`
}

// ScreenshotTestHTML returns HTML optimized for screenshot testing
func ScreenshotTestHTML() string {
	return `<html>
	<head><style>
		body { font-family: Arial; margin: 20px; }
		.red-box { width: 100px; height: 100px; background: red; }
	</style></head>
	<body>
		<h1>Screenshot Test</h1>
		<div class="red-box"></div>
	</body>
	</html>`
}
