package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	whKeyboardLL   = 13
	wmKeyDown      = 0x0100
	wmSysKeyDown   = 0x0104
	inputMouse     = 0
	mouseEventMove = 0x0001
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procSendInput           = user32.NewProc("SendInput")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")

	hhook   uintptr
	stunned atomic.Bool
)

type kbdllHookStruct struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

// mouseInputEvent mirrors Windows INPUT struct for mouse on 64-bit:
// type(4) + pad(4) + dx(4) + dy(4) + mouseData(4) + dwFlags(4) + time(4) + pad(4) + extraInfo(8) = 40 bytes
type mouseInputEvent struct {
	inputType   uint32
	_           [4]byte
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	_           [4]byte
	dwExtraInfo uintptr
}

type winMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      [8]byte
}

func activateStun() {
	if !stunned.CompareAndSwap(false, true) {
		return // already active
	}
	fmt.Printf("[stun] клавиши заблокированы на %s\n", cfgStunTime)
	time.AfterFunc(cfgStunTime, func() {
		stunned.Store(false)
		fmt.Println("[stun] клавиши разблокированы")
	})
}

func moveMouse180() {
	ev := mouseInputEvent{
		inputType: inputMouse,
		dx:        cfgTurnDist,
		dwFlags:   mouseEventMove,
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&ev)), unsafe.Sizeof(ev))
}

func hookCallback(nCode int, wParam, lParam uintptr) uintptr {
	if nCode < 0 {
		r, _, _ := procCallNextHookEx.Call(hhook, uintptr(nCode), wParam, lParam)
		return r
	}

	// Block all key events during stun
	if stunned.Load() {
		return 1
	}

	if wParam == wmKeyDown || wParam == wmSysKeyDown {
		ks := (*kbdllHookStruct)(unsafe.Pointer(lParam))
		if cfgTurnKey != 0 && ks.vkCode == cfgTurnKey {
			fmt.Printf("[keyboard] %s → 180°\n", cfg.Debug180Key)
			moveMouse180()
		} else if cfgStunKey != 0 && ks.vkCode == cfgStunKey {
			fmt.Printf("[keyboard] %s → стан\n", cfg.DebugStunKey)
			activateStun()
		}
	}

	r, _, _ := procCallNextHookEx.Call(hhook, uintptr(nCode), wParam, lParam)
	return r
}

func runKeyboardHook() {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	cb := syscall.NewCallback(hookCallback)
	h, _, err := procSetWindowsHookEx.Call(whKeyboardLL, cb, hInst, 0)
	if h == 0 {
		fmt.Fprintln(os.Stderr, "SetWindowsHookEx failed:", err)
		os.Exit(1)
	}
	hhook = h

	var m winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 || r == ^uintptr(0) {
			break
		}
	}
	procUnhookWindowsHookEx.Call(hhook)
}

func stopKeyboardHook() {
	procUnhookWindowsHookEx.Call(hhook)
}
