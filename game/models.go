package game

import "math"

// Farbpalette "Moderne Zukunft"
var (
	mWhite   = hex(0xeef2f5)
	mLight   = hex(0xc9d1d9)
	mGraph   = hex(0x3a4048)
	mDarkG   = hex(0x22272e)
	mTeal    = hex(0x2fd4e0)
	mCyan    = hex(0x5ef2ff)
	mWarm    = hex(0xffa64a)
	mWood    = hex(0xb07a4a)
	mGreen   = hex(0x5f9a3c)
	mGreenD  = hex(0x3f7a30)
	mConc    = hex(0xb9bec4)
	mOrange  = hex(0xff7a2a)
	mRed     = hex(0xff4f6a)
	mPurple  = hex(0xa070ff)
	mBlueAcc = hex(0x3a7bd5)
)

func plinth(b *MB, w, h float64) {
	b.BoxC(0.03, -0.8, 0.03, w-0.03, 0.05, h-0.03, mConc, hex(0xd5d9de), MatMatte, MatMatte)
}

// Dachgarten mit kleinen Bäumen
func roofGarden(b *MB, x0, z0, x1, z1, y float64, seed int) {
	b.Box(x0, y, z0, x1, y+0.04, z1, hex(0x3d6a2c), MatMatte)
	n := 2 + seed%3
	for i := 0; i < n; i++ {
		x := x0 + 0.15 + hash2(i, seed, 1)*(x1-x0-0.3)
		z := z0 + 0.15 + hash2(i, seed, 2)*(z1-z0-0.3)
		r := 0.09 + hash2(i, seed, 3)*0.06
		b.Cyl(x, z, 0.015, y, y+0.12, 5, mWood, MatMatte, false)
		b.Sphere(x, y+0.12+r*0.8, z, r, r*0.9, r, 0, 1, 5, 8, mGreen.Mul(0.9+hash2(i, seed, 4)*0.2), MatLeaf)
	}
}

// Glasturm-Segment mit Paneel-Lisenen
func glassBlock(b *MB, x0, y0, z0, x1, y1, z1 float64, fin RGB) {
	b.BoxC(x0, y0, z0, x1, y1, z1, mWhite, mLight, MatGlass, MatPanel)
	// vertikale Lisenen
	nx := int(math.Max(1, (x1-x0)/0.35))
	for i := 0; i <= nx; i++ {
		x := x0 + (x1-x0)*float64(i)/float64(nx)
		b.Box(x-0.018, y0, z1-0.01, x+0.018, y1, z1+0.025, fin, MatPanel)
		b.Box(x-0.018, y0, z0-0.025, x+0.018, y1, z0+0.01, fin, MatPanel)
	}
	nz := int(math.Max(1, (z1-z0)/0.35))
	for i := 0; i <= nz; i++ {
		z := z0 + (z1-z0)*float64(i)/float64(nz)
		b.Box(x1-0.01, y0, z-0.018, x1+0.025, y1, z+0.018, fin, MatPanel)
		b.Box(x0-0.025, y0, z-0.018, x0+0.01, y1, z+0.018, fin, MatPanel)
	}
}

func emitRing(b *MB, x0, y, z0, x1, z1, h float64, col RGB) {
	b.Box(x0-0.012, y, z0-0.012, x1+0.012, y+h, z1+0.012, col, MatEmit)
}

func BuildModel(t, v, tier int) *MB {
	b := NewMB()
	bt := &BTypes[t]
	W, H := float64(bt.W), float64(bt.H)
	acc := []RGB{mTeal, mWarm, hex(0x9be15d), mBlueAcc}[v%4]
	if Opt.KenneyModels && kenneyBuilding(b, t, v, tier) {
		return b
	}
	switch t {
	case BTurret:
		turretBase(b)
		return b
	case BRaumhafen:
		raumhafenModel(b)
		return b
	case BHe3:
		he3Model(b)
		return b
	case BFeuer:
		feuerwacheModel(b)
		return b
	case BHouse:
		plinth(b, W, H)
		switch tier {
		case 0:
			// moderne Villa: versetzte Kuben
			h1 := 0.55 + float64(v%2)*0.1
			b.BoxC(0.25, 0.05, 0.3, 1.55, h1, 1.7, mWhite, mLight, MatPanel, MatPanel)
			b.Box(0.24, 0.12, 0.85, 1.56, h1-0.08, 1.71, mWhite, MatGlass)
			b.BoxC(0.6, h1, 0.25, 1.8, h1+0.45, 1.3, mWhite, mLight, MatPanel, MatPanel)
			b.Box(0.59, h1+0.06, 0.24, 1.81, h1+0.38, 1.0, mWhite, MatGlass)
			b.Box(1.55, 0.05, 1.3, 1.85, h1, 1.75, mWood, MatMatte)
			emitRing(b, 0.6, h1+0.43, 0.25, 1.8, 1.3, 0.02, acc)
			if v%2 == 0 {
				b.Box(0.75, h1+0.45, 0.35, 1.7, h1+0.47, 1.2, mDarkG, MatSolar)
			} else {
				roofGarden(b, 0.7, 0.35, 1.7, 1.2, h1+0.45, v)
			}
			// Pool / Terrasse
			b.Box(0.3, 0.05, 1.75, 1.2, 0.08, 1.95, hex(0xdfe5ea), MatPanel)
			b.Plate(0.35, 1.78, 1.15, 1.92, 0.081, hex(0x3ac6d8), MatWaterP)
		case 1:
			// Mittelhohes Wohnhaus mit Balkonen
			floors := 4 + v%2
			fh := 0.36
			for f := 0; f < floors; f++ {
				y := 0.05 + float64(f)*fh
				off := 0.0
				if f%2 == 1 {
					off = 0.08
				}
				b.BoxC(0.25+off, y, 0.25, 1.75+off*0.5, y+fh-0.06, 1.75-off, mWhite, mLight, MatGlass, MatPanel)
				b.Box(0.15, y+fh-0.06, 0.15, 1.85, y+fh, 1.85, mWhite, MatPanel)
				// Balkon-Brüstung
				b.Box(0.15, y+fh-0.06, 1.8, 1.85, y+fh+0.08, 1.85, hex(0xa9c6d6), MatGlass)
			}
			top := 0.05 + float64(floors)*fh
			emitRing(b, 0.15, top-0.02, 0.15, 1.85, 1.85, 0.025, acc)
			roofGarden(b, 0.25, 0.25, 1.75, 1.75, top, v+3)
			b.Box(0.3, top, 0.3, 0.7, top+0.25, 0.7, mLight, MatPanel)
		case 2:
			// Hochhaus mit Glasfassade und Krone
			base := 0.55
			b.BoxC(0.1, 0.05, 0.1, 1.9, base, 1.9, mWhite, mLight, MatPanel, MatPanel)
			b.Box(0.09, 0.08, 0.09, 1.91, base-0.08, 1.91, mWhite, MatGlass)
			top := 4.2 + float64(v%3)*0.8
			mid := base + (top-base)*0.55
			glassBlock(b, 0.35, base, 0.35, 1.65, mid, 1.65, mWhite)
			b.Box(0.25, mid, 0.25, 1.75, mid+0.08, 1.75, mWhite, MatPanel)
			roofGarden(b, 0.3, 0.3, 1.7, 1.7, mid+0.08, v+7)
			glassBlock(b, 0.5, mid+0.08, 0.5, 1.5, top, 1.5, mWhite)
			emitRing(b, 0.5, top-0.12, 0.5, 1.5, 1.5, 0.04, acc)
			b.Box(0.45, top, 0.45, 1.55, top+0.1, 1.55, mLight, MatPanel)
			// Dachtechnik
			b.Box(0.55, top+0.1, 0.55, 0.85, top+0.28, 0.8, mGraph, MatMetal)
			b.Box(1.2, top+0.1, 1.15, 1.45, top+0.22, 1.45, mLight, MatMetal)
			b.Cyl(1.3, 0.7, 0.1, top+0.1, top+0.25, 10, mGraph, MatMetal, true)
			if v%2 == 0 {
				b.Cyl(1.0, 1.0, 0.035, top+0.1, top+1.0, 6, mWhite, MatMetal, true)
				b.Sphere(1.0, top+1.02, 1.0, 0.05, 0.05, 0.05, 0, 1, 4, 6, mRed, MatEmit)
			} else {
				b.Sphere(1.0, top+0.1, 1.0, 0.42, 0.35, 0.42, 0.5, 1, 6, 16, hex(0xa9d6e6), MatGlass)
			}
			roofGarden(b, 0.12, 0.12, 0.6, 1.88, base, v)
		}
	case BPark:
		b.BoxC(0.02, -0.6, 0.02, 0.98, 0.04, 0.98, mConc, hex(0x5fa040), MatMatte, MatMatte)
		b.Cyl(0.5, 0.5, 0.2, 0.04, 0.09, 16, mWhite, MatPanel, false)
		b.Plate(0.32, 0.32, 0.68, 0.68, 0.085, hex(0x3ac6d8), MatWaterP)
		b.Cyl(0.5, 0.5, 0.02, 0.04, 0.3, 6, mWhite, MatMetal, true)
		for i, q := range [][2]float64{{0.18, 0.2}, {0.82, 0.25}, {0.2, 0.82}, {0.8, 0.8}} {
			r := 0.1 + float64((v+i)%3)*0.02
			b.Cyl(q[0], q[1], 0.015, 0.04, 0.2, 5, mWood, MatMatte, false)
			b.Sphere(q[0], 0.2+r, q[1], r, r*1.1, r, 0, 1, 5, 8, mGreen, MatLeaf)
		}
	case BKontor:
		plinth(b, W, H)
		// Terminal mit geschwungenem Dach
		b.Box(0.25, 0.05, 0.3, 2.2, 0.7, 2.0, mWhite, MatGlass)
		b.Push(Translate(0.2, 0.7, 1.15))
		b.Push(RotZ(-math.Pi / 2))
		b.Push(ScaleM(0.35, 1, 1))
		b.Cone(0, 0, 0.9, 0.9, 0, 2.05, 24, mWhite, MatPanel, true)
		b.Pop()
		b.Pop()
		b.Pop()
		emitRing(b, 0.25, 0.68, 0.3, 2.2, 2.0, 0.03, mCyan)
		// Kontrollturm
		b.Cyl(2.55, 0.55, 0.22, 0.05, 2.6, 16, mWhite, MatPanel, true)
		b.Cone(2.55, 0.55, 0.22, 0.42, 2.6, 2.9, 16, mWhite, MatPanel, false)
		b.Cyl(2.55, 0.55, 0.42, 2.9, 3.2, 16, mWhite, MatGlass, false)
		b.Cyl(2.55, 0.55, 0.44, 3.2, 3.3, 16, mLight, MatPanel, true)
		b.Cyl(2.55, 0.55, 0.43, 3.0, 3.03, 16, mCyan, MatEmit, false)
		b.Cyl(2.55, 0.55, 0.02, 3.3, 3.9, 4, mWhite, MatMetal, true)
		b.Sphere(2.55, 3.92, 0.55, 0.05, 0.05, 0.05, 0, 1, 4, 6, mRed, MatEmit)
		// Container + Kran
		cc := []RGB{mOrange, mTeal, mWhite, hex(0x9be15d), mBlueAcc}
		for i := 0; i < 6; i++ {
			x := 0.35 + float64(i%3)*0.42
			y := 0.05 + float64(i/3)*0.22
			b.Box(x, y, 2.25, x+0.38, y+0.21, 2.85, cc[(i+v)%5], MatMetal)
		}
		b.Box(1.75, 0.05, 2.3, 1.85, 1.8, 2.4, mWarm, MatMetal)
		b.Box(1.75, 0.05, 2.75, 1.85, 1.8, 2.85, mWarm, MatMetal)
		b.Box(1.5, 1.75, 2.25, 2.95, 1.88, 2.9, mWarm, MatMetal)
	case BLager:
		plinth(b, W, H)
		b.BoxC(0.15, 0.05, 0.15, 1.85, 0.75, 1.35, mLight, mWhite, MatMetal, MatPanel)
		for i := 0; i < 9; i++ {
			x := 0.2 + float64(i)*0.2
			b.Box(x, 0.05, 1.34, x+0.04, 0.75, 1.37, mGraph, MatMetal)
		}
		emitRing(b, 0.15, 0.72, 0.15, 1.85, 1.35, 0.025, mCyan)
		b.Box(0.15, 0.75, 0.15, 1.85, 0.77, 1.35, mDarkG, MatSolar)
		cc := []RGB{mOrange, mTeal, mWhite}
		for i := 0; i < 3; i++ {
			b.Box(0.2+float64(i)*0.55, 0.05, 1.5, 0.65+float64(i)*0.55, 0.3, 1.88, cc[(i+v)%3], MatMetal)
		}
	case BWerft:
		plinth(b, W, H)
		b.Box(0.3, 0.05, 0.3, 2.7, 0.08, 2.7, mGraph, MatMatte)
		b.Push(Translate(1.5, 0.08, 1.5))
		b.Push(RotY(math.Pi / 4))
		b.Prism([][2]float64{{-1.0, -0.35}, {0.85, -0.35}, {1.25, 0}, {0.85, 0.35}, {-1.0, 0.35}}, 0, 0.45, mWhite, MatPanel, mLight, MatPanel)
		b.Box(-0.7, 0.45, -0.25, 0.3, 0.75, 0.25, mOrange, MatMetal)
		b.Pop()
		b.Pop()
		for _, q := range [][2]float64{{0.35, 0.35}, {2.5, 0.35}, {0.35, 2.5}, {2.5, 2.5}} {
			b.Box(q[0], 0.05, q[1], q[0]+0.15, 2.3, q[1]+0.15, mWarm, MatMetal)
		}
		b.Box(0.35, 2.2, 0.35, 2.65, 2.4, 0.5, mWarm, MatMetal)
		b.Box(0.35, 2.2, 2.5, 2.65, 2.4, 2.65, mWarm, MatMetal)
		b.Box(1.4, 2.35, 0.35, 1.6, 2.5, 2.65, mGraph, MatMetal)
		emitRing(b, 0.35, 2.4, 0.35, 2.65, 0.5, 0.02, mCyan)
	case BKomm:
		plinth(b, W, H)
		b.Cyl(1.5, 1.5, 1.3, 0.05, 0.1, 24, hex(0xdfe5ea), MatPanel, true)
		b.Cyl(1.5, 1.5, 0.75, 0.1, 0.9, 20, mWhite, MatGlass, true)
		b.Cyl(1.5, 1.5, 0.78, 0.9, 1.0, 20, mWhite, MatPanel, true)
		b.Cone(1.5, 1.5, 0.35, 0.18, 1.0, 3.8, 16, mWhite, MatPanel, true)
		for i := 0; i < 3; i++ {
			y := 1.6 + float64(i)*0.65
			b.Cyl(1.5, 1.5, 0.55-float64(i)*0.08, y, y+0.06, 20, mLight, MatPanel, true)
			b.Cyl(1.5, 1.5, 0.56-float64(i)*0.08, y+0.06, y+0.09, 20, mCyan, MatEmit, false)
		}
		b.Cyl(1.5, 1.5, 0.03, 3.8, 4.4, 4, mWhite, MatMetal, true)
		b.Sphere(1.5, 4.42, 1.5, 0.06, 0.06, 0.06, 0, 1, 4, 6, mRed, MatEmit)
		// Parabolspiegel
		b.Push(Translate(1.5, 2.9, 1.5))
		b.Push(RotY(float64(v)))
		b.Push(RotX(-0.9))
		b.Sphere(0, 0.25, 0, 0.35, 0.12, 0.35, 0, 0.45, 4, 14, mWhite, MatPanel)
		b.Pop()
		b.Pop()
		b.Pop()
		for i := 0; i < 6; i++ {
			a := float64(i) / 6 * 2 * math.Pi
			x, z := 1.5+math.Cos(a)*1.15, 1.5+math.Sin(a)*1.15
			b.Cyl(x, z, 0.015, 0.1, 0.25, 5, mWood, MatMatte, false)
			b.Sphere(x, 0.37, z, 0.12, 0.13, 0.12, 0, 1, 5, 8, mGreen, MatLeaf)
		}
	case BMedi:
		plinth(b, W, H)
		b.Push(Translate(1.3, 0.05, 1.2))
		for i := 0; i < 3; i++ {
			y := float64(i) * 0.38
			b.Cyl(0, 0, 1.0-float64(i)*0.12, y, y+0.32, 28, mWhite, MatGlass, false)
			b.Cyl(0, 0, 1.04-float64(i)*0.12, y+0.32, y+0.38, 28, mWhite, MatPanel, true)
		}
		b.Pop()
		b.Cyl(1.3, 1.2, 0.3, 1.19, 1.25, 16, mRed, MatEmit, true)
		b.Box(1.25, 1.25, 0.95, 1.35, 1.27, 1.45, mWhite, MatEmit)
		b.Box(1.05, 1.25, 1.15, 1.55, 1.27, 1.25, mWhite, MatEmit)
		// Hubschrauberlandeplatz
		b.Box(2.3, 0.05, 2.2, 2.9, 0.4, 2.9, mLight, MatPanel)
		b.Cyl(2.6, 2.55, 0.26, 0.4, 0.42, 16, mGraph, MatMatte, true)
		b.Cyl(2.6, 2.55, 0.2, 0.42, 0.43, 16, mWarm, MatEmit, false)
	case BLabor:
		plinth(b, W, H)
		b.Cyl(1.3, 1.3, 1.15, 0.05, 0.35, 28, mWhite, MatPanel, true)
		b.Cyl(1.3, 1.3, 1.17, 0.3, 0.33, 28, mPurple, MatEmit, false)
		b.Sphere(1.3, 0.35, 1.3, 1.05, 1.0, 1.05, 0.5, 1, 10, 28, hex(0xcfe0ff), MatGlass)
		b.Box(2.2, 0.05, 0.3, 2.9, 0.9, 2.7, mWhite, MatGlass)
		b.Box(2.18, 0.9, 0.28, 2.92, 0.98, 2.72, mWhite, MatPanel)
		b.Cyl(2.55, 2.4, 0.04, 0.98, 2.2, 6, mWhite, MatMetal, true)
		b.Sphere(2.55, 2.25, 2.4, 0.09, 0.09, 0.09, 0, 1, 5, 8, mPurple, MatEmit)
	case BWind:
		b.BoxC(0.2, -0.6, 0.2, 0.8, 0.06, 0.8, mConc, mLight, MatMatte, MatMatte)
		b.Cone(0.5, 0.5, 0.1, 0.05, 0.06, 3.3, 12, mWhite, MatPanel, true)
		b.Push(Translate(0.5, 3.35, 0.5))
		b.Push(RotY(-math.Pi / 4))
		b.Box(-0.08, -0.07, -0.25, 0.08, 0.07, 0.12, mWhite, MatPanel)
		b.Box(-0.081, -0.02, -0.2, 0.081, 0.0, 0.05, acc, MatEmit)
		b.Pop()
		b.Pop()
	case BSolar:
		b.BoxC(0.02, -0.6, 0.02, 2.98, 0.03, 2.98, mConc, hex(0x8a9a6a), MatMatte, MatMatte)
		for r := 0; r < 4; r++ {
			for c := 0; c < 3; c++ {
				x0 := 0.15 + float64(c)*0.95
				z0 := 0.15 + float64(r)*0.7
				b.Box(x0+0.38, 0.03, z0+0.2, x0+0.46, 0.28, z0+0.28, mLight, MatMetal)
				b.Push(Translate(x0+0.42, 0.33, z0+0.25))
				b.Push(RotX(-0.45))
				b.Box(-0.42, -0.015, -0.26, 0.42, 0.015, 0.26, mDarkG, MatSolar)
				b.Pop()
				b.Pop()
			}
		}
	case BFusion:
		plinth(b, W, H)
		b.Cyl(1.6, 1.6, 1.35, 0.05, 0.5, 32, mWhite, MatPanel, true)
		// Torus aus Segmenten
		for i := 0; i < 24; i++ {
			a := float64(i) / 24 * 2 * math.Pi
			x, z := 1.6+math.Cos(a)*0.95, 1.6+math.Sin(a)*0.95
			b.Sphere(x, 0.85, z, 0.32, 0.32, 0.32, 0, 1, 6, 10, mLight, MatMetal)
		}
		b.Cyl(1.6, 1.6, 1.0, 0.82, 0.88, 32, mOrange, MatEmit, false)
		b.Sphere(1.6, 0.5, 1.6, 0.6, 0.9, 0.6, 0.5, 1, 8, 20, hex(0xdfe6ee), MatPanel)
		b.Cyl(1.6, 1.6, 0.18, 1.3, 1.45, 12, mOrange, MatEmit, true)
		for _, q := range [][2]float64{{3.35, 0.7}, {3.35, 1.8}, {0.7, 3.35}, {1.8, 3.35}} {
			b.Cone(q[0], q[1], 0.42, 0.3, 0.05, 1.2, 20, mWhite, MatPanel, false)
			b.Cone(q[0], q[1], 0.3, 0.36, 1.2, 1.9, 20, mWhite, MatPanel, false)
			b.Cyl(q[0], q[1], 0.33, 1.84, 1.86, 20, mDarkG, MatMatte, true)
		}
		b.Cyl(3.35, 3.35, 0.12, 0.05, 2.9, 10, mWhite, MatMetal, true)
		b.Sphere(3.35, 2.95, 3.35, 0.1, 0.1, 0.1, 0, 1, 5, 8, mOrange, MatEmit)
	case BSchrott:
		b.BoxC(0.03, -0.6, 0.03, 1.97, 0.03, 1.97, mConc, hex(0x7a7468), MatMatte, MatMatte)
		cols := []RGB{hex(0x6a6560), hex(0x8a8f98), hex(0x9b6a4a), hex(0x4f5660)}
		for i := 0; i < 10; i++ {
			x := 0.3 + hash2(i, v, 1)*1.1
			z := 0.3 + hash2(i, v, 2)*1.1
			b.Push(Translate(x, 0.03, z))
			b.Push(RotY(hash2(i, 0, 5) * 3))
			b.Push(RotZ((hash2(i, 0, 6) - 0.5) * 0.6))
			s := 0.12 + hash2(i, 0, 3)*0.15
			b.Box(-s, 0, -s*0.6, s, s*1.1, s*0.6, cols[i%4], MatMetal)
			b.Pop()
			b.Pop()
			b.Pop()
		}
		b.BoxC(1.3, 0.03, 1.3, 1.9, 0.6, 1.9, mWhite, mLight, MatPanel, MatPanel)
		b.Box(1.29, 0.1, 1.6, 1.91, 0.5, 1.91, mWhite, MatGlass)
		b.Box(0.15, 0.03, 1.6, 0.27, 1.5, 1.72, mWarm, MatMetal)
		b.Box(0.15, 1.4, 0.8, 0.27, 1.52, 1.72, mWarm, MatMetal)
	case BRecycle:
		plinth(b, W, H)
		b.BoxC(0.2, 0.05, 0.3, 1.45, 0.85, 1.8, mWhite, mLight, MatPanel, MatPanel)
		b.Box(0.19, 0.15, 0.6, 1.46, 0.7, 1.81, mWhite, MatGlass)
		emitRing(b, 0.2, 0.82, 0.3, 1.45, 1.8, 0.025, hex(0x9be15d))
		b.Cyl(1.7, 0.6, 0.15, 0.05, 1.7, 12, mLight, MatMetal, true)
		b.Cyl(1.7, 0.6, 0.155, 1.4, 1.45, 12, hex(0x9be15d), MatEmit, false)
		b.Box(1.5, 0.05, 1.2, 1.9, 0.35, 1.55, mGraph, MatMetal)
		b.Box(0.3, 0.85, 0.4, 1.35, 0.87, 1.7, mDarkG, MatSolar)
	case BAlgen:
		b.BoxC(0.03, -0.6, 0.03, W-0.03, 0.04, H-0.03, mConc, hex(0xd5d9de), MatMatte, MatMatte)
		for i := 0; i < 6; i++ {
			z := 0.25 + float64(i)*0.2
			b.Push(Translate(0.15, 0.14, z))
			b.Push(RotZ(-math.Pi / 2))
			b.Cyl(0, 0, 0.075, 0, 2.7, 10, hex(0x3fcf8e), MatGlass, false)
			b.Pop()
			b.Pop()
			b.Push(Translate(0.15, 0.14, z))
			b.Push(RotZ(-math.Pi / 2))
			b.Cyl(0, 0, 0.05, 0, 2.7, 8, hex(0x3fcf8e), MatEmit, false)
			b.Pop()
			b.Pop()
		}
		for i := 0; i < 4; i++ {
			x := 0.4 + float64(i)*0.7
			b.Cyl(x, 1.65, 0.17, 0.04, 0.85, 14, mWhite, MatPanel, true)
			b.Cyl(x, 1.65, 0.175, 0.6, 0.63, 14, hex(0x3fcf8e), MatEmit, false)
		}
	case BNahrung:
		plinth(b, W, H)
		b.BoxC(0.2, 0.05, 0.7, 1.75, 0.75, 1.8, mWhite, mLight, MatPanel, MatPanel)
		b.Box(0.19, 0.15, 0.95, 1.76, 0.6, 1.81, mWhite, MatGlass)
		emitRing(b, 0.2, 0.72, 0.7, 1.75, 1.8, 0.025, mWarm)
		for _, x := range []float64{0.5, 1.0, 1.5} {
			b.Cyl(x, 0.38, 0.2, 0.05, 1.05, 14, mWhite, MatPanel, false)
			b.Sphere(x, 1.05, 0.38, 0.2, 0.12, 0.2, 0.5, 1, 4, 14, mLight, MatPanel)
		}
	case BWasser:
		plinth(b, W, H)
		b.Cyl(0.6, 0.6, 0.42, 0.05, 0.75, 20, mWhite, MatPanel, true)
		b.Cyl(0.6, 0.6, 0.425, 0.5, 0.53, 20, hex(0x4fc3ff), MatEmit, false)
		b.Cyl(1.45, 0.55, 0.3, 0.05, 0.6, 16, mWhite, MatPanel, true)
		b.BoxC(0.25, 0.05, 1.2, 1.8, 0.5, 1.8, mWhite, mLight, MatPanel, MatPanel)
		b.Box(0.24, 0.12, 1.4, 1.81, 0.42, 1.81, mWhite, MatGlass)
		b.Plate(0.3, 1.25, 1.75, 1.75, 0.505, hex(0x3ac6d8), MatWaterP)
	case BErz, BSilMine:
		b.BoxC(0.03, -0.6, 0.03, 1.97, 0.03, 1.97, mConc, hex(0x8a8070), MatMatte, MatMatte)
		pile := hex(0xc0703a)
		pm := MatMatte
		if t == BSilMine {
			pile = hex(0x7fe3e0)
			pm = MatGlass
		}
		for i := 0; i < 4; i++ {
			x := 0.35 + hash2(i, 9, 1)*0.55
			z := 1.15 + hash2(i, 9, 2)*0.5
			b.Sphere(x, 0.03, z, 0.25, 0.22, 0.25, 0.5, 1, 4, 8, pile.Mul(0.85+hash2(i, 9, 3)*0.3), pm)
		}
		b.BoxC(1.0, 0.03, 0.25, 1.75, 0.5, 1.0, mWhite, mLight, MatPanel, MatPanel)
		for _, q := range [][2]float64{{1.05, 0.3}, {1.6, 0.3}, {1.05, 0.85}, {1.6, 0.85}} {
			b.Box(q[0], 0.5, q[1], q[0]+0.08, 1.9, q[1]+0.08, mWarm, MatMetal)
		}
		b.Box(1.0, 1.9, 0.25, 1.75, 2.0, 1.0, mWarm, MatMetal)
		b.Cyl(1.37, 0.62, 0.06, 0.03, 2.2, 8, mGraph, MatMetal, true)
		if t == BSilMine {
			b.Cyl(1.37, 0.62, 0.12, 0.5, 0.55, 10, mCyan, MatEmit, false)
		}
	case BStahl:
		plinth(b, W, H)
		b.BoxC(0.2, 0.05, 0.3, 2.15, 1.0, 1.8, hex(0x6a727c), mGraph, MatMetal, MatMetal)
		for i := 0; i < 5; i++ {
			x := 0.25 + float64(i)*0.38
			b.Box(x, 1.0, 0.3, x+0.3, 1.12, 1.8, mLight, MatMetal)
		}
		b.Box(0.5, 0.05, 1.79, 1.8, 0.45, 1.83, mOrange, MatEmit)
		b.Cyl(2.5, 0.6, 0.2, 0.05, 2.2, 12, mLight, MatMetal, true)
		b.Cyl(2.5, 1.3, 0.2, 0.05, 2.0, 12, mLight, MatMetal, true)
		b.Cyl(2.5, 0.6, 0.205, 1.9, 1.95, 12, mOrange, MatEmit, false)
	case BKaffeePl, BFaserPl:
		ground := hex(0x6a5236)
		crop := hex(0x2f6b2a)
		if t == BFaserPl {
			ground = hex(0x8a7a48)
			crop = hex(0xc8d86b)
		}
		b.BoxC(0.03, -0.6, 0.03, 2.97, 0.03, 2.97, mConc, ground, MatMatte, MatMatte)
		for r := 0; r < 6; r++ {
			z := 0.3 + float64(r)*0.45
			for c := 0; c < 6; c++ {
				x := 0.3 + float64(c)*0.42
				if x > 2.0 && z < 1.0 {
					continue
				}
				if t == BKaffeePl {
					b.Sphere(x, 0.03, z, 0.17, 0.2, 0.17, 0.45, 1, 4, 8, crop.Mul(0.9+hash2(r, c, 4)*0.2), MatLeaf)
					b.Sphere(x+0.08, 0.15, z+0.06, 0.03, 0.03, 0.03, 0, 1, 3, 5, hex(0xd0302a), MatMatte)
				} else {
					h := 0.45 + hash2(r, c, 5)*0.2
					b.Cyl(x, z, 0.03, 0.03, h, 5, crop.Mul(0.85+hash2(r, c, 4)*0.25), MatLeaf, false)
					b.Sphere(x, h, z, 0.07, 0.1, 0.07, 0, 1, 4, 6, hex(0xe6e0a0), MatLeaf)
				}
			}
		}
		// Bewässerungsturm
		b.Cyl(2.5, 0.55, 0.22, 0.03, 1.4, 14, mWhite, MatPanel, true)
		b.Sphere(2.5, 1.4, 0.55, 0.3, 0.25, 0.3, 0, 1, 6, 14, mWhite, MatPanel)
		b.Cyl(2.5, 0.55, 0.305, 1.38, 1.42, 14, mCyan, MatEmit, false)
	case BRoest:
		plinth(b, W, H)
		b.BoxC(0.2, 0.05, 0.2, 1.35, 0.9, 1.8, hex(0x8a5a3a), mGraph, MatPanel, MatMetal)
		b.Box(0.19, 0.2, 0.5, 1.36, 0.75, 1.81, mWhite, MatGlass)
		b.Cyl(1.65, 0.7, 0.16, 0.05, 1.5, 12, mLight, MatMetal, true)
		b.Push(Translate(1.55, 0.35, 1.4))
		b.Push(RotX(math.Pi / 2))
		b.Cyl(0, 0, 0.25, -0.3, 0.3, 14, hex(0xb5651d), MatMetal, true)
		b.Pop()
		b.Pop()
	case BTextil:
		plinth(b, W, H)
		b.BoxC(0.2, 0.05, 0.2, 1.8, 0.7, 1.7, mWhite, mLight, MatPanel, MatPanel)
		for i := 0; i < 3; i++ {
			z := 0.2 + float64(i)*0.5
			b.Quad(V3{0.2, 0.7, z}, V3{0.2, 1.0, z + 0.5}, V3{1.8, 1.0, z + 0.5}, V3{1.8, 0.7, z}, hex(0xc061ff), MatGlass)
			b.Quad(V3{0.2, 0.7, z + 0.5}, V3{1.8, 0.7, z + 0.5}, V3{1.8, 1.0, z + 0.5}, V3{0.2, 1.0, z + 0.5}, mWhite, MatPanel)
		}
		b.Box(0.19, 0.15, 0.8, 1.81, 0.55, 1.71, mWhite, MatGlass)
		emitRing(b, 0.2, 0.67, 0.2, 1.8, 1.7, 0.025, hex(0xc061ff))
	case BChip:
		plinth(b, W, H)
		b.BoxC(0.2, 0.05, 0.25, 2.8, 0.9, 1.75, mWhite, mLight, MatPanel, MatPanel)
		b.Box(0.19, 0.3, 0.24, 2.81, 0.36, 1.76, hex(0x46f0a0), MatEmit)
		b.Box(0.19, 0.6, 0.24, 2.81, 0.66, 1.76, hex(0x46f0a0), MatEmit)
		for i := 0; i < 4; i++ {
			x := 0.45 + float64(i)*0.6
			b.Box(x, 0.9, 0.6, x+0.4, 1.15, 1.0, mLight, MatMetal)
			b.Cyl(x+0.2, 0.8, 0.13, 1.15, 1.18, 12, mDarkG, MatMatte, true)
		}
	case BHolo:
		plinth(b, W, H)
		b.BoxC(0.2, 0.05, 0.2, 2.8, 1.0, 1.2, mWhite, mLight, MatGlass, MatPanel)
		b.BoxC(1.8, 0.05, 1.2, 2.8, 0.7, 2.8, mWhite, mLight, MatPanel, MatPanel)
		b.Cyl(0.9, 2.0, 0.55, 0.05, 0.12, 20, mLight, MatPanel, true)
		// Holo-Pyramide
		tp := V3{0.9, 1.9, 2.0}
		ps := []V3{{0.45, 0.15, 1.55}, {1.35, 0.15, 1.55}, {1.35, 0.15, 2.45}, {0.45, 0.15, 2.45}}
		for i := 0; i < 4; i++ {
			b.Tri(ps[i], tp, ps[(i+1)%4], hex(0x5aa8ff), MatEmit)
		}
		emitRing(b, 0.2, 0.97, 0.2, 2.8, 1.2, 0.03, hex(0x5aa8ff))
	}
	addDetails(b, t, v, tier)
	return b
}

// Rotor der Windturbine (um Z drehbar, Nabe im Ursprung)
func RotorModel() *MB {
	b := NewMB()
	b.Sphere(0, 0, 0.02, 0.09, 0.09, 0.12, 0, 1, 5, 8, mWhite, MatPanel)
	for k := 0; k < 3; k++ {
		b.Push(RotZ(float64(k) * 2 * math.Pi / 3))
		b.Quad(V3{-0.07, 0.06, 0.02}, V3{0.08, 0.06, 0.02}, V3{0.025, 1.35, 0.0}, V3{-0.02, 1.35, 0.0}, mWhite, MatPanel)
		b.Quad(V3{0.08, 0.06, 0.0}, V3{-0.07, 0.06, 0.0}, V3{-0.02, 1.35, -0.01}, V3{0.025, 1.35, -0.01}, mLight, MatPanel)
		b.Pop()
	}
	return b
}

// Bäume / Deko (Instanzen)
func TreeModel(kind int) *MB {
	b := NewMB()
	switch kind {
	case 0: // Laubbaum
		b.Cone(0, 0, 0.05, 0.03, 0, 0.35, 6, hex(0x6a4a30), MatMatte, false)
		b.Sphere(0, 0.55, 0, 0.3, 0.28, 0.3, 0, 1, 6, 9, hex(0x3f7f30), MatLeaf)
		b.Sphere(0.12, 0.72, 0.05, 0.18, 0.16, 0.18, 0, 1, 5, 8, hex(0x4f9238), MatLeaf)
	case 1: // Nadelbaum
		b.Cyl(0, 0, 0.04, 0, 0.2, 5, hex(0x5a3e28), MatMatte, false)
		for i := 0; i < 3; i++ {
			y := 0.15 + float64(i)*0.28
			r := 0.32 - float64(i)*0.08
			b.Cone(0, 0, r, 0.0, y, y+0.45, 9, hex(0x2f6a3a).Mul(0.9+float64(i)*0.08), MatLeaf, false)
		}
	case 2: // Zukunftsbaum (schlank, schirmförmig)
		b.Cone(0, 0, 0.04, 0.025, 0, 0.8, 6, hex(0xd8d0c0), MatMatte, false)
		b.Sphere(0, 0.85, 0, 0.36, 0.12, 0.36, 0, 1, 5, 12, hex(0x3f8a50), MatLeaf)
		b.Sphere(0, 0.95, 0, 0.22, 0.08, 0.22, 0, 1, 4, 10, hex(0x5fb060), MatLeaf)
	case 3: // Felsen
		b.Sphere(0, 0, 0, 0.3, 0.22, 0.25, 0.3, 1, 4, 7, hex(0x8a857c), MatMatte)
		b.Sphere(0.22, 0, 0.1, 0.18, 0.15, 0.16, 0.3, 1, 3, 6, hex(0x9a948a), MatMatte)
	case 4: // Leuchtkristalle
		for i := 0; i < 4; i++ {
			a := float64(i) * 1.7
			x, z := math.Cos(a)*0.15, math.Sin(a)*0.15
			b.Cone(x, z, 0.05, 0, 0, 0.3+float64(i%3)*0.12, 4, hex(0x5fe8d0), MatEmit, false)
		}
	case 5: // Erzbrocken
		for i := 0; i < 3; i++ {
			a := float64(i) * 2.1
			b.Sphere(math.Cos(a)*0.2, 0, math.Sin(a)*0.2, 0.16, 0.14, 0.16, 0.3, 1, 3, 6, hex(0xc0703a), MatMetal)
		}
	case 6: // Siliziumkristalle
		for i := 0; i < 5; i++ {
			a := float64(i) * 1.3
			x, z := math.Cos(a)*0.2, math.Sin(a)*0.2
			b.Cone(x, z, 0.07, 0, 0, 0.35+float64(i%3)*0.15, 5, hex(0x7fe3e0), MatGlass, false)
		}
	}
	return b
}

func ShipModel(kind int) *MB {
	if Opt.KenneyModels && kind <= 4 {
		key := "sp_craft_cargoA"
		col := mCyan
		switch {
		case kind == 1:
			key, col = "sp_craft_cargoB", hex(0xffc24a)
		case kind >= 2:
			col = RivalDefs[kind-2].Col
		}
		if KM(key) != nil {
			b := KenneyCentered(key, 2.7, 1, accentTo(col, true))
			// Schwebedüsen
			for _, x := range []float64{-0.9, 0.7} {
				for _, z := range []float64{-0.45, 0.45} {
					b.Cyl(x, z, 0.12, 0.02, 0.06, 10, col, MatEmit, true)
				}
			}
			return b
		}
	}
	b := NewMB()
	hull := mWhite
	acc := mCyan
	if kind == 1 {
		hull = hex(0x2a2e36)
		acc = hex(0xffc24a)
	}
	if kind >= 2 {
		acc = RivalDefs[kind-2].Col
		hull = hex(0x3a3f4a).Mix(acc, 0.15)
	}
	// Rumpf (Länge entlang X)
	b.Prism([][2]float64{{-1.3, -0.42}, {0.9, -0.42}, {1.45, 0}, {0.9, 0.42}, {-1.3, 0.42}}, 0.05, 0.38, hull, MatPanel, mLight, MatPanel)
	b.Prism([][2]float64{{-1.25, -0.44}, {0.92, -0.44}, {1.46, 0}, {0.92, 0.44}, {-1.25, 0.44}}, 0.2, 0.24, acc, MatEmit, acc, MatEmit)
	cc := []RGB{mOrange, mTeal, mWhite, hex(0x9be15d), mBlueAcc}
	if kind == 1 {
		cc = []RGB{hex(0xffc24a), hex(0xc0703a), hex(0x8a2a4a)}
	}
	if kind >= 2 {
		cc = []RGB{acc, acc.Mul(0.6), mLight, mGraph}
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			for k := 0; k < 2; k++ {
				x := 0.55 - float64(i)*0.38
				z := -0.29 + float64(j)*0.3
				y := 0.38 + float64(k)*0.17
				b.Box(x-0.17, y, z, x+0.17, y+0.16, z+0.28, cc[(i+j+k)%len(cc)], MatMetal)
			}
		}
	}
	b.BoxC(-1.2, 0.38, -0.32, -0.85, 0.85, 0.32, mWhite, mLight, MatPanel, MatPanel)
	b.Box(-1.21, 0.68, -0.33, -0.84, 0.8, 0.33, mWhite, MatGlass)
	b.Cyl(-1.02, 0, 0.02, 0.85, 1.2, 4, mWhite, MatMetal, true)
	// Schwebedüsen
	for _, x := range []float64{-0.9, 0.6} {
		for _, z := range []float64{-0.3, 0.3} {
			b.Cyl(x, z, 0.12, 0.0, 0.06, 10, acc, MatEmit, true)
		}
	}
	return b
}

func CartModel() *MB {
	b := NewMB()
	b.BoxC(-0.2, 0.05, -0.1, 0.2, 0.15, 0.1, mWhite, mLight, MatPanel, MatPanel)
	b.Box(-0.18, 0.15, -0.09, 0.08, 0.28, 0.09, mOrange, MatMetal)
	b.Box(0.08, 0.15, -0.09, 0.2, 0.24, 0.09, mWhite, MatGlass)
	b.Box(-0.19, 0.02, -0.08, 0.19, 0.05, 0.08, mCyan, MatEmit)
	return b
}

// ---------------------------------------------------------------- Detail-Elemente

const MatAd = 10   // animierte Leuchtreklame
const MatBird = 11 // Flügelschlag

func railing(b *MB, x0, z0, x1, z1, y, h float64) {
	l := math.Hypot(x1-x0, z1-z0)
	n := int(l / 0.1)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(imax(1, n))
		x, z := x0+(x1-x0)*t, z0+(z1-z0)*t
		b.Box(x-0.006, y, z-0.006, x+0.006, y+h, z+0.006, mLight, MatMetal)
	}
	if x0 == x1 {
		b.Box(x0-0.01, y+h, math.Min(z0, z1), x0+0.01, y+h+0.012, math.Max(z0, z1), mLight, MatMetal)
	} else {
		b.Box(math.Min(x0, x1), y+h, z0-0.01, math.Max(x0, x1), y+h+0.012, z0+0.01, mLight, MatMetal)
	}
}

func acUnit(b *MB, x, y, z float64) {
	b.Box(x-0.07, y, z-0.05, x+0.07, y+0.07, z+0.05, hex(0xb8c0c8), MatMetal)
	b.Cyl(x, z, 0.035, y+0.07, y+0.075, 8, mGraph, MatMatte, true)
}

func antenna(b *MB, x, y, z, h float64) {
	b.Cyl(x, z, 0.01, y, y+h, 4, mLight, MatMetal, true)
	for i := 1; i <= 3; i++ {
		yy := y + h*float64(i)/4
		w := 0.06 * float64(4-i) / 3
		b.Box(x-w, yy, z-0.004, x+w, yy+0.008, z+0.004, mLight, MatMetal)
	}
	b.Sphere(x, y+h, z, 0.015, 0.015, 0.015, 0, 1, 3, 5, mRed, MatEmit)
}

func waterTank(b *MB, x, y, z float64) {
	for _, q := range [][2]float64{{-0.07, -0.07}, {0.07, -0.07}, {-0.07, 0.07}, {0.07, 0.07}} {
		b.Box(x+q[0]-0.01, y, z+q[1]-0.01, x+q[0]+0.01, y+0.12, z+q[1]+0.01, mGraph, MatMetal)
	}
	b.Cyl(x, z, 0.1, y+0.12, y+0.3, 12, hex(0x9aa6b2), MatMetal, false)
	b.Cone(x, z, 0.1, 0.0, y+0.3, y+0.38, 12, hex(0x7a8692), MatMetal, false)
}

func dish(b *MB, x, y, z, r, ang float64) {
	b.Cyl(x, z, 0.012, y, y+0.12, 5, mLight, MatMetal, false)
	b.Push(Translate(x, y+0.14, z))
	b.Push(RotY(ang))
	b.Push(RotX(-0.8))
	b.Sphere(0, r*0.6, 0, r, r*0.35, r, 0, 0.45, 3, 10, mWhite, MatPanel)
	b.Pop()
	b.Pop()
	b.Pop()
}

// Leuchtreklame auf der +Z-Fassade
func billboardZ(b *MB, x0, y0, x1, y1, z float64) {
	b.Box(x0-0.02, y0-0.02, z, x1+0.02, y1+0.02, z+0.02, mGraph, MatMetal)
	b.Quad(V3{x0, y0, z + 0.022}, V3{x1, y0, z + 0.022}, V3{x1, y1, z + 0.022}, V3{x0, y1, z + 0.022}, hex(0x101820), MatAd)
}

// Leuchtreklame auf der +X-Fassade
func billboardX(b *MB, z0, y0, z1, y1, x float64) {
	b.Box(x, y0-0.02, z0-0.02, x+0.02, y1+0.02, z1+0.02, mGraph, MatMetal)
	b.Quad(V3{x + 0.022, y0, z1}, V3{x + 0.022, y0, z0}, V3{x + 0.022, y1, z0}, V3{x + 0.022, y1, z1}, hex(0x101820), MatAd)
}

// Rohr zwischen zwei Punkten (horizontal)
func pipe(b *MB, x0, z0, x1, z1, y, r float64, col RGB) {
	ang := math.Atan2(z1-z0, x1-x0)
	l := math.Hypot(x1-x0, z1-z0)
	b.Push(Translate(x0, y, z0))
	b.Push(RotY(-ang))
	b.Push(RotZ(-math.Pi / 2))
	b.Cyl(0, 0, r, 0, l, 8, col, MatMetal, false)
	b.Pop()
	b.Pop()
	b.Pop()
}

func ladder(b *MB, x, z, y0, y1 float64) {
	b.Box(x-0.03, y0, z-0.005, x-0.022, y1, z+0.005, mLight, MatMetal)
	b.Box(x+0.022, y0, z-0.005, x+0.03, y1, z+0.005, mLight, MatMetal)
	for y := y0 + 0.04; y < y1; y += 0.05 {
		b.Box(x-0.03, y, z-0.004, x+0.03, y+0.006, z+0.004, mLight, MatMetal)
	}
}

func planter(b *MB, x, z, y float64, seed int) {
	b.Box(x-0.08, y, z-0.08, x+0.08, y+0.07, z+0.08, hex(0x8a929c), MatPanel)
	b.Sphere(x, y+0.11, z, 0.07, 0.06, 0.07, 0, 1, 4, 7, mGreen.Mul(0.85+hash2(seed, 1, 3)*0.3), MatLeaf)
}

func hazard(b *MB, x0, y0, z, x1 float64) {
	n := int((x1 - x0) / 0.06)
	for i := 0; i < n; i++ {
		col := hex(0xffc21a)
		if i%2 == 1 {
			col = hex(0x202020)
		}
		xx := x0 + float64(i)*0.06
		b.Box(xx, y0, z, xx+0.06, y0+0.04, z+0.012, col, MatMatte)
	}
}

func mergeAt(b *MB, o *MB, m Mat4) {
	b.Push(m)
	base := uint32(len(b.V) / 10)
	for i := 0; i < len(o.V); i += 10 {
		p := V3{float64(o.V[i]), float64(o.V[i+1]), float64(o.V[i+2])}
		n := V3{float64(o.V[i+3]), float64(o.V[i+4]), float64(o.V[i+5])}
		x, y, z, _ := b.tf.MulV(p, 1)
		nx, ny, nz, _ := b.tf.MulV(n, 0)
		nn := V3{nx, ny, nz}.Norm()
		b.V = append(b.V, float32(x), float32(y), float32(z), float32(nn.X), float32(nn.Y), float32(nn.Z), o.V[i+6], o.V[i+7], o.V[i+8], o.V[i+9])
	}
	for _, i := range o.I {
		b.I = append(b.I, i+base)
	}
	b.Pop()
}

func addDetails(b *MB, t, v, tier int) {
	switch t {
	case BHouse:
		switch tier {
		case 0:
			h1 := 0.55 + float64(v%2)*0.1
			railing(b, 0.25, 1.69, 1.55, 1.69, h1, 0.07)
			railing(b, 0.26, 1.3, 0.26, 1.69, h1, 0.07)
			acUnit(b, 1.45, h1+0.45, 1.15)
			if v%3 == 0 {
				dish(b, 0.8, h1+0.45, 1.15, 0.08, 2.4)
			}
			// Eingang mit Vordach und Lampe
			b.Box(0.6, 0.38, 1.7, 1.0, 0.4, 1.85, mWhite, MatPanel)
			b.Box(0.98, 0.3, 1.71, 1.0, 0.34, 1.73, hex(0xffd8a0), 9)
			planter(b, 0.4, 1.9, 0.05, v)
			planter(b, 1.2, 1.9, 0.05, v+1)
			// geparkter Schwebewagen
			car := HoverCarModel()
			mergeAt(b, car, Translate(1.72, 0.12, 1.55).Mul(RotY(math.Pi/2)).Mul(ScaleM(0.9, 0.9, 0.9)))
		case 1:
			floors := 4 + v%2
			fh := 0.36
			for f := 0; f < floors; f++ {
				y := 0.05 + float64(f)*fh + fh - 0.06
				railing(b, 0.18, 1.86, 1.82, 1.86, y, 0.06)
				railing(b, 1.86, 0.18, 1.86, 1.82, y, 0.06)
				if f%2 == 0 {
					acUnit(b, 1.78, y-0.12, 0.5)
				}
			}
			top := 0.05 + float64(floors)*fh
			waterTank(b, 1.45, top, 0.5)
			acUnit(b, 0.5, top, 1.45)
			acUnit(b, 0.75, top, 1.45)
			antenna(b, 0.45, top+0.25, 0.45, 0.4)
			if v%2 == 0 {
				billboardX(b, 0.5, 0.8, 1.2, 1.5, 1.86)
			}
			planter(b, 0.15, 1.92, 0.05, v)
			planter(b, 1.92, 0.3, 0.05, v+2)
		case 2:
			top := 4.2 + float64(v%3)*0.8
			base := 0.55
			mid := base + (top-base)*0.55
			switch v % 3 {
			case 0:
				billboardZ(b, 0.7, mid+0.4, 1.3, mid+1.1, 1.53)
				billboardX(b, 0.7, base+0.4, 1.3, base+1.1, 1.67)
			case 1:
				// Hubschrauberplattform
				b.Cyl(1.0, 1.0, 0.5, top+0.1, top+0.14, 20, mGraph, MatMatte, true)
				b.Cyl(1.0, 1.0, 0.36, top+0.141, top+0.145, 20, mWarm, MatEmit, false)
				b.Cyl(1.0, 1.0, 0.33, top+0.142, top+0.146, 20, mGraph, MatMatte, true)
				b.Box(0.92, top+0.147, 0.85, 0.95, top+0.15, 1.15, mWhite, MatEmit)
				b.Box(1.05, top+0.147, 0.85, 1.08, top+0.15, 1.15, mWhite, MatEmit)
				b.Box(0.95, top+0.147, 0.985, 1.05, top+0.15, 1.015, mWhite, MatEmit)
			default:
				antenna(b, 0.6, top+0.1, 0.6, 0.7)
				antenna(b, 1.4, top+0.1, 1.35, 0.5)
				dish(b, 1.35, top+0.1, 0.65, 0.12, 0.8)
			}
			// Fassaden-Gondel
			b.Box(0.8, mid-0.6, 1.66, 1.2, mid-0.52, 1.74, mLight, MatMetal)
			b.Box(0.82, mid-0.52, 1.66, 0.83, top, 1.67, mLight, MatMetal)
			// Strahler am Sockel
			for _, x := range []float64{0.3, 1.0, 1.7} {
				b.Box(x-0.03, 0.05, 1.92, x+0.03, 0.09, 1.96, hex(0xfff0d0), 9)
			}
			railing(b, 0.25, 1.76, 1.75, 1.76, mid+0.08, 0.07)
			railing(b, 1.76, 0.25, 1.76, 1.75, mid+0.08, 0.07)
		}
	case BKontor:
		dish(b, 2.55, 3.3, 0.55, 0.1, 1.2)
		for i := 0; i < 3; i++ {
			x := 0.5 + float64(i)*0.55
			b.Cyl(x, 2.05, 0.012, 0.05, 0.9, 4, mLight, MatMetal, true)
			b.Box(x, 0.68, 2.05, x+0.2, 0.86, 2.06, []RGB{mTeal, mWhite, mWarm}[i], MatPanel)
		}
		hazard(b, 0.3, 0.05, 2.01, 2.2)
	case BLager:
		ladder(b, 1.8, 0.6, 0.05, 0.75)
		acUnit(b, 0.5, 0.77, 0.5)
		hazard(b, 0.2, 0.05, 1.36, 1.8)
	case BStahl:
		ladder(b, 2.72, 0.6, 0.05, 2.1)
		pipe(b, 2.15, 0.6, 2.3, 0.6, 0.6, 0.06, mLight)
		pipe(b, 2.15, 1.3, 2.3, 1.3, 0.5, 0.06, mLight)
		for i := 0; i < 4; i++ {
			acUnit(b, 0.5+float64(i)*0.45, 1.12, 1.0)
		}
		hazard(b, 0.5, 0.45, 1.83, 1.8)
	case BRecycle:
		ladder(b, 1.87, 0.6, 0.05, 1.7)
		pipe(b, 1.45, 0.6, 1.55, 0.6, 0.5, 0.05, mLight)
		acUnit(b, 0.5, 0.87, 0.6)
	case BNahrung:
		pipe(b, 0.5, 0.38, 1.5, 0.38, 0.85, 0.035, mLight)
		pipe(b, 1.0, 0.58, 1.0, 0.7, 0.5, 0.04, mLight)
		ladder(b, 0.5, 0.6, 0.05, 1.0)
		acUnit(b, 1.4, 0.75, 1.3)
		hazard(b, 0.3, 0.05, 1.81, 1.0)
	case BWasser:
		pipe(b, 0.6, 0.6, 1.45, 0.55, 0.35, 0.05, mBlueAcc)
		pipe(b, 1.0, 0.6, 1.0, 1.2, 0.25, 0.05, mBlueAcc)
		ladder(b, 0.6, 1.04, 0.05, 0.75)
	case BChip:
		for i := 0; i < 5; i++ {
			acUnit(b, 0.4+float64(i)*0.5, 0.9, 1.45)
		}
		pipe(b, 0.3, 1.8, 2.7, 1.8, 0.75, 0.04, mLight)
	case BTextil, BRoest, BHolo:
		acUnit(b, 0.5, 0.75, 0.5)
		acUnit(b, 0.8, 0.75, 0.5)
		hazard(b, 0.4, 0.05, 1.71, 1.2)
	case BMedi:
		amb := HoverCarModel()
		for i := 0; i < len(amb.V); i += 10 {
			if amb.V[i+6] > 0.8 && amb.V[i+7] > 0.8 && amb.V[i+9] == MatPanel {
				amb.V[i+7], amb.V[i+8] = 0.5, 0.5
			}
		}
		mergeAt(b, amb, Translate(2.3, 0.12, 0.4).Mul(ScaleM(1.2, 1.2, 1.2)))
		dish(b, 1.3, 1.25, 1.6, 0.08, 2)
	case BLabor:
		dish(b, 2.4, 0.98, 0.7, 0.14, 0.5)
		dish(b, 2.4, 0.98, 1.4, 0.1, 2.1)
	case BFusion:
		pipe(b, 2.9, 1.6, 3.35, 1.6, 0.4, 0.08, mLight)
		pipe(b, 1.6, 2.9, 1.6, 3.35, 0.4, 0.08, mLight)
		pipe(b, 3.35, 0.7, 3.35, 1.8, 0.25, 0.06, mLight)
		pipe(b, 0.7, 3.35, 1.8, 3.35, 0.25, 0.06, mLight)
		ladder(b, 3.47, 3.35, 0.05, 2.8)
	case BKomm:
		for i := 0; i < 4; i++ {
			a := float64(i)*math.Pi/2 + 0.4
			x, z := 1.5+math.Cos(a)*0.95, 1.5+math.Sin(a)*0.95
			b.Box(x-0.05, 0.1, z-0.02, x+0.05, 0.14, z+0.02, mWood, MatMatte)
		}
	}
}

func GrassModel() *MB {
	b := NewMB()
	g := hex(0x4f8a34)
	for i := 0; i < 6; i++ {
		a := float64(i) * 1.05
		dx, dz := math.Cos(a)*0.03, math.Sin(a)*0.03
		tx, tz := math.Cos(a)*0.06, math.Sin(a)*0.06
		h := 0.07 + float64(i%3)*0.025
		b.Tri(V3{dx - 0.008, 0, dz}, V3{dx + 0.008, 0, dz}, V3{tx, h, tz}, g.Mul(0.85+float64(i%3)*0.1), MatLeaf)
	}
	return b
}

func FlowerModel() *MB {
	b := NewMB()
	b.Cyl(0, 0, 0.004, 0, 0.07, 3, hex(0x6a9a40), MatLeaf, false)
	b.Sphere(0, 0.075, 0, 0.018, 0.012, 0.018, 0, 1, 3, 6, hex(0xffffff), MatMatte)
	return b
}

func PebbleModel() *MB {
	b := NewMB()
	b.Sphere(0, 0, 0, 0.06, 0.035, 0.05, 0.4, 1, 3, 6, hex(0xa8a196), MatMatte)
	b.Sphere(0.07, 0, 0.03, 0.035, 0.025, 0.03, 0.4, 1, 3, 5, hex(0x8f897f), MatMatte)
	return b
}

// Fahnenmast mit leuchtendem Banner in Fraktionsfarbe
func FlagModel(col RGB) *MB {
	b := NewMB()
	b.Cyl(0, 0, 0.035, 0, 2.2, 6, mLight, MatMetal, true)
	b.Box(0.03, 1.55, -0.01, 0.62, 2.12, 0.01, col, MatEmit)
	b.Box(0.03, 1.5, -0.012, 0.62, 1.55, 0.012, mWhite, MatPanel)
	b.Sphere(0, 2.24, 0, 0.05, 0.05, 0.05, 0, 1, 4, 6, col, MatEmit)
	return b
}

// Baugerüst (Höhe 1, wird im Renderer skaliert)
func ScaffoldModel(w, h int) *MB {
	b := NewMB()
	pole := hex(0xb8bec6)
	rail := hex(0xff9a2a)
	W, H := float64(w), float64(h)
	for x := 0.0; x <= W+0.001; x += 0.5 {
		for _, z := range []float64{0, H} {
			b.Box(x-0.012, 0, z-0.012, x+0.012, 1, z+0.012, pole, MatMetal)
		}
	}
	for z := 0.5; z < H-0.001; z += 0.5 {
		for _, x := range []float64{0, W} {
			b.Box(x-0.012, 0, z-0.012, x+0.012, 1, z+0.012, pole, MatMetal)
		}
	}
	for y := 0.25; y <= 1.001; y += 0.25 {
		b.Box(0, y-0.01, -0.02, W, y+0.01, 0.02, rail, MatPanel)
		b.Box(0, y-0.01, H-0.02, W, y+0.01, H+0.02, rail, MatPanel)
		b.Box(-0.02, y-0.01, 0, 0.02, y+0.01, H, rail, MatPanel)
		b.Box(W-0.02, y-0.01, 0, W+0.02, y+0.01, H, rail, MatPanel)
	}
	// Netzplanen
	net := hex(0x2a8a7a)
	b.Box(0.02, 0.5, -0.03, W*0.45, 0.95, -0.02, net, MatGlass)
	b.Box(W*0.55, 0.05, H+0.02, W-0.02, 0.45, H+0.03, net, MatGlass)
	return b
}

// Baukran: Turm (Höhe 1, skaliert) und drehbarer Ausleger
func CraneModel() *MB {
	b := NewMB()
	y := hex(0xffc22a)
	b.Box(-0.18, 0, -0.18, 0.18, 0.02, 0.18, mGraph, MatMetal)
	for _, c := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		b.Box(c[0]*0.07-0.012, 0, c[1]*0.07-0.012, c[0]*0.07+0.012, 1, c[1]*0.07+0.012, y, MatMetal)
	}
	for h := 0.1; h < 1; h += 0.1 {
		b.Box(-0.08, h, -0.08, 0.08, h+0.008, 0.08, y, MatMetal)
	}
	return b
}

func CraneJibModel() *MB {
	b := NewMB()
	y := hex(0xffc22a)
	b.Box(-0.5, 0, -0.035, 1.9, 0.07, 0.035, y, MatMetal)
	b.Box(-0.09, 0.07, -0.09, 0.09, 0.22, 0.09, hex(0xe8ecf0), MatPanel)
	b.Box(-0.08, 0.12, -0.091, 0.08, 0.19, -0.089, mCyan, MatGlass)
	b.Box(-0.55, -0.12, -0.08, -0.3, 0.0, 0.08, mConc, MatMatte)
	b.Cyl(0, 0, 0.01, 0.22, 0.42, 4, y, MatMetal, false)
	// Seil + Last
	b.Box(1.5, -0.9, -0.004, 1.508, 0, 0.004, mGraph, MatMetal)
	b.Box(1.38, -1.05, -0.1, 1.62, -0.9, 0.1, hex(0x6a7480), MatPanel)
	b.Box(1.62, 0.07, -0.01, 1.66, 0.1, 0.01, hex(0xff3030), MatEmit)
	return b
}

// Korvette: schlanker Rumpf, Geschützturm, Brücke
func WarshipModel(acc RGB, hull RGB) *MB {
	b := NewMB()
	b.Prism([][2]float64{{-1.2, -0.34}, {0.7, -0.36}, {1.55, 0}, {0.7, 0.36}, {-1.2, 0.34}}, 0.05, 0.34, hull, MatPanel, hull.Mul(1.15), MatPanel)
	b.Prism([][2]float64{{-1.15, -0.36}, {0.72, -0.38}, {1.56, 0}, {0.72, 0.38}, {-1.15, 0.36}}, 0.18, 0.21, acc, MatEmit, acc, MatEmit)
	// Aufbauten
	b.Prism([][2]float64{{-0.9, -0.22}, {0.1, -0.24}, {0.4, 0}, {0.1, 0.24}, {-0.9, 0.22}}, 0.34, 0.6, hull.Mul(1.2), MatPanel, mLight, MatPanel)
	b.Box(-0.15, 0.48, -0.245, 0.25, 0.56, 0.245, hex(0x203040), MatGlass)
	b.Box(-0.7, 0.6, -0.14, -0.2, 0.82, 0.14, hull.Mul(1.25), MatPanel)
	b.Box(-0.21, 0.66, -0.145, -0.19, 0.76, 0.145, acc, MatEmit)
	b.Cyl(-0.45, 0, 0.02, 0.82, 1.3, 4, mLight, MatMetal, true)
	b.Box(-0.6, 1.12, -0.012, -0.3, 1.14, 0.012, mLight, MatMetal)
	b.Sphere(-0.45, 1.32, 0, 0.04, 0.04, 0.04, 0, 1, 4, 6, hex(0xff3030), MatEmit)
	// Geschütze vorne und hinten
	for _, gx := range []float64{0.75, -1.0} {
		b.Cyl(gx, 0, 0.14, 0.34, 0.46, 10, hull.Mul(1.1), MatMetal, true)
		dir := 1.0
		if gx < 0 {
			dir = -1
		}
		b.Box(gx, 0.39, -0.05, gx+0.45*dir, 0.43, -0.02, mGraph, MatMetal)
		b.Box(gx, 0.39, 0.02, gx+0.45*dir, 0.43, 0.05, mGraph, MatMetal)
		b.Box(gx+0.43*dir, 0.39, -0.05, gx+0.47*dir, 0.43, 0.05, acc, MatEmit)
	}
	for _, x := range []float64{-0.8, 0.5} {
		for _, z := range []float64{-0.25, 0.25} {
			b.Cyl(x, z, 0.1, 0.0, 0.05, 10, acc, MatEmit, true)
		}
	}
	return b
}

// Abwehrturm: Sockel (Gebäudemodell) + drehbarer Kopf
func turretBase(b *MB) {
	plinth(b, 1, 1)
	b.Cone(0.5, 0.5, 0.36, 0.24, 0.06, 0.8, 8, hex(0xc4ccd6), MatPanel, true)
	b.Cyl(0.5, 0.5, 0.25, 0.8, 0.86, 8, mGraph, MatMetal, true)
	emitRing(b, 0.27, 0.5, 0.27, 0.73, 0.73, 0.03, hex(0xff4040))
	for _, c := range [][2]float64{{0.12, 0.12}, {0.88, 0.12}, {0.12, 0.88}, {0.88, 0.88}} {
		b.Box(c[0]-0.03, 0.06, c[1]-0.03, c[0]+0.03, 0.3, c[1]+0.03, hex(0xffc22a), MatPanel)
	}
}

func TurretHeadModel() *MB {
	b := NewMB()
	b.Sphere(0, 0.06, 0, 0.2, 0.15, 0.2, 0, 1, 5, 10, hex(0xe8ecf0), MatPanel)
	b.Box(0.05, 0.04, -0.06, 0.5, 0.09, -0.025, mGraph, MatMetal)
	b.Box(0.05, 0.04, 0.025, 0.5, 0.09, 0.06, mGraph, MatMetal)
	b.Box(0.48, 0.035, -0.065, 0.53, 0.095, 0.065, hex(0x60f0ff), MatEmit)
	b.Box(-0.12, 0.12, -0.12, 0.05, 0.2, 0.12, hex(0x203040), MatGlass)
	return b
}

func feuerwacheModel(b *MB) {
	plinth(b, 2, 2)
	red := hex(0xe8402a)
	// Halle mit Toren
	b.BoxC(0.15, 0.06, 0.2, 1.4, 0.75, 1.85, mWhite, mLight, MatPanel, MatPanel)
	for i := 0; i < 3; i++ {
		z := 0.35 + float64(i)*0.5
		b.Box(1.401, 0.08, z, 1.41, 0.55, z+0.38, red, MatPanel)
		b.Box(1.402, 0.5, z, 1.412, 0.53, z+0.38, mCyan, MatEmit)
	}
	b.Box(0.14, 0.62, 0.19, 1.41, 0.67, 1.86, red, MatEmit)
	// Turm
	b.BoxC(0.25, 0.75, 0.3, 0.7, 1.7, 0.75, mWhite, red, MatPanel, MatPanel)
	b.Box(0.24, 1.4, 0.29, 0.71, 1.5, 0.76, hex(0x203040), MatGlass)
	b.Sphere(0.475, 1.76, 0.525, 0.07, 0.07, 0.07, 0, 1, 4, 6, hex(0x3080ff), MatEmit)
	// Drohnenlandeplatz
	b.Cyl(1.0, 1.2, 0.28, 0.75, 0.78, 12, mGraph, MatPanel, true)
	b.Cyl(1.0, 1.2, 0.2, 0.781, 0.785, 12, hex(0xffc22a), MatEmit, true)
	// Löschdrohne
	b.Box(0.92, 0.8, 1.12, 1.08, 0.86, 1.28, red, MatPanel)
	for _, c := range [][2]float64{{0.86, 1.06}, {1.14, 1.06}, {0.86, 1.34}, {1.14, 1.34}} {
		b.Cyl(c[0], c[1], 0.05, 0.86, 0.87, 8, mLight, MatMetal, true)
	}
	// Fahrzeug vorm Tor
	b.BoxC(1.5, 0.06, 0.4, 1.9, 0.22, 0.75, red, red, MatPanel, MatPanel)
	b.Box(1.75, 0.22, 0.42, 1.88, 0.3, 0.73, hex(0x203040), MatGlass)
	b.Box(1.6, 0.22, 0.5, 1.68, 0.25, 0.65, hex(0x3080ff), MatEmit)
}

// Raumhafen: Startplatz, Turm, Hangar, Tanks (Rakete separat)
func raumhafenModel(b *MB) {
	plinth(b, 4, 4)
	// Startplatz
	b.Cyl(2.6, 1.4, 1.15, 0.06, 0.2, 20, hex(0x6a727c), MatPanel, true)
	b.Cyl(2.6, 1.4, 0.95, 0.2, 0.22, 20, mGraph, MatPanel, true)
	for k := 0; k < 12; k++ {
		a := float64(k) * math.Pi / 6
		b.Cyl(2.6+math.Cos(a)*1.05, 1.4+math.Sin(a)*1.05, 0.04, 0.2, 0.24, 6, hex(0xffc22a), MatEmit, true)
	}
	// Startturm
	for _, c := range [][2]float64{{3.55, 0.75}, {3.85, 0.75}, {3.55, 1.05}, {3.85, 1.05}} {
		b.Box(c[0]-0.025, 0.2, c[1]-0.025, c[0]+0.025, 3.6, c[1]+0.025, hex(0xd84a2a), MatMetal)
	}
	for h := 0.4; h < 3.6; h += 0.4 {
		b.Box(3.52, h, 0.72, 3.88, h+0.03, 1.08, hex(0xd84a2a), MatMetal)
	}
	b.Box(3.0, 2.6, 0.86, 3.55, 2.66, 0.94, mLight, MatMetal)
	b.Box(3.0, 1.5, 0.86, 3.55, 1.56, 0.94, mLight, MatMetal)
	b.Sphere(3.7, 3.7, 0.9, 0.06, 0.06, 0.06, 0, 1, 4, 6, hex(0xff3030), MatEmit)
	// Hangar und Kontrollzentrum
	b.BoxC(0.2, 0.06, 2.6, 1.9, 0.9, 3.8, mWhite, mLight, MatPanel, MatPanel)
	b.Box(0.19, 0.5, 2.7, 0.2, 0.8, 3.7, hex(0x203040), MatGlass)
	b.Box(0.2, 0.88, 2.6, 1.9, 0.92, 3.8, mCyan, MatEmit)
	glassBlock(b, 0.3, 0.92, 2.8, 1.2, 1.6, 3.6, mLight)
	b.Cyl(0.75, 3.2, 0.25, 1.6, 1.68, 12, mGraph, MatMetal, true)
	b.Cyl(0.75, 3.2, 0.012, 1.68, 2.3, 4, mLight, MatMetal, false)
	b.Sphere(0.75, 2.3, 3.2, 0.18, 0.06, 0.18, 0, 1, 4, 10, mWhite, MatPanel)
	// Treibstofftanks
	for i, z := range []float64{0.5, 1.2} {
		b.Sphere(0.65, 0.55, z, 0.32, 0.32, 0.32, 0, 1, 6, 12, []RGB{mWhite, hex(0xe8f4ff)}[i], MatMetal)
		b.Box(0.33, 0.06, z-0.05, 0.97, 0.3, z+0.05, mGraph, MatMetal)
	}
	b.Box(0.95, 0.3, 0.82, 1.7, 0.35, 0.88, mLight, MatMetal)
	hazard(b, 2.2, 0.06, 3.9, 3.8)
}

func RocketModel() *MB {
	b := NewMB()
	w := hex(0xf2f4f6)
	b.Cyl(0, 0, 0.32, 0.25, 2.4, 14, w, MatPanel, false)
	b.Cone(0, 0, 0.32, 0.0, 2.4, 3.3, 14, w, MatPanel, false)
	b.Cyl(0, 0, 0.325, 1.6, 1.75, 14, hex(0x203040), MatGlass, false)
	b.Cyl(0, 0, 0.33, 0.9, 0.98, 14, mCyan, MatEmit, false)
	b.Cone(0, 0, 0.26, 0.34, 0.0, 0.25, 12, mGraph, MatMetal, true)
	for k := 0; k < 4; k++ {
		a := float64(k) * math.Pi / 2
		cx, cz := math.Cos(a), math.Sin(a)
		b.Push(Translate(cx*0.32, 0, cz*0.32).Mul(RotY(-a)))
		b.Box(0, 0.05, -0.015, 0.32, 0.75, 0.015, hex(0xd84a2a), MatPanel)
		b.Pop()
	}
	return b
}

func he3Model(b *MB) {
	plinth(b, 2, 2)
	// Förderkuppel
	b.Sphere(0.75, 0.06, 0.8, 0.6, 0.55, 0.6, 0, 0.5, 6, 14, hex(0xdfe5ec), MatPanel)
	b.Sphere(0.75, 0.07, 0.8, 0.61, 0.56, 0.61, 0.25, 0.32, 2, 14, mCyan, MatEmit)
	// Bohrturm
	b.Box(1.35, 0.06, 0.25, 1.75, 1.5, 0.65, mGraph, MatMetal)
	b.Box(1.33, 1.3, 0.23, 1.77, 1.36, 0.67, hex(0xffc22a), MatPanel)
	b.Cyl(1.55, 0.45, 0.05, 1.5, 1.9, 6, mLight, MatMetal, true)
	// Helium-Tanks (leuchtend)
	for i, x := range []float64{0.4, 0.95} {
		b.Cyl(x, 1.6, 0.18, 0.06, 0.75, 12, hex(0xe8f4ff), MatMetal, true)
		b.Cyl(x, 1.6, 0.185, 0.35+float64(i)*0.08, 0.42+float64(i)*0.08, 12, hex(0x7ab8ff), MatEmit, false)
	}
	b.Box(1.3, 0.06, 1.3, 1.9, 0.5, 1.9, mWhite, MatPanel)
	b.Box(1.29, 0.32, 1.35, 1.3, 0.45, 1.85, hex(0x203040), MatGlass)
}

// ---------------------------------------------------------------- Gebäude aus den Kenney-Sets

func kenneyBuilding(b *MB, t, v, tier int) bool {
	bt := &BTypes[t]
	W, H := float64(bt.W), float64(bt.H)
	acc := []RGB{mTeal, mCyan, hex(0x9be15d), mBlueAcc}[v%4]
	var key string
	maxH := 0.0
	switch t {
	case BHouse:
		switch tier {
		case 0:
			key = []string{"com_a", "com_b", "com_c", "com_d"}[v%4]
			maxH = 1.6
		case 1:
			key = []string{"com_f", "com_g", "com_i", "com_l"}[v%4]
			maxH = 2.8
		default:
			key = []string{"sky_a", "sky_b", "sky_c", "sky_e"}[v%4]
			maxH = 5.2
		}
	case BStahl:
		key = "ind_e"
	case BRecycle:
		key = "ind_c"
	case BChip:
		key = []string{"ind_t", "ind_q"}[v%2]
	case BTextil:
		key = "ind_k"
	case BRoest:
		key = "ind_m"
	case BNahrung:
		key = "ind_i"
	case BSchrott:
		key = "ind_j"
	case BSolar:
		key = "ind_solar-panel-landscape-group"
	default:
		return false
	}
	if KM(key) == nil {
		return false
	}
	plinth(b, W, H)
	rot := v % 4
	if t != BHouse {
		rot = (v % 2) * 2
	}
	mrg := 0.14
	if t == BSolar {
		// Solarpark: mehrere Feldgruppen
		for _, c := range [][2]float64{{0, 0}, {1.5, 0}, {0, 1.5}, {1.5, 1.5}} {
			b.Kenney(key, c[0]+0.12, c[1]+0.12, 1.26, 1.26, 0.05, 0, 0, nil)
		}
		b.Box(1.42, 0.05, 0.1, 1.58, 0.35, 2.9, mGraph, MatMetal)
		b.Box(1.44, 0.3, 0.1, 1.56, 0.33, 2.9, mCyan, MatEmit)
		return true
	}
	h := b.Kenney(key, mrg, mrg, W-2*mrg, H-2*mrg, 0.05, rot, maxH, accentTo(acc, t == BHouse && tier == 2))
	// futuristische Akzente
	b.Box(0.06, 0.05, 0.06, W-0.06, 0.09, 0.1, acc, MatEmit)
	b.Box(0.06, 0.05, H-0.1, W-0.06, 0.09, H-0.06, acc, MatEmit)
	if t == BHouse {
		if tier >= 1 {
			b.Cyl(W*0.5+0.25, H*0.5-0.2, 0.012, h, h+0.5+0.3*float64(tier), 4, mLight, MatMetal, false)
			b.Sphere(W*0.5+0.25, h+0.52+0.3*float64(tier), H*0.5-0.2, 0.03, 0.03, 0.03, 0, 1, 3, 5, hex(0xff3030), MatEmit)
		}
		if tier == 2 && v%3 == 0 {
			b.Box(W*0.5-0.35, h*0.55, 0.12, W*0.5+0.35, h*0.55+0.45, 0.14, acc, MatAd)
		}
		// Bäumchen vor dem Haus
		if tier < 2 && KM("sub_tree-small") != nil {
			b.Kenney("sub_tree-small", W-0.42, H-0.42, 0.34, 0.34, 0.05, v%4, 0, nil)
		}
	} else {
		// Fabriken: kleine Containerreihe und Leuchtschild
		if v%2 == 0 && KM("ind_shipping-container-b") != nil {
			b.Kenney("ind_shipping-container-b", 0.08, H-0.36, 0.5, 0.26, 0.05, 1, 0, accentTo(acc, false))
		}
	}
	return true
}
