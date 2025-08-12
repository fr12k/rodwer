package rodwer

// TestHTMLTemplates provides reusable HTML templates for testing
// This module has been reduced to contain only actively used templates.

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
