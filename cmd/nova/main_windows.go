//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"
	"unsafe"

	g "novaarchipel/game"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")

	pRegisterClassExW   = user32.NewProc("RegisterClassExW")
	pCreateWindowExW    = user32.NewProc("CreateWindowExW")
	pDefWindowProcW     = user32.NewProc("DefWindowProcW")
	pPeekMessageW       = user32.NewProc("PeekMessageW")
	pTranslateMessage   = user32.NewProc("TranslateMessage")
	pDispatchMessageW   = user32.NewProc("DispatchMessageW")
	pPostQuitMessage    = user32.NewProc("PostQuitMessage")
	pGetDC              = user32.NewProc("GetDC")
	pReleaseDC          = user32.NewProc("ReleaseDC")
	pLoadCursorW        = user32.NewProc("LoadCursorW")
	pShowWindow         = user32.NewProc("ShowWindow")
	pGetClientRect      = user32.NewProc("GetClientRect")
	pSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	pSetWindowLongPtrW  = user32.NewProc("SetWindowLongPtrW")
	pSetWindowPos       = user32.NewProc("SetWindowPos")
	pMonitorFromWindow  = user32.NewProc("MonitorFromWindow")
	pGetMonitorInfoW    = user32.NewProc("GetMonitorInfoW")
	pSetCapture         = user32.NewProc("SetCapture")
	pReleaseCapture     = user32.NewProc("ReleaseCapture")
	pDestroyWindow      = user32.NewProc("DestroyWindow")
	pMessageBoxW        = user32.NewProc("MessageBoxW")
	pSendMessageW       = user32.NewProc("SendMessageW")
	pCreateIconIndirect = user32.NewProc("CreateIconIndirect")
	pStretchDIBits      = gdi32.NewProc("StretchDIBits")
	pChoosePixelFormat  = gdi32.NewProc("ChoosePixelFormat")
	pSetPixelFormat     = gdi32.NewProc("SetPixelFormat")
	pSwapBuffers        = gdi32.NewProc("SwapBuffers")
	opengl32            = syscall.NewLazyDLL("opengl32.dll")
	pWglCreateContext   = opengl32.NewProc("wglCreateContext")
	pWglMakeCurrent     = opengl32.NewProc("wglMakeCurrent")
	pWglDeleteContext   = opengl32.NewProc("wglDeleteContext")
	pWglGetProcAddress  = opengl32.NewProc("wglGetProcAddress")
	pCreateBitmap       = gdi32.NewProc("CreateBitmap")
	pGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	pTimeBeginPeriod    = winmm.NewProc("timeBeginPeriod")
)

const (
	WM_DESTROY       = 0x0002
	WM_SIZE          = 0x0005
	WM_CLOSE         = 0x0010
	WM_QUIT          = 0x0012
	WM_ERASEBKGND    = 0x0014
	WM_SETICON       = 0x0080
	WM_KEYDOWN       = 0x0100
	WM_KEYUP         = 0x0101
	WM_SYSKEYDOWN    = 0x0104
	WM_SYSKEYUP      = 0x0105
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_RBUTTONDOWN   = 0x0204
	WM_RBUTTONUP     = 0x0205
	WM_MBUTTONDOWN   = 0x0207
	WM_MBUTTONUP     = 0x0208
	WM_MOUSEWHEEL    = 0x020A
	WM_KILLFOCUS     = 0x0008
	PM_REMOVE        = 0x0001
	WS_OVERLAPPEDWIN = 0x00CF0000
	WS_POPUP         = 0x80000000
	WS_VISIBLE       = 0x10000000
	CS_HREDRAW       = 0x0002
	CS_VREDRAW       = 0x0001
	CS_OWNDC         = 0x0020
	SW_SHOWMAXIMIZED = 3
	IDC_ARROW        = 32512
	SRCCOPY          = 0x00CC0020
	GWL_STYLE        = ^uintptr(15) // -16
	SWP_FRAMECHANGED = 0x0020
	SWP_NOZORDER     = 0x0004
	SWP_NOMOVE       = 0x0002
	SWP_NOSIZE       = 0x0001
	MONITOR_NEAREST  = 2
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type point struct{ X, Y int32 }
type msgT struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}
type rect struct{ L, T, R, B int32 }
type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}
type bmiHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}
type pixelFormatDesc struct {
	Size        uint16
	Version     uint16
	Flags       uint32
	PixelType   uint8
	ColorBits   uint8
	Pad1        [11]uint8
	AccumBits   [4]uint8
	DepthBits   uint8
	StencilBits uint8
	AuxBuffers  uint8
	LayerType   uint8
	Reserved    uint8
	LayerMask   uint32
	VisibleMask uint32
	DamageMask  uint32
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

var (
	hwnd       uintptr
	hdc        uintptr
	canvas     *g.Canvas
	scale      = 1.0
	fullscreen = false
	vsyncOn    = -1
	swapProc   uintptr
	winRect    rect
	quitting   bool
)

func vkToKey(vk uintptr) int {
	if k := g.BoundKey(int(vk)); k != g.KNone {
		return k
	}
	switch vk {
	case 0x25:
		return g.KLeft
	case 0x27:
		return g.KRight
	case 0x26:
		return g.KUp
	case 0x28:
		return g.KDown
	case 0x1B:
		return g.KEsc
	case 0x31, 0x61:
		return g.K1
	case 0x32, 0x62:
		return g.K2
	case 0x33, 0x63:
		return g.K3
	case 0x34, 0x64:
		return g.K4
	case 0x7A:
		return g.KF11
	case 0xBB, 0x6B:
		return g.KPlus
	case 0xBD, 0x6D:
		return g.KMinus
	case 0x10:
		return g.KShift
	case 0x11:
		return g.KCtrl
	case 0x09:
		return g.KTab
	}
	return g.KNone
}

func mouseXY(l uintptr) (int, int) {
	x := int(int16(l & 0xFFFF))
	y := int(int16((l >> 16) & 0xFFFF))
	return int(float64(x) / scale), int(float64(y) / scale)
}

func wndProc(h, msg, wp, lp uintptr) uintptr {
	switch uint32(msg) {
	case WM_CLOSE:
		quitting = true
		pDestroyWindow.Call(h)
		return 0
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_SIZE:
		resize()
		return 0
	case WM_MOUSEMOVE:
		x, y := mouseXY(lp)
		g.In.MouseMove(x, y)
		return 0
	case WM_LBUTTONDOWN:
		x, y := mouseXY(lp)
		g.In.MouseMove(x, y)
		g.In.MouseButton(0, true)
		pSetCapture.Call(h)
		return 0
	case WM_LBUTTONUP:
		x, y := mouseXY(lp)
		g.In.MouseMove(x, y)
		g.In.MouseButton(0, false)
		pReleaseCapture.Call()
		return 0
	case WM_RBUTTONDOWN:
		x, y := mouseXY(lp)
		g.In.MouseMove(x, y)
		g.In.MouseButton(1, true)
		pSetCapture.Call(h)
		return 0
	case WM_RBUTTONUP:
		x, y := mouseXY(lp)
		g.In.MouseMove(x, y)
		g.In.MouseButton(1, false)
		pReleaseCapture.Call()
		return 0
	case WM_MBUTTONDOWN:
		g.In.MouseButton(2, true)
		pSetCapture.Call(h)
		return 0
	case WM_MBUTTONUP:
		g.In.MouseButton(2, false)
		pReleaseCapture.Call()
		return 0
	case WM_MOUSEWHEEL:
		d := int(int16((wp >> 16) & 0xFFFF))
		if d > 0 {
			g.In.Wheel++
		} else if d < 0 {
			g.In.Wheel--
		}
		return 0
	case WM_KEYDOWN, WM_SYSKEYDOWN:
		if wp == 0x73 && uint32(msg) == WM_SYSKEYDOWN { // Alt+F4
			break
		}
		g.In.LastVK = int(wp)
		k := vkToKey(wp)
		if k == g.KF11 {
			if (lp>>30)&1 == 0 {
				g.Opt.Fullscreen = !g.Opt.Fullscreen
			}
			return 0
		}
		g.In.Key(k, true)
		return 0
	case WM_KEYUP, WM_SYSKEYUP:
		g.In.Key(vkToKey(wp), false)
		if wp == 0x79 || wp == 0x12 { // F10/Alt kein Systemmenü
			return 0
		}
	case WM_KILLFOCUS:
		g.In = g.InputState{MX: g.In.MX, MY: g.In.MY}
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

func clientSize() (int, int) {
	var r rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return int(r.R - r.L), int(r.B - r.T)
}

func resize() {
	if hwnd == 0 {
		return
	}
	w, h := clientSize()
	if w < 64 || h < 64 {
		return
	}
	scale = g.Opt.UIScale
	if h >= 2000 {
		scale *= 1.5
	}
	if scale < 0.5 {
		scale = 1
	}
	cw, ch := int(float64(w)/scale), int(float64(h)/scale)
	if canvas == nil || canvas.W != cw || canvas.H != ch {
		canvas = g.NewCanvas(cw, ch)
	}
}

func toggleFullscreen() {
	if !fullscreen {
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&winRect)))
		mon, _, _ := pMonitorFromWindow.Call(hwnd, MONITOR_NEAREST)
		mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		pSetWindowLongPtrW.Call(hwnd, GWL_STYLE, WS_POPUP|WS_VISIBLE)
		pSetWindowPos.Call(hwnd, 0, uintptr(mi.Monitor.L), uintptr(mi.Monitor.T), uintptr(mi.Monitor.R-mi.Monitor.L), uintptr(mi.Monitor.B-mi.Monitor.T), SWP_FRAMECHANGED|SWP_NOZORDER)
		fullscreen = true
	} else {
		pSetWindowLongPtrW.Call(hwnd, GWL_STYLE, WS_OVERLAPPEDWIN|WS_VISIBLE)
		pSetWindowPos.Call(hwnd, 0, 80, 60, 1440, 860, SWP_FRAMECHANGED|SWP_NOZORDER)
		pShowWindow.Call(hwnd, SW_SHOWMAXIMIZED)
		fullscreen = false
	}
	resize()
}

func wglProc(name string) uintptr {
	cn := append([]byte(name), 0)
	p, _, _ := pWglGetProcAddress.Call(uintptr(unsafe.Pointer(&cn[0])))
	if p == 0 || p == 1 || p == 2 || p == 3 || p == ^uintptr(0) {
		return 0
	}
	return p
}

// OpenGL-Kontext anlegen (bevorzugt 3.3 Core, sonst Kompatibilitätskontext)
func initGL() error {
	hdc, _, _ = pGetDC.Call(hwnd)
	pfd := pixelFormatDesc{Version: 1, Flags: 0x4 | 0x20 | 0x1, PixelType: 0, ColorBits: 32, DepthBits: 24, StencilBits: 8}
	pfd.Size = uint16(unsafe.Sizeof(pfd))
	f, _, _ := pChoosePixelFormat.Call(hdc, uintptr(unsafe.Pointer(&pfd)))
	if f == 0 {
		return fmt.Errorf("kein passendes Pixelformat")
	}
	if r, _, _ := pSetPixelFormat.Call(hdc, f, uintptr(unsafe.Pointer(&pfd))); r == 0 {
		return fmt.Errorf("SetPixelFormat fehlgeschlagen")
	}
	rc, _, _ := pWglCreateContext.Call(hdc)
	if rc == 0 {
		return fmt.Errorf("OpenGL-Kontext konnte nicht erstellt werden")
	}
	pWglMakeCurrent.Call(hdc, rc)
	if ca := wglProc("wglCreateContextAttribsARB"); ca != 0 {
		attrs := []int32{0x2091, 3, 0x2092, 3, 0x9126, 1, 0}
		nrc, _, _ := syscall.SyscallN(ca, hdc, 0, uintptr(unsafe.Pointer(&attrs[0])))
		if nrc != 0 {
			pWglMakeCurrent.Call(hdc, nrc)
			pWglDeleteContext.Call(rc)
		}
	}
	swapProc = wglProc("wglSwapIntervalEXT")
	return nil
}

func swap() { pSwapBuffers.Call(hdc) }

func makeIcon() uintptr {
	const n = 64
	pix := g.AppIcon(n)
	color, _, _ := pCreateBitmap.Call(n, n, 1, 32, uintptr(unsafe.Pointer(&pix[0])))
	mask := make([]byte, n*n/8)
	mbm, _, _ := pCreateBitmap.Call(n, n, 1, 1, uintptr(unsafe.Pointer(&mask[0])))
	ii := iconInfo{FIcon: 1, HbmMask: mbm, HbmColor: color}
	ic, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	return ic
}

func crashLog(r interface{}) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	dir = filepath.Join(dir, "NovaArchipel")
	os.MkdirAll(dir, 0755)
	p := filepath.Join(dir, "absturz.txt")
	txt := fmt.Sprintf("Nova Archipel %s – Absturz\n%v\n\n%s", g.Version, r, debug.Stack())
	os.WriteFile(p, []byte(txt), 0644)
	t, _ := syscall.UTF16PtrFromString("Nova Archipel ist leider abgestürzt.\n\nDetails wurden gespeichert unter:\n" + p + "\n\n" + fmt.Sprint(r))
	c, _ := syscall.UTF16PtrFromString("Nova Archipel")
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
}

func init() { runtime.LockOSThread() }

func main() {
	defer func() {
		if r := recover(); r != nil {
			crashLog(r)
		}
	}()
	pSetProcessDPIAware.Call()
	g.LoadSettings()
	pTimeBeginPeriod.Call(1)
	inst, _, _ := pGetModuleHandleW.Call(0)
	cls, _ := syscall.UTF16PtrFromString("NovaArchipelWnd")
	title, _ := syscall.UTF16PtrFromString("Nova Archipel – Kolonien der Zukunft")
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	icon := makeIcon()
	wc := wndClassEx{Style: CS_HREDRAW | CS_VREDRAW | CS_OWNDC, WndProc: syscall.NewCallback(wndProc), Instance: inst, Cursor: cur, ClassName: cls, Icon: icon, IconSm: icon}
	wc.Size = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ = pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)), WS_OVERLAPPEDWIN|WS_VISIBLE,
		80, 60, 1440, 860, 0, 0, inst, 0)
	if hwnd == 0 {
		crashLog("Fenster konnte nicht erstellt werden")
		return
	}
	if icon != 0 {
		pSendMessageW.Call(hwnd, WM_SETICON, 1, icon)
		pSendMessageW.Call(hwnd, WM_SETICON, 0, icon)
	}
	pShowWindow.Call(hwnd, SW_SHOWMAXIMIZED)
	if g.Opt.Fullscreen {
		toggleFullscreen()
	}
	resize()
	if err := initGL(); err != nil {
		crashLog("Grafik konnte nicht gestartet werden: " + err.Error())
		return
	}
	renderer, err := g.NewRenderer()
	if err != nil {
		crashLog("Dein Grafiktreiber unterstützt den 3D-Renderer nicht (OpenGL 3.3 nötig).\n\n" + err.Error())
		return
	}
	g.StartAudio()
	app := g.NewApp(renderer)
	g.CleanupUpdate()
	if g.Opt.AutoUpdate {
		g.CheckUpdate()
	}
	last := time.Now()
	lastUI := g.Opt.UIScale
	var m msgT
	for !quitting {
		for {
			r, _, _ := pPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, PM_REMOVE)
			if r == 0 {
				break
			}
			if m.Message == WM_QUIT {
				quitting = true
				break
			}
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
		if quitting {
			break
		}
		now := time.Now()
		dt := now.Sub(last).Seconds()
		last = now
		if dt > 0.1 {
			dt = 0.1
		}
		if g.Opt.Fullscreen != fullscreen {
			toggleFullscreen()
		}
		v := 0
		if g.Opt.VSync {
			v = 1
		}
		if v != vsyncOn && swapProc != 0 {
			syscall.SyscallN(swapProc, uintptr(v))
			vsyncOn = v
		}
		if g.Opt.UIScale != lastUI {
			lastUI = g.Opt.UIScale
			resize()
		}
		cw, ch := clientSize()
		if canvas != nil && cw > 64 && ch > 64 {
			if err := renderer.Resize(cw, ch); err != nil {
				crashLog(err.Error())
				return
			}
			app.Frame(canvas, dt)
			app.Render(canvas, dt)
			swap()
		}
		g.In.EndFrame()
		if g.In.Quit {
			quitting = true
			pDestroyWindow.Call(hwnd)
		}
		if el := time.Since(now); el < 4*time.Millisecond {
			time.Sleep(4*time.Millisecond - el)
		}
	}
	if app.Cur != nil && app.Cur.Time > 30 {
		app.Cur.SaveAuto()
	}
	g.SaveSettings()
}
