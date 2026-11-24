package game

import (
	"math"
)

func pnoise(u, v float64, k int, seed int) float64 {
	x, y := u*float64(k), v*float64(k)
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := smooth(x-float64(xi)), smooth(y-float64(yi))
	m := func(a int) int { return ((a % k) + k) % k }
	a := hash2(m(xi), m(yi), seed)
	b := hash2(m(xi+1), m(yi), seed)
	c := hash2(m(xi), m(yi+1), seed)
	d := hash2(m(xi+1), m(yi+1), seed)
	return a + (b-a)*fx + (c-a)*fy + (a-b-c+d)*fx*fy
}

var (
	colDeep    = hex(0x10507f)
	colMid     = hex(0x1a7fae)
	colShallow = hex(0x3cc8c8)
	colSand    = hex(0xecd8a4)
	colWetSand = hex(0xc9b585)
	colGrass   = [2]RGB{hex(0x79b04a), hex(0xb0a85a)}
	colGrass2  = [2]RGB{hex(0x5a9640), hex(0x948c48)}
	colRock    = hex(0xbcae96)
)

// ------------------------------------------------------------ Icons

var iconCache = map[string]*Sprite{}

func GoodIcon(g, size int) *Sprite {
	key := Tf("g%d_%d", g, size)
	if s, ok := iconCache[key]; ok {
		return s
	}
	p := NewPainter(size, size, 4)
	f := float64(size)
	c := Goods[g].Col
	cx, cy := f/2, f/2
	dark := c.Mul(0.6)
	p.Ellipse(cx, cy, f*0.47, f*0.47, hex(0x0e2236), 0.9)
	p.Ellipse(cx, cy, f*0.47, f*0.47, c, 0.18)
	switch g {
	case GSchrott:
		p.Poly([]P{{f * .2, f * .7}, {f * .35, f * .3}, {f * .55, f * .45}, {f * .5, f * .75}}, c, 1)
		p.Poly([]P{{f * .5, f * .7}, {f * .62, f * .3}, {f * .82, f * .5}, {f * .72, f * .75}}, dark, 1)
	case GBaustoffe:
		for i := 0; i < 3; i++ {
			y := f * (0.3 + float64(i)*0.15)
			p.Poly([]P{{f * .22, y}, {f * .78, y}, {f * .78, y + f*.11}, {f * .22, y + f*.11}}, c.Mul(1-float64(i)*0.12), 1)
		}
	case GAlgen:
		p.Ellipse(cx-f*.1, cy, f*.12, f*.28, c, 1)
		p.Ellipse(cx+f*.12, cy+f*.05, f*.1, f*.22, c.Mul(0.8), 1)
	case GNahrung:
		p.Poly([]P{{f * .25, f * .35}, {f * .75, f * .35}, {f * .72, f * .75}, {f * .28, f * .75}}, c, 1)
		p.Poly([]P{{f * .25, f * .35}, {f * .75, f * .35}, {f * .7, f * .27}, {f * .3, f * .27}}, c.Mul(1.2), 1)
	case GWasser:
		p.Poly([]P{{cx, f * .18}, {f * .7, f * .55}, {f * .3, f * .55}}, c, 1)
		p.Ellipse(cx, f*.6, f*.2, f*.18, c, 1)
		p.Ellipse(cx-f*.07, f*.6, f*.05, f*.07, RGB{255, 255, 255}, 0.7)
	case GErz:
		p.Ellipse(cx-f*.1, cy+f*.08, f*.2, f*.16, c, 1)
		p.Ellipse(cx+f*.12, cy-f*.05, f*.15, f*.13, dark, 1)
	case GStahl:
		p.Poly([]P{{f * .2, f * .3}, {f * .8, f * .3}, {f * .8, f * .38}, {f * .56, f * .38}, {f * .56, f * .62}, {f * .8, f * .62}, {f * .8, f * .7}, {f * .2, f * .7}, {f * .2, f * .62}, {f * .44, f * .62}, {f * .44, f * .38}, {f * .2, f * .38}}, c, 1)
	case GKaffee:
		p.Ellipse(cx, cy, f*.18, f*.26, c, 1)
		p.LineW(cx, cy-f*.22, cx, cy+f*.22, f*.04, dark.Mul(0.6), 1)
	case GSynthkaffee:
		p.Poly([]P{{f * .28, f * .3}, {f * .66, f * .3}, {f * .62, f * .72}, {f * .32, f * .72}}, hex(0xf0f0f0), 1)
		p.Ellipse(f*.7, f*.48, f*.08, f*.1, hex(0xf0f0f0), 1)
		p.Poly([]P{{f * .3, f * .33}, {f * .64, f * .33}, {f * .63, f * .4}, {f * .31, f * .4}}, c, 1)
	case GFasern:
		for i := 0; i < 3; i++ {
			x := f * (0.35 + float64(i)*0.15)
			p.LineW(x, f*.75, x+f*.05, f*.25, f*.06, c, 1)
		}
	case GTextil:
		p.Poly([]P{{f * .3, f * .25}, {f * .7, f * .25}, {f * .8, f * .4}, {f * .7, f * .45}, {f * .7, f * .75}, {f * .3, f * .75}, {f * .3, f * .45}, {f * .2, f * .4}}, c, 1)
	case GSilizium:
		p.Poly([]P{{cx, f * .18}, {f * .7, f * .45}, {cx, f * .82}, {f * .3, f * .45}}, c, 1)
		p.Poly([]P{{cx, f * .18}, {f * .7, f * .45}, {cx, f * .82}}, dark, 1)
	case GChips:
		p.Poly([]P{{f * .3, f * .3}, {f * .7, f * .3}, {f * .7, f * .7}, {f * .3, f * .7}}, hex(0x1f2a36), 1)
		for i := 0; i < 3; i++ {
			y := f * (0.36 + float64(i)*0.12)
			p.LineW(f*.2, y, f*.8, y, f*.04, c, 1)
		}
		p.Poly([]P{{f * .38, f * .38}, {f * .62, f * .38}, {f * .62, f * .62}, {f * .38, f * .62}}, c.Mul(0.7), 1)
	case GHelium3:
		p.Ellipse(cx, cy, f*.26, f*.26, hex(0x7ab8ff), 0.5)
		p.Ellipse(cx, cy, f*.1, f*.1, c, 1)
		for k := 0; k < 3; k++ {
			a := float64(k) * 2.094
			p.Ellipse(cx+math.Cos(a)*f*.22, cy+math.Sin(a)*f*.22, f*.06, f*.06, hex(0xb0e0ff), 1)
		}
	case GHolo:
		p.Poly([]P{{f * .25, f * .3}, {f * .75, f * .3}, {f * .75, f * .72}, {f * .25, f * .72}}, hex(0x1f2a36), 1)
		p.Poly([]P{{f * .3, f * .35}, {f * .7, f * .35}, {f * .7, f * .67}, {f * .3, f * .67}}, c, 0.8)
		p.Ellipse(cx, cy, f*.1, f*.1, RGB{220, 240, 255}, 0.8)
	}
	sp := p.ToSprite(0, 0)
	iconCache[key] = sp
	return sp
}

const (
	IcoCredits = iota
	IcoEnergy
	IcoPop
	IcoSpeed
	IcoMenu
	IcoBuild
	IcoDemolish
	IcoShip
	IcoQuest
	IcoStore
)

func UIIcon(kind, size int, col RGB) *Sprite {
	key := Tf("u%d_%d_%d", kind, size, col.U())
	if s, ok := iconCache[key]; ok {
		return s
	}
	p := NewPainter(size, size, 4)
	f := float64(size)
	cx, cy := f/2, f/2
	switch kind {
	case IcoCredits:
		p.Ellipse(cx, cy, f*.42, f*.42, hex(0xb8860b), 1)
		p.Ellipse(cx, cy, f*.36, f*.36, hex(0xffd24a), 1)
		p.Poly([]P{{cx - f*.12, cy - f*.2}, {cx + f*.14, cy - f*.2}, {cx + f*.14, cy - f*.1}, {cx - f*.02, cy - f*.1}, {cx - f*.02, cy + f*.1}, {cx + f*.14, cy + f*.1}, {cx + f*.14, cy + f*.2}, {cx - f*.12, cy + f*.2}}, hex(0x8a5a00), 1)
	case IcoEnergy:
		p.Poly([]P{{f * .55, f * .08}, {f * .22, f * .55}, {f * .47, f * .55}, {f * .4, f * .92}, {f * .78, f * .42}, {f * .53, f * .42}}, col, 1)
	case IcoPop:
		p.Ellipse(cx, f*.3, f*.17, f*.17, col, 1)
		p.Poly([]P{{f * .2, f * .9}, {f * .25, f * .6}, {cx, f * .5}, {f * .75, f * .6}, {f * .8, f * .9}}, col, 1)
	case IcoMenu:
		for i := 0; i < 3; i++ {
			y := f * (0.28 + float64(i)*0.22)
			p.LineW(f*.2, y, f*.8, y, f*.1, col, 1)
		}
	case IcoBuild:
		p.LineW(f*.25, f*.8, f*.7, f*.35, f*.14, col, 1)
		p.Poly([]P{{f * .55, f * .15}, {f * .9, f * .4}, {f * .75, f * .55}, {f * .45, f * .3}}, col, 1)
	case IcoDemolish:
		p.LineW(f*.2, f*.2, f*.8, f*.8, f*.14, col, 1)
		p.LineW(f*.8, f*.2, f*.2, f*.8, f*.14, col, 1)
	case IcoShip:
		p.Poly([]P{{f * .1, f * .55}, {f * .9, f * .55}, {f * .75, f * .78}, {f * .22, f * .78}}, col, 1)
		p.Poly([]P{{f * .25, f * .35}, {f * .5, f * .35}, {f * .5, f * .55}, {f * .25, f * .55}}, col.Mul(0.8), 1)
		p.Poly([]P{{f * .55, f * .25}, {f * .75, f * .25}, {f * .75, f * .55}, {f * .55, f * .55}}, col, 1)
	case IcoQuest:
		p.Ellipse(cx, cy, f*.42, f*.42, col, 1)
		p.Poly([]P{{cx - f*.05, f * .22}, {cx + f*.05, f * .22}, {cx + f*.04, f * .6}, {cx - f*.04, f * .6}}, hex(0x0e2236), 1)
		p.Ellipse(cx, f*.74, f*.06, f*.06, hex(0x0e2236), 1)
	case IcoStore:
		p.Poly([]P{{f * .15, f * .4}, {cx, f * .15}, {f * .85, f * .4}, {f * .85, f * .85}, {f * .15, f * .85}}, col, 1)
		p.Poly([]P{{f * .35, f * .55}, {f * .65, f * .55}, {f * .65, f * .85}, {f * .35, f * .85}}, hex(0x0e2236), 1)
	}
	sp := p.ToSprite(0, 0)
	iconCache[key] = sp
	return sp
}
