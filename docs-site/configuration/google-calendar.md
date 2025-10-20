# Google Calendar Setup

To use Night Routine Scheduler, you need to set up OAuth 2.0 credentials in Google Cloud Console and configure the Google Calendar API.

## Prerequisites

- A Google account
- Access to Google Cloud Console
- A Google Calendar (can be your primary calendar or a dedicated one)

## Step-by-Step Setup

### 1. Create a Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Click on the project dropdown at the top of the page
3. Click "New Project"
4. Enter a project name (e.g., "Night Routine Scheduler")
5. Click "Create"
6. Wait for the project to be created, then select it from the project dropdown

### 2. Enable the Google Calendar API

1. In the Google Cloud Console, go to **APIs & Services** → **Library**
2. Search for "Google Calendar API"
3. Click on "Google Calendar API" in the results
4. Click the "Enable" button
5. Wait for the API to be enabled

### 3. Configure OAuth Consent Screen

1. Go to **APIs & Services** → **OAuth consent screen**
2. Select **External** user type (or **Internal** if you have a Google Workspace account)
3. Click "Create"

4. **Fill in the App Information:**
    - **App name:** `Night Routine Scheduler` (or your preferred name)
    - **User support email:** Your email address
    - **App logo:** (Optional) Upload a logo
    - **Application home page:** (Optional) Your website
    - **Authorized domains:** (Leave empty for local development)
    - **Developer contact information:** Your email address

5. Click "Save and Continue"

6. **Add Scopes:**
    - Click "Add or Remove Scopes"
    - Filter for "Google Calendar API"
    - Select the following scopes:
        - `https://www.googleapis.com/auth/calendar` (See, edit, share, and permanently delete all calendars)
        - `https://www.googleapis.com/auth/calendar.events` (View and edit events on all your calendars)
    - Click "Update"
    - Click "Save and Continue"

7. **Add Test Users** (for External apps in testing):
    - Click "Add Users"
    - Add your email address and any other users who will test the app
    - Click "Add"
    - Click "Save and Continue"

8. Review the summary and click "Back to Dashboard"

!!! note "Publishing Status"
    For personal use, you can leave the app in "Testing" status. Only test users will be able to authenticate. To use with more than 100 users, you'll need to submit the app for verification.

### 4. Create OAuth 2.0 Credentials

1. Go to **APIs & Services** → **Credentials**
2. Click "Create Credentials" → "OAuth client ID"
3. Select **Application type:** "Web application"
4. **Name:** `Night Routine Scheduler` (or your preferred name)

5. **Add Authorized Redirect URIs:**

    For local development:
    ```
    http://localhost:8080/oauth/callback
    ```

    For production (replace with your actual domain):
    ```
    https://night-routine.example.com/oauth/callback
    ```

    !!! important "Callback URL Format"
        The callback URL must match: `<app_url>/oauth/callback`
        
        Where `app_url` is the value in your `routine.toml` configuration.

6. Click "Create"

7. **Save Your Credentials:**
    - A dialog will appear with your **Client ID** and **Client Secret**
    - **Copy both values** - you'll need them for configuration
    - You can also download the JSON file for backup

!!! danger "Keep Credentials Secret"
    Never commit your Client Secret to version control or share it publicly. Treat it like a password.

### 5. Configure Environment Variables

Set the OAuth credentials as environment variables:

```bash
export GOOGLE_OAUTH_CLIENT_ID="123456789-abcdefghijklmnopqrstuvwxyz.apps.googleusercontent.com"
export GOOGLE_OAUTH_CLIENT_SECRET="GOCSPX-your-secret-here"
```

Or add them to your `.env` file:

```bash
GOOGLE_OAUTH_CLIENT_ID=123456789-abcdefghijklmnopqrstuvwxyz.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET=GOCSPX-your-secret-here
```

For Docker:

```yaml
environment:
  - GOOGLE_OAUTH_CLIENT_ID=123456789-abcdefghijklmnopqrstuvwxyz.apps.googleusercontent.com
  - GOOGLE_OAUTH_CLIENT_SECRET=GOCSPX-your-secret-here
```

## OAuth Scopes Explained

The application requires the following Google Calendar API scopes:

| Scope | Purpose |
|-------|---------|
| `calendar` | Full access to calendars - needed to create, update, and delete night routine events |
| `calendar.events` | Access to calendar events - needed to read and modify event details |

!!! info "Why Full Access?"
    The application needs full calendar access to:
    
    - Create all-day events for night routine assignments
    - Update events when the schedule changes
    - Delete events that are no longer needed
    - Set up and manage webhook notifications
    - Read event details to detect manual overrides

## Testing the Setup

### 1. Start the Application

```bash
./night-routine
```

Or with Docker:

```bash
docker-compose up -d
```

### 2. Access the Web Interface

Open your browser to the configured `app_url` (default: `http://localhost:8080`)

### 3. Connect to Google Calendar

1. Click the "Connect Google Calendar" button
2. You'll be redirected to Google's OAuth consent screen
3. Select your Google account
4. Review the permissions requested
5. Click "Continue" (or "Allow")
6. You'll be redirected back to the application

!!! warning "Unverified App Warning"
    If your app is in "Testing" status, you'll see a warning that says "Google hasn't verified this app." Click "Advanced" → "Go to Night Routine Scheduler (unsafe)" to proceed. This is normal for apps in testing mode.

### 4. Select a Calendar

After authentication, you'll be redirected to the calendar selection page where you can choose which Google Calendar to use for night routine events.

## Production Deployment

### Domain Configuration

For production deployments with a custom domain:

1. **Add your domain to Authorized Domains** (in OAuth consent screen)
2. **Add the production callback URL** to Authorized Redirect URIs:
    ```
    https://your-domain.com/oauth/callback
    ```

3. **Update your configuration** to match:
    ```toml
    [app]
    app_url = "https://your-domain.com"
    public_url = "https://your-domain.com"
    ```

### HTTPS Requirements

!!! danger "HTTPS Required for Production"
    Google OAuth requires HTTPS for production deployments. Use a reverse proxy (nginx, Caddy, Traefik) with Let's Encrypt for SSL certificates.

### Multiple Environments

If you have multiple environments (development, staging, production), create separate OAuth credentials for each:

1. Create separate Google Cloud projects for each environment, or
2. Create multiple OAuth clients in the same project with different redirect URIs

**Example:**

| Environment | Redirect URI |
|-------------|-------------|
| Development | `http://localhost:8080/oauth/callback` |
| Staging | `https://staging.example.com/oauth/callback` |
| Production | `https://night-routine.example.com/oauth/callback` |

## Troubleshooting

### "Error 400: redirect_uri_mismatch"

**Cause:** The redirect URI in your request doesn't match any of the authorized redirect URIs in Google Cloud Console.

**Solution:**

1. Check your `app_url` in `routine.toml`
2. Verify the callback URL is constructed as `<app_url>/oauth/callback`
3. Ensure this exact URL is added to Authorized Redirect URIs in Google Cloud Console
4. Check for typos, trailing slashes, and protocol (http vs https)

### "Access Blocked: This app's request is invalid"

**Cause:** OAuth consent screen is not properly configured.

**Solution:**

1. Complete all required fields in the OAuth consent screen
2. Ensure scopes are added correctly
3. If using "External" type in testing, add your email as a test user

### "This app is blocked"

**Cause:** The app is not published and you're not a test user.

**Solution:**

1. Go to OAuth consent screen in Google Cloud Console
2. Add your email address to test users
3. Or publish the app (requires verification for 100+ users)

### "Calendar API has not been used in project"

**Cause:** The Google Calendar API is not enabled for your project.

**Solution:**

1. Go to **APIs & Services** → **Library**
2. Search for "Google Calendar API"
3. Click "Enable"

### Credentials Not Working

**Checklist:**

- [ ] Calendar API is enabled
- [ ] OAuth consent screen is configured
- [ ] Credentials are created as "Web application" type
- [ ] Redirect URI matches exactly (including protocol and path)
- [ ] Environment variables are set correctly
- [ ] No extra whitespace in environment variables
- [ ] Client Secret hasn't been regenerated (old secret is invalid)

## Security Best Practices

### Protect Your Credentials

- Never commit credentials to version control
- Use environment variables or secret management systems
- Rotate Client Secret if it's ever exposed
- Use different credentials for development and production

### Limit Scope Access

- Only request the scopes you need
- Review granted permissions regularly
- Revoke unused OAuth tokens

### Monitor Usage

- Check Google Cloud Console for unusual API usage
- Set up billing alerts
- Enable audit logs

### Restrict Access

- Use "Internal" user type if you have Google Workspace
- Limit test users to trusted accounts
- Regularly review who has access to the app

## Managing OAuth Tokens

OAuth tokens are stored in the SQLite database and automatically refreshed when needed.

### Viewing Token Status

The application shows authentication status on the home page.

### Revoking Access

To revoke access:

1. Go to your [Google Account Permissions](https://myaccount.google.com/permissions)
2. Find "Night Routine Scheduler"
3. Click "Remove Access"
4. Restart the application and re-authenticate

### Token Refresh

Access tokens expire after 1 hour but are automatically refreshed using the stored refresh token. You don't need to do anything.

## API Quotas and Limits

Google Calendar API has the following default limits:

- **Queries per day:** 1,000,000
- **Queries per 100 seconds per user:** 10,000

Night Routine Scheduler is well within these limits for normal use:

- Initial setup: ~5-10 API calls
- Daily sync: ~2-5 API calls
- Webhook notifications: 1 API call per calendar change

!!! tip "Monitoring Usage"
    You can monitor your API usage in Google Cloud Console under **APIs & Services** → **Dashboard**

## Next Steps

- [Configure environment variables](environment.md)
- [Set up TOML configuration](toml.md)
- [Complete first-time setup](../user-guide/setup.md)
