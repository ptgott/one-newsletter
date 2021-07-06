package html

import (
	"divnews/linksrc"
	"html/template"
	"strings"
	"sync"
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

	bsc.Overview = "Here are the latest links:"
	return bsc

}

// Template meant to be populated with a []linksrc.Set
// Using tables for layout to avoid cross-client irregularities.
// See here for best practices:
// https://www.smashingmagazine.com/2017/01/introduction-building-sending-html-email-for-web-developers/#using-html-tables-for-layout
const emailBodyHTML = `<html>
<head>
</head>
<body>
	<table>
		<tbody>
			<h1>Here are some new links!</h1>
			{{ range . }}
				<h2>{{ .PubName }}</h2>
				<p>{{ .Overview }}</p>
				{{ range .Items }}
					<p>{{ .Caption }} (<a href="{{ .LinkURL }}">here</a>)</p>
				{{ end }}
			{{ end }}
		</tbody>
	</table>
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

// EmailData contains metadata for the body of an email to send
// with a newsletter etc. Since each linksrc.Set in linksets
// comes from a different upstream, this is designed to support
// concurrent access. You should create this with NewEmailData.
type EmailData struct {
	content []BodySectionContent
	mtx     *sync.Mutex
}

// NewEmailData safely creates an EmailData.
func NewEmailData() *EmailData {
	return &EmailData{
		content: []BodySectionContent{},
		mtx:     &sync.Mutex{},
	}
}

// Add stores a new linksrc.Set in the EmailData in a
// goroutine-safe way. Callers must use Add for adding
// linksrc.Sets to the EmailData.
func (ed *EmailData) Add(s linksrc.Set) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	ed.content = append(ed.content, NewBodySectionContent(s))
}

// populateEmailTemplate executes a package-local template with the provided
// EmailData and performs any last-minute checks needed to do this.
func populateEmailTemplate(ed *EmailData, tmp string) string {
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
func (ed *EmailData) GenerateBody() string {
	return populateEmailTemplate(ed, emailBodyHTML)
}

// GenerateText produces an email body to send based on the unformatted
// content, satisfying the text/plain MIME type. It's meant to include multiple
// sources of links in the same email to reduce the number of emails we send.
// Any scraping- or parsing- related error messages are included in the text.
func (ed *EmailData) GenerateText() string {
	return populateEmailTemplate(ed, emailBodyText)
}
