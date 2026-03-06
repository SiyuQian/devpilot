## ADDED Requirements

### Requirement: Slack OAuth login
The system SHALL authenticate with Slack via OAuth V2 browser flow using the shared `auth.StartFlow` component with bot token scopes (chat:write, channels:read).

#### Scenario: Successful login
- **WHEN** user runs `devpilot login slack`
- **THEN** the system opens the browser to Slack OAuth consent, completes the flow, stores the bot access_token, team_id, and team_name, and prints "Logged in to Slack (workspace: <team_name>)."

#### Scenario: Login with existing credentials
- **WHEN** user runs `devpilot login slack` while already logged in
- **THEN** the system SHALL overwrite existing credentials with the new token

### Requirement: Slack logout
The system SHALL remove stored Slack credentials when the user logs out.

#### Scenario: Successful logout
- **WHEN** user runs `devpilot logout slack`
- **THEN** the system removes Slack credentials from storage and prints "Logged out of Slack."

### Requirement: Slack login status
The system SHALL report Slack authentication status through the existing `devpilot status` command.

#### Scenario: Status when logged in
- **WHEN** user runs `devpilot status` and Slack credentials exist
- **THEN** the system SHALL show "slack: logged in (<team_name>)"

#### Scenario: Status when not logged in
- **WHEN** user runs `devpilot status` and no Slack credentials exist
- **THEN** the system SHALL show "slack: not logged in"
