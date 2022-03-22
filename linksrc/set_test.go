package linksrc

import (
	"io"
	"net/url"
	"os"
	"path"
	"reflect"
	"testing"

	css "github.com/andybalholm/cascadia"
)

// mustReadFile reads the file at path p and fails the test on error. Must
// only call if the file is known to exist with the correct permissions.
func mustReadFile(p string, t *testing.T) io.Reader {
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// mustParseURL is a test utility for returning a single value
// from url.Parse where the input isn't user-defined and
// we'd rather panic on the error than return it.
func mustParseURL(raw string) url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return *u
}

func TestNewSet(t *testing.T) {
	tests := []struct {
		html    io.Reader
		name    string
		conf    Config
		code    int
		want    Set
		wantErr bool
	}{
		{
			name: "canonical/intended case",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("div a.itemName"),
				LinkSelector:    css.MustCompile("div a.itemName"),
			},
			want: Set{
				Name: "My Cool Publication",
				items: map[string]LinkItem{
					"http://www.example.com/stories/hot-take": {
						LinkURL: "http://www.example.com/stories/hot-take",
						Caption: "This is a hot take!",
					},
					"http://www.example.com/stories/stuff-happened": {
						LinkURL: "http://www.example.com/stories/stuff-happened",
						Caption: "Stuff happened today, yikes.",
					},
					"http://www.example.com/storiesreally-true": {
						LinkURL: "http://www.example.com/storiesreally-true",
						Caption: "Is this supposition really true?",
					},
				},
			},
		},
		{
			name: "canonical/intended case with relative link URLs",
			html: mustReadFile(path.Join("testdata", "straightforward-relative-links.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("a.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			want: Set{
				Name: "My Cool Publication",
				items: map[string]LinkItem{
					"http://www.example.com/stories/hot-take": {
						LinkURL: "http://www.example.com/stories/hot-take",
						Caption: "This is a hot take!",
					},
					"http://www.example.com/stories/stuff-happened": {
						LinkURL: "http://www.example.com/stories/stuff-happened",
						Caption: "Stuff happened today, yikes.",
					},
					"http://www.example.com/storiesreally-true": {
						LinkURL: "http://www.example.com/storiesreally-true",
						Caption: "Is this supposition really true?",
					},
				},
			},
		},
		{
			name: "canonical/intended case with only a link selector",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:         "My Cool Publication",
				URL:          mustParseURL("http://www.example.com"),
				LinkSelector: css.MustCompile("a"),
			},
			want: Set{
				Name: "My Cool Publication",
				items: map[string]LinkItem{
					"http://www.example.com/stories/hot-take": {
						LinkURL: "http://www.example.com/stories/hot-take",
						Caption: "This is a hot take!",
					},
					"http://www.example.com/stories/stuff-happened": {
						LinkURL: "http://www.example.com/stories/stuff-happened",
						Caption: "Stuff happened today, yikes.",
					},
					"http://www.example.com/storiesreally-true": {
						LinkURL: "http://www.example.com/storiesreally-true",
						Caption: "Is this supposition really true?",
					},
				},
			},
		},
		{
			name: "canonical/intended case with relative link URLs and only a link selector",
			html: mustReadFile(path.Join("testdata", "straightforward-relative-links.html"), t),
			conf: Config{
				Name:         "My Cool Publication",
				URL:          mustParseURL("http://www.example.com"),
				LinkSelector: css.MustCompile("a"),
			},
			want: Set{
				Name: "My Cool Publication",
				items: map[string]LinkItem{
					"http://www.example.com/stories/hot-take": {
						LinkURL: "http://www.example.com/stories/hot-take",
						Caption: "This is a hot take!",
					},
					"http://www.example.com/stories/stuff-happened": {
						LinkURL: "http://www.example.com/stories/stuff-happened",
						Caption: "Stuff happened today, yikes.",
					},
					"http://www.example.com/storiesreally-true": {
						LinkURL: "http://www.example.com/storiesreally-true",
						Caption: "Is this supposition really true?",
					},
				},
			},
		},
		{
			name: "ambiguous link selector",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("*"),
			},
			want: Set{
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
				messages: []string{
					"The link selector is ambiguous, so we couldn't parse any link items.",
				},
			},
		},
		{
			name: "ambiguous caption selector",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("div"),
				LinkSelector:    css.MustCompile("a"),
			},
			want: Set{
				Name: "My Cool Publication",
				items: map[string]LinkItem{
					"http://www.example.com/stories/hot-take": {
						LinkURL: "http://www.example.com/stories/hot-take",
						Caption: "[Missing caption due to ambiguous selector]",
					},
					"http://www.example.com/stories/stuff-happened": {
						LinkURL: "http://www.example.com/stories/stuff-happened",
						Caption: "[Missing caption due to ambiguous selector]",
					},
					"http://www.example.com/storiesreally-true": {
						LinkURL: "http://www.example.com/storiesreally-true",
						Caption: "[Missing caption due to ambiguous selector]",
					},
				},
			},
		},
		{
			name: "no link selector matches",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a:nth-of-type(2)"),
			},
			want: Set{
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
				messages: []string{
					"There are no links in the list item. Double-check your configuration.",
				},
			},
		},
		{
			name: "the link selector matches a non-link",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile(".itemName"),
				LinkSelector:    css.MustCompile(".itemHolder"),
			},
			want: Set{
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
				messages: []string{
					"The link selector does not match a link but rather div.",
				},
			},
		},
		{
			name: "the caption selector has no matches",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("span.noMatch"),
				LinkSelector:    css.MustCompile("a"),
			},
			want: Set{
				Name: "My Cool Publication",
				items: map[string]LinkItem{
					"http://www.example.com/stories/hot-take": {
						LinkURL: "http://www.example.com/stories/hot-take",
						Caption: "",
					},
					"http://www.example.com/stories/stuff-happened": {
						LinkURL: "http://www.example.com/stories/stuff-happened",
						Caption: "",
					},
					"http://www.example.com/storiesreally-true": {
						LinkURL: "http://www.example.com/storiesreally-true",
						Caption: "",
					},
				},
			},
		},
		{
			name: "400 status code",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			code: 400,
			want: Set{
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
				messages: []string{
					"Got a 400 error sending the scrape request—check your config.",
				},
			},
		},
		{
			name: "500 status code",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			code: 500,
			want: Set{
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
				messages: []string{
					"Got a 500 error sending the scrape request—check manually to see if this is temporary.",
				},
			},
		},
		{
			name: "unexpected status code",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			code: 700,
			want: Set{
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
				messages: []string{
					"Unexpected status code 700. Try visiting the site manually.",
				},
			},
		},
		{
			name: "autodetect: ny magazine intelligencer",
			html: mustReadFile(path.Join("testdata", "intelligencer-feed.html"), t),
			conf: Config{
				Name:         "Intelligencer",
				URL:          mustParseURL("http://www.example.com"),
				LinkSelector: css.MustCompile("a.feed-item.article"),
				MaxItems:     3,
			},
			want: Set{
				Name: "Intelligencer",
				items: map[string]LinkItem{
					"http://www.example.com/intelligencer/2022/04/subway-shooting-proved-regular-new-yorkers-fight-crime-too.html": {
						LinkURL: "http://www.example.com/intelligencer/2022/04/subway-shooting-proved-regular-new-yorkers-fight-crime-too.html",
						Caption: "Mayor Adams needs to realize that cops aren’t the only crimefighters, as average New Yorkers proved during the hunt for the subway shooter.",
					},
					"http://www.example.com/intelligencer/2022/04/what-happened-to-paxlovid-the-covid-19-wonder-drug.html": {
						LinkURL: "http://www.example.com/intelligencer/2022/04/what-happened-to-paxlovid-the-covid-19-wonder-drug.html",
						Caption: "The much-hyped antiviral arrived too late for the Omicron wave, but it remains a powerful — and potentially versatile — weapon against COVID-19.",
					},
					"http://www.example.com/intelligencer/article/what-republicans-mean-rigged-election.html": {
						LinkURL: "http://www.example.com/intelligencer/article/what-republicans-mean-rigged-election.html",
						Caption: "Republicans claim Democrats are breaking election and voter laws. But deep down the complaint may be that perfectly legal votes are bad for the GOP.",
					},
				},
				messages: nil,
			},
		},
		{
			name: "autodetect: arts and letters daily",
			html: mustReadFile(path.Join("testdata", "aldaily.html"), t),
			conf: Config{
				Name:         "Arts and Letters Daily",
				URL:          mustParseURL("https://www.example.com"),
				LinkSelector: css.MustCompile("div.content-pad p a:nth-of-type(2)"),
				MaxItems:     3,
			},
			want: Set{
				Name: "Arts and Letters Daily",
				items: map[string]LinkItem{
					"https://www.example.com/2022/05/05/books/carlo-rovelli-physicist-book.html": {
						LinkURL: "https://www.example.com/2022/05/05/books/carlo-rovelli-physicist-book.html",
						Caption: "May 6, 2022 | “Capital ‘T,’ ‘the Truth’ … I don’t think it’s interesting,” says Carlo Rovelli. “The interesting thing is the small ‘t’...more»",
					},
					"https://www.example.com/archive/great-debates/weighing-evidence": {
						LinkURL: "https://www.example.com/archive/great-debates/weighing-evidence",
						Caption: "May 5, 2022 | Science advances not by convincing skeptics they are wrong, but by waiting until those skeptics die. Consider Galileo...more»",
					},
					"https://www.example.com/latest/miloszs-magic-mountain-neumeyer": {
						LinkURL: "https://www.example.com/latest/miloszs-magic-mountain-neumeyer",
						Caption: "May 4, 2022 | It's been said that every intellectual forced to emigrate is mutilated. So it was with Czeslaw Miloszin California...more»",
					},
				},
				messages: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSet(tt.html, tt.conf, tt.code)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSet() = %+v\nwanted: %+v", got, tt.want)
			}
		})
	}
}

func TestNewSetWithMaxLinks(t *testing.T) {
	tests := []struct {
		name          string
		conf          Config
		code          int
		wantSetLength int
	}{
		{
			name: "returned links over max link count",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("a.itemName"),
				LinkSelector:    css.MustCompile("a"),
				MaxItems:        2,
			},
			wantSetLength: 2,
		},
		{
			name: "returned links under max link count",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("a.itemName"),
				LinkSelector:    css.MustCompile("a"),
				MaxItems:        5,
			},
			wantSetLength: 3,
		},
		{
			name: "no max link count",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ol li"),
				CaptionSelector: css.MustCompile("a.itemName"),
				LinkSelector:    css.MustCompile("a"),
				MaxItems:        0,
			},
			wantSetLength: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSet(mustReadFile(path.Join("testdata", "straightforward.html"), t), tt.conf, tt.code)
			if len(got.items) != tt.wantSetLength {
				t.Errorf("wanted a Set with %v links but got %v", tt.wantSetLength, got)
			}
		})
	}
}

func TestRemoveItem(t *testing.T) {
	testCases := []struct {
		desc     string
		s        *Set
		toRemove LinkItem
		expected map[string]LinkItem
	}{
		{
			desc: "removing middle item",
			s: &Set{
				Name: "my set",
				items: map[string]LinkItem{
					"https://www.example.com/my-post1": {
						LinkURL: "https://www.example.com/my-post1",
						Caption: "This is another post",
					},
					"https://www.example.com/my-post2": {
						LinkURL: "https://www.example.com/my-post2",
						Caption: "This is a second post",
					},
					"https://www.example.com/my-post3": {
						LinkURL: "https://www.example.com/my-post3",
						Caption: "This is the final post",
					},
				},
			},
			toRemove: LinkItem{
				LinkURL: "https://www.example.com/my-post2",
				Caption: "This is a second post",
			},
			expected: map[string]LinkItem{
				"https://www.example.com/my-post1": {
					LinkURL: "https://www.example.com/my-post1",
					Caption: "This is another post",
				},
				"https://www.example.com/my-post3": {
					LinkURL: "https://www.example.com/my-post3",
					Caption: "This is the final post",
				},
			},
		},
		{
			desc: "no items",
			s: &Set{
				Name:  "my set",
				items: map[string]LinkItem{},
			},
			toRemove: LinkItem{},
			expected: map[string]LinkItem{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.s.RemoveLinkItem(tc.toRemove)
			if !reflect.DeepEqual(tc.s.items, tc.expected) {
				t.Errorf("wanted %v but got %v", tc.expected, tc.s.items)
			}
		})
	}
}
