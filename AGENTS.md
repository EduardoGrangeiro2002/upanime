# Project Rules

## Testing
- Every feature must have unit tests and integration tests. Integration tests are the priority.
- Integration tests must use the real database service, connected to a fake/test database — never mock the database layer.

## Code Style
- Keep functions small and focused — no giant functions or God Classes.
- Prefer pure functions whenever possible.
- Use guard clauses with early return — never use if/else or if/else if/else chains.
