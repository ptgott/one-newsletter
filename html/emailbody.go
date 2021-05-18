package html

import (
	"bytes"
	"divnews/linksrc"
	"errors"
	"html/template"
	"sync"
)

// BodySectionContent is used to populate email body templates
type BodySectionContent struct {
	PubName string
	Items   []linksrc.LinkItem
	Status  string
}

// NewBodySectionContent readies a linksrc.Set for inclusion in an email body.
// We want to keep linksrc.Set as close as possible to what a scraper had
// originally parsed, and BodySectionContent as close as possible to what
// a reader would want to see, while decoupling the two.
func NewBodySectionContent(s linksrc.Set) BodySectionContent {
	bsc := BodySectionContent{
		Items:   s.Items,
		PubName: s.Name,
	}

	if s.Status == linksrc.StatusOK && len(s.Items) == 0 {
		bsc.Status = "We could not find any links for this site! You might want to check your configuration."
		return bsc
	} else if s.Status == linksrc.StatusOK {
		bsc.Status = "Here are the latest links:"
		return bsc
	}

	errPreamble := "We could not find any links because of an error"
	errorMessages := map[linksrc.Status]string{
		linksrc.StatusNotAllowed:      ": we don't have permission to get links from this website. Check your configuration.",
		linksrc.StatusNotFound:        ": we couldn't find the website at this URL. Maybe it changed?",
		linksrc.StatusRateLimited:     ": we're being rate limited. You should change your configuration to check this site less frequently.",
		linksrc.StatusMiscClientError: "with our request to the site. Try reaching the site manually for more information.",
		linksrc.StatusServerError:     "with the site itself. Try reaching the site manually for more information.",
	}

	m, ok := errorMessages[s.Status]
	if !ok {
		// This assumes a linksrc.Status we haven't anticipated here.
		bsc.Status = errPreamble + ". Try reaching the site manually for more information."
		return bsc
	}
	bsc.Status = errPreamble + m

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
				<p>{{ .Status }}</p>
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

{{.Status}}
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
	linkSets []linksrc.Set // These must not be written to directly
	mtx      *sync.Mutex
}

// NewEmailData safely creates an EmailData.
func NewEmailData() *EmailData {
	return &EmailData{
		linkSets: []linksrc.Set{},
		mtx:      &sync.Mutex{},
	}
}

// Add stores a new linksrc.Set in the EmailData in a
// goroutine-safe way. Callers must use Add for adding
// linksrc.Sets to the EmailData.
func (ed *EmailData) Add(s linksrc.Set) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	ed.linkSets = append(ed.linkSets, s)
}

// GenerateBody produces an HTML email body to send based on the unformatted
// content. It's meant to include multiple sources of links in the same
// email to reduce the number of emails we send.
func (ed *EmailData) GenerateBody() (string, error) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	ls := ed.LinkSets()

	if len(ls) == 0 {
		return "",
			errors.New(
				"can't generate an email body from empty data",
			)
	}

	var buf bytes.Buffer

	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(emailBodyHTML)

	bc := make([]BodySectionContent, len(ls), len(ls))
	for i := range ls {
		bc[i] = NewBodySectionContent(ls[i])
	}

	tmpl.Execute(&buf, bc)

	return string(buf.Bytes()), nil
}

// GenerateText produces an email body to send based on the unformatted
// content, satisfying the text/plain MIME type. It's meant to include multiple
// sources of links in the same email to reduce the number of emails we send.
func (ed *EmailData) GenerateText() (string, error) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	ls := ed.LinkSets()

	if len(ls) == 0 {
		return "",
			errors.New(
				"can't generate an email text body from empty data",
			)
	}

	var buf bytes.Buffer

	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(emailBodyText)

	bc := make([]BodySectionContent, len(ls), len(ls))
	for i := range ls {
		bc[i] = NewBodySectionContent(ls[i])
	}

	tmpl.Execute(&buf, bc)

	return string(buf.Bytes()), nil
}

// LinkSets returns a copy of the linksrc.Sets currently tracked by the
// EmailData
func (ed *EmailData) LinkSets() []linksrc.Set {
	l := make([]linksrc.Set, len(ed.linkSets), len(ed.linkSets))

	for i := range l {
		l[i] = ed.linkSets[i]
	}
	return l
}
