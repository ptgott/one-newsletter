package html

import (
	"bytes"
	"divnews/linksrc"
	"os"
	"sync"
	"testing"
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

	_, err = os.Stat(relativeGoldenHTMLFilePath)

	// This will always be an *os.PathError
	// https://golang.org/pkg/os/#Stat
	if err != nil {
		// not handling the error since it will only be a path error in
		// os.openFileNoLog, which os.Create wraps via os.OpenFile.
		gf, _ := os.Create(relativeGoldenHTMLFilePath)
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

	f, err := os.Open(relativeGoldenHTMLFilePath)

	if err != nil {
		t.Errorf("couldn't open the golden file for reading: %v", err)
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

func TestGenerateEmptyBody(t *testing.T) {
	s := []linksrc.Set{}
	ed := EmailData{
		linkSets: s,
		mtx:      &sync.Mutex{},
	}
	_, err := ed.GenerateBody()

	if err == nil {
		t.Error(
			"expected an error but not nil",
		)
	}
}
func TestGenerateEmptyText(t *testing.T) {
	s := []linksrc.Set{}
	ed := EmailData{
		linkSets: s,
		mtx:      &sync.Mutex{},
	}
	_, err := ed.GenerateText()

	if err == nil {
		t.Error(
			"expected an error but not nil",
		)
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

	h, err := ed.GenerateText()
	if err != nil {
		t.Errorf("couldn't generate text from the EmailData: %v", err)
	}

	_, err = os.Stat(relativeGoldenTextFilePath)

	// This will always be an *os.PathError
	// https://golang.org/pkg/os/#Stat
	if err != nil {
		// not handling the error since it will only be a path error in
		// os.openFileNoLog, which os.Create wraps via os.OpenFile.
		gf, _ := os.Create(relativeGoldenTextFilePath)
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

		// Don't check the in-memory text against the file we just created
		return

	}

	f, err := os.Open(relativeGoldenTextFilePath)

	if err != nil {
		t.Errorf("couldn't open the golden file for reading: %v", err)
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

func TestAdd(t *testing.T) {
	ed := NewEmailData()
	ed.Add(linksrc.Set{
		Name: "My Magazine",
		Items: []linksrc.LinkItem{
			{
				LinkURL: "http://www.example.com",
				Caption: "Something happened!",
			},
		},
	})

	if len(ed.linkSets) != 1 {
		t.Error("could not add to the EmailData")
	}
}
