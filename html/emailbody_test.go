package html

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Paths for golden file tests. To update a golden file, delete it and run the
// test again.
const (
	relativeGoldenHTMLFilePath        string = "golden-email-body.html"
	relativeGoldenTextFilePath        string = "golden-email-body.txt"
	relativeGoldenTextSummaryFilePath string = "golden-email-summary-body.txt"
)

// testGoldenFile opens the file at path or, if it doesn't exist, creates it
// with expected. If the file exists, checks expected against the content of the
// file.
func testGoldenFile(t *testing.T, path string, expected string) {
	_, err := os.Stat(path)

	// This will always be an *os.PathError
	// https://golang.org/pkg/os/#Stat
	if err != nil {
		// not handling the error since it will only be a path error in
		// os.openFileNoLog, which os.Create wraps via os.OpenFile.
		gf, _ := os.Create(path)
		defer gf.Close()

		_, err = gf.Write([]byte(expected))
		require.NoError(t, err)

		// Don't check the in-memory HTML against the file we just created
		return
	}

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var content bytes.Buffer
	_, err = content.ReadFrom(f)
	require.NoError(t, err)
	assert.Equal(t, expected, content.String())
}

func TestNewsletterEmailData_GenerateBody(t *testing.T) {
	ed := NewsletterEmailData{
		mtx: &sync.Mutex{},
		content: []BodySectionContent{
			{
				PubName:  "Example Site 1",
				Overview: "Here are the latest links:",
				Items: []linksrc.LinkItem{
					{
						LinkURL: "www.example.com/stories/hot-take",
						Caption: "This is a hot take!",
					},
					{
						LinkURL: "www.example.com/stories/stuff-happened",
						Caption: "Stuff happened today, yikes.",
					},
					{
						LinkURL: "www.example.com/storiesreally-true",
						Caption: "Is this supposition really true?",
					},
				},
			},
			{
				PubName:  "Example Site 2",
				Overview: "Here are the latest links:",
				Items: []linksrc.LinkItem{
					{
						LinkURL: "www.example.com/stories/tragedy",
						Caption: "This was a tragedy",
					},
					{
						LinkURL: "www.example.com/stories/heartfelt",
						Caption: "This story is heartfelt",
					},
				},
			},
		},
	}

	h := ed.GenerateBody()
	testGoldenFile(t, relativeGoldenHTMLFilePath, h)
}

func TestNewsletterEmailData_GenerateText(t *testing.T) {
	ed := NewsletterEmailData{
		mtx: &sync.Mutex{},
		content: []BodySectionContent{
			{
				PubName:  "Example Site 1",
				Overview: "Here are the latest links:",
				Items: []linksrc.LinkItem{
					{
						LinkURL: "www.example.com/stories/hot-take",
						Caption: "This is a hot take!",
					},
					{
						LinkURL: "www.example.com/stories/stuff-happened",
						Caption: "Stuff happened today, yikes.",
					},
					{
						LinkURL: "www.example.com/storiesreally-true",
						Caption: "Is this supposition really true?",
					},
				},
			},
			{
				PubName:  "Example Site 2",
				Overview: "Here are the latest links:",
				Items: []linksrc.LinkItem{
					{
						LinkURL: "www.example.com/stories/tragedy",
						Caption: "This was a tragedy",
					},
					{
						LinkURL: "www.example.com/stories/heartfelt",
						Caption: "This story is heartfelt",
					},
				},
			},
		},
	}

	h := ed.GenerateText()
	testGoldenFile(t, relativeGoldenTextFilePath, h)
}

func TestSummaryEmailData_GenerateText(t *testing.T) {
	sd := SummaryEmailData{
		mtx: &sync.Mutex{},
		content: []SummaryContent{
			{
				Name:     "News",
				URL:      "https://example.com/news",
				Schedule: "Mondays and Thursdays at 13:00",
			},
			{
				Name:     "Jokes",
				URL:      "https://example.com/jokes",
				Schedule: "Wednesdays at 16:00",
			},
			{
				Name:     "Events",
				URL:      "https://example.com/events",
				Schedule: "Fridays at 12:00",
			},
		},
	}

	h := sd.GenerateText()
	testGoldenFile(t, relativeGoldenTextSummaryFilePath, h)
}
