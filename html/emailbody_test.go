package html

import (
	"bytes"
	"divnews/linksrc"
	"os"
	"testing"
)

const relativeGoldenFilePath string = "golden-email-body.html"

// GenerateBody straightforwardly populates a template and takes no input. As
// a result, there's not much that can go wrong. Still, we want to catch
// regressions, so we'll use a golden file here. To update the golden file,
// delete the file at $relativeGoldenFilePath before running this test. Edits
// to the golden file should be checked into version control.
func TestGenerateBody(t *testing.T) {

	ed := EmailData{
		linkSets: []linksrc.Set{
			{
				Name: "Example Site 1",
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
				Name: "Example Site 2",
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

	h, err := ed.GenerateBody()
	if err != nil {
		t.Errorf("couldn't generate HTML from the EmailData: %v", err)
	}

	_, err = os.Stat(relativeGoldenFilePath)

	// This will always be an *os.PathError
	// https://golang.org/pkg/os/#Stat
	if err != nil {
		// not handling the error since it will only be a path error in
		// os.openFileNoLog, which os.Create wraps via os.OpenFile.
		gf, _ := os.Create(relativeGoldenFilePath)
		defer gf.Close()

		_, err = gf.Write([]byte(h))

		if err != nil {
			t.Errorf("couldn't write to the golden file: %v", err)
		}

		// testing-internal error: shouldn't happen unless there's an issue
		// with your filesystem
		if err != nil {
			t.Errorf("error creating the golden file: %v", err)
		}

		// Don't check the in-memory HTML against the file we just created
		return

	}

	f, err := os.Open(relativeGoldenFilePath)

	if err != nil {
		t.Errorf("couldn't open the golden file for reading: %v", err)
	}

	var content bytes.Buffer
	_, err = content.ReadFrom(f)
	if err != nil {
		t.Errorf("couldn't read from the golden file %v", relativeGoldenFilePath)
	}
	if string(content.Bytes()) != h {
		t.Errorf("the HTML generated from GenerateBody does not match the golden file at %v", relativeGoldenFilePath)
	}

}
