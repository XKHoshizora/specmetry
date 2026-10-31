//go:build darwin

package e2waitid

import (
	"syscall"
	"unsafe"
)

// The runtime's syscall6 (syscall/syscall_darwin.go): it calls fn on the
// system stack and gives errno(r1, e), which is e only when int32(r1) is
// -1 and 0 otherwise (§16.2). The same declaration as x/sys's
// unix/syscall_darwin_libSystem.go.
//
//go:linkname syscall_syscall6 syscall.syscall6
func syscall_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

// libc_waitid_trampoline_addr is the address of the trampoline
// (waitid_darwin_*.s), which jumps to libSystem's waitid.
var libc_waitid_trampoline_addr uintptr

//go:cgo_import_dynamic libc_waitid waitid "/usr/lib/libSystem.B.dylib"

// Waitid calls int waitid(idtype_t, id_t, siginfo_t *, int) once. info
// points to the caller's buffer, which the caller keeps alive across the
// call (runtime.KeepAlive). It returns r1 as the runtime gave it and the
// filtered errno; the caller reads int32(r1) with err (§16.7).
func Waitid(idtype, id uint32, info unsafe.Pointer, options uint32) (r1 uintptr, err syscall.Errno) {
	r1, _, err = syscall_syscall6(libc_waitid_trampoline_addr, uintptr(idtype), uintptr(id), uintptr(info), uintptr(options), 0, 0)
	return r1, err
}
