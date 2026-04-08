## MODIFIED Requirements

### Requirement: Graceful catalog fetch failure
The system SHALL display an error message and fall back to showing only installed skills when the catalog fetch fails (network error, HTTP error, etc.).

#### Scenario: Network error during catalog fetch
- **WHEN** user runs `devpilot skill list` and the raw URL is unreachable
- **THEN** the system prints a warning about the catalog fetch failure
- **AND** the system falls back to displaying installed skills only
