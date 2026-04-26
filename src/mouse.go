package main

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	whKeyboardLL        = 13
	whMouseLL           = 14
	wmKeyDown           = 0x0100
	wmSysKeyDown        = 0x0104
	wmMouseMove         = 0x0200
	inputMouse          = 0
	inputKeyboard       = 1
	mouseEventMove      = 0x0001
	mouseEventLeftDown  = 0x0002
	mouseEventLeftUp    = 0x0004
	mouseEventRightDown = 0x0008
	mouseEventRightUp   = 0x0010
	keyEventKeyUp       = 0x0002

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
	procBlockInput          = user32.NewProc("BlockInput")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetCursorPos        = user32.NewProc("SetCursorPos")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowRect       = user32.NewProc("GetWindowRect")

	hhook        uintptr
	hmousehook   uintptr
	stunned      atomic.Bool
	blockInputCh = make(chan bool, 1)
)

func init() {
	go func() {
		runtime.LockOSThread()
		// Probe: BlockInput(false) is safe to call anytime and tells us if we have admin.
		ret, _, _ := procBlockInput.Call(0)
		if ret != 0 {
			fmt.Println("[stun] режим администратора: Raw Input мыши тоже будет заблокирован")
		} else {
			fmt.Println("[stun] без администратора: блокируются только Win32-клики (WH_MOUSE_LL)")
		}
		for block := range blockInputCh {
			val := uintptr(0)
			if block {
				val = 1
			}
			procBlockInput.Call(val)
		}
	}()
}

type point struct{ x, y int32 }
type rect struct{ left, top, right, bottom int32 }

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
	blockInputCh <- true
	fmt.Printf("[stun] клавиши и кнопки мыши заблокированы на %s\n", cfgStunTime)
	playSound("sounds/stun.mp3")
	time.AfterFunc(cfgStunTime, func() {
		blockInputCh <- false
		stunned.Store(false)
		fmt.Println("[stun] клавиши и кнопки мыши разблокированы")
	})
}

func moveMouse180() {
	playSound("sounds/180.mp3")
	go func() {
		send := func(flags uint32, dx int32) {
			ev := mouseInputEvent{inputType: inputMouse, dx: dx, dwFlags: flags}
			procSendInput.Call(1, uintptr(unsafe.Pointer(&ev)), unsafe.Sizeof(ev))
		}
		if cfgTurnModDown != 0 {
			var p point
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))

			hwnd, _, _ := procGetForegroundWindow.Call()
			var r rect
			procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
			procSetCursorPos.Call(uintptr(r.left), uintptr(r.top))

			send(cfgTurnModDown, 0)
			time.Sleep(32 * time.Millisecond)
			const (
				turnDuration = 100 * time.Millisecond
				stepInterval = 16 * time.Millisecond
				turnSteps    = int(turnDuration / stepInterval)
			)
			dxPerStep := float64(cfgTurnDist) / float64(turnSteps)
			var acc float64
			for i := 0; i < turnSteps; i++ {
				acc += dxPerStep
				dx := int32(acc)
				acc -= float64(dx)
				send(mouseEventMove, dx)
				time.Sleep(stepInterval)
			}
			send(cfgTurnModUp, 0)
			time.Sleep(32 * time.Millisecond)
			procSetCursorPos.Call(uintptr(p.x), uintptr(p.y))
		} else {
			send(mouseEventMove, cfgTurnDist)
		}
	}()
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

func mouseHookCallback(nCode int, wParam, lParam uintptr) uintptr {
	// Block all mouse events except movement during stun.
	if nCode >= 0 && stunned.Load() && wParam != wmMouseMove {
		return 1
	}
	r, _, _ := procCallNextHookEx.Call(hmousehook, uintptr(nCode), wParam, lParam)
	return r
}

func runKeyboardHook() {
	hInst, _, _ := procGetModuleHandleW.Call(0)

	kbCb := syscall.NewCallback(hookCallback)
	h, _, err := procSetWindowsHookEx.Call(whKeyboardLL, kbCb, hInst, 0)
	if h == 0 {
		fmt.Fprintln(os.Stderr, "SetWindowsHookEx (keyboard) failed:", err)
		os.Exit(1)
	}
	hhook = h

	mCb := syscall.NewCallback(mouseHookCallback)
	mh, _, err := procSetWindowsHookEx.Call(whMouseLL, mCb, hInst, 0)
	if mh == 0 {
		fmt.Fprintln(os.Stderr, "SetWindowsHookEx (mouse) failed:", err)
		os.Exit(1)
	}
	hmousehook = mh

	var m winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 || r == ^uintptr(0) {
			break
		}
	}
	procUnhookWindowsHookEx.Call(hhook)
	procUnhookWindowsHookEx.Call(hmousehook)
}

func stopKeyboardHook() {
	procUnhookWindowsHookEx.Call(hhook)
	procUnhookWindowsHookEx.Call(hmousehook)
}
