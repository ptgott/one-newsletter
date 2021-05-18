package linksrc

import (
	"bytes"
	"net/url"
	"reflect"
	"testing"

	css "github.com/andybalholm/cascadia"
)

// Using one HTML string in all unit tests. Just like
// in a real case, we can't change the HTML we want to
// scrape.
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
				<a href="www.example.com/stories/hot-take">
				Click here
				</a>
			</li>
			<li>
				<img src="img2.png">This is an image</img>
				<span class="itemHolder">
					<span class="itemNumber">2.</span>
					<span class="itemName">Stuff happened today, yikes.</span>
				</span>
				<a href="www.example.com/stories/stuff-happened">
				Click here
				</a>
			</li>
			<li>
				<img src="img3.png">This is also an image</img>
				<span class="itemHolder">
					<span class="itemNumber">3.</span>
					<span class="itemName">Is this supposition really true?</span>
				</span>
				<a href="www.example.com/storiesreally-true">
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
func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestNewSet(t *testing.T) {
	tests := []struct {
		name    string
		conf    Config
		code    int
		want    Set
		wantErr bool
	}{
		{
			name: "canonical/intended case",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			wantErr: false,
			want: Set{
				Name: "My Cool Publication",
				Items: []LinkItem{
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
		},
		{
			name: "ambiguous link selector",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("*"),
			},
			wantErr: true,
			want:    Set{},
		},
		{
			name: "ambiguous caption selector",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span"),
				LinkSelector:    css.MustCompile("a"),
			},
			wantErr: false,
			want: Set{
				Name: "My Cool Publication",
				Items: []LinkItem{
					{
						LinkURL: "www.example.com/stories/hot-take",
						Caption: "[Missing caption due to ambiguous selector]",
					},
					{
						LinkURL: "www.example.com/stories/stuff-happened",
						Caption: "[Missing caption due to ambiguous selector]",
					},
					{
						LinkURL: "www.example.com/storiesreally-true",
						Caption: "[Missing caption due to ambiguous selector]",
					},
				},
			},
		},
		{
			name: "no link selector matches",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a:nth-of-type(2)"),
			},
			wantErr: true,
			want:    Set{},
		},
		{
			name: "the link selector matches a non-link",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("span.itemName"),
			},
			wantErr: true,
			want:    Set{},
		},
		{
			name: "the caption selector has no matches",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.noMatch"),
				LinkSelector:    css.MustCompile("a"),
			},
			wantErr: false,
			want: Set{
				Name: "My Cool Publication",
				Items: []LinkItem{
					{
						LinkURL: "www.example.com/stories/hot-take",
						Caption: "",
					},
					{
						LinkURL: "www.example.com/stories/stuff-happened",
						Caption: "",
					},
					{
						LinkURL: "www.example.com/storiesreally-true",
						Caption: "",
					},
				},
			},
		},
		{
			name: "400 status code",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			code:    400,
			wantErr: false,
			want: Set{
				Name:   "My Cool Publication",
				Items:  []LinkItem{},
				Status: StatusMiscClientError,
			},
		},
		{
			name: "500 status code",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			code:    500,
			wantErr: false,
			want: Set{
				Name:   "My Cool Publication",
				Items:  []LinkItem{},
				Status: StatusServerError,
			},
		},
		{
			name: "unexpected status code",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
			},
			code:    700,
			wantErr: true,
			want:    Set{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewBuffer([]byte(testHTML))
			got, err := NewSet(r, tt.conf, tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
		wantErr       bool
	}{
		{
			name: "returned links over max link count",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
				MaxItems:        2,
			},
			wantSetLength: 2,
			wantErr:       false,
		},
		{
			name: "returned links under max link count",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
				MaxItems:        5,
			},
			wantSetLength: 3,
			wantErr:       false,
		},
		{
			name: "no max link count",
			conf: Config{
				Name:            "My Cool Publication",
				URL:             *(mustParseURL("http://www.example.com")),
				ItemSelector:    css.MustCompile("body div#mostRead ul li"),
				CaptionSelector: css.MustCompile("span.itemName"),
				LinkSelector:    css.MustCompile("a"),
				MaxItems:        0,
			},
			wantSetLength: 3,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewBuffer([]byte(testHTML))
			got, err := NewSet(r, tt.conf, tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got.Items) != tt.wantSetLength {
				t.Errorf("wanted a Set with %v links but got %v", tt.wantSetLength, got)
			}
		})
	}
}
