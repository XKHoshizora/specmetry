package gitx

import (
	"fmt"

	"golang.org/x/sys/windows"
)

var (
	ntSuspendConsumer = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtSuspendProcess")
	ntResumeConsumer  = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess")
)

func changeConsumerState(pid int, proc *windows.LazyProc) error {
	h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, uint32(pid))
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	status, _, _ := proc.Call(uintptr(h))
	if int32(status) < 0 {
		return fmt.Errorf("cannot change test Git process state: %x", status)
	}
	return nil
}
func pauseConsumer(pid int) error  { return changeConsumerState(pid, ntSuspendConsumer) }
func resumeConsumer(pid int) error { return changeConsumerState(pid, ntResumeConsumer) }

func watchConsumerExit(pid int) (func() error, func(), error) {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return nil, nil, err
	}
	wait := func() error {
		state, err := windows.WaitForSingleObject(h, 30000)
		if err != nil {
			return err
		}
		if state != windows.WAIT_OBJECT_0 {
			return fmt.Errorf("Git exit was not observed: %d", state)
		}
		return nil
	}
	return wait, func() { _ = windows.CloseHandle(h) }, nil
}
