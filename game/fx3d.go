package game

import (
	"math"
	"math/rand"
	"sort"
)

// ---------------------------------------------------------------- Partikel (Rauch, Dampf)

type particle struct {
	x, y, z    float64
	vx, vy, vz float64
	age, life  float64
	size, grow float64
	white      float64
	rot, spin  float64
	shade      float64
}

type hovercar struct {
	x, y, z  float64
	ax, az   float64
	bx, bz   float64
	h        float64
	t, speed float64
	tint     RGB
}

type FX struct {
	parts   []particle
	acc     map[int]float64
	cars    []hovercar
	rng     *rand.Rand
	quad    *Instanced
	carInst *Instanced
	flashes []flash
	beams   []beam
	glowMB  *MB
	glowM   *GMesh
}

type flash struct {
	x, y, z, t, life, size float64
	col                    RGB
}

type beam struct {
	x1, y1, z1, x2, y2, z2 float64
	t, life                float64
	col                    RGB
}

func newFX() *FX {
	f := &FX{acc: map[int]float64{}, rng: rand.New(rand.NewSource(5))}
	q := NewMB()
	q.V = []float32{-1, -1, 0, 0, 0, 1, 1, 1, 1, 0, 1, -1, 0, 0, 0, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 1, 1, 1, 1, 0, -1, 1, 0, 0, 0, 1, 1, 1, 1, 0}
	q.I = []uint32{0, 1, 2, 0, 2, 3}
	f.quad = NewInstanced(q.Upload())
	f.carInst = NewInstanced(HoverCarModel().Upload())
	return f
}

// Schornsteine je Gebäudetyp: (x, Höhe, z, weiß?)
var chimneys = map[int][][4]float64{
	BStahl:   {{2.5, 2.2, 0.6, 0}, {2.5, 2.0, 1.3, 0}},
	BRecycle: {{1.7, 1.7, 0.6, 0.3}},
	BFusion:  {{3.35, 1.9, 0.7, 1}, {3.35, 1.9, 1.8, 1}, {0.7, 1.9, 3.35, 1}, {1.8, 1.9, 3.35, 1}},
	BRoest:   {{1.65, 1.5, 0.7, 0.7}},
	BChip:    {{0.65, 1.2, 0.8, 1}},
}

func (f *FX) update(g *Game, dt float64, u *UIState) {
	if dt > 0.1 {
		dt = 0.1
	}
	wind := V3{0.25, 0, 0.12}.Mul(1 + g.Storm*2.5)
	// Ereignisse aus dem Spiel
	for _, e := range g.Events {
		f.event(e)
	}
	g.Events = g.Events[:0]
	fl := f.flashes[:0]
	for _, x := range f.flashes {
		x.t += dt
		if x.t < x.life {
			fl = append(fl, x)
		}
	}
	f.flashes = fl
	bm := f.beams[:0]
	for _, x := range f.beams {
		x.t += dt
		if x.t < x.life {
			bm = append(bm, x)
		}
	}
	f.beams = bm
	// Raketenstart: Flammen und Rauch
	if g.Other != nil && g.Launch < 9 && !g.OnMoon() {
		for _, b := range g.B {
			if b.T != BRaumhafen {
				continue
			}
			lt := g.Launch
			x, z := float64(b.X)+2.6, float64(b.Y)+1.4
			y := b.BaseY + 0.22 + lt*lt*0.9
			f.flashes = append(f.flashes, flash{x: x, y: y - 0.1, z: z, life: 0.12, size: 0.9, col: hex(0xffa040)})
			for k := 0; k < 4 && len(f.parts) < 3900 && Opt.Particles; k++ {
				f.parts = append(f.parts, particle{
					x: x + (f.rng.Float64()-0.5)*0.2, y: y - 0.1, z: z + (f.rng.Float64()-0.5)*0.2,
					vx: (f.rng.Float64() - 0.5) * 0.8, vy: -0.6 - f.rng.Float64()*0.6, vz: (f.rng.Float64() - 0.5) * 0.8,
					life: 2.5 + f.rng.Float64()*1.5, size: 0.2, grow: 0.5, white: 0.9, rot: f.rng.Float64() * 6, spin: 0.4, shade: 1.0,
				})
			}
		}
	}
	// Baustaub
	if Opt.Particles {
		for _, b := range g.allBuildings() {
			if b.Constr <= 0 {
				continue
			}
			cx, cz := b.Center()
			if math.Hypot(cx-u.TX, cz-u.TZ) > u.Dist*1.2+12 {
				continue
			}
			bt := &BTypes[b.T]
			key := -b.ID - 1
			f.acc[key] += dt * 3 * float64(bt.W)
			for f.acc[key] >= 1 && len(f.parts) < 3500 {
				f.acc[key]--
				f.parts = append(f.parts, particle{
					x: float64(b.X) + f.rng.Float64()*float64(bt.W), y: b.BaseY + 0.05, z: float64(b.Y) + f.rng.Float64()*float64(bt.H),
					vx: (f.rng.Float64() - 0.5) * 0.3, vy: 0.15 + f.rng.Float64()*0.2, vz: (f.rng.Float64() - 0.5) * 0.3,
					life: 1.5 + f.rng.Float64(), size: 0.1, grow: 0.25, white: 0.55, rot: f.rng.Float64() * 6, spin: 0.3, shade: 1.15,
				})
			}
		}
	}
	// Ausstoß
	if Opt.Particles {
		for _, b := range g.allBuildings() {
			ch, ok := chimneys[b.T]
			if !ok || b.Paused || b.Prod < 0.15 {
				continue
			}
			cx, cz := b.Center()
			if math.Hypot(cx-u.TX, cz-u.TZ) > u.Dist*1.4+18 {
				continue
			}
			for k, c := range ch {
				key := b.ID*8 + k
				rate := 5.0 * (0.4 + 0.6*b.Prod)
				if c[3] > 0.9 {
					rate = 7
				}
				f.acc[key] += dt * rate
				for f.acc[key] >= 1 && len(f.parts) < 3500 {
					f.acc[key]--
					big := 1.0
					if b.T == BFusion {
						big = 1.8
					}
					p := particle{
						x: float64(b.X) + c[0] + (f.rng.Float64()-0.5)*0.08, y: b.BaseY + c[1], z: float64(b.Y) + c[2] + (f.rng.Float64()-0.5)*0.08,
						vx: (f.rng.Float64() - 0.5) * 0.08, vy: 0.45 + f.rng.Float64()*0.25, vz: (f.rng.Float64() - 0.5) * 0.08,
						life: 4 + f.rng.Float64()*2.5, size: 0.12 * big, grow: 0.22 * big,
						white: c[3], rot: f.rng.Float64() * 6, spin: (f.rng.Float64() - 0.5) * 0.6, shade: 0.85 + f.rng.Float64()*0.3,
					}
					f.parts = append(f.parts, p)
				}
			}
		}
	}
	k := f.parts[:0]
	for _, p := range f.parts {
		p.age += dt
		if p.age >= p.life || !Opt.Particles {
			continue
		}
		p.x += (p.vx + wind.X*p.age*0.35) * dt
		p.y += p.vy * dt
		p.z += (p.vz + wind.Z*p.age*0.35) * dt
		p.vy *= 1 - dt*0.25
		p.rot += p.spin * dt
		k = append(k, p)
	}
	f.parts = k
	// Gleiter-Taxis
	want := 0
	if Opt.Traffic {
		pop := g.Pop[0] + g.Pop[1]*2 + g.Pop[2]*4
		want = imin(70, pop/45)
	}
	for len(f.cars) < want {
		f.cars = append(f.cars, f.newCar(g))
	}
	if len(f.cars) > want {
		f.cars = f.cars[:want]
	}
	for i := range f.cars {
		c := &f.cars[i]
		d := math.Hypot(c.bx-c.ax, c.bz-c.az)
		c.t += dt * c.speed / math.Max(1, d)
		if c.t >= 1 {
			*c = f.newCar(g)
			continue
		}
		e := c.t * c.t * (3 - 2*c.t)
		c.x = c.ax + (c.bx-c.ax)*e
		c.z = c.az + (c.bz-c.az)*e
		c.y = c.h + math.Sin(c.t*math.Pi)*0.6
	}
}

func (f *FX) newCar(g *Game) hovercar {
	var hs []*Building
	for _, b := range g.B {
		if b.T == BHouse || b.T == BKontor || b.T == BKomm || b.T == BMedi || b.T == BLabor {
			hs = append(hs, b)
		}
	}
	if len(hs) < 2 {
		return hovercar{t: 1}
	}
	a := hs[f.rng.Intn(len(hs))]
	b := hs[f.rng.Intn(len(hs))]
	ax, az := a.Center()
	bx, bz := b.Center()
	tints := []RGB{mWhite, mWhite, hex(0xffd04a), mTeal, hex(0xff6a4a)}
	return hovercar{ax: ax, az: az, bx: bx, bz: bz, x: ax, z: az, h: 1.2 + f.rng.Float64()*2.6 + math.Max(a.BaseY, b.BaseY),
		t: f.rng.Float64() * 0.5, speed: 2.2 + f.rng.Float64()*1.8, tint: tints[f.rng.Intn(len(tints))]}
}

func (f *FX) uploadCars() {
	var data []float32
	for _, c := range f.cars {
		if c.t >= 1 {
			continue
		}
		ang := math.Atan2(c.bz-c.az, c.bx-c.ax)
		data = append(data, float32(c.x), float32(c.y), float32(c.z), float32(-ang), 1,
			float32(c.tint.R)/255, float32(c.tint.G)/255, float32(c.tint.B)/255)
	}
	f.carInst.Set(data)
}

func (f *FX) drawParticles(r *Renderer, u *UIState, ls sunSet) {
	if len(f.parts) == 0 {
		return
	}
	eye := u.Eye
	sort.Slice(f.parts, func(i, j int) bool {
		a, b := f.parts[i], f.parts[j]
		return (a.x-eye.X)*(a.x-eye.X)+(a.y-eye.Y)*(a.y-eye.Y)+(a.z-eye.Z)*(a.z-eye.Z) >
			(b.x-eye.X)*(b.x-eye.X)+(b.y-eye.Y)*(b.y-eye.Y)+(b.z-eye.Z)*(b.z-eye.Z)
	})
	data := make([]float32, 0, len(f.parts)*8)
	for _, p := range f.parts {
		t := p.age / p.life
		a := math.Min(1, p.age*3) * (1 - t) * 0.55
		if p.white > 0.9 {
			a *= 0.8
		}
		data = append(data, float32(p.x), float32(p.y), float32(p.z), float32(p.size+p.grow*p.age),
			float32(a), float32(p.shade), float32(p.white), float32(p.rot))
	}
	f.quad.Set(data)
	inv := u.InvVP
	_ = inv
	// Kamera-Achsen
	fwd := V3{u.TX, u.camTY, u.TZ}.Sub(u.Eye).Norm()
	right := fwd.Cross(V3{0, 1, 0}).Norm()
	up := right.Cross(fwd).Norm()
	p := r.pPart
	p.Use()
	glUniformMat(p.L("uVP"), &u.VP)
	glUniformV(p.L("uRight"), v3f(right)...)
	glUniformV(p.L("uUp"), v3f(up)...)
	glUniformV(p.L("uSunCol"), v3f(ls.sunCol)...)
	glUniformV(p.L("uSky"), v3f(ls.sky)...)
	glUniformV(p.L("uNight"), float32(ls.night))
	gl("glEnable", GL_BLEND)
	gl("glBlendFunc", GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA)
	gl("glDepthMask", 0)
	f.quad.Draw()
	gl("glDepthMask", 1)
}

// Schwebe-Taxi
func HoverCarModel() *MB {
	if Opt.KenneyModels && KM("sp_craft_speederA") != nil {
		b := KenneyCentered("sp_craft_speederA", 0.5, 1, accentTo(mCyan, true))
		b.Box(-0.26, 0.04, -0.06, -0.24, 0.08, 0.06, hex(0xff3040), MatEmit)
		return b
	}
	b := NewMB()
	b.Prism([][2]float64{{-0.22, -0.09}, {0.16, -0.09}, {0.26, 0}, {0.16, 0.09}, {-0.22, 0.09}}, -0.04, 0.05, mWhite, MatPanel, mLight, MatPanel)
	b.Sphere(0.04, 0.05, 0, 0.13, 0.07, 0.08, 0.5, 1, 4, 10, hex(0x1a2a3a), MatGlass)
	b.Box(-0.23, -0.02, -0.08, -0.21, 0.03, 0.08, hex(0xff3040), MatEmit)
	b.Box(0.24, -0.02, -0.04, 0.27, 0.02, 0.04, hex(0xfff2d0), MatEmit)
	b.Box(-0.15, -0.07, -0.07, 0.12, -0.04, 0.07, mCyan, MatEmit)
	return b
}

// ---------------------------------------------------------------- Bewohner, Möwen, Boote

type walker struct {
	tx, ty, nx, ny int
	px, py         int
	t, speed, side float64
	kind           int
}

type bird struct {
	cx, cz, r, h, ph, sp float64
}

type boat struct {
	x, z, ax, az, bx, bz float64
	t, speed             float64
	kind                 int
}

type Life struct {
	walkers []walker
	birds   []bird
	boats   []boat
	people  [4]*Instanced
	birdI   *Instanced
	boatI   *Instanced
	rng     *rand.Rand
	roads   [][2]int
	roadSig int
}

func PersonModel(shirt RGB) *MB {
	b := NewMB()
	b.Cyl(0, 0, 0.012, 0, 0.032, 6, hex(0x2a3040), MatMatte, false)
	b.Cone(0, 0, 0.016, 0.013, 0.032, 0.066, 6, shirt, MatMatte, true)
	b.Sphere(0, 0.078, 0, 0.011, 0.012, 0.011, 0, 1, 4, 6, hex(0xd8a888), MatMatte)
	return b
}

func BirdModel() *MB {
	b := NewMB()
	w := hex(0xf4f6f8)
	b.Tri(V3{0.04, 0, 0}, V3{-0.03, 0, 0.12}, V3{-0.02, 0, 0}, w, MatBird)
	b.Tri(V3{0.04, 0, 0}, V3{-0.02, 0, 0}, V3{-0.03, 0, -0.12}, w, MatBird)
	b.Box(-0.05, -0.008, -0.008, 0.05, 0.008, 0.008, w, MatMatte)
	return b
}

func BoatModel() *MB {
	if Opt.KenneyModels && KM("wc_boat-sail-a") != nil {
		return KenneyCentered("wc_boat-sail-a", 0.85, 1, nil)
	}
	b := NewMB()
	b.Prism([][2]float64{{-0.35, -0.12}, {0.25, -0.12}, {0.42, 0}, {0.25, 0.12}, {-0.35, 0.12}}, 0.0, 0.1, mWhite, MatPanel, hex(0xb89060), MatMatte)
	b.BoxC(-0.15, 0.1, -0.08, 0.1, 0.2, 0.08, mWhite, mLight, MatPanel, MatPanel)
	b.Box(-0.151, 0.13, -0.081, 0.101, 0.18, 0.081, mWhite, MatGlass)
	b.Cyl(0.0, 0, 0.008, 0.2, 0.75, 4, mLight, MatMetal, true)
	b.Tri(V3{0.02, 0.24, 0}, V3{0.02, 0.72, 0}, V3{0.32, 0.24, 0}, hex(0xf2f4f6), MatPanel)
	b.Box(-0.36, 0.02, -0.06, -0.34, 0.06, 0.06, mCyan, MatEmit)
	return b
}

func newLife() *Life {
	l := &Life{rng: rand.New(rand.NewSource(9))}
	shirts := []RGB{hex(0x3a7bd5), hex(0xe0e4e8), hex(0xff7a4a), hex(0x4fc08a)}
	for i := range l.people {
		l.people[i] = NewInstanced(PersonModel(shirts[i]).Upload())
	}
	l.birdI = NewInstanced(BirdModel().Upload())
	l.boatI = NewInstanced(BoatModel().Upload())
	return l
}

func (l *Life) update(g *Game, u *UIState, dt float64) {
	w := g.W
	if dt > 0.1 {
		dt = 0.1
	}
	if !Opt.Life || w.Moon {
		l.walkers, l.birds, l.boats = nil, nil, nil
		return
	}
	// Straßenkacheln nahe der Kamera
	sig := int(u.TX/6)*1000 + int(u.TZ/6) + w.Sig()*7
	if sig != l.roadSig {
		l.roadSig = sig
		l.roads = l.roads[:0]
		rad := int(u.Dist*0.9) + 12
		for y := int(u.TZ) - rad; y < int(u.TZ)+rad; y++ {
			for x := int(u.TX) - rad; x < int(u.TX)+rad; x++ {
				if w.In(x, y) && w.Road[w.Idx(x, y)] {
					l.roads = append(l.roads, [2]int{x, y})
				}
			}
		}
	}
	pop := g.Pop[0] + g.Pop[1] + g.Pop[2]
	want := imin(600, pop/3)
	if len(l.roads) < 4 {
		want = 0
	}
	for len(l.walkers) < want {
		r := l.roads[l.rng.Intn(len(l.roads))]
		wk := walker{tx: r[0], ty: r[1], nx: r[0], ny: r[1], px: -1, speed: 0.25 + l.rng.Float64()*0.2, side: []float64{-0.36, 0.36}[l.rng.Intn(2)], kind: l.rng.Intn(4)}
		l.walkers = append(l.walkers, wk)
	}
	if len(l.walkers) > want {
		l.walkers = l.walkers[:want]
	}
	for i := range l.walkers {
		k := &l.walkers[i]
		k.t += dt * k.speed
		if k.t >= 1 || (k.nx == k.tx && k.ny == k.ty) {
			k.t = 0
			k.px, k.py = k.tx, k.ty
			k.tx, k.ty = k.nx, k.ny
			var opts [][2]int
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, y := k.tx+d[0], k.ty+d[1]
				if w.In(x, y) && w.Road[w.Idx(x, y)] && !(x == k.px && y == k.py) {
					opts = append(opts, [2]int{x, y})
				}
			}
			if len(opts) == 0 {
				k.nx, k.ny = k.px, k.py
			} else {
				o := opts[l.rng.Intn(len(opts))]
				k.nx, k.ny = o[0], o[1]
			}
			if math.Hypot(float64(k.tx)-u.TX, float64(k.ty)-u.TZ) > u.Dist*1.3+20 && len(l.roads) > 0 {
				r := l.roads[l.rng.Intn(len(l.roads))]
				k.tx, k.ty, k.nx, k.ny = r[0], r[1], r[0], r[1]
			}
		}
	}
	// Möwen über der Küste
	if len(l.birds) == 0 {
		for f := 0; f < 5; f++ {
			cx := u.TX + (l.rng.Float64()-0.5)*40
			cz := u.TZ + (l.rng.Float64()-0.5)*40
			for i := 0; i < 7; i++ {
				l.birds = append(l.birds, bird{cx: cx + l.rng.Float64()*2, cz: cz + l.rng.Float64()*2, r: 3 + l.rng.Float64()*5, h: 3.5 + l.rng.Float64()*2.5, ph: l.rng.Float64() * 6.28, sp: 0.25 + l.rng.Float64()*0.15})
			}
		}
	}
	for i := range l.birds {
		b := &l.birds[i]
		b.ph += dt * b.sp
		// Schwarm langsam zur Kamera ziehen
		if math.Hypot(b.cx-u.TX, b.cz-u.TZ) > u.Dist+25 {
			b.cx += (u.TX - b.cx) * dt * 0.05
			b.cz += (u.TZ - b.cz) * dt * 0.05
		}
	}
	// Boote
	wantB := 14
	for len(l.boats) < wantB {
		l.boats = append(l.boats, l.newBoat(g, u))
	}
	for i := range l.boats {
		b := &l.boats[i]
		d := math.Hypot(b.bx-b.ax, b.bz-b.az)
		b.t += dt * b.speed / math.Max(1, d)
		if b.t >= 1 {
			*b = l.newBoat(g, u)
			continue
		}
		b.x = b.ax + (b.bx-b.ax)*b.t
		b.z = b.az + (b.bz-b.az)*b.t
	}
}

func (l *Life) waterSpot(g *Game, cx, cz float64, rad float64) (float64, float64, bool) {
	for i := 0; i < 40; i++ {
		x := cx + (l.rng.Float64()-0.5)*rad*2
		z := cz + (l.rng.Float64()-0.5)*rad*2
		xi, zi := int(x), int(z)
		if g.W.In(xi, zi) {
			j := g.W.Idx(xi, zi)
			if g.W.Terr[j] == TWater && g.W.Dist[j] >= 2 && g.W.Dist[j] <= 7 {
				return x, z, true
			}
		}
	}
	return 0, 0, false
}

func (l *Life) newBoat(g *Game, u *UIState) boat {
	ax, az, ok := l.waterSpot(g, u.TX, u.TZ, 45)
	if !ok {
		return boat{t: 1}
	}
	for i := 0; i < 10; i++ {
		bx, bz, ok2 := l.waterSpot(g, ax, az, 14)
		if !ok2 {
			continue
		}
		clear := true
		for s := 0.0; s <= 1; s += 0.05 {
			x, z := ax+(bx-ax)*s, az+(bz-az)*s
			if g.W.HeightAt(x, z) > -0.1 {
				clear = false
				break
			}
		}
		if clear {
			return boat{x: ax, z: az, ax: ax, az: az, bx: bx, bz: bz, speed: 0.6 + l.rng.Float64()*0.6, t: 0}
		}
	}
	return boat{t: 1}
}

func (l *Life) upload(g *Game, t float64) {
	w := g.W
	var data [4][]float32
	for _, k := range l.walkers {
		f := k.t
		ax, az := float64(k.tx)+0.5, float64(k.ty)+0.5
		bx, bz := float64(k.nx)+0.5, float64(k.ny)+0.5
		dx, dz := bx-ax, bz-az
		dl := math.Max(0.001, math.Hypot(dx, dz))
		px, pz := -dz/dl*k.side, dx/dl*k.side
		if dl < 0.01 {
			px, pz = k.side, 0
		}
		x, z := ax+dx*f+px, az+dz*f+pz
		y := w.HeightAt(x, z) + 0.03 + math.Abs(math.Sin(t*9+float64(k.tx*7+k.ty)))*0.006
		ang := math.Atan2(dz, dx)
		data[k.kind] = append(data[k.kind], float32(x), float32(y), float32(z), float32(-ang), 2.1, 1, 1, 1)
	}
	for i := range l.people {
		l.people[i].Set(data[i])
	}
	var bd []float32
	for _, b := range l.birds {
		x := b.cx + math.Cos(b.ph)*b.r
		z := b.cz + math.Sin(b.ph)*b.r
		y := b.h + math.Sin(b.ph*2.3)*0.3
		bd = append(bd, float32(x), float32(y), float32(z), float32(-(b.ph + math.Pi/2)), 0.75, 1, 1, 1)
	}
	l.birdI.Set(bd)
	var od []float32
	for _, b := range l.boats {
		if b.t >= 1 {
			continue
		}
		ang := math.Atan2(b.bz-b.az, b.bx-b.ax)
		od = append(od, float32(b.x), float32(0.02+math.Sin(t*1.7+b.ax)*0.015), float32(b.z), float32(-ang), 1.4, 1, 1, 1)
	}
	l.boatI.Set(od)
}

func (l *Life) draw(shadow bool) {
	if !Opt.Life {
		return
	}
	for _, p := range l.people {
		p.Draw()
	}
	l.boatI.Draw()
	if !shadow {
		l.birdI.Draw()
	}
}

func (f *FX) event(e GEvent) {
	switch e.Kind {
	case EvBuilt:
		if !Opt.Particles {
			return
		}
		n := 18 + int(e.S*8)
		for i := 0; i < n && len(f.parts) < 3800; i++ {
			a := f.rng.Float64() * 6.283
			sp := 0.6 + f.rng.Float64()*0.8
			f.parts = append(f.parts, particle{
				x: e.X + math.Cos(a)*e.S*0.4, y: e.Y + 0.05, z: e.Z + math.Sin(a)*e.S*0.4,
				vx: math.Cos(a) * sp, vy: 0.2 + f.rng.Float64()*0.3, vz: math.Sin(a) * sp,
				life: 1.4 + f.rng.Float64(), size: 0.15, grow: 0.35, white: 0.6, rot: a, spin: 0.5, shade: 1.2,
			})
		}
	case EvExplode:
		sc := math.Max(0.6, e.S)
		for i := 0; i < 40 && len(f.parts) < 3900 && Opt.Particles; i++ {
			a := f.rng.Float64() * 6.283
			el := f.rng.Float64() * 1.4
			sp := (0.8 + f.rng.Float64()*1.6) * sc
			f.parts = append(f.parts, particle{
				x: e.X, y: e.Y, z: e.Z,
				vx: math.Cos(a) * math.Cos(el) * sp, vy: math.Sin(el)*sp + 0.4, vz: math.Sin(a) * math.Cos(el) * sp,
				life: 1.0 + f.rng.Float64()*1.0, size: 0.14 * sc, grow: 0.32, white: 0.05, rot: a, spin: 1, shade: 0.5 + f.rng.Float64()*0.4,
			})
		}
		f.flashes = append(f.flashes, flash{x: e.X, y: e.Y + 0.3, z: e.Z, life: 0.55, size: 1.8 * sc, col: hex(0xffa040)})
	case EvSplash:
		for i := 0; i < 14 && len(f.parts) < 3900 && Opt.Particles; i++ {
			a := f.rng.Float64() * 6.283
			f.parts = append(f.parts, particle{
				x: e.X, y: 0.05, z: e.Z, vx: math.Cos(a) * 0.4, vy: 1.2 + f.rng.Float64()*0.8, vz: math.Sin(a) * 0.4,
				life: 0.9, size: 0.08, grow: 0.15, white: 1, rot: a, spin: 0.2, shade: 1.3,
			})
		}
	case EvLaser:
		f.beams = append(f.beams, beam{x1: e.X, y1: e.Y, z1: e.Z, x2: e.X2, y2: e.Y2, z2: e.Z2, life: 0.35, col: e.Col})
		f.flashes = append(f.flashes, flash{x: e.X2, y: e.Y2, z: e.Z2, life: 0.3, size: 0.7, col: e.Col})
		f.flashes = append(f.flashes, flash{x: e.X, y: e.Y, z: e.Z, life: 0.2, size: 0.45, col: e.Col})
	case EvFire:
		if Opt.Particles && len(f.parts) < 3900 {
			f.parts = append(f.parts, particle{
				x: e.X + (f.rng.Float64()-0.5)*e.S, y: e.Y, z: e.Z + (f.rng.Float64()-0.5)*e.S,
				vx: (f.rng.Float64() - 0.5) * 0.1, vy: 0.7 + f.rng.Float64()*0.4, vz: (f.rng.Float64() - 0.5) * 0.1,
				life: 2.5 + f.rng.Float64()*1.5, size: 0.15, grow: 0.35, white: 0.02, rot: f.rng.Float64() * 6, spin: 0.6, shade: 0.35,
			})
		}
		f.flashes = append(f.flashes, flash{x: e.X + (f.rng.Float64()-0.5)*e.S, y: e.Y + 0.2, z: e.Z + (f.rng.Float64()-0.5)*e.S, life: 0.6, size: 0.5 * e.S, col: hex(0xff6a18)})
	}
}

// Leuchteffekte (Mündungsblitze, Laser, Explosionen, Feuer) additiv zeichnen
func (f *FX) drawGlowFX(r *Renderer, u *UIState) {
	if len(f.flashes) == 0 && len(f.beams) == 0 {
		return
	}
	if f.glowMB == nil {
		f.glowMB = NewMB()
	}
	b := f.glowMB
	b.V, b.I = b.V[:0], b.I[:0]
	fwd := V3{u.TX, 0, u.TZ}.Sub(u.Eye).Norm()
	right := fwd.Cross(V3{0, 1, 0}).Norm()
	up := right.Cross(fwd).Norm()
	for _, x := range f.flashes {
		k := 1 - x.t/x.life
		s := x.size * (0.6 + 0.4*(1-k))
		c := V3{x.x, x.y, x.z}
		k0 := uint32(len(b.V) / 10)
		for _, q := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			p := c.Add(right.Mul(q[0] * s)).Add(up.Mul(q[1] * s))
			b.V = append(b.V, float32(p.X), float32(p.Y), float32(p.Z), float32(q[0]), float32(q[1]), 0,
				float32(float64(x.col.R)/255*2.2*k), float32(float64(x.col.G)/255*2.2*k), float32(float64(x.col.B)/255*2.2*k), 0)
		}
		b.I = append(b.I, k0, k0+1, k0+2, k0, k0+2, k0+3)
	}
	for _, x := range f.beams {
		k := 1 - x.t/x.life
		a, c := V3{x.x1, x.y1, x.z1}, V3{x.x2, x.y2, x.z2}
		d := c.Sub(a)
		side := d.Cross(fwd).Norm().Mul(0.1)
		k0 := uint32(len(b.V) / 10)
		for _, q := range [][2]float64{{-1, 0}, {1, 0}, {1, 1}, {-1, 1}} {
			p := a.Add(side.Mul(q[0])).Add(d.Mul(q[1]))
			b.V = append(b.V, float32(p.X), float32(p.Y), float32(p.Z), float32(q[0]), 0, 0,
				float32(float64(x.col.R)/255*2.5*k), float32(float64(x.col.G)/255*2.5*k), float32(float64(x.col.B)/255*2.5*k), 0)
		}
		b.I = append(b.I, k0, k0+1, k0+2, k0, k0+2, k0+3)
	}
	if f.glowM == nil {
		f.glowM = b.Upload()
	} else {
		f.glowM.Update(b)
	}
	gl("glEnable", GL_BLEND)
	gl("glBlendFunc", GL_ONE, GL_ONE)
	gl("glDepthMask", 0)
	r.pGlow.Use()
	glUniformMat(r.pGlow.L("uVP"), &u.VP)
	glUniformV(r.pGlow.L("uNight"), float32(1.0))
	f.glowM.Draw()
	gl("glDepthMask", 1)
	gl("glDisable", GL_BLEND)
}
