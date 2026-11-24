package game

import (
	"math"
	"unsafe"
)

// ---------------------------------------------------------------- Mathe

type V3 struct{ X, Y, Z float64 }

func v3(x, y, z float64) V3   { return V3{x, y, z} }
func (a V3) Add(b V3) V3      { return V3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a V3) Sub(b V3) V3      { return V3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a V3) Mul(f float64) V3 { return V3{a.X * f, a.Y * f, a.Z * f} }
func (a V3) Dot(b V3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a V3) Cross(b V3) V3    { return V3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X} }
func (a V3) Len() float64     { return math.Sqrt(a.Dot(a)) }
func (a V3) Norm() V3 {
	l := a.Len()
	if l == 0 {
		return V3{0, 1, 0}
	}
	return a.Mul(1 / l)
}

// Spaltenweise 4x4-Matrix
type Mat4 [16]float32

func Ident() Mat4 { return Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }

func (a Mat4) Mul(b Mat4) Mat4 {
	var r Mat4
	for c := 0; c < 4; c++ {
		for rr := 0; rr < 4; rr++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[k*4+rr] * b[c*4+k]
			}
			r[c*4+rr] = s
		}
	}
	return r
}

func (m Mat4) MulV(v V3, w float64) (float64, float64, float64, float64) {
	x, y, z := float32(v.X), float32(v.Y), float32(v.Z)
	ww := float32(w)
	return float64(m[0]*x + m[4]*y + m[8]*z + m[12]*ww),
		float64(m[1]*x + m[5]*y + m[9]*z + m[13]*ww),
		float64(m[2]*x + m[6]*y + m[10]*z + m[14]*ww),
		float64(m[3]*x + m[7]*y + m[11]*z + m[15]*ww)
}

func Translate(x, y, z float64) Mat4 {
	m := Ident()
	m[12], m[13], m[14] = float32(x), float32(y), float32(z)
	return m
}
func ScaleM(x, y, z float64) Mat4 {
	m := Ident()
	m[0], m[5], m[10] = float32(x), float32(y), float32(z)
	return m
}
func RotY(a float64) Mat4 {
	c, s := float32(math.Cos(a)), float32(math.Sin(a))
	m := Ident()
	m[0], m[2], m[8], m[10] = c, -s, s, c
	return m
}
func RotX(a float64) Mat4 {
	c, s := float32(math.Cos(a)), float32(math.Sin(a))
	m := Ident()
	m[5], m[6], m[9], m[10] = c, s, -s, c
	return m
}
func RotZ(a float64) Mat4 {
	c, s := float32(math.Cos(a)), float32(math.Sin(a))
	m := Ident()
	m[0], m[1], m[4], m[5] = c, s, -s, c
	return m
}

func Perspective(fovy, aspect, near, far float64) Mat4 {
	f := 1 / math.Tan(fovy/2)
	var m Mat4
	m[0] = float32(f / aspect)
	m[5] = float32(f)
	m[10] = float32((far + near) / (near - far))
	m[11] = -1
	m[14] = float32(2 * far * near / (near - far))
	return m
}

func Ortho(l, r, b, t, n, f float64) Mat4 {
	m := Ident()
	m[0] = float32(2 / (r - l))
	m[5] = float32(2 / (t - b))
	m[10] = float32(-2 / (f - n))
	m[12] = float32(-(r + l) / (r - l))
	m[13] = float32(-(t + b) / (t - b))
	m[14] = float32(-(f + n) / (f - n))
	return m
}

func LookAt(eye, at, up V3) Mat4 {
	f := at.Sub(eye).Norm()
	s := f.Cross(up).Norm()
	u := s.Cross(f)
	m := Ident()
	m[0], m[4], m[8] = float32(s.X), float32(s.Y), float32(s.Z)
	m[1], m[5], m[9] = float32(u.X), float32(u.Y), float32(u.Z)
	m[2], m[6], m[10] = float32(-f.X), float32(-f.Y), float32(-f.Z)
	m[12] = float32(-s.Dot(eye))
	m[13] = float32(-u.Dot(eye))
	m[14] = float32(f.Dot(eye))
	return m
}

func (m Mat4) Inverse() Mat4 {
	var inv [16]float64
	a := [16]float64{}
	for i := range m {
		a[i] = float64(m[i])
	}
	inv[0] = a[5]*a[10]*a[15] - a[5]*a[11]*a[14] - a[9]*a[6]*a[15] + a[9]*a[7]*a[14] + a[13]*a[6]*a[11] - a[13]*a[7]*a[10]
	inv[4] = -a[4]*a[10]*a[15] + a[4]*a[11]*a[14] + a[8]*a[6]*a[15] - a[8]*a[7]*a[14] - a[12]*a[6]*a[11] + a[12]*a[7]*a[10]
	inv[8] = a[4]*a[9]*a[15] - a[4]*a[11]*a[13] - a[8]*a[5]*a[15] + a[8]*a[7]*a[13] + a[12]*a[5]*a[11] - a[12]*a[7]*a[9]
	inv[12] = -a[4]*a[9]*a[14] + a[4]*a[10]*a[13] + a[8]*a[5]*a[14] - a[8]*a[6]*a[13] - a[12]*a[5]*a[10] + a[12]*a[6]*a[9]
	inv[1] = -a[1]*a[10]*a[15] + a[1]*a[11]*a[14] + a[9]*a[2]*a[15] - a[9]*a[3]*a[14] - a[13]*a[2]*a[11] + a[13]*a[3]*a[10]
	inv[5] = a[0]*a[10]*a[15] - a[0]*a[11]*a[14] - a[8]*a[2]*a[15] + a[8]*a[3]*a[14] + a[12]*a[2]*a[11] - a[12]*a[3]*a[10]
	inv[9] = -a[0]*a[9]*a[15] + a[0]*a[11]*a[13] + a[8]*a[1]*a[15] - a[8]*a[3]*a[13] - a[12]*a[1]*a[11] + a[12]*a[3]*a[9]
	inv[13] = a[0]*a[9]*a[14] - a[0]*a[10]*a[13] - a[8]*a[1]*a[14] + a[8]*a[2]*a[13] + a[12]*a[1]*a[10] - a[12]*a[2]*a[9]
	inv[2] = a[1]*a[6]*a[15] - a[1]*a[7]*a[14] - a[5]*a[2]*a[15] + a[5]*a[3]*a[14] + a[13]*a[2]*a[7] - a[13]*a[3]*a[6]
	inv[6] = -a[0]*a[6]*a[15] + a[0]*a[7]*a[14] + a[4]*a[2]*a[15] - a[4]*a[3]*a[14] - a[12]*a[2]*a[7] + a[12]*a[3]*a[6]
	inv[10] = a[0]*a[5]*a[15] - a[0]*a[7]*a[13] - a[4]*a[1]*a[15] + a[4]*a[3]*a[13] + a[12]*a[1]*a[7] - a[12]*a[3]*a[5]
	inv[14] = -a[0]*a[5]*a[14] + a[0]*a[6]*a[13] + a[4]*a[1]*a[14] - a[4]*a[2]*a[13] - a[12]*a[1]*a[6] + a[12]*a[2]*a[5]
	inv[3] = -a[1]*a[6]*a[11] + a[1]*a[7]*a[10] + a[5]*a[2]*a[11] - a[5]*a[3]*a[10] - a[9]*a[2]*a[7] + a[9]*a[3]*a[6]
	inv[7] = a[0]*a[6]*a[11] - a[0]*a[7]*a[10] - a[4]*a[2]*a[11] + a[4]*a[3]*a[10] + a[8]*a[2]*a[7] - a[8]*a[3]*a[6]
	inv[11] = -a[0]*a[5]*a[11] + a[0]*a[7]*a[9] + a[4]*a[1]*a[11] - a[4]*a[3]*a[9] - a[8]*a[1]*a[7] + a[8]*a[3]*a[5]
	inv[15] = a[0]*a[5]*a[10] - a[0]*a[6]*a[9] - a[4]*a[1]*a[10] + a[4]*a[2]*a[9] + a[8]*a[1]*a[6] - a[8]*a[2]*a[5]
	det := a[0]*inv[0] + a[1]*inv[4] + a[2]*inv[8] + a[3]*inv[12]
	var r Mat4
	if det == 0 {
		return Ident()
	}
	for i := range r {
		r[i] = float32(inv[i] / det)
	}
	return r
}

// ---------------------------------------------------------------- Materialien (im Alphakanal der Farbe)

const (
	MatMatte   = 0
	MatPanel   = 1 // glänzende Fassadenpaneele
	MatGlass   = 2 // Glasfassade mit Fenstern
	MatEmit    = 3 // Leuchtend
	MatLeaf    = 4 // Laub (Wind)
	MatSolar   = 5
	MatTerrain = 6
	MatMetal   = 7
	MatWaterP  = 8 // Wasserbecken
)

// ---------------------------------------------------------------- Mesh-Bau

type MB struct {
	V   []float32 // pos3 nrm3 col4
	I   []uint32
	tf  Mat4
	stk []Mat4
}

func NewMB() *MB { return &MB{tf: Ident()} }

func (b *MB) Push(m Mat4) {
	b.stk = append(b.stk, b.tf)
	b.tf = b.tf.Mul(m)
}
func (b *MB) Pop() {
	b.tf = b.stk[len(b.stk)-1]
	b.stk = b.stk[:len(b.stk)-1]
}

func (b *MB) vert(p, n V3, c RGB, mat int) uint32 {
	x, y, z, _ := b.tf.MulV(p, 1)
	nx, ny, nz, _ := b.tf.MulV(n, 0)
	nn := V3{nx, ny, nz}.Norm()
	b.V = append(b.V, float32(x), float32(y), float32(z), float32(nn.X), float32(nn.Y), float32(nn.Z),
		srgb(c.R), srgb(c.G), srgb(c.B), float32(mat))
	return uint32(len(b.V)/10 - 1)
}

// sRGB -> linear
func srgb(v uint8) float32 {
	f := float64(v) / 255
	return float32(math.Pow(f, 2.2))
}

func (b *MB) Tri(a, c, d V3, col RGB, mat int) {
	n := c.Sub(a).Cross(d.Sub(a)).Norm()
	i := b.vert(a, n, col, mat)
	b.vert(c, n, col, mat)
	b.vert(d, n, col, mat)
	b.I = append(b.I, i, i+1, i+2)
}

// Viereck (gegen den Uhrzeigersinn von außen gesehen)
func (b *MB) Quad(a, c, d, e V3, col RGB, mat int) {
	n := c.Sub(a).Cross(e.Sub(a)).Norm()
	i := b.vert(a, n, col, mat)
	b.vert(c, n, col, mat)
	b.vert(d, n, col, mat)
	b.vert(e, n, col, mat)
	b.I = append(b.I, i, i+1, i+2, i, i+2, i+3)
}

// Quader; top/bottom optional über Flags
func (b *MB) Box(x0, y0, z0, x1, y1, z1 float64, col RGB, mat int) {
	b.BoxC(x0, y0, z0, x1, y1, z1, col, col, mat, mat)
}

// Quader mit eigener Farbe/Material für die Oberseite
func (b *MB) BoxC(x0, y0, z0, x1, y1, z1 float64, side, top RGB, smat, tmat int) {
	p := func(x, y, z float64) V3 { return V3{x, y, z} }
	// +Y
	b.Quad(p(x0, y1, z0), p(x0, y1, z1), p(x1, y1, z1), p(x1, y1, z0), top, tmat)
	// -Z
	b.Quad(p(x0, y0, z0), p(x0, y1, z0), p(x1, y1, z0), p(x1, y0, z0), side, smat)
	// +Z
	b.Quad(p(x1, y0, z1), p(x1, y1, z1), p(x0, y1, z1), p(x0, y0, z1), side, smat)
	// -X
	b.Quad(p(x0, y0, z1), p(x0, y1, z1), p(x0, y1, z0), p(x0, y0, z0), side, smat)
	// +X
	b.Quad(p(x1, y0, z0), p(x1, y1, z0), p(x1, y1, z1), p(x1, y0, z1), side, smat)
}

// Zylinder entlang Y
func (b *MB) Cyl(cx, cz, r, y0, y1 float64, seg int, col RGB, mat int, cap bool) {
	b.Cone(cx, cz, r, r, y0, y1, seg, col, mat, cap)
}

// Kegelstumpf
func (b *MB) Cone(cx, cz, r0, r1, y0, y1 float64, seg int, col RGB, mat int, cap bool) {
	slope := (r0 - r1) / math.Max(0.0001, y1-y0)
	base := uint32(len(b.V) / 10)
	for i := 0; i <= seg; i++ {
		a := float64(i) / float64(seg) * 2 * math.Pi
		c, s := math.Cos(a), math.Sin(a)
		n := V3{c, slope, s}.Norm()
		b.vert(V3{cx + c*r0, y0, cz + s*r0}, n, col, mat)
		b.vert(V3{cx + c*r1, y1, cz + s*r1}, n, col, mat)
	}
	for i := 0; i < seg; i++ {
		k := base + uint32(i*2)
		b.I = append(b.I, k, k+1, k+3, k, k+3, k+2)
	}
	if cap && r1 > 0.001 {
		cb := b.vert(V3{cx, y1, cz}, V3{0, 1, 0}, col, mat)
		for i := 0; i <= seg; i++ {
			a := float64(i) / float64(seg) * 2 * math.Pi
			b.vert(V3{cx + math.Cos(a)*r1, y1, cz + math.Sin(a)*r1}, V3{0, 1, 0}, col, mat)
		}
		for i := 0; i < seg; i++ {
			b.I = append(b.I, cb, cb+uint32(i)+2, cb+uint32(i)+1)
		}
	}
}

// Kugel(-abschnitt): v0..v1 in [0,1] von unten nach oben (0.5..1 = Kuppel)
func (b *MB) Sphere(cx, cy, cz, rx, ry, rz float64, v0, v1 float64, rings, seg int, col RGB, mat int) {
	base := uint32(len(b.V) / 10)
	for r := 0; r <= rings; r++ {
		t := v0 + (v1-v0)*float64(r)/float64(rings)
		phi := (t - 0.5) * math.Pi
		cy0, sy0 := math.Cos(phi), math.Sin(phi)
		for i := 0; i <= seg; i++ {
			a := float64(i) / float64(seg) * 2 * math.Pi
			n := V3{math.Cos(a) * cy0, sy0, math.Sin(a) * cy0}
			p := V3{cx + n.X*rx, cy + n.Y*ry, cz + n.Z*rz}
			b.vert(p, V3{n.X / rx, n.Y / ry, n.Z / rz}.Norm(), col, mat)
		}
	}
	for r := 0; r < rings; r++ {
		for i := 0; i < seg; i++ {
			k := base + uint32(r*(seg+1)+i)
			k2 := k + uint32(seg+1)
			b.I = append(b.I, k, k2, k2+1, k, k2+1, k+1)
		}
	}
}

// Prisma aus Grundriss (XZ, gegen Uhrzeigersinn von oben)
func (b *MB) Prism(pts [][2]float64, y0, y1 float64, col RGB, mat int, top RGB, tmat int) {
	n := len(pts)
	for i := 0; i < n; i++ {
		a, c := pts[i], pts[(i+1)%n]
		b.Quad(V3{a[0], y0, a[1]}, V3{a[0], y1, a[1]}, V3{c[0], y1, c[1]}, V3{c[0], y0, c[1]}, col, mat)
	}
	// Deckel als Fächer (konvex angenommen)
	cb := b.vert(V3{pts[0][0], y1, pts[0][1]}, V3{0, 1, 0}, top, tmat)
	for i := 1; i < n; i++ {
		b.vert(V3{pts[i][0], y1, pts[i][1]}, V3{0, 1, 0}, top, tmat)
	}
	for i := 1; i < n-1; i++ {
		b.I = append(b.I, cb, cb+uint32(i+1), cb+uint32(i))
	}
}

// Flache horizontale Platte
func (b *MB) Plate(x0, z0, x1, z1, y float64, col RGB, mat int) {
	b.Quad(V3{x0, y, z0}, V3{x0, y, z1}, V3{x1, y, z1}, V3{x1, y, z0}, col, mat)
}

func (b *MB) Merge(o *MB) {
	base := uint32(len(b.V) / 10)
	b.V = append(b.V, o.V...)
	for _, i := range o.I {
		b.I = append(b.I, i+base)
	}
}

// ---------------------------------------------------------------- GPU-Mesh

type GMesh struct {
	VAO, VBO, IBO uint32
	N             int
}

func (b *MB) Upload() *GMesh {
	m := &GMesh{}
	m.VAO = glGenVertexArray()
	gl("glBindVertexArray", uintptr(m.VAO))
	m.VBO = glGenBuffer()
	gl("glBindBuffer", GL_ARRAY_BUFFER, uintptr(m.VBO))
	glBufferDataF(GL_ARRAY_BUFFER, b.V, GL_STATIC_DRAW)
	m.IBO = glGenBuffer()
	gl("glBindBuffer", GL_ELEMENT_ARRAY_BUFFER, uintptr(m.IBO))
	glBufferDataU(GL_ELEMENT_ARRAY_BUFFER, b.I, GL_STATIC_DRAW)
	stride := uintptr(40)
	gl("glEnableVertexAttribArray", 0)
	gl("glVertexAttribPointer", 0, 3, GL_FLOAT, 0, stride, 0)
	gl("glEnableVertexAttribArray", 1)
	gl("glVertexAttribPointer", 1, 3, GL_FLOAT, 0, stride, 12)
	gl("glEnableVertexAttribArray", 2)
	gl("glVertexAttribPointer", 2, 4, GL_FLOAT, 0, stride, 24)
	gl("glBindVertexArray", 0)
	m.N = len(b.I)
	return m
}

func (m *GMesh) Update(b *MB) {
	gl("glBindBuffer", GL_ARRAY_BUFFER, uintptr(m.VBO))
	glBufferDataF(GL_ARRAY_BUFFER, b.V, GL_DYNAMIC_DRAW)
	gl("glBindBuffer", GL_ELEMENT_ARRAY_BUFFER, uintptr(m.IBO))
	gl("glBindVertexArray", uintptr(m.VAO))
	gl("glBindBuffer", GL_ELEMENT_ARRAY_BUFFER, uintptr(m.IBO))
	glBufferDataU(GL_ELEMENT_ARRAY_BUFFER, b.I, GL_DYNAMIC_DRAW)
	gl("glBindVertexArray", 0)
	m.N = len(b.I)
}

func (m *GMesh) Draw() {
	if m == nil || m.N == 0 {
		return
	}
	gl("glBindVertexArray", uintptr(m.VAO))
	gl("glDrawElements", GL_TRIANGLES, uintptr(m.N), GL_UNSIGNED_INT, 0)
}

// Instanz-Puffer (iPos: xyz + Drehung, iScl: Skalierung + Farbton)
type Instanced struct {
	M    *GMesh
	IVBO uint32
	N    int
}

func NewInstanced(m *GMesh) *Instanced {
	in := &Instanced{M: m}
	gl("glBindVertexArray", uintptr(m.VAO))
	in.IVBO = glGenBuffer()
	gl("glBindBuffer", GL_ARRAY_BUFFER, uintptr(in.IVBO))
	gl("glEnableVertexAttribArray", 3)
	gl("glVertexAttribPointer", 3, 4, GL_FLOAT, 0, 32, 0)
	gl("glVertexAttribDivisor", 3, 1)
	gl("glEnableVertexAttribArray", 4)
	gl("glVertexAttribPointer", 4, 4, GL_FLOAT, 0, 32, 16)
	gl("glVertexAttribDivisor", 4, 1)
	gl("glBindVertexArray", 0)
	return in
}

func (in *Instanced) Set(data []float32) {
	gl("glBindBuffer", GL_ARRAY_BUFFER, uintptr(in.IVBO))
	glBufferDataF(GL_ARRAY_BUFFER, data, GL_DYNAMIC_DRAW)
	in.N = len(data) / 8
}

func (in *Instanced) Draw() {
	if in.N == 0 {
		return
	}
	gl("glBindVertexArray", uintptr(in.M.VAO))
	gl("glDrawElementsInstanced", GL_TRIANGLES, uintptr(in.M.N), GL_UNSIGNED_INT, 0, uintptr(in.N))
}

var _ = unsafe.Pointer(nil)
