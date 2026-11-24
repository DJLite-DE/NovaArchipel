package game

import (
	"math"
)

// ---------------------------------------------------------------- Regionen: Erde und Mond

// Eine Region hält alles, was zu einer Karte gehört. Die aktive Region liegt
// direkt im Game (g.W, g.B …), die andere wartet in g.Other.
type Region struct {
	W       *World
	Islands []*Island
	B       []*Building
	BP      []*Building
	RB      []*Building
	Ships   []*Ship
	Carts   []*Cart
	Dirty   bool
	TX, TZ  float64
	Dist    float64
	Yaw     float64
}

// Inhalte tauschen (für die Simulation, ohne Kamera)
func (g *Game) swapRegion() {
	if g.Other == nil {
		return
	}
	cur := &Region{W: g.W, Islands: g.Islands, B: g.B, BP: g.BP, RB: g.RB, Ships: g.Ships, Carts: g.Carts, Dirty: g.dirty,
		TX: g.Other.TX, TZ: g.Other.TZ, Dist: g.Other.Dist, Yaw: g.Other.Yaw}
	o := g.Other
	g.W, g.Islands, g.B, g.BP, g.RB, g.Ships, g.Carts, g.dirty = o.W, o.Islands, o.B, o.BP, o.RB, o.Ships, o.Carts, o.Dirty
	g.Other = cur
	g.Reg = 1 - g.Reg
}

// Sichtbarer Wechsel mit eigener Kamera je Region
func (g *Game) SwitchRegion() {
	if g.Other == nil {
		return
	}
	u := g.UI
	tx, tz, d, y := u.TX, u.TZ, u.Dist, u.Yaw
	ntx, ntz, nd, ny := g.Other.TX, g.Other.TZ, g.Other.Dist, g.Other.Yaw
	g.swapRegion()
	g.Other.TX, g.Other.TZ, g.Other.Dist, g.Other.Yaw = tx, tz, d, y
	if nd == 0 {
		nd, ny = 30, math.Pi/4
		if len(g.Islands) > 0 {
			ntx, ntz = float64(g.Islands[0].CX), float64(g.Islands[0].CY)
		}
	}
	u.TX, u.TZ, u.Dist, u.Yaw = ntx, ntz, nd, ny
	u.SelB, u.SelShip, u.SelBP, u.Tool = 0, 0, 0, -1
	u.world = g.W
	u.mini = nil
	PlaySfx(SfxShip)
}

func (g *Game) OnMoon() bool { return g.W != nil && g.W.Moon }

// Daten einer Region unabhängig davon, ob sie gerade aktiv ist
func (g *Game) regionData(r int) ([]*Island, []*Building) {
	if g.Reg == r {
		return g.Islands, g.B
	}
	if g.Other == nil {
		return nil, nil
	}
	return g.Other.Islands, g.Other.B
}

// Mondbasis freischalten
func (g *Game) createMoon() {
	w, isl := GenMoon(g.Seed)
	reg := &Region{W: w, Islands: isl, Dirty: true}
	// Landefähre auf dem Staubmeer vor dem Tranquility-Plateau
	I := isl[0]
	bx, by, bd := -1, -1, 1e9
	for y := 0; y < w.N; y++ {
		for x := 0; x < w.N; x++ {
			i := w.Idx(x, y)
			if w.Terr[i] == TWater && w.Dist[i] >= 2 && w.Dist[i] <= 3 {
				d := math.Hypot(float64(x-I.CX), float64(y-I.CY))
				if d < bd {
					bx, by, bd = x, y, d
				}
			}
		}
	}
	if bx >= 0 {
		sh := &Ship{ID: g.newID(), Name: "Mondgleiter I", X: float64(bx) + 0.5, Y: float64(by) + 0.5}
		sh.Cargo[0] = Cargo{GBaustoffe, 50}
		sh.Cargo[1] = Cargo{GStahl, 30}
		sh.Cargo[2] = Cargo{GNahrung, 30}
		sh.Cargo[3] = Cargo{GWasser, 30}
		sh.Route = Route{GAB: -1, GBA: -1}
		reg.Ships = append(reg.Ships, sh)
	}
	reg.TX, reg.TZ, reg.Dist, reg.Yaw = float64(I.CX)+6, float64(I.CY)+6, 32, math.Pi/4
	g.Other = reg
	if g.Reg != 0 {
		// darf nicht passieren, aber sicher ist sicher
		g.Reg = 0
	}
	g.RocketT = 60
	g.Dialogs = append(g.Dialogs, Dialog{SpVoss, "Der Raumhafen ist startklar! Eine Landefähre mit Baumaterial wartet bereits auf dem Staubmeer des Mondes. Wechsle oben rechts mit „Mond“ dorthin und gründe einen Kontor auf dem Tranquility-Plateau."},
		Dialog{SpVoss, "Auf dem Mond gibt es Helium-3. Damit laufen deine Fusionsreaktoren auf der Erde 60 % stärker. Nahrung, Wasser und Baustoffe bringen die Raketen automatisch nach oben."})
	g.Notify("Die Mondbasis ist freigeschaltet!", hex(0xe8f4ff))
	PlaySfx(SfxQuest)
}

// Raketen zwischen Raumhafen (Erde) und Kontor (Mond)
func (g *Game) updateRockets(dt float64) {
	if g.Other == nil {
		// Raumhafen fertig? → Mond freischalten
		if g.Reg == 0 {
			for _, b := range g.B {
				if b.T == BRaumhafen && b.Constr == 0 {
					g.createMoon()
					break
				}
			}
		}
		return
	}
	g.Launch += dt
	eIsl, eB := g.regionData(0)
	mIsl, mB := g.regionData(1)
	var port *Building
	for _, b := range eB {
		if b.T == BRaumhafen && b.Constr == 0 && !b.Paused && b.Fire == 0 {
			port = b
		}
	}
	var mk *Building
	for _, b := range mB {
		if b.T == BKontor {
			mk = b
			break
		}
	}
	if port == nil || mk == nil {
		return
	}
	g.RocketT -= dt
	if g.RocketT > 0 {
		return
	}
	g.RocketT = map[bool]float64{true: 60, false: 90}[g.HasTech(TechRaketen)]
	E := eIsl[port.Isl]
	M := mIsl[mk.Isl]
	up, down := 0, 0
	capa := 120
	for _, gd := range []int{GNahrung, GWasser, GBaustoffe, GStahl, GSynthkaffee, GTextil, GChips, GHolo} {
		// Ware von der Erdinsel mit dem größten Vorrat
		var src *Island
		for _, I := range eIsl {
			if I.Owned && (src == nil || I.Stock[gd] > src.Stock[gd]) {
				src = I
			}
		}
		if src == nil {
			continue
		}
		need := math.Min(float64(M.Cap)-M.Stock[gd], 60-M.Stock[gd])
		have := src.Stock[gd] - 10
		n := int(math.Min(math.Min(need, have), 30))
		if n <= 0 || capa <= 0 {
			continue
		}
		n = imin(n, capa)
		src.Stock[gd] -= float64(n)
		M.Stock[gd] += float64(n)
		capa -= n
		up += n
	}
	if h := int(M.Stock[GHelium3]); h > 0 {
		h = imin(h, E.Cap-int(E.Stock[GHelium3]))
		if h > 0 {
			M.Stock[GHelium3] -= float64(h)
			E.Stock[GHelium3] += float64(h)
			down = h
		}
	}
	if up+down > 0 {
		g.Launch = 0
		g.LaunchPos = [2]float64{float64(port.X) + 2, float64(port.Y) + 2}
		g.LaunchY = port.BaseY
		msg := Tf("Rakete gestartet: %d Waren zum Mond", up)
		if down > 0 {
			msg += Tf(", %d Helium-3 zur Erde", down)
		}
		g.Notify(msg, hex(0xe8f4ff))
	}
}

// Ausnahmen auf dem Mond
func (g *Game) moonRule(t int) string {
	bt := &BTypes[t]
	if g.OnMoon() {
		if t == BRaumhafen || bt.Fert != FertNone || t == BAlgen || t == BWasser || t == BNahrung || t == BPark {
			return "Auf dem Mond nicht möglich"
		}
		return ""
	}
	if t == BRaumhafen {
		for _, b := range g.B {
			if b.T == BRaumhafen {
				return "Es gibt bereits einen Raumhafen"
			}
		}
	}
	return ""
}

// Für Vorschaubilder: Raumhafen bauen, Mond freischalten und eine Mondbasis errichten
func (g *Game) BuildMoonDemo() {
	if g.Other == nil {
		for y := 0; y < g.W.N-4 && g.Other == nil; y++ {
			for x := 0; x < g.W.N-4; x++ {
				i := g.W.Idx(x, y)
				is := int(g.W.Isl[i])
				if is < 0 || !g.Islands[is].Owned || !g.okFoot(x, y, 4, 4) {
					continue
				}
				if b := g.demoPlace(BRaumhafen, x, y); b != nil {
					b.Constr = 0
					g.connectRoad(b)
					break
				}
			}
			if len(g.B) > 0 && g.B[len(g.B)-1].T == BRaumhafen {
				break
			}
		}
		g.updateRockets(0)
		g.Dialogs = nil
	}
	if g.Other == nil {
		return
	}
	g.swapRegion()
	I := g.Islands[0]
	kx, ky, ok := g.findKontorSpot(0, float64(I.CX)+4, float64(I.CY)+10)
	if !ok {
		g.swapRegion()
		return
	}
	g.Credits += float64(BTypes[BKontor].Cost)
	g.forcePlace(BKontor, kx, ky, nil)
	g.Islands[0].Stock[GBaustoffe] += 200
	g.Islands[0].Stock[GStahl] += 200
	g.Islands[0].Stock[GChips] += 100
	g.Recompute()
	// Straßenraster
	ox, oy := kx+3, ky+3
	for y := ky - 14; y < ky+18; y++ {
		for x := kx - 14; x < kx+18; x++ {
			if !g.W.In(x, y) || math.Hypot(float64(x-kx), float64(y-ky)) > 15 {
				continue
			}
			if ((x-ox)%5+5)%5 != 0 && ((y-oy)%5+5)%5 != 0 {
				continue
			}
			i := g.W.Idx(x, y)
			if g.W.Infl[i] && g.W.Terr[i] != TWater && g.W.Terr[i] != TRock && g.W.Occ[i] == 0 {
				g.W.Road[i] = true
				g.W.Deco[i] = 0
			}
		}
	}
	g.Recompute()
	types := []int{BHouse, BHouse, BSolar, BHouse, BLager, BHouse, BKomm, BHouse, BSolar, BHouse, BMedi, BHouse, BChip, BHouse, BLabor, BHouse, BFusion}
	k := 0
	for r := 2; r < 16 && k < len(types); r++ {
		for y := oy - r*5; y <= oy+r*5 && k < len(types); y += 5 {
			for x := ox - r*5; x <= ox+r*5 && k < len(types); x += 5 {
				bx, by := x+1, y+1
				if math.Hypot(float64(bx-kx), float64(by-ky)) > float64(r*5) {
					continue
				}
				t := types[k]
				if t == BHouse {
					n := 0
					for _, d := range [][2]int{{0, 0}, {2, 0}, {0, 2}, {2, 2}} {
						if b := g.demoPlace(BHouse, bx+d[0], by+d[1]); b != nil {
							b.Tier = 2 - (n % 2)
							b.Res = float64(Tiers[b.Tier].MaxRes)
							n++
						}
					}
					if n > 0 {
						k++
					}
				} else if g.demoPlace(t, bx, by) != nil {
					k++
				}
			}
		}
	}
	// Helium-3-Förderung auf den Vorkommen
	for y := 0; y < g.W.N; y++ {
		for x := 0; x < g.W.N; x++ {
			if g.W.Dep[g.W.Idx(x, y)] == DepHe3 && g.CanPlace(BHe3, x, y).OK {
				if b := g.demoPlace(BHe3, x, y); b != nil {
					g.connectRoad(b)
				}
			}
		}
	}
	for _, b := range g.B {
		b.Constr = 0
	}
	g.Recompute()
	g.swapRegion()
	g.Other.TX, g.Other.TZ, g.Other.Dist = float64(kx)+4, float64(ky)+4, 24
}
