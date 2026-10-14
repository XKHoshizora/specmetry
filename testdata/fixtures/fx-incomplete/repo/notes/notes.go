// Package notes keeps short work notes for a small team.
package notes

import (
	"errors"
	"sort"
	"strings"
	"time"
)

// Note is one work note.
type Note struct {
	ID        int
	Text      string
	Tags      []string
	CreatedAt time.Time
}

// ErrEmpty is returned for a note without text.
var ErrEmpty = errors.New("note text is empty")

// ErrNotFound is returned for a note that does not exist.
var ErrNotFound = errors.New("note not found")

// Book holds the notes in memory.
type Book struct {
	now   func() time.Time
	next  int
	notes map[int]*Note
}

// NewBook returns an empty book that reads the time from now.
func NewBook(now func() time.Time) *Book {
	return &Book{now: now, notes: map[int]*Note{}}
}

// Add stores a note.
func (b *Book) Add(text string) (*Note, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmpty
	}
	b.next++
	n := &Note{ID: b.next, Text: text, CreatedAt: b.now()}
	b.notes[n.ID] = n
	return n, nil
}

// Get returns one note.
func (b *Book) Get(id int) (*Note, error) {
	n, ok := b.notes[id]
	if !ok {
		return nil, ErrNotFound
	}
	return n, nil
}

// List returns every note, newest first.
func (b *Book) List() []*Note {
	out := make([]*Note, 0, len(b.notes))
	for _, n := range b.notes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

// Delete removes a note.
func (b *Book) Delete(id int) error {
	if _, ok := b.notes[id]; !ok {
		return ErrNotFound
	}
	delete(b.notes, id)
	return nil
}
