package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
)

const ansiReset = "\x1b[0m"

var (
	ansiBg       = "\x1b[48;2;29;32;48m"
	ansiSelectBg = "\x1b[48;2;54;59;83m"
	ansiFg       = "\x1b[38;2;205;213;244m"
	ansiMuted    = "\x1b[38;2;127;136;169m"
	ansiBlue     = "\x1b[38;2;137;180;250m"
	ansiCyan     = "\x1b[38;2;116;199;236m"
	ansiGreen    = "\x1b[38;2;166;227;161m"
	ansiBorder   = "\x1b[38;2;137;180;250m"
)

const normalHelpLine = "?:help tab/S-tab p/P j/k [/] folders J/K u R enter a/A e/m/d q"
const treeBodyStartLine = 5
const mouseScrollLines = 3
const helpSideWidth = 38
const mouseWheelButtonFlag = 64
const mouseButtonMask = 3
const mouseWheelUpButton = 0
const mouseWheelDownButton = 1
const mouseWheelUp = mouseWheelButtonFlag | mouseWheelUpButton
const mouseWheelDown = mouseWheelButtonFlag | mouseWheelDownButton

type treeRow struct {
	node          *bookmarks.Node
	parent        *bookmarks.Node
	last          bool
	ancestorsLast []bool
	root          bool
}

type folderChoice struct {
	node  *bookmarks.Node
	depth int
	path  string
}

type app struct {
	dataPath   string
	configPath string
	config     *bookmarks.Config
	store      *bookmarks.Store
	treeRows   []treeRow
	cursor     int
	offset     int
	spaceIndex int
	tabRanges  []tabRange
	status     string
	filter     string
	showHelp   bool
	undo       *undoState

	pendingSpaceClick bool
	pendingSpaceIndex int
}

type undoState struct {
	store        *bookmarks.Store
	spaceID      string
	cursorNodeID string
	action       string
}

type tabRange struct {
	index int
	start int
	end   int
}

func Run(dataPath string, configPath string, cfg *bookmarks.Config) error {
	store, err := bookmarks.Load(dataPath)
	if err != nil {
		return err
	}
	cfg = bookmarks.NormalizeConfig(cfg)
	applyTheme(cfg)
	term, err := enterTerminal()
	if err != nil {
		return err
	}
	defer term.restore()

	a := &app{
		dataPath:   dataPath,
		configPath: configPath,
		config:     cfg,
		store:      store,
		status:     "ready",
	}
	a.selectConfiguredSpace()
	a.rebuildRows()
	a.loop()
	return nil
}

func (a *app) loop() {
	for {
		a.render()
		k := readKey()
		if a.handleKey(k) {
			return
		}
	}
}

func (a *app) handleKey(k key) bool {
	switch k.name {
	case "ctrl-c":
		return true
	case "mouse":
		a.handleMouse(k)
		return false
	case "rune":
		return a.handleRuneKey(k.r)
	default:
		a.handleNamedKey(k)
		return false
	}
}

func (a *app) handleNamedKey(k key) {
	if a.showHelp && k.name == "esc" {
		a.showHelp = false
		a.status = "help hidden"
		return
	}
	switch k.name {
	case "tab":
		a.switchSpace(1)
	case "shift-tab":
		a.switchSpace(-1)
	case "pageup", "ctrl-21":
		a.pageScroll(-1)
	case "pagedown", "ctrl-4":
		a.pageScroll(1)
	case "home":
		a.cursor = 0
		a.offset = 0
	case "end":
		a.cursor = len(a.treeRows) - 1
	case "up":
		a.move(-1)
	case "down":
		a.move(1)
	case "left":
		a.collapseOrParent()
	case "right":
		a.expand()
	case "enter", "space":
		a.openOrToggle()
	}
}

func (a *app) handleRuneKey(r rune) bool {
	switch r {
	case 'q':
		return true
	case 'j':
		a.move(1)
	case 'k':
		a.move(-1)
	case 'h':
		a.collapseOrParent()
	case 'l':
		a.expand()
	case 'o', ' ':
		a.openOrToggle()
	case 'g':
		a.cursor = 0
		a.offset = 0
	case 'G':
		a.cursor = len(a.treeRows) - 1
	case 'J':
		a.reorderSelected(1)
	case 'K':
		a.reorderSelected(-1)
	case '[':
		a.jumpFolder(-1)
	case ']':
		a.jumpFolder(1)
	case 'R':
		a.reload()
	case 'a':
		a.addBookmark()
	case 'A':
		a.addFolder()
	case 'd':
		a.deleteSelected()
	case 'e':
		a.editSelected()
	case 'm':
		a.moveSelectedToFolder()
	case 'p':
		a.cycleProfile()
	case 'P':
		a.selectOrCreateProfile()
	case 'r':
		a.renameSelected()
	case 't':
		a.editTags()
	case 'u':
		a.undoLast()
	case '/':
		a.search()
	case 'c':
		a.filter = ""
		a.status = "search cleared"
		a.rebuildRows()
	case '?':
		a.showHelp = !a.showHelp
		if a.showHelp {
			a.status = "help shown"
		} else {
			a.status = "help hidden"
		}
	}
	return false
}

func (a *app) handleMouse(k key) {
	if delta, ok := mouseWheelDelta(k.mouseButton); ok {
		a.pendingSpaceClick = false
		a.mouseScroll(delta)
		return
	}
	if k.mouseRelease {
		a.releaseSpaceClick(k)
		return
	}
	a.pendingSpaceClick = false
	if k.mouseButton != 0 {
		return
	}
	if k.mouseY == treeBodyStartLine-1 {
		if index, ok := a.spaceIndexAtX(k.mouseX); ok {
			a.pendingSpaceClick = true
			a.pendingSpaceIndex = index
		}
		return
	}
	cols, _ := terminalSize()
	if a.usesHelpSidePane(cols) && k.mouseX > cols-helpSideWidth {
		return
	}
	index := a.rowIndexAtMouseY(k.mouseY)
	if index < 0 {
		return
	}
	if index == a.cursor {
		a.openOrToggle()
		return
	}
	a.cursor = index
	a.status = "selected: " + a.treeRows[index].node.Title
}

func (a *app) releaseSpaceClick(k key) {
	if !a.pendingSpaceClick {
		return
	}
	defer func() { a.pendingSpaceClick = false }()
	if k.mouseY != treeBodyStartLine-1 {
		return
	}
	index, ok := a.spaceIndexAtX(k.mouseX)
	if !ok || index != a.pendingSpaceIndex {
		return
	}
	a.switchSpaceTo(index)
}

func mouseWheelDelta(button int) (int, bool) {
	if button&mouseWheelButtonFlag == 0 {
		return 0, false
	}
	switch button & mouseButtonMask {
	case mouseWheelUpButton:
		return -mouseScrollLines, true
	case mouseWheelDownButton:
		return mouseScrollLines, true
	default:
		return 0, true
	}
}

func (a *app) mouseScroll(delta int) {
	cols, rows := terminalSize()
	a.scrollRows(delta, a.visibleBodyHeight(rows, cols))
}

func (a *app) pageScroll(direction int) {
	cols, rows := terminalSize()
	bodyHeight := a.visibleBodyHeight(rows, cols)
	step := max(1, bodyHeight-1)
	a.scrollRows(direction*step, bodyHeight)
}

func (a *app) scrollRows(delta int, visibleHeight int) {
	if len(a.treeRows) == 0 {
		a.cursor = 0
		a.offset = 0
		return
	}
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	maxOffset := max(0, len(a.treeRows)-visibleHeight)
	a.offset += delta
	if a.offset < 0 {
		a.offset = 0
	}
	if a.offset > maxOffset {
		a.offset = maxOffset
	}
	if a.cursor < a.offset {
		a.cursor = a.offset
	}
	if a.cursor >= a.offset+visibleHeight {
		a.cursor = min(len(a.treeRows)-1, a.offset+visibleHeight-1)
	}
}

func (a *app) rowIndexAtMouseY(y int) int {
	cols, rows := terminalSize()
	if y >= treeBodyStartLine+a.visibleBodyHeight(rows, cols) {
		return -1
	}
	visibleLine := y - treeBodyStartLine
	if visibleLine < 0 {
		return -1
	}
	index := a.offset + visibleLine
	if index < 0 || index >= len(a.treeRows) {
		return -1
	}
	return index
}

func (a *app) spaces() []*bookmarks.Node {
	if a == nil || a.store == nil || a.store.Root == nil {
		return nil
	}
	spaces := make([]*bookmarks.Node, 0, len(a.store.Root.Children))
	for _, child := range a.store.Root.Children {
		if child.IsFolder() {
			spaces = append(spaces, child)
		}
	}
	if len(spaces) == 0 {
		spaces = append(spaces, a.store.Root)
	}
	return spaces
}

func (a *app) activeSpace() *bookmarks.Node {
	spaces := a.spaces()
	if len(spaces) == 0 {
		return nil
	}
	if a.spaceIndex < 0 {
		a.spaceIndex = 0
	}
	if a.spaceIndex >= len(spaces) {
		a.spaceIndex = len(spaces) - 1
	}
	return spaces[a.spaceIndex]
}

func (a *app) switchSpace(delta int) {
	spaces := a.spaces()
	if len(spaces) == 0 {
		return
	}
	a.spaceIndex = (a.spaceIndex + delta) % len(spaces)
	if a.spaceIndex < 0 {
		a.spaceIndex += len(spaces)
	}
	a.cursor = 0
	a.offset = 0
	a.rebuildRows()
	if space := a.activeSpace(); space != nil {
		a.saveDefaultSpace("space: " + space.Title)
	}
}

func (a *app) switchSpaceTo(index int) bool {
	spaces := a.spaces()
	if index < 0 || index >= len(spaces) {
		return false
	}
	a.spaceIndex = index
	a.cursor = 0
	a.offset = 0
	a.rebuildRows()
	if space := a.activeSpace(); space != nil {
		a.saveDefaultSpace("space: " + space.Title)
	}
	return true
}

func (a *app) spaceIndexAtX(x int) (int, bool) {
	for _, tab := range a.tabRanges {
		if x >= tab.start && x <= tab.end {
			return tab.index, true
		}
	}
	return 0, false
}

func (a *app) selectSpaceByID(id string) bool {
	for i, space := range a.spaces() {
		if space.ID == id {
			a.spaceIndex = i
			return true
		}
	}
	return false
}

func (a *app) selectSpaceByTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	for i, space := range a.spaces() {
		if strings.EqualFold(space.Title, title) {
			a.spaceIndex = i
			return true
		}
	}
	return false
}

func (a *app) selectConfiguredSpace() {
	cfg := bookmarks.NormalizeConfig(a.config)
	defaultSpace := strings.TrimSpace(cfg.DefaultSpace)
	if defaultSpace == "" {
		return
	}
	for i, space := range a.spaces() {
		if space.ID != "root" && strings.EqualFold(space.Title, defaultSpace) {
			a.spaceIndex = i
			return
		}
	}
}

func (a *app) saveDefaultSpace(message string) {
	space := a.activeSpace()
	if space == nil || space.ID == "root" || strings.TrimSpace(space.Title) == "" {
		a.status = message
		return
	}
	cfg := bookmarks.NormalizeConfig(a.config)
	if cfg.DefaultSpace == space.Title {
		a.config = cfg
		a.status = message
		return
	}
	cfg.DefaultSpace = space.Title
	a.config = cfg
	if strings.TrimSpace(a.configPath) == "" {
		a.status = message
		return
	}
	if err := bookmarks.SaveConfig(a.configPath, a.config); err != nil {
		a.status = err.Error()
		return
	}
	a.status = message
}

func (a *app) activeProfile() bookmarks.BrowserProfile {
	cfg := bookmarks.NormalizeConfig(a.config)
	profile, ok := bookmarks.FindBrowserProfile(cfg.BrowserProfiles, cfg.ActiveProfile)
	if !ok {
		profile, _ = bookmarks.FindBrowserProfile(cfg.BrowserProfiles, "default")
	}
	return profile
}

func (a *app) activeProfileName() string {
	return a.activeProfile().Name
}

func (a *app) cycleProfile() {
	cfg := bookmarks.NormalizeConfig(a.config)
	if len(cfg.BrowserProfiles) == 0 {
		cfg = bookmarks.DefaultConfig()
	}
	index := 0
	for i, profile := range cfg.BrowserProfiles {
		if strings.EqualFold(profile.Name, cfg.ActiveProfile) {
			index = i
			break
		}
	}
	next := cfg.BrowserProfiles[(index+1)%len(cfg.BrowserProfiles)].Name
	cfg.ActiveProfile = next
	a.config = cfg
	a.saveConfig("profile: " + next)
}

func (a *app) selectOrCreateProfile() {
	current := a.activeProfileName()
	name, ok := a.promptWithCompletions("Profile", current, a.profileNames())
	if !ok || strings.TrimSpace(name) == "" {
		a.status = "profile selection canceled"
		return
	}
	name = strings.TrimSpace(name)
	cfg := bookmarks.NormalizeConfig(a.config)
	if _, ok := bookmarks.FindBrowserProfile(cfg.BrowserProfiles, name); !ok {
		cfg = bookmarks.UpsertBrowserProfile(cfg, bookmarks.BrowserProfile{
			Name:    name,
			Browser: "Google Chrome",
			Kind:    "managed",
		})
	}
	cfg, _ = bookmarks.SelectBrowserProfile(cfg, name)
	a.config = cfg
	a.saveConfig("profile: " + name)
}

func (a *app) profileNames() []string {
	cfg := bookmarks.NormalizeConfig(a.config)
	names := make([]string, 0, len(cfg.BrowserProfiles))
	for _, profile := range cfg.BrowserProfiles {
		names = append(names, profile.Name)
	}
	return names
}

func (a *app) saveConfig(message string) {
	if err := bookmarks.SaveConfig(a.configPath, a.config); err != nil {
		a.status = err.Error()
		return
	}
	a.status = message
}

func (a *app) rebuildRows() {
	a.treeRows = nil
	root := a.activeSpace()
	if root == nil {
		a.cursor = 0
		a.offset = 0
		return
	}
	if a.filter == "" || matchesTree(root, a.filter) {
		a.treeRows = append(a.treeRows, treeRow{node: root, root: true, last: true, ancestorsLast: nil})
	}
	if len(a.treeRows) == 0 {
		a.cursor = 0
		a.offset = 0
		return
	}
	if a.filter == "" {
		if root.Expanded {
			a.rebuildVisibleRows(root.Children, root, nil)
		}
	} else {
		a.rebuildFilteredRows(root.Children, root, nil)
	}
	if a.cursor >= len(a.treeRows) {
		a.cursor = len(a.treeRows) - 1
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
}

func (a *app) rebuildVisibleRows(nodes []*bookmarks.Node, parent *bookmarks.Node, ancestorsLast []bool) {
	for i, n := range nodes {
		last := i == len(nodes)-1
		a.treeRows = append(a.treeRows, treeRow{
			node:          n,
			parent:        parent,
			last:          last,
			ancestorsLast: append([]bool(nil), ancestorsLast...),
		})
		if n.IsFolder() && n.Expanded {
			a.rebuildVisibleRows(n.Children, n, append(ancestorsLast, last))
		}
	}
}

func (a *app) rebuildFilteredRows(nodes []*bookmarks.Node, parent *bookmarks.Node, ancestorsLast []bool) {
	visible := make([]int, 0, len(nodes))
	for i, n := range nodes {
		if matchesTree(n, a.filter) {
			visible = append(visible, i)
		}
	}
	for visibleIndex, nodeIndex := range visible {
		n := nodes[nodeIndex]
		last := visibleIndex == len(visible)-1
		a.treeRows = append(a.treeRows, treeRow{
			node:          n,
			parent:        parent,
			last:          last,
			ancestorsLast: append([]bool(nil), ancestorsLast...),
		})
		if n.IsFolder() {
			a.rebuildFilteredRows(n.Children, n, append(ancestorsLast, last))
		}
	}
}

func matchesTree(n *bookmarks.Node, filter string) bool {
	if matchesFilter(n, filter) {
		return true
	}
	if !n.IsFolder() {
		return false
	}
	for _, child := range n.Children {
		if matchesTree(child, filter) {
			return true
		}
	}
	return false
}

func matchesFilter(n *bookmarks.Node, filter string) bool {
	filter = strings.ToLower(filter)
	values := []string{n.Title, n.URL, strings.Join(n.Tags, " ")}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), filter) {
			return true
		}
	}
	return false
}

func (a *app) selected() *treeRow {
	if len(a.treeRows) == 0 || a.cursor < 0 || a.cursor >= len(a.treeRows) {
		return nil
	}
	return &a.treeRows[a.cursor]
}

func (a *app) selectedParentForAdd() *bookmarks.Node {
	selected := a.selected()
	if selected == nil {
		if space := a.activeSpace(); space != nil {
			return space
		}
		return a.store.Root
	}
	if selected.node.IsFolder() {
		return selected.node
	}
	if selected.parent != nil {
		return selected.parent
	}
	return a.store.Root
}

func (a *app) move(delta int) {
	if len(a.treeRows) == 0 {
		return
	}
	a.cursor += delta
	if a.cursor < 0 {
		a.cursor = 0
	}
	if a.cursor >= len(a.treeRows) {
		a.cursor = len(a.treeRows) - 1
	}
}

func (a *app) jumpFolder(direction int) {
	if len(a.treeRows) == 0 || direction == 0 {
		return
	}
	start := min(max(a.cursor, 0), len(a.treeRows)-1)
	for index := start + direction; index >= 0 && index < len(a.treeRows); index += direction {
		if a.treeRows[index].node.IsFolder() {
			a.cursor = index
			a.status = "folder: " + a.treeRows[index].node.Title
			return
		}
	}
	if direction < 0 {
		a.status = "first folder"
	} else {
		a.status = "last folder"
	}
}

func (a *app) expand() {
	selected := a.selected()
	if selected == nil || !selected.node.IsFolder() {
		return
	}
	selected.node.Expanded = true
	a.status = "expanded"
	a.rebuildRows()
}

func (a *app) collapseOrParent() {
	selected := a.selected()
	if selected == nil {
		return
	}
	if selected.node.IsFolder() && selected.node.Expanded {
		selected.node.Expanded = false
		a.status = "collapsed"
		a.rebuildRows()
		return
	}
	if selected.parent == nil || selected.parent.ID == "root" {
		return
	}
	for i, item := range a.treeRows {
		if item.node.ID == selected.parent.ID {
			a.cursor = i
			return
		}
	}
}

func (a *app) openOrToggle() {
	selected := a.selected()
	if selected == nil {
		return
	}
	if selected.node.IsFolder() {
		selected.node.Expanded = !selected.node.Expanded
		a.status = "toggled"
		a.rebuildRows()
		return
	}
	profile := a.activeProfile()
	if err := bookmarks.OpenURLWithProfile(selected.node.URL, profile); err != nil {
		a.status = err.Error()
		return
	}
	a.status = "opened: " + selected.node.Title + "  profile=" + profile.Name
}

func (a *app) addBookmark() {
	parent := a.selectedParentForAdd()
	url, ok := a.prompt("URL", "")
	if !ok || strings.TrimSpace(url) == "" {
		a.status = "add canceled"
		return
	}
	a.status = "fetching page metadata..."
	a.render()
	titleDefault := bookmarks.DefaultTitle(url)
	title, ok := a.prompt("Title", titleDefault)
	if !ok {
		a.status = "add canceled"
		return
	}
	tagsRaw, ok := a.prompt("Tags comma/space separated", "")
	if !ok {
		a.status = "add canceled"
		return
	}
	a.clearUndo()
	node := bookmarks.NewBookmark(title, url, bookmarks.SplitTags(tagsRaw))
	parent.Children = append(parent.Children, node)
	parent.Expanded = true
	parent.Touch()
	a.save("bookmark added")
	a.rebuildRows()
}

func (a *app) addFolder() {
	parent := a.selectedParentForAdd()
	title, ok := a.prompt("Folder", "")
	if !ok || strings.TrimSpace(title) == "" {
		a.status = "folder add canceled"
		return
	}
	a.clearUndo()
	parent.Children = append(parent.Children, bookmarks.NewFolder(title))
	parent.Expanded = true
	parent.Touch()
	a.save("folder added")
	a.rebuildRows()
}

func (a *app) editSelected() {
	selected := a.selected()
	if selected == nil {
		return
	}
	title, ok := a.prompt("Title", selected.node.Title)
	if !ok {
		a.status = "edit canceled"
		return
	}
	newTitle := strings.TrimSpace(title)
	newURL := selected.node.URL
	if selected.node.IsBookmark() {
		url, ok := a.prompt("URL", selected.node.URL)
		if !ok {
			a.status = "edit canceled"
			return
		}
		newURL = strings.TrimSpace(url)
	}
	tagsRaw, ok := a.prompt("Tags comma/space separated", strings.Join(selected.node.Tags, ","))
	if !ok {
		a.status = "edit canceled"
		return
	}
	a.clearUndo()
	selected.node.Title = newTitle
	selected.node.URL = newURL
	selected.node.Tags = bookmarks.SplitTags(tagsRaw)
	selected.node.Touch()
	a.save("edited")
	a.rebuildRows()
}

func (a *app) renameSelected() {
	selected := a.selected()
	if selected == nil {
		return
	}
	title, ok := a.prompt("Title", selected.node.Title)
	if !ok || strings.TrimSpace(title) == "" {
		a.status = "rename canceled"
		return
	}
	a.clearUndo()
	selected.node.Title = strings.TrimSpace(title)
	selected.node.Touch()
	a.save("renamed")
	a.rebuildRows()
}

func (a *app) editTags() {
	selected := a.selected()
	if selected == nil {
		return
	}
	tagsRaw, ok := a.prompt("Tags comma/space separated", strings.Join(selected.node.Tags, ","))
	if !ok {
		a.status = "tag edit canceled"
		return
	}
	a.clearUndo()
	selected.node.Tags = bookmarks.SplitTags(tagsRaw)
	selected.node.Touch()
	a.save("tags updated")
}

func (a *app) moveSelectedToFolder() {
	selected := a.selected()
	if selected == nil {
		return
	}
	if selected.node.ID == "root" {
		a.status = "root folder cannot be moved"
		return
	}
	movedID := selected.node.ID
	dest, ok := a.selectMoveDestination(selected.node, selected.parent)
	if !ok {
		a.status = "move canceled"
		return
	}
	previousUndo := a.undo
	a.captureUndo("move", movedID)
	message, moved := a.moveNodeToFolder(movedID, dest.ID)
	if !moved {
		a.undo = previousUndo
		a.status = message
		return
	}
	a.save(message)
	a.rebuildRows()
	a.focusNode(movedID)
}

func (a *app) selectMoveDestination(moving *bookmarks.Node, currentParent *bookmarks.Node) (*bookmarks.Node, bool) {
	allChoices := buildFolderChoices(a.store, moving)
	if len(allChoices) == 0 {
		return nil, false
	}
	cursor := 0
	if currentParent != nil {
		for i, choice := range allChoices {
			if choice.node.ID == currentParent.ID {
				cursor = i
				break
			}
		}
	}
	offset := 0
	query := ""
	for {
		choices := filterFolderChoices(allChoices, query)
		cols, rows := terminalSize()
		bodyHeight := folderPickerBodyHeight(rows)
		cursor, offset = clampCursorOffset(cursor, offset, len(choices), bodyHeight)
		a.renderMoveDestinationPicker(cols, rows, moving, choices, cursor, offset, query)
		k := readKey()
		switch {
		case k.name == "esc" || k.name == "ctrl-c":
			return nil, false
		case k.name == "enter" && len(choices) > 0:
			return choices[cursor].node, true
		case k.name == "tab" && len(choices) > 0:
			if strings.TrimSpace(query) != "" {
				selectedID := choices[cursor].node.ID
				query = choices[cursor].path
				cursor = indexFolderChoice(filterFolderChoices(allChoices, query), selectedID)
				offset = 0
				continue
			}
			cursor++
		case k.name == "shift-tab":
			cursor--
		case k.name == "up" || (k.name == "rune" && k.r == 'k'):
			cursor--
		case k.name == "down" || (k.name == "rune" && k.r == 'j'):
			cursor++
		case k.name == "pageup":
			cursor -= max(1, bodyHeight-1)
		case k.name == "pagedown" || k.name == "ctrl-4":
			cursor += max(1, bodyHeight-1)
		case k.name == "home":
			cursor = 0
		case k.name == "end":
			cursor = len(choices) - 1
		case k.name == "backspace":
			if query != "" {
				runes := []rune(query)
				query = string(runes[:len(runes)-1])
				cursor = 0
				offset = 0
			}
		case k.name == "ctrl-21":
			query = ""
			cursor = 0
			offset = 0
		case k.name == "space":
			query += " "
			cursor = 0
			offset = 0
		case k.name == "rune":
			query += string(k.r)
			cursor = 0
			offset = 0
		}
	}
}

func (a *app) renderMoveDestinationPicker(cols int, rows int, moving *bookmarks.Node, choices []folderChoice, cursor int, offset int, query string) {
	if rows < 8 {
		rows = 8
	}
	bodyHeight := folderPickerBodyHeight(rows)
	end := min(len(choices), offset+bodyHeight)
	title := "Move"
	input := "Move " + moving.Title
	if strings.TrimSpace(query) != "" {
		input += "  Path: " + query
	}
	counter := "0/0"
	if len(choices) > 0 {
		counter = fmt.Sprintf("%d/%d", cursor+1, len(choices))
	}

	var b strings.Builder
	b.WriteString("\x1b[?25l\x1b[H\x1b[2J")
	b.WriteString(styleLine(boxTop(cols, title), ansiBorder, cols))
	b.WriteString("\n")
	b.WriteString(renderInputLine(cols, input, counter))
	b.WriteString("\n")
	b.WriteString(styleLine(boxBottom(cols), ansiBorder, cols))
	b.WriteString("\n")
	for line := 0; line < bodyHeight; line++ {
		index := offset + line
		if index < end {
			b.WriteString(renderFolderChoice(choices[index], cols, index == cursor))
		} else if len(choices) == 0 && line == 0 {
			b.WriteString(styleLine("  No matching folders.", ansiMuted, cols))
		} else {
			b.WriteString(styleLine("", ansiFg, cols))
		}
		b.WriteString("\n")
	}
	detail := ""
	if cursor >= 0 && cursor < len(choices) {
		detail = "to: " + choices[cursor].path
	}
	b.WriteString(styleLine(detail, ansiMuted, cols))
	b.WriteString("\n")
	b.WriteString(styleLine("type filter  tab complete/next  enter move  esc cancel  j/k move", ansiMuted, cols))
	fmt.Print(toTerminalLines(b.String()))
}

func folderPickerBodyHeight(rows int) int {
	if rows < 8 {
		rows = 8
	}
	return max(1, rows-5)
}

func renderFolderChoice(choice folderChoice, cols int, selected bool) string {
	title := choice.node.Title
	if choice.node.ID == "root" {
		title = "Bookmarks"
	}
	line := "  " + strings.Repeat("  ", choice.depth) + folderIcon + " " + title
	line = truncateWidth(line, cols)
	line = padRight(line, cols)
	style := ansiBg + ansiBlue
	if selected {
		style = ansiSelectBg + ansiBlue
	}
	return style + line + ansiReset
}

func clampCursorOffset(cursor int, offset int, total int, visibleHeight int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+visibleHeight {
		offset = cursor - visibleHeight + 1
	}
	maxOffset := max(0, total-visibleHeight)
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	return cursor, offset
}

func buildFolderChoices(store *bookmarks.Store, moving *bookmarks.Node) []folderChoice {
	if store == nil || store.Root == nil {
		return nil
	}
	var choices []folderChoice
	var walk func(node *bookmarks.Node, depth int, parts []string)
	walk = func(node *bookmarks.Node, depth int, parts []string) {
		if node == nil || !node.IsFolder() {
			return
		}
		if moving != nil && moving.IsFolder() && nodeContains(moving, node.ID) {
			return
		}
		path := "Bookmarks"
		if len(parts) > 0 {
			path = strings.Join(parts, "/")
		}
		choices = append(choices, folderChoice{node: node, depth: depth, path: path})
		for _, child := range node.Children {
			if !child.IsFolder() {
				continue
			}
			childParts := parts
			if child.ID != "root" {
				childParts = append(append([]string(nil), parts...), child.Title)
			}
			walk(child, depth+1, childParts)
		}
	}
	walk(store.Root, 0, nil)
	return choices
}

func filterFolderChoices(choices []folderChoice, query string) []folderChoice {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return choices
	}
	filtered := make([]folderChoice, 0, len(choices))
	for _, choice := range choices {
		title := strings.ToLower(choice.node.Title)
		path := strings.ToLower(choice.path)
		if strings.Contains(title, query) || strings.Contains(path, query) {
			filtered = append(filtered, choice)
		}
	}
	return filtered
}

func indexFolderChoice(choices []folderChoice, id string) int {
	for i, choice := range choices {
		if choice.node.ID == id {
			return i
		}
	}
	return 0
}

func (a *app) moveNodeToFolder(nodeID, destID string) (string, bool) {
	node, parent, index := bookmarks.Find(a.store, nodeID)
	if node == nil || parent == nil || index < 0 {
		return "selected item was not found", false
	}
	if node.ID == "root" {
		return "root folder cannot be moved", false
	}
	dest, _, _ := bookmarks.Find(a.store, destID)
	if dest == nil || !dest.IsFolder() {
		return "destination folder was not found", false
	}
	if node.IsFolder() && nodeContains(node, dest.ID) {
		return "folder cannot be moved into itself", false
	}
	if dest.ID == parent.ID {
		return "already in: " + displayFolderPath(a.store, dest), false
	}
	parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
	parent.Touch()
	dest.Children = append(dest.Children, node)
	dest.Expanded = true
	dest.Touch()
	node.Touch()
	return "moved to: " + displayFolderPath(a.store, dest), true
}

func nodeContains(root *bookmarks.Node, id string) bool {
	if root == nil {
		return false
	}
	if root.ID == id {
		return true
	}
	for _, child := range root.Children {
		if nodeContains(child, id) {
			return true
		}
	}
	return false
}

func (a *app) reorderSelected(delta int) {
	selected := a.selected()
	if selected == nil {
		return
	}
	if strings.TrimSpace(a.filter) != "" {
		a.status = "clear search before reordering"
		return
	}
	activeSpaceID := ""
	if space := a.activeSpace(); space != nil {
		activeSpaceID = space.ID
	}
	movedID := selected.node.ID
	previousUndo := a.undo
	a.captureUndo("reorder", movedID)
	message, moved := a.reorderNode(movedID, delta)
	if !moved {
		a.undo = previousUndo
		a.status = message
		return
	}
	if activeSpaceID != "" {
		a.selectSpaceByID(activeSpaceID)
	}
	a.save(message)
	a.rebuildRows()
	a.focusNode(movedID)
}

func (a *app) reorderNode(nodeID string, delta int) (string, bool) {
	node, parent, index := bookmarks.Find(a.store, nodeID)
	if node == nil || parent == nil || index < 0 {
		return "selected item was not found", false
	}
	if node.ID == "root" {
		return "root folder cannot be reordered", false
	}
	if delta < 0 {
		if index == 0 {
			return "already at top: " + node.Title, false
		}
		otherIndex := index - 1
		parent.Children[index], parent.Children[otherIndex] = parent.Children[otherIndex], parent.Children[index]
		parent.Touch()
		node.Touch()
		parent.Children[index].Touch()
		return "moved up: " + node.Title, true
	}
	if delta > 0 {
		if index >= len(parent.Children)-1 {
			return "already at bottom: " + node.Title, false
		}
		otherIndex := index + 1
		parent.Children[index], parent.Children[otherIndex] = parent.Children[otherIndex], parent.Children[index]
		parent.Touch()
		node.Touch()
		parent.Children[index].Touch()
		return "moved down: " + node.Title, true
	}
	return "position unchanged: " + node.Title, false
}

func (a *app) folderPath(folder *bookmarks.Node) string {
	if a == nil || a.store == nil || a.store.Root == nil || folder == nil || folder.ID == a.store.Root.ID {
		return ""
	}
	parts, ok := nodePath(a.store.Root, folder.ID)
	if !ok {
		return ""
	}
	return strings.Join(parts, "/")
}

func displayFolderPath(store *bookmarks.Store, folder *bookmarks.Node) string {
	if store == nil || store.Root == nil || folder == nil || folder.ID == store.Root.ID {
		return "Bookmarks"
	}
	parts, ok := nodePath(store.Root, folder.ID)
	if !ok || len(parts) == 0 {
		return folder.Title
	}
	return strings.Join(parts, "/")
}

func nodePath(root *bookmarks.Node, id string) ([]string, bool) {
	if root == nil {
		return nil, false
	}
	if root.ID == id {
		return nil, true
	}
	for _, child := range root.Children {
		if child.ID == id {
			return []string{child.Title}, true
		}
		if child.IsFolder() {
			if parts, ok := nodePath(child, id); ok {
				return append([]string{child.Title}, parts...), true
			}
		}
	}
	return nil, false
}

func (a *app) focusNode(id string) {
	for i, item := range a.treeRows {
		if item.node.ID == id {
			a.cursor = i
			if a.cursor < a.offset {
				a.offset = a.cursor
			}
			return
		}
	}
}

func (a *app) deleteSelected() {
	selected := a.selected()
	if selected == nil {
		return
	}
	if selected.node.ID == "root" {
		a.status = "root folder cannot be deleted"
		return
	}
	answer, ok := a.prompt("Delete "+selected.node.Title+"? type y", "")
	if !ok || answer != "y" {
		a.status = "delete canceled"
		return
	}
	previousUndo := a.undo
	a.captureUndo("delete", selected.node.ID)
	if _, ok := bookmarks.Remove(a.store, selected.node.ID); ok {
		a.save("deleted")
		a.rebuildRows()
		return
	}
	a.undo = previousUndo
}

func (a *app) search() {
	query, ok := a.prompt("Search", a.filter)
	if !ok {
		a.status = "search canceled"
		return
	}
	a.filter = strings.TrimSpace(query)
	if a.filter == "" {
		a.status = "search cleared"
	} else {
		a.status = "search: " + a.filter
	}
	a.cursor = 0
	a.offset = 0
	a.rebuildRows()
}

func (a *app) save(message string) {
	if err := bookmarks.Save(a.dataPath, a.store); err != nil {
		a.status = err.Error()
		return
	}
	a.status = message
}

func (a *app) captureUndo(action string, cursorNodeID string) {
	if a == nil || a.store == nil {
		return
	}
	if strings.TrimSpace(cursorNodeID) == "" {
		if selected := a.selected(); selected != nil {
			cursorNodeID = selected.node.ID
		}
	}
	spaceID := ""
	if space := a.activeSpace(); space != nil {
		spaceID = space.ID
	}
	a.undo = &undoState{
		store:        cloneStore(a.store),
		spaceID:      spaceID,
		cursorNodeID: cursorNodeID,
		action:       action,
	}
}

func (a *app) clearUndo() {
	if a != nil {
		a.undo = nil
	}
}

func (a *app) undoLast() {
	if a == nil || a.undo == nil || a.undo.store == nil {
		a.status = "nothing to undo"
		return
	}
	undo := a.undo
	a.store = cloneStore(undo.store)
	a.undo = nil
	if undo.spaceID != "" {
		a.selectSpaceByID(undo.spaceID)
	}
	a.rebuildRows()
	if undo.cursorNodeID != "" {
		a.focusNode(undo.cursorNodeID)
	}
	if err := bookmarks.Save(a.dataPath, a.store); err != nil {
		a.status = err.Error()
		return
	}
	a.status = "undone: " + undo.action
}

func (a *app) reload() {
	if a == nil {
		return
	}
	selectedID := ""
	if selected := a.selected(); selected != nil {
		selectedID = selected.node.ID
	}
	activeID := ""
	activeTitle := ""
	if space := a.activeSpace(); space != nil {
		activeID = space.ID
		activeTitle = space.Title
	}
	store, err := bookmarks.Load(a.dataPath)
	if err != nil {
		a.status = err.Error()
		return
	}
	a.store = store
	a.clearUndo()
	if activeID != "" && !a.selectSpaceByID(activeID) && activeTitle != "" {
		a.selectSpaceByTitle(activeTitle)
	}
	a.rebuildRows()
	if selectedID != "" {
		a.focusNode(selectedID)
	}
	a.status = "reloaded"
}

func cloneStore(store *bookmarks.Store) *bookmarks.Store {
	if store == nil {
		return nil
	}
	return &bookmarks.Store{
		Version: store.Version,
		Root:    cloneNode(store.Root),
	}
}

func cloneNode(node *bookmarks.Node) *bookmarks.Node {
	if node == nil {
		return nil
	}
	cloned := *node
	if node.Tags != nil {
		cloned.Tags = append([]string(nil), node.Tags...)
	}
	if node.Children != nil {
		cloned.Children = make([]*bookmarks.Node, 0, len(node.Children))
		for _, child := range node.Children {
			cloned.Children = append(cloned.Children, cloneNode(child))
		}
	}
	return &cloned
}

func (a *app) prompt(label, initial string) (string, bool) {
	return a.promptWithCompletions(label, initial, nil)
}

func (a *app) promptWithCompletions(label, initial string, completions []string) (string, bool) {
	editor := newPromptEditor(initial)
	completions = normalizeCompletions(completions)
	for {
		a.renderPrompt(label, editor, len(completions) > 0)
		k := readKey()
		switch k.name {
		case "enter":
			return editor.value(), true
		case "esc", "ctrl-c":
			return "", false
		case "tab":
			editor.complete(completions, 1)
		case "shift-tab":
			editor.complete(completions, -1)
		default:
			editor.apply(k)
		}
	}
}

// PromptWithCompletions presents a standalone, terminal-native prompt. Tab and
// Shift-Tab cycle through candidates matching the current prefix.
func PromptWithCompletions(label, initial string, completions []string) (string, bool, error) {
	term, err := enterTerminal()
	if err != nil {
		return "", false, err
	}
	defer term.restore()

	editor := newPromptEditor(initial)
	completions = normalizeCompletions(completions)
	for {
		renderStandalonePrompt(label, editor, completions)
		k := readKey()
		switch k.name {
		case "enter":
			return editor.value(), true, nil
		case "esc", "ctrl-c":
			return "", false, nil
		case "tab":
			editor.complete(completions, 1)
		case "shift-tab":
			editor.complete(completions, -1)
		case "error":
			return "", false, fmt.Errorf("read prompt input")
		default:
			editor.apply(k)
		}
	}
}

func renderStandalonePrompt(label string, editor *promptEditor, completions []string) {
	cols, rows := terminalSize()
	if rows < 8 {
		rows = 8
	}
	input := label + ": " + editor.value()
	matches := matchingCompletions(completions, editor.value())
	maxSuggestions := min(5, max(1, rows-5))

	var b strings.Builder
	b.WriteString("\x1b[?25l\x1b[H\x1b[2J")
	b.WriteString(styleLine(boxTop(cols, "brmk add"), ansiBorder, cols))
	b.WriteString("\n")
	b.WriteString(renderInputLine(cols, input, fmt.Sprintf("%d", len(matches))))
	b.WriteString("\n")
	b.WriteString(styleLine(boxBottom(cols), ansiBorder, cols))
	b.WriteString("\n")
	if len(completions) == 0 {
		b.WriteString(styleLine("  No existing folders. Enter a new path to create it.", ansiMuted, cols))
		b.WriteString("\n")
	} else if len(matches) == 0 {
		b.WriteString(styleLine("  No matching folders. Enter a new path to create it.", ansiMuted, cols))
		b.WriteString("\n")
	} else {
		for i := 0; i < min(len(matches), maxSuggestions); i++ {
			b.WriteString(styleLine("  "+matches[i], ansiFg, cols))
			b.WriteString("\n")
		}
		if len(matches) > maxSuggestions {
			b.WriteString(styleLine(fmt.Sprintf("  … %d more", len(matches)-maxSuggestions), ansiMuted, cols))
			b.WriteString("\n")
		}
	}
	b.WriteString(styleLine("tab/S-tab complete  enter select or create  esc cancel", ansiMuted, cols))
	fmt.Print(toTerminalLines(b.String()))

	cursorText := label + ": " + string(editor.input[:editor.cursor])
	cursorCol := min(cols-1, 3+displayWidth(cursorText))
	fmt.Printf("\x1b[2;%dH\x1b[?25h", cursorCol)
}

type promptEditor struct {
	input             []rune
	cursor            int
	allSelected       bool
	completionBase    string
	completionMatches []string
	completionIndex   int
}

func newPromptEditor(initial string) *promptEditor {
	input := []rune(initial)
	return &promptEditor{input: input, cursor: len(input)}
}

func (e *promptEditor) value() string {
	if e == nil {
		return ""
	}
	return string(e.input)
}

func (e *promptEditor) apply(k key) {
	if e == nil {
		return
	}
	e.resetCompletion()
	switch k.name {
	case "left":
		if e.allSelected {
			e.allSelected = false
			e.cursor = 0
			return
		}
		if e.cursor > 0 {
			e.cursor--
		}
	case "right":
		if e.allSelected {
			e.allSelected = false
			e.cursor = len(e.input)
			return
		}
		if e.cursor < len(e.input) {
			e.cursor++
		}
	case "home", "ctrl-2":
		e.allSelected = false
		e.cursor = 0
	case "end", "ctrl-5":
		e.allSelected = false
		e.cursor = len(e.input)
	case "ctrl-1":
		e.allSelected = len(e.input) > 0
		e.cursor = len(e.input)
	case "ctrl-21":
		e.clear()
	case "ctrl-11":
		if e.allSelected {
			e.clear()
			return
		}
		e.input = e.input[:e.cursor]
	case "backspace":
		if e.allSelected {
			e.clear()
			return
		}
		if e.cursor > 0 {
			e.input = append(e.input[:e.cursor-1], e.input[e.cursor:]...)
			e.cursor--
		}
	case "delete", "ctrl-4":
		if e.allSelected {
			e.clear()
			return
		}
		if e.cursor < len(e.input) {
			e.input = append(e.input[:e.cursor], e.input[e.cursor+1:]...)
		}
	case "space":
		e.insert(' ')
	case "rune":
		e.insert(k.r)
	}
}

func (e *promptEditor) complete(candidates []string, direction int) bool {
	if e == nil || len(candidates) == 0 {
		return false
	}
	current := e.value()
	if len(e.completionMatches) == 0 || !stringInSlice(e.completionMatches, current) {
		e.completionBase = current
		e.completionMatches = matchingCompletions(candidates, e.completionBase)
		if len(e.completionMatches) == 0 {
			return false
		}
		e.completionIndex = indexOfString(e.completionMatches, current)
	}
	if direction < 0 {
		e.completionIndex--
	} else {
		e.completionIndex++
	}
	if e.completionIndex < 0 {
		e.completionIndex = len(e.completionMatches) - 1
	}
	if e.completionIndex >= len(e.completionMatches) {
		e.completionIndex = 0
	}
	e.input = []rune(e.completionMatches[e.completionIndex])
	e.cursor = len(e.input)
	e.allSelected = false
	return true
}

func (e *promptEditor) completionStatus() string {
	if e == nil || len(e.completionMatches) == 0 {
		return ""
	}
	return fmt.Sprintf("  completion %d/%d", e.completionIndex+1, len(e.completionMatches))
}

func (e *promptEditor) resetCompletion() {
	if e == nil {
		return
	}
	e.completionBase = ""
	e.completionMatches = nil
	e.completionIndex = 0
}

func (e *promptEditor) insert(r rune) {
	if e.allSelected {
		e.input = nil
		e.cursor = 0
		e.allSelected = false
	}
	e.input = append(e.input, 0)
	copy(e.input[e.cursor+1:], e.input[e.cursor:])
	e.input[e.cursor] = r
	e.cursor++
}

func (e *promptEditor) clear() {
	e.input = nil
	e.cursor = 0
	e.allSelected = false
}

func normalizeCompletions(values []string) []string {
	seen := map[string]bool{}
	completions := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		completions = append(completions, value)
	}
	sort.Strings(completions)
	return completions
}

func matchingCompletions(candidates []string, prefix string) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	matches := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if prefix == "" || strings.HasPrefix(strings.ToLower(candidate), prefix) {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func stringInSlice(values []string, needle string) bool {
	return indexOfString(values, needle) >= 0
}

func indexOfString(values []string, needle string) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}

func (a *app) renderPrompt(label string, editor *promptEditor, hasCompletions bool) {
	cols, rows := terminalSize()
	value := editor.value()
	input := label + ": " + value
	status := "editing " + label
	if editor.allSelected {
		status += "  all selected"
	}
	status += editor.completionStatus()
	help := "enter save  esc cancel  arrows move  ^a all  ^u clear"
	if hasCompletions {
		help = "tab/S-tab complete  enter save  esc cancel  arrows move  ^a all  ^u clear"
	}
	a.draw(cols, rows, input, status, help)
	cursorText := label + ": " + string(editor.input[:editor.cursor])
	cursorCol := min(cols-1, 3+displayWidth(cursorText))
	fmt.Printf("\x1b[2;%dH\x1b[?25h", cursorCol)
}

func (a *app) render() {
	cols, rows := terminalSize()
	a.draw(cols, rows, a.defaultInput(), a.status, normalHelpLine)
}

func (a *app) defaultInput() string {
	if a.filter != "" {
		return "/" + a.filter
	}
	return ">"
}

func (a *app) draw(cols, rows int, input string, status string, help string) {
	if rows < 9 {
		rows = 9
	}
	bodyHeight := a.visibleBodyHeight(rows, cols)
	bottomHelpHeight := a.bottomHelpHeight(rows, cols)
	treeCols := cols
	sideHelp := a.usesHelpSidePane(cols)
	if sideHelp {
		treeCols = cols - helpSideWidth
	}
	a.ensureCursorVisible(bodyHeight)

	var b strings.Builder
	b.WriteString("\x1b[?25l\x1b[H\x1b[2J")
	b.WriteString(styleLine(boxTop(cols, "Explorer"), ansiBorder, cols))
	b.WriteString("\n")
	b.WriteString(renderInputLine(cols, input, a.counter()))
	b.WriteString("\n")
	b.WriteString(styleLine(boxBottom(cols), ansiBorder, cols))
	b.WriteString("\n")
	b.WriteString(a.renderTabsLine(cols))
	b.WriteString("\n")

	end := min(len(a.treeRows), a.offset+bodyHeight)
	for line := 0; line < bodyHeight; line++ {
		index := a.offset + line
		if len(a.treeRows) == 0 && line == 0 {
			b.WriteString(styleLine("  No bookmarks. Press a to add a bookmark or A to add a folder.", ansiMuted, treeCols))
		} else if index < end {
			item := a.treeRows[index]
			b.WriteString(renderRow(item, treeCols, index == a.cursor))
		} else {
			b.WriteString(styleLine("", ansiFg, treeCols))
		}
		if sideHelp {
			b.WriteString(renderHelpPaneLine(line, helpSideWidth))
		}
		b.WriteString("\n")
	}

	for line := 0; line < bottomHelpHeight; line++ {
		b.WriteString(renderHelpPaneLine(line, cols))
		b.WriteString("\n")
	}
	b.WriteString(styleLine(statusLine(status), ansiMuted, cols))
	b.WriteString("\n")
	b.WriteString(styleLine(detailFooter(a.selectedNode()), ansiMuted, cols))
	b.WriteString("\n")
	b.WriteString(styleLine(help, ansiMuted, cols))
	fmt.Print(toTerminalLines(b.String()))
}

func (a *app) visibleBodyHeight(rows int, cols int) int {
	height := bodyHeightForRows(rows)
	if a != nil && a.showHelp && !a.usesHelpSidePane(cols) {
		height -= a.bottomHelpHeight(rows, cols)
	}
	return max(1, height)
}

func (a *app) bottomHelpHeight(rows int, cols int) int {
	if rows < 9 {
		rows = 9
	}
	if a == nil || !a.showHelp || a.usesHelpSidePane(cols) {
		return 0
	}
	maxHeight := max(0, rows-8)
	return min(len(helpPaneLines()), maxHeight)
}

func (a *app) usesHelpSidePane(cols int) bool {
	return a != nil && a.showHelp && cols >= 100 && cols-helpSideWidth >= 40
}

func renderHelpPaneLine(index int, cols int) string {
	lines := helpPaneLines()
	text := ""
	style := ansiMuted
	if index >= 0 && index < len(lines) {
		text = lines[index]
		if index == 0 {
			style = ansiBorder
		}
	}
	return styleLine(text, style, cols)
}

func helpPaneLines() []string {
	return []string{
		" Help  (?/esc close)",
		" enter/o/space  open or toggle",
		" j/k or arrows   move selection",
		" [/]             prev / next folder",
		" h/l             collapse / expand",
		" tab/S-tab       switch Space",
		" p/P             cycle / select profile",
		" a/A             add bookmark / folder",
		" e/r/t           edit / rename / tags",
		" m               move to folder",
		" J/K             reorder in folder",
		" d               delete",
		" u               undo last delete/move/order",
		" / and c         search / clear",
		" R               reload from disk",
		" g/G Home/End    top / bottom",
		" q or Ctrl-c     quit",
	}
}

func bodyHeightForRows(rows int) int {
	if rows < 9 {
		rows = 9
	}
	return rows - 7
}

func (a *app) ensureCursorVisible(bodyHeight int) {
	if len(a.treeRows) == 0 {
		a.cursor = 0
		a.offset = 0
		return
	}
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
	if a.cursor >= len(a.treeRows) {
		a.cursor = len(a.treeRows) - 1
	}
	if a.cursor < a.offset {
		a.offset = a.cursor
	}
	if a.cursor >= a.offset+bodyHeight {
		a.offset = a.cursor - bodyHeight + 1
	}
	maxOffset := max(0, len(a.treeRows)-bodyHeight)
	if a.offset > maxOffset {
		a.offset = maxOffset
	}
	if a.offset < 0 {
		a.offset = 0
	}
}

func (a *app) counter() string {
	if len(a.treeRows) == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d", a.cursor+1, len(a.treeRows))
}

func (a *app) renderTabsLine(cols int) string {
	a.tabRanges = nil
	spaces := a.spaces()
	if len(spaces) == 0 {
		return styleLine(" Spaces", ansiMuted, cols)
	}
	if a.spaceIndex < 0 {
		a.spaceIndex = 0
	}
	if a.spaceIndex >= len(spaces) {
		a.spaceIndex = len(spaces) - 1
	}
	var left strings.Builder
	left.WriteString(" Spaces ")
	for i, space := range spaces {
		title := space.Title
		if strings.TrimSpace(title) == "" {
			title = "Untitled"
		}
		start := displayWidth(left.String()) + 1
		segment := " " + title + " "
		if i == a.spaceIndex {
			segment = "[" + title + "]"
		}
		left.WriteString(segment)
		end := displayWidth(left.String())
		a.tabRanges = append(a.tabRanges, tabRange{index: i, start: start, end: end})
	}
	right := " profile=" + a.activeProfileName()
	leftText := left.String()
	if displayWidth(leftText)+displayWidth(right)+1 > cols {
		return styleLine(leftText, ansiMuted, cols)
	}
	gap := cols - displayWidth(leftText) - displayWidth(right)
	return ansiBg + ansiMuted + leftText + strings.Repeat(" ", max(0, gap)) + right + ansiReset
}

func (a *app) selectedNode() *bookmarks.Node {
	selected := a.selected()
	if selected == nil {
		return nil
	}
	return selected.node
}

func formatRow(item treeRow) string {
	prefix := treePrefix(item)
	n := item.node
	meta := formatNodeMeta(n)
	if n.IsFolder() {
		state := "▸"
		if n.Expanded {
			state = "▾"
		}
		return fmt.Sprintf("%s%s %s %s%s", prefix, state, folderIcon, n.Title, meta)
	}
	icon := bookmarkIcon(n)
	return fmt.Sprintf("%s%s%s %s%s", prefix, strings.Repeat(" ", bookmarkIconOffset), padRight(icon, bookmarkImageCols), n.Title, meta)
}

func treePrefix(item treeRow) string {
	if item.root {
		return ""
	}
	var b strings.Builder
	for _, last := range item.ancestorsLast {
		if last {
			b.WriteString("    ")
		} else {
			b.WriteString("│   ")
		}
	}
	if item.last {
		b.WriteString("└── ")
	} else {
		b.WriteString("├── ")
	}
	return b.String()
}

func formatNodeMeta(n *bookmarks.Node) string {
	parts := []string{}
	for _, tag := range n.Tags {
		parts = append(parts, "#"+tag)
	}
	if len(parts) == 0 {
		return ""
	}
	return "  " + strings.Join(parts, " ")
}

func statusLine(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return ""
	}
	return "status: " + status
}

func detailFooter(n *bookmarks.Node) string {
	if n == nil {
		return ""
	}
	if n.IsBookmark() {
		return fmt.Sprintf("%s  %s%s", n.Title, n.URL, formatNodeMeta(n))
	}
	return fmt.Sprintf("%s  folder  %d bookmarks%s", n.Title, bookmarks.CountBookmarks(n), formatNodeMeta(n))
}

func boxTop(cols int, title string) string {
	if cols < 4 {
		return strings.Repeat("─", max(cols, 0))
	}
	inner := cols - 2
	label := " " + title + " "
	labelWidth := displayWidth(label)
	if labelWidth >= inner {
		return "┌" + strings.Repeat("─", inner) + "┐"
	}
	left := (inner - labelWidth) / 2
	right := inner - labelWidth - left
	return "┌" + strings.Repeat("─", left) + label + strings.Repeat("─", right) + "┐"
}

func boxBottom(cols int) string {
	if cols < 4 {
		return strings.Repeat("─", max(cols, 0))
	}
	return "└" + strings.Repeat("─", cols-2) + "┘"
}

func renderInputLine(cols int, input string, counter string) string {
	if cols < 4 {
		return styleLine("", ansiBorder, cols)
	}
	inner := cols - 2
	left := " " + input
	right := counter + " "
	rightWidth := displayWidth(right)
	leftLimit := inner - rightWidth - 1
	if leftLimit < 1 {
		leftLimit = inner
		right = ""
		rightWidth = 0
	}
	left = truncateWidth(left, leftLimit)
	gap := inner - displayWidth(left) - rightWidth
	if gap < 0 {
		gap = 0
	}
	line := "│" + left + strings.Repeat(" ", gap) + right + "│"
	return ansiBg + ansiBorder + line + ansiReset
}

func renderRow(item treeRow, cols int, selected bool) string {
	line := "  " + formatRow(item)
	line = truncateWidth(line, cols)
	line = padRight(line, cols)

	style := ansiBg
	if selected {
		style = ansiSelectBg
	}
	if item.node.IsFolder() {
		style += ansiBlue
	} else {
		style += ansiFg
	}
	if item.root {
		style += ansiCyan
	}
	return style + line + ansiReset
}

func styleLine(s string, style string, cols int) string {
	return ansiBg + style + padRight(truncateWidth(s, cols), cols) + ansiReset
}

func truncateWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = stripControls(s)
	if displayWidth(s) <= width {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := runeWidth(r)
		if w == 0 {
			b.WriteRune(r)
			continue
		}
		if used+w >= width {
			if width-used > 0 {
				b.WriteRune('~')
			}
			break
		}
		b.WriteRune(r)
		used += w
	}
	return b.String()
}

func stripControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)
}

func padRight(s string, width int) string {
	current := displayWidth(s)
	if current >= width {
		return s
	}
	return s + strings.Repeat(" ", width-current)
}

func displayWidth(s string) int {
	width := 0
	for _, r := range stripControls(s) {
		width += runeWidth(r)
	}
	return width
}

func runeWidth(r rune) int {
	if r == '\t' {
		return 4
	}
	if r == 0 {
		return 0
	}
	if r < 32 || r == 127 {
		return 0
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

func isWideRune(r rune) bool {
	return (r >= 0x1100 && r <= 0x115f) ||
		r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff01 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x2fffd) ||
		(r >= 0x30000 && r <= 0x3fffd)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func toTerminalLines(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}
