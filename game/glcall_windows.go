//go:build windows

package game

import (
	"syscall"
	"unsafe"
)

var (
	opengl32           = syscall.NewLazyDLL("opengl32.dll")
	pWglGetProcAddress = opengl32.NewProc("wglGetProcAddress")
)

func glGetProc(name string) uintptr {
	cn := append([]byte(name), 0)
	p, _, _ := pWglGetProcAddress.Call(uintptr(unsafe.Pointer(&cn[0])))
	if p == 0 || p == 1 || p == 2 || p == 3 || p == ^uintptr(0) {
		pr := opengl32.NewProc(name)
		if pr.Find() != nil {
			return 0
		}
		return pr.Addr()
	}
	return p
}

func glCall(f uintptr, a ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(f, a...)
	return r
}
