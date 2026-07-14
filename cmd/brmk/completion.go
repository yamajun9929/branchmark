package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
	"github.com/yamajun9929/branchmark/internal/tui"
)

func runCompletion(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: brmk completion [zsh|bash|fish]")
	}
	switch args[0] {
	case "zsh":
		printZshCompletion()
	case "bash":
		printBashCompletion()
	case "fish":
		printFishCompletion()
	default:
		return errors.New("usage: brmk completion [zsh|bash|fish]")
	}
	return nil
}

func runInternalComplete(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: brmk __complete [profiles|spaces|themes]")
	}
	switch args[0] {
	case "profiles":
		return completeProfiles(args[1:])
	case "spaces":
		return completeSpaces(args[1:])
	case "themes":
		return completeThemes(args[1:])
	default:
		return errors.New("usage: brmk __complete [profiles|spaces|themes]")
	}
}

func completeProfiles(args []string) error {
	fs := flag.NewFlagSet("brmk __complete profiles", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	cfg, err := bookmarks.LoadConfig(*configPath)
	if errors.Is(err, os.ErrNotExist) {
		cfg = bookmarks.DefaultConfig()
	} else if err != nil {
		return err
	}
	var names []string
	for _, profile := range bookmarks.NormalizeConfig(cfg).BrowserProfiles {
		names = append(names, profile.Name)
	}
	printCompletionLines(names)
	return nil
}

func completeSpaces(args []string) error {
	fs := flag.NewFlagSet("brmk __complete spaces", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataPath := fs.String("data", bookmarks.DefaultStorePath(), "bookmark data file")
	normalized, err := intersperseFlags(args, map[string]bool{"data": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	store, err := bookmarks.Load(*dataPath)
	if err != nil {
		return err
	}
	printCompletionLines(folderPaths(store))
	return nil
}

func completeThemes(args []string) error {
	fs := flag.NewFlagSet("brmk __complete themes", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	cfg, err := bookmarks.LoadConfig(*configPath)
	if errors.Is(err, os.ErrNotExist) {
		cfg = bookmarks.DefaultConfig()
	} else if err != nil {
		return err
	}
	printCompletionLines(tui.ThemeNames(cfg))
	return nil
}

func printCompletionLines(values []string) {
	values = uniqueCompletionLines(values)
	for _, value := range values {
		fmt.Println(value)
	}
}

func uniqueCompletionLines(values []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	sort.Strings(cleaned)
	return cleaned
}

func printZshCompletion() {
	fmt.Print(strings.TrimLeft(`
#compdef brmk

_brmk_profiles() {
  local -a profiles
  profiles=("${(@f)$(_call_program brmk-profiles "$words[1]" __complete profiles 2>/dev/null)}")
  _describe -t profiles 'profile' profiles
}

_brmk_spaces() {
  local -a paths
  paths=("${(@f)$(_call_program brmk-spaces "$words[1]" __complete spaces 2>/dev/null)}")
  _describe -t paths 'folder path' paths
}

_brmk_themes() {
  local -a themes
  themes=("${(@f)$(_call_program brmk-themes "$words[1]" __complete themes 2>/dev/null)}")
  _describe -t themes 'theme' themes
}

_brmk() {
  local curcontext="$curcontext" state line
  typeset -A opt_args
  local -a commands profile_commands theme_commands shells kinds

  commands=(
    'add:add a bookmark'
    'config:create and print the config file path'
    'completion:print shell completion script'
    'export:export bookmarks as Markdown'
    'import:import bookmarks from Markdown'
    'path:print the data file path'
    'profile:manage browser profiles'
    'theme:manage color themes'
    'version:print the version'
    'help:show help'
  )
  profile_commands=(
    'list:list browser profiles'
    'create:create a browser profile'
    'select:select active browser profile'
    'path:print managed profile directory'
  )
  theme_commands=('list:list color themes' 'set:select a color theme')
  shells=('zsh:zsh completion' 'bash:bash completion' 'fish:fish completion')
  kinds=('default:use the OS default browser' 'managed:use a brmk-managed browser profile' 'existing:use an existing browser profile')

  if (( CURRENT == 2 )); then
    _describe -t commands 'brmk command' commands
    return
  fi

  case "$words[2]" in
    add)
      _arguments -C \
        '--data=[bookmark data file]:file:_files' \
        '--config=[config file]:file:_files' \
        '--browser=[browser app name]:browser:' \
        '--space=[folder path]:folder path:->spaces' \
        '--title=[bookmark title]:title:' \
        '--tags=[comma-separated tags]:tags:' \
        '--no-prompt[add without prompting]' \
        '--yes[add without prompting]' \
        '--dry-run[show without saving]' \
        '--verbose[show config, data, and source]' \
        '1:URL:_urls'
      if [[ "$state" == spaces ]]; then
        _brmk_spaces
      fi
      ;;
    config)
      _arguments '--config=[config file]:file:_files'
      ;;
    completion)
      _describe -t shells 'shell' shells
      ;;
    export)
      _arguments '--data=[bookmark data file]:file:_files' '1:output file:_files'
      ;;
    import)
      _arguments \
        '--data=[bookmark data file]:file:_files' \
        '--merge[merge matching folders]' \
        '--replace[replace current store]' \
        '1:Markdown file:_files'
      ;;
    profile)
      if (( CURRENT == 3 )); then
        _describe -t profile-commands 'profile command' profile_commands
      else
        case "$words[3]" in
          list)
            _arguments '--config=[config file]:file:_files'
            ;;
          create)
            _arguments \
              '--config=[config file]:file:_files' \
              '--browser=[browser app name]:browser:' \
              '--kind=[profile kind]:kind:(default managed existing)' \
              '--path=[profile path]:path:_files -/' \
              '--directory=[existing browser profile directory]:directory:' \
              '--select[select this profile after creation]' \
              '1:profile name:'
            ;;
          select|path)
            _arguments '--config=[config file]:file:_files' '1:profile name:_brmk_profiles'
            ;;
        esac
      fi
      ;;
    theme)
      if (( CURRENT == 3 )); then
        _describe -t theme-commands 'theme command' theme_commands
      else
        case "$words[3]" in
          list)
            _arguments '--config=[config file]:file:_files'
            ;;
          set)
            _arguments '--config=[config file]:file:_files' '1:theme:_brmk_themes'
            ;;
        esac
      fi
      ;;
    path|version|help)
      ;;
  esac
}

_brmk "$@"
`, "\n"))
}

func printBashCompletion() {
	fmt.Print(strings.TrimLeft(`
_brmk_complete_from_lines() {
  local cur="$1"
  local item
  COMPREPLY=()
  while IFS= read -r item; do
    [[ "$item" == "$cur"* ]] && COMPREPLY+=("$item")
  done
}

_brmk() {
  local cur prev cmd sub
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  cmd="${COMP_WORDS[0]}"

  case "$prev" in
    --data|--config|--path)
      COMPREPLY=( $(compgen -f -- "$cur") )
      return 0
      ;;
    --space)
      _brmk_complete_from_lines "$cur" < <("$cmd" __complete spaces 2>/dev/null)
      return 0
      ;;
  esac

  if [[ "$COMP_CWORD" -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "add config completion export import path profile theme version help" -- "$cur") )
    return 0
  fi

  case "${COMP_WORDS[1]}" in
    add)
      COMPREPLY=( $(compgen -W "--data --config --browser --space --title --tags --no-prompt --yes --dry-run --verbose" -- "$cur") )
      ;;
    completion)
      COMPREPLY=( $(compgen -W "zsh bash fish" -- "$cur") )
      ;;
    import)
      COMPREPLY=( $(compgen -W "--data --merge --replace" -- "$cur") )
      [[ ${#COMPREPLY[@]} -eq 0 ]] && COMPREPLY=( $(compgen -f -- "$cur") )
      ;;
    export)
      COMPREPLY=( $(compgen -W "--data" -- "$cur") )
      [[ ${#COMPREPLY[@]} -eq 0 ]] && COMPREPLY=( $(compgen -f -- "$cur") )
      ;;
    config)
      COMPREPLY=( $(compgen -W "--config" -- "$cur") )
      ;;
    profile)
      sub="${COMP_WORDS[2]}"
      if [[ "$COMP_CWORD" -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "list create select path" -- "$cur") )
      elif [[ "$sub" == "select" || "$sub" == "path" ]]; then
        _brmk_complete_from_lines "$cur" < <("$cmd" __complete profiles 2>/dev/null)
      elif [[ "$sub" == "create" ]]; then
        COMPREPLY=( $(compgen -W "--config --browser --kind --path --directory --select" -- "$cur") )
      elif [[ "$sub" == "list" ]]; then
        COMPREPLY=( $(compgen -W "--config" -- "$cur") )
      fi
      ;;
    theme)
      sub="${COMP_WORDS[2]}"
      if [[ "$COMP_CWORD" -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "list set" -- "$cur") )
      elif [[ "$sub" == "set" ]]; then
        _brmk_complete_from_lines "$cur" < <("$cmd" __complete themes 2>/dev/null)
      elif [[ "$sub" == "list" ]]; then
        COMPREPLY=( $(compgen -W "--config" -- "$cur") )
      fi
      ;;
  esac
}

complete -F _brmk brmk
`, "\n"))
}

func printFishCompletion() {
	fmt.Print(strings.TrimLeft(`
complete -c brmk -f
complete -c brmk -n '__fish_use_subcommand' -a add -d 'Add a bookmark'
complete -c brmk -n '__fish_use_subcommand' -a config -d 'Create and print the config file path'
complete -c brmk -n '__fish_use_subcommand' -a completion -d 'Print shell completion script'
complete -c brmk -n '__fish_use_subcommand' -a export -d 'Export bookmarks as Markdown'
complete -c brmk -n '__fish_use_subcommand' -a import -d 'Import bookmarks from Markdown'
complete -c brmk -n '__fish_use_subcommand' -a path -d 'Print the data file path'
complete -c brmk -n '__fish_use_subcommand' -a profile -d 'Manage browser profiles'
complete -c brmk -n '__fish_use_subcommand' -a theme -d 'Manage color themes'
complete -c brmk -n '__fish_use_subcommand' -a version -d 'Print the version'
complete -c brmk -n '__fish_seen_subcommand_from completion' -a 'zsh bash fish'

complete -c brmk -n '__fish_seen_subcommand_from add' -l data -r -F -d 'Bookmark data file'
complete -c brmk -n '__fish_seen_subcommand_from add' -l config -r -F -d 'Config file'
complete -c brmk -n '__fish_seen_subcommand_from add' -l browser -r -d 'Browser app name'
complete -c brmk -n '__fish_seen_subcommand_from add' -l space -r -a '(brmk __complete spaces)' -d 'Folder path'
complete -c brmk -n '__fish_seen_subcommand_from add' -l title -r -d 'Bookmark title'
complete -c brmk -n '__fish_seen_subcommand_from add' -l tags -r -d 'Comma-separated tags'
complete -c brmk -n '__fish_seen_subcommand_from add' -l no-prompt -d 'Add without prompting'
complete -c brmk -n '__fish_seen_subcommand_from add' -l yes -d 'Add without prompting'
complete -c brmk -n '__fish_seen_subcommand_from add' -l dry-run -d 'Show without saving'
complete -c brmk -n '__fish_seen_subcommand_from add' -l verbose -d 'Show config, data, and source'

complete -c brmk -n '__fish_seen_subcommand_from config' -l config -r -F -d 'Config file'
complete -c brmk -n '__fish_seen_subcommand_from export' -l data -r -F -d 'Bookmark data file'
complete -c brmk -n '__fish_seen_subcommand_from import' -l data -r -F -d 'Bookmark data file'
complete -c brmk -n '__fish_seen_subcommand_from import' -l merge -d 'Merge matching folders'
complete -c brmk -n '__fish_seen_subcommand_from import' -l replace -d 'Replace current store'

complete -c brmk -n '__fish_seen_subcommand_from profile; and not __fish_seen_subcommand_from list create select path' -a list -d 'List browser profiles'
complete -c brmk -n '__fish_seen_subcommand_from profile; and not __fish_seen_subcommand_from list create select path' -a create -d 'Create browser profile'
complete -c brmk -n '__fish_seen_subcommand_from profile; and not __fish_seen_subcommand_from list create select path' -a select -d 'Select active browser profile'
complete -c brmk -n '__fish_seen_subcommand_from profile; and not __fish_seen_subcommand_from list create select path' -a path -d 'Print managed profile directory'
complete -c brmk -n '__fish_seen_subcommand_from profile; and __fish_seen_subcommand_from list create select' -l config -r -F -d 'Config file'
complete -c brmk -n '__fish_seen_subcommand_from profile; and __fish_seen_subcommand_from create' -l browser -r -d 'Browser app name'
complete -c brmk -n '__fish_seen_subcommand_from profile; and __fish_seen_subcommand_from create' -l kind -r -a 'default managed existing' -d 'Profile kind'
complete -c brmk -n '__fish_seen_subcommand_from profile; and __fish_seen_subcommand_from create' -l path -r -F -d 'Profile path'
complete -c brmk -n '__fish_seen_subcommand_from profile; and __fish_seen_subcommand_from create' -l directory -r -d 'Existing browser profile directory'
complete -c brmk -n '__fish_seen_subcommand_from profile; and __fish_seen_subcommand_from create' -l select -d 'Select this profile after creation'
complete -c brmk -n '__fish_seen_subcommand_from profile; and __fish_seen_subcommand_from select path' -a '(brmk __complete profiles)' -d 'Profile'
complete -c brmk -n '__fish_seen_subcommand_from theme; and not __fish_seen_subcommand_from list set' -a list -d 'List color themes'
complete -c brmk -n '__fish_seen_subcommand_from theme; and not __fish_seen_subcommand_from list set' -a set -d 'Select a color theme'
complete -c brmk -n '__fish_seen_subcommand_from theme; and __fish_seen_subcommand_from list set' -l config -r -F -d 'Config file'
complete -c brmk -n '__fish_seen_subcommand_from theme; and __fish_seen_subcommand_from set' -a '(brmk __complete themes)' -d 'Theme'
`, "\n"))
}
