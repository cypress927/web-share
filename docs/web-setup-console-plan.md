# Web Setup Console Plan

## Goal

Move first-run setup, Windows integration, and runtime configuration into the local Web UI.

Target user flow:

1. User double-clicks `web-share.exe`
2. Program starts the local manager
3. Browser opens `http://127.0.0.1:21910/setup`
4. User completes language selection and Windows integration from the page

After setup is complete, the default entry page becomes `http://127.0.0.1:21910/manage`.

## Why

Current initialization still feels fragmented:

- built-in CLI commands exist
- PowerShell scripts still exist
- right-click flow starts the manager implicitly
- tray is a separate runtime surface

For end users, the cleanest model is:

- one executable
- one browser-based setup page
- one browser-based management page

## Product Direction

Use Web as the primary control surface for:

- first-run initialization
- language selection
- context-menu installation
- auto-start enable or disable
- tray start or stop
- repair actions
- uninstall actions

CLI commands remain available, but become secondary and mainly useful for power users or development.

## Scope

### In Scope

- first-run setup page
- system settings page inside manager
- status checks for Windows integration
- Web-triggered install, repair, and uninstall actions
- browser auto-open behavior on first launch

### Out of Scope

- replacing tray with Web
- replacing right-click sharing flow
- remote administration from other devices
- elevation and machine-wide install

This plan is for per-user installation under `HKCU`.

## User Flows

### First Run

1. User launches `web-share.exe`
2. App ensures local manager is running
3. App checks `setup_completed`
4. If `false`, app opens `/setup`
5. User selects:
   - default language
   - install context menu
   - enable auto start
   - start tray now
6. User clicks `Finish Setup`
7. App persists settings, applies integration, and redirects to `/manage`

### Normal Launch

1. User launches `web-share.exe`
2. App ensures manager is running
3. App checks `setup_completed`
4. If `true`, app opens `/manage`

### Repair

User opens `Manage -> System Settings` and clicks:

- reinstall context menu
- re-enable auto start
- restart tray
- repair all integration

### Uninstall Integration

User opens `Manage -> System Settings` and clicks:

- remove context menu
- disable auto start
- stop tray
- uninstall integration

Optional destructive action:

- remove local data

## Architecture

### Entry Model

Add a default no-argument entry behavior:

- if manager is not running, start manager
- query local setup status
- open `/setup` or `/manage`

Suggested command behavior:

- `web-share.exe`
  - user-facing default launcher
- `web-share.exe run-manager`
  - background manager only
- `web-share.exe tray`
  - tray only
- `web-share.exe enqueue ...`
  - share target path

Keep current CLI commands, but treat them as implementation tools:

- `install`
- `start`
- `repair`
- `uninstall`

## Data Model

Extend the settings store with:

- `default_lang`
- `setup_completed`
- `auto_open_browser`
- `autostart_mode`

Recommended values:

- `setup_completed`: `true` or `false`
- `auto_open_browser`: `true` or `false`
- `autostart_mode`: `off` or `run_key`

Avoid overdesign here. These settings are enough for the first Web-based setup version.

## Backend Changes

### 1. Settings Store

Extend [settings_store.go](C:/Users/zhjun/Desktop/code/web-share/internal/manager/settings_store.go):

- `GetSetupCompleted()`
- `SetSetupCompleted(bool)`
- `GetAutoOpenBrowser()`
- `SetAutoOpenBrowser(bool)`
- `GetAutostartMode()`
- `SetAutostartMode(string)`

SQLite-backed implementation should continue to use the existing key-value table.

### 2. Manager Routes

Add local-only routes:

- `GET /setup`
- `GET /manage/settings/system`
- `GET /api/setup/status`
- `POST /api/setup/apply`
- `POST /api/setup/context-menu/install`
- `POST /api/setup/context-menu/uninstall`
- `POST /api/setup/autostart/enable`
- `POST /api/setup/autostart/disable`
- `POST /api/setup/tray/start`
- `POST /api/setup/tray/stop`
- `POST /api/setup/repair`
- `POST /api/setup/uninstall`

Reuse the existing local-request guard pattern already used by:

- `/api/shutdown`
- `/api/shares`

### 3. Integration Service

Create a small service layer, for example:

- `internal/integration`

Responsibilities:

- install or uninstall context menu
- enable or disable auto start
- start or stop tray
- gather integration status
- run repair actions

This keeps manager handlers thin and avoids pushing Windows-specific code into HTML handlers.

## Windows Integration Strategy

### Context Menu

Current implementation still shells out to `reg.exe`.

Recommended next step:

- replace `reg.exe` calls with Go registry API
- use `golang.org/x/sys/windows/registry`

Benefits:

- no command window flicker
- better error handling
- simpler Web-triggered operations

### Auto Start

Recommended implementation:

- use `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

Do not keep scheduled tasks as the primary approach for user setup.

Reason:

- simpler
- per-user
- no `schtasks.exe`
- fewer visible system side effects
- easier to manage from Go and Web

Scheduled tasks can remain as an advanced fallback only if needed later.

### Password Prompt

Current folder-password right-click flow still relies on `wscript.exe` and generated VBS.

Recommended later change:

- replace VBS prompt with a tiny Go native dialog

This is not required for the first Web-setup milestone, but it should stay on the cleanup list.

## Frontend Changes

### Setup Page

New page: `/setup`

Sections:

1. Welcome
   - what Web Share does
   - manager address
2. Language
   - English / 中文
3. Windows Integration
   - install context menu
   - enable auto start
   - start tray now
4. Status
   - manager running
   - tray running
   - context menu installed
   - auto start enabled
5. Actions
   - finish setup
   - repair integration
   - skip for now

### System Settings Page

New page under manager:

- `/manage/settings/system`

Sections:

- default language
- context menu status and reinstall button
- auto start status and toggle
- tray status and restart button
- uninstall integration
- uninstall integration plus local data

### UI Behavior

Keep actions optimistic but explicit:

- disable buttons while request is running
- show inline success or error state
- refresh status after each action

Do not open new windows for each action.

## i18n

Add translation keys for:

- setup page title and description
- integration status labels
- install and uninstall button labels
- repair and finish-setup messages
- system settings section labels

The setup page must honor:

- query-string language override
- current session language
- default system language

This should follow the same rules already used by the existing pages.

## Startup Behavior

### Desired Behavior

When user runs `web-share.exe` directly:

1. ensure manager is running
2. determine setup state
3. open browser to `/setup` or `/manage`
4. optionally ensure tray is running if setup already completed

### Suggested Implementation

Add a default launcher path in [app.go](C:/Users/zhjun/Desktop/code/web-share/internal/app/app.go):

- if no arguments:
  - start manager if needed
  - query `setup_completed`
  - open browser to the correct page

This gives a clean double-click experience.

## Status Model

`GET /api/setup/status` should return:

- `defaultLanguage`
- `setupCompleted`
- `managerRunning`
- `trayRunning`
- `contextMenuInstalled`
- `autostartEnabled`
- `autostartMode`
- `dbPath`
- `manageURL`
- `setupURL`

This becomes the single source of truth for the setup UI.

## Security Model

These endpoints should remain local-only.

Rules:

- reject non-local requests
- no CORS exposure
- only allow actions from `127.0.0.1` or loopback

This is consistent with the current manager control surface.

## Migration Plan

### Phase 1

- add `setup_completed` setting
- add `/setup`
- add `/api/setup/status`
- add browser-launch default entry

### Phase 2

- add Web actions for:
  - context menu
  - auto start
  - tray control
- add `/manage/settings/system`

### Phase 3

- replace `reg.exe` with registry API
- replace `schtasks.exe` usage with Run-key auto start

### Phase 4

- optionally retire PowerShell scripts from user-facing docs
- keep scripts only as developer utilities

## Risks

### 1. Mixed Install Paths

If CLI, Web, and legacy scripts all remain active, behavior can drift.

Mitigation:

- define Web as the primary user path
- define CLI as advanced/manual path
- define scripts as legacy/dev path

### 2. Tray State Drift

The Web page may say tray is stopped while tray is starting or exiting.

Mitigation:

- use current mutex-based detection
- refresh status after each action

### 3. Right-Click Language Drift

Changing language from Web must still reinstall context menu with the new labels.

Mitigation:

- continue to route language changes through the same apply-system-language logic

## Recommended First Implementation Slice

Start with the smallest useful end-to-end slice:

1. make no-arg launch open the browser
2. add `setup_completed`
3. add `/setup`
4. add status API
5. allow language selection and finish-setup

Do not start with full uninstall or full repair UI.

Once the setup page exists and becomes the main entry, the rest can be layered onto it cleanly.

## Acceptance Criteria

This feature is complete when:

- double-clicking `web-share.exe` opens a browser page
- first run opens `/setup`
- setup page can set default language
- setup page can install context menu
- setup page can enable auto start
- setup page can start tray
- setup completion is persisted
- later launches open `/manage`
- system settings page can reconfigure integration

