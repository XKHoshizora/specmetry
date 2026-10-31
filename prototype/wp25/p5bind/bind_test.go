//go:build darwin || wp25p5local

package main

// The decoder's model controls (docs/wp2.5-prototype.md §16.17.4): minimal
// Mach-O images built by code, with the outputs of otool and nm -m written
// to agree with them by A9's grammar. Each case's class and earliest
// failing rule are fixed in the table by the rules, never produced by the
// decoder under test. The base image:
//
//   - segments 0 __PAGEZERO, 1 __TEXT (r-x), 2 __DATA_CONST (rw-, vmaddr
//     0x100004000, vmsize 0x40) with one section __got over all of it,
//     3 __LINKEDIT;
//   - one LC_LOAD_DYLIB /usr/lib/libSystem.B.dylib (ordinal 1);
//   - one undefined external _waitid;
//   - the bind stream SET_TYPE_IMM 1, SET_SEGMENT_AND_OFFSET_ULEB 2 X,
//     SET_DYLIB_ORDINAL_IMM 1, SET_SYMBOL_TRAILING_FLAGS_IMM 0 "_waitid",
//     DO_BIND, DONE, zero padding.
//
// Model controls only: no evidence of a real binary.

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

const bin = "/tmp/wp25.test"

type bsect struct {
	name       string
	addr, size uint64
	flags      uint32
}

type bseg struct {
	name           string
	vmaddr, vmsize uint64
	initprot       uint32
	sects          []bsect
}

// mimg describes one image to build.
type mimg struct {
	segs     []bseg
	dylibs   []uint32 // the command of each library load (LC_LOAD_DYLIB, ...)
	paths    []string
	extra    []uint32 // other load commands, empty bodies
	bind     []byte
	weak     uint32
	lazy     uint32
	syms     []string // undefined external symbols
	fat      bool
	cpu      uint32
	textHint string
}

func base() *mimg {
	return &mimg{
		segs: []bseg{
			{"__PAGEZERO", 0, 0x100000000, 0, nil},
			{"__TEXT", 0x100000000, 0x4000, 5, []bsect{{"__text", 0x100000000, 0x4000, 0}}},
			{"__DATA_CONST", 0x100004000, 0x40, 3, []bsect{{"__got", 0x100004000, 0x40, 0}}},
			{"__LINKEDIT", 0x100008000, 0x4000, 1, nil},
		},
		dylibs: []uint32{lcLoadDylib},
		paths:  []string{libSystem},
		bind:   stream(0),
		syms:   []string{"_waitid"},
		cpu:    cpus["arm64"],
	}
}

// ul is a ULEB128.
func ul(v uint64) []byte {
	var b []byte
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b = append(b, c|0x80)
			continue
		}
		return append(b, c)
	}
}

func cat(parts ...[]byte) []byte { return bytes.Join(parts, nil) }

func sym(name string, flags byte) []byte { return append(append([]byte{0x40 | flags}, name...), 0) }

var (
	typ1  = []byte{0x51}
	ord1  = []byte{0x11}
	bind  = []byte{0x90}
	done  = []byte{0x00}
	seg2X = func(x uint64) []byte { return cat([]byte{0x72}, ul(x)) }
)

// stream is the base stream at offset x.
func stream(x uint64) []byte { return cat(typ1, seg2X(x), ord1, sym("_waitid", 0), bind, done) }

// build lays the image out: the header, the load commands, then
// __LINKEDIT's contents (the bind stream padded to 16, the symbol table,
// its strings).
func (m *mimg) build() []byte {
	le := binary.LittleEndian
	var cmds [][]byte
	seg := func(s bseg, fileoff, filesize uint64) []byte {
		c := make([]byte, 72+80*len(s.sects))
		le.PutUint32(c[0:], lcSegment64)
		le.PutUint32(c[4:], uint32(len(c)))
		copy(c[8:24], s.name)
		le.PutUint64(c[24:], s.vmaddr)
		le.PutUint64(c[32:], s.vmsize)
		le.PutUint64(c[40:], fileoff)
		le.PutUint64(c[48:], filesize)
		le.PutUint32(c[56:], 7)
		le.PutUint32(c[60:], s.initprot)
		le.PutUint32(c[64:], uint32(len(s.sects)))
		for i, x := range s.sects {
			o := 72 + 80*i
			copy(c[o:o+16], x.name)
			copy(c[o+16:o+32], s.name)
			le.PutUint64(c[o+32:], x.addr)
			le.PutUint64(c[o+40:], x.size)
			le.PutUint32(c[o+64:], x.flags)
		}
		return c
	}
	// The sizes first: the commands' size fixes where __LINKEDIT begins.
	ncmds := len(m.segs) + 3 + len(m.dylibs) + len(m.extra)
	dyl := func(cmd uint32, path string) []byte {
		n := 24 + len(path) + 1
		n = (n + 7) &^ 7
		c := make([]byte, n)
		le.PutUint32(c[0:], cmd)
		le.PutUint32(c[4:], uint32(n))
		le.PutUint32(c[8:], 24)
		copy(c[24:], path)
		return c
	}
	size := 0
	for _, s := range m.segs {
		size += 72 + 80*len(s.sects)
	}
	size += 48 + 24 + 80
	for i, d := range m.dylibs {
		size += len(dyl(d, m.paths[i]))
	}
	size += 8 * len(m.extra)
	linkoff := uint64(32 + size)
	bindBytes := append([]byte(nil), m.bind...)
	for len(bindBytes)%16 != 0 {
		bindBytes = append(bindBytes, 0)
	}
	var strs []byte
	strs = append(strs, ' ', 0)
	var syms []byte
	for _, s := range m.syms {
		e := make([]byte, 16)
		le.PutUint32(e[0:], uint32(len(strs)))
		e[4] = 0x01 // N_UNDF | N_EXT
		syms = append(syms, e...)
		strs = append(strs, s...)
		strs = append(strs, 0)
	}
	link := cat(bindBytes, syms, strs)
	for _, s := range m.segs {
		fo, fs := uint64(0), uint64(0)
		if s.name == "__LINKEDIT" {
			fo, fs = linkoff, uint64(len(link))
		}
		cmds = append(cmds, seg(s, fo, fs))
	}
	info := make([]byte, 48)
	le.PutUint32(info[0:], lcDyldInfoOnly)
	le.PutUint32(info[4:], 48)
	le.PutUint32(info[16:], uint32(linkoff))
	le.PutUint32(info[20:], uint32(len(bindBytes)))
	le.PutUint32(info[28:], m.weak)
	le.PutUint32(info[36:], m.lazy)
	cmds = append(cmds, info)
	st := make([]byte, 24)
	le.PutUint32(st[0:], lcSymtab)
	le.PutUint32(st[4:], 24)
	le.PutUint32(st[8:], uint32(linkoff)+uint32(len(bindBytes)))
	le.PutUint32(st[12:], uint32(len(m.syms)))
	le.PutUint32(st[16:], uint32(linkoff)+uint32(len(bindBytes)+len(syms)))
	le.PutUint32(st[20:], uint32(len(strs)))
	cmds = append(cmds, st)
	ds := make([]byte, 80)
	le.PutUint32(ds[0:], lcDysymtab)
	le.PutUint32(ds[4:], 80)
	cmds = append(cmds, ds)
	for i, d := range m.dylibs {
		cmds = append(cmds, dyl(d, m.paths[i]))
	}
	for _, x := range m.extra {
		c := make([]byte, 8)
		le.PutUint32(c[0:], x)
		le.PutUint32(c[4:], 8)
		cmds = append(cmds, c)
	}
	h := make([]byte, 32)
	le.PutUint32(h[0:], 0xfeedfacf)
	le.PutUint32(h[4:], m.cpu)
	le.PutUint32(h[12:], 2)
	le.PutUint32(h[16:], uint32(ncmds))
	le.PutUint32(h[20:], uint32(size))
	le.PutUint32(h[24:], 0x200004)
	out := cat(h, bytes.Join(cmds, nil), link)
	if m.fat {
		f := make([]byte, 8)
		binary.BigEndian.PutUint32(f[0:], 0xcafebabe)
		binary.BigEndian.PutUint32(f[4:], 1)
		out = cat(f, out)
	}
	return out
}

// texts are the tools' outputs that agree with the image, by A9's grammar.
func (m *mimg) texts() [5]tool {
	b := m.build()
	le := binary.LittleEndian
	var h, l, L, nm strings.Builder
	fmt.Fprintf(&h, "%s:\nMach header\n%s\n", bin, hHeader)
	fmt.Fprintf(&h, " 0x%08x %7d %10d  0x%02x  %10d %5d %10d 0x%08x\n", 0xfeedfacf, int32(m.cpu), 0, 0, 2, le.Uint32(b[16:]), le.Uint32(b[20:]), le.Uint32(b[24:]))
	fmt.Fprintf(&l, "%s:\n", bin)
	n := 0
	block := func(cmd string, body ...string) {
		fmt.Fprintf(&l, "Load command %d\n      cmd %s\n  cmdsize 0\n", n, cmd)
		for _, x := range body {
			l.WriteString(x + "\n")
		}
		n++
	}
	for range m.segs {
		block("LC_SEGMENT_64", "  segname x")
	}
	info := []string{}
	bo := 32 + le.Uint32(b[20:])
	vals := []uint32{0, 0, bo, uint32(len(m.bind)+15) &^ 15, 0, m.weak, 0, m.lazy, 0, 0}
	for i, name := range infoNames {
		info = append(info, fmt.Sprintf("%15s %d", name, vals[i]))
	}
	block("LC_DYLD_INFO_ONLY", info...)
	block("LC_SYMTAB")
	block("LC_DYSYMTAB")
	fmt.Fprintf(&L, "%s:\n", bin)
	for i, d := range m.dylibs {
		block(lcNames[d], fmt.Sprintf("         name %s (offset 24)", m.paths[i]))
		fmt.Fprintf(&L, "\t%s (compatibility version 0.0.0, current version 0.0.0)\n", m.paths[i])
	}
	for _, x := range m.extra {
		block(lcNames[x])
	}
	nm.WriteString("0000000100000000 (__TEXT,__text) external _main\n")
	for _, s := range m.syms {
		fmt.Fprintf(&nm, "                 (undefined) external %s\n", s)
	}
	return [5]tool{
		{Name: "otool -h", out: []byte(h.String())},
		{Name: "otool -hv", out: []byte(bin + ":\nMach header\n      magic  cputype cpusubtype  caps    filetype ncmds sizeofcmds      flags\nMH_MAGIC_64 ARM64 ALL 0x00 EXECUTE 0 0 DYLDLINK PIE\n")},
		{Name: "otool -l", out: []byte(l.String())},
		{Name: "otool -L", out: []byte(L.String())},
		{Name: "nm -m", out: []byte(nm.String())},
	}
}

// try runs A1 to A9 on the image, with the sizes and hash of its own bytes
// (as the step records them), and the tools' texts after edit.
func try(m *mimg, edit func(*[5]tool)) (*image, *fail) {
	b := m.build()
	sum := sha256.Sum256(b)
	ts := m.texts()
	if edit != nil {
		edit(&ts)
	}
	return check(b, bin, "arm64", int64(len(b)), hex.EncodeToString(sum[:]), ts)
}

func replace(i int, old, new string) func(*[5]tool) {
	return func(ts *[5]tool) { ts[i].out = []byte(strings.Replace(string(ts[i].out), old, new, 1)) }
}

func TestBind(t *testing.T) {
	two := func(m *mimg) { m.dylibs, m.paths = []uint32{lcLoadDylib, lcLoadDylib}, []string{libSystem, libSystem} }
	cases := []struct {
		name  string
		mod   func(*mimg)
		edit  func(*[5]tool)
		class string // "ok" or the T
		rule  string
		addr  uint64 // ok: the target's address
		total int    // ok: the records expanded
	}{
		// The positive cases.
		{name: "base X=0x00", class: "ok", addr: 0x100004000, total: 1},
		{name: "the last whole pointer X=0x38", mod: func(m *mimg) { m.bind = stream(0x38) }, class: "ok", addr: 0x100004038, total: 1},
		{name: "count=1 skip=0", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0xc0}, ul(1), ul(0), done)
		}, class: "ok", addr: 0x100004000, total: 1},
		{name: "scaled _other then _waitid", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0), ord1, sym("_other", 0), []byte{0xb2}, sym("_waitid", 0), bind, done)
		}, class: "ok", addr: 0x100004018, total: 2},
		// The negative cases.
		{name: "scaled, both _waitid", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0xb2}, bind, done)
		}, class: tAttribution, rule: "A5"},
		{name: "ordinal -2", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), []byte{0x3e}, sym("_waitid", 0), bind, done) }, class: tAttribution, rule: "A6"},
		{name: "ordinal 0", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), []byte{0x30}, sym("_waitid", 0), bind, done) }, class: tAttribution, rule: "A6"},
		{name: "ordinal -1", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), []byte{0x3f}, sym("_waitid", 0), bind, done) }, class: tAttribution, rule: "A6"},
		{name: "ordinal -3", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), []byte{0x3d}, sym("_waitid", 0), bind, done) }, class: tAttribution, rule: "A6"},
		{name: "ordinal 2 out of range", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), []byte{0x12}, sym("_waitid", 0), bind, done) }, class: tParser, rule: "A4.v3"},
		{name: "path libSystem.dylib", mod: func(m *mimg) { m.paths = []string{"/usr/lib/libSystem.dylib"} }, class: tAttribution, rule: "A6"},
		{name: "the same path twice", mod: two, class: tAttribution, rule: "A6"},
		{name: "LC_LOAD_WEAK_DYLIB", mod: func(m *mimg) { m.dylibs = []uint32{0x80000018} }, class: tParser, rule: "A3"},
		{name: "WEAK_IMPORT", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 1), bind, done) }, class: tAttribution, rule: "A6"},
		{name: "an addend", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0x60, 0x01}, bind, done) }, class: tAttribution, rule: "A6"},
		{name: "two DO_BIND", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), bind, bind, done) }, class: tAttribution, rule: "A5"},
		{name: "no record", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), done) }, class: tAttribution, rule: "A5"},
		{name: "_waitid$NOCANCEL", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), bind, sym("_waitid$NOCANCEL", 0), bind, done)
		}, class: tAttribution, rule: "A5"},
		{name: "THREADED", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0xd0}, done) }, class: tParser, rule: "A4.stream"},
		{name: "opcode 0xE0", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0xe0}, done) }, class: tParser, rule: "A4.stream"},
		// 16 bytes, so no padding follows: the last ULEB runs out at the end.
		{name: "a truncated ULEB", mod: func(m *mimg) { m.bind = cat(typ1, bytes.Repeat(ord1, 13), []byte{0x72, 0x80}) }, class: tParser, rule: "A4.leb"},
		{name: "LC_DYLD_CHAINED_FIXUPS", mod: func(m *mimg) { m.extra = []uint32{lcChainedFixups} }, class: tParser, rule: "A3"},
		{name: "LC_DYLD_INFO", mod: func(m *mimg) { m.extra = []uint32{lcDyldInfo} }, class: tParser, rule: "A3"},
		{name: "fat", mod: func(m *mimg) { m.fat = true }, class: tParser, rule: "A2"},
		{name: "no undefined _waitid in the symbol table", mod: func(m *mimg) { m.syms = []string{"_other"} }, class: tContradiction, rule: "A8"},
		{name: "otool -l's path differs", edit: replace(2, "name "+libSystem, "name /usr/lib/libFoo.dylib"), class: tContradiction, rule: "A9"},
		{name: "otool -h's flags differ", edit: replace(0, "0x00200004", "0x00200085"), class: tContradiction, rule: "A9"},
		{name: "nm -m from libFoo", edit: replace(4, "external _waitid", "external _waitid (from libFoo)"), class: tContradiction, rule: "A9"},
		// Added by the second draft (B-P112-2).
		{name: "vmsize 4, its section shrunk", mod: func(m *mimg) {
			m.segs[2].vmsize, m.segs[2].sects = 4, []bsect{{"__got", 0x100004000, 4, 0}}
		}, class: tParser, rule: "A4.v2"},
		{name: "vmsize 4, its section not shrunk", mod: func(m *mimg) { m.segs[2].vmsize = 4 }, class: tParser, rule: "A3"},
		{name: "across two sections", mod: func(m *mimg) {
			m.segs[2].sects = []bsect{{"__got", 0x100004000, 0x20, 0}, {"__other", 0x100004020, 0x20, 0}}
			m.bind = stream(0x1c)
		}, class: tParser, rule: "A4.v5"},
		{name: "initprot r--", mod: func(m *mimg) { m.segs[2].initprot = 1 }, class: tParser, rule: "A4.v4"},
		{name: "initprot rwx", mod: func(m *mimg) { m.segs[2].initprot = 7 }, class: tParser, rule: "A4.v4"},
		{name: "vmaddr + vmsize overflows", mod: func(m *mimg) {
			m.segs[2].vmaddr, m.segs[2].sects = ^uint64(0)-0x1f, []bsect{{"__got", ^uint64(0) - 0x1f, 0x20, 0}}
		}, class: tParser, rule: "A3"},
		{name: "ADD_ADDR_ULEB overflows", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(8), []byte{0x80}, ul(^uint64(0)-3), ord1, sym("_waitid", 0), bind, done)
		}, class: tParser, rule: "A4.arith"},
		{name: "an 11-byte ULEB", mod: func(m *mimg) {
			m.bind = cat(typ1, []byte{0x72}, bytes.Repeat([]byte{0x80}, 10), []byte{0x00}, ord1, sym("_waitid", 0), bind, done)
		}, class: tParser, rule: "A4.leb"},
		{name: "a ULEB's 10th byte 0x02", mod: func(m *mimg) {
			m.bind = cat(typ1, []byte{0x72}, bytes.Repeat([]byte{0x80}, 9), []byte{0x02}, ord1, sym("_waitid", 0), bind, done)
		}, class: tParser, rule: "A4.leb"},
		{name: "count 2^62 skip 8", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0xc0}, ul(1<<62), ul(8), done)
		}, class: tParser, rule: "A4.expand"},
		{name: "X=0x30 count 3 skip 0", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0x30), ord1, sym("_waitid", 0), []byte{0xc0}, ul(3), ul(0), done)
		}, class: tParser, rule: "A4.expand"},
		{name: "count 2 skip 0", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0xc0}, ul(2), ul(0), done)
		}, class: tAttribution, rule: "A5"},
		{name: "count 0", mod: func(m *mimg) {
			m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), []byte{0xc0}, ul(0), ul(0), done)
		}, class: tAttribution, rule: "A5"},
		{name: "a non-zero byte after DONE", mod: func(m *mimg) { m.bind = cat(stream(0), []byte{0x00, 0x07}) }, class: tParser, rule: "A4.stream"},
		// 16 bytes, so no zero padding (a DONE) follows.
		{name: "no DONE", mod: func(m *mimg) { m.bind = cat(typ1, seg2X(0), ord1, sym("_waitid", 0), bind, ord1, ord1) }, class: tParser, rule: "A4.stream"},
		{name: "weak_bind_size", mod: func(m *mimg) { m.weak = 16 }, class: tParser, rule: "A3"},
		{name: "lazy_bind_size", mod: func(m *mimg) { m.lazy = 16 }, class: tParser, rule: "A3"},
		{name: "SET_TYPE_IMM 2", mod: func(m *mimg) { m.bind = cat([]byte{0x52}, seg2X(0), ord1, sym("_waitid", 0), bind, done) }, class: tParser, rule: "A4.stream"},
		// Apple's output's grammar.
		{name: "the file's line with an architecture", edit: replace(0, bin+":", bin+" (architecture arm64):"), class: tParser, rule: "A9"},
		{name: "-L with , weak", edit: replace(3, "current version 0.0.0)", "current version 0.0.0, weak)"), class: tContradiction, rule: "A9"},
		{name: "-L without its versions", edit: replace(3, " (compatibility version 0.0.0, current version 0.0.0)", ""), class: tParser, rule: "A9"},
		{name: "-l's blocks not ncmds", edit: func(ts *[5]tool) {
			s := string(ts[2].out)
			i := strings.LastIndex(s, "Load command ")
			ts[2].out = []byte(s[:i])
		}, class: tParser, rule: "A9"},
		{name: "-h's numbers with 7 fields", edit: replace(0, "  0x00 ", " "), class: tParser, rule: "A9"},
		// The owner's decision (§16.19 supplement 9): an unmapped segment
		// (vmsize 0) as the fixed Go linker writes __DWARF, its sections
		// outside it but each a debug section; and one that is not.
		{name: "an unmapped debug segment", mod: func(m *mimg) {
			m.segs = append(m.segs[:3], bseg{"__DWARF", 0x100008000, 0, 0, []bsect{{"__zdebug_info", 0x100008000, 0x100, sAttrDebug}}}, m.segs[3])
		}, class: "ok", addr: 0x100004000, total: 1},
		{name: "an unmapped segment's section not for debugging", mod: func(m *mimg) {
			m.segs = append(m.segs[:3], bseg{"__DWARF", 0x100008000, 0, 0, []bsect{{"__zdebug_info", 0x100008000, 0x100, 0}}}, m.segs[3])
		}, class: tParser, rule: "A3"},
		// The tools' status and the identity.
		{name: "a tool exited 1", edit: func(ts *[5]tool) { ts[4].Status = 1 }, class: tTool, rule: "A9"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			if c.mod != nil {
				c.mod(m)
			}
			img, f := try(m, c.edit)
			switch {
			case c.class == "ok" && f != nil:
				t.Fatalf("want ok, got %s at %s: %s", f.Class, f.Rule, f.Reason)
			case c.class == "ok":
				if img.Target == nil || img.Target.Addr != c.addr || img.Target.Sect != "__got" || img.Total != c.total {
					t.Fatalf("the target %+v, %d records; want addr %#x in __got, %d records", img.Target, img.Total, c.addr, c.total)
				}
			case f == nil:
				t.Fatalf("want %s at %s, got ok", c.class, c.rule)
			case f.Class != c.class || f.Rule != c.rule:
				t.Fatalf("want %s at %s, got %s at %s: %s", c.class, c.rule, f.Class, f.Rule, f.Reason)
			}
		})
	}
}

// A1 and M5's shape: the bytes not the recorded ones; a file that is not
// Mach-O fails at A2 once A1 holds.
func TestBindIdentity(t *testing.T) {
	b := base().build()
	sum := sha256.Sum256(b)
	if _, f := decode(b, "arm64", int64(len(b))+1, hex.EncodeToString(sum[:])); f == nil || f.Class != tContradiction || f.Rule != "A1" {
		t.Fatalf("a size not recorded: %+v", f)
	}
	gs := []byte("golang.org/x/sys v0.49.0 h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA=\n")
	gsum := sha256.Sum256(gs)
	if _, f := decode(gs, "arm64", int64(len(gs)), hex.EncodeToString(gsum[:])); f == nil || f.Class != tParser || f.Rule != "A2" {
		t.Fatalf("not a Mach-O: %+v", f)
	}
}
