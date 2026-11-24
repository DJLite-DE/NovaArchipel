package game

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Minimale OpenGL-3.3-Anbindung ohne cgo-Abhängigkeit (Windows: syscall, Linux-Test: cgo).
// Es werden ausschließlich Funktionen mit Ganzzahl-/Zeigerargumenten benutzt.

const (
	GL_DEPTH_BUFFER_BIT         = 0x00000100
	GL_COLOR_BUFFER_BIT         = 0x00004000
	GL_TRIANGLES                = 0x0004
	GL_LINES                    = 0x0001
	GL_LINE_STRIP               = 0x0003
	GL_TRIANGLE_STRIP           = 0x0005
	GL_UNSIGNED_INT             = 0x1405
	GL_UNSIGNED_BYTE            = 0x1401
	GL_FLOAT                    = 0x1406
	GL_HALF_FLOAT               = 0x140B
	GL_ARRAY_BUFFER             = 0x8892
	GL_ELEMENT_ARRAY_BUFFER     = 0x8893
	GL_STATIC_DRAW              = 0x88E4
	GL_DYNAMIC_DRAW             = 0x88E8
	GL_VERTEX_SHADER            = 0x8B31
	GL_FRAGMENT_SHADER          = 0x8B30
	GL_COMPILE_STATUS           = 0x8B81
	GL_LINK_STATUS              = 0x8B82
	GL_TEXTURE_2D               = 0x0DE1
	GL_TEXTURE0                 = 0x84C0
	GL_TEXTURE_MIN_FILTER       = 0x2801
	GL_TEXTURE_MAG_FILTER       = 0x2800
	GL_TEXTURE_WRAP_S           = 0x2802
	GL_TEXTURE_WRAP_T           = 0x2803
	GL_LINEAR                   = 0x2601
	GL_NEAREST                  = 0x2600
	GL_CLAMP_TO_EDGE            = 0x812F
	GL_CLAMP_TO_BORDER          = 0x812D
	GL_TEXTURE_BORDER_COLOR     = 0x1004
	GL_TEXTURE_COMPARE_MODE     = 0x884C
	GL_TEXTURE_COMPARE_FUNC     = 0x884D
	GL_COMPARE_REF_TO_TEXTURE   = 0x884E
	GL_LEQUAL                   = 0x0203
	GL_LESS                     = 0x0201
	GL_RGBA                     = 0x1908
	GL_RGBA8                    = 0x8058
	GL_RGBA16F                  = 0x881A
	GL_R32F                     = 0x822E
	GL_RED                      = 0x1903
	GL_BGRA                     = 0x80E1
	GL_DEPTH_COMPONENT          = 0x1902
	GL_DEPTH_COMPONENT24        = 0x81A6
	GL_DEPTH_COMPONENT32F       = 0x8CAC
	GL_FRAMEBUFFER              = 0x8D40
	GL_READ_FRAMEBUFFER         = 0x8CA8
	GL_DRAW_FRAMEBUFFER         = 0x8CA9
	GL_RENDERBUFFER             = 0x8D41
	GL_COLOR_ATTACHMENT0        = 0x8CE0
	GL_DEPTH_ATTACHMENT         = 0x8D00
	GL_FRAMEBUFFER_COMPLETE     = 0x8CD5
	GL_DEPTH_TEST               = 0x0B71
	GL_BLEND                    = 0x0BE2
	GL_CULL_FACE                = 0x0B44
	GL_BACK                     = 0x0405
	GL_FRONT                    = 0x0404
	GL_SRC_ALPHA                = 0x0302
	GL_ONE_MINUS_SRC_ALPHA      = 0x0303
	GL_ONE                      = 1
	GL_ZERO                     = 0
	GL_COLOR                    = 0x1800
	GL_DEPTH                    = 0x1801
	GL_PACK_ALIGNMENT           = 0x0D05
	GL_UNPACK_ALIGNMENT         = 0x0CF5
	GL_MULTISAMPLE              = 0x809D
	GL_VERSION                  = 0x1F02
	GL_RENDERER                 = 0x1F01
	GL_MAX_SAMPLES              = 0x8D57
	GL_UNSIGNED_INT_8_8_8_8_REV = 0x8367
	GL_INFO_LOG_LENGTH          = 0x8B84
	GL_FRAMEBUFFER_SRGB         = 0x8DB9
	GL_LINE_SMOOTH              = 0x0B20
)

var glProcs = map[string]uintptr{}

func glProc(name string) uintptr {
	if p, ok := glProcs[name]; ok {
		return p
	}
	p := glGetProc(name)
	if p == 0 {
		panic("OpenGL-Funktion fehlt: " + name)
	}
	glProcs[name] = p
	return p
}

func gl(name string, a ...uintptr) uintptr {
	return glCall(glProc(name), a...)
}

func b2u(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

func cstr(s string) *byte {
	b := append([]byte(s), 0)
	return &b[0]
}

func ptr(p unsafe.Pointer) uintptr { return uintptr(p) }

// ---------------------------------------------------------------- Wrapper

func glGenBuffer() uint32 {
	var id uint32
	gl("glGenBuffers", 1, ptr(unsafe.Pointer(&id)))
	return id
}
func glGenVertexArray() uint32 {
	var id uint32
	gl("glGenVertexArrays", 1, ptr(unsafe.Pointer(&id)))
	return id
}
func glGenTexture() uint32 {
	var id uint32
	gl("glGenTextures", 1, ptr(unsafe.Pointer(&id)))
	return id
}
func glGenFramebuffer() uint32 {
	var id uint32
	gl("glGenFramebuffers", 1, ptr(unsafe.Pointer(&id)))
	return id
}
func glGenRenderbuffer() uint32 {
	var id uint32
	gl("glGenRenderbuffers", 1, ptr(unsafe.Pointer(&id)))
	return id
}

func glBufferDataF(target uint32, data []float32, usage uint32) {
	if len(data) == 0 {
		gl("glBufferData", uintptr(target), 0, 0, uintptr(usage))
		return
	}
	gl("glBufferData", uintptr(target), uintptr(len(data)*4), ptr(unsafe.Pointer(&data[0])), uintptr(usage))
	runtime.KeepAlive(data)
}
func glBufferDataU(target uint32, data []uint32, usage uint32) {
	if len(data) == 0 {
		gl("glBufferData", uintptr(target), 0, 0, uintptr(usage))
		return
	}
	gl("glBufferData", uintptr(target), uintptr(len(data)*4), ptr(unsafe.Pointer(&data[0])), uintptr(usage))
	runtime.KeepAlive(data)
}

func glUniformMat(loc int32, m *Mat4) {
	if loc < 0 {
		return
	}
	gl("glUniformMatrix4fv", uintptr(loc), 1, 0, ptr(unsafe.Pointer(&m[0])))
	runtime.KeepAlive(m)
}
func glUniformV(loc int32, v ...float32) {
	if loc < 0 {
		return
	}
	switch len(v) {
	case 1:
		gl("glUniform1fv", uintptr(loc), 1, ptr(unsafe.Pointer(&v[0])))
	case 2:
		gl("glUniform2fv", uintptr(loc), 1, ptr(unsafe.Pointer(&v[0])))
	case 3:
		gl("glUniform3fv", uintptr(loc), 1, ptr(unsafe.Pointer(&v[0])))
	case 4:
		gl("glUniform4fv", uintptr(loc), 1, ptr(unsafe.Pointer(&v[0])))
	}
	runtime.KeepAlive(v)
}
func glUniform4Arr(loc int32, v []float32) {
	if loc < 0 || len(v) == 0 {
		return
	}
	gl("glUniform4fv", uintptr(loc), uintptr(len(v)/4), ptr(unsafe.Pointer(&v[0])))
	runtime.KeepAlive(v)
}
func glUniformI(loc int32, i int) {
	if loc < 0 {
		return
	}
	gl("glUniform1i", uintptr(loc), uintptr(i))
}

func glClearColorBuf(r, g, b, a float32) {
	c := []float32{r, g, b, a}
	gl("glClearBufferfv", GL_COLOR, 0, ptr(unsafe.Pointer(&c[0])))
	runtime.KeepAlive(c)
}
func glClearDepthBuf() {
	d := []float32{1}
	gl("glClearBufferfv", GL_DEPTH, 0, ptr(unsafe.Pointer(&d[0])))
	runtime.KeepAlive(d)
}

func glString(name uint32) string {
	p := gl("glGetString", uintptr(name))
	if p == 0 {
		return ""
	}
	var b []byte
	for i := 0; i < 400; i++ {
		c := *(*byte)(unsafe.Pointer(p + uintptr(i)))
		if c == 0 {
			break
		}
		b = append(b, c)
	}
	return string(b)
}

// ---------------------------------------------------------------- Shader

type Prog struct {
	ID   uint32
	locs map[string]int32
}

func compileShader(kind uint32, src string) (uint32, error) {
	sh := uint32(gl("glCreateShader", uintptr(kind)))
	cs := cstr(src)
	pp := []uintptr{uintptr(unsafe.Pointer(cs))}
	gl("glShaderSource", uintptr(sh), 1, ptr(unsafe.Pointer(&pp[0])), 0)
	runtime.KeepAlive(cs)
	runtime.KeepAlive(pp)
	gl("glCompileShader", uintptr(sh))
	var ok int32
	gl("glGetShaderiv", uintptr(sh), GL_COMPILE_STATUS, ptr(unsafe.Pointer(&ok)))
	if ok == 0 {
		log := make([]byte, 4096)
		var n int32
		gl("glGetShaderInfoLog", uintptr(sh), 4096, ptr(unsafe.Pointer(&n)), ptr(unsafe.Pointer(&log[0])))
		return 0, fmt.Errorf("Shader-Fehler: %s", string(log[:n]))
	}
	return sh, nil
}

func NewProg(vs, fs string) (*Prog, error) {
	v, err := compileShader(GL_VERTEX_SHADER, vs)
	if err != nil {
		return nil, err
	}
	f, err := compileShader(GL_FRAGMENT_SHADER, fs)
	if err != nil {
		return nil, err
	}
	p := uint32(gl("glCreateProgram"))
	gl("glAttachShader", uintptr(p), uintptr(v))
	gl("glAttachShader", uintptr(p), uintptr(f))
	for i, n := range []string{"aPos", "aNrm", "aCol", "iPos", "iScl"} {
		cn := cstr(n)
		gl("glBindAttribLocation", uintptr(p), uintptr(i), ptr(unsafe.Pointer(cn)))
		runtime.KeepAlive(cn)
	}
	gl("glLinkProgram", uintptr(p))
	var ok int32
	gl("glGetProgramiv", uintptr(p), GL_LINK_STATUS, ptr(unsafe.Pointer(&ok)))
	if ok == 0 {
		log := make([]byte, 4096)
		var n int32
		gl("glGetProgramInfoLog", uintptr(p), 4096, ptr(unsafe.Pointer(&n)), ptr(unsafe.Pointer(&log[0])))
		return nil, fmt.Errorf("Link-Fehler: %s", string(log[:n]))
	}
	return &Prog{ID: p, locs: map[string]int32{}}, nil
}

func (p *Prog) Use() { gl("glUseProgram", uintptr(p.ID)) }
func (p *Prog) L(name string) int32 {
	if l, ok := p.locs[name]; ok {
		return l
	}
	cn := cstr(name)
	l := int32(gl("glGetUniformLocation", uintptr(p.ID), ptr(unsafe.Pointer(cn))))
	runtime.KeepAlive(cn)
	p.locs[name] = l
	return l
}
