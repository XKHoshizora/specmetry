// Command p5bind is P-5's binding decoder (docs/wp2.5-prototype.md
// §16.17.4, the second version of the symbol checks). It is test-only: the
// diagnostic's macOS static step builds and runs it on the test binary it
// built (go test -c), with the outputs of otool -h, -hv, -l, -L and nm -m
// that step saved. Standard library only, independent of the platform.
//
// It reads the Mach-O image itself and judges it by rules A1 to A9, in
// order; the first rule that fails gives the class (T(parser),
// T(contradiction), T(attribution) or T(tool)) and the run ends there:
//
//   - A1 the bytes it read are the ones the step recorded (size, SHA-256);
//   - A2 one 64-bit little-endian MH_EXECUTE of the expected CPU;
//   - A3 the load commands, segments and sections (a mapped segment's
//     within it; an unmapped one's, of vmsize 0, debug sections only), the
//     library ordinals (LC_LOAD_DYLIB only), one LC_DYLD_INFO_ONLY with an
//     empty weak and lazy stream, one __LINKEDIT holding the bind stream;
//   - A4 the plain bind stream by its supported grammar, every record it
//     expands checked as a whole 8-byte pointer in a writable, not
//     executable segment, inside one section;
//   - A5 exactly one record binds _waitid, no other symbol begins so;
//   - A6 that record is a plain pointer to the one LC_LOAD_DYLIB of
//     /usr/lib/libSystem.B.dylib;
//   - A7 where it lies is recorded (Go's prediction is __DATA_CONST,__got);
//   - A8 exactly one undefined external _waitid in the symbol table;
//   - A9 Apple's tools, parsed by the grammar of their printing code, agree.
//
// It prints one WP25P5BIND line (JSON), lines for a reader, and last one
// line "WP25P5BIND-VERDICT <class> <rule>" for the step's own matcher. It
// exits 0 for ok, 1 for a T, 2 for a usage error.
package main

import (
	"crypto/sha256"
	"debug/macho"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	format    = "p5bind/1"
	libSystem = "/usr/lib/libSystem.B.dylib"
	target    = "_waitid"
)

// fail is the first rule that did not hold.
type fail struct {
	Class  string `json:"class"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
}

func failf(class, rule, f string, args ...any) *fail {
	return &fail{class, rule, fmt.Sprintf(f, args...)}
}

const (
	tParser        = "T(parser)"
	tContradiction = "T(contradiction)"
	tAttribution   = "T(attribution)"
	tTool          = "T(tool)"
)

type section struct {
	Name   string `json:"sectname"`
	Seg    string `json:"segname"`
	Addr   uint64 `json:"addr"`
	Size   uint64 `json:"size"`
	Offset uint32 `json:"offset"`
	Flags  uint32 `json:"flags"`
}

type segment struct {
	Index    int       `json:"index"`
	Name     string    `json:"segname"`
	VMAddr   uint64    `json:"vmaddr"`
	VMSize   uint64    `json:"vmsize"`
	FileOff  uint64    `json:"fileoff"`
	FileSize uint64    `json:"filesize"`
	MaxProt  uint32    `json:"maxprot"`
	InitProt uint32    `json:"initprot"`
	Sections []section `json:"sections"`
}

type dylib struct {
	Ordinal int    `json:"ordinal"`
	Path    string `json:"path"`
}

// The ten fields of LC_DYLD_INFO_ONLY, in otool's order and spelling.
var infoNames = []string{"rebase_off", "rebase_size", "bind_off", "bind_size", "weak_bind_off", "weak_bind_size", "lazy_bind_off", "lazy_bind_size", "export_off", "export_size"}

type record struct {
	Seg     string `json:"segment"`
	Sect    string `json:"section"`
	Offset  uint64 `json:"offset"`
	Addr    uint64 `json:"addr"`
	Ordinal int64  `json:"ordinal"`
	Symbol  string `json:"symbol"`
	Flags   uint8  `json:"flags"`
	Type    uint8  `json:"type"`
	Addend  int64  `json:"addend"`
	Opcode  string `json:"opcode"`
	I       uint64 `json:"i"`
}

// image is what the decoder read.
type image struct {
	Size       int        `json:"size"`
	SHA256     string     `json:"sha256"`
	CPU        uint32     `json:"cputype"`
	Filetype   uint32     `json:"filetype"`
	Ncmds      uint32     `json:"ncmds"`
	Sizeofcmds uint32     `json:"sizeofcmds"`
	Flags      uint32     `json:"flags"`
	FlagNames  []string   `json:"flag_names"`
	TwoLevel   bool       `json:"twolevel"` // recorded only (§16.17.4 A2)
	Cmds       []string   `json:"cmds"`
	Segments   []segment  `json:"segments"`
	Linkedit   int        `json:"linkedit"`
	Dylibs     []dylib    `json:"dylibs"`
	Info       [10]uint32 `json:"dyld_info_only"`
	Records    []record   `json:"records"`
	Total      int        `json:"records_total"`
	Target     *record    `json:"target,omitempty"`
	NDesc      *uint16    `json:"nlist_desc,omitempty"`
	symoff     uint32
	nsyms      uint32
	stroff     uint32
	strsize    uint32
}

var cpus = map[string]uint32{"arm64": uint32(macho.CpuArm64), "x86_64": uint32(macho.CpuAmd64)}

// The load commands of <mach-o/loader.h>, by otool's names. A command not
// here is an unknown representation.
var lcNames = map[uint32]string{
	0x1: "LC_SEGMENT", 0x2: "LC_SYMTAB", 0x3: "LC_SYMSEG", 0x4: "LC_THREAD", 0x5: "LC_UNIXTHREAD",
	0x6: "LC_LOADFVMLIB", 0x7: "LC_IDFVMLIB", 0x8: "LC_IDENT", 0x9: "LC_FVMFILE", 0xa: "LC_PREPAGE",
	0xb: "LC_DYSYMTAB", 0xc: "LC_LOAD_DYLIB", 0xd: "LC_ID_DYLIB", 0xe: "LC_LOAD_DYLINKER", 0xf: "LC_ID_DYLINKER",
	0x10: "LC_PREBOUND_DYLIB", 0x11: "LC_ROUTINES", 0x12: "LC_SUB_FRAMEWORK", 0x13: "LC_SUB_UMBRELLA",
	0x14: "LC_SUB_CLIENT", 0x15: "LC_SUB_LIBRARY", 0x16: "LC_TWOLEVEL_HINTS", 0x17: "LC_PREBIND_CKSUM",
	0x80000018: "LC_LOAD_WEAK_DYLIB", 0x19: "LC_SEGMENT_64", 0x1a: "LC_ROUTINES_64", 0x1b: "LC_UUID",
	0x8000001c: "LC_RPATH", 0x1d: "LC_CODE_SIGNATURE", 0x1e: "LC_SEGMENT_SPLIT_INFO",
	0x8000001f: "LC_REEXPORT_DYLIB", 0x20: "LC_LAZY_LOAD_DYLIB", 0x21: "LC_ENCRYPTION_INFO",
	0x22: "LC_DYLD_INFO", 0x80000022: "LC_DYLD_INFO_ONLY", 0x80000023: "LC_LOAD_UPWARD_DYLIB",
	0x24: "LC_VERSION_MIN_MACOSX", 0x25: "LC_VERSION_MIN_IPHONEOS", 0x26: "LC_FUNCTION_STARTS",
	0x27: "LC_DYLD_ENVIRONMENT", 0x80000028: "LC_MAIN", 0x29: "LC_DATA_IN_CODE", 0x2a: "LC_SOURCE_VERSION",
	0x2b: "LC_DYLIB_CODE_SIGN_DRS", 0x2c: "LC_ENCRYPTION_INFO_64", 0x2d: "LC_LINKER_OPTION",
	0x2e: "LC_LINKER_OPTIMIZATION_HINT", 0x2f: "LC_VERSION_MIN_TVOS", 0x30: "LC_VERSION_MIN_WATCHOS",
	0x31: "LC_NOTE", 0x32: "LC_BUILD_VERSION", 0x80000033: "LC_DYLD_EXPORTS_TRIE",
	0x80000034: "LC_DYLD_CHAINED_FIXUPS", 0x80000035: "LC_FILESET_ENTRY",
}

const (
	lcSegment64     = 0x19
	lcSymtab        = 0x2
	lcDysymtab      = 0xb
	lcLoadDylib     = 0xc
	lcDyldInfo      = 0x22
	lcDyldInfoOnly  = 0x80000022
	lcChainedFixups = 0x80000034
	sAttrDebug      = 0x02000000 // S_ATTR_DEBUG
)

// The library loading commands other than LC_LOAD_DYLIB: each an
// unsupported representation (A3).
var otherLoads = map[uint32]bool{0x80000018: true, 0x8000001f: true, 0x20: true, 0x80000023: true}

var flagNames = []struct {
	bit  uint32
	name string
}{
	{0x1, "MH_NOUNDEFS"}, {0x2, "MH_INCRLINK"}, {0x4, "MH_DYLDLINK"}, {0x8, "MH_BINDATLOAD"},
	{0x10, "MH_PREBOUND"}, {0x20, "MH_SPLIT_SEGS"}, {0x40, "MH_LAZY_INIT"}, {0x80, "MH_TWOLEVEL"},
	{0x100, "MH_FORCE_FLAT"}, {0x200, "MH_NOMULTIDEFS"}, {0x400, "MH_NOFIXPREBINDING"}, {0x800, "MH_PREBINDABLE"},
	{0x1000, "MH_ALLMODSBOUND"}, {0x2000, "MH_SUBSECTIONS_VIA_SYMBOLS"}, {0x4000, "MH_CANONICAL"},
	{0x8000, "MH_WEAK_DEFINES"}, {0x10000, "MH_BINDS_TO_WEAK"}, {0x20000, "MH_ALLOW_STACK_EXECUTION"},
	{0x40000, "MH_ROOT_SAFE"}, {0x80000, "MH_SETUID_SAFE"}, {0x100000, "MH_NO_REEXPORTED_DYLIBS"},
	{0x200000, "MH_PIE"}, {0x400000, "MH_DEAD_STRIPPABLE_DYLIB"}, {0x800000, "MH_HAS_TLV_DESCRIPTORS"},
	{0x1000000, "MH_NO_HEAP_EXECUTION"}, {0x2000000, "MH_APP_EXTENSION_SAFE"},
	{0x4000000, "MH_NLIST_OUTOFSYNC_WITH_DYLDINFO"}, {0x8000000, "MH_SIM_SUPPORT"}, {0x80000000, "MH_DYLIB_IN_CACHE"},
}

var le = binary.LittleEndian

// add is an addition that reports its overflow.
func add(a, b uint64) (uint64, bool) { return a + b, a+b >= a }

// mul is a multiplication that reports its overflow.
func mul(a, b uint64) (uint64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	return c, c/b == a
}

// cstr is the NUL-terminated string at the start of b; ok is false
// without a NUL.
func cstr(b []byte) (string, bool) {
	for i, c := range b {
		if c == 0 {
			return string(b[:i]), true
		}
	}
	return "", false
}

// fixed is a fixed-size name field: up to its first NUL, or all of it.
func fixed(b []byte) string {
	if s, ok := cstr(b); ok {
		return s
	}
	return string(b)
}

// decode applies A1 to A8 to the bytes b.
func decode(b []byte, arch string, size int64, sha string) (*image, *fail) {
	sum := sha256.Sum256(b)
	img := &image{Size: len(b), SHA256: hex.EncodeToString(sum[:]), Linkedit: -1, Dylibs: []dylib{}, Records: []record{}}
	// A1: the bytes the step recorded.
	if int64(len(b)) != size || img.SHA256 != sha {
		return img, failf(tContradiction, "A1", "the decoder read %d bytes with SHA-256 %s; the step recorded %d and %s", len(b), img.SHA256, size, sha)
	}
	// A2: one 64-bit little-endian MH_EXECUTE of the expected CPU.
	if len(b) < 32 {
		return img, failf(tParser, "A2", "%d bytes: no Mach-O header", len(b))
	}
	switch be := binary.BigEndian.Uint32(b); {
	case be == 0xcafebabe || be == 0xcafebabf:
		return img, failf(tParser, "A2", "a fat file (magic %#x): not one architecture", be)
	case le.Uint32(b) == 0xcafebabe || le.Uint32(b) == 0xcafebabf:
		return img, failf(tParser, "A2", "a fat file (magic %#x, swapped): not one architecture", be)
	}
	if m := le.Uint32(b); m != uint32(macho.Magic64) {
		return img, failf(tParser, "A2", "magic %#x: not a 64-bit little-endian Mach-O (%#x)", m, uint32(macho.Magic64))
	}
	img.CPU, img.Filetype = le.Uint32(b[4:]), le.Uint32(b[12:])
	img.Ncmds, img.Sizeofcmds, img.Flags = le.Uint32(b[16:]), le.Uint32(b[20:]), le.Uint32(b[24:])
	for _, f := range flagNames {
		if img.Flags&f.bit != 0 {
			img.FlagNames = append(img.FlagNames, f.name)
		}
	}
	img.TwoLevel = img.Flags&0x80 != 0
	if img.Filetype != uint32(macho.TypeExec) {
		return img, failf(tParser, "A2", "filetype %d, not MH_EXECUTE", img.Filetype)
	}
	if img.CPU != cpus[arch] {
		return img, failf(tParser, "A2", "cputype %d, not %s's %d", img.CPU, arch, cpus[arch])
	}
	if f := loadCommands(b, img); f != nil {
		return img, f
	}
	if f := bindStream(b, img); f != nil {
		return img, f
	}
	if f := attribution(img); f != nil {
		return img, f
	}
	return img, symtab(b, img)
}

// loadCommands is A3.
func loadCommands(b []byte, img *image) *fail {
	end, ok := add(32, uint64(img.Sizeofcmds))
	if !ok || end > uint64(len(b)) {
		return failf(tParser, "A3", "the load commands (%d bytes) run past the file (%d bytes)", img.Sizeofcmds, len(b))
	}
	var nInfo, nSymtab, nDysymtab int
	off := uint64(32)
	for i := uint32(0); i < img.Ncmds; i++ {
		if off+8 > end {
			return failf(tParser, "A3", "load command %d starts past the commands' end", i)
		}
		cmd, size := le.Uint32(b[off:]), uint64(le.Uint32(b[off+4:]))
		if size < 8 || size%8 != 0 || off+size > end {
			return failf(tParser, "A3", "load command %d (%#x): cmdsize %d is not a multiple of 8 within the commands", i, cmd, size)
		}
		c := b[off : off+size]
		name, known := lcNames[cmd]
		if !known {
			return failf(tParser, "A3", "load command %d: %#x is not a known load command (an unknown representation)", i, cmd)
		}
		img.Cmds = append(img.Cmds, name)
		switch {
		case cmd == lcSegment64:
			if f := segment64(c, i, img); f != nil {
				return f
			}
		case cmd == lcSymtab:
			if size < 24 {
				return failf(tParser, "A3", "LC_SYMTAB: cmdsize %d", size)
			}
			nSymtab++
			img.symoff, img.nsyms, img.stroff, img.strsize = le.Uint32(c[8:]), le.Uint32(c[12:]), le.Uint32(c[16:]), le.Uint32(c[20:])
		case cmd == lcDysymtab:
			nDysymtab++
		case cmd == lcDyldInfoOnly:
			if size < 48 {
				return failf(tParser, "A3", "LC_DYLD_INFO_ONLY: cmdsize %d", size)
			}
			nInfo++
			for k := range img.Info {
				img.Info[k] = le.Uint32(c[8+4*k:])
			}
		case cmd == lcDyldInfo || cmd == lcChainedFixups:
			return failf(tParser, "A3", "%s: an unsupported representation", name)
		case otherLoads[cmd]:
			return failf(tParser, "A3", "load command %d is %s: only LC_LOAD_DYLIB is supported", i, name)
		case cmd == lcLoadDylib:
			if size < 24 {
				return failf(tParser, "A3", "LC_LOAD_DYLIB: cmdsize %d", size)
			}
			no := uint64(le.Uint32(c[8:]))
			if no < 24 || no >= size {
				return failf(tParser, "A3", "LC_LOAD_DYLIB: name offset %d outside its %d bytes", no, size)
			}
			path, ok := cstr(c[no:])
			if !ok {
				return failf(tParser, "A3", "LC_LOAD_DYLIB: its name has no NUL within the command")
			}
			img.Dylibs = append(img.Dylibs, dylib{len(img.Dylibs) + 1, path})
		}
		off += size
	}
	for _, s := range img.Segments {
		if s.Name == "__LINKEDIT" {
			if img.Linkedit >= 0 {
				return failf(tParser, "A3", "two __LINKEDIT segments")
			}
			img.Linkedit = s.Index
		}
	}
	switch {
	case img.Linkedit < 0:
		return failf(tParser, "A3", "no __LINKEDIT segment")
	case nInfo != 1:
		return failf(tParser, "A3", "%d LC_DYLD_INFO_ONLY commands, want 1", nInfo)
	case nSymtab != 1 || nDysymtab != 1:
		return failf(tParser, "A3", "%d LC_SYMTAB and %d LC_DYSYMTAB commands, want 1 each", nSymtab, nDysymtab)
	case img.Info[5] != 0 || img.Info[7] != 0:
		return failf(tParser, "A3", "weak_bind_size %d, lazy_bind_size %d: only the plain bind stream is supported", img.Info[5], img.Info[7])
	}
	bo, bs := uint64(img.Info[2]), uint64(img.Info[3])
	be, _ := add(bo, bs) // two 32-bit values: no overflow in 64 bits
	lk := img.Segments[img.Linkedit]
	lend, ok := add(lk.FileOff, lk.FileSize)
	switch {
	case bs == 0:
		return failf(tParser, "A3", "bind_size is 0")
	case be > uint64(len(b)):
		return failf(tParser, "A3", "the bind stream [%d, %d) runs past the file (%d bytes)", bo, be, len(b))
	case !ok || bo < lk.FileOff || be > lend:
		return failf(tParser, "A3", "the bind stream [%d, %d) is not within __LINKEDIT's file range", bo, be)
	}
	return nil
}

// segment64 reads one LC_SEGMENT_64 and its sections (A3).
func segment64(c []byte, i uint32, img *image) *fail {
	if len(c) < 72 {
		return failf(tParser, "A3", "load command %d: LC_SEGMENT_64 of %d bytes", i, len(c))
	}
	s := segment{Index: len(img.Segments), Name: fixed(c[8:24]), VMAddr: le.Uint64(c[24:]), VMSize: le.Uint64(c[32:]),
		FileOff: le.Uint64(c[40:]), FileSize: le.Uint64(c[48:]), MaxProt: le.Uint32(c[56:]), InitProt: le.Uint32(c[60:]), Sections: []section{}}
	n := uint64(le.Uint32(c[64:]))
	if n > uint64(len(c)-72)/80 {
		return failf(tParser, "A3", "segment %s: %d sections do not fit its command", s.Name, n)
	}
	send, ok := add(s.VMAddr, s.VMSize)
	if !ok {
		return failf(tParser, "A3", "segment %s: vmaddr %#x + vmsize %#x overflows", s.Name, s.VMAddr, s.VMSize)
	}
	// A segment of vmsize 0 is not mapped: the fixed Go linker writes its
	// __DWARF so (macho.go:753 to 757), its sections marked S_ATTR_DEBUG
	// (:651 to 653). Its sections need not lie within it, but each must be
	// a debug section (the owner's decision, §16.19 supplement 9). It holds
	// no bind record: A4's second rule wants vmsize 8 at least.
	unmapped := s.VMSize == 0
	for k := uint64(0); k < n; k++ {
		x := c[72+80*k:]
		sc := section{Name: fixed(x[0:16]), Seg: fixed(x[16:32]), Addr: le.Uint64(x[32:]), Size: le.Uint64(x[40:]), Offset: le.Uint32(x[48:]), Flags: le.Uint32(x[64:])}
		xe, ok := add(sc.Addr, sc.Size)
		switch {
		case sc.Seg != s.Name:
			return failf(tParser, "A3", "section %s names segment %s, in segment %s", sc.Name, sc.Seg, s.Name)
		case !ok:
			return failf(tParser, "A3", "section %s,%s: addr %#x + size %#x overflows", sc.Seg, sc.Name, sc.Addr, sc.Size)
		case unmapped && sc.Flags&sAttrDebug == 0:
			return failf(tParser, "A3", "section %s,%s of the unmapped segment %s (vmsize 0) is not a debug section (flags %#x)", sc.Seg, sc.Name, s.Name, sc.Flags)
		case !unmapped && (sc.Addr < s.VMAddr || xe > send):
			return failf(tParser, "A3", "section %s,%s [%#x, %#x) is not within its segment [%#x, %#x)", sc.Seg, sc.Name, sc.Addr, xe, s.VMAddr, send)
		}
		s.Sections = append(s.Sections, sc)
	}
	img.Segments = append(img.Segments, s)
	return nil
}

// The bind stream's opcodes (<mach-o/loader.h>).
var opNames = map[byte]string{
	0x00: "DONE", 0x10: "SET_DYLIB_ORDINAL_IMM", 0x20: "SET_DYLIB_ORDINAL_ULEB", 0x30: "SET_DYLIB_SPECIAL_IMM",
	0x40: "SET_SYMBOL_TRAILING_FLAGS_IMM", 0x50: "SET_TYPE_IMM", 0x60: "SET_ADDEND_SLEB",
	0x70: "SET_SEGMENT_AND_OFFSET_ULEB", 0x80: "ADD_ADDR_ULEB", 0x90: "DO_BIND", 0xa0: "DO_BIND_ADD_ADDR_ULEB",
	0xb0: "DO_BIND_ADD_ADDR_IMM_SCALED", 0xc0: "DO_BIND_ULEB_TIMES_SKIPPING_ULEB",
}

// state is the bind stream's state (A4).
type state struct {
	typ    uint8
	segSet bool
	seg    int
	off    uint64
	ordSet bool
	ord    int64
	symSet bool
	sym    string
	flags  uint8
	addend int64
}

// uleb reads a ULEB128 (A4): at most 10 bytes, fitting 64 bits, ending
// within the stream.
func uleb(b []byte, p *int) (uint64, *fail) {
	var v uint64
	for i := 0; i < 10; i++ {
		if *p >= len(b) {
			return 0, failf(tParser, "A4.leb", "a ULEB128 truncated at the stream's end")
		}
		c := b[*p]
		*p++
		if i == 9 {
			switch {
			case c&0x80 != 0:
				return 0, failf(tParser, "A4.leb", "a ULEB128 longer than 10 bytes")
			case c > 1:
				return 0, failf(tParser, "A4.leb", "a ULEB128 whose 10th byte %#x exceeds 64 bits", c)
			}
			return v | uint64(c)<<63, nil
		}
		v |= uint64(c&0x7f) << (7 * i)
		if c&0x80 == 0 {
			return v, nil
		}
	}
	panic("unreachable")
}

// sleb reads an SLEB128 by the same rules; its 10th byte is 0x00 or 0x7f.
func sleb(b []byte, p *int) (int64, *fail) {
	var v uint64
	for i := 0; i < 10; i++ {
		if *p >= len(b) {
			return 0, failf(tParser, "A4.leb", "an SLEB128 truncated at the stream's end")
		}
		c := b[*p]
		*p++
		if i == 9 {
			switch {
			case c&0x80 != 0:
				return 0, failf(tParser, "A4.leb", "an SLEB128 longer than 10 bytes")
			case c != 0 && c != 0x7f:
				return 0, failf(tParser, "A4.leb", "an SLEB128 whose 10th byte %#x does not fit 64 bits", c)
			}
			return int64(v | uint64(c&1)<<63), nil
		}
		v |= uint64(c&0x7f) << (7 * i)
		if c&0x80 == 0 {
			if c&0x40 != 0 {
				v |= ^uint64(0) << (7 * (i + 1))
			}
			return int64(v), nil
		}
	}
	panic("unreachable")
}

// valid checks one record the stream expands at offset off (A4's five
// rules), and gives it.
func valid(img *image, st *state, off uint64, op string, i uint64) (record, *fail) {
	if !st.segSet || st.seg >= img.Linkedit || st.seg >= len(img.Segments) {
		return record{}, failf(tParser, "A4.v1", "%s: segment %d not set, or not before __LINKEDIT (%d)", op, st.seg, img.Linkedit)
	}
	s := img.Segments[st.seg]
	if s.VMSize < 8 || off > s.VMSize-8 {
		return record{}, failf(tParser, "A4.v2", "%s: an 8-byte pointer at offset %#x does not fit segment %s's vmsize %#x", op, off, s.Name, s.VMSize)
	}
	if !st.symSet || !st.ordSet || st.ord < -3 || st.ord > int64(len(img.Dylibs)) {
		return record{}, failf(tParser, "A4.v3", "%s: symbol set %v, ordinal set %v, ordinal %d of %d libraries", op, st.symSet, st.ordSet, st.ord, len(img.Dylibs))
	}
	if st.typ != 1 || s.InitProt&2 == 0 || s.InitProt&4 != 0 {
		return record{}, failf(tParser, "A4.v4", "%s: type %d, segment %s initprot %#x: want a pointer in a writable, not executable segment", op, st.typ, s.Name, s.InitProt)
	}
	addr, ok := add(s.VMAddr, off)
	if !ok {
		return record{}, failf(tParser, "A4.v5", "%s: vmaddr %#x + offset %#x overflows", op, s.VMAddr, off)
	}
	r := record{Seg: s.Name, Offset: off, Addr: addr, Ordinal: st.ord, Symbol: st.sym, Flags: st.flags, Type: st.typ, Addend: st.addend, Opcode: op, I: i}
	pend, ok := add(addr, 8)
	for _, sc := range s.Sections {
		xe, _ := add(sc.Addr, sc.Size) // checked by A3
		if ok && addr >= sc.Addr && pend <= xe {
			r.Sect = sc.Name
			return r, nil
		}
	}
	return record{}, failf(tParser, "A4.v5", "%s: [%#x, %#x) is not within one section of segment %s", op, addr, pend, s.Name)
}

// bindStream is A4.
func bindStream(b []byte, img *image) *fail {
	bo, bs := int(img.Info[2]), int(img.Info[3])
	s := b[bo : bo+bs]
	var st state
	emit := func(off uint64, op string, i uint64) *fail {
		r, f := valid(img, &st, off, op, i)
		if f != nil {
			return f
		}
		img.Records = append(img.Records, r)
		img.Total++
		return nil
	}
	advance := func(n uint64, rule string) *fail {
		v, ok := add(st.off, n)
		if !ok {
			return failf(tParser, rule, "the offset %#x + %#x overflows", st.off, n)
		}
		st.off = v
		return nil
	}
	for p := 0; p < len(s); {
		c := s[p]
		p++
		op, imm := c&0xf0, c&0x0f
		name := opNames[op]
		switch op {
		case 0x00:
			for q := p; q < len(s); q++ {
				if s[q] != 0 {
					return failf(tParser, "A4.stream", "a non-zero byte %#x after DONE (offset %d): an unknown representation", s[q], q)
				}
			}
			return nil
		case 0x10:
			st.ord, st.ordSet = int64(imm), true
		case 0x20:
			v, f := uleb(s, &p)
			if f != nil {
				return f
			}
			if v > 1<<31-1 {
				return failf(tParser, "A4.stream", "SET_DYLIB_ORDINAL_ULEB %d exceeds 2^31 - 1", v)
			}
			st.ord, st.ordSet = int64(v), true
		case 0x30:
			st.ord, st.ordSet = 0, true
			if imm != 0 {
				st.ord = int64(int8(0xf0 | imm))
			}
		case 0x40:
			sym, ok := cstr(s[p:])
			if !ok {
				return failf(tParser, "A4.stream", "SET_SYMBOL_TRAILING_FLAGS_IMM: no NUL within the stream")
			}
			st.flags, st.sym, st.symSet = imm, sym, true
			p += len(sym) + 1
		case 0x50:
			if imm != 1 {
				return failf(tParser, "A4.stream", "SET_TYPE_IMM %d: only BIND_TYPE_POINTER (1) is supported", imm)
			}
			st.typ = imm
		case 0x60:
			v, f := sleb(s, &p)
			if f != nil {
				return f
			}
			st.addend = v
		case 0x70:
			v, f := uleb(s, &p)
			if f != nil {
				return f
			}
			st.seg, st.segSet, st.off = int(imm), true, v
		case 0x80:
			v, f := uleb(s, &p)
			if f != nil {
				return f
			}
			if f := advance(v, "A4.arith"); f != nil {
				return f
			}
		case 0x90:
			if f := emit(st.off, name, 0); f != nil {
				return f
			}
			if f := advance(8, "A4.arith"); f != nil {
				return f
			}
		case 0xa0:
			v, f := uleb(s, &p)
			if f != nil {
				return f
			}
			if f := emit(st.off, name, 0); f != nil {
				return f
			}
			n, ok := add(v, 8)
			if !ok {
				return failf(tParser, "A4.arith", "DO_BIND_ADD_ADDR_ULEB %#x + 8 overflows", v)
			}
			if f := advance(n, "A4.arith"); f != nil {
				return f
			}
		case 0xb0:
			if f := emit(st.off, name, 0); f != nil {
				return f
			}
			if f := advance(uint64(imm)*8+8, "A4.arith"); f != nil {
				return f
			}
		case 0xc0:
			count, f := uleb(s, &p)
			if f != nil {
				return f
			}
			skip, f := uleb(s, &p)
			if f != nil {
				return f
			}
			if count == 0 {
				continue
			}
			// The last record's offset, without overflow, checked before any
			// record is made; the records are not expanded otherwise.
			step, ok1 := add(skip, 8)
			span, ok2 := mul(count-1, step)
			last, ok3 := add(st.off, span)
			if !ok1 || !ok2 || !ok3 {
				return failf(tParser, "A4.expand", "%s count %d skip %d from %#x: the last offset overflows", name, count, skip, st.off)
			}
			if _, f := valid(img, &st, last, name, count-1); f != nil {
				return failf(tParser, "A4.expand", "%s count %d skip %d: its last record: %s (%s)", name, count, skip, f.Reason, f.Rule)
			}
			for i := uint64(0); i < count; i++ {
				if f := emit(st.off+i*step, name, i); f != nil {
					return f
				}
			}
			total, ok := mul(count, step)
			if !ok {
				return failf(tParser, "A4.arith", "%s: count %d x %d overflows", name, count, step)
			}
			if f := advance(total, "A4.arith"); f != nil {
				return f
			}
		default:
			return failf(tParser, "A4.stream", "opcode %#x (immediate %d) at offset %d: an unknown representation", op, imm, p-1)
		}
	}
	return failf(tParser, "A4.stream", "the bind stream ends without DONE")
}

// attribution is A5 to A7.
func attribution(img *image) *fail {
	var hits []int
	for i, r := range img.Records {
		switch {
		case r.Symbol == target:
			hits = append(hits, i)
		case strings.HasPrefix(r.Symbol, target):
			return failf(tAttribution, "A5", "a record binds %s", r.Symbol)
		}
	}
	switch len(hits) {
	case 0:
		return failf(tAttribution, "A5", "no record binds %s (of %d)", target, img.Total)
	case 1:
	default:
		return failf(tAttribution, "A5", "%d records bind %s: not unique", len(hits), target)
	}
	r := img.Records[hits[0]]
	img.Target = &r
	switch {
	case r.Flags&0x1 != 0 || r.Flags&0x8 != 0:
		return failf(tAttribution, "A6", "the record's flags %#x carry WEAK_IMPORT or NON_WEAK_DEFINITION", r.Flags)
	case r.Addend != 0:
		return failf(tAttribution, "A6", "the record's addend is %d", r.Addend)
	case r.Ordinal <= 0:
		return failf(tAttribution, "A6", "the record's ordinal %d is a special one: no library is named", r.Ordinal)
	case r.Ordinal > int64(len(img.Dylibs)):
		return failf(tParser, "A6", "the record's ordinal %d exceeds %d libraries", r.Ordinal, len(img.Dylibs))
	}
	if p := img.Dylibs[r.Ordinal-1].Path; p != libSystem {
		return failf(tAttribution, "A6", "the record's ordinal %d is %s, not %s", r.Ordinal, p, libSystem)
	}
	n := 0
	for _, d := range img.Dylibs {
		if d.Path == libSystem {
			n++
		}
	}
	if n != 1 {
		return failf(tAttribution, "A6", "%d LC_LOAD_DYLIB commands of %s, want 1", n, libSystem)
	}
	return nil // A7: the record's segment and section are recorded with it
}

// symtab is A8.
func symtab(b []byte, img *image) *fail {
	n, ok1 := mul(uint64(img.nsyms), 16)
	se, ok2 := add(uint64(img.symoff), n)
	te, _ := add(uint64(img.stroff), uint64(img.strsize))
	if !ok1 || !ok2 || se > uint64(len(b)) || te > uint64(len(b)) {
		return failf(tParser, "A8", "the symbol table [%d, %d) or its strings [%d, %d) run past the file", img.symoff, se, img.stroff, te)
	}
	str := b[img.stroff:te]
	found := 0
	for k := uint64(0); k < uint64(img.nsyms); k++ {
		x := b[uint64(img.symoff)+16*k:]
		strx, typ, desc := le.Uint32(x), x[4], le.Uint16(x[6:])
		if typ&0xe0 != 0 || typ&0x0e != 0 || typ&0x01 == 0 {
			continue // a stab, defined, or not external
		}
		if uint64(strx) >= uint64(len(str)) {
			return failf(tParser, "A8", "nlist %d: n_strx %d outside the strings", k, strx)
		}
		name, ok := cstr(str[strx:])
		if !ok {
			return failf(tParser, "A8", "nlist %d: its name has no NUL", k)
		}
		if name == target {
			found++
			d := desc
			img.NDesc = &d
		}
	}
	if found != 1 {
		return failf(tContradiction, "A8", "%d undefined external nlist entries named %s, want 1", found, target)
	}
	return nil
}

// tool is one of Apple's tools' outputs and its exit status.
type tool struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	out    []byte
}

// lines splits a tool's output (A9): it ends with a newline, which ends
// the last line; no empty line; not empty.
func lines(t tool) ([]string, *fail) {
	b := t.out
	switch {
	case len(b) == 0:
		return nil, failf(tParser, "A9", "%s: the output is empty", t.Name)
	case b[len(b)-1] != '\n':
		return nil, failf(tParser, "A9", "%s: the output does not end with a newline", t.Name)
	}
	ls := strings.Split(string(b[:len(b)-1]), "\n")
	for i, l := range ls {
		if l == "" {
			return nil, failf(tParser, "A9", "%s: line %d is empty", t.Name, i+1)
		}
	}
	return ls, nil
}

var (
	reNum      = regexp.MustCompile(`^[0-9]+$`)
	reInt      = regexp.MustCompile(`^-?[0-9]+$`)
	reHex8     = regexp.MustCompile(`^0x[0-9a-f]{8}$`)
	reHex2     = regexp.MustCompile(`^0x[0-9a-f]{2}$`)
	reLoad     = regexp.MustCompile(`^Load command ([0-9]+)$`)
	reCmd      = regexp.MustCompile(`^ *cmd (\S+)$`)
	reName     = regexp.MustCompile(`^ *name (.+) \(offset ([0-9]+)\)$`)
	reBadName  = regexp.MustCompile(`^ *name ?\(bad offset`)
	reL        = regexp.MustCompile(`^\t(.+) \(compatibility version ([0-9]+)\.([0-9]+)\.([0-9]+), current version ([0-9]+)\.([0-9]+)\.([0-9]+)(, (weak|reexport|upward|lazy))?\)$`)
	reNmTarget = regexp.MustCompile(`^ *\(undefined\) external _waitid( \(from ([^)]+)\))?$`)
)

const hHeader = "      magic  cputype cpusubtype  caps    filetype ncmds sizeofcmds      flags"

// tools is A9: each tool exited 0; otool -h, -hv, -l, -L and nm -m parse
// by their grammar and agree with the decoder.
func tools(img *image, bin, arch string, h, hv, l, L, nm tool) *fail {
	for _, t := range []tool{h, hv, l, L, nm} {
		if t.Status != 0 {
			return failf(tTool, "A9", "%s exited %d", t.Name, t.Status)
		}
	}
	file := bin + ":"
	// otool -h.
	hl, f := lines(h)
	if f != nil {
		return f
	}
	switch {
	case len(hl) != 4:
		return failf(tParser, "A9", "otool -h: %d lines, want 4", len(hl))
	case hl[0] != file:
		return failf(tParser, "A9", "otool -h: the first line %q is not %q", hl[0], file)
	case hl[1] != "Mach header" || hl[2] != hHeader:
		return failf(tParser, "A9", "otool -h: lines 2 and 3 are %q and %q", hl[1], hl[2])
	}
	num := hl[3]
	if !strings.HasPrefix(num, " ") || strings.HasSuffix(num, " ") || strings.ContainsRune(num, '\t') {
		return failf(tParser, "A9", "otool -h: the numbers' line %q is not of its printed shape", num)
	}
	fs := strings.FieldsFunc(num, func(r rune) bool { return r == ' ' })
	if len(fs) != 8 || !reHex8.MatchString(fs[0]) || !reInt.MatchString(fs[1]) || !reInt.MatchString(fs[2]) || !reHex2.MatchString(fs[3]) ||
		!reNum.MatchString(fs[4]) || !reNum.MatchString(fs[5]) || !reNum.MatchString(fs[6]) || !reHex8.MatchString(fs[7]) {
		return failf(tParser, "A9", "otool -h: the numbers' line %q has %d fields, not the 8 of its printed shape", num, len(fs))
	}
	want := []string{"0xfeedfacf", strconv.Itoa(int(int32(cpus[arch]))), "", "", "2", strconv.FormatUint(uint64(img.Ncmds), 10), strconv.FormatUint(uint64(img.Sizeofcmds), 10), fmt.Sprintf("0x%08x", img.Flags)}
	for i, w := range want {
		if w != "" && fs[i] != w {
			return failf(tContradiction, "A9", "otool -h: field %d is %s, the decoder's %s", i+1, fs[i], w)
		}
	}
	// otool -hv: recorded only.
	hvl, f := lines(hv)
	if f != nil {
		return f
	}
	if hvl[0] != file {
		return failf(tParser, "A9", "otool -hv: the first line %q is not %q", hvl[0], file)
	}
	// otool -l.
	ll, f := lines(l)
	if f != nil {
		return f
	}
	if ll[0] != file {
		return failf(tParser, "A9", "otool -l: the first line %q is not %q", ll[0], file)
	}
	var starts []int
	for i := 1; i < len(ll); i++ {
		if m := reLoad.FindStringSubmatch(ll[i]); m != nil {
			if m[1] != strconv.Itoa(len(starts)) {
				return failf(tParser, "A9", "otool -l: %q where Load command %d was due", ll[i], len(starts))
			}
			starts = append(starts, i)
		}
	}
	switch {
	case len(starts) == 0 || starts[0] != 1:
		return failf(tParser, "A9", "otool -l: no Load command 0 right after the file's line")
	case len(starts) != int(img.Ncmds):
		return failf(tParser, "A9", "otool -l: %d blocks, otool -h's ncmds %d", len(starts), img.Ncmds)
	}
	var cmds, paths []string
	var info [10]string
	for k, s := range starts {
		e := len(ll)
		if k+1 < len(starts) {
			e = starts[k+1]
		}
		blk := ll[s+1 : e]
		if len(blk) == 0 {
			return failf(tParser, "A9", "otool -l: block %d is empty", k)
		}
		m := reCmd.FindStringSubmatch(blk[0])
		if m == nil {
			return failf(tParser, "A9", "otool -l: block %d begins %q, not its cmd", k, blk[0])
		}
		cmds = append(cmds, m[1])
		switch m[1] {
		case "LC_LOAD_DYLIB", "LC_LOAD_WEAK_DYLIB", "LC_REEXPORT_DYLIB", "LC_LAZY_LOAD_DYLIB", "LC_LOAD_UPWARD_DYLIB":
			var got []string
			for _, x := range blk {
				if reBadName.MatchString(x) {
					return failf(tParser, "A9", "otool -l: block %d: %q", k, x)
				}
				if n := reName.FindStringSubmatch(x); n != nil {
					got = append(got, n[1])
				}
			}
			if len(got) != 1 {
				return failf(tParser, "A9", "otool -l: block %d (%s) has %d name lines, want 1", k, m[1], len(got))
			}
			paths = append(paths, got[0])
		case "LC_DYLD_INFO_ONLY":
			for j, name := range infoNames {
				re := regexp.MustCompile(`^ *` + name + ` (.*)$`)
				var got []string
				for _, x := range blk {
					if n := re.FindStringSubmatch(x); n != nil {
						got = append(got, n[1])
					}
				}
				if len(got) != 1 || !reNum.MatchString(got[0]) {
					return failf(tParser, "A9", "otool -l: LC_DYLD_INFO_ONLY's %s: %q, want one decimal line", name, got)
				}
				info[j] = got[0]
			}
		}
	}
	if strings.Join(cmds, " ") != strings.Join(img.Cmds, " ") {
		return failf(tContradiction, "A9", "otool -l's commands %v, the decoder's %v", cmds, img.Cmds)
	}
	var dpaths []string
	for _, d := range img.Dylibs {
		dpaths = append(dpaths, d.Path)
	}
	if strings.Join(paths, "\n") != strings.Join(dpaths, "\n") {
		return failf(tContradiction, "A9", "otool -l's libraries %q, the decoder's %q", paths, dpaths)
	}
	for j, name := range infoNames {
		if info[j] != strconv.FormatUint(uint64(img.Info[j]), 10) {
			return failf(tContradiction, "A9", "otool -l's %s %s, the decoder's %d", name, info[j], img.Info[j])
		}
	}
	// otool -L.
	Ll, f := lines(L)
	if f != nil {
		return f
	}
	if Ll[0] != file {
		return failf(tParser, "A9", "otool -L: the first line %q is not %q", Ll[0], file)
	}
	var lpaths []string
	for _, x := range Ll[1:] {
		m := reL.FindStringSubmatch(x)
		switch {
		case m == nil:
			return failf(tParser, "A9", "otool -L: %q is not a library's line", x)
		case m[9] != "":
			return failf(tContradiction, "A9", "otool -L: %q is %s, A3 allows LC_LOAD_DYLIB only", x, m[9])
		}
		lpaths = append(lpaths, m[1])
	}
	if strings.Join(lpaths, "\n") != strings.Join(dpaths, "\n") {
		return failf(tContradiction, "A9", "otool -L's libraries %q, the decoder's %q", lpaths, dpaths)
	}
	// nm -m.
	ml, f := lines(nm)
	if f != nil {
		return f
	}
	var tl []string
	for _, x := range ml {
		for _, w := range strings.Fields(x) {
			if w == target {
				tl = append(tl, x)
				break
			}
		}
	}
	if len(tl) != 1 {
		return failf(tContradiction, "A9", "nm -m: %d lines name %s, want 1", len(tl), target)
	}
	n := reNmTarget.FindStringSubmatch(tl[0])
	switch {
	case n == nil:
		return failf(tContradiction, "A9", "nm -m: %q is not an undefined external %s", tl[0], target)
	case n[1] != "" && n[2] != "libSystem":
		return failf(tContradiction, "A9", "nm -m: %q names %s, not libSystem", tl[0], n[2])
	}
	return nil
}

// result is the WP25P5BIND line.
type result struct {
	Format string `json:"format"`
	Go     string `json:"go"`
	Bin    string `json:"bin"`
	Arch   string `json:"arch"`
	OK     bool   `json:"ok"`
	*fail
	Image *image `json:"image"`
	Tools []tool `json:"tools"`
}

// check runs A1 to A9.
func check(b []byte, bin, arch string, size int64, sha string, ts [5]tool) (*image, *fail) {
	img, f := decode(b, arch, size, sha)
	if f != nil {
		return img, f
	}
	return img, tools(img, bin, arch, ts[0], ts[1], ts[2], ts[3], ts[4])
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("p5bind", flag.ContinueOnError)
	fl.SetOutput(stderr)
	bin := fl.String("bin", "", "the test binary")
	arch := fl.String("arch", "", "the expected architecture: arm64 or x86_64")
	size := fl.Int64("size", -1, "its size as the step recorded it")
	sha := fl.String("sha256", "", "its SHA-256 as the step recorded it")
	names := []string{"otool -h", "otool -hv", "otool -l", "otool -L", "nm -m"}
	flags := []string{"otool-h", "otool-hv", "otool-l", "otool-L", "nm-m"}
	var files [5]*string
	var stats [5]*int
	for i, n := range flags {
		files[i] = fl.String(n, "", "the saved output of "+names[i])
		stats[i] = fl.Int(n+"-status", -1, "the exit status of "+names[i])
	}
	if err := fl.Parse(args); err != nil {
		return 2
	}
	usage := *bin == "" || cpus[*arch] == 0 || *size < 0 || len(*sha) != 64 || fl.NArg() != 0
	for i := range files {
		usage = usage || *files[i] == "" || *stats[i] < 0
	}
	if usage {
		fmt.Fprintln(stderr, "p5bind: usage: -bin, -arch arm64|x86_64, -size, -sha256 and each tool's output and -status are required")
		return 2
	}
	res := result{Format: format, Go: runtime.Version(), Bin: *bin, Arch: *arch}
	b, err := os.ReadFile(*bin)
	var f *fail
	var img *image
	if err != nil {
		f = failf(tTool, "A1", "reading the binary: %v", err)
	} else {
		var ts [5]tool
		for i := range ts {
			ts[i] = tool{Name: names[i], Status: *stats[i]}
			out, err := os.ReadFile(*files[i])
			if err != nil {
				f = failf(tTool, "A9", "reading %s's output: %v", names[i], err)
				break
			}
			ts[i].out = out
			res.Tools = append(res.Tools, ts[i])
		}
		if f == nil {
			img, f = check(b, *bin, *arch, *size, *sha, ts)
		}
	}
	res.Image, res.OK, res.fail = img, f == nil, f
	j, err := json.Marshal(res)
	if err != nil {
		fmt.Fprintln(stderr, "p5bind:", err)
		return 2
	}
	fmt.Fprintln(stdout, "WP25P5BIND "+string(j))
	if img != nil {
		fmt.Fprintf(stdout, "p5bind: %d bytes, SHA-256 %s, cputype %d, flags %#x %v\n", img.Size, img.SHA256, img.CPU, img.Flags, img.FlagNames)
		fmt.Fprintf(stdout, "p5bind: libraries %v; bind records %d\n", img.Dylibs, img.Total)
		if img.Target != nil {
			fmt.Fprintf(stdout, "p5bind: %s: ordinal %d, %s,%s offset %#x addr %#x, flags %#x, type %d, addend %d\n",
				target, img.Target.Ordinal, img.Target.Seg, img.Target.Sect, img.Target.Offset, img.Target.Addr, img.Target.Flags, img.Target.Type, img.Target.Addend)
		}
	}
	if f != nil {
		fmt.Fprintf(stdout, "p5bind: %s at %s: %s\n", f.Class, f.Rule, f.Reason)
		fmt.Fprintf(stdout, "WP25P5BIND-VERDICT %s %s\n", f.Class, f.Rule)
		return 1
	}
	fmt.Fprintln(stdout, "WP25P5BIND-VERDICT ok -")
	return 0
}
