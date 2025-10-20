# Features

Night Routine Scheduler offers a comprehensive set of features designed to make managing parental night routine duties simple, fair, and automated.

## Core Scheduling

### Advanced Fairness Algorithm

The application uses a sophisticated multi-criteria fairness algorithm to ensure equitable distribution of night routine duties:

- **Total Assignment Count Balancing** - Tracks lifetime assignments to maintain overall equality
- **Recent Assignment Count Consideration** - Prioritizes parents who haven't had recent assignments
- **Consecutive Assignment Limits** - Prevents one parent from being assigned too many nights in a row
- **Alternating Pattern Maintenance** - Strives to maintain a regular alternating schedule when possible
- **Parent Availability Constraints** - Respects configured unavailable days for each parent
- **Decision Reason Tracking** - Provides transparency into why each assignment was made

### Flexible Scheduling Options

- **Configurable Update Frequencies** - Choose from daily, weekly, or monthly updates
- **Look-Ahead Scheduling** - Schedule assignments for a configurable number of days in advance (default: 30 days)
- **Manual Sync on Startup** - Optionally synchronize schedules when the application starts (enabled by default)
- **On-Demand Synchronization** - Trigger manual schedule updates via the web interface

## Google Calendar Integration

### OAuth2 Authentication

- **Secure Token Management** - OAuth2 tokens are encrypted and stored securely in the SQLite database
- **Automatic Token Refresh** - Tokens are automatically refreshed when they expire
- **One-Time Setup** - Authentication persists between application restarts

### Automatic Event Creation

All calendar events are created with:

- **Consistent Naming** - Events follow the format: `[ParentName] 🌃👶Routine`
- **All-Day Events** - Events span the entire day for simplicity
- **Decision Reasons** - Event descriptions include the reason for the assignment
- **No Reminders** - Events are created without reminders to avoid notification fatigue
- **Intelligent Updates** - Existing events are updated rather than deleted and recreated

### Webhook Support

- **Real-Time Notifications** - Receive instant updates when calendar events change
- **Automatic Channel Management** - Notification channels are automatically created and renewed before expiration
- **Manual Override Detection** - Detects when event titles are manually edited in Google Calendar

### Manual Override Support

Users can override assignments by editing event titles directly in Google Calendar:

- **Configurable Threshold** - Only accepts changes for events within a specified timeframe (default: 5 days in the past)
- **Automatic Recalculation** - Future assignments are recalculated to maintain fairness after manual overrides
- **Transparent Tracking** - Override decisions are tracked and visible in the interface

## Web Interface

### Home Page

- **Authentication Status Display** - Shows whether you're connected to Google Calendar
- **Visual Monthly Assignment Calendar**:
    - Color-coded assignments (blue for Parent A, orange for Parent B)
    - Yellow highlight for today's date
    - Gray background for padding days from previous/next months
    - Assignment decision reasons (hover tooltip on desktop, always visible on mobile)
- **Quick Actions**:
    - Connect Google Calendar button (when not authenticated)
    - Change Calendar button (switch to a different calendar)
    - Sync Now button (manually trigger schedule update)
    - View Statistics link (access historical data)

### Calendar Selection Page

- **List All Available Calendars** - Shows all Google Calendars from your account
- **Simple Selection** - Choose which calendar to use for night routine events
- **Automatic Webhook Setup** - Notification channels are configured automatically after selection

### Statistics Page

- **Monthly Assignment Counts** - View assignment counts per parent for each month
- **12-Month History** - Displays data for the last 12 months
- **Fair Distribution Verification** - Helps verify equitable distribution over time
- **Clean Presentation** - Only shows months with actual assignments

### Responsive Design

- **Desktop Optimized** - Clean layout with hover tooltips for additional information
- **Mobile Friendly** - Touch-optimized interface with tap-to-toggle decision reasons
- **Modern UI** - Clean, intuitive design that works across all devices

## Data Management

### SQLite Database

The application uses SQLite for persistent storage with the following features:

- **Assignment History** - Complete record of all assignments with decision reasons
- **OAuth2 Tokens** - Securely stored tokens with automatic refresh capability
- **Calendar Configuration** - Selected Google Calendar settings
- **Notification Channels** - Management of webhook notification channels
- **WAL Mode** - Write-Ahead Logging for better concurrency
- **Automatic Migrations** - Database schema is updated automatically on startup
- **Foreign Key Constraints** - Data integrity is enforced at the database level
- **Incremental Auto-Vacuum** - Automatic database maintenance

### Configurable Availability

- **Days of Week Configuration** - Set which days each parent is unavailable
- **Flexible Constraints** - Define availability patterns that match your family's schedule
- **Automatic Adherence** - The fairness algorithm respects configured availability

### Assignment Decision Tracking

Every assignment includes a tracked decision reason:

- **Unavailability** - One parent was not available on that day
- **Total Count** - Parent had fewer total assignments overall
- **Recent Count** - Parent had fewer recent assignments
- **Consecutive Limit** - Assignment made to avoid too many consecutive duties
- **Alternating** - Maintains fair alternating pattern
- **Manual Override** - User manually changed the assignment via Google Calendar

## Operations & Deployment

### Structured Logging

Powered by [zerolog](https://github.com/rs/zerolog):

- **Configurable Log Levels** - Choose from trace, debug, info, warn, error, fatal, or panic
- **Pretty Console Output** - Human-readable format for development
- **JSON Output** - Machine-parseable format for production
- **Environment-Based Switching** - Automatically use JSON logging in production

### Docker Containerization

- **Pre-Built Images** - Available in GitHub Container Registry
- **Multi-Architecture Support** - Native support for amd64 and arm64
- **Signed Images** - Images are signed using Sigstore Cosign for verification
- **SBOM Attestations** - Software Bill of Materials included for security auditing
- **Tagged Releases** - Available as `latest` or specific version tags (e.g., `v1.0.0`)

### High Performance

- **WAL Mode SQLite** - Better concurrency for database operations
- **Graceful Shutdown** - Properly handles termination signals
- **Efficient Updates** - Only updates changed calendar events
- **Minimal Resource Usage** - Lightweight Go binary with small footprint

## Security Features

- **Environment Variable Credentials** - OAuth2 credentials stored securely outside the codebase
- **Encrypted Token Storage** - Database storage for sensitive authentication tokens
- **HTTPS Recommended** - Use with reverse proxy for production deployments
- **Regular Dependency Updates** - Automated dependency updates via Renovate
- **Signed Container Images** - Cosign signatures for image verification
- **SBOM Generation** - Complete software bill of materials for security auditing
