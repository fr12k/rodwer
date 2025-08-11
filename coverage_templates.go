package rodwer

import (
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// getFilterOptions returns CoverageFilterOptions based on the specified profile
func getFilterOptions(profile string) CoverageFilterOptions {
	options := CoverageFilterOptions{
		ExcludeEmptyURLs:                true,
		ExcludeDevTools:                 true,
		ExcludeBrowserExt:               true,
		ExcludeFrameworkTools:           true,
		ExcludeCDNLibraries:             true,
		ExcludeMinifiedCode:             true,
		ExcludeTestFrameworks:           true,
		ExcludeHighDensityInlineScripts: true,
		ExcludeInlineSystemScripts:      true,
		MinScriptSize:                   30,
		MaxStatementsPerLine:            50,
		CustomExcludePatterns:           []string{},
		CustomIncludePatterns:           []string{},
	}

	switch profile {
	case "development":
		options.ExcludeFrameworkTools = false
		options.ExcludeMinifiedCode = false
		options.ExcludeTestFrameworks = false
		options.ExcludeHighDensityInlineScripts = false
		options.MinScriptSize = 10
		options.MaxStatementsPerLine = 100
	case "production":
		options.MinScriptSize = 50
		options.MaxStatementsPerLine = 5
	case "application":
		options.MinScriptSize = 15
		options.MaxStatementsPerLine = 5
	}

	return options
}

// isApplicationScript determines if a script should be included in coverage reports
func isApplicationScript(scriptCoverage *proto.ProfilerScriptCoverage, source string, options CoverageFilterOptions) (bool, string) {
	// Check custom include patterns first
	for _, pattern := range options.CustomIncludePatterns {
		if strings.Contains(strings.ToLower(scriptCoverage.URL), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(source), strings.ToLower(pattern)) {
			return true, "custom_include"
		}
	}

	// Block all inline scripts
	if strings.HasPrefix(scriptCoverage.URL, "inline-script-") {
		return false, "inline_script_blocked"
	}

	// Exclude scripts with empty URLs
	if options.ExcludeEmptyURLs && scriptCoverage.URL == "" {
		return false, "empty_url"
	}

	// Exclude browser extensions
	if options.ExcludeBrowserExt && (strings.Contains(scriptCoverage.URL, "chrome-extension://") ||
		strings.Contains(scriptCoverage.URL, "moz-extension://") ||
		strings.Contains(scriptCoverage.URL, "safari-extension://")) {
		return false, "browser_extension"
	}

	// Exclude DevTools patterns
	if options.ExcludeDevTools {
		devToolsPatterns := []string{"functions.selectable", "functions.element", "f.toString", "__coverage__", "webdriver", "puppeteer", "playwright", "rod"}
		sourceLower := strings.ToLower(source)
		for _, pattern := range devToolsPatterns {
			if strings.Contains(sourceLower, strings.ToLower(pattern)) {
				return false, "devtools_framework"
			}
		}
	}

	// Exclude very small scripts
	if len(strings.TrimSpace(source)) < options.MinScriptSize {
		return false, "too_small"
	}

	// More filtering logic would go here...

	return true, "application_script"
}

// Template constants for coverage report generation

const filteringStatsTemplate = `
<div class="bg-white rounded-lg shadow-md mb-8">
    <div class="px-6 py-4 border-b border-gray-200">
        <h2 class="text-xl font-semibold text-gray-900 flex items-center">
            🔍 Filtering Statistics
            <span class="ml-2 text-sm font-normal text-gray-500">
                (Processing time: {{printf "%.1f" .ProcessingTimeMs}}ms, avg: {{printf "%.2f" .AverageTimePerScript}}ms per script)
            </span>
        </h2>
    </div>
    <div class="p-6">
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">{{range .Reasons}}
            <div class="bg-gray-50 rounded-lg p-4">
                <div class="flex items-center justify-between mb-2">
                    <span class="text-sm font-medium text-gray-700">{{.Icon}} {{.Description}}</span>
                    <span class="text-lg font-bold text-gray-900">{{.Count}}</span>
                </div>
                <div class="text-xs text-gray-500 mb-2">{{printf "%.1f" .Percentage}}% of scripts</div>
                <div class="bg-gray-200 rounded-full h-2">
                    <div class="bg-blue-600 h-2 rounded-full" style="width: {{printf "%.1f" .Percentage}}%"></div>
                </div>
            </div>{{end}}
        </div>
    </div>
</div>`

const summaryCardsTemplate = `{{range .}}
<div class="bg-white rounded-lg shadow-md p-6 {{.BgColor}}">
    <div class="flex items-center justify-between">
        <div>
            <p class="text-sm font-medium text-gray-600">{{.Icon}} {{.Title}}</p>
            <p class="text-2xl font-bold text-gray-900">{{printf "%.1f" .Pct}}%</p>
            <p class="text-xs text-gray-500">{{.Covered}}/{{.Total}} covered</p>
        </div>
        <div class="text-2xl">{{.Icon}}</div>
    </div>
    <div class="mt-4">
        <div class="bg-gray-200 rounded-full h-2">
            <div class="bg-blue-600 h-2 rounded-full" style="width: {{printf "%.1f" .Pct}}%"></div>
        </div>
    </div>
</div>{{end}}`

const fileTableTemplate = `{{range .}}
<tr class="hover:bg-gray-50 cursor-pointer" onclick="toggleFile('file-{{.ScriptID}}')">
    <td class="px-6 py-4 text-sm text-blue-600 hover:text-blue-800">{{.FileName}}</td>
    <td class="px-6 py-4 text-sm text-gray-900">
        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.StmtBadgeColor}}">
            {{printf "%.1f" .Metrics.Statements.Pct}}% ({{.Metrics.Statements.Covered}}/{{.Metrics.Statements.Total}})
        </span>
    </td>
    <td class="px-6 py-4 text-sm text-gray-900">
        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.FuncBadgeColor}}">
            {{printf "%.1f" .Metrics.Functions.Pct}}% ({{.Metrics.Functions.Covered}}/{{.Metrics.Functions.Total}})
        </span>
    </td>
    <td class="px-6 py-4 text-sm text-gray-900">
        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.LinesBadgeColor}}">
            {{printf "%.1f" .Metrics.Lines.Pct}}% ({{.Metrics.Lines.Covered}}/{{.Metrics.Lines.Total}})
        </span>
    </td>
</tr>{{end}}`

const fileDetailsTemplate = `{{range .}}
<div id="file-{{.ScriptID}}" class="hidden bg-white rounded-lg shadow-md mb-6">
    <div class="px-6 py-4 border-b border-gray-200">
        <h3 class="text-lg font-semibold text-gray-900">{{.FileName}}</h3>
        <div class="mt-2 flex space-x-4 text-sm text-gray-600">
            <span>Statements: {{printf "%.1f" .Metrics.Statements.Pct}}%</span>
            <span>Functions: {{printf "%.1f" .Metrics.Functions.Pct}}%</span>
            <span>Lines: {{printf "%.1f" .Metrics.Lines.Pct}}%</span>
        </div>
    </div>
    <div class="p-0">
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <tbody>{{.SourceLines}}</tbody>
            </table>
        </div>
    </div>
</div>{{end}}`

const sourceLineTemplate = `{{range .}}
<tr class="{{.LineClass}}">
    <td class="line-number px-4 py-1 text-right text-gray-500 select-none w-16">{{.LineNumber}}</td>
    <td class="px-4 py-1">
        <pre class="whitespace-pre-wrap font-mono text-xs"><code class="language-javascript">{{.EscapedLine}}</code></pre>
    </td>
</tr>{{end}}`

// Template data structures

type reasonData struct {
	Icon        string
	Description string
	Count       int
	Percentage  float64
}

type filteringData struct {
	ProcessingTimeMs     float64
	AverageTimePerScript float64
	Reasons              []reasonData
}

type cardData struct {
	Title   string
	Icon    string
	Pct     float64
	Covered int
	Total   int
	BgColor string
}

type fileData struct {
	ScriptID        string
	FileName        string
	Metrics         CoverageMetrics
	StmtBadgeColor  string
	FuncBadgeColor  string
	LinesBadgeColor string
	SourceLines     string
}

type lineData struct {
	LineNumber  int
	LineClass   string
	EscapedLine string
}

// Template generation functions

func generateFilteringStats(stats FilteringStats) string {
	if len(stats.FilterReasons) == 0 {
		return ""
	}

	var reasons []reasonData
	for reason, count := range stats.FilterReasons {
		icon, description := getFilterReasonDetails(reason)
		reasons = append(reasons, reasonData{
			Icon:        icon,
			Description: description,
			Count:       count,
			Percentage:  float64(count) / float64(stats.TotalScripts) * 100,
		})
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i].Count > reasons[j].Count })

	tmpl := template.Must(template.New("filtering").Parse(filteringStatsTemplate))
	data := filteringData{
		ProcessingTimeMs:     float64(stats.ProcessingTimeMs),
		AverageTimePerScript: stats.AverageTimePerScript,
		Reasons:              reasons,
	}

	var buf strings.Builder
	tmpl.Execute(&buf, data)
	return buf.String()
}

func getFilterReasonDetails(reason string) (string, string) {
	reasonMap := map[string]struct {
		icon string
		desc string
	}{
		"application_script":    {"✅", "Application Scripts"},
		"empty_url":             {"🚫", "Empty URLs (Browser Internals)"},
		"browser_extension":     {"🧩", "Browser Extensions"},
		"devtools_framework":    {"🔧", "DevTools & Automation"},
		"framework_tools":       {"⚛️", "Framework Development Tools"},
		"cdn_library":           {"🌐", "CDN Libraries"},
		"minified_code":         {"📦", "Minified Code"},
		"generated_code":        {"🤖", "Auto-Generated Code"},
		"minified_heuristic":    {"🔍", "Minified (Heuristic)"},
		"test_framework":        {"🧪", "Test Frameworks"},
		"browser_internal":      {"🔒", "Browser Internal Scripts"},
		"too_small":             {"📏", "Scripts Too Small"},
		"source_unavailable":    {"❌", "Source Unavailable"},
		"custom_exclude":        {"⚙️", "Custom Exclusions"},
		"custom_include":        {"✨", "Custom Inclusions"},
		"high_density_inline":   {"📊", "High-Density Inline Scripts"},
		"inline_system_script":  {"🔧", "Inline System Scripts"},
		"inline_script_blocked": {"🚫", "Inline Scripts (All Blocked)"},
	}

	if details, exists := reasonMap[reason]; exists {
		return details.icon, details.desc
	}
	return "❓", reason
}

func generateSummaryCards(metrics CoverageMetrics) string {
	cards := []cardData{
		{"Statements", "📊", metrics.Statements.Pct, metrics.Statements.Covered, metrics.Statements.Total, getCoverageColor(metrics.Statements.Pct)},
		{"Functions", "⚡", metrics.Functions.Pct, metrics.Functions.Covered, metrics.Functions.Total, getCoverageColor(metrics.Functions.Pct)},
		{"Lines", "📝", metrics.Lines.Pct, metrics.Lines.Covered, metrics.Lines.Total, getCoverageColor(metrics.Lines.Pct)},
		{"Overall", "🎯", (metrics.Statements.Pct + metrics.Functions.Pct + metrics.Lines.Pct) / 3, 0, 0, getCoverageColor((metrics.Statements.Pct + metrics.Functions.Pct + metrics.Lines.Pct) / 3)},
	}

	tmpl := template.Must(template.New("cards").Parse(summaryCardsTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, cards)
	return buf.String()
}

func generateFileTable(entries []FileEntry) string {
	var files []fileData
	for _, entry := range entries {
		fileName := entry.URL
		if fileName == "" {
			fileName = fmt.Sprintf("Script %s", entry.ScriptID)
		}
		files = append(files, fileData{
			ScriptID:        string(entry.ScriptID),
			FileName:        fileName,
			Metrics:         entry.Metrics,
			StmtBadgeColor:  getCoverageBadgeColor(entry.Metrics.Statements.Pct),
			FuncBadgeColor:  getCoverageBadgeColor(entry.Metrics.Functions.Pct),
			LinesBadgeColor: getCoverageBadgeColor(entry.Metrics.Lines.Pct),
		})
	}

	tmpl := template.Must(template.New("fileTable").Parse(fileTableTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, files)
	return buf.String()
}

func generateFileDetails(entries []FileEntry) string {
	var files []fileData
	for _, entry := range entries {
		fileName := entry.URL
		if fileName == "" {
			fileName = fmt.Sprintf("Script %s", entry.ScriptID)
		}
		files = append(files, fileData{
			ScriptID:    string(entry.ScriptID),
			FileName:    fileName,
			Metrics:     entry.Metrics,
			SourceLines: generateSourceLines(entry),
		})
	}

	tmpl := template.Must(template.New("fileDetails").Parse(fileDetailsTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, files)
	return buf.String()
}

func generateSourceLines(entry FileEntry) string {
	sourceLen := len(entry.Source)
	coverage := make([]bool, sourceLen)
	for _, r := range entry.Ranges {
		if r.Count > 0 {
			for i := r.StartOffset; i < r.EndOffset && i < sourceLen; i++ {
				coverage[i] = true
			}
		}
	}

	var lines []lineData
	for lineNum, line := range entry.Lines {
		lineStart := 0
		for i := 0; i < lineNum; i++ {
			lineStart += len(entry.Lines[i]) + 1
		}
		lineEnd := lineStart + len(line)

		lineClass := ""
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			lineCovered := false
			for k := lineStart; k < lineEnd && k < len(coverage); k++ {
				if coverage[k] {
					lineCovered = true
					break
				}
			}
			if lineCovered {
				lineClass = "line-covered"
			} else {
				lineClass = "line-uncovered"
			}
		}

		lines = append(lines, lineData{
			LineNumber:  lineNum + 1,
			LineClass:   lineClass,
			EscapedLine: strings.Replace(strings.Replace(line, "<", "&lt;", -1), ">", "&gt;", -1),
		})
	}

	tmpl := template.Must(template.New("sourceLines").Parse(sourceLineTemplate))
	var buf strings.Builder
	tmpl.Execute(&buf, lines)
	return buf.String()
}

// Helper functions for coverage calculations

func getCoverageColor(pct float64) string {
	switch {
	case pct >= 80:
		return "coverage-high"
	case pct >= 60:
		return "coverage-medium"
	default:
		return "coverage-low"
	}
}

func getCoverageBadgeColor(pct float64) string {
	switch {
	case pct >= 80:
		return "bg-green-100 text-green-800"
	case pct >= 60:
		return "bg-yellow-100 text-yellow-800"
	default:
		return "bg-red-100 text-red-800"
	}
}

func calculatePct(covered, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}

func calculateCoverageMetrics(source string, ranges []*proto.ProfilerCoverageRange, functions []*proto.ProfilerFunctionCoverage) CoverageMetrics {
	sourceLen := len(source)
	lines := strings.Split(source, "\n")

	// Create coverage map
	coverage := make([]bool, sourceLen)
	for _, r := range ranges {
		if r.Count > 0 {
			for i := r.StartOffset; i < r.EndOffset && i < sourceLen; i++ {
				coverage[i] = true
			}
		}
	}

	// Calculate statements coverage (simplified as character-based)
	coveredChars := 0
	for _, covered := range coverage {
		if covered {
			coveredChars++
		}
	}

	// Calculate lines coverage
	linesCovered := 0
	executableLines := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue // Skip empty lines and comments
		}
		executableLines++

		// Check if any part of this line is covered
		lineStart := 0
		for j := 0; j < i; j++ {
			lineStart += len(lines[j]) + 1 // +1 for newline
		}
		lineEnd := lineStart + len(line)

		lineCovered := false
		for k := lineStart; k < lineEnd && k < len(coverage); k++ {
			if coverage[k] {
				lineCovered = true
				break
			}
		}
		if lineCovered {
			linesCovered++
		}
	}

	// Functions coverage (count each function individually)
	functionsCovered := 0
	functionCount := len(functions)

	for _, fn := range functions {
		// Check if this function has any covered ranges
		hasCoverage := false
		for _, r := range fn.Ranges {
			if r.Count > 0 {
				hasCoverage = true
				break
			}
		}
		if hasCoverage {
			functionsCovered++
		}
	}

	return CoverageMetrics{
		Statements: CoverageStat{
			Total:   sourceLen,
			Covered: coveredChars,
			Pct:     calculatePct(coveredChars, sourceLen),
		},
		Functions: CoverageStat{
			Total:   functionCount,
			Covered: functionsCovered,
			Pct:     calculatePct(functionsCovered, functionCount),
		},
		Lines: CoverageStat{
			Total:   executableLines,
			Covered: linesCovered,
			Pct:     calculatePct(linesCovered, executableLines),
		},
	}
}

// filterApplicationScriptsWithStats filters scripts and returns detailed statistics
func filterApplicationScriptsWithStats(scripts []*proto.ProfilerScriptCoverage, sources map[int]string, options CoverageFilterOptions) ([]int, FilteringStats) {
	startTime := time.Now()

	var applicationScripts []int
	stats := FilteringStats{
		TotalScripts:  len(scripts),
		FilterReasons: make(map[string]int),
	}

	for i, script := range scripts {
		source := sources[i]
		if source == "" {
			stats.FilterReasons["source_unavailable"]++
			continue
		}

		isApp, reason := isApplicationScript(script, source, options)
		stats.FilterReasons[reason]++

		if isApp {
			applicationScripts = append(applicationScripts, i)
		}
	}

	stats.ApplicationScripts = len(applicationScripts)
	stats.FilteredOut = stats.TotalScripts - stats.ApplicationScripts

	// Calculate timing metrics
	processingTime := time.Since(startTime)
	stats.ProcessingTimeMs = processingTime.Nanoseconds() / 1000000
	if stats.TotalScripts > 0 {
		stats.AverageTimePerScript = float64(stats.ProcessingTimeMs) / float64(stats.TotalScripts)
	}

	return applicationScripts, stats
}
