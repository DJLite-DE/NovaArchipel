package game

import (
	_ "embed"
	"encoding/binary"
	"math"
)

// Framebuffer: 0x00RRGGBB pro Pixel (passt direkt zu Windows-DIB 32bpp)
type Canvas struct {
	W, H int
	Pix  []uint32
	// Clip-Rechteck
	CX0, CY0, CX1, CY1 int
}

func NewCanvas(w, h int) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]uint32, w*h)}
	c.ResetClip()
	return c
}

func (c *Canvas) ResetClip() { c.CX0, c.CY0, c.CX1, c.CY1 = 0, 0, c.W, c.H }
func (c *Canvas) SetClip(x0, y0, x1, y1 int) {
	c.CX0, c.CY0, c.CX1, c.CY1 = imax(0, x0), imax(0, y0), imin(c.W, x1), imin(c.H, y1)
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func clampi(v, a, b int) int {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}
func clampf(v, a, b float64) float64 {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}

type RGB struct{ R, G, B uint8 }

func (c RGB) U() uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }
func (c RGB) Mul(f float64) RGB {
	return RGB{uint8(clampf(float64(c.R)*f, 0, 255)), uint8(clampf(float64(c.G)*f, 0, 255)), uint8(clampf(float64(c.B)*f, 0, 255))}
}
func (c RGB) Mix(o RGB, t float64) RGB {
	return RGB{uint8(float64(c.R)*(1-t) + float64(o.R)*t), uint8(float64(c.G)*(1-t) + float64(o.G)*t), uint8(float64(c.B)*(1-t) + float64(o.B)*t)}
}
func hex(v uint32) RGB { return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)} }

// vormultiplizierte Mischung inkl. Alphakanal (für die UI-Ebene über der 3D-Szene)
func blend(dst uint32, col uint32, a uint32) uint32 {
	if a >= 255 {
		return col | 0xFF000000
	}
	ia := 255 - a
	rb := ((col&0xFF00FF)*a + (dst&0xFF00FF)*ia) >> 8 & 0xFF00FF
	g := ((col&0x00FF00)*a + (dst&0x00FF00)*ia) >> 8 & 0x00FF00
	da := dst >> 24
	na := a + (da*ia)>>8
	if na > 255 {
		na = 255
	}
	return na<<24 | rb | g
}

func (c *Canvas) Clear(col RGB) {
	v := col.U()
	for i := range c.Pix {
		c.Pix[i] = v
	}
}

func (c *Canvas) FillRect(x, y, w, h int, col RGB, alpha int) {
	x0, y0 := imax(x, c.CX0), imax(y, c.CY0)
	x1, y1 := imin(x+w, c.CX1), imin(y+h, c.CY1)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	v := col.U()
	a := uint32(clampi(alpha, 0, 255))
	for yy := y0; yy < y1; yy++ {
		row := c.Pix[yy*c.W:]
		if a >= 255 {
			for xx := x0; xx < x1; xx++ {
				row[xx] = v | 0xFF000000
			}
		} else {
			for xx := x0; xx < x1; xx++ {
				row[xx] = blend(row[xx], v, a)
			}
		}
	}
}

// Vertikaler Farbverlauf
func (c *Canvas) GradRect(x, y, w, h int, top, bot RGB, alpha int) {
	for i := 0; i < h; i++ {
		t := float64(i) / float64(imax(1, h-1))
		c.FillRect(x, y+i, w, 1, top.Mix(bot, t), alpha)
	}
}

func (c *Canvas) Rect(x, y, w, h int, col RGB, alpha int) {
	c.FillRect(x, y, w, 1, col, alpha)
	c.FillRect(x, y+h-1, w, 1, col, alpha)
	c.FillRect(x, y+1, 1, h-2, col, alpha)
	c.FillRect(x+w-1, y+1, 1, h-2, col, alpha)
}

func (c *Canvas) Plot(x, y int, col uint32, a uint32) {
	if x < c.CX0 || y < c.CY0 || x >= c.CX1 || y >= c.CY1 {
		return
	}
	i := y*c.W + x
	c.Pix[i] = blend(c.Pix[i], col, a)
}

// Antialiasierte Linie (Xiaolin-Wu-artig, einfach)
func (c *Canvas) Line(x0, y0, x1, y1 float64, col RGB, alpha int) {
	dx, dy := x1-x0, y1-y0
	n := int(math.Max(math.Abs(dx), math.Abs(dy))) + 1
	v := col.U()
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := x0+dx*t, y0+dy*t
		if math.Abs(dx) > math.Abs(dy) {
			yi := math.Floor(y)
			f := y - yi
			c.Plot(int(x), int(yi), v, uint32(float64(alpha)*(1-f)))
			c.Plot(int(x), int(yi)+1, v, uint32(float64(alpha)*f))
		} else {
			xi := math.Floor(x)
			f := x - xi
			c.Plot(int(xi), int(y), v, uint32(float64(alpha)*(1-f)))
			c.Plot(int(xi)+1, int(y), v, uint32(float64(alpha)*f))
		}
	}
}

// Gefüllter Kreis mit weichem Rand
func (c *Canvas) Circle(cx, cy, r float64, col RGB, alpha int) {
	v := col.U()
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			cov := clampf(r-d+0.5, 0, 1)
			if cov > 0 {
				c.Plot(x, y, v, uint32(float64(alpha)*cov))
			}
		}
	}
}

// Weicher Leuchtpunkt (additiv-ähnlich)
func (c *Canvas) Glow(cx, cy, r float64, col RGB, alpha int) {
	v := col.U()
	for y := int(cy - r); y <= int(cy+r); y++ {
		for x := int(cx - r); x <= int(cx+r); x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) / r
			if d < 1 {
				f := (1 - d) * (1 - d)
				c.Plot(x, y, v, uint32(float64(alpha)*f))
			}
		}
	}
}

// Isometrische Ellipsen-Kontur (für Radien)
func (c *Canvas) EllipseOutline(cx, cy, rx, ry float64, col RGB, alpha int) {
	n := int(rx+ry) * 2
	px, py := cx+rx, cy
	for i := 1; i <= n; i++ {
		a := float64(i) / float64(n) * 2 * math.Pi
		x, y := cx+math.Cos(a)*rx, cy+math.Sin(a)*ry
		c.Line(px, py, x, y, col, alpha)
		px, py = x, y
	}
}

// ---------------------------------------------------------------- Moderner Stil: Milchglas, Rundungen, Cyan-Akzent

var (
	uiNavy    = RGB{14, 20, 30}
	uiNavyL   = RGB{34, 44, 60}
	uiGold    = RGB{90, 210, 255} // Akzent (Cyan)
	uiGoldD   = RGB{70, 90, 115}  // dezente Kontur
	uiPurple  = RGB{30, 120, 200} // Auswahl
	uiPurpleL = RGB{50, 150, 230}
)

// Rechteck mit abgerundeten, geglätteten Ecken
func (c *Canvas) RoundRect(x, y, w, h, r int, col RGB, alpha int) {
	if r*2 > h {
		r = h / 2
	}
	if r*2 > w {
		r = w / 2
	}
	v := col.U()
	for yy := 0; yy < h; yy++ {
		py := y + yy
		if py < c.CY0 || py >= c.CY1 {
			continue
		}
		for xx := 0; xx < w; xx++ {
			px := x + xx
			if px < c.CX0 || px >= c.CX1 {
				continue
			}
			cov := 1.0
			var cx, cy float64
			inCorner := false
			if xx < r && yy < r {
				cx, cy, inCorner = float64(r), float64(r), true
			} else if xx >= w-r && yy < r {
				cx, cy, inCorner = float64(w-r), float64(r), true
			} else if xx < r && yy >= h-r {
				cx, cy, inCorner = float64(r), float64(h-r), true
			} else if xx >= w-r && yy >= h-r {
				cx, cy, inCorner = float64(w-r), float64(h-r), true
			}
			if inCorner {
				d := math.Hypot(float64(xx)+0.5-cx, float64(yy)+0.5-cy)
				cov = clampf(float64(r)-d+0.5, 0, 1)
				if cov <= 0 {
					continue
				}
			}
			i := py*c.W + px
			c.Pix[i] = blend(c.Pix[i], v, uint32(float64(alpha)*cov))
		}
	}
}

// Kontur eines abgerundeten Rechtecks
func (c *Canvas) RoundRectLine(x, y, w, h, r int, col RGB, alpha int) {
	v := col.U()
	fr := float64(r)
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			// Abstand zum Rand der Rundform
			px, py := float64(xx)+0.5, float64(yy)+0.5
			qx := math.Max(math.Abs(px-float64(w)/2)-(float64(w)/2-fr), 0)
			qy := math.Max(math.Abs(py-float64(h)/2)-(float64(h)/2-fr), 0)
			d := math.Hypot(qx, qy) - fr
			if math.Abs(qx) == 0 && math.Abs(qy) == 0 {
				d = math.Max(math.Abs(px-float64(w)/2)-float64(w)/2, math.Abs(py-float64(h)/2)-float64(h)/2)
			}
			cov := clampf(1-math.Abs(d+0.5), 0, 1)
			if cov > 0 {
				c.Plot(x+xx, y+yy, v, uint32(float64(alpha)*cov))
			}
		}
	}
}

// (Kompatibilität) dezenter Untergrund
func (c *Canvas) Fabric(x, y, w, h int, col RGB, alpha int) {
	c.RoundRect(x, y, w, h, 6, col, alpha*3/4)
}

// (Kompatibilität) Kopfzeilen-Verlauf
func (c *Canvas) Marble(x, y, w, h int, alpha int) {
	for i := 0; i < h; i++ {
		t := float64(i) / float64(imax(1, h))
		c.FillRect(x, y+i, w, 1, RGB{40, 150, 210}, int(float64(alpha)*0.16*(1-t)))
	}
}

func (c *Canvas) GoldFrame(x, y, w, h int) {
	c.RoundRectLine(x, y, w, h, 10, RGB{255, 255, 255}, 46)
}

// (Kompatibilität) feine Akzentlinie
func (c *Canvas) Meander(x, y, w int, col RGB, alpha int) {
	for i := 0; i < w; i++ {
		t := float64(i) / float64(w)
		a := math.Sin(t*math.Pi) * float64(alpha)
		c.FillRect(x+i, y+3, 1, 2, uiGold, int(a))
	}
}

func (c *Canvas) Panel(x, y, w, h int) {
	c.RoundRect(x, y, w, h, 10, RGB{10, 16, 26}, 110)
	c.RoundRectLine(x, y, w, h, 10, RGB{255, 255, 255}, 46)
	c.FillRect(x+12, y, w-24, 1, RGB{255, 255, 255}, 60)
}

func (c *Canvas) PanelHead(x, y, w, h, headH int) {
	c.Panel(x, y, w, h)
}

// ---------------------------------------------------------------- Sprites

// Sprite: vormultipliziertes ARGB
type Sprite struct {
	W, H   int
	OX, OY int // Ankerpunkt im Sprite
	Pix    []uint32
}

func (c *Canvas) Blit(s *Sprite, x, y int) {
	if s == nil {
		return
	}
	x -= s.OX
	y -= s.OY
	sx0, sy0 := imax(0, c.CX0-x), imax(0, c.CY0-y)
	sx1, sy1 := imin(s.W, c.CX1-x), imin(s.H, c.CY1-y)
	for sy := sy0; sy < sy1; sy++ {
		srow := s.Pix[sy*s.W:]
		base := (y+sy)*c.W + x
		for sx := sx0; sx < sx1; sx++ {
			p := srow[sx]
			di := base + sx
			a := p >> 24
			if a == 0 {
				continue
			}
			if a == 255 {
				c.Pix[di] = p
				continue
			}
			d := c.Pix[di]
			ia := 255 - a
			rb := ((d&0xFF00FF)*ia>>8)&0xFF00FF + p&0xFF00FF
			g := ((d&0x00FF00)*ia>>8)&0x00FF00 + p&0x00FF00
			na := a + ((d>>24)*ia)>>8
			if na > 255 {
				na = 255
			}
			c.Pix[di] = na<<24 | rb&0xFF00FF | g&0x00FF00
		}
	}
}

// Getöntes Blit (für Bauvorschau)
func (c *Canvas) BlitTint(s *Sprite, x, y int, tint RGB, alpha int) {
	if s == nil {
		return
	}
	x -= s.OX
	y -= s.OY
	tv := tint.U()
	for sy := 0; sy < s.H; sy++ {
		yy := y + sy
		if yy < c.CY0 || yy >= c.CY1 {
			continue
		}
		for sx := 0; sx < s.W; sx++ {
			xx := x + sx
			if xx < c.CX0 || xx >= c.CX1 {
				continue
			}
			p := s.Pix[sy*s.W+sx]
			a := p >> 24
			if a == 0 {
				continue
			}
			// Original leicht + Tönung
			r, g, b := (p>>16)&255, (p>>8)&255, p&255
			if a > 0 {
				r, g, b = r*255/a, g*255/a, b*255/a
			}
			col := blend(r<<16|g<<8|b, tv, 120)
			i := yy*c.W + xx
			c.Pix[i] = blend(c.Pix[i], col, a*uint32(alpha)/255)
		}
	}
}

// Skaliertes Blit (nearest + Box) für Icons/Thumbnails
func (c *Canvas) BlitScaled(s *Sprite, x, y, w, h int) {
	if s == nil || w <= 0 || h <= 0 {
		return
	}
	for yy := 0; yy < h; yy++ {
		dy := y + yy
		if dy < c.CY0 || dy >= c.CY1 {
			continue
		}
		sy0 := yy * s.H / h
		sy1 := imax(sy0+1, (yy+1)*s.H/h)
		for xx := 0; xx < w; xx++ {
			dx := x + xx
			if dx < c.CX0 || dx >= c.CX1 {
				continue
			}
			sx0 := xx * s.W / w
			sx1 := imax(sx0+1, (xx+1)*s.W/w)
			var ar, rr, gg, bb, n uint32
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					p := s.Pix[sy*s.W+sx]
					ar += p >> 24
					rr += (p >> 16) & 255
					gg += (p >> 8) & 255
					bb += p & 255
					n++
				}
			}
			ar /= n
			rr /= n
			gg /= n
			bb /= n
			if ar == 0 {
				continue
			}
			i := dy*c.W + dx
			d := c.Pix[i]
			ia := 255 - ar
			r := ((d>>16)&255)*ia/255 + rr
			g := ((d>>8)&255)*ia/255 + gg
			b := (d&255)*ia/255 + bb
			na := ar + (d>>24)*ia/255
			c.Pix[i] = imin32(na, 255)<<24 | imin32(r, 255)<<16 | imin32(g, 255)<<8 | imin32(b, 255)
		}
	}
}

func imin32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------------- Schrift

//go:embed font.bin
var fontData []byte

type Glyph struct {
	Adv, XOff, YOff int
	W, H            int
	Data            []byte
}

type Font struct {
	Size, LineH, Ascent int
	G                   map[rune]*Glyph
}

const (
	FSmall = iota
	FNormal
	FBold
	FBig
	FTitle
	FLight
	FSerif
)

var Fonts []*Font

func init() {
	d := fontData
	p := 4
	u32 := func() int { v := binary.LittleEndian.Uint32(d[p:]); p += 4; return int(v) }
	i16 := func() int { v := int16(binary.LittleEndian.Uint16(d[p:])); p += 2; return int(v) }
	u16 := func() int { v := binary.LittleEndian.Uint16(d[p:]); p += 2; return int(v) }
	n := u32()
	for i := 0; i < n; i++ {
		f := &Font{G: map[rune]*Glyph{}}
		f.Size = u32()
		f.LineH = u32()
		f.Ascent = u32()
		ng := u32()
		for j := 0; j < ng; j++ {
			r := rune(u32())
			g := &Glyph{}
			g.Adv = i16()
			g.XOff = i16()
			g.YOff = i16()
			g.W = u16()
			g.H = u16()
			g.Data = d[p : p+g.W*g.H]
			p += g.W * g.H
			f.G[r] = g
		}
		Fonts = append(Fonts, f)
	}
}

func TextW(font int, s string) int {
	s = T(s)
	f := Fonts[font]
	w := 0
	for _, r := range s {
		if g, ok := f.G[r]; ok {
			w += g.Adv
		} else {
			w += f.Size / 2
		}
	}
	return w
}

// Text zeichnen; y = Oberkante
func (c *Canvas) Text(font int, x, y int, s string, col RGB) int {
	return c.TextA(font, x, y, s, col, 255)
}

func (c *Canvas) TextA(font int, x, y int, s string, col RGB, alpha int) int {
	s = T(s)
	f := Fonts[font]
	v := col.U()
	for _, r := range s {
		g, ok := f.G[r]
		if !ok {
			x += f.Size / 2
			continue
		}
		for gy := 0; gy < g.H; gy++ {
			yy := y + g.YOff + gy
			if yy < c.CY0 || yy >= c.CY1 {
				continue
			}
			for gx := 0; gx < g.W; gx++ {
				a := g.Data[gy*g.W+gx]
				if a == 0 {
					continue
				}
				xx := x + g.XOff + gx
				if xx < c.CX0 || xx >= c.CX1 {
					continue
				}
				i := yy*c.W + xx
				c.Pix[i] = blend(c.Pix[i], v, uint32(a)*uint32(alpha)/255)
			}
		}
		x += g.Adv
	}
	return x
}

// Text mit Schatten
func (c *Canvas) TextS(font int, x, y int, s string, col RGB) int {
	c.TextA(font, x+1, y+1, s, RGB{0, 0, 0}, 170)
	return c.Text(font, x, y, s, col)
}

func (c *Canvas) TextC(font int, cx, y int, s string, col RGB) {
	c.TextS(font, cx-TextW(font, s)/2, y, s, col)
}

func (c *Canvas) TextR(font int, rx, y int, s string, col RGB) {
	c.Text(font, rx-TextW(font, s), y, s, col)
}

// Zeilenumbruch
func WrapText(font int, s string, maxW int) []string {
	s = T(s)
	var lines []string
	cur := ""
	word := ""
	flush := func() {
		if cur == "" {
			cur = word
		} else if TextW(font, cur+" "+word) <= maxW {
			cur += " " + word
		} else {
			lines = append(lines, cur)
			cur = word
		}
		word = ""
	}
	for _, r := range s {
		if r == ' ' {
			flush()
		} else if r == '\n' {
			flush()
			lines = append(lines, cur)
			cur = ""
		} else {
			word += string(r)
		}
	}
	if word != "" {
		flush()
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// gefülltes Dreieck (Scanline)
func (c *Canvas) Tri(x0, y0, x1, y1, x2, y2 float64, col RGB, alpha int) {
	minY := int(math.Floor(math.Min(y0, math.Min(y1, y2))))
	maxY := int(math.Ceil(math.Max(y0, math.Max(y1, y2))))
	pts := [][2]float64{{x0, y0}, {x1, y1}, {x2, y2}}
	for y := minY; y <= maxY; y++ {
		fy := float64(y) + 0.5
		xa, xb := 1e9, -1e9
		for i := 0; i < 3; i++ {
			a, b := pts[i], pts[(i+1)%3]
			if (a[1] <= fy && b[1] > fy) || (b[1] <= fy && a[1] > fy) {
				x := a[0] + (fy-a[1])/(b[1]-a[1])*(b[0]-a[0])
				xa, xb = math.Min(xa, x), math.Max(xb, x)
			}
		}
		if xa <= xb {
			c.FillRect(int(xa), y, int(xb-xa)+1, 1, col, alpha)
		}
	}
}
