package html

import (
	"bytes"
	"divnews/linksrc"
	"html/template"
	"sync"
)

// Template meant to be populated with an EmailData.
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
			{{ range .LinkSets }}
				<h2>{{.Name}}</h2>
				{{ range .Items }}
					<p>{{.Caption}} (<a href="{{.LinkURL}}">here</a>)</p>
				{{ end }}
			{{ end }}
		</tbody>
	</table>
</body>
</html>`

// EmailData contains metadata for the body of an email to send
// with a newsletter etc. Since each linksrc.Set in linksets
// comes from a different upstream, this is designed to support
// concurrent access.
type EmailData struct {
	linkSets []linksrc.Set // These must not be written to directly
	mtx      sync.Mutex
}

// Add stores a new linksrc.Set in the EmailData in a
// goroutine-safe way. Callers must use Add for adding
// linksrc.Sets to the EmailData.
func (ed *EmailData) Add(s linksrc.Set) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	ed.linkSets = append(ed.linkSets, s)
}

// GenerateBody produces an email body to send based on the unformatted
// content. It's meant to include multiple sources of links in the same
// email to reduce the number of emails we send.
func (ed *EmailData) GenerateBody() (string, error) {
	ed.mtx.Lock()
	defer ed.mtx.Unlock()

	var buf bytes.Buffer

	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(emailBodyHTML)

	err := tmpl.Execute(&buf, ed)
	if err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil
}
