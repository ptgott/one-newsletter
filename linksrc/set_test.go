package linksrc

import (
	"bytes"
	"net/url"
	"reflect"
	"testing"

	css "github.com/andybalholm/cascadia"
)

const testHTML = `<!doctype html5>
<html>
<head>
</head>
<body>
	<h1>This is my cool website</h1>
	<div id="mostRead">
		<h2>Most read posts today</h2>
		<ul>
			<li>
				<img src="img1.png">A cool image</img>
				<span class="itemHolder">
					<span class="itemNumber">1.</span>
					<span class="itemName">This is a hot take!</span>
				</span>
				<a href="http://www.example.com/stories/hot-take">
				Click here
				</a>
			</li>
			<li>
				<img src="img2.png">This is an image</img>
				<span class="itemHolder">
					<span class="itemNumber">2.</span>
					<span class="itemName">Stuff happened today, yikes.</span>
				</span>
				<a href="http://www.example.com/stories/stuff-happened">
				Click here
				</a>
			</li>
			<li>
				<img src="img3.png">This is also an image</img>
				<span class="itemHolder">
					<span class="itemNumber">3.</span>
					<span class="itemName">Is this supposition really true?</span>
				</span>
				<a href="http://www.example.com/storiesreally-true">
				Click here
				</a>
			</li>
		<ul>
	</div>
</body>
</html>`

// The origin here is http://www.example.com
const testHTMLRelativeLinks = `<!doctype html5>
<html>
<head>
</head>
<body>
	<h1>This is my cool website</h1>
	<div id="mostRead">
		<h2>Most read posts today</h2>
		<ul>
			<li>
				<img src="img1.png">A cool image</img>
				<span class="itemHolder">
					<span class="itemNumber">1.</span>
					<span class="itemName">This is a hot take!</span>
				</span>
				<a href="/stories/hot-take">
				Click here
				</a>
			</li>
			<li>
				<img src="img2.png">This is an image</img>
				<span class="itemHolder">
					<span class="itemNumber">2.</span>
					<span class="itemName">Stuff happened today, yikes.</span>
				</span>
				<a href="/stories/stuff-happened">
				Click here
				</a>
			</li>
			<li>
				<img src="img3.png">This is also an image</img>
				<span class="itemHolder">
					<span class="itemNumber">3.</span>
					<span class="itemName">Is this supposition really true?</span>
				</span>
				<a href="/storiesreally-true">
				Click here
				</a>
			</li>
		<ul>
	</div>
</body>
</html>`

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
		html    string
		name    string
		conf    Config
		code    int
		want    Set
		wantErr bool
	}{
		{
			name: "canonical/intended case",
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
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
			name: "canonical/intended case with relative link URLs",
			html: testHTMLRelativeLinks,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
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
			name: "ambiguous link selector",
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
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
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span"),
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
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
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
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("span.itemName"),
			},
			want: Set{
				Name:  "My Cool Publication",
				items: map[string]LinkItem{},
				messages: []string{
					"The link selector does not match a link but rather span.",
				},
			},
		},
		{
			name: "the caption selector has no matches",
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
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
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
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
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
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
			html: testHTML,
			conf: Config{
				Name:            "My Cool Publication",
				URL:             mustParseURL("http://www.example.com"),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewBuffer([]byte(tt.html))
			got := NewSet(r, tt.conf, tt.code)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSet() = %v, want %v", got, tt.want)
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
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
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
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
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
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
				MaxItems:        0,
			},
			wantSetLength: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewBuffer([]byte(testHTML))
			got := NewSet(r, tt.conf, tt.code)
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
