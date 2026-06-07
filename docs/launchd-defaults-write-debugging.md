# LaunchDaemon + `defaults write`: bootstrap context gotcha

## The problem

A LaunchDaemon (system-level, runs as root) that calls `sudo -u <user> defaults write` will
**silently fail** on every run. The script exits 0, the log shows "✗ Failed", but no error
message appears because `defaults` writes its error to stderr and the script redirects
`2>/dev/null`.

### Why it fails

`defaults` communicates with `cfprefsd` over a Mach bootstrap port. Each user session has its
own `cfprefsd` instance registered in the **user bootstrap namespace**.

A LaunchDaemon runs in the **system bootstrap namespace**. `sudo -u blake` changes the UID but
does **not** move the process into blake's bootstrap namespace. So `defaults` can't find the
right `cfprefsd` and the write fails.

This is easy to miss because running the same `sudo -u blake defaults write ...` command
manually in a terminal *works* — your terminal is already inside the user bootstrap context, so
`sudo -u` is enough there.

### Symptoms

- Script runs on schedule (launchd fires correctly, log shows start/end timestamps)
- All `defaults write` lines print `✗ Failed`
- The target plist is never actually modified (data keeps accumulating)
- No visible error in the log because stderr is suppressed

## The fix

Wrap `defaults` (and any command that needs user-session context) with `launchctl asuser <uid>`:

```bash
USER_UID=$(id -u "${USER}")
launchctl asuser "${USER_UID}" sudo -u "${USER}" defaults write <domain> <key> -array
```

`launchctl asuser <uid>` executes the command inside the user's bootstrap session, giving
`defaults` a path to the correct `cfprefsd` instance.

The double-wrap (`launchctl asuser` + `sudo -u`) is the recommended pattern from Armin Briegel's
article (see reference below) because it works in all contexts — including when the command must
run as the user *and* have access to their session state.

## Other commands that need this treatment

Any command that talks to user-session infrastructure from a LaunchDaemon:

| Command | Why it needs user context |
|---|---|
| `defaults` | Communicates with `cfprefsd` via bootstrap port |
| `osascript` | Needs a UI/scripting session |
| `open` | Targets the user's window server |
| `launchctl` (load/unload agents) | Agents live in the user's launchd domain |
| `plutil` (writing prefs) | Same cfprefsd path as `defaults` |

## Debugging checklist

1. **Is the daemon even firing?** — `sudo launchctl print system/<label>` shows `runs = N`
   and last exit code.
2. **Does the script run but commands fail?** — Check `StandardOutPath` / `StandardErrorPath`
   log. Remove `2>/dev/null` temporarily to capture real errors.
3. **Does the command work interactively but not from the daemon?** — Bootstrap namespace
   mismatch. Add `launchctl asuser <uid>`.
4. **Is cfprefsd even running for the user?** — `ps aux | grep cfprefsd`. If the user is
   logged out, there may be no session to join; consider whether the daemon should only run
   when the user is logged in.
5. **Is there a `.blocked` file?** — A `<domain>.plist.blocked` sentinel in the Preferences
   directory can indicate sandbox/TCC restrictions on that domain.

## Reference

- Armin Briegel, "Running a Command as Another User" (2020):
  https://scriptingosx.com/2020/08/running-a-command-as-another-user/
- Apple TN: `launchctl asuser` docs — `man launchctl`, search "asuser"

## Applied in this repo

`flush_wallpaper_cache.sh` runs via a **LaunchAgent** (`~/Library/LaunchAgents/`) rather than
a LaunchDaemon. This gives it native access to the user's cfprefsd session, so `defaults write`
works without any sudo or bootstrap bridging. All cache files (JPG, BMP) are user-owned, so no
elevated privileges are needed for file deletion either.

History of attempts on the `defaults write` problem:
1. `sudo -u blake defaults write ...` from LaunchDaemon — fails silently (wrong bootstrap namespace)
2. `launchctl asuser <uid> sudo -u blake defaults write ...` from LaunchDaemon — still fails when display is asleep (no active user session to attach to)
3. **LaunchAgent** — works: runs inside the user session regardless of display sleep state
