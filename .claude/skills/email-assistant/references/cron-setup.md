# Cron Setup for Email Assistant

## Crontab Entries

Add these entries to run the email digest twice daily (morning and evening):

```bash
# Edit crontab
crontab -e

# Add these lines:
0 8 * * * cd /path/to/your/project && claude -p "/email-assistant --channel daily-digest" >> /tmp/email-digest.log 2>&1
0 18 * * * cd /path/to/your/project && claude -p "/email-assistant --channel daily-digest" >> /tmp/email-digest.log 2>&1
```

Replace `/path/to/your/project` with the actual project directory path, and `daily-digest` with your target Slack channel name.

## Full `claude -p` Invocation

```bash
claude -p "/email-assistant --channel <channel-name>"
```

This runs Claude in non-interactive (pipe) mode, invokes the email-assistant skill, and exits when complete. No user interaction is required.

## macOS-Specific Considerations

### Sleep Prevention

macOS may sleep and skip cron jobs. Options:

1. **caffeinate** — Prevent sleep temporarily:
   ```bash
   0 8 * * * caffeinate -s cd /path/to/project && claude -p "/email-assistant --channel daily-digest"
   ```

2. **pmset** — Schedule wake before cron:
   ```bash
   # Wake at 7:55 AM daily
   sudo pmset repeat wake MTWRFSU 07:55:00
   ```

3. **Energy Saver settings** — In System Settings > Energy Saver, enable "Prevent your Mac from automatically sleeping when the display is off."

### launchd Alternative

For more reliable scheduling on macOS, use launchd instead of cron:

1. Create `~/Library/LaunchAgents/com.devpilot.email-digest.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.devpilot.email-digest</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>-c</string>
        <string>cd /path/to/your/project && claude -p "/email-assistant --channel daily-digest"</string>
    </array>
    <key>StartCalendarInterval</key>
    <array>
        <dict>
            <key>Hour</key>
            <integer>8</integer>
            <key>Minute</key>
            <integer>0</integer>
        </dict>
        <dict>
            <key>Hour</key>
            <integer>18</integer>
            <key>Minute</key>
            <integer>0</integer>
        </dict>
    </array>
    <key>StandardOutPath</key>
    <string>/tmp/email-digest.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/email-digest-error.log</string>
</dict>
</plist>
```

2. Load the agent:
```bash
launchctl load ~/Library/LaunchAgents/com.devpilot.email-digest.plist
```

3. Verify it is loaded:
```bash
launchctl list | grep devpilot
```

launchd will wake the machine if needed and is the recommended approach on macOS.
