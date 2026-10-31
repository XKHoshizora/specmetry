// The trampoline of libSystem's waitid (docs/wp2.5-prototype.md §16.3),
// written as x/sys writes kevent's (unix/zsyscall_darwin_arm64.s).

#include "textflag.h"

TEXT libc_waitid_trampoline<>(SB),NOSPLIT,$0-0
	JMP	libc_waitid(SB)
GLOBL	·libc_waitid_trampoline_addr(SB), RODATA, $8
DATA	·libc_waitid_trampoline_addr(SB)/8, $libc_waitid_trampoline<>(SB)
