package rodwer

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// SourceProvider is a function type that provides source code for a given script index and ScriptCoverage
type SourceProvider func(index int, script *proto.ProfilerScriptCoverage) (string, error)

// CoverageMetrics represents coverage statistics
type CoverageMetrics struct {
	Statements CoverageStat `json:"statements"`
	Branches   CoverageStat `json:"branches"`
	Functions  CoverageStat `json:"functions"`
	Lines      CoverageStat `json:"lines"`
}

// CoverageStat represents a single coverage metric
type CoverageStat struct {
	Total   int     `json:"total"`
	Covered int     `json:"covered"`
	Skipped int     `json:"skipped"`
	Pct     float64 `json:"pct"`
}

// FilteringStats contains filtering statistics
type FilteringStats struct {
	TotalScripts         int
	ApplicationScripts   int
	FilteredOut          int
	FilterReasons        map[string]int
	ProcessingTimeMs     int64
	AverageTimePerScript float64
}

// FileEntry represents a file with coverage information
type FileEntry struct {
	ScriptID proto.RuntimeScriptID
	URL      string
	Source   string
	Lines    []string
	Ranges   []*proto.ProfilerCoverageRange
	Metrics  CoverageMetrics
}

// CoverageReporter handles JavaScript coverage report generation
type CoverageReporter struct {
	filterOptions CoverageFilterOptions
	debugMode     bool
}

// NewCoverageReporter creates a new coverage reporter
func NewCoverageReporter() *CoverageReporter {
	return &CoverageReporter{
		filterOptions: getFilterOptions("application"),
		debugMode:     false,
	}
}

// SetDebugMode enables/disables debug logging
func (cr *CoverageReporter) SetDebugMode(enabled bool) {
	cr.debugMode = enabled
}

// SetFilterProfile sets the filtering profile for coverage reports
func (cr *CoverageReporter) SetFilterProfile(profile string) {
	cr.filterOptions = getFilterOptions(profile)
}

// GenerateReport generates a complete coverage report
func (cr *CoverageReporter) GenerateReport(entries []CoverageEntry, outputPath string) error {
	// Convert to old format for compatibility
	oldFormat := cr.convertToOldCoverageFormat(entries)

	// Create source provider
	sourceProvider := cr.createSourceProviderFromEntries(entries)

	// Generate report
	outputFunc := func(format string, args ...interface{}) {
		if cr.debugMode {
			fmt.Printf(format+"\n", args...)
		}
	}

	cr.generateJSReportUnified(oldFormat, sourceProvider, outputFunc)

	// Calculate coverage percentage
	jsPct := cr.computeJavaScriptCoverageFromEntries(entries)

	// Generate index file
	return cr.generateCoverageIndex(jsPct, outputPath)
}

// GenerateReportFromPage generates a report directly from a Rod page
func (cr *CoverageReporter) GenerateReportFromPage(page *rod.Page, raw []*proto.ProfilerScriptCoverage) FilteringStats {
	sourceProvider := func(index int, script *proto.ProfilerScriptCoverage) (string, error) {
		srcResp, err := proto.DebuggerGetScriptSource{ScriptID: script.ScriptID}.Call(page)
		if err != nil {
			return "", fmt.Errorf("failed to get script source for ScriptID %s: %w", script.ScriptID, err)
		}
		if srcResp.ScriptSource == "" {
			return "", fmt.Errorf("empty script source for ScriptID %s", script.ScriptID)
		}
		return srcResp.ScriptSource, nil
	}

	outputFunc := func(format string, args ...interface{}) {
		if format == "JavaScript coverage report written to %s" {
			fmt.Printf("✅ Wrote enhanced JS coverage report (%d application scripts): %s\n",
				len(raw), args[0])
		}
	}

	return cr.generateJSReportUnified(raw, sourceProvider, outputFunc)
}

// convertToOldCoverageFormat converts new CoverageEntry to old format for compatibility
func (cr *CoverageReporter) convertToOldCoverageFormat(entries []CoverageEntry) []*proto.ProfilerScriptCoverage {
	var result []*proto.ProfilerScriptCoverage

	for i, entry := range entries {
		scriptCov := &proto.ProfilerScriptCoverage{
			ScriptID: proto.RuntimeScriptID(fmt.Sprintf("script-%d", i)),
			URL:      entry.URL,
		}

		// Convert ranges to ProfilerFunctionCoverage format
		if len(entry.Ranges) > 0 {
			functions := make([]*proto.ProfilerFunctionCoverage, 1)
			functions[0] = &proto.ProfilerFunctionCoverage{
				FunctionName: "",
				Ranges:       make([]*proto.ProfilerCoverageRange, 0),
			}

			for _, r := range entry.Ranges {
				functions[0].Ranges = append(functions[0].Ranges, &proto.ProfilerCoverageRange{
					StartOffset: r.Start,
					EndOffset:   r.End,
					Count:       r.Count,
				})
			}
			scriptCov.Functions = functions
		}

		result = append(result, scriptCov)
	}

	return result
}

// createSourceProviderFromEntries creates a source provider from coverage entries
func (cr *CoverageReporter) createSourceProviderFromEntries(entries []CoverageEntry) SourceProvider {
	return func(index int, script *proto.ProfilerScriptCoverage) (string, error) {
		if index < 0 || index >= len(entries) {
			return "", fmt.Errorf("index %d out of range", index)
		}
		source := entries[index].Source
		if source == "" {
			return "", fmt.Errorf("source unavailable for index %d", index)
		}
		return source, nil
	}
}

// computeJavaScriptCoverageFromEntries computes coverage percentage from new format
func (cr *CoverageReporter) computeJavaScriptCoverageFromEntries(entries []CoverageEntry) float64 {
	totalBytes := 0
	coveredBytes := 0

	for _, entry := range entries {
		if entry.Source == "" {
			continue
		}

		totalBytes += len(entry.Source)

		// Calculate covered bytes from ranges
		covered := make([]bool, len(entry.Source))
		for _, r := range entry.Ranges {
			if r.Count > 0 && r.Start >= 0 && r.End <= len(entry.Source) {
				for i := r.Start; i < r.End; i++ {
					covered[i] = true
				}
			}
		}

		for _, c := range covered {
			if c {
				coveredBytes++
			}
		}
	}

	if totalBytes == 0 {
		return 0
	}

	return float64(coveredBytes) / float64(totalBytes) * 100
}

// generateJSReportUnified generates Istanbul.js-style report with flexible source fetching
func (cr *CoverageReporter) generateJSReportUnified(raw []*proto.ProfilerScriptCoverage, sourceProvider SourceProvider, outputFunc func(string, ...interface{})) FilteringStats {
	entries := make([]FileEntry, 0, len(raw))
	var totalMetrics CoverageMetrics
	var filterStats FilteringStats

	filterStats.TotalScripts = len(raw)
	filterStats.FilterReasons = make(map[string]int)

	// Process each script individually to avoid losing scripts with same URL
	for i, r := range raw {
		// Get script source using the provided strategy
		scriptSource, err := sourceProvider(i, r)
		if err != nil || scriptSource == "" {
			filterStats.FilterReasons["source_unavailable"]++
			continue
		}

		// Apply filtering logic
		isApp, reason := isApplicationScript(r, scriptSource, cr.filterOptions)
		filterStats.FilterReasons[reason]++

		if !isApp {
			continue // Skip this script
		}

		// Create unique URL identifier to distinguish scripts with same URL
		url := r.URL
		if url == "" {
			url = fmt.Sprintf("Script_%s", r.ScriptID)
		} else {
			// Add script ID to make each script entry unique
			url = fmt.Sprintf("%s#%s", url, r.ScriptID)
		}

		// Collect all ranges from all functions for this script
		var allRanges []*proto.ProfilerCoverageRange
		for _, function := range r.Functions {
			if function.Ranges != nil {
				allRanges = append(allRanges, function.Ranges...)
			}
		}

		lines := strings.Split(scriptSource, "\n")

		// Calculate metrics for this individual script
		metrics := calculateCoverageMetrics(scriptSource, allRanges, r.Functions)

		entry := FileEntry{
			ScriptID: r.ScriptID,
			URL:      url,
			Source:   scriptSource,
			Lines:    lines,
			Ranges:   allRanges,
			Metrics:  metrics,
		}

		entries = append(entries, entry)

		// Add to total metrics
		totalMetrics.Statements.Total += metrics.Statements.Total
		totalMetrics.Statements.Covered += metrics.Statements.Covered
		totalMetrics.Functions.Total += metrics.Functions.Total
		totalMetrics.Functions.Covered += metrics.Functions.Covered
		totalMetrics.Lines.Total += metrics.Lines.Total
		totalMetrics.Lines.Covered += metrics.Lines.Covered
	}

	// Calculate final filtering statistics
	filterStats.ApplicationScripts = len(entries)
	filterStats.FilteredOut = filterStats.TotalScripts - filterStats.ApplicationScripts

	// Calculate total percentages
	if totalMetrics.Statements.Total > 0 {
		totalMetrics.Statements.Pct = float64(totalMetrics.Statements.Covered) / float64(totalMetrics.Statements.Total) * 100
	}
	if totalMetrics.Functions.Total > 0 {
		totalMetrics.Functions.Pct = float64(totalMetrics.Functions.Covered) / float64(totalMetrics.Functions.Total) * 100
	}
	if totalMetrics.Lines.Total > 0 {
		totalMetrics.Lines.Pct = float64(totalMetrics.Lines.Covered) / float64(totalMetrics.Lines.Total) * 100
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].URL < entries[j].URL })

	html := generateIstanbulStyleHTML(entries, totalMetrics, filterStats)

	jsHTML := "coverage/js-coverage.html"
	_ = os.WriteFile(jsHTML, []byte(html), 0644)

	outputFunc("JavaScript coverage report written to %s", jsHTML)
	outputFunc("Coverage Summary - Statements: %.1f%%, Functions: %.1f%%, Lines: %.1f%%",
		totalMetrics.Statements.Pct, totalMetrics.Functions.Pct, totalMetrics.Lines.Pct)

	return filterStats
}

// generateCoverageIndex generates the main coverage index HTML file
func (cr *CoverageReporter) generateCoverageIndex(jsPct float64, outputPath string) error {
	if outputPath == "" {
		outputPath = "coverage/index.html"
	}

	content := fmt.Sprintf(`<!DOCTYPE html>
<html><head><title>Unified Coverage Report</title></head>
<body>
	<h1>Unified Coverage Report</h1>
	<h2>Coverage Summary</h2>
	<p>JavaScript Coverage: %.1f%%</p>
	<ul>
		<li><a href="js-coverage.html">✅ JavaScript Coverage Report</a></li>
		<li><a href="screenshot-page.png">🖼️ Screenshot - Initial</a></li>
		<li><a href="screenshot-after-click.png">🖼️ Screenshot - After Copy Click</a></li>
	</ul>
</body></html>`, jsPct)

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// HTML Report Generation

const istanbulHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JavaScript Coverage Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism.min.css" rel="stylesheet">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-javascript.min.js"></script>
    <style>
        .coverage-high { background-color: #d4edda; }
        .coverage-medium { background-color: #fff3cd; }
        .coverage-low { background-color: #f8d7da; }
        .line-covered { background-color: #d4edda; }
        .line-uncovered { background-color: #f8d7da; }
        .line-number { background-color: #f8f9fa; border-right: 1px solid #dee2e6; }
    </style>
</head>
<body class="bg-gray-50 text-gray-900">
    <div class="container mx-auto px-4 py-8">
        <div class="mb-8">
            <h1 class="text-3xl font-bold text-gray-900 mb-2">JavaScript Coverage Report</h1>
            <p class="text-gray-600">Generated on {{.Timestamp}}</p>
            <div class="mt-3 flex flex-wrap gap-4 text-sm">
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                    📁 {{.FilterStats.ApplicationScripts}} Application Scripts
                </span>
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                    🚫 {{.FilterStats.FilteredOut}} Scripts Filtered
                </span>
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                    📊 {{.FilterStats.TotalScripts}} Total Scripts Analyzed
                </span>
            </div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">{{.SummaryCards}}</div>
        {{.FilteringStats}}
        <div class="bg-white rounded-lg shadow-md mb-8">
            <div class="px-6 py-4 border-b border-gray-200">
                <h2 class="text-xl font-semibold text-gray-900">File Coverage</h2>
            </div>
            <div class="overflow-x-auto">
                <table class="min-w-full">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Statements</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Functions</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Lines</th>
                        </tr>
                    </thead>
                    <tbody class="bg-white divide-y divide-gray-200">{{.FileTable}}</tbody>
                </table>
            </div>
        </div>
        {{.FileDetails}}
    </div>
    <script>
        function toggleFile(fileId) {
            const element = document.getElementById(fileId);
            element.classList.toggle('hidden');
        }
        Prism.highlightAll();
    </script>
</body>
</html>`

// generateIstanbulStyleHTML generates the HTML report
func generateIstanbulStyleHTML(entries []FileEntry, totalMetrics CoverageMetrics, filterStats FilteringStats) string {
	tmpl := template.Must(template.New("coverage").Parse(istanbulHTMLTemplate))

	data := htmlData{
		Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		FilterStats:    filterStats,
		SummaryCards:   generateSummaryCards(totalMetrics),
		FilteringStats: generateFilteringStats(filterStats),
		FileTable:      generateFileTable(entries),
		FileDetails:    generateFileDetails(entries),
	}

	var buf strings.Builder
	tmpl.Execute(&buf, data)
	return buf.String()
}

type htmlData struct {
	Timestamp      string
	FilterStats    FilteringStats
	SummaryCards   string
	FilteringStats string
	FileTable      string
	FileDetails    string
}

// Additional types for filtering
type CoverageFilterOptions struct {
	ExcludeEmptyURLs                bool
	ExcludeDevTools                 bool
	ExcludeBrowserExt               bool
	ExcludeFrameworkTools           bool
	ExcludeCDNLibraries             bool
	ExcludeMinifiedCode             bool
	ExcludeTestFrameworks           bool
	ExcludeHighDensityInlineScripts bool
	ExcludeInlineSystemScripts      bool
	MinScriptSize                   int
	MaxStatementsPerLine            int
	CustomExcludePatterns           []string
	CustomIncludePatterns           []string
}
