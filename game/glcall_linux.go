//go:build linux && cgo

package game

/*
#cgo LDFLAGS: -l:libEGL.so.1
#include <stdint.h>
#include <stdlib.h>
typedef void* EGLDisplay; typedef void* EGLContext; typedef void* EGLConfig; typedef int EGLint; typedef unsigned int EGLBoolean; typedef unsigned int EGLenum;
extern void* eglGetProcAddress(const char*);
extern EGLDisplay eglGetPlatformDisplay(EGLenum, void*, const intptr_t*);
extern EGLBoolean eglInitialize(EGLDisplay, EGLint*, EGLint*);
extern EGLBoolean eglBindAPI(EGLenum);
extern EGLBoolean eglChooseConfig(EGLDisplay, const EGLint*, EGLConfig*, EGLint, EGLint*);
extern EGLContext eglCreateContext(EGLDisplay, EGLConfig, EGLContext, const EGLint*);
extern EGLBoolean eglMakeCurrent(EGLDisplay, void*, void*, EGLContext);
static int egl_setup() {
  EGLDisplay d = eglGetPlatformDisplay(0x31DD, 0, 0);
  if(!d) return -1;
  EGLint ma,mi; if(!eglInitialize(d,&ma,&mi)) return -2;
  eglBindAPI(0x30A2);
  EGLint at[]={0x3040,0x0008,0x3033,0x0001,0x3038};
  EGLConfig c; EGLint n; if(!eglChooseConfig(d,at,&c,1,&n)||n<1) return -3;
  EGLint ca[]={0x3098,3,0x30FB,3,0x30FD,1,0x3038};
  EGLContext ctx=eglCreateContext(d,c,0,ca); if(!ctx) return -4;
  if(!eglMakeCurrent(d,0,0,ctx)) return -5;
  return 0;
}
static uintptr_t glcall12(void* f, uintptr_t* a){
 return ((uintptr_t(*)(uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t,uintptr_t))f)(a[0],a[1],a[2],a[3],a[4],a[5],a[6],a[7],a[8],a[9],a[10],a[11]);
}
static void* getproc(const char* n){ return eglGetProcAddress(n); }
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

// Headless-Kontext für Tests/Screenshots
func InitHeadlessGL() error {
	runtime.LockOSThread()
	if r := C.egl_setup(); r != 0 {
		return fmt.Errorf("EGL-Fehler %d", int(r))
	}
	return nil
}

func glGetProc(name string) uintptr {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	return uintptr(C.getproc(cn))
}

func glCall(f uintptr, a ...uintptr) uintptr {
	var args [12]C.uintptr_t
	for i, v := range a {
		args[i] = C.uintptr_t(v)
	}
	return uintptr(C.glcall12(unsafe.Pointer(f), &args[0]))
}
