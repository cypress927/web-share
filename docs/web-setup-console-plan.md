# Tray-First Usage Plan

## Goal

Adjust the product to a tray-first usage model:

1. User double-clicks `web-share.exe`
2. Program starts in the background
3. Tray icon appears
4. A startup-complete notification is shown
5. User opens the management page from the tray menu when needed
6. Web pages are used as a settings center, not as a mandatory first-run wizard

This replaces the previous direction where double-clicking the program immediately opened `/setup` or `/manage`.

## Target User Experience

### Main Entry

Double-clicking `web-share.exe` should do this by default:

- start the local manager
- start the tray
- show a startup-complete notification
- keep the program in the background
- not automatically open the browser

If the program is already running:

- do not open a second tray icon
- optionally show an “already running” notification, or stay silent

### Tray

The tray remains the main visible control entry.

Required tray behavior:

- tray icon is visible after launch
- tray provides `Open Manager`
- tray provides `Share Clipboard`
- tray provides `Exit Program`

### Web

The Web UI becomes a settings and management center:

- share management stays in `/manage`
- system settings stay in `/manage/settings/system`
- `/setup` is no longer the default page opened on launch

### Default Language

The software default language should follow the system language on first run.

Behavior:

- if there is no saved default language yet
- read Windows/system language
- normalize to `en-US` or `zh-CN`
- persist that as the software default language

After first initialization:

- the saved value becomes the program default
- user can change it from the Web settings page

### Right-Click Menu

The context menu should be installed automatically by default when the user launches the program.

Behavior:

- on startup, check whether context menu is installed
- if missing, install it automatically
- if installation fails, keep the program running and show a notification

### Settings and Self-Uninstall

The Web settings page should allow the user to remove integration items:

- uninstall context menu
- disable auto start
- stop tray
- stop program

The intended final uninstall path is:

1. user removes integration from Web settings
2. user deletes `web-share.exe`
3. user optionally deletes local data directory

No separate uninstall executable is required for the normal user flow.

## Why This Direction

This model is simpler for normal users:

- one executable
- one obvious action: double-click to run
- one persistent control point: tray icon
- one optional settings surface: local Web page

It avoids forcing users into a browser flow before the tray exists.

It also better matches the character of this product:

- background utility
- file sharing helper
- Windows shell integration tool

## Current Status

The main tray-first flow is now aligned in code.

### Implemented

- manager can run in background
- tray exists
- startup notifications exist
- tray can open manager
- Web system settings page exists
- context menu install/uninstall exists
- auto-start enable/disable exists
- double-click no longer opens browser automatically
- default language is initialized from system language on first run
- context menu is auto-installed on normal launch if missing

### Remaining Work

- `/setup` still exists as an optional page, though it is no longer the default launch path
- the current system page can now stop the whole program, but uninstall is still “remove integration first, then delete files manually”

## Required Changes

### 1. Change Default Launch Behavior

Update no-arg launch behavior in [app.go](C:/Users/zhjun/Desktop/code/web-share/internal/app/app.go):

Current behavior:

- ensure manager
- ensure tray
- open `/setup` or `/manage`

Target behavior:

- ensure manager
- ensure tray
- auto-install context menu if missing
- show startup notification
- do not open browser

Optional behavior:

- if user explicitly launches with a CLI flag later, browser opening can still be supported

### 2. Initialize Default Language from System

Add first-run default language initialization.

Suggested logic:

- if `default_lang` is absent or empty
- detect Windows/system preferred UI language
- normalize to `zh-CN` or `en-US`
- save into settings store

This should happen before tray labels and notifications are created.

### 3. Auto-Install Context Menu on Launch

Add startup integration check:

- if context menu is not installed
- install it automatically with the current default language

Error handling:

- failure should not stop startup
- surface failure through notification and logs

### Password Prompt

Current state:

- folder password sharing still uses `wscript.exe` plus generated `VBS InputBox`

Target state:

- replace the VBS password prompt with a Go-native Windows password dialog

Reason:

- reduce script dependencies
- make the right-click share experience feel native
- avoid visible script-related pop-up behavior

Recommended implementation:

- use a lightweight native Win32 dialog
- do not introduce a large GUI framework just for password input

### 4. Reposition `/setup`

`/setup` should no longer be the mandatory first screen.

Options:

- keep it as a lightweight onboarding/status page
- or merge it gradually into `/manage/settings/system`

Recommended immediate choice:

- keep `/setup` temporarily
- remove it from default launch path
- let tray open `/manage`

### 5. Expand System Settings Actions

The system settings page should remain the place to manage integration:

- default language
- context menu install/uninstall
- auto-start enable/disable
- tray start/stop
- setup state markers if still needed

Potential next addition:

- `Stop Program`

This can call the existing local shutdown endpoint.

## Product Model After Change

### User Sees

- one executable
- one tray icon
- one startup notification
- one optional browser-based settings page

### User Does Not Need

- PowerShell scripts
- manual install command
- manual first-run setup page
- scheduled task management

### User Can Still Do

- right-click share files and folders
- share clipboard from tray
- open manager from tray
- change language and integration settings in Web

## Technical Direction

### Tray-First Startup

The no-argument launch path becomes the main product path.

This should be treated as the primary runtime mode.

### System Language Detection

Use Windows APIs or an equivalent reliable method to determine the system UI language on first run.

If full detection is unavailable:

- fall back to existing browser/request language logic only as a secondary heuristic

The persisted program default should remain stable after first initialization.

### Context Menu Language

Context menu language should continue to be derived from the saved default language.

If the user changes language from Web settings:

- persist new default language
- reinstall context menu with new labels
- restart tray so menu text updates immediately

### Auto Start

The current native `Run` registry implementation is the desired long-term direction.

No need to return to Scheduled Task for the main path.

Legacy PowerShell scripts may remain in the repo for compatibility, but should not define product behavior.

## Migration Plan

### Phase 1

- stop auto-opening browser on no-arg launch
- ensure tray starts
- show startup-complete notification

### Phase 2

- initialize default language from system language on first run
- persist it before tray/menu creation

### Phase 3

- auto-install context menu on startup when missing
- show error notification on failure

### Phase 4

- simplify `/setup` role
- keep `/manage/settings/system` as primary settings center

### Phase 5

- add stop-program action in Web settings if desired
- update all user-facing docs to tray-first language

### Phase 6

- replace VBS password prompt with Go-native Win32 password dialog

## Acceptance Criteria

This plan is complete when:

- double-clicking `web-share.exe` starts the background service and tray
- a startup-complete notification appears
- tray can open the manager page
- browser does not auto-open on normal launch
- default language follows system language on first run
- context menu is installed automatically on first launch if missing
- Web settings can uninstall context menu
- Web settings can disable auto start
- user can remove the executable and local data manually after removing integration
