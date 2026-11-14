package collect

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

// Ref names a record item: its kind and key, and the SHA-256 of its line.
// Derived items name their parents with it (method record §3.6).
type Ref struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// Recorder takes the tool's record items. Producers never touch files: a
// Recorder only appends to memory (plan §6.4); the record chain writes.
type Recorder interface {
	Raw(name string, v any) Ref
	Derived(name, step string, parents []Ref, v any) Ref
}

// Item is one record line before it is appended.
type Item struct {
	File string // "raw" or "derived"
	Line []byte // one JSON object and its LF
}

// Line builds a record line: a JSON object with the item's name, its step
// and parents for derived items, and the value. The SHA-256 of the line
// without its LF is the item's reference.
func Line(file, name, step string, parents []Ref, v any) (Item, Ref, error) {
	body := struct {
		Name    string `json:"name"`
		Step    string `json:"step,omitempty"`
		Parents []Ref  `json:"parents,omitempty"`
		Value   any    `json:"value"`
	}{name, step, parents, v}
	b, err := json.Marshal(body)
	if err != nil {
		return Item{}, Ref{}, fmt.Errorf("record %s: %w", name, err)
	}
	sum := sha256.Sum256(b)
	return Item{File: file, Line: append(b, '\n')}, Ref{Name: name, SHA256: hex.EncodeToString(sum[:])}, nil
}

// MemRecorder keeps the items in memory and hands each one to Sink (for
// example the record chain's log), in order.
type MemRecorder struct {
	mu     sync.Mutex
	Sink   func(Item)
	items  []Item
	errors []string
}

// Raw appends a raw record.
func (m *MemRecorder) Raw(name string, v any) Ref { return m.add("raw", name, "", nil, v) }

// Derived appends a derived record.
func (m *MemRecorder) Derived(name, step string, parents []Ref, v any) Ref {
	return m.add("derived", name, step, parents, v)
}

func (m *MemRecorder) add(file, name, step string, parents []Ref, v any) Ref {
	it, ref, err := Line(file, name, step, parents, v)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.errors = append(m.errors, err.Error())
		return Ref{Name: name}
	}
	m.items = append(m.items, it)
	if m.Sink != nil {
		m.Sink(it)
	}
	return ref
}

// Items returns the items so far.
func (m *MemRecorder) Items() []Item {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Item(nil), m.items...)
}

// Errors returns the marshalling errors (tool defects).
func (m *MemRecorder) Errors() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.errors...)
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
