## 1. Refactor list command to fetch and merge catalog

- [x] 1.1 Add `--installed` bool flag to `skillListCmd`
- [x] 1.2 When `--installed` is false (default), call `FetchCatalog` to get all available skills, merge with installed skills from both config levels, and display unified table with columns: NAME, DESCRIPTION, VERSION, LEVEL
- [x] 1.3 When `--installed` is true, display only installed skills (current behavior but with added DESCRIPTION column)
- [x] 1.4 Truncate descriptions longer than 40 characters with "..."
- [x] 1.5 On catalog fetch failure, print warning and fall back to installed-only view

## 2. Tests

- [x] 2.1 Test default list output shows full catalog with installed status markers
- [x] 2.2 Test `--installed` flag shows only installed skills
- [x] 2.3 Test description truncation at 40 characters
- [x] 2.4 Test graceful fallback when catalog fetch fails
