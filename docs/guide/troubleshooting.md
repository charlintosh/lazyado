# Troubleshooting

- [lazyado exits right after starting](#lazyado-exits-right-after-starting)
- [API errors](#api-errors)
- [I can't edit my comments or vote](#i-cant-edit-my-comments-or-vote)
- [Mouse hover and drag don't work](#mouse-hover-and-drag-dont-work)
- [Debug logging](#debug-logging)
- [Reporting a bug](#reporting-a-bug)

## lazyado exits right after starting

**"Created default config file at ~/.config/lazyado/config.yaml"**: lazyado couldn't load a complete configuration. Open the file and make sure `organization`, `project`, `team` and `pat` are all set (or set the matching [environment variables](configuration.md#environment-variables)). lazyado never overwrites an existing file, so if you already had one, a required value is missing from it.

**"… is required (set in config or AZURE_DEVOPS_…)"**: the named setting is empty. See [Configuration](configuration.md).

## API errors

Errors from Azure DevOps are shown in a dialog with the message the server returned.

| Status | Usual cause |
| ------ | ----------- |
| `401` | The token is wrong or has expired. Create a new one. |
| `403` | The token is missing a scope for that action. See [the scopes lazyado needs](configuration.md#personal-access-token). |
| `404` | The organization, project or team name is misspelled, or you don't have access to it. |

After fixing the token or configuration, restart lazyado. For temporary failures, `Ctrl+r` reloads.

## I can't edit my comments or vote

lazyado decides which comments are yours by asking Azure DevOps who owns the token. If that request fails, comments become read-only and voting is disabled, and lazyado tells you so when you try. Press `Ctrl+r` to retry. See [Who you are](configuration.md#who-you-are).

`e` and `d` only work on comments you wrote, and `+` only on other people's. This is by design.

## Mouse hover and drag don't work

Dragging tasks on the Sprints taskboard needs a terminal that reports mouse events. The hover effect also needs mouse *motion* reporting. Most modern terminals support both, but terminal multiplexers such as tmux may need mouse support turned on (`set -g mouse on`). Everything you can do with the mouse also has a key.

## Debug logging

Start lazyado with `-debug` to log every API request and its result:

```bash
lazyado -debug
```

The log is written to `lazyado-debug.log` in your system's temporary directory (`/tmp/lazyado-debug.log` on Linux, `$TMPDIR/lazyado-debug.log` on macOS, `%TEMP%\lazyado-debug.log` on Windows). It is cleared each time you start with `-debug`.

## Reporting a bug

Open an issue on [GitHub](https://github.com/charlintosh/lazyado/issues) with:

- the lazyado version, your operating system and terminal;
- what you did, what you expected, and what happened;
- the relevant part of the debug log.

The log contains request URLs, which include your organization and project names. Remove anything you don't want to share before posting it. Your token is not written to the log.
