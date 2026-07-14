package tui

import (
	"net/url"
	"strings"

	"github.com/yamajun9929/branchmark/internal/bookmarks"
)

const (
	folderIcon          = ""
	defaultBookmarkIcon = "󰖟"
	bookmarkIconOffset  = 0
	bookmarkImageCols   = 1
)

const (
	iconApple               = ""
	iconAws                 = "󰸏"
	iconBitbucket           = ""
	iconCalendar            = ""
	iconChat                = ""
	iconCloud               = ""
	iconCode                = ""
	iconDocument            = ""
	iconDropbox             = ""
	iconFigma               = ""
	iconForms               = ""
	iconGitHub              = ""
	iconGitLab              = ""
	iconGo                  = ""
	iconFirebase            = "󰥧"
	iconGoogle              = "󰊭"
	iconGoogleAds           = "󰲇"
	iconGoogleAnalytics     = "󰟌"
	iconGoogleChrome        = "󰊯"
	iconGoogleClassroom     = "󰋀"
	iconGoogleCloud         = "󱇶"
	iconGoogleDrive         = "󰊶"
	iconGoogleEarth         = "󰊷"
	iconGoogleKeep          = "󰛜"
	iconGoogleMaps          = "󰗵"
	iconGooglePlay          = "󰊼"
	iconGoogleSpreadsheet   = "󰧷"
	iconGoogleTranslate     = "󰊿"
	iconImage               = ""
	iconJira                = ""
	iconMail                = ""
	iconGmail               = "󰊫"
	iconMap                 = ""
	iconMarkdown            = ""
	iconMicrosoft           = "󰍲"
	iconMicrosoftAzure      = "󰠅"
	iconMicrosoftBing       = "󰂤"
	iconMicrosoftExcel      = "󱎏"
	iconMicrosoftOffice     = "󰏆"
	iconMicrosoftOneDrive   = "󰏊"
	iconMicrosoftOneNote    = "󰝇"
	iconMicrosoftOutlook    = "󰴢"
	iconMicrosoftPowerPoint = "󱎐"
	iconMicrosoftSharePoint = "󱎑"
	iconMicrosoftTeams      = "󰊻"
	iconMicrosoftWindows    = "󰖳"
	iconMicrosoftWord       = "󱎒"
	iconNode                = ""
	iconNotion              = "󰈙"
	iconPresentation        = ""
	iconSalesforce          = "󰢎"
	iconSheet               = ""
	iconSlack               = ""
	iconStats               = ""
	iconUsers               = ""
	iconVideo               = ""
	iconWindows             = iconMicrosoftWindows
	iconYahoo               = ""
	iconYouTube             = "󰗃"
)

type domainIconRule struct {
	icon    string
	domains []string
}

var domainIconRules = []domainIconRule{
	{iconGitHub, []string{"github.com"}},
	{iconGitLab, []string{"gitlab.com"}},
	{iconBitbucket, []string{"bitbucket.org"}},
	{iconYouTube, []string{"youtube.com", "youtu.be", "youtube-nocookie.com"}},
	{iconSalesforce, []string{"salesforce.com", "force.com", "visualforce.com", "salesforceiq.com"}},
	{iconYahoo, []string{"yahoo.com", "yahoo.co.jp"}},
	{iconMicrosoft, []string{"microsoft.com", "office.com", "office.net", "microsoft365.com"}},
	{iconWindows, []string{"windows.com"}},
	{iconApple, []string{"apple.com", "icloud.com"}},
	{iconAws, []string{"aws.amazon.com", "console.aws.amazon.com", "signin.aws.amazon.com", "amazonaws.com"}},
	{iconSlack, []string{"slack.com"}},
	{iconDropbox, []string{"dropbox.com"}},
	{iconFigma, []string{"figma.com"}},
	{iconNotion, []string{"notion.so", "notion.site"}},
	{iconJira, []string{"atlassian.net", "atlassian.com", "jira.com"}},
	{iconGo, []string{"go.dev", "golang.org"}},
	{iconNode, []string{"npmjs.com", "nodejs.org"}},
}

func bookmarkIcon(n *bookmarks.Node) string {
	if n == nil {
		return defaultBookmarkIcon
	}
	return iconForURL(n.URL)
}

func iconForURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return iconForRawURL(rawURL)
	}
	host := normalizeHost(parsed.Hostname())
	path := strings.ToLower(parsed.EscapedPath())

	if icon := googleIcon(host, path); icon != "" {
		return icon
	}
	if icon := microsoftIcon(host, path); icon != "" {
		return icon
	}
	for _, rule := range domainIconRules {
		if hostMatchesAny(host, rule.domains) {
			return rule.icon
		}
	}
	if strings.HasSuffix(strings.ToLower(rawURL), ".md") || strings.HasSuffix(path, ".md") {
		return iconMarkdown
	}
	return defaultBookmarkIcon
}

func googleIcon(host, path string) string {
	switch {
	case hostMatches(host, "drive.google.com"):
		return iconGoogleDrive
	case hostMatches(host, "docs.google.com") && strings.HasPrefix(path, "/spreadsheets"):
		return iconGoogleSpreadsheet
	case hostMatches(host, "docs.google.com") && strings.HasPrefix(path, "/document"):
		return iconDocument
	case hostMatches(host, "docs.google.com") && strings.HasPrefix(path, "/presentation"):
		return iconPresentation
	case hostMatches(host, "docs.google.com") && strings.HasPrefix(path, "/forms"):
		return iconForms
	case hostMatches(host, "sheets.google.com"):
		return iconGoogleSpreadsheet
	case hostMatches(host, "slides.google.com"):
		return iconPresentation
	case hostMatches(host, "mail.google.com") || hostMatches(host, "gmail.com") || hostMatches(host, "inbox.google.com"):
		return iconGmail
	case hostMatches(host, "calendar.google.com"):
		return iconCalendar
	case hostMatches(host, "meet.google.com"):
		return iconVideo
	case hostMatches(host, "chat.google.com"):
		return iconChat
	case hostMatches(host, "photos.google.com"):
		return iconImage
	case hostMatches(host, "maps.google.com") || hostMatches(host, "google.com") && strings.HasPrefix(path, "/maps"):
		return iconGoogleMaps
	case hostMatches(host, "cloud.google.com") || hostMatches(host, "console.cloud.google.com"):
		return iconGoogleCloud
	case hostMatches(host, "colab.research.google.com"):
		return iconCode
	case hostMatches(host, "lookerstudio.google.com"):
		return iconStats
	case hostMatches(host, "analytics.google.com"):
		return iconGoogleAnalytics
	case hostMatches(host, "ads.google.com"):
		return iconGoogleAds
	case hostMatches(host, "classroom.google.com"):
		return iconGoogleClassroom
	case hostMatches(host, "keep.google.com"):
		return iconGoogleKeep
	case hostMatches(host, "play.google.com"):
		return iconGooglePlay
	case hostMatches(host, "translate.google.com"):
		return iconGoogleTranslate
	case hostMatches(host, "earth.google.com"):
		return iconGoogleEarth
	case hostMatches(host, "firebase.google.com") || hostMatches(host, "console.firebase.google.com"):
		return iconFirebase
	case hostMatches(host, "chrome.google.com") || hostMatches(host, "chromewebstore.google.com"):
		return iconGoogleChrome
	case hostMatches(host, "google.com"):
		return iconGoogle
	default:
		return ""
	}
}

func microsoftIcon(host, path string) string {
	switch {
	case hostMatches(host, "teams.microsoft.com"):
		return iconMicrosoftTeams
	case hostMatches(host, "outlook.office.com") || hostMatches(host, "outlook.live.com") || hostMatches(host, "outlook.com"):
		return iconMicrosoftOutlook
	case hostMatches(host, "onedrive.live.com") || hostMatches(host, "onedrive.com"):
		return iconMicrosoftOneDrive
	case hostMatches(host, "sharepoint.com"):
		return iconMicrosoftSharePoint
	case hostMatches(host, "excel.office.com") || hostMatches(host, "office.com") && strings.Contains(path, "excel"):
		return iconMicrosoftExcel
	case hostMatches(host, "word.office.com") || hostMatches(host, "office.com") && strings.Contains(path, "word"):
		return iconMicrosoftWord
	case hostMatches(host, "powerpoint.office.com") || hostMatches(host, "office.com") && strings.Contains(path, "powerpoint"):
		return iconMicrosoftPowerPoint
	case hostMatches(host, "onenote.com") || hostMatches(host, "office.com") && strings.Contains(path, "onenote"):
		return iconMicrosoftOneNote
	case hostMatches(host, "portal.azure.com") || hostMatches(host, "azure.microsoft.com"):
		return iconMicrosoftAzure
	case hostMatches(host, "dev.azure.com"):
		return iconCode
	case hostMatches(host, "app.powerbi.com") || hostMatches(host, "powerbi.microsoft.com"):
		return iconStats
	case hostMatches(host, "bing.com"):
		return iconMicrosoftBing
	case hostMatches(host, "office.com") || hostMatches(host, "microsoft365.com"):
		return iconMicrosoftOffice
	default:
		return ""
	}
}

func iconForRawURL(rawURL string) string {
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(rawURL)), ".md") {
		return iconMarkdown
	}
	return defaultBookmarkIcon
}

func hostMatchesAny(host string, domains []string) bool {
	for _, domain := range domains {
		if hostMatches(host, domain) {
			return true
		}
	}
	return false
}

func hostMatches(host, domain string) bool {
	host = normalizeHost(host)
	domain = normalizeHost(domain)
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func normalizeHost(host string) string {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	return strings.TrimPrefix(host, "www.")
}
