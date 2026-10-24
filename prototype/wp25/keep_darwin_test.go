//go:build darwin

package wp25

// fionread asks how many bytes wait in a pipe: FIONREAD, _IOR('f', 127,
// int) in <sys/filio.h>; golang.org/x/sys/unix does not name it.
const fionread = 0x4004667f
