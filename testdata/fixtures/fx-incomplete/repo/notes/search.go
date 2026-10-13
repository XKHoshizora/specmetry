package notes

import (
	"slices"
	"strings"
)

// Search returns the notes whose text contains query, ignoring case,
// newest first.
func (b *Book) Search(query string) []*Note {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []*Note
	for _, n := range b.List() {
		if query != "" && strings.Contains(strings.ToLower(n.Text), query) {
			out = append(out, n)
		}
	}
	return out
}

// Tag adds a tag to a note once.
func (b *Book) Tag(id int, tag string) error {
	n, err := b.Get(id)
	if err != nil {
		return err
	}
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag != "" && !slices.Contains(n.Tags, tag) {
		n.Tags = append(n.Tags, tag)
	}
	return nil
}
