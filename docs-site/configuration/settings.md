# Configuration Settings

Night Routine Scheduler provides a user-friendly web interface for managing runtime configuration settings. This allows you to adjust parent names, availability constraints, and scheduling parameters without needing to restart the application.

## Overview

The Settings page provides a centralized location to manage all configurable aspects of your night routine scheduling:

- **Parent Names**: Define the names that appear in calendar events
- **Availability**: Set which days each parent is unavailable
- **Schedule Settings**: Configure update frequency and scheduling parameters

![Settings Page](../images/settings-page.png)

*The Settings page provides an intuitive interface for all configuration options*

## Accessing Settings

Navigate to the Settings page by:

1. Click the **⚙️ Settings** link from the home page navigation menu
2. Or directly visit: `http://your-server:port/settings`

!!! info "No Authentication Required"
    The Settings page is accessible without Google Calendar authentication, allowing you to configure the application before connecting to your calendar.

## Configuration Sections

### Parent Names

Configure the names of the two parents who will be assigned night routine duties.

**Fields:**
- **Parent A Name**: The name for the first parent (appears in calendar events)
- **Parent B Name**: The name for the second parent (appears in calendar events)

**Example:**
```
Parent A: Antoine
Parent B: Taina
```

These names will be used in:
- Calendar event titles
- Decision logs and fairness calculations
- Assignment notifications

---

### Availability

Define which days each parent is unavailable for night routine duties. This ensures the scheduler won't assign duties on days when a parent can't fulfill them.

**Configuration:**
- Select checkboxes for days when each parent is **unavailable**
- Leave checkboxes **unchecked** for available days
- If a parent is available all days, leave all checkboxes unchecked

**Example Scenario:**
```
Antoine - Unavailable: Wednesday (has evening class)
Taina - Unavailable: Tuesday, Thursday (works late shifts)
```

!!! tip "Fairness Algorithm"
    The scheduler's fairness algorithm automatically accounts for availability differences when making assignments. If one parent has more unavailable days, the algorithm ensures fair distribution on the days both parents are available.

---

### Schedule Settings

Configure how the scheduler generates and updates night routine assignments.

#### Update Frequency

How often the schedule should be automatically updated.

**Options:**
- **Daily**: Updates the schedule every day
- **Weekly**: Updates the schedule once per week (recommended)
- **Monthly**: Updates the schedule once per month

**Recommendation:** Weekly updates provide a good balance between maintaining an up-to-date schedule and avoiding excessive calendar modifications.

#### Look Ahead Days

Number of days in advance to schedule night routine events.

- **Range**: 1 to 365 days
- **Recommended**: 7-30 days
- **Default**: 7 days

**Considerations:**
- **Shorter periods (7-14 days)**: More responsive to recent changes, less advance planning
- **Longer periods (21-30 days)**: Better for advance planning, shows patterns more clearly

#### Past Event Threshold

How many days in the past to accept manual calendar changes.

- **Range**: 0 to 30 days
- **Recommended**: 3-7 days
- **Default**: 5 days

This setting allows you to manually adjust past events in your calendar (for example, if someone else covered a shift) and have the scheduler recognize those changes when recalculating fairness.

**Example:**
```
Setting: 5 days
Behavior: Manual changes made to events in the last 5 days 
          will be respected by the fairness algorithm
```

---

## Making Changes

### Save Settings

After making your desired changes:

1. Review all modifications
2. Click the **Save Settings** button
3. The application will:
   - Validate all inputs
   - Save changes to the database
   - **Automatically trigger a calendar sync** (if connected)
   - Update the schedule based on new settings

!!! warning "Important"
    Changes to settings will affect the fairness algorithm and existing schedule assignments. The calendar will be automatically synced after saving changes to reflect the new configuration.

### Cancel Changes

To discard changes without saving:

- Click the **Cancel** button
- Or click **← Back to Home**

All modifications will be discarded and the previous settings will remain active.

---

## Automatic Sync

When you save settings changes, the application automatically triggers a calendar synchronization (if you're connected to Google Calendar). This ensures your calendar is immediately updated to reflect the new configuration.

**What happens during auto-sync:**

1. Settings are validated and saved to database
2. Scheduler recalculates assignments using new settings
3. Calendar events are updated/created based on new schedule
4. Fairness algorithm considers the updated constraints

**Sync Status:**

- A success message confirms when the sync completes
- Any errors during sync are displayed with details
- You can manually trigger additional syncs from the home page if needed

---

## Configuration Persistence

### Database Storage

All settings configured through the web UI are stored in a SQLite database (`data/state.db`). This provides:

- **Persistence**: Settings survive application restarts
- **Transactionality**: Changes are atomic and consistent
- **Validation**: Database constraints ensure data integrity

### TOML File Role

The TOML configuration file (`configs/routine.toml`) is now primarily used for:

1. **Initial Seeding**: On first run, values from TOML seed the database
2. **App-Level Settings**: Port, URLs, log level (require restart to change)
3. **Reference**: Documentation of available configuration options

!!! info "Configuration Priority"
    Once the database is seeded, it becomes the authoritative source for parent, availability, and schedule settings. Changes to these sections in the TOML file are **ignored** after initial setup. Use the Settings UI instead.

---

## Automatic Migration

When upgrading from an older version without database-backed configuration:

1. Application detects empty configuration tables
2. Automatically migrates settings from TOML to database
3. Logs the migration process
4. Database becomes the authoritative source going forward

**Migration Log Example:**
```
INFO: No configuration found in database, migrating from TOML config file
INFO: Parent configuration seeded successfully
INFO: Availability configuration seeded successfully  
INFO: Schedule configuration seeded successfully
INFO: Configuration migration from TOML completed successfully
```

---

## Validation

All settings are validated both client-side and server-side:

### Parent Names
- Must not be empty
- Can contain any characters (including Unicode)
- Must be different from each other

### Availability  
- Days must be valid days of the week
- Multiple days can be selected per parent
- No validation if no days selected (available all days)

### Schedule Settings
- **Update Frequency**: Must be one of: daily, weekly, monthly
- **Look Ahead Days**: Must be between 1 and 365
- **Past Event Threshold**: Must be between 0 and 30

Invalid inputs are rejected with clear error messages indicating what needs to be corrected.

---

## Troubleshooting

### Settings Not Saving

**Problem**: Click Save but settings don't persist

**Solutions:**
- Check application logs for validation errors
- Ensure database file (`data/state.db`) is writable
- Verify no database corruption (check logs)

### Schedule Not Updating

**Problem**: Settings saved but calendar doesn't reflect changes

**Solutions:**
- Verify Google Calendar connection status (Home page)
- Check if automatic sync completed successfully
- Manually trigger sync from Home page
- Review application logs for sync errors

### Invalid Input Errors

**Problem**: Error message when trying to save

**Solutions:**
- Check that parent names are not empty and different
- Ensure look-ahead days are between 1-365
- Ensure past event threshold is between 0-30
- Verify no special characters causing issues

---

## Best Practices

1. **Test Changes**: After saving settings, review the calendar to ensure assignments look correct
2. **Document Reasons**: Keep notes about why certain availability patterns exist
3. **Regular Review**: Periodically review settings to ensure they match current needs
4. **Backup Database**: Backup `data/state.db` before making major configuration changes
5. **Monitor Logs**: Check application logs after changes to verify successful updates

---

## Related Documentation

- [TOML Configuration](toml.md) - File-based configuration reference
- [Web Interface](../user-guide/web-interface.md) - Using the web UI
- [Fairness Algorithm](../architecture/fairness-algorithm.md) - How availability affects assignments
- [Database Schema](../architecture/database.md) - Configuration table structure
