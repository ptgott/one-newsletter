package html

import (
	"bytes"
	"divnews/linksrc"
	"errors"
	"html/template"
	"sync"
)

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
			<tr><h1>Here are the latest links:</h1>
			{{ range . }}
				<h2>{{.Name}}</h2>
				{{ range .Items }}
					<p>{{.Caption}} (<a href="{{.LinkURL}}">here</a>)</p>
				{{ end }}
			{{ end }}
		</tbody>
	</table>
</body>
</html>`

// Template meant to be populated with a []linksrc.Set.
// Meant to satisfy the text/plain MIME type.
const emailBodyText = `Here are the latest links:
{{ range . }}
{{.Name}}
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

	if len(ed.linkSets) == 0 {
		return "",
			errors.New(
				"can't generate an email body from empty data",
			)
	}

	var buf bytes.Buffer

	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(emailBodyHTML)

	tmpl.Execute(&buf, ed.LinkSets())

	return string(buf.Bytes()), nil
}

// GenerateText produces an email body to send based on the unformatted
// content, satisfying the text/plain MIME type. It's meant to include multiple
// sources of links in the same email to reduce the number of emails we send.
func (ed *EmailData) GenerateText() (string, error) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	if len(ed.linkSets) == 0 {
		return "",
			errors.New(
				"can't generate an email text body from empty data",
			)
	}

	var buf bytes.Buffer

	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(emailBodyText)

	tmpl.Execute(&buf, ed.LinkSets())

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
