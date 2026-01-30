# Testing Patterns

**Analysis Date:** 2026-01-30

## Test Framework

**Runner:**
- Backend: Go's built-in `testing` package
- Frontend: Not detected - no test framework or configuration found

**Assertion Library:**
- Backend: Standard Go testing with manual assertions (no assertion library like testify)
- Frontend: Not applicable

**Run Commands:**
```bash
go test ./...            # Run all tests in backend
go test -v ./...         # Run with verbose output
go test ./bot -run TestSQLiteStore_IncGet  # Run specific test
```

## Test File Organization

**Location:**
- Backend: Co-located in same directory as source files
  - Pattern: `store.go` paired with `store_test.go`, `store_givers_test.go`
  - Location: `/home/tdolfen/Projects/github.com/BeerBILabs/backend/bot/`

**Naming:**
- Backend: `{subject}_test.go` or `{subject}_{aspect}_test.go`
  - Example: `store_test.go` (general store tests), `store_givers_test.go` (giver-specific tests)

**Structure:**
```
backend/bot/
├── main.go
├── store.go
├── store_test.go           # Tests for store functionality
├── store_givers_test.go    # Tests for giver-specific operations
└── testdata/
    └── test.db             # Generated during tests
```

## Test Structure

**Suite Organization:**
```go
package main

import (
    "database/sql"
    "os"
    "testing"
    "time"
    "fmt"

    _ "github.com/mattn/go-sqlite3"
)

func TestSQLiteStore_IncGet(t *testing.T) {
    // Setup
    dbPath := "./testdata/test.db"
    _ = os.Remove(dbPath)
    if err := os.MkdirAll("./testdata", 0o755); err != nil {
        t.Fatalf("mkdir testdata: %v", err)
    }
    // ...

    // Assertion
    if c, _ := s.GetCount(user, emoji); c != 0 {
        t.Fatalf("expected 0, got %d", c)
    }
}
```

**Patterns:**
- Setup: Create test database, initialize store, clean up after with defer
- Assertions: Manual comparison with `if actual != expected { t.Fatalf(...) }`
- Defer cleanup: `defer func() { db.Close(); _ = os.Remove(dbPath) }()`
- Error handling: `t.Fatalf()` for test failures, stops test execution

## Mocking

**Framework:** Not used - tests use real dependencies (SQLite database)

**Patterns:**
- Real SQLite database created per test: `sql.Open("sqlite3", dbPath+"?_foreign_keys=1")`
- Database is created fresh for each test: `_ = os.Remove(dbPath)` clears old test data
- Deferred cleanup: `defer func() { db.Close(); _ = os.Remove(dbPath) }()`

**What to Mock:**
- Slack API connections: Not mocked in visible tests; likely tested at integration level
- Network calls: Implicit in `socketmode.Client` usage, not mocked in unit tests

**What NOT to Mock:**
- Database: Use real SQLite with test database files
- Store implementation: Test actual SQLiteStore with real schema

## Fixtures and Factories

**Test Data:**
```go
// From store_test.go
user := "U123"
emoji := "beer"

// From store_test.go TestSQLiteStore_Beers
giver := "U1"
recv := "U2"
now := time.Now().UTC()
ts1 := fmt.Sprintf("%d.000000", now.Unix())
ts2 := fmt.Sprintf("%d.000000", now.Add(time.Second).Unix())

// Setup store with data
if err := s.AddBeer(giver, recv, ts1, now, 1); err != nil { t.Fatalf("addbeer: %v", err) }
if err := s.AddBeer(giver, recv, ts2, now.Add(time.Second), 1); err != nil { t.Fatalf("addbeer2: %v", err) }
```

**Location:**
- Test data defined inline within test functions
- Testdata directory: `./testdata/` contains generated SQLite database files
- No separate fixtures or factory functions; data created by calling store methods

## Coverage

**Requirements:** None enforced - no coverage configuration detected

**View Coverage:**
```bash
go test ./bot -cover                    # Show coverage percentage
go test ./bot -coverprofile=coverage.out  # Generate coverage report
go tool cover -html=coverage.out        # View in browser
```

## Test Types

**Unit Tests:**
- Scope: Test individual store operations (increment counters, add beers, count operations)
- Approach: Isolated tests with fresh SQLite database per test
- Examples:
  - `TestSQLiteStore_IncGet`: Tests emoji count increment and retrieval
  - `TestSQLiteStore_Beers`: Tests beer addition and counting operations

**Integration Tests:**
- Not explicitly present in codebase
- Slack connection handling likely tested in main.go but no visible test files

**E2E Tests:**
- Not used in this codebase

## Common Patterns

**Synchronous Testing:**
- Go's standard `testing.T` is synchronous
- Async operations in production code (like socketmode) not tested in visible test suite

**Error Testing:**
```go
// Pattern: Test operations that may return errors
if c, _ := s.GetCount(user, emoji); c != 0 {
    t.Fatalf("expected 0, got %d", c)
}

// Error ignored with _ if not critical to test
// Error checked with t.Fatalf() if critical
if err := s.IncEmoji(user, emoji); err != nil {
    t.Fatalf("inc1: %v", err)
}
```

**Database Isolation:**
```go
// Pattern: Fresh database per test
func TestSQLiteStore_IncGet(t *testing.T) {
    dbPath := "./testdata/test.db"
    _ = os.Remove(dbPath)  // Remove any old test database
    // ...
    defer func() {
        db.Close()
        _ = os.Remove(dbPath)  // Cleanup after test
    }()
}
```

## Frontend Testing

**Status:** No testing framework configured

**Notes:**
- No Jest, Vitest, or other testing packages in package.json
- No .test.tsx or .spec.ts files found
- No test configuration files (jest.config.js, vitest.config.ts)
- Components are interactive but untested (UsersList, ThemeToggle, DateRangePicker)
- API routes (`app/api/health/route.ts`, `app/api/proxy/[...path]/route.ts`) are untested

**Recommendation for Future Implementation:**
- Consider adding Vitest (lightweight, TypeScript-first alternative to Jest)
- Test React components with React Testing Library or Vitest's shallow rendering
- Test API routes with node-fetch or API test utilities

---

*Testing analysis: 2026-01-30*
