# Manual Overrides

You can manually override night routine assignments by editing calendar event titles directly in Google Calendar. This allows you to make quick changes without using the web interface.

## How Manual Overrides Work

The application automatically detects when you change a night routine event title in Google Calendar and updates its internal schedule accordingly.

### Override Detection

The system detects overrides through:

1. **Webhook Notifications** - Google Calendar sends real-time notifications when events change
2. **Event Title Parsing** - The application parses the parent name from the title
3. **Database Update** - The internal assignment is updated to match
4. **Fairness Recalculation** - Future assignments are recalculated to maintain balance

## Making a Manual Override

### Step 1: Find the Event

1. Open [Google Calendar](https://calendar.google.com)
2. Navigate to the date you want to change
3. Find the night routine event (e.g., `[Parent1] 🌃👶Routine`)

### Step 2: Edit the Event Title

1. Click on the event to open it
2. Click the edit button (pencil icon)
3. Change the parent name in the title:
    - From: `[Parent1] 🌃👶Routine`
    - To: `[Parent2] 🌃👶Routine`

!!! important "Title Format"
    Keep the format: `[ParentName] 🌃👶Routine`
    
    Only change the name inside the square brackets. The name must match one of your configured parent names exactly.

4. Click "Save" to save the change

### Step 3: Verification

The application will:

1. Receive a webhook notification from Google Calendar (usually within seconds)
2. Parse the new parent name from the event title
3. Check if the date is within the allowed threshold
4. Update the internal database
5. Recalculate future assignments to maintain fairness

You can verify the change by:

- Checking the application logs for override detection messages
- Viewing the web interface (the assignment should update after refresh)
- Looking for the "Override" decision reason in future assignments

## Time Window for Overrides

By default, the application only accepts manual overrides for events within **5 days in the past** (configurable).

### Configuration

Set the threshold in your `routine.toml`:

```toml
[schedule]
past_event_threshold_days = 5  # Accept changes up to 5 days ago
```

### Examples

If today is January 15th and `past_event_threshold_days = 5`:

| Date | Can Override? |
|------|---------------|
| Jan 9 | ❌ No (6 days ago) |
| Jan 10 | ✅ Yes (5 days ago) |
| Jan 11-15 | ✅ Yes (within 5 days) |
| Jan 16+ | ✅ Yes (future dates) |

### Why a Threshold?

The threshold prevents:

- **Accidental historical changes** - Old events being modified unintentionally
- **Fairness calculation errors** - Past assignments affecting current balance
- **Data integrity issues** - Conflicting historical records

!!! tip "Adjusting the Threshold"
    Increase the threshold if you frequently need to correct past assignments:
    ```toml
    past_event_threshold_days = 14  # Allow 2 weeks of changes
    ```

## Impact on Future Assignments

When you manually override an assignment, the fairness algorithm recalculates future assignments to compensate.

### Example Scenario

**Initial schedule:**

| Date | Assignment | Reason |
|------|------------|--------|
| Jan 15 | Parent A | Alternating |
| Jan 16 | Parent B | Alternating |
| Jan 17 | Parent A | Alternating |
| Jan 18 | Parent B | Alternating |

**You manually change Jan 15 from Parent A to Parent B**

**Updated schedule:**

| Date | Assignment | Reason |
|------|------------|--------|
| Jan 15 | Parent B | **Override** |
| Jan 16 | Parent A | **Total Count** (compensating) |
| Jan 17 | Parent B | Alternating |
| Jan 18 | Parent A | Alternating |

Notice how the algorithm adjusted Jan 16 to maintain balance.

## Valid Override Formats

The application recognizes several event title formats:

### Standard Format

```
[ParentName] 🌃👶Routine
```

Example: `[Alice] 🌃👶Routine`

### Without Emojis

```
[ParentName] Routine
```

Example: `[Bob] Routine`

### Case Sensitivity

Parent names are **case-sensitive** and must match your configuration exactly.

✅ **Valid:**
```toml
[parents]
parent_a = "Alice"
parent_b = "Bob"
```
- `[Alice] 🌃👶Routine` ✅
- `[Bob] 🌃👶Routine` ✅

❌ **Invalid:**
- `[alice] 🌃👶Routine` ❌ (wrong case)
- `[ALICE] 🌃👶Routine` ❌ (wrong case)
- `[AliceSmith] 🌃👶Routine` ❌ (doesn't match)

## Common Use Cases

### Swapping Assignments

**Scenario:** Parent A can't do tonight, switch with Parent B

1. Find tonight's event in Google Calendar
2. Change title from `[Parent A]` to `[Parent B]`
3. Find Parent B's next assignment
4. Change it from `[Parent B]` to `[Parent A]`

Result: The nights are swapped, and future assignments remain balanced.

### Emergency Changes

**Scenario:** Parent is sick and unavailable for several days

1. Edit each affected event's title to assign the other parent
2. The algorithm will compensate in future assignments
3. Once the parent is available again, normal scheduling resumes

### Preference-Based Adjustments

**Scenario:** One parent prefers weekend duties

1. Edit weekend events to assign the preferred parent
2. The algorithm balances weekday assignments accordingly

## Monitoring Overrides

### In the Web Interface

Overridden assignments show a special decision reason:

- **Reason:** "Override"
- **Indicator:** Visible in hover tooltips (desktop) or tap-to-view (mobile)

### In Google Calendar

Overridden events still appear as normal night routine events, but the description will show:

```
Decision: Override
```

### In Application Logs

Watch for log messages:

```
INF Processing calendar change notification
INF Override detected date=2024-01-15 old_parent=Parent1 new_parent=Parent2
INF Recalculating future assignments
```

## Troubleshooting

### Override Not Detected

**Problem:** You changed an event title but the web interface doesn't update

**Possible causes:**

1. **Webhook delay** - Wait 30-60 seconds and refresh
2. **Outside threshold** - Event is too far in the past
3. **Invalid format** - Parent name doesn't match configuration
4. **Webhook not configured** - Check application logs

**Solutions:**

1. Click "Sync Now" in the web interface to force synchronization
2. Check the event date vs. `past_event_threshold_days`
3. Verify parent name matches configuration exactly (case-sensitive)
4. Check logs for webhook errors

### Wrong Parent Name

**Problem:** You mistyped the parent name

**Solution:**

1. Edit the event again in Google Calendar
2. Correct the parent name
3. Save the change
4. The override will be reprocessed

### Future Assignments Didn't Recalculate

**Problem:** You made an override but future dates didn't change

**Explanation:**

The algorithm only recalculates when necessary to maintain fairness. If the override doesn't create an imbalance, future assignments may remain unchanged.

**To force recalculation:**

1. Click "Sync Now" in the web interface
2. Check the statistics page to verify balance

### Past Event Beyond Threshold

**Problem:** Cannot override an event from last month

**Solution:**

1. Increase `past_event_threshold_days` in configuration:
    ```toml
    [schedule]
    past_event_threshold_days = 30
    ```
2. Restart the application
3. Make the override in Google Calendar
4. Consider: Do you really need to change historical data?

## Best Practices

### Use Overrides Sparingly

- Overrides should be for exceptions, not routine changes
- Frequent overrides can disrupt the fairness algorithm
- Consider adjusting configuration instead for permanent changes

### Keep Titles Consistent

- Use the same format for all overrides
- Don't modify the emoji or "Routine" text
- Only change the parent name in brackets

### Document Major Changes

- If you make significant overrides, note why in event descriptions
- This helps track unusual patterns in statistics

### Verify After Overriding

- Check the web interface after overrides
- Review statistics to ensure balance is maintained
- Click "Sync Now" if changes don't appear

### Communicate Changes

- If you override an assignment, inform the other parent
- Update any family calendars or notifications
- Ensure both parents are aware of the change

## Alternative: Configuration Changes

For permanent changes, update configuration instead of using overrides:

### Changing Availability

**Instead of:** Manually overriding every Wednesday

**Do this:**

```toml
[availability]
parent_a_unavailable = ["Wednesday"]
```

### Changing Frequency

**Instead of:** Manually spreading assignments further apart

**Do this:**

```toml
[schedule]
update_frequency = "weekly"  # instead of "daily"
```

## Next Steps

- [Learn more about the assignment logic](../architecture/assignment-logic.md)
- [Understand the web interface](web-interface.md)
- [Explore configuration options](../configuration/toml.md)
