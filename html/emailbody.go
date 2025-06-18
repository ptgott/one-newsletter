package html

import (
	"html/template"
	"strings"
	"sync"

	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/ptgott/one-newsletter/userconfig"
)

// BodySectionContent is used to populate email body templates
type BodySectionContent struct {
	PubName  string
	Items    []linksrc.LinkItem
	Overview string // General statement about the links scraped for the site
}

// NewBodySectionContent readies a linksrc.Set for inclusion in an email body.
// We want to keep linksrc.Set as close as possible to what a scraper had
// originally parsed, and BodySectionContent as close as possible to what
// a reader would want to see, while decoupling the two.
func NewBodySectionContent(s linksrc.Set) BodySectionContent {
	li := s.LinkItems()
	bsc := BodySectionContent{
		Items:   li,
		PubName: s.Name,
	}

	if len(li) == 0 {
		bsc.Overview = "We could not find any links for this site. "
		bsc.Overview = bsc.Overview + strings.Join(s.Messages(), " ")
		return bsc
	}

	bsc.Overview = ""
	return bsc
}

// Template meant to be populated with a []linksrc.Set
const emailBodyHTML = `<html>
<head>
</head>
<body>
	<p>One Newsletter found the following links.</p>
	{{ range . }}
		<h2>{{ .PubName }}</h2>
		<p>{{ .Overview }}</p>
		<ul>
		{{ range .Items }}
			<li>{{ .Caption }} (<a href="{{ .LinkURL }}">here</a>)</li>
		{{ end }}
		</ul>
	{{ end }}
</body>
</html>`

// Template meant to be populated with a []linksrc.Set.
// Meant to satisfy the text/plain MIME type.
const emailBodyText = `{{ range . }}
{{.PubName}}

{{.Overview}}
{{ range .Items }}
- {{.Caption}}
  {{.LinkURL}}

{{ end }}
{{ end }}
`

// NewsletterEmailData contains metadata for the body of an email to send
// with a newsletter etc. Since each linksrc.Set in linksets
// comes from a different upstream, this is designed to support
// concurrent access. You should create this with NewEmailData.
type NewsletterEmailData struct {
	content []BodySectionContent
	mtx     *sync.Mutex
}

// NewNewsletterEmailData safely creates an EmailData.
func NewNewsletterEmailData() *NewsletterEmailData {
	return &NewsletterEmailData{
		content: []BodySectionContent{},
		mtx:     &sync.Mutex{},
	}
}

// Add stores a new linksrc.Set in the EmailData in a
// goroutine-safe way. Callers must use Add for adding
// linksrc.Sets to the EmailData.
func (ed *NewsletterEmailData) Add(s linksrc.Set) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	ed.content = append(ed.content, NewBodySectionContent(s))
}

// populateEmailTemplate executes a package-local template with the provided
// EmailData and performs any last-minute checks needed to do this.
func populateEmailTemplate(ed *NewsletterEmailData, tmp string) string {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	var str strings.Builder
	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(tmp)
	tmpl.Execute(&str, ed.content)

	return str.String()
}

// GenerateBody produces an HTML email body to send based on the unformatted
// content. It's meant to include multiple sources of links in the same
// email to reduce the number of emails we send. Any scraping- or parsing-
// related error messages are included in the text.
func (ed *NewsletterEmailData) GenerateBody() string {
	return populateEmailTemplate(ed, emailBodyHTML)
}

// GenerateText produces an email body to send based on the unformatted
// content, satisfying the text/plain MIME type. It's meant to include multiple
// sources of links in the same email to reduce the number of emails we send.
// Any scraping- or parsing- related error messages are included in the text.
func (ed *NewsletterEmailData) GenerateText() string {
	return populateEmailTemplate(ed, emailBodyText)
}

// SummaryContent includes configuration details for a newsletter. Used to
// summarize all configured newsletters in an initial email.
type SummaryContent struct {
	Name     string
	Schedule string
}

// SummaryEmailData contains information for summarizing all configured
// newsletters in an initial email.
type SummaryEmailData struct {
	Content []SummaryContent
	mtx     *sync.Mutex
}

func NewSummaryEmailData(m *userconfig.Meta) SummaryEmailData {
	content := make([]SummaryContent, len(m.Newsletters))
	var i int
	for k, n := range m.Newsletters {
		content[i] = SummaryContent{
			Name:     k,
			Schedule: n.Schedule.String(),
		}
		i++
	}
	return SummaryEmailData{
		Content: content,
		mtx:     &sync.Mutex{},
	}
}

// populateSummaryEmailTemplate executes a package-local template with the
// provided SummaryEmailData and performs any last-minute checks needed to do this.
func populateSummaryEmailTemplate(ed *SummaryEmailData, tmp string) string {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	var str strings.Builder
	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(tmp)
	tmpl.Execute(&str, ed.Content)

	return str.String()
}

// summaryEmailBodyText is a template meant to be populated with a
// []SummaryContent.  Meant to satisfy the text/plain MIME type.
const summaryEmailBodyText = `You have configured the following newsletters:
{{ range . -}}
- {{.Name}}: {{.Schedule}}
{{ end }}
`

// Template meant to be populated with a []SummaryContent.
// Using tables for layout to avoid cross-client irregularities.
const summaryEmailBodyHTML = `<html>
<head>
</head>
<body>
	<p>You have configured the following newsletters:</p>
	<ul>
	{{ range . }}
	    <li>{{.Name}}: {{.Schedule}}</li>
	{{ end }}
	</ul>
</body>
</html>`

// GenerateText produces an email body to send based on the unformatted
// content, satisfying the text/plain MIME type. It's meant to include a summary
// of configured newsletters to include in an initial email.
func (ed *SummaryEmailData) GenerateText() string {
	return populateSummaryEmailTemplate(ed, summaryEmailBodyText)
}

// GenerateBody produces an HTML email body to send based on the unformatted
// content.
func (ed *SummaryEmailData) GenerateBody() string {
	return populateSummaryEmailTemplate(ed, summaryEmailBodyHTML)
}
