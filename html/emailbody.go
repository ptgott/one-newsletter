package html

import (
	"bytes"
	"divnews/linksrc"
	"html/template"
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
// with a newsletter etc.
type EmailData struct {
	LinkSets []linksrc.Set
}

// GenerateBody produces an email body to send based on the unformatted
// content. It's meant to include multiple sources of links in the same
// email to reduce the number of emails we send.
func (ed EmailData) GenerateBody() (string, error) {

	var buf bytes.Buffer

	// The template text is constant, so suppressing the error
	tmpl, _ := template.New("body").Parse(emailBodyHTML)

	err := tmpl.Execute(&buf, ed)
	if err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil
}
