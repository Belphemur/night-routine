# First-Time Setup

This guide walks you through the initial setup of Night Routine Scheduler after installation.

## Prerequisites

Before starting, ensure you have:

- [x] Installed Night Routine Scheduler ([Docker](../installation/docker.md), [Docker Compose](../installation/docker-compose.md), or [locally](../installation/local.md))
- [x] Created Google OAuth credentials ([Google Calendar Setup](../configuration/google-calendar.md))
- [x] Configured environment variables ([Environment Variables](../configuration/environment.md))
- [x] Created a TOML configuration file ([TOML Configuration](../configuration/toml.md))

## Setup Steps

### 1. Start the Application

=== "Docker"
    ```bash
    docker run -d \
      --name night-routine \
      -e GOOGLE_OAUTH_CLIENT_ID=your-client-id \
      -e GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret \
      -e CONFIG_FILE=/app/config/routine.toml \
      -v ~/night-routine/config:/app/config \
      -v ~/night-routine/data:/app/data \
      -p 8080:8080 \
      ghcr.io/belphemur/night-routine:latest
    ```

=== "Docker Compose"
    ```bash
    docker-compose up -d
    ```

=== "Local Binary"
    ```bash
    export GOOGLE_OAUTH_CLIENT_ID=your-client-id
    export GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
    export CONFIG_FILE=configs/routine.toml
    ./night-routine
    ```

You should see log output indicating the application has started:

```
INF Starting Night Routine Scheduler
INF Connecting to database file=data/state.db
INF Web server listening on :8080
```

### 2. Access the Web Interface

Open your web browser and navigate to the configured `app_url`:

```
http://localhost:8080
```

You should see the home page with the "Connect Google Calendar" button.

### 3. Connect to Google Calendar

1. Click the **"Connect Google Calendar"** button on the home page
2. You'll be redirected to Google's OAuth consent screen
3. **Select your Google account**
4. **Review the permissions** requested:
    - See, edit, share, and permanently delete all calendars you can access using Google Calendar
    - View and edit events on all your calendars
5. Click **"Allow"** to grant permissions

!!! info "Unverified App Warning"
    If your app is in "Testing" status in Google Cloud Console, you'll see a warning. Click "Advanced" → "Go to Night Routine Scheduler (unsafe)" to proceed. This is expected for apps in testing mode.

6. After granting permissions, you'll be automatically redirected back to the application

### 4. Select a Calendar

After authentication, you're redirected to the **Calendar Selection** page.

1. **Review available calendars** - All calendars from your Google account are listed
2. **Choose a calendar:**
    - Use your primary calendar, or
    - Create a dedicated "Night Routine" calendar in Google Calendar first
3. **Click the calendar** you want to use

!!! tip "Dedicated Calendar Recommended"
    We recommend creating a dedicated "Night Routine" calendar in Google Calendar to keep these events separate from your other events.

4. The application will:
    - Save your calendar selection
    - Set up webhook notifications for real-time updates
    - Return you to the home page

### 5. Review the Initial Schedule

Back on the home page, you'll see:

- **Authentication status:** ✅ Connected
- **Monthly calendar view** with color-coded assignments:
    - **Blue** = Parent A
    - **Orange** = Parent B
    - **Yellow highlight** = Today
- **Assignment details:**
    - Hover over dates (desktop) to see decision reasons
    - Tap dates (mobile) to toggle decision reasons

The application has automatically:

- Generated assignments for the next 30 days (or your configured `look_ahead_days`)
- Created events in your Google Calendar
- Applied the fairness algorithm to ensure equitable distribution

### 6. Verify Google Calendar Events

1. Open [Google Calendar](https://calendar.google.com)
2. Check the selected calendar
3. You should see events like:
    - `[Parent1] 🌃👶Routine`
    - `[Parent2] 🌃👶Routine`

4. Click on an event to see:
    - **Title:** Parent name with routine emoji
    - **All-day event**
    - **Description:** Decision reason (e.g., "Alternating", "Total Count")

## Initial Configuration Verification

### Check the Schedule is Fair

1. Click **"View Statistics"** on the home page
2. Review the assignment counts for the current month
3. Verify both parents have similar numbers (should be within 1-2 assignments)

### Test Manual Sync

1. Click **"Sync Now"** on the home page
2. Wait for the sync to complete
3. Verify the calendar updates (if any changes were made)

### Verify Availability Constraints

Check that your configured unavailable days are respected:

1. Look at the calendar view
2. Find days marked as unavailable in your configuration
3. Verify the unavailable parent is never assigned on those days

Example: If Parent A is unavailable on Wednesdays, all Wednesday entries should show Parent B.

## Common First-Time Issues

### OAuth Callback Error (redirect_uri_mismatch)

**Symptom:** Error 400 when trying to connect to Google Calendar

**Cause:** OAuth redirect URI doesn't match

**Solution:**

1. Check your `app_url` in `routine.toml`
2. Ensure the callback URL `<app_url>/oauth/callback` is added to Authorized Redirect URIs in Google Cloud Console
3. Restart the application

### No Calendars Available

**Symptom:** Calendar selection page shows no calendars

**Cause:** 

- OAuth scopes not granted correctly
- Calendar API not enabled
- No calendars in the Google account

**Solution:**

1. Revoke access and re-authenticate
2. Verify Calendar API is enabled in Google Cloud Console
3. Create a calendar in Google Calendar if none exist

### Events Not Appearing in Google Calendar

**Symptom:** Web interface shows assignments but Google Calendar is empty

**Cause:**

- Wrong calendar selected
- Sync hasn't completed
- Calendar is hidden in Google Calendar view

**Solution:**

1. Check which calendar was selected (it's shown on the home page)
2. Click "Sync Now" to force a synchronization
3. In Google Calendar, ensure the calendar is visible (check the sidebar)

### Database Permission Errors

**Symptom:** Cannot write to database errors

**Cause:** Insufficient file permissions on the data directory

**Solution:**

```bash
# Ensure the data directory is writable
chmod 755 data

# Or for Docker
sudo chown -R 1000:1000 ~/night-routine/data
```

## Post-Setup Configuration

### Adjusting Availability

If you need to change parent availability after setup:

1. Edit `routine.toml`:
    ```toml
    [availability]
    parent_a_unavailable = ["Wednesday", "Sunday"]
    parent_b_unavailable = ["Monday"]
    ```

2. Restart the application

3. Click "Sync Now" to regenerate the schedule

### Changing Update Frequency

To change how often the schedule updates:

1. Edit `routine.toml`:
    ```toml
    [schedule]
    update_frequency = "weekly"  # or "daily" or "monthly"
    ```

2. Restart the application

### Switching Calendars

To use a different Google Calendar:

1. Click **"Change Calendar"** on the home page
2. Select a new calendar from the list
3. The application will:
    - Update all future events to the new calendar
    - Set up webhooks for the new calendar

## Webhook Verification

The application sets up webhooks for real-time calendar updates. To verify:

### Check Webhook Status

Look for log messages indicating webhook setup:

```
INF Setting up calendar notification channel
INF Notification channel created id=<channel-id>
```

### Test Webhook Functionality

1. Go to Google Calendar
2. Edit a night routine event title (e.g., change `[Parent1]` to `[Parent2]`)
3. Save the change
4. Check the application logs for webhook processing:
    ```
    INF Processing calendar change notification
    INF Override detected for date=2024-01-15
    ```

5. Return to the web interface and verify the change is reflected

!!! warning "Webhook Expiration"
    Google Calendar webhooks expire and need renewal. The application automatically manages this, but you may see renewal messages in the logs.

## Next Steps

Now that you've completed the initial setup:

- [Learn about the web interface](web-interface.md)
- [Understand manual overrides](manual-overrides.md)
- [Explore all features](../features.md)
- [View architecture details](../architecture/overview.md)

## Getting Help

If you encounter issues:

1. Check the [troubleshooting sections](../installation/docker.md#troubleshooting) in installation guides
2. Review the logs for error messages
3. Verify your configuration against the examples
4. Open an issue on [GitHub](https://github.com/Belphemur/night-routine/issues) with:
    - Application version
    - Configuration (remove sensitive data)
    - Error messages or logs
    - Steps to reproduce
