## Purpose

Defines the `devpilot slack send` command for posting messages to Slack channels, including channel name resolution.

## Requirements

### Requirement: Send message to Slack channel
The system SHALL provide a `devpilot slack send` command that sends a text message to a specified Slack channel.

#### Scenario: Send message by channel name
- **WHEN** user runs `devpilot slack send --channel "general" --message "Hello world"`
- **THEN** the system SHALL resolve the channel name to an ID, post the message via `chat.postMessage` API, and print "Message sent to #general."

#### Scenario: Channel name with hash prefix
- **WHEN** user runs `devpilot slack send --channel "#general" --message "Hello"`
- **THEN** the system SHALL strip the `#` prefix and resolve the channel name normally

#### Scenario: Message from stdin
- **WHEN** user runs `echo "Hello" | devpilot slack send --channel "general"`
- **THEN** the system SHALL read the message from stdin when `--message` flag is not provided

#### Scenario: Bot not in channel
- **WHEN** the bot has not been added to the target channel
- **THEN** the system SHALL return an error: "Bot is not a member of #<channel>. Run: /invite @devpilot in the channel."

#### Scenario: Channel not found
- **WHEN** the specified channel name does not exist
- **THEN** the system SHALL return an error: "Channel not found: <channel>"

#### Scenario: Not logged in
- **WHEN** user runs `devpilot slack send` without being logged in
- **THEN** the system SHALL return an error: "Not logged in to Slack. Run: devpilot login slack"

### Requirement: Channel name resolution
The system SHALL resolve human-readable channel names to Slack channel IDs before posting messages.

#### Scenario: Resolve public channel
- **WHEN** the system needs to post to a channel by name
- **THEN** it SHALL call `conversations.list` API to find the matching channel ID by exact name match

#### Scenario: Ambiguous channel name
- **WHEN** multiple channels match the given name
- **THEN** the system SHALL return an error listing the matches and asking the user to be more specific
