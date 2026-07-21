# Branchmark

[English version](README.md)

`Branchmark` は、URL ブックマークをツリーで管理する軽量 TUI ツールです。コマンド名は `brmk` です。

<img width="850" height="420" alt="Image" src="https://github.com/user-attachments/assets/7191ac53-7be6-44c9-8d7b-8eb2691c12e3" />

## 主な機能

- **⚡ 軽快で直感的な TUI**: 美しく応答性に優れたターミナルインターフェースから、すべてのブックマークを整理・閲覧できます。
- **🌐 ブラウザ連携と自動追加 (`add` コマンド)**: コマンド1つで、現在ブラウザで開いているアクティブなタブのURLとタイトルを自動キャプチャして追加します（Chrome、Brave、Edge、Firefox等に対応）。
- **🔍 爆速のタグ・キーワードフィルタリング**: タイトル、URL、またはカスタムタグを使って、膨大なブックマークを一瞬でリアルタイム検索・フィルタリングします。
- **📂 マルチスペースと階層ツリー**: `Work` や `Personal` などのワークスペース（Space）をタブで切り替え、フォルダーツリーによる無限の階層管理をサポート。
- **🔒 ブラウザプロフィールの隔離**: 選択したブックマークを隔離されたブラウザ環境（Profile）で開くことができ、仕事用と個人用セッションを完全に分離。
- **🎨 豊富なカスタムテーマ**: Catppuccin、Tokyo Night、Dracula、Nord などの人気カラーテーマを内蔵。カスタムJSONテーマも作成可能。

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

## TUI 操作キー

| キー | アクション |
| --- | --- |
| `tab` / `Shift-tab` | Spaceタブを順方向 / 逆方向に切り替え |
| `p` | ブラウザプロフィールを順に切り替え |
| `P` | ブラウザプロフィールを選択または作成 |
| `j` / `k` | カーソルを下に移動 / 上に移動 |
| `[` / `]` | 前 / 次の表示されているフォルダーへジャンプ |
| `J` / `K` | 選択中の項目を同じフォルダー内で下 / 上に移動（並べ替え） |
| マウスホイール | スクロール |
| `Ctrl-u` / `Ctrl-d` | 上にスクロール / 下にスクロール |
| `PageUp` / `PageDown` | 上にスクロール / 下にスクロール |
| `h` / `l` | フォルダーを閉じる / 展開する |
| `enter` / `o` | URLを開く、またはフォルダーの展開を切り替え |
| `S` | ルート直下に新しいSpaceタブを追加 |
| `a` | 選択中のフォルダー配下にブックマークを追加 |
| `A` | 選択中のフォルダー配下に新しいフォルダーを追加 |
| `e` | 選択中の項目を編集 |
| `m` | 選択中の項目をフォルダーピッカーで移動（文字入力でフィルタ、`Tab`キーで補完） |
| `r` | タイトルを変更 |
| `R` | ディスクからブックマークを再読み込み |
| `t` | タグを編集 |
| `d` | 選択中の項目を削除 |
| `u` | 直前の削除、移動、または並べ替え操作を元に戻す（Undo） |
| `/` | 検索（フィルタリング）を実行 |
| `c` | 検索フィルターをクリア |
| `?` | ヘルプペインを表示または非表示に切り替え |
| `g` / `G` または `Home` / `End` | ツリーの先頭 / 末尾にジャンプ |
| 左クリック | 行を選択 |
| Spaceタブを左クリック | Spaceタブを切り替え |
| 選択中の行を再度クリック | URLを開く、またはフォルダーの展開を切り替え |
| `q` | 終了 |

プロンプト入力時の編集キー:

- `tab` / `Shift-tab`: プロフィール入力などで、候補を補完または順次切り替え
- `Left` / `Right`: カーソルを移動
- `Home` / `End`: 行頭 / 行末にカーソルを移動
- `Backspace` / `Delete`: カーソルの前後を削除
- `Ctrl-a`: すべて選択
- `Ctrl-u`: すべてクリア
- 全選択状態で文字入力すると入力した値で上書き

macOSのターミナルアプリは通常、`Cmd`キーの組み合わせをTUIプログラムに送信しません。WezTermを使用する場合、`brmk`がフォアグラウンドプロセスである時のみ `Cmd-a` を `Ctrl-a` にマッピングできます：

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

本リポジトリのローカル WezTerm 設定では、同様の挙動を以下のように記述できます。

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

## Markdown フォーマット

エクスポートとインポートは、インデント（スペースによる階層化）ベースのMarkdownフォーマットを使用します。

```md
# Branchmark bookmark tree v1

- space: Work {tags=team,docs}
  - [Go](https://go.dev) {tags=golang,docs}
  - folder: Tools
    - [Example](https://example.com)
```

最上位のフォルダーは `- space: 名前` としてエクスポートされます。入れ子になったフォルダーは `- folder: 名前` を使用します。既存のファイルで最上位レベルに `- folder: 名前` を使用している場合でもインポート可能です。ブックマークは通常のMarkdownリンクを使用します。メタデータはオプションです：

- `tags=tag1,tag2`

インポートの挙動：

- `brmk import FILE`: インポートされた最上位ノードをそのまま末尾に追加します。
- `brmk import FILE --merge`: インポートされたフォルダーを、同じツリー階層にある既存の同名フォルダーとマージします。
- `brmk import FILE --replace`: 現在の保存データを、インポートしたMarkdownの内容で完全に置き換えます。

マージインポート用のスターターテンプレートファイルが `examples/merge-import.md` に用意されています。

```sh
./brmk import examples/merge-import.md --merge
```

## Homebrew

リリースタグ作成後、`Formula/brmk.rb` の `url` と `sha256` を実際の tarball に合わせて更新します。

```sh
brew install ./Formula/brmk.rb
```

公式 tap にする場合は、Formula を `homebrew-brmk` などの tap リポジトリへ置きます。
