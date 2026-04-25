package main

import (
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	winmm             = syscall.NewLazyDLL("winmm.dll")
	procMciSendString = winmm.NewProc("mciSendStringW")
	soundSeq          atomic.Int64
)

// playSound plays an MP3 file asynchronously via Windows MCI.
// Silently does nothing if the file doesn't exist.
func playSound(file string) {
	if cfg.Mute {
		return
	}
	go func() {
		id := soundSeq.Add(1)
		alias := fmt.Sprintf("snd%d", id)
		mciExec(fmt.Sprintf(`open "%s" type mpegvideo alias %s`, file, alias))
		mciExec(fmt.Sprintf(`play %s from 0 wait`, alias))
		mciExec(fmt.Sprintf(`close %s`, alias))
	}()
}

func mciExec(cmd string) {
	ptr, err := syscall.UTF16PtrFromString(cmd)
	if err != nil {
		return
	}
	procMciSendString.Call(uintptr(unsafe.Pointer(ptr)), 0, 0, 0)
}
