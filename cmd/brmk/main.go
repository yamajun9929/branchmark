package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
	"github.com/yamajun9929/branchmark/internal/tui"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "brmk:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return runTUI(args)
	}

	switch args[0] {
	case "add":
		return runAdd(args[1:])
	case "config":
		return runConfig(args[1:])
	case "completion":
		return runCompletion(args[1:])
	case "export":
		return runExport(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "__complete":
		return runInternalComplete(args[1:])
	case "import":
		return runImport(args[1:])
	case "path":
		fmt.Println(bookmarks.DefaultStorePath())
		return nil
	case "profile":
		return runProfile(args[1:])
	case "theme":
		return runTheme(args[1:])
	case "version", "--version":
		fmt.Println(version)
		return nil
	default:
		return runTUI(args)
	}
}

func runTUI(args []string) error {
	fs := flag.NewFlagSet("brmk", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataPath := fs.String("data", bookmarks.DefaultStorePath(), "bookmark data file")
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"data": true, "config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	cfg, err := bookmarks.EnsureConfig(*configPath)
	if err != nil {
		return err
	}
	return tui.Run(*dataPath, *configPath, cfg)
}

func runConfig(args []string) error {
	fs := flag.NewFlagSet("brmk config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("usage: brmk config [--config FILE]")
	}
	if _, err := bookmarks.EnsureConfig(*configPath); err != nil {
		return err
	}
	fmt.Println(*configPath)
	return nil
}

func runProfile(args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "create":
		return runProfileCreate(args[1:])
	case "list":
		return runProfileList(args[1:])
	case "path":
		return runProfilePath(args[1:])
	case "select":
		return runProfileSelect(args[1:])
	default:
		return errors.New("usage: brmk profile [list|create|select|path]")
	}
}

func runTheme(args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list":
		return runThemeList(args[1:])
	case "set":
		return runThemeSet(args[1:])
	default:
		return errors.New("usage: brmk theme [list|set NAME]")
	}
}

func runThemeList(args []string) error {
	fs := flag.NewFlagSet("brmk theme list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: brmk theme list [--config FILE]")
	}
	cfg, err := bookmarks.EnsureConfig(*configPath)
	if err != nil {
		return err
	}
	for _, name := range tui.ThemeNames(cfg) {
		marker := " "
		if name == cfg.Theme {
			marker = "*"
		}
		fmt.Printf("%s %s\n", marker, name)
	}
	return nil
}

func runThemeSet(args []string) error {
	fs := flag.NewFlagSet("brmk theme set", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: brmk theme set NAME [--config FILE]")
	}
	cfg, err := bookmarks.EnsureConfig(*configPath)
	if err != nil {
		return err
	}
	name, ok := tui.CanonicalThemeName(cfg, fs.Arg(0))
	if !ok {
		return fmt.Errorf("unknown theme: %s (run 'brmk theme list')", fs.Arg(0))
	}
	cfg.Theme = name
	if err := bookmarks.SaveConfig(*configPath, cfg); err != nil {
		return err
	}
	fmt.Printf("theme selected: %s\n", name)
	return nil
}

func runProfileList(args []string) error {
	fs := flag.NewFlagSet("brmk profile list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	cfg, err := bookmarks.EnsureConfig(*configPath)
	if err != nil {
		return err
	}
	for _, profile := range cfg.BrowserProfiles {
		marker := " "
		if strings.EqualFold(profile.Name, cfg.ActiveProfile) {
			marker = "*"
		}
		fmt.Printf("%s %s\t%s\t%s\n", marker, profile.Name, profile.Kind, profile.Browser)
	}
	return nil
}

func runProfileCreate(args []string) error {
	fs := flag.NewFlagSet("brmk profile create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	browser := fs.String("browser", "Google Chrome", "browser app name or alias")
	kind := fs.String("kind", "managed", "default, managed, or existing")
	path := fs.String("path", "", "managed profile path or Firefox profile path")
	directory := fs.String("directory", "", "existing Chromium profile directory")
	selectProfile := fs.Bool("select", true, "select this profile after creation")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true, "browser": true, "kind": true, "path": true, "directory": true, "select": false})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: brmk profile create NAME [--browser BROWSER] [--kind managed|existing|default]")
	}
	cfg, err := bookmarks.EnsureConfig(*configPath)
	if err != nil {
		return err
	}
	name := fs.Arg(0)
	cfg = bookmarks.UpsertBrowserProfile(cfg, bookmarks.BrowserProfile{
		Name:      name,
		Browser:   *browser,
		Kind:      *kind,
		Path:      *path,
		Directory: *directory,
	})
	if *selectProfile {
		cfg, _ = bookmarks.SelectBrowserProfile(cfg, name)
	}
	if err := bookmarks.SaveConfig(*configPath, cfg); err != nil {
		return err
	}
	fmt.Printf("profile created: %s\n", name)
	return nil
}

func runProfileSelect(args []string) error {
	fs := flag.NewFlagSet("brmk profile select", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	normalized, err := intersperseFlags(args, map[string]bool{"config": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: brmk profile select NAME")
	}
	cfg, err := bookmarks.EnsureConfig(*configPath)
	if err != nil {
		return err
	}
	var ok bool
	cfg, ok = bookmarks.SelectBrowserProfile(cfg, fs.Arg(0))
	if !ok {
		return fmt.Errorf("profile not found: %s", fs.Arg(0))
	}
	if err := bookmarks.SaveConfig(*configPath, cfg); err != nil {
		return err
	}
	fmt.Printf("profile selected: %s\n", cfg.ActiveProfile)
	return nil
}

func runProfilePath(args []string) error {
	fs := flag.NewFlagSet("brmk profile path", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	normalized, err := intersperseFlags(args, nil)
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: brmk profile path NAME")
	}
	fmt.Println(bookmarks.DefaultBrowserProfilePath(fs.Arg(0)))
	return nil
}

func runAdd(args []string) error {
	fs := flag.NewFlagSet("brmk add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataPath := fs.String("data", bookmarks.DefaultStorePath(), "bookmark data file")
	configPath := fs.String("config", bookmarks.DefaultConfigPath(), "config file")
	browser := fs.String("browser", "", "browser app name for current-tab capture")
	space := fs.String("space", "", "space")
	title := fs.String("title", "", "bookmark title")
	tagsRaw := fs.String("tags", "", "comma-separated tags")
	noPrompt := fs.Bool("no-prompt", false, "add without prompting")
	yes := fs.Bool("yes", false, "compatibility alias for --no-prompt")
	dryRun := fs.Bool("dry-run", false, "show what would be added without saving")
	verbose := fs.Bool("verbose", false, "show config, data, and source")
	normalized, err := intersperseFlags(args, map[string]bool{
		"browser":   true,
		"config":    true,
		"data":      true,
		"dry-run":   false,
		"no-prompt": false,
		"space":     true,
		"tags":      true,
		"title":     true,
		"verbose":   false,
		"yes":       false,
	})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return errors.New("usage: brmk add [URL] [--browser BROWSER] [--space SPACE|?] [--title TITLE] [--tags TAGS] [--no-prompt] [--dry-run] [--verbose]")
	}

	provided := providedFlags(fs)
	shouldPrompt := !*yes && !*noPrompt
	promptReader := bufio.NewReader(os.Stdin)
	promptValue := func(label, initial string) (string, error) {
		return readPromptWithDefault(promptReader, os.Stderr, label, initial)
	}
	cfg, err := loadAddConfig(*configPath, *dryRun)
	if err != nil {
		return err
	}

	source := "argument"
	captureBrowser := ""
	if fs.NArg() == 0 {
		captureBrowser = bookmarks.AddBrowser(cfg, *browser)
		source = "current tab (" + captureBrowser + ")"
	}
	if *verbose {
		printAddContext(os.Stderr, *dataPath, *configPath, source)
	}

	if provided["space"] && strings.TrimSpace(*space) == "?" {
		if err := printSpaceChoices(*dataPath, os.Stderr, bookmarks.AddSpace(cfg)); err != nil {
			return err
		}
		if !shouldPrompt {
			return errors.New("--space ? requires prompts; pass --space SPACE with --no-prompt")
		}
		delete(provided, "space")
		*space = ""
	}

	url := ""
	fetchedURL := ""
	tabTitle := ""
	if fs.NArg() == 1 {
		url = fs.Arg(0)
	} else {
		tab, err := bookmarks.CurrentTab(captureBrowser)
		if err != nil {
			return err
		}
		url = tab.URL
		fetchedURL = tab.URL
		tabTitle = tab.Title
		if shouldPrompt {
			url, err = promptValue("URL", url)
			if err != nil {
				return err
			}
		}
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return bookmarks.ErrURLRequired
	}

	if !provided["space"] {
		*space = bookmarks.AddSpace(cfg)
		if shouldPrompt {
			*space, err = promptSpace(promptReader, os.Stderr, *dataPath, *space)
			if err != nil {
				return err
			}
		}
	}
	if strings.TrimSpace(*space) == "" {
		*space = bookmarks.DefaultSpace
	}

	if !provided["title"] {
		defaultTitle := tabTitle
		if fetchedURL != "" && strings.TrimSpace(url) != strings.TrimSpace(fetchedURL) {
			defaultTitle = ""
		}
		if strings.TrimSpace(defaultTitle) == "" {
			defaultTitle = bookmarks.DefaultTitle(url)
		}
		*title = defaultTitle
		if shouldPrompt {
			*title, err = promptValue("Title", *title)
			if err != nil {
				return err
			}
		}
	}

	if !provided["tags"] && shouldPrompt {
		*tagsRaw, err = promptValue("Tags", "")
		if err != nil {
			return err
		}
	}

	tags := bookmarks.SplitTags(*tagsRaw)
	if *dryRun {
		printAddDryRun(os.Stdout, *space, *title, url, tags)
		return nil
	}

	node, err := bookmarks.AddBookmark(*dataPath, *space, *title, url, tags)
	if err != nil {
		return err
	}
	fmt.Printf("added: %s\n", node.Title)
	return nil
}

func loadAddConfig(path string, dryRun bool) (*bookmarks.Config, error) {
	if dryRun {
		cfg, err := bookmarks.LoadConfig(path)
		if errors.Is(err, os.ErrNotExist) {
			return bookmarks.DefaultConfig(), nil
		}
		return cfg, err
	}
	return bookmarks.EnsureConfig(path)
}

func printAddContext(writer io.Writer, dataPath, configPath, source string) {
	fmt.Fprintf(writer, "config: %s\n", configPath)
	fmt.Fprintf(writer, "data:   %s\n", dataPath)
	fmt.Fprintf(writer, "source: %s\n", source)
}

func printAddDryRun(writer io.Writer, space, title, rawURL string, tags []string) {
	fmt.Fprintln(writer, "dry-run: bookmark not added")
	fmt.Fprintf(writer, "title:  %s\n", title)
	fmt.Fprintf(writer, "url:    %s\n", rawURL)
	fmt.Fprintf(writer, "space:  %s\n", space)
	if len(tags) == 0 {
		fmt.Fprintln(writer, "tags:   ")
	} else {
		fmt.Fprintf(writer, "tags:   %s\n", strings.Join(tags, ","))
	}
}

func printSpaceChoices(dataPath string, writer io.Writer, defaultSpace string) error {
	store, err := bookmarks.Load(dataPath)
	if err != nil {
		return err
	}
	paths := folderPaths(store)
	if len(paths) == 0 {
		fmt.Fprintln(writer, "Folders: (none)")
		return nil
	}
	fmt.Fprintln(writer, "Folders:")
	for _, path := range paths {
		marker := " "
		if strings.EqualFold(path, defaultSpace) {
			marker = "*"
		}
		fmt.Fprintf(writer, "%s %s\n", marker, path)
	}
	return nil
}

func spaceTitles(store *bookmarks.Store) []string {
	if store == nil || store.Root == nil {
		return nil
	}
	var spaces []string
	for _, child := range store.Root.Children {
		if child.IsFolder() && strings.TrimSpace(child.Title) != "" {
			spaces = append(spaces, child.Title)
		}
	}
	return spaces
}

func providedFlags(fs *flag.FlagSet) map[string]bool {
	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})
	return provided
}

func readPromptWithDefault(reader *bufio.Reader, writer io.Writer, label, initial string) (string, error) {
	fmt.Fprintf(writer, "%-5s [%s]: ", label, initial)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && line == "" {
		return "", fmt.Errorf("input ended while reading %s", strings.ToLower(label))
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return initial, nil
	}
	return line, nil
}

func promptSpace(reader *bufio.Reader, writer io.Writer, dataPath, initial string) (string, error) {
	store, err := bookmarks.Load(dataPath)
	if err != nil {
		return "", err
	}
	paths := folderPaths(store)
	if terminalPromptAvailable() {
		value, accepted, err := tui.PromptWithCompletions("Space", initial, paths)
		if err == nil {
			if !accepted {
				return "", errors.New("space input canceled")
			}
			return value, nil
		}
	}
	return readPromptWithDefault(reader, writer, "Space", initial)
}

func terminalPromptAvailable() bool {
	for _, file := range []*os.File{os.Stdin, os.Stdout} {
		info, err := file.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("brmk export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataPath := fs.String("data", bookmarks.DefaultStorePath(), "bookmark data file")
	normalized, err := intersperseFlags(args, map[string]bool{"data": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: brmk export FILE")
	}
	store, err := bookmarks.Load(*dataPath)
	if err != nil {
		return err
	}
	return os.WriteFile(fs.Arg(0), []byte(bookmarks.ExportMarkdown(store)), 0o644)
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("brmk import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataPath := fs.String("data", bookmarks.DefaultStorePath(), "bookmark data file")
	merge := fs.Bool("merge", false, "merge imported folders into matching existing folders")
	replace := fs.Bool("replace", false, "replace the current store instead of appending")
	normalized, err := intersperseFlags(args, map[string]bool{"data": true, "merge": false, "replace": false})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalized); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: brmk import FILE [--merge|--replace]")
	}
	if *merge && *replace {
		return errors.New("--merge and --replace cannot be used together")
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	imported, err := bookmarks.ImportMarkdown(string(data))
	if err != nil {
		return err
	}
	importedBookmarks := bookmarks.CountBookmarks(imported.Root)
	mode := "append"
	switch {
	case *replace:
		mode = "replace"
	case *merge:
		mode = "merge"
		current, err := bookmarks.Load(*dataPath)
		if err != nil {
			return err
		}
		imported = bookmarks.MergeStore(current, imported)
	default:
		current, err := bookmarks.Load(*dataPath)
		if err != nil {
			return err
		}
		current.Root.Children = append(current.Root.Children, imported.Root.Children...)
		imported = current
	}
	if err := bookmarks.Save(*dataPath, imported); err != nil {
		return err
	}
	fmt.Printf("imported: %d bookmarks (%s)\n", importedBookmarks, mode)
	return nil
}

func printUsage() {
	fmt.Print(strings.TrimSpace(`
Branchmark - a bookmark branch manager

Usage:
  brmk                         open the TUI
  brmk add [URL] [options]     add a bookmark
  brmk config                  create and print the config file path
  brmk completion SHELL        print shell completion script
  brmk export FILE             export bookmarks as Markdown
  brmk import FILE [options]   import bookmarks from Markdown
  brmk profile list            list browser profiles
  brmk profile create NAME     create a managed browser profile
  brmk profile select NAME     select active browser profile
  brmk profile path NAME       print managed profile directory
	brmk theme list               list available color themes
	brmk theme set NAME           select a color theme
  brmk path                    print the data file path
  brmk version                 print the version

Options:
  --data FILE                 use a custom JSON data file
  --config FILE               use a custom config file

Add options:
  --browser BROWSER           browser app name for current-tab capture
  --space PATH                add into this folder path (for example, Work/Docs)
  --space ?                   list folder paths, then prompt for the destination
  --title TITLE               bookmark title
  --tags TAGS                 comma-separated tags
  --no-prompt                 add without prompting
  --dry-run                   show what would be added without saving
  --verbose                   show config, data, and source
  --yes                       compatibility alias for --no-prompt

Import options:
  --merge                     merge folders with matching names at the same level
  --replace                   replace the current store instead of appending
`) + "\n")
}

func intersperseFlags(args []string, valueFlags map[string]bool) ([]string, error) {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if before, _, ok := strings.Cut(name, "="); ok {
			name = before
		}
		needsValue, known := valueFlags[name]
		if !known {
			flags = append(flags, arg)
			continue
		}
		flags = append(flags, arg)
		if needsValue && !strings.Contains(arg, "=") {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag needs an argument: --%s", name)
			}
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...), nil
}
