# Testing

Night Routine Scheduler includes comprehensive tests to ensure reliability and correctness.

## Test Structure

Tests are organized alongside the code they test:

```
internal/
├── fairness/
│   ├── fairness.go
│   └── fairness_test.go
├── database/
│   ├── database.go
│   └── database_test.go
└── ...
```

## Running Tests

### All Tests

```bash
go test -v ./...
```

### Specific Package

```bash
go test -v ./internal/fairness/
```

### Specific Test

```bash
go test -v -run TestFairnessAlgorithm ./internal/fairness/
```

### With Coverage

```bash
go test -v -cover ./...
```

### Race Detection

```bash
go test -v -race ./...
```

## Test Types

### Unit Tests

Test individual functions and components in isolation.

**Location:** `*_test.go` files next to source code

**Example:**
```go
func TestAssignmentDecision(t *testing.T) {
    t.Run("unavailability takes precedence", func(t *testing.T) {
        // Arrange
        config := Config{
            ParentA: "Alice",
            ParentB: "Bob",
            Availability: Availability{
                ParentAUnavailable: []string{"Monday"},
            },
        }
        
        // Act
        assignment := DecideAssignment(config, Monday)
        
        // Assert
        assert.Equal(t, "Bob", assignment.Parent)
        assert.Equal(t, Unavailability, assignment.Reason)
    })
}
```

### Integration Tests

Test interaction between components.

**Example:** Database operations, API calls, webhook handling

```go
func TestDatabaseAssignmentStorage(t *testing.T) {
    // Create test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Create and store assignment
    assignment := Assignment{
        Date: time.Now(),
        Parent: "Alice",
        Reason: "Alternating",
    }
    err := db.SaveAssignment(assignment)
    
    // Verify storage
    assert.NoError(t, err)
    retrieved, err := db.GetAssignment(assignment.Date)
    assert.NoError(t, err)
    assert.Equal(t, assignment.Parent, retrieved.Parent)
}
```

### Table-Driven Tests

Test multiple scenarios with the same logic.

```go
func TestParentNameValidation(t *testing.T) {
    tests := []struct {
        name      string
        parentA   string
        parentB   string
        wantError bool
    }{
        {
            name:      "valid names",
            parentA:   "Alice",
            parentB:   "Bob",
            wantError: false,
        },
        {
            name:      "duplicate names",
            parentA:   "Alice",
            parentB:   "Alice",
            wantError: true,
        },
        {
            name:      "empty name",
            parentA:   "",
            parentB:   "Bob",
            wantError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateParentNames(tt.parentA, tt.parentB)
            if tt.wantError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Test Helpers

### Database Test Helpers

```go
func setupTestDB(t *testing.T) *Database {
    t.Helper()
    dbPath := fmt.Sprintf("test_%d.db", time.Now().UnixNano())
    db, err := Open(dbPath)
    require.NoError(t, err)
    return db
}

func cleanupTestDB(t *testing.T, db *Database) {
    t.Helper()
    dbPath := db.Path()
    db.Close()
    os.Remove(dbPath)
    os.Remove(dbPath + "-shm")
    os.Remove(dbPath + "-wal")
}
```

### Time Mocking

```go
type MockClock struct {
    CurrentTime time.Time
}

func (m *MockClock) Now() time.Time {
    return m.CurrentTime
}

func TestTimeDependent(t *testing.T) {
    clock := &MockClock{
        CurrentTime: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
    }
    
    // Use mock clock in tests
    result := CalculateWithClock(clock)
    assert.Equal(t, expected, result)
}
```

## Coverage

### Generate Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Coverage by Package

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -E '^github.com'
```

### Coverage Goals

| Component | Target Coverage |
|-----------|----------------|
| Core Logic (fairness, scheduler) | 80%+ |
| Database Operations | 70%+ |
| HTTP Handlers | 60%+ |
| Configuration | 70%+ |
| Utilities | 50%+ |

## Continuous Integration

Tests run automatically on every push and pull request via GitHub Actions.

**Workflow:** `.github/workflows/ci.yml`

### CI Test Steps

1. **Lint** - Code style and quality checks
2. **Unit Tests** - All unit tests with race detection
3. **Coverage** - Upload to Codecov
4. **Build** - Verify binary builds successfully

### Viewing CI Results

- Check the "Actions" tab on GitHub
- View test results in pull request checks
- Coverage reports on Codecov

## Benchmarking

### Running Benchmarks

```bash
go test -bench=. -benchmem ./...
```

### Writing Benchmarks

```go
func BenchmarkFairnessCalculation(b *testing.B) {
    config := setupBenchConfig()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = CalculateFairness(config)
    }
}
```

### Comparing Benchmarks

```bash
# Run baseline
go test -bench=. -benchmem > old.txt

# Make changes
# ...

# Run new benchmarks
go test -bench=. -benchmem > new.txt

# Compare
benchstat old.txt new.txt
```

## Mock Objects

### HTTP Client Mocking

```go
type MockHTTPClient struct {
    DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    if m.DoFunc != nil {
        return m.DoFunc(req)
    }
    return nil, errors.New("DoFunc not set")
}

func TestGoogleCalendarAPI(t *testing.T) {
    mockClient := &MockHTTPClient{
        DoFunc: func(req *http.Request) (*http.Response, error) {
            // Return mock response
            return &http.Response{
                StatusCode: 200,
                Body: io.NopCloser(strings.NewReader(`{"items": []}`)),
            }, nil
        },
    }
    
    service := NewCalendarService(mockClient)
    // Test with mock client
}
```

## Test Data

### Fixture Data

```go
func loadTestConfig(t *testing.T, filename string) Config {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", filename))
    require.NoError(t, err)
    
    var config Config
    err = toml.Unmarshal(data, &config)
    require.NoError(t, err)
    
    return config
}
```

### Testdata Directory

```
internal/fairness/
├── fairness.go
├── fairness_test.go
└── testdata/
    ├── config1.toml
    ├── config2.toml
    └── assignments.json
```

## Common Testing Patterns

### Setup and Teardown

```go
func TestMain(m *testing.M) {
    // Setup
    setupTestEnvironment()
    
    // Run tests
    code := m.Run()
    
    // Teardown
    cleanupTestEnvironment()
    
    os.Exit(code)
}
```

### Subtests for Organization

```go
func TestAssignmentLogic(t *testing.T) {
    t.Run("group1", func(t *testing.T) {
        t.Run("test1", func(t *testing.T) { /* ... */ })
        t.Run("test2", func(t *testing.T) { /* ... */ })
    })
    
    t.Run("group2", func(t *testing.T) {
        t.Run("test3", func(t *testing.T) { /* ... */ })
    })
}
```

### Parallel Tests

```go
func TestIndependentLogic(t *testing.T) {
    t.Parallel()  // Can run in parallel with other tests
    
    // Test logic
}
```

## Debugging Tests

### Verbose Output

```bash
go test -v ./...
```

### Print Debug Info

```go
func TestSomething(t *testing.T) {
    t.Logf("Debug info: %v", value)
    // Test continues...
}
```

### Run Specific Test Only

```bash
go test -v -run TestName ./package
```

### Test with Delve

```bash
dlv test ./internal/fairness -- -test.run TestFairness
```

## Best Practices

### 1. Test Independence

Each test should be independent and not rely on other tests:

```go
// Good - Independent
func TestA(t *testing.T) {
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    // Test logic
}

// Bad - Depends on external state
var sharedDB *Database
func TestB(t *testing.T) {
    // Uses sharedDB - not independent!
}
```

### 2. Clear Test Names

```go
// Good - Describes what is being tested
func TestAssignment_WhenBothParentsAvailable_AlternatesParents(t *testing.T) { }

// Bad - Unclear what is being tested
func TestAssignment1(t *testing.T) { }
```

### 3. Arrange-Act-Assert Pattern

```go
func TestCalculation(t *testing.T) {
    // Arrange - Set up test data
    input := 5
    expected := 25
    
    // Act - Execute the code under test
    result := Square(input)
    
    // Assert - Verify the result
    assert.Equal(t, expected, result)
}
```

### 4. Test Edge Cases

```go
func TestDivision(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"normal", 10, 2, 5},
        {"zero dividend", 0, 5, 0},
        {"divide by one", 10, 1, 10},
        // Edge cases
        {"negative numbers", -10, -2, 5},
        {"large numbers", 1000000, 2, 500000},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Divide(tt.a, tt.b)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 5. Use Test Helpers

```go
func requireNoError(t *testing.T, err error) {
    t.Helper()  // Marks this as a helper function
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

## Troubleshooting Tests

### Database Locked

**Problem:** Tests fail with "database is locked"

**Solution:**
- Use unique database files per test
- Properly close databases in cleanup
- Don't run parallel tests that share database

### Flaky Tests

**Problem:** Tests pass sometimes, fail other times

**Causes:**
- Time-dependent logic without mocking
- Race conditions
- External dependencies (network, filesystem)

**Solutions:**
- Mock time/clock
- Use `-race` flag to detect races
- Mock external dependencies

### Slow Tests

**Problem:** Tests take too long to run

**Solutions:**
- Use `t.Parallel()` for independent tests
- Mock slow operations (network, file I/O)
- Use in-memory databases for tests
- Run specific packages instead of all tests

## Next Steps

- [Set up local development](local.md)
- [Read contributing guidelines](contributing.md)
- [Understand the architecture](../architecture/overview.md)
