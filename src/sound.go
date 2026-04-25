package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	winmm             = syscall.NewLazyDLL("winmm.dll")
	procMciSendString = winmm.NewProc("mciSendStringW")
	soundSeq          atomic.Int64
)

func playSound(file string) {
	if cfg.Mute {
		return
	}
	go func() {
		// LockOSThread pins this goroutine to one OS thread for the entire
		// duration — MCI (mpegvideo) has COM apartment requirements and
		// behaves unreliably when open/play/close span different threads.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		id := soundSeq.Add(1)
		alias := fmt.Sprintf("snd%d", id)
		if mciExec(fmt.Sprintf(`open "%s" alias %s`, file, alias)) != 0 {
			return
		}
		mciExec(fmt.Sprintf(`play %s from 0 wait`, alias))
		mciExec(fmt.Sprintf(`close %s`, alias))
	}()
}

// mciExec sends an MCI command and returns the error code (0 = success).
func mciExec(cmd string) uintptr {
	ptr, err := syscall.UTF16PtrFromString(cmd)
	if err != nil {
		return 1
	}
	ret, _, _ := procMciSendString.Call(uintptr(unsafe.Pointer(ptr)), 0, 0, 0)
	if ret != 0 {
		fmt.Printf("[sound] MCI error %d: %s\n", ret, cmd)
	}
	return ret
}
