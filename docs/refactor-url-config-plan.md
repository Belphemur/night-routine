# Refactoring Plan: Application URL Configuration

## Goal

Refactor the application to use two distinct URLs (`AppUrl` for internal use and `PublicUrl` for external use like webhooks) sourced from the TOML configuration file instead of a single `APP_URL` from environment variables.

## Steps

1.  **Create Git Branch:** Create a new branch named `feature/url-config`.
2.  **Modify Config Struct (`internal/config/config.go`):**
    - Rename the `Url` field in `ApplicationConfig` to `AppUrl`.
    - Add a new `PublicUrl` field (`string`) to `ApplicationConfig`.
3.  **Update Config Loading (`internal/config/config.go`):**
    - Remove the logic reading `APP_URL` from environment variables.
    - Add logic to read `app_url` and `public_url` from a new `[app]` section in `configs/routine.toml`.
    - Ensure both `app_url` and `public_url` are present in the TOML file, returning an error if missing.
4.  **Update Code Usage:**
    - Modify the Google OAuth `RedirectURL` construction in `internal/config/config.go` to use `cfg.App.AppUrl`.
    - Modify the webhook address construction in `internal/calendar/notification.go` to use `cfg.App.PublicUrl`.
    - Modify the `calendar.EventSource` URL in `internal/calendar/calendar.go` to use `cfg.App.AppUrl`.
5.  **Update Configuration Files:**
    - Add the `[app]` section with `app_url = "..."` and `public_url = "..."` keys to `configs/routine.toml`.
    - Remove the `APP_URL` variable definition from `.env.example`.
    - Add comments to `.env.example` guiding the user to set the URLs in `configs/routine.toml`.
    - Remind the user to remove `APP_URL` from their local `.env` file.

## Diagram

```mermaid
graph TD
    A[Start Refactoring] --> B(Create Git Branch: feature/url-config);
    B --> C{Modify Config Struct};
        C -- Add --> C1(Field: PublicUrl string);
        C -- Rename --> C2(Field: Url -> AppUrl string);
    C --> D{Update Config Loading};
        D -- Remove --> D1(Read APP_URL from Env);
        D -- Add --> D2(Read app_url, public_url from TOML);
        D -- Add --> D3(Make TOML fields mandatory);
    D --> E{Update Code Usage};
        E -- Use AppUrl --> E1(OAuth Redirect URL);
        E -- Use PublicUrl --> E2(Webhook Address);
        E -- Use AppUrl --> E3(Calendar Event Source URL);
    E --> F{Update Config Files};
        F -- Add Section --> F1(configs/routine.toml: [app] app_url, public_url);
        F -- Remove Var --> F2(.env.example: Remove APP_URL);
        F -- Add Comment --> F3(.env.example: Guide to TOML);
    F --> G[End Refactoring];

    subgraph "internal/config/config.go"
        C1; C2; D1; D2; D3; E1;
    end
    subgraph "internal/calendar/notification.go"
        E2;
    end
     subgraph "internal/calendar/calendar.go"
        E3;
    end
     subgraph "Configuration Files"
        F1; F2; F3;
    end
```
