package html

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/ptgott/one-newsletter/linksrc"
)

const (
	relativeGoldenHTMLFilePath string = "golden-email-body.html"
	relativeGoldenTextFilePath string = "golden-email-body.txt"
)

// GenerateBody straightforwardly populates a template and takes no input. As
// a result, there's not much that can go wrong. Still, we want to catch
// regressions, so we'll use a golden file here. To update the golden file,
// delete the file at $relativeGoldenHTMLFilePath before running this test. Edits
// to the golden file should be checked into version control.
func TestGenerateBody(t *testing.T) {
	ed := EmailData{
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

	_, err := os.Stat(relativeGoldenHTMLFilePath)

	// This will always be an *os.PathError
	// https://golang.org/pkg/os/#Stat
	if err != nil {
		// not handling the error since it will only be a path error in
		// os.openFileNoLog, which os.Create wraps via os.OpenFile.
		gf, _ := os.Create(relativeGoldenHTMLFilePath)
		defer gf.Close()

		_, err = gf.Write([]byte(h))

		if err != nil {
			t.Errorf("couldn't write to the golden file: %w", err)
		}
		// Don't check the in-memory HTML against the file we just created
		return
	}

	f, err := os.Open(relativeGoldenHTMLFilePath)

	if err != nil {
		t.Errorf("couldn't open the golden file for reading: %w", err)
	}

	var content bytes.Buffer
	_, err = content.ReadFrom(f)
	if err != nil {
		t.Errorf("couldn't read from the golden file %v", relativeGoldenHTMLFilePath)
	}
	if string(content.Bytes()) != h {
		t.Errorf("the HTML generated from GenerateBody does not match the golden file at %v", relativeGoldenHTMLFilePath)
	}

}

// GenerateText straightforwardly populates a template and takes no input. As
// a result, there's not much that can go wrong. Still, we want to catch
// regressions, so we'll use a golden file here. To update the golden file,
// delete the file at $relativeGoldenTextFilePath before running this test. Edits
// to the golden file should be checked into version control.
func TestGenerateText(t *testing.T) {
	ed := EmailData{
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

	_, err := os.Stat(relativeGoldenTextFilePath)

	// This will always be an *os.PathError
	// https://golang.org/pkg/os/#Stat
	if err != nil {
		// not handling the error since it will only be a path error in
		// os.openFileNoLog, which os.Create wraps via os.OpenFile.
		gf, _ := os.Create(relativeGoldenTextFilePath)
		defer gf.Close()

		_, err = gf.Write([]byte(h))

		if err != nil {
			t.Errorf("couldn't write to the golden file: %w", err)
		}

		// Don't check the in-memory text against the file we just created
		return

	}

	f, err := os.Open(relativeGoldenTextFilePath)

	if err != nil {
		t.Errorf("couldn't open the golden file for reading: %w", err)
	}

	var content bytes.Buffer
	_, err = content.ReadFrom(f)
	if err != nil {
		t.Errorf("couldn't read from the golden file %v", relativeGoldenTextFilePath)
	}
	if string(content.Bytes()) != h {
		t.Errorf("the text generated from GenerateBody does not match the golden file at %v", relativeGoldenTextFilePath)
	}
}
