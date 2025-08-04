# Test Suite Performance Optimization Task

## Current Problem
Test suite takes ~120 seconds to complete, impacting developer productivity and CI/CD pipeline efficiency.

## Performance Analysis

### Major Bottlenecks Identified:
1. **Browser Creation Overhead**: Each test creates new browser instances (~3-5s each)
2. **Network Dependencies**: External URLs (example.com, httpbin.org) with 5-10s timeouts
3. **Sequential Execution**: No parallelization of independent tests
4. **TestConcurrentBrowsers**: Takes 78+ seconds alone
5. **Coverage Collection**: Complex DevTools Protocol operations add overhead

### Current Test Structure:
- `framework_test.go`: Comprehensive API tests (21.82s)
- `browser_test.go`: Core functionality + coverage (4.03s)
- `examples_test.go`: Integration examples (82+ seconds for concurrent test)

## Optimization Strategy

### 1. Test Categorization with Speed Tiers

**Quick Tests (`-short` flag) - Target: <15 seconds**
- Basic API validation
- Browser lifecycle tests
- Element selection (data URLs only)
- No network dependencies
- Reduced timeouts (1-2s)

**Fast Tests (default) - Target: <30 seconds**
- All quick tests
- Screenshot capabilities
- Local test server interactions
- Optimized browser sharing

**Full Tests (integration) - Target: <60 seconds**
- All fast tests
- Network-dependent tests
- Coverage collection
- Concurrent operations

### 2. Browser Instance Optimization

**Shared Browser Pattern:**
```go
// Instead of creating browser per test
func TestSuite(t *testing.T) {
    browser, cleanup := setupSharedBrowser()
    defer cleanup()
    
    t.Run("test1", func(t *testing.T) { /* use browser */ })
    t.Run("test2", func(t *testing.T) { /* use browser */ })
}
```

**Benefits:**
- Reduce browser creation from ~15 instances to ~3-5
- 60-80% reduction in setup overhead
- Maintain test isolation through page management

### 3. Parallel Test Execution

**Parallelizable Test Categories:**
- Browser creation tests (independent instances)
- Element interaction tests (isolated pages)
- Screenshot tests (independent operations)
- Validation tests (no shared state)

**Implementation:**
```go
func TestParallelGroup(t *testing.T) {
    t.Run("browser_creation", func(t *testing.T) {
        t.Parallel()
        // test code
    })
    t.Run("element_selection", func(t *testing.T) {
        t.Parallel()
        // test code
    })
}
```

### 4. Network Independence

**Replace External Dependencies:**
- `example.com` → embedded test server with same content
- `httpbin.org/delay/5` → local server with configurable delays
- Use `data:` URLs for simple HTML tests
- Mock network conditions instead of real delays

**Local Test Server:**
```go
func createTestServer() *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/":
            w.Write([]byte(`<html><body><h1>Example Domain</h1></body></html>`))
        case "/delay":
            time.Sleep(time.Duration(getDelayParam(r)) * time.Millisecond)
            w.Write([]byte("OK"))
        }
    }))
}
```

### 5. Timeout Optimization

**Aggressive Timeout Reduction:**
- Default timeouts: 10s → 2s
- Quick tests: 5s → 1s
- Element waits: 5s → 2s
- Use exponential backoff for retries

## Implementation Plan

### Phase 1: Quick Test Suite (High Priority)
1. Create `quick_test.go` with essential smoke tests
2. Implement shared browser pattern
3. Use data URLs exclusively
4. Target: <15 seconds execution

### Phase 2: Browser Optimization (High Priority)
1. Refactor existing tests to use shared browsers
2. Implement proper cleanup between tests
3. Add browser pooling for parallel tests
4. Target: 50% reduction in current times

### Phase 3: Parallelization (Medium Priority)
1. Add `t.Parallel()` to independent tests
2. Create parallel test groups
3. Optimize resource contention
4. Target: 30-40% additional speedup

### Phase 4: Network Independence (Medium Priority)
1. Replace all external URLs with local servers
2. Create comprehensive test fixtures
3. Mock network conditions
4. Target: Eliminate network-related timeouts

### Phase 5: Advanced Optimizations (Low Priority)
1. Browser process pooling
2. Test result caching
3. Smart test selection
4. CI/CD pipeline integration

## Expected Results

**Performance Targets:**
- Quick smoke tests: 10-15 seconds (90% improvement)
- Fast development tests: 20-30 seconds (75% improvement)
- Full test suite: 45-60 seconds (50% improvement)

**Development Experience:**
- Instant feedback for basic changes
- Fast iteration cycles
- Maintained comprehensive coverage
- Better CI/CD pipeline performance

## Commands Structure

```bash
# Ultra-fast smoke test
go test -short -v -run="Quick"

# Fast development cycle
go test -short -v

# Full test suite
go test -v

# Specific test categories
go test -v -run="Framework"
go test -v -run="Coverage"
go test -v -run="Integration"

# Parallel execution
go test -v -parallel=4
```

## Success Metrics

1. **Time Reduction**: 60-75% improvement in test execution
2. **Developer Productivity**: Sub-30s feedback for most changes
3. **CI/CD Efficiency**: Faster build pipeline
4. **Test Coverage**: Maintain 100% of current functionality
5. **Reliability**: Reduce flaky network-dependent failures

## Risks & Mitigation

**Risk**: Test coverage gaps from optimization
**Mitigation**: Comprehensive test mapping and validation

**Risk**: Parallel execution race conditions
**Mitigation**: Careful isolation and resource management

**Risk**: Over-optimization complexity
**Mitigation**: Incremental changes with performance measurement

## Next Steps

1. ✅ Write this plan to .claude/tasks/
2. 🔄 Create quick_test.go with essential smoke tests
3. ⏳ Implement shared browser pattern
4. ⏳ Add parallel execution support
5. ⏳ Replace network dependencies
6. ⏳ Measure and validate improvements