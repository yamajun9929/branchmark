# Branchmark

[English version](README.md)

`Branchmark` は、URL ブックマークをツリーで管理する軽量 TUI ツールです。コマンド名は `brmk` です。

<img width="850" height="420" alt="Image" src="https://github.com/user-attachments/assets/7191ac53-7be6-44c9-8d7b-8eb2691c12e3" />

設計方針:

- Go の単体バイナリとして配布し、Homebrew で入れやすくする
- 外部 Go 依存を持たず、脆弱性管理の対象を増やしにくくする
- 常駐するのは任意のローカル追加 API だけにし、TUI は必要なときだけ起動する
- 描画は ANSI エスケープと標準ライブラリ中心で行い、CPU/メモリ使用量を抑える

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

`add` は URL を省略すると、設定したブラウザの現在タブを取得して追加します。足りない項目は入力ラインで確認し、タイトルを空にした場合は URL 先の HTML `<title>` を自動取得します。取得できない場合は URL がタイトルになります。`--space` には `Work/Engineering` のような階層パスを指定でき、存在しないフォルダは作成されます。対話入力時は既存フォルダを入力に合わせて候補表示し、`Tab` / `Shift-Tab` で補完できます。`--space ?` は既存のフォルダパスを表示してから入力できます。`--no-prompt` は確認なしで追加し、`--dry-run` は保存せず追加内容だけ表示します。config path、data path、取得元を確認したい場合は `--verbose` を使います。

TUI の通常表示は、nvim のファイルエクスプローラーに近い Nerd Font 風の文字アイコンです。

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

Nerd Font アイコンは URL のドメインと一部パスから選びます。GitHub、YouTube、Google 系サービス（Drive / Sheets / Docs / Slides / Forms / Gmail / Calendar / Meet / Maps / Cloud / Analytics / Ads / Classroom / Keep / Play / Translate / Earth / Firebase / Chrome など）、AWS、Salesforce、Microsoft 系サービス（Office / Teams / Outlook / OneDrive / SharePoint / Excel / Word / PowerPoint / OneNote / Azure / Bing など）、Yahoo は専用アイコンに対応しています。

ルート直下のフォルダは Space として扱われ、TUI上部のタブで切り替えられます。TUI で選択した Space は config の `default_space` に保存され、次回の `brmk add` の既定 Space になります。Profile は Space に紐づけず、TUI全体の「今使うブラウザ環境」として切り替えます。選択中のURLは現在のProfileの新規ウィンドウで開きます。

```sh
./brmk profile list
./brmk profile create work --browser "Google Chrome"
./brmk profile create private --browser "Brave Browser"
./brmk profile select work
./brmk profile path work
```

`managed` profile は `brmk` が専用ディレクトリを作り、Chromium系では `--user-data-dir`、Firefoxでは `--profile` で開きます。`default` profile は従来どおりOSの既定ブラウザで開きます。Arc は外部CLIから独立プロファイルを安定制御しづらいため、Profile分離用途では Chrome / Brave / Edge / Firefox が向いています。

データファイルの既定値は `~/.config/brmk/bookmarks.json` です。`brmk path` でも確認できます。別ファイルを使う場合は `--data FILE` または `BRMK_DATA` を使います。

## Shell Completion

`zsh` / `bash` / `fish` の補完スクリプトを出力できます。

```sh
./brmk completion zsh
./brmk completion bash
./brmk completion fish
```

`zsh` で手元のシェルだけに読み込む場合:

```sh
source <(./brmk completion zsh)
```

`profile select` / `profile path` は設定済みProfile名、`add --space` は既存のフォルダパスを補完します。`--data` や `--config`、import/exportのファイル引数はファイルパス補完を使います。

## Config

設定ファイルは `brmk config` で作成し、パスを確認できます。既定では `~/.config/brmk/config.json` です。`XDG_CONFIG_HOME` がある場合は `$XDG_CONFIG_HOME/brmk/config.json`、明示的に変える場合は `BRMK_CONFIG` または `--config FILE` を使います。

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

TUI のカラーテーマは `brmk theme list` で確認し、`brmk theme set NAME` で切り替えられます。組み込みテーマは `catppuccin-mocha`（既定）、`tokyonight`、`dracula`、`nord`、`gruvbox-dark`、`gruvbox-light`、`monochrome`、`terminal` です。

`terminal` は背景色と通常の文字色を指定せず、端末のテーマになじませます。選択行と枠線だけを反転・太字で強調します。

独自テーマは config の `themes` に追加できます。色は `#RRGGBB` 形式で、指定しない色は `catppuccin-mocha` を引き継ぎます。

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
| `tab` / `Shift-tab` | switch Space tab forward / backward |
| `p` | cycle browser profile |
| `P` | select or create browser profile |
| `j` / `k` | move down / up |
| `[` / `]` | jump to previous / next visible folder |
| `J` / `K` | reorder selected item down / up within the same folder |
| mouse wheel | scroll |
| `Ctrl-u` / `Ctrl-d` | scroll up / down |
| `PageUp` / `PageDown` | scroll up / down |
| `h` / `l` | collapse / expand |
| `enter` / `o` | open URL or toggle folder |
| `a` | add bookmark under the current folder |
| `A` | add folder under the current folder |
| `e` | edit selected item |
| `m` | move selected item with a folder picker; type to filter, `tab` completes the current path |
| `r` | rename title |
| `R` | reload bookmarks from disk |
| `t` | edit tags |
| `d` | delete selected item |
| `u` | undo the last delete, move, or reorder |
| `/` | search |
| `c` | clear search |
| `?` | show or hide the help pane |
| `g` / `G` or `Home` / `End` | jump to top / bottom |
| left click | select row |
| left click Space tab | switch Space tab |
| second click on selected row | open URL or toggle folder |
| `q` | quit |

Prompt editing:

- `tab` / `Shift-tab`: complete or cycle candidates where available, such as Profile input
- `Left` / `Right`: move cursor
- `Home` / `End`: move to start / end
- `Backspace` / `Delete`: delete around cursor
- `Ctrl-a`: select all
- `Ctrl-u`: clear all
- typing while all selected replaces the whole value

macOS terminal apps usually do not send `Cmd` key combinations to TUI programs. With WezTerm, map `Cmd-a` to `Ctrl-a` only while `brmk` is the foreground process:

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

In this repository's local WezTerm config, the same behavior is expressed as:

```lua
local function brmk_key_or(send_key, fallback)
  return wezterm.action_callback(function(window, pane)
    if is_brmk_process(pane) then
      window:perform_action(wezterm.action.SendKey(send_key), pane)
    else
      window:perform_action(fallback, pane)
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

Export/import uses an indentation-based Markdown format:

```md
# Branchmark bookmark tree v1

- space: Work {tags=team,docs}
  - [Go](https://go.dev) {tags=golang,docs}
  - folder: Tools
    - [Example](https://example.com)
```

Top-level folders are exported as `- space: Name`. Nested folders use `- folder: Name`. Existing files using `- folder: Name` at the top level are still importable. Bookmarks use normal Markdown links. Metadata is optional:

- `tags=tag1,tag2`

Import behavior:

- `brmk import FILE`: append the imported top-level nodes as-is
- `brmk import FILE --merge`: merge imported folders into matching existing folders at the same tree level
- `brmk import FILE --replace`: replace the current store with the imported Markdown

A merge import starter file is available at `examples/merge-import.md`:

```sh
./brmk import examples/merge-import.md --merge
```

## Homebrew

リリースタグ作成後、`Formula/brmk.rb` の `url` と `sha256` を実際の tarball に合わせて更新します。

```sh
brew install ./Formula/brmk.rb
```

公式 tap にする場合は、Formula を `homebrew-brmk` などの tap リポジトリへ置きます。
