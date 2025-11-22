Generate comprehensive Go tests for {file}:

1. Table-driven tests for all exported functions
2. Subtests for different scenarios (t.Run)
3. Edge cases: nil checks, empty inputs, boundary values
4. Error path testing
5. Benchmarks for performance-critical functions

Follow patterns from:
- squads/squads_test.go (table-driven structure)
- squadcombat_test.go (combat scenarios)

Output: Complete test file with TestMain if needed
