# Branchmark

[日本語版](README.ja.md)

`Branchmark` is a lightweight TUI tool for managing URL bookmarks in a tree. Its command is `brmk`.

<img width="850" height="420" alt="Image" src="https://github.com/user-attachments/assets/7191ac53-7be6-44c9-8d7b-8eb2691c12e3" />

## Features

- **⚡ Instant & Interactive TUI**: Manage, browse, and organize all your bookmarks from a beautiful, responsive terminal interface.
- **🌐 Active Browser Integration (`add` command)**: Automatically capture the active URL and title from your current browser tab with a single command (supports Chrome, Brave, Edge, Firefox, and more).
- **🔍 Blazing Fast Tag Filtering**: Instantly search and filter thousands of bookmarks by title, URL, or custom tags in real-time.
- **📂 Multi-Space Hierarchy**: Group your bookmarks into separate workspaces (Spaces) like `Work` and `Personal` with full folder-tree support.
- **🔒 Isolated Browser Profiles**: Open selected URLs in isolated browser environments/profiles—perfect for separating work and private sessions.
- **🎨 Rich Custom Themes**: Built-in support for popular developer themes like Catppuccin, Tokyo Night, Dracula, and Nord, plus custom JSON themes.

Design principles:

- Distribute a single Go binary that is easy to install with Homebrew
- Avoid external Go dependencies to keep the vulnerability-management surface small
- Keep only the optional local add API running; start the TUI only when needed
- Use ANSI escape sequences and the standard library for rendering to keep CPU and memory usage low

## Build

```sh
go build -trimpath -ldflags "-s -w" -o brmk ./cmd/brmk
```

## Usage

```sh
./brmk
./brmk add
./brmk add https://example.com --space Work --title Example --tags web,readlater
./brmk add https://example.com --space Work --title Example --no-prompt
./brmk add https://example.com --space ? --title Example
./brmk add https://example.com --space Work --title Example --dry-run
./brmk add https://example.com --verbose
./brmk export bookmarks.md
./brmk import bookmarks.md
./brmk import bookmarks.md --merge
./brmk profile create work --browser "Google Chrome"
./brmk profile select work
./brmk config
./brmk theme list
./brmk theme set tokyonight
./brmk completion zsh
```

When the URL is omitted, `add` captures the active tab from the configured browser. It prompts for any missing fields. If the title is left empty, it fetches the page's HTML `<title>`; if that cannot be fetched, the URL is used as the title.

`--space` accepts a hierarchical path such as `Work/Engineering`; missing folders are created. During an interactive Space prompt, existing folder paths are shown as you type, and `Tab` / `Shift-Tab` completes or cycles through them. `--space ?` lists the existing folder paths before prompting. `--no-prompt` adds without confirmation, and `--dry-run` prints what would be added without saving. Use `--verbose` to see the config path, data path, and input source.

The standard TUI view features Nerd Font glyphs, similar to NeoVim file explorers (such as neo-tree or nvim-tree).

```text
▾  Bookmarks
├── ▾  Work
│   ├──    GitHub
│   ├──    Go
│   ├──   󰗃 YouTube
│   ├──   󰊶 Google Drive
│   ├──   󰧷 Google Sheets
│   ├──   󰊫 Gmail
│   ├──   󱇶 Google Cloud
│   ├──   󰸏 AWS
│   ├──   󰢎 Salesforce
│   ├──   󰍲 Microsoft
│   ├──   󰏆 Office
│   ├──   󰊻 Teams
│   ├──   󰴢 Outlook
│   ├──   󰏊 OneDrive
│   ├──    Yahoo
│   └──   󰖟 Example
```

Nerd Font icons are selected from the URL domain and, for some services, the path. Dedicated icons are supported for GitHub, YouTube, Google services (Drive, Sheets, Docs, Slides, Forms, Gmail, Calendar, Meet, Maps, Cloud, Analytics, Ads, Classroom, Keep, Play, Translate, Earth, Firebase, Chrome, and more), AWS, Salesforce, Microsoft services (Office, Teams, Outlook, OneDrive, SharePoint, Excel, Word, PowerPoint, OneNote, Azure, Bing, and more), and Yahoo.

Top-level folders are treated as Spaces and can be switched from the tabs at the top of the TUI. The Space selected in the TUI is saved as `default_space` in the config and becomes the default for the next `brmk add`. Profiles are not tied to Spaces: they represent the browser environment currently in use throughout the TUI. The selected URL opens in a new window of the active profile.

```sh
./brmk profile list
./brmk profile create work --browser "Google Chrome"
./brmk profile create private --browser "Brave Browser"
./brmk profile select work
./brmk profile path work
```

A `managed` profile has a dedicated directory created by `brmk`. Chromium-based browsers use `--user-data-dir`, and Firefox uses `--profile`. The `default` profile continues to use the operating system's default browser. Arc is difficult to control reliably as an isolated profile from an external CLI, so Chrome, Brave, Edge, or Firefox is recommended when profile isolation is needed.

By default, the data file is `~/.config/brmk/bookmarks.json`. You can also check it with `brmk path`. To use another file, pass `--data FILE` or set `BRMK_DATA`.

## Shell Completion

Completion scripts are available for `zsh`, `bash`, and `fish`.

```sh
./brmk completion zsh
./brmk completion bash
./brmk completion fish
```

To load it only in the current `zsh` session:

```sh
source <(./brmk completion zsh)
```

`profile select` and `profile path` complete configured profile names. `add --space` completes existing folder paths. `--data`, `--config`, and the file arguments for import and export use file-path completion.

## Config

Create the config file and print its path with `brmk config`. By default, it is `~/.config/brmk/config.json`. When `XDG_CONFIG_HOME` is set, it is `$XDG_CONFIG_HOME/brmk/config.json`. To override it explicitly, set `BRMK_CONFIG` or pass `--config FILE`.

```json
{
  "active_profile": "default",
  "web_browser": "Google Chrome",
  "default_space": "Inbox",
  "theme": "catppuccin-mocha",
  "browser_profiles": [
    {
      "name": "default",
      "kind": "default"
    },
    {
      "name": "work",
      "browser": "Google Chrome",
      "kind": "managed"
    }
  ]
}
```

## Theme

List the TUI color themes with `brmk theme list`, then select one with `brmk theme set NAME`. The built-in themes are `catppuccin-mocha` (the default), `tokyonight`, `dracula`, `nord`, `gruvbox-dark`, `gruvbox-light`, `monochrome`, and `terminal`.

`terminal` inherits the background and foreground colors of your terminal theme. It uses reverse video and bold text only for selection and borders.

Add custom themes under `themes` in the config. Colors use the `#RRGGBB` format; omitted colors inherit from `catppuccin-mocha`.

```json
{
  "theme": "my-theme",
  "themes": {
    "my-theme": {
      "background": "#101418",
      "selection_background": "#28313a",
      "foreground": "#d8dee9",
      "muted": "#8292a2",
      "accent": "#88c0d0",
      "highlight": "#81a1c1",
      "success": "#a3be8c",
      "border": "#88c0d0"
    }
  }
}
```

## TUI Keys

| Key | Action |
| --- | --- |
| `tab` / `Shift-tab` | Switch to the next / previous Space tab |
| `p` | Cycle browser profiles |
| `P` | Select or create a browser profile |
| `j` / `k` | Move down / up |
| `[` / `]` | Jump to the previous / next visible folder |
| `J` / `K` | Move the selected item down / up within its folder |
| Mouse wheel | Scroll |
| `Ctrl-u` / `Ctrl-d` | Scroll up / down |
| `PageUp` / `PageDown` | Scroll up / down |
| `h` / `l` | Collapse / expand |
| `enter` / `o` | Open a URL or toggle a folder |
| `H` / `L` | Move the current Space tab left / right |
| `S` | Add a new top-level Space tab |
| `a` | Add a bookmark under the current folder |
| `A` | Add a folder under the current folder |
| `e` | Edit the selected item |
| `m` | Move the selected item with a folder picker; type to filter and press `tab` to complete the current path |
| `r` | Rename the title |
| `R` | Reload bookmarks from disk |
| `t` | Edit tags |
| `d` | Delete the selected item |
| `u` | Undo the last delete, move, or reorder |
| `/` | Search |
| `c` | Clear the search |
| `?` | Show or hide the help pane |
| `g` / `G` or `Home` / `End` | Jump to the top / bottom |
| Left click | Select a row |
| Left click a Space tab | Switch Space tabs |
| Click the selected row again | Open a URL or toggle a folder |
| `q` | Quit |

Prompt editing:

- `tab` / `Shift-tab`: complete or cycle candidates where available, such as Profile input
- `Left` / `Right`: move the cursor
- `Home` / `End`: move to the start / end
- `Backspace` / `Delete`: delete around the cursor
- `Ctrl-a`: select all
- `Ctrl-u`: clear all
- Typing while all text is selected replaces the whole value

macOS terminal apps normally do not send `Cmd` key combinations to TUI programs. With WezTerm, map `Cmd-a` to `Ctrl-a` only while `brmk` is the foreground process:

```lua
local function is_brmk_process(pane)
  local ok, process_name = pcall(function()
    return pane:get_foreground_process_name()
  end)
  if not ok or not process_name then
    return false
  end
  return (process_name:match("([^/]+)$") or process_name):lower() == "brmk"
end

config.keys = config.keys or {}
table.insert(config.keys, {
  key = 'a',
  mods = 'CMD',
  action = wezterm.action_callback(function(window, pane)
    if is_brmk_process(pane) then
      window:perform_action(wezterm.action.SendKey { key = 'a', mods = 'CTRL' }, pane)
    else
      window:perform_action(wezterm.action.Nop, pane)
    end
  end),
})
```

The same behavior is expressed in this repository's local WezTerm configuration as follows:

```lua
local function brmk_key_or(send_key, fallback)
  return wezterm.action_callback(function(window, pane)
    if is_brmk_process(pane) then
      window:perform_action(wezterm.action.SendKey(send_key), pane)
    else
      window:perform_action(fallback)
    end
  end)
end

config.keys = {
  {
    key = 'a',
    mods = 'CMD',
    action = brmk_key_or({ key = 'a', mods = 'CTRL' }, wezterm.action.Nop),
  },
}
```

## Markdown Format

Export and import use an indentation-based Markdown format:

```md
# Branchmark bookmark tree v1

- space: Work {tags=team,docs}
  - [Go](https://go.dev) {tags=golang,docs}
  - folder: Tools
    - [Example](https://example.com)
```

Top-level folders are exported as `- space: Name`. Nested folders use `- folder: Name`. Legacy files using `- folder: Name` at the top level are still fully supported for import. Bookmarks use normal Markdown links. Metadata is optional:

- `tags=tag1,tag2`

Import behavior:

- `brmk import FILE`: append the imported top-level nodes as-is
- `brmk import FILE --merge`: merge imported folders into matching existing folders at the same tree level
- `brmk import FILE --replace`: replace the current store with the imported Markdown

A merge-import starter file is available at `examples/merge-import.md`:

```sh
./brmk import examples/merge-import.md --merge
```

## Homebrew

After creating a release tag, update the `url` and `sha256` in `Formula/brmk.rb` to match the actual tarball.

```sh
brew install ./Formula/brmk.rb
```

To publish an official tap, put the Formula in a tap repository such as `homebrew-brmk`.

