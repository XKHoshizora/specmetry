package notes

import (
	"errors"
	"slices"
	"testing"
)

func TestSearchAndTag(t *testing.T) {
	b := newBook()
	weekly, _ := b.Add("Weekly sync moved to Wednesday")
	_, _ = b.Add("release checklist")
	if got := b.Search("WEEKLY"); len(got) != 1 || got[0] != weekly {
		t.Fatalf("Search = %v", got)
	}
	if got := b.Search(" "); len(got) != 0 {
		t.Fatalf("empty query = %v", got)
	}
	for _, tag := range []string{"Meeting", "meeting", " "} {
		if err := b.Tag(weekly.ID, tag); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(weekly.Tags, []string{"meeting"}) {
		t.Fatalf("tags = %v", weekly.Tags)
	}
	if err := b.Tag(99, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing note: %v", err)
	}
}
