package notes

import (
	"errors"
	"testing"
	"time"
)

func newBook() *Book {
	now := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	return NewBook(func() time.Time {
		now = now.Add(time.Minute)
		return now
	})
}

func TestAddGetListDelete(t *testing.T) {
	b := newBook()
	if _, err := b.Add("  "); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty note: %v", err)
	}
	first, _ := b.Add("周会改到周三")
	second, _ := b.Add("发布前检查清单")
	if got, err := b.Get(first.ID); err != nil || got.Text != "周会改到周三" {
		t.Fatalf("Get = %v, %v", got, err)
	}
	if list := b.List(); len(list) != 2 || list[0] != second {
		t.Fatalf("List = %v", list)
	}
	if err := b.Delete(first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Get(first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted note: %v", err)
	}
	if err := b.Delete(first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete twice: %v", err)
	}
}
