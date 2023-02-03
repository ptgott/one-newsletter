package linksrc

import (
	"context"
	"io"
	"net/url"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

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

// stringFromFile reads a file at path p and returns a UTF-8 encoded string
func stringFromFile(p string, t *testing.T) string {
	f := mustReadFile(p, t)
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
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
				Name:               "My Cool Publication",
				URL:                mustParseURL("http://www.example.com"),
				ItemSelector:       css.MustCompile("body div#mostRead ol li"),
				CaptionSelector:    css.MustCompile("div a.itemName"),
				LinkSelector:       css.MustCompile("div a.itemName"),
				ShortElementFilter: 3,
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
				Name:               "My Cool Publication",
				URL:                mustParseURL("http://www.example.com"),
				ItemSelector:       css.MustCompile("body div#mostRead ol li"),
				CaptionSelector:    css.MustCompile("a.itemName"),
				LinkSelector:       css.MustCompile("a"),
				ShortElementFilter: 3,
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
				Name:               "My Cool Publication",
				URL:                mustParseURL("http://www.example.com"),
				LinkSelector:       css.MustCompile("a"),
				ShortElementFilter: 3,
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
				Name:               "My Cool Publication",
				URL:                mustParseURL("http://www.example.com"),
				LinkSelector:       css.MustCompile("a"),
				ShortElementFilter: 3,
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
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
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
			name: "autodetect with link selector: ny magazine intelligencer",
			html: mustReadFile(path.Join("testdata", "intelligencer-feed.html"), t),
			conf: Config{
				Name:               "Intelligencer",
				URL:                mustParseURL("http://www.example.com"),
				LinkSelector:       css.MustCompile("a.feed-item.article"),
				MaxItems:           3,
				ShortElementFilter: 3,
			},
			want: Set{
				Name: "Intelligencer",
				items: map[string]LinkItem{
					"http://www.example.com/intelligencer/2022/04/subway-shooting-proved-regular-new-yorkers-fight-crime-too.html": {
						LinkURL: "http://www.example.com/intelligencer/2022/04/subway-shooting-proved-regular-new-yorkers-fight-crime-too.html",
						Caption: "Regular New Yorkers Fight Crime, Too. Mayor Adams needs to realize that cops aren’t the only crimefighters, as average...",
					},
					"http://www.example.com/intelligencer/2022/04/what-happened-to-paxlovid-the-covid-19-wonder-drug.html": {
						LinkURL: "http://www.example.com/intelligencer/2022/04/what-happened-to-paxlovid-the-covid-19-wonder-drug.html",
						Caption: "What Happened to Paxlovid, the COVID Wonder Drug? The much-hyped antiviral arrived too late for the Omicron wave, but it...",
					},
					"http://www.example.com/intelligencer/article/what-republicans-mean-rigged-election.html": {
						LinkURL: "http://www.example.com/intelligencer/article/what-republicans-mean-rigged-election.html",
						Caption: "What Is a ‘Rigged’ Election Anyway? Republicans claim Democrats are breaking election and voter laws. But deep down the complaint...",
					},
				},
				messages: nil,
			},
		},
		{
			name: "autodetect with link selector: arts and letters daily",
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
						Caption: "May 6, 2022 | “Capital ‘T,’ ‘the Truth’ … I don’t think it’s interesting,” says Carlo Rovelli. “The interesting thing...",
					},
					"https://www.example.com/archive/great-debates/weighing-evidence": {
						LinkURL: "https://www.example.com/archive/great-debates/weighing-evidence",
						Caption: "May 5, 2022 | Science advances not by convincing skeptics they are wrong, but by waiting until those skeptics die. Consider...",
					},
					"https://www.example.com/latest/miloszs-magic-mountain-neumeyer": {
						LinkURL: "https://www.example.com/latest/miloszs-magic-mountain-neumeyer",
						Caption: "May 4, 2022 | It's been said that every intellectual forced to emigrate is mutilated. So it was with Czeslaw...",
					},
				},
				messages: nil},
		},
		{
			name: "news source with a lot of short block-level HTMl text",
			html: mustReadFile(path.Join("testdata", "music-reviews.html"), t),
			conf: Config{
				Name:               "Music Review Site",
				URL:                mustParseURL("https://www.example.com"),
				LinkSelector:       css.MustCompile("div.review a.review__link"),
				ShortElementFilter: 0,
			},
			want: Set{
				Name: "Music Review Site",
				items: map[string]LinkItem{
					"https://www.example.com/reviews/albums/100-gecs-snake-eyes-ep/": LinkItem{
						LinkURL: "https://www.example.com/reviews/albums/100-gecs-snake-eyes-ep/",
						Caption: "100 gecs. Snake Eyes EP. Experimental. Electronic. by: Joshua Minsoo Kim. December 12 2022.",
					},
					"https://www.example.com/reviews/albums/brakence-hypochondriac/": LinkItem{
						LinkURL: "https://www.example.com/reviews/albums/brakence-hypochondriac/",
						Caption: "brakence. hypochondriac. Rock. by: H.D. Angel. December 15 2022.",
					},
				},
				messages: nil,
			},
			wantErr: false,
		},
		{
			name: "canonical/intended case with a URL-only config",
			html: mustReadFile(path.Join("testdata", "straightforward.html"), t),
			conf: Config{
				Name:               "My Cool Publication",
				URL:                mustParseURL("http://www.example.com"),
				ShortElementFilter: 3,
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
			name: "URL-only config with multiple container types",
			html: mustReadFile(path.Join("testdata", "straightforward_multiple_container_types.html"), t),
			conf: Config{
				Name:               "My Cool Publication",
				URL:                mustParseURL("http://www.example.com"),
				ShortElementFilter: 3,
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
					"http://www.example.com/stories/cool-headline": {
						LinkURL: "http://www.example.com/stories/cool-headline",
						Caption: "This is a headline for an article.",
					},
					"http://www.example.com/stories/cool-story": {
						LinkURL: "http://www.example.com/stories/cool-story",
						Caption: "This is a headline for another article.",
					},
				},
			},
		},
		{
			name: "autodetect in URL-only mode: NY magazine intelligencer",
			html: mustReadFile(path.Join("testdata", "intelligencer-feed.html"), t),
			conf: Config{
				Name:               "Intelligencer",
				URL:                mustParseURL("http://www.example.com"),
				MaxItems:           3,
				ShortElementFilter: 3,
			},
			want: Set{
				Name: "Intelligencer",
				items: map[string]LinkItem{
					"http://www.example.com/intelligencer/2022/04/subway-shooting-proved-regular-new-yorkers-fight-crime-too.html": {
						LinkURL: "http://www.example.com/intelligencer/2022/04/subway-shooting-proved-regular-new-yorkers-fight-crime-too.html",
						Caption: "Regular New Yorkers Fight Crime, Too. Mayor Adams needs to realize that cops aren’t the only crimefighters, as average...",
					},
					"http://www.example.com/intelligencer/2022/04/what-happened-to-paxlovid-the-covid-19-wonder-drug.html": {
						LinkURL: "http://www.example.com/intelligencer/2022/04/what-happened-to-paxlovid-the-covid-19-wonder-drug.html",
						Caption: "What Happened to Paxlovid, the COVID Wonder Drug? The much-hyped antiviral arrived too late for the Omicron wave, but it...",
					},
					"http://www.example.com/intelligencer/article/what-republicans-mean-rigged-election.html": {
						LinkURL: "http://www.example.com/intelligencer/article/what-republicans-mean-rigged-election.html",
						Caption: "What Is a ‘Rigged’ Election Anyway? Republicans claim Democrats are breaking election and voter laws. But deep down the complaint...",
					},
				},
				messages: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got := NewSet(ctx, tt.html, tt.conf, tt.code)
			assert.Equal(t, tt.want, got)
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
			got := NewSet(
				context.Background(),
				mustReadFile(path.Join("testdata", "straightforward.html"),
					t,
				), tt.conf, tt.code)
			if len(got.items) != tt.wantSetLength {
				t.Errorf("wanted a Set with %v links but got %v", tt.wantSetLength, got)
			}
		})
	}
}

func TestSetClean(t *testing.T) {
	testCases := []struct {
		description string
		input       Set
		expected    Set
	}{
		{
			description: "already clean set",
			input: Set{
				Name: "My Site 1",
				items: map[string]LinkItem{
					"item1": LinkItem{
						LinkURL: "https://www.example.com/article1",
						Caption: "This is my caption.",
					},
				},
				messages: []string{},
			},
			expected: Set{
				Name: "My Site 1",
				items: map[string]LinkItem{
					"item1": LinkItem{
						LinkURL: "https://www.example.com/article1",
						Caption: "This is my caption.",
					},
				},
				messages: []string{},
			},
		},
		{
			description: "whitespace-only caption",
			input: Set{
				Name: "My Site 1",
				items: map[string]LinkItem{
					"item1": LinkItem{
						LinkURL: "https://www.example.com/article1",
						Caption: " ",
					},
					"item2": LinkItem{
						LinkURL: "https://www.example.com/article2",
						Caption: "Something happened today.",
					},
				},
				messages: []string{},
			},
			expected: Set{
				Name: "My Site 1",
				items: map[string]LinkItem{"item2": LinkItem{
					LinkURL: "https://www.example.com/article2",
					Caption: "Something happened today.",
				},
				},
				messages: []string{},
			},
		},
	}

	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			actual := cleanSet(c.input)
			if !reflect.DeepEqual(actual, c.expected) {
				t.Fatalf("%v: expected %+v but got %+v", c.description, c.expected, actual)
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
