package game

import (
	"math"
	"sort"
)

// Painter: überabgetasteter Vektor-Zeichner für antialiasierte Sprites
type Painter struct {
	W, H int // Zielgröße
	SS   int // Supersampling
	bw   int
	bh   int
	buf  []float32 // RGBA, nicht vormultipliziert über "over"-Operator -> wir speichern premultiplied
}

func NewPainter(w, h, ss int) *Painter {
	p := &Painter{W: w, H: h, SS: ss, bw: w * ss, bh: h * ss}
	p.buf = make([]float32, p.bw*p.bh*4)
	return p
}

type P struct{ X, Y float64 }

// Füllt ein Polygon (even-odd) mit Farbe und Deckkraft a (0..1)
func (p *Painter) Poly(pts []P, col RGB, a float64) {
	if len(pts) < 3 || a <= 0 {
		return
	}
	ss := float64(p.SS)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, q := range pts {
		minY = math.Min(minY, q.Y*ss)
		maxY = math.Max(maxY, q.Y*ss)
	}
	y0 := imax(0, int(math.Floor(minY)))
	y1 := imin(p.bh-1, int(math.Ceil(maxY)))
	cr, cg, cb := float32(col.R)/255*float32(a), float32(col.G)/255*float32(a), float32(col.B)/255*float32(a)
	fa := float32(a)
	xs := make([]float64, 0, 16)
	n := len(pts)
	for y := y0; y <= y1; y++ {
		sy := float64(y) + 0.5
		xs = xs[:0]
		for i := 0; i < n; i++ {
			a1, b1 := pts[i], pts[(i+1)%n]
			ay, by := a1.Y*ss, b1.Y*ss
			if (ay <= sy && by > sy) || (by <= sy && ay > sy) {
				t := (sy - ay) / (by - ay)
				xs = append(xs, a1.X*ss+t*(b1.X*ss-a1.X*ss))
			}
		}
		if len(xs) < 2 {
			continue
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			xa := imax(0, int(math.Ceil(xs[i]-0.5)))
			xb := imin(p.bw-1, int(math.Floor(xs[i+1]-0.5)))
			row := p.buf[(y*p.bw)*4:]
			for x := xa; x <= xb; x++ {
				k := x * 4
				ia := 1 - fa
				row[k] = cr + row[k]*ia
				row[k+1] = cg + row[k+1]*ia
				row[k+2] = cb + row[k+2]*ia
				row[k+3] = fa + row[k+3]*ia
			}
		}
	}
}

func (p *Painter) Ellipse(cx, cy, rx, ry float64, col RGB, a float64) {
	n := 32
	pts := make([]P, n)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n) * 2 * math.Pi
		pts[i] = P{cx + math.Cos(t)*rx, cy + math.Sin(t)*ry}
	}
	p.Poly(pts, col, a)
}

// Weiches Leuchten (mehrere Ellipsen)
func (p *Painter) Glow(cx, cy, r float64, col RGB, a float64) {
	steps := 6
	for i := steps; i >= 1; i-- {
		f := float64(i) / float64(steps)
		p.Ellipse(cx, cy, r*f, r*f*0.8, col, a/float64(steps)*1.3)
	}
}

func (p *Painter) LineW(x0, y0, x1, y1, w float64, col RGB, a float64) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx, ny := -dy/l*w/2, dx/l*w/2
	p.Poly([]P{{x0 + nx, y0 + ny}, {x1 + nx, y1 + ny}, {x1 - nx, y1 - ny}, {x0 - nx, y0 - ny}}, col, a)
}

func (p *Painter) ToSprite(ox, oy int) *Sprite {
	s := &Sprite{W: p.W, H: p.H, OX: ox, OY: oy, Pix: make([]uint32, p.W*p.H)}
	ss := p.SS
	inv := 1 / float32(ss*ss)
	for y := 0; y < p.H; y++ {
		for x := 0; x < p.W; x++ {
			var r, g, b, a float32
			for sy := 0; sy < ss; sy++ {
				row := ((y*ss+sy)*p.bw + x*ss) * 4
				for sx := 0; sx < ss; sx++ {
					k := row + sx*4
					r += p.buf[k]
					g += p.buf[k+1]
					b += p.buf[k+2]
					a += p.buf[k+3]
				}
			}
			r, g, b, a = r*inv, g*inv, b*inv, a*inv
			if a <= 0.004 {
				continue
			}
			s.Pix[y*p.W+x] = uint32(clampf(float64(a)*255, 0, 255))<<24 | uint32(clampf(float64(r)*255, 0, 255))<<16 | uint32(clampf(float64(g)*255, 0, 255))<<8 | uint32(clampf(float64(b)*255, 0, 255))
		}
	}
	return s
}

// ---------------------------------------------------------------- Iso-Helfer

type Iso struct {
	P      *Painter
	OX, OY float64  // Bildschirmposition des Welt-Ursprungs im Sprite
	HW, HH float64  // halbe Kachelbreite/-höhe
	ZS     float64  // Pixel pro Höheneinheit
	SP     *Painter // Schattenebene (nur im Schattendurchlauf gesetzt)
}

// Schattenrichtung (Weltkoordinaten pro Höheneinheit): Sonne von links oben
const shX, shY = 0.62, -0.2

func (s *Iso) shadowPt(x, y, z float64) P {
	return s.Pt(x+z*shX, y+z*shY, 0)
}

// konvexe Hülle (Monotone Chain)
func hull(ps []P) []P {
	if len(ps) < 3 {
		return ps
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].X == ps[j].X {
			return ps[i].Y < ps[j].Y
		}
		return ps[i].X < ps[j].X
	})
	cross := func(o, a, b P) float64 { return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X) }
	var lo, up []P
	for _, p := range ps {
		for len(lo) >= 2 && cross(lo[len(lo)-2], lo[len(lo)-1], p) <= 0 {
			lo = lo[:len(lo)-1]
		}
		lo = append(lo, p)
	}
	for i := len(ps) - 1; i >= 0; i-- {
		p := ps[i]
		for len(up) >= 2 && cross(up[len(up)-2], up[len(up)-1], p) <= 0 {
			up = up[:len(up)-1]
		}
		up = append(up, p)
	}
	return append(lo[:len(lo)-1], up[:len(up)-1]...)
}

// Schatten eines Körpers: Hülle aus Grundfläche und projizierter Oberseite
func (s *Iso) shadowHull(pts [][3]float64) {
	var ps []P
	for _, q := range pts {
		ps = append(ps, s.Pt(q[0], q[1], 0), s.shadowPt(q[0], q[1], q[2]))
	}
	s.SP.Poly(hull(ps), RGB{0, 0, 0}, 1)
}

func (s *Iso) Pt(x, y, z float64) P {
	return P{s.OX + (x-y)*s.HW, s.OY + (x+y)*s.HH - z*s.ZS}
}

func (s *Iso) Quad(col RGB, a float64, pts ...[3]float64) {
	if s.SP != nil {
		hi := false
		for _, q := range pts {
			if q[2] > 0.08 {
				hi = true
			}
		}
		if hi && a > 0.5 {
			s.shadowHull(pts)
		}
		return
	}
	ps := make([]P, len(pts))
	for i, q := range pts {
		ps[i] = s.Pt(q[0], q[1], q[2])
	}
	s.P.Poly(ps, col, a)
}

// Licht: obere Fläche 1.0, rechte (x-Seite) 0.68, linke (y-Seite) 0.84
func (s *Iso) Box(x0, y0, x1, y1, z0, z1 float64, col RGB) {
	s.BoxA(x0, y0, x1, y1, z0, z1, col, 1)
}

func (s *Iso) BoxA(x0, y0, x1, y1, z0, z1 float64, col RGB, a float64) {
	if s.SP != nil {
		if z1 > 0.08 {
			s.shadowHull([][3]float64{{x0, y0, z1}, {x1, y0, z1}, {x1, y1, z1}, {x0, y1, z1}})
		}
		return
	}
	// linke Seite (y = y1)
	s.Quad(col.Mul(0.82), a, [3]float64{x0, y1, z0}, [3]float64{x1, y1, z0}, [3]float64{x1, y1, z1}, [3]float64{x0, y1, z1})
	// rechte Seite (x = x1)
	s.Quad(col.Mul(0.64), a, [3]float64{x1, y0, z0}, [3]float64{x1, y1, z0}, [3]float64{x1, y1, z1}, [3]float64{x1, y0, z1})
	// Deckfläche
	s.Quad(col, a, [3]float64{x0, y0, z1}, [3]float64{x1, y0, z1}, [3]float64{x1, y1, z1}, [3]float64{x0, y1, z1})
}

// Fensterreihe auf der linken Seite (y=y1) eines Quaders
func (s *Iso) WindowsL(x0, x1, y, z0, z1 float64, nx, nz int, col RGB, a float64) {
	w := (x1 - x0) / float64(nx)
	h := (z1 - z0) / float64(nz)
	for i := 0; i < nx; i++ {
		for j := 0; j < nz; j++ {
			ax := x0 + w*float64(i) + w*0.2
			bx := ax + w*0.6
			az := z0 + h*float64(j) + h*0.25
			bz := az + h*0.5
			s.Quad(col, a, [3]float64{ax, y + 0.001, az}, [3]float64{bx, y + 0.001, az}, [3]float64{bx, y + 0.001, bz}, [3]float64{ax, y + 0.001, bz})
		}
	}
}

// Fensterreihe auf der rechten Seite (x=x)
func (s *Iso) WindowsR(y0, y1, x, z0, z1 float64, ny, nz int, col RGB, a float64) {
	w := (y1 - y0) / float64(ny)
	h := (z1 - z0) / float64(nz)
	for i := 0; i < ny; i++ {
		for j := 0; j < nz; j++ {
			ay := y0 + w*float64(i) + w*0.2
			by := ay + w*0.6
			az := z0 + h*float64(j) + h*0.25
			bz := az + h*0.5
			s.Quad(col, a, [3]float64{x + 0.001, ay, az}, [3]float64{x + 0.001, by, az}, [3]float64{x + 0.001, by, bz}, [3]float64{x + 0.001, ay, bz})
		}
	}
}

// Leuchtband
func (s *Iso) BandL(x0, x1, y, z, h float64, col RGB, a float64) {
	s.Quad(col, a, [3]float64{x0, y + 0.002, z}, [3]float64{x1, y + 0.002, z}, [3]float64{x1, y + 0.002, z + h}, [3]float64{x0, y + 0.002, z + h})
}
func (s *Iso) BandR(y0, y1, x, z, h float64, col RGB, a float64) {
	s.Quad(col, a, [3]float64{x + 0.002, y0, z}, [3]float64{x + 0.002, y1, z}, [3]float64{x + 0.002, y1, z + h}, [3]float64{x + 0.002, y0, z + h})
}

// Zylinder (vertikal) mit Schattierung
func (s *Iso) Cyl(cx, cy, r, z0, z1 float64, col RGB) {
	if s.SP != nil {
		var pts [][3]float64
		for i := 0; i < 16; i++ {
			t := float64(i) / 16 * 2 * math.Pi
			pts = append(pts, [3]float64{cx + math.Cos(t)*r, cy + math.Sin(t)*r, z1})
		}
		s.shadowHull(pts)
		return
	}
	n := 28
	for i := 0; i < n; i++ {
		a0 := float64(i) / float64(n) * 2 * math.Pi
		a1 := float64(i+1) / float64(n) * 2 * math.Pi
		am := (a0 + a1) / 2
		nx, ny := math.Cos(am), math.Sin(am)
		if nx+ny < -0.05 {
			continue // Rückseite
		}
		// Licht von links-oben: Richtung (-0.3, 1) in Iso -> wir nehmen y-Seite heller
		l := 0.62 + 0.3*clampf(ny*0.8-nx*0.3+0.3, 0, 1)
		x0, y0 := cx+math.Cos(a0)*r, cy+math.Sin(a0)*r
		x1, y1 := cx+math.Cos(a1)*r, cy+math.Sin(a1)*r
		s.Quad(col.Mul(l), 1, [3]float64{x0, y0, z0}, [3]float64{x1, y1, z0}, [3]float64{x1, y1, z1}, [3]float64{x0, y0, z1})
	}
	s.Disc(cx, cy, r, z1, col.Mul(1.05), 1)
}

func (s *Iso) Disc(cx, cy, r, z float64, col RGB, a float64) {
	if s.SP != nil {
		return
	}
	n := 32
	pts := make([]P, n)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n) * 2 * math.Pi
		pts[i] = s.Pt(cx+math.Cos(t)*r, cy+math.Sin(t)*r, z)
	}
	s.P.Poly(pts, col, a)
}

// Kuppel
func (s *Iso) Dome(cx, cy, r, z0 float64, col RGB, a float64) {
	if s.SP != nil {
		var pts [][3]float64
		for i := 0; i < 16; i++ {
			t := float64(i) / 16 * 2 * math.Pi
			pts = append(pts, [3]float64{cx + math.Cos(t)*r*0.8, cy + math.Sin(t)*r*0.8, z0 + r*0.7})
		}
		s.shadowHull(pts)
		return
	}
	steps := 10
	for i := 0; i < steps; i++ {
		t0 := float64(i) / float64(steps) * math.Pi / 2
		rr := r * math.Cos(t0)
		z := z0 + r*math.Sin(t0)*0.9
		l := 0.7 + 0.35*float64(i)/float64(steps)
		s.Disc(cx, cy, rr, z, col.Mul(l), a)
	}
	// Glanzlicht
	hp := s.Pt(cx-r*0.25, cy+r*0.1, z0+r*0.75)
	s.P.Ellipse(hp.X, hp.Y, r*s.HW*0.35, r*s.HH*0.5, RGB{255, 255, 255}, 0.35*a)
}

// Gedrehter Quader (für Schiffe usw.), Winkel in Radiant in der Weltebene
func (s *Iso) RBox(cx, cy, l, w, ang, z0, z1 float64, col RGB) {
	ca, sa := math.Cos(ang), math.Sin(ang)
	pts := [4][2]float64{{l / 2, w / 2}, {-l / 2, w / 2}, {-l / 2, -w / 2}, {l / 2, -w / 2}}
	var c [4][2]float64
	for i, q := range pts {
		c[i] = [2]float64{cx + q[0]*ca - q[1]*sa, cy + q[0]*sa + q[1]*ca}
	}
	s.RPrism(c[:], z0, z1, col)
}

// Prisma aus beliebigem konvexen Grundriss (Gegen-Uhrzeigersinn egal)
func (s *Iso) RPrism(c [][2]float64, z0, z1 float64, col RGB) {
	if s.SP != nil {
		var pts [][3]float64
		for _, q := range c {
			pts = append(pts, [3]float64{q[0], q[1], z1})
		}
		s.shadowHull(pts)
		return
	}
	n := len(c)
	// Zentrum
	var mx, my float64
	for _, q := range c {
		mx += q[0]
		my += q[1]
	}
	mx /= float64(n)
	my /= float64(n)
	type face struct {
		d   float64
		i   int
		lit float64
	}
	var faces []face
	for i := 0; i < n; i++ {
		a, b := c[i], c[(i+1)%n]
		ex, ey := b[0]-a[0], b[1]-a[1]
		nx, ny := ey, -ex
		// nach außen ausrichten
		if (a[0]+b[0])/2-mx < 0 && nx > 0 || false {
		}
		if nx*((a[0]+b[0])/2-mx)+ny*((a[1]+b[1])/2-my) < 0 {
			nx, ny = -nx, -ny
		}
		l := math.Hypot(nx, ny)
		nx, ny = nx/l, ny/l
		if nx+ny <= 0.02 {
			continue
		}
		lit := 0.6 + 0.28*clampf((ny-nx*0.6+1)/2, 0, 1)
		faces = append(faces, face{(a[0] + b[0] + a[1] + b[1]) / 2, i, lit})
	}
	sort.Slice(faces, func(i, j int) bool { return faces[i].d < faces[j].d })
	for _, f := range faces {
		a, b := c[f.i], c[(f.i+1)%n]
		s.Quad(col.Mul(f.lit), 1, [3]float64{a[0], a[1], z0}, [3]float64{b[0], b[1], z0}, [3]float64{b[0], b[1], z1}, [3]float64{a[0], a[1], z1})
	}
	top := make([][3]float64, n)
	for i, q := range c {
		top[i] = [3]float64{q[0], q[1], z1}
	}
	s.Quad(col, 1, top...)
}

// Flache Kachelfläche
func (s *Iso) Floor(x0, y0, x1, y1, z float64, col RGB, a float64) {
	if s.SP != nil {
		return
	}
	s.Quad(col, a, [3]float64{x0, y0, z}, [3]float64{x1, y0, z}, [3]float64{x1, y1, z}, [3]float64{x0, y1, z})
}

// überträgt die Schattenebene als halbtransparenten Schatten
func (p *Painter) ApplyShadow(sp *Painter, strength float32) {
	col := [3]float32{0.10, 0.12, 0.20}
	for i := 0; i < len(p.buf); i += 4 {
		a := sp.buf[i+3] * strength
		if a <= 0 {
			continue
		}
		ia := 1 - a
		p.buf[i] = col[0]*a + p.buf[i]*ia
		p.buf[i+1] = col[1]*a + p.buf[i+1]*ia
		p.buf[i+2] = col[2]*a + p.buf[i+2]*ia
		p.buf[i+3] = a + p.buf[i+3]*ia
	}
}

// Walmdach (Terrakotta). Grat entlang der längeren Seite.
func (s *Iso) HipRoof(x0, y0, x1, y1, z, h float64, col RGB) {
	o := 0.06 // Dachüberstand
	x0, y0, x1, y1 = x0-o, y0-o, x1+o, y1+o
	lines := col.Mul(0.78)
	if x1-x0 >= y1-y0 {
		d := (y1 - y0) / 2
		cy := (y0 + y1) / 2
		r0, r1 := [3]float64{x0 + d, cy, z + h}, [3]float64{x1 - d, cy, z + h}
		s.Quad(col.Mul(1.06), 1, [3]float64{x0, y0, z}, [3]float64{x1, y0, z}, r1, r0)
		s.Quad(col.Mul(0.98), 1, [3]float64{x0, y0, z}, r0, [3]float64{x0, y1, z})
		s.Quad(col.Mul(0.84), 1, [3]float64{x0, y1, z}, [3]float64{x1, y1, z}, r1, r0)
		s.Quad(col.Mul(0.66), 1, [3]float64{x1, y0, z}, r1, [3]float64{x1, y1, z})
		if s.SP == nil {
			n := int((x1 - x0) * 5)
			for i := 1; i < n; i++ {
				t := float64(i) / float64(n)
				bx := x0 + (x1-x0)*t
				rx := clampf(bx, r0[0], r1[0])
				a, b := s.Pt(bx, y1, z), s.Pt(rx, cy, z+h)
				s.P.LineW(a.X, a.Y, b.X, b.Y, math.Max(0.6, s.HW/40), lines, 0.55)
			}
		}
	} else {
		d := (x1 - x0) / 2
		cx := (x0 + x1) / 2
		r0, r1 := [3]float64{cx, y0 + d, z + h}, [3]float64{cx, y1 - d, z + h}
		s.Quad(col.Mul(1.06), 1, [3]float64{x0, y0, z}, [3]float64{x1, y0, z}, r0)
		s.Quad(col.Mul(0.98), 1, [3]float64{x0, y0, z}, r0, r1, [3]float64{x0, y1, z})
		s.Quad(col.Mul(0.84), 1, [3]float64{x0, y1, z}, [3]float64{x1, y1, z}, r1)
		s.Quad(col.Mul(0.66), 1, [3]float64{x1, y0, z}, r0, r1, [3]float64{x1, y1, z})
		if s.SP == nil {
			n := int((y1 - y0) * 5)
			for i := 1; i < n; i++ {
				t := float64(i) / float64(n)
				by := y0 + (y1-y0)*t
				ry := clampf(by, r0[1], r1[1])
				a, b := s.Pt(x1, by, z), s.Pt(cx, ry, z+h)
				s.P.LineW(a.X, a.Y, b.X, b.Y, math.Max(0.6, s.HW/40), lines, 0.55)
			}
		}
	}
}

// Satteldach mit Giebel (Tempel). Grat entlang x.
func (s *Iso) GableRoof(x0, y0, x1, y1, z, h float64, col, gable RGB) {
	o := 0.05
	y0, y1 = y0-o, y1+o
	cy := (y0 + y1) / 2
	s.Quad(col.Mul(1.05), 1, [3]float64{x0, y0, z}, [3]float64{x1, y0, z}, [3]float64{x1, cy, z + h}, [3]float64{x0, cy, z + h})
	s.Quad(col.Mul(0.84), 1, [3]float64{x0, y1, z}, [3]float64{x1, y1, z}, [3]float64{x1, cy, z + h}, [3]float64{x0, cy, z + h})
	s.Quad(gable.Mul(0.7), 1, [3]float64{x1, y0, z}, [3]float64{x1, cy, z + h}, [3]float64{x1, y1, z})
	if s.SP == nil {
		// Giebelrahmen
		a, b, c := s.Pt(x1+0.01, y0, z), s.Pt(x1+0.01, cy, z+h), s.Pt(x1+0.01, y1, z)
		w := math.Max(0.8, s.HW/30)
		s.P.LineW(a.X, a.Y, b.X, b.Y, w, gable, 1)
		s.P.LineW(b.X, b.Y, c.X, c.Y, w, gable, 1)
		s.P.LineW(a.X, a.Y, c.X, c.Y, w, gable, 1)
		n := int((x1 - x0) * 5)
		for i := 1; i < n; i++ {
			bx := x0 + (x1-x0)*float64(i)/float64(n)
			p, q := s.Pt(bx, y1, z), s.Pt(bx, cy, z+h)
			s.P.LineW(p.X, p.Y, q.X, q.Y, math.Max(0.6, s.HW/40), col.Mul(0.7), 0.5)
		}
	}
}

// Säulenreihe entlang der Vorderseite y (links sichtbar)
func (s *Iso) ColsL(x0, x1, y, z0, z1 float64, n int, col RGB) {
	r := 0.055
	for i := 0; i < n; i++ {
		x := x0 + (x1-x0)*(float64(i)+0.5)/float64(n)
		s.Cyl(x, y, r, z0, z1, col)
		s.Box(x-r*1.4, y-r*1.4, x+r*1.4, y+r*1.4, z1-0.04, z1, col.Mul(1.02))
	}
}

// Säulenreihe entlang der Vorderseite x (rechts sichtbar)
func (s *Iso) ColsR(y0, y1, x, z0, z1 float64, n int, col RGB) {
	r := 0.055
	for i := 0; i < n; i++ {
		y := y0 + (y1-y0)*(float64(i)+0.5)/float64(n)
		s.Cyl(x, y, r, z0, z1, col)
		s.Box(x-r*1.4, y-r*1.4, x+r*1.4, y+r*1.4, z1-0.04, z1, col.Mul(1.02))
	}
}

// Bogenreihe (Arkaden) auf der linken Fassade (y fest)
func (s *Iso) ArchesL(x0, x1, y, z0, z1 float64, n int, col RGB, a float64) {
	w := (x1 - x0) / float64(n)
	for i := 0; i < n; i++ {
		cx := x0 + w*(float64(i)+0.5)
		r := w * 0.32
		top := z1 - r*1.1
		var pts [][3]float64
		pts = append(pts, [3]float64{cx - r, y + 0.002, z0}, [3]float64{cx + r, y + 0.002, z0})
		for k := 0; k <= 8; k++ {
			t := float64(k) / 8 * math.Pi
			pts = append(pts, [3]float64{cx + math.Cos(t)*r, y + 0.002, top + math.Sin(t)*r*1.6})
		}
		s.Quad(col, a, pts...)
	}
}

func (s *Iso) ArchesR(y0, y1, x, z0, z1 float64, n int, col RGB, a float64) {
	w := (y1 - y0) / float64(n)
	for i := 0; i < n; i++ {
		cy := y0 + w*(float64(i)+0.5)
		r := w * 0.32
		top := z1 - r*1.1
		var pts [][3]float64
		pts = append(pts, [3]float64{x + 0.002, cy + r, z0}, [3]float64{x + 0.002, cy - r, z0})
		for k := 0; k <= 8; k++ {
			t := float64(k) / 8 * math.Pi
			pts = append(pts, [3]float64{x + 0.002, cy - math.Cos(t)*r, top + math.Sin(t)*r*1.6})
		}
		s.Quad(col, a, pts...)
	}
}
