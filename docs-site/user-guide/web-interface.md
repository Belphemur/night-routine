# Web Interface Guide

The Night Routine Scheduler web interface provides an intuitive way to manage your night routine schedule, view statistics, and control synchronization.

## Home Page

The home page (`/`) is your main dashboard for managing night routine assignments.

### Authentication Status

At the top of the page, you'll see your connection status:

=== "Connected"
    ```
    ✅ Connected to Google Calendar
    📅 Using calendar: Night Routine
    ```

=== "Not Connected"
    ```
    ❌ Not connected to Google Calendar
    [Connect Google Calendar Button]
    ```

### Visual Monthly Calendar

The centerpiece of the home page is a visual calendar showing the current month's assignments.

#### Color Coding

- **Blue background** - Parent A is assigned
- **Orange background** - Parent B is assigned  
- **Yellow border** - Today's date
- **Gray background** - Days from previous/next month (padding)

#### Assignment Details

Click on any assignment to view detailed information about how the fairness algorithm made its decision:

![Assignment Details Modal - Desktop](../screenshots/assignment-details-modal-desktop.png)
_Click any assignment to see the calculation details in a modal dialog_

The assignment details modal shows:

- **Calculation Date** - When the fairness algorithm evaluated this assignment
- **Parent Statistics** - Both parents' total assignments and last 30-day counts at decision time
- **Decision Explanation** - How the algorithm compared these statistics

=== "Desktop"
    **Click** on any assignment cell to open the details modal. The modal displays:
    
    - Calculation date
    - Parent A's total count and last 30 days
    - Parent B's total count and last 30 days
    - Explanation of the fairness algorithm's decision process

=== "Mobile"
    **Tap** on any assignment cell to open the details modal. The modal is fully responsive and provides the same information on mobile devices.

![Assignment Details Modal - Mobile](../screenshots/assignment-details-modal-mobile.png)
_Mobile view of the assignment details modal_

!!! note "Override Assignments"
    Clicking on an assignment marked as "Override" (with 🔒 icon) will show the override removal modal instead of the details modal, allowing you to remove the manual override if needed.

#### Decision Reasons

Each assignment includes a reason explaining why that parent was chosen:

| Reason | Meaning |
|--------|---------|
| **Unavailability** | The other parent was unavailable on this day |
| **Total Count** | This parent has fewer total assignments |
| **Recent Count** | This parent has had fewer recent assignments |
| **Consecutive Limit** | Prevents too many consecutive assignments |
| **Alternating** | Maintains an alternating pattern |
| **Override** | Manually changed via Google Calendar |

### Quick Actions

The home page provides several action buttons:

#### Connect Google Calendar

**When:** Not yet authenticated

**Action:** Initiates the OAuth flow to connect your Google account

**Result:** Redirects to Google login, then to calendar selection after authorization

#### Change Calendar

**When:** Already authenticated

**Action:** Allows you to switch to a different Google Calendar

**Result:** Opens calendar selection page

#### Sync Now

**When:** Authenticated and calendar selected

**Action:** Manually triggers a schedule synchronization

**Result:**

- Calculates new assignments based on current fairness state
- Updates Google Calendar events
- Refreshes the web interface

**Use cases:**

- After changing configuration
- To fill in newly available dates
- After manually modifying events in Google Calendar

#### View Statistics

**When:** Authenticated

**Action:** Opens the statistics page

**Result:** Shows monthly assignment counts for the last 12 months

#### Settings

**When:** Authenticated

**Action:** Opens the settings page

**Result:** Allows you to update parent names, availability, and schedule settings without restarting the application

## Settings Page

The settings page (`/settings`) provides a web interface for managing your configuration.

### Configurable Settings

The following settings can be updated via the web interface:

#### Parent Names

- **Parent A Name** - First parent's display name
- **Parent B Name** - Second parent's display name

Changes to parent names affect:

- Future calendar event titles
- Assignment display in the web interface
- Statistics calculations

!!! warning "Name Restrictions"
    - Both names must be provided
    - Names must be different from each other
    - Changes apply immediately without restart

#### Availability

Select days when each parent is unavailable for night routine duties.

- **Parent A Unavailable Days** - Days when Parent A can't do the routine
- **Parent B Unavailable Days** - Days when Parent B can't do the routine

**Valid days:** Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday

!!! tip "Multiple Days"
    Select multiple days by checking all applicable checkboxes. Leave all unchecked if parent is always available.

#### Schedule Settings

- **Update Frequency** - How often to automatically update (daily, weekly, monthly)
- **Look Ahead Days** - Number of days in advance to schedule (1-365)
- **Past Event Threshold Days** - Days in the past to accept manual changes (0-30)

!!! info "Immediate Effect"
    All settings changes take effect immediately. No application restart required. Consider clicking "Sync Now" on the home page after making changes.

### Saving Settings

1. Make your desired changes in the form
2. Click the **Save Settings** button
3. You'll see a success message confirming the save
4. Settings are stored in the database and persist across restarts

### Settings Validation

The form validates your input before saving:

- **Parent names** must not be empty or identical
- **Update frequency** must be daily, weekly, or monthly
- **Look ahead days** must be at least 1
- **Past event threshold** cannot be negative

If validation fails, you'll see an error message explaining what needs to be corrected

### Calendar Navigation

Use the **Previous** and **Next** buttons to navigate between months:

- **← Previous** - View the previous month
- **Next →** - View the next month

The current month and year are displayed in the center.

## Calendar Selection Page

The calendar selection page (`/calendars`) appears after initial authentication or when you click "Change Calendar".

### Calendar List

All calendars from your Google account are displayed:

- **Primary Calendar** - Your main Google Calendar
- **Secondary Calendars** - Other calendars you own or have write access to
- **Shared Calendars** - Calendars shared with you (if you have write access)

### Selecting a Calendar

1. Click on the calendar you want to use
2. The application will:
    - Save your selection to the database
    - Set up webhook notifications for this calendar
    - Create initial night routine events
    - Redirect you to the home page

### Changing Calendars

To switch to a different calendar:

1. From the home page, click "Change Calendar"
2. Select a new calendar from the list
3. The application will:
    - **Delete** all existing night routine events from the old calendar
    - **Create** new events in the new calendar
    - **Update** webhook subscriptions

!!! warning "Calendar Switch Impact"
    Switching calendars will remove all night routine events from the previous calendar. This action cannot be undone.

## Statistics Page

The statistics page (`/statistics`) provides a historical view of assignment distribution.

### Monthly Breakdown

Displays a table showing:

| Month | Parent A | Parent B | Total |
|-------|----------|----------|-------|
| Jan 2024 | 15 | 16 | 31 |
| Feb 2024 | 14 | 14 | 28 |
| ... | ... | ... | ... |

### Features

- **Last 12 months** - Shows up to 12 months of historical data
- **Only assigned months** - Months without assignments are hidden
- **Fair distribution verification** - Quickly see if assignments are balanced
- **Total counts** - Sum of assignments per month

### Interpreting Statistics

**Balanced distribution:**
```
Month     | Parent A | Parent B | Total
----------|----------|----------|------
Jan 2024  | 15       | 16       | 31
```
✅ Difference of 1 is normal for odd-day months

**Imbalanced distribution:**
```
Month     | Parent A | Parent B | Total
----------|----------|----------|------
Jan 2024  | 20       | 11       | 31
```
⚠️ Large difference may indicate:

- One parent has more unavailable days
- Manual overrides have skewed the distribution
- Configuration issues

## API Endpoints

While you typically interact through the web interface, the application also exposes API endpoints.

### Authentication Endpoints

#### `GET /auth`

Initiates Google OAuth2 flow.

**Usage:** Automatically called when clicking "Connect Google Calendar"

**Response:** Redirects to Google OAuth consent screen

#### `GET /oauth/callback`

OAuth2 callback handler.

**Usage:** Automatic redirect from Google after authentication

**Response:** Redirects to calendar selection page

### Calendar Management

#### `GET /calendars`

Lists available Google Calendars and allows selection.

**Response:** HTML page with calendar list

#### `POST /calendars/select`

Selects a calendar to use for night routine events.

**Parameters:**

- `calendar_id` - The Google Calendar ID

**Response:** Redirects to home page

### Synchronization

#### `POST /sync`

Manually triggers schedule synchronization.

**Usage:** Called when clicking "Sync Now" button

**Response:** Redirects back to home page

### Webhooks

#### `POST /api/webhook/calendar`

Receives Google Calendar change notifications.

**Usage:** Called automatically by Google Calendar when events change

**Authentication:** Validates Google webhook signature

**Response:** 200 OK on success

## Responsive Design

The web interface is fully responsive and optimized for different screen sizes.

### Desktop (1024px+)

- Full calendar grid layout
- Hover tooltips for assignment details
- Side-by-side action buttons
- Wide statistics table

### Tablet (768px - 1023px)

- Adjusted calendar grid
- Tap-to-toggle assignment details
- Stacked action buttons
- Scrollable statistics table

### Mobile (<768px)

- Compact calendar layout
- Touch-optimized controls
- Tap to show/hide decision reasons
- Full-width buttons
- Simplified navigation

## Browser Compatibility

The web interface supports modern browsers:

- ✅ Chrome 90+
- ✅ Firefox 88+
- ✅ Safari 14+
- ✅ Edge 90+

!!! warning "Internet Explorer"
    Internet Explorer is not supported. Please use a modern browser.

## Keyboard Shortcuts

The interface supports basic keyboard navigation:

| Shortcut | Action |
|----------|--------|
| `Tab` | Navigate between interactive elements |
| `Enter` | Activate buttons/links |
| `←` / `→` | Navigate calendar months (when focused) |

## Accessibility

The interface includes accessibility features:

- **Semantic HTML** - Proper heading structure and landmarks
- **ARIA labels** - Screen reader friendly
- **Keyboard navigation** - All functions accessible without mouse
- **Color contrast** - WCAG AA compliant
- **Focus indicators** - Visible focus states

## Troubleshooting

### Calendar Not Updating

**Problem:** Changes don't appear after clicking "Sync Now"

**Solutions:**

1. Check browser console for errors (F12)
2. Hard refresh the page (Ctrl+F5 or Cmd+Shift+R)
3. Clear browser cache
4. Check application logs for errors

### Assignment Details Not Showing (Mobile)

**Problem:** Tapping dates doesn't show decision reasons

**Solutions:**

1. Ensure JavaScript is enabled
2. Try tapping directly on the date number
3. Refresh the page
4. Check browser compatibility

### Webhook Notifications Not Working

**Problem:** Manual changes in Google Calendar don't reflect immediately

**Solutions:**

1. Check application logs for webhook errors
2. Verify `public_url` is accessible from the internet
3. Test by manually clicking "Sync Now"
4. Webhook may need to be renewed (happens automatically)

### Page Layout Issues

**Problem:** Calendar layout is broken or overlapping

**Solutions:**

1. Clear browser cache
2. Disable browser extensions
3. Try a different browser
4. Check browser console for CSS errors

## Best Practices

### Regular Checks

- Review the calendar weekly to ensure proper scheduling
- Check statistics monthly to verify fair distribution
- Sync manually after configuration changes

### Calendar Hygiene

- Don't manually delete night routine events in Google Calendar (change the parent instead)
- Use the web interface for changing calendars
- Let automatic sync handle routine updates

### Performance

- Minimize frequent manual syncs (let automatic updates handle it)
- Use statistics page sparingly (it processes historical data)
- Clear old browser data periodically

## Next Steps

- [Learn about manual overrides](manual-overrides.md)
- [Understand the assignment logic](../architecture/assignment-logic.md)
- [Explore all features](../features.md)
