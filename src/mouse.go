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
	inputKeyboard  = 1
	mouseEventMove = 0x0001
	keyEventKeyUp  = 0x0002

	// stunReleaseTag marks synthetic KEYUP events sent by releaseAllKeys so the
	// hook can identify and pass them through even while stunned is true.
	stunReleaseTag uintptr = 0xCAFEBABE
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procSendInput           = user32.NewProc("SendInput")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
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

// keyInputEvent mirrors Windows INPUT struct for keyboard on 64-bit (40 bytes total).
// type(4) + pad(4) + wVk(2) + wScan(2) + dwFlags(4) + time(4) + pad(4) + extraInfo(8) + pad(8) = 40 bytes
type keyInputEvent struct {
	inputType   uint32
	_           [4]byte
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	_           [4]byte
	dwExtraInfo uintptr
	_           [8]byte // pad union to match MOUSEINPUT size
}

type winMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      [8]byte
}

// releaseAllKeys sends a synthetic KEYUP for every key currently held down,
// preventing keys pressed before the stun from staying "stuck" in games.
func releaseAllKeys() {
	for vk := uintptr(0); vk < 256; vk++ {
		state, _, _ := procGetAsyncKeyState.Call(vk)
		if state&0x8000 == 0 {
			continue
		}
		fmt.Printf("[keyboard] Release key %d\n", vk)

		ev := keyInputEvent{
			inputType:   inputKeyboard,
			wVk:         uint16(vk),
			dwFlags:     keyEventKeyUp,
			dwExtraInfo: stunReleaseTag,
		}
		procSendInput.Call(1, uintptr(unsafe.Pointer(&ev)), unsafe.Sizeof(ev))
	}
}

func activateStun() {
	if !stunned.CompareAndSwap(false, true) {
		return // already active
	}
	releaseAllKeys()
	fmt.Printf("[stun] клавиши заблокированы на %s\n", cfgStunTime)
	playSound("sounds/stun.mp3")
	time.AfterFunc(cfgStunTime, func() {
		stunned.Store(false)
		fmt.Println("[stun] клавиши разблокированы")
	})
}

func moveMouse180() {
	playSound("sounds/180.mp3")
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

	// Block all key events during stun, except our own synthetic KEYUP releases.
	if stunned.Load() {
		ks := (*kbdllHookStruct)(unsafe.Pointer(lParam))
		if ks.dwExtraInfo == stunReleaseTag {
			r, _, _ := procCallNextHookEx.Call(hhook, uintptr(nCode), wParam, lParam)
			return r
		}
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
