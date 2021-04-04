package linksrc

import (
	"testing"
	"testing/quick"
)

func TestLinkItem_Key(t *testing.T) {

	tests := []struct {
		name     string
		LinkItem LinkItem
	}{
		{
			name: "values for LinkURL and Caption",
			LinkItem: LinkItem{
				LinkURL: "http://www.example.com",
				Caption: "This is a link",
			},
		},
		{
			name:     "empty values",
			LinkItem: LinkItem{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We're expecting a 32-byte hash here
			if got := tt.LinkItem.Key(); len(got) == 0 || len(got) > 32 {
				t.Errorf("unexpected key length %v", len(got))
			}
		})
	}
}

func TestLinkItem_NewKVEntry(t *testing.T) {
	// NewKVentry is really straightforward, so we'll just call the
	// function a ton of times with arbitrary inputs and see if
	// it outputs unwanted zero values.
	if err := quick.Check(func(linkURL string, caption string) bool {
		li := LinkItem{
			LinkURL: linkURL,
			Caption: caption,
		}

		kv := li.NewKVEntry()
		if len(kv.Key) == 0 || len(kv.Value) == 0 {
			return false
		}
		return true
	}, &quick.Config{
		MaxCount: 10000,
	}); err != nil {
		t.Error(err)
	}
}
