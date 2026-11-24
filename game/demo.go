package game

import (
	"math"
	"sort"
)

// Automatischer Städtebauer – für den Menü-Hintergrund und Vorschaubilder

func (g *Game) findKontorSpot(isl int, nearX, nearY float64) (int, int, bool) {
	best, bd := [2]int{}, 1e9
	found := false
	I := g.Islands[isl]
	for y := I.CY - 40; y < I.CY+40; y++ {
		for x := I.CX - 40; x < I.CX+40; x++ {
			if !g.W.In(x, y) || int(g.W.Isl[g.W.Idx(x, y)]) != isl {
				continue
			}
			if g.islandAtFoot(x, y, 3, 3) != isl {
				continue
			}
			ok := true
			for yy := y; yy < y+3 && ok; yy++ {
				for xx := x; xx < x+3; xx++ {
					i := g.W.Idx(xx, yy)
					if g.W.Terr[i] == TWater || g.W.Terr[i] == TRock || g.W.Occ[i] != 0 {
						ok = false
						break
					}
				}
			}
			if !ok {
				continue
			}
			if c, _ := g.coastal(x, y, 3, 3); !c {
				continue
			}
			d := math.Hypot(float64(x)-nearX, float64(y)-nearY)
			if d < bd {
				bd = d
				best = [2]int{x, y}
				found = true
			}
		}
	}
	return best[0], best[1], found
}

func (g *Game) okFoot(x, y, w, h int) bool {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if !g.W.In(xx, yy) {
				return false
			}
			i := g.W.Idx(xx, yy)
			if !g.W.Infl[i] || g.W.Terr[i] == TWater || g.W.Terr[i] == TRock || g.W.Occ[i] != 0 || g.W.Road[i] {
				return false
			}
		}
	}
	return true
}

type block struct{ x, y int }

func (g *Game) demoPlace(t, x, y int) *Building {
	if !g.okFoot(x, y, BTypes[t].W, BTypes[t].H) && BTypes[t].Deposit == DepNone {
		return nil
	}
	I := g.Islands[g.W.Isl[g.W.Idx(x, y)]]
	for k, m := range BTypes[t].Mat {
		I.Stock[MatGoods[k]] += float64(m)
	}
	g.Credits += float64(BTypes[t].Cost)
	b := g.forcePlace(t, x, y, nil)
	g.Recompute()
	return b
}

func BuildDemo(seed int, phases int) *Game {
	g := NewGame(seed)
	g.Notifs = nil
	sh := g.Ships[0]
	kx, ky, ok := g.findKontorSpot(0, sh.X-4, sh.Y-6)
	if !ok {
		return g
	}
	g.forcePlace(BKontor, kx, ky, sh)
	g.Recompute()
	k := g.B[0]
	I := g.Islands[0]
	// Straßenraster
	period := 5
	ox, oy := kx+3, ky+3
	onLine := func(x, y int) bool {
		return ((x-ox)%period+period)%period == 0 || ((y-oy)%period+period)%period == 0
	}
	layRoads := func() {
		for y := 0; y < g.W.N; y++ {
			for x := 0; x < g.W.N; x++ {
				i := g.W.Idx(x, y)
				if onLine(x, y) && g.W.Infl[i] && g.W.Terr[i] != TWater && g.W.Terr[i] != TRock && g.W.Occ[i] == 0 && !g.W.Road[i] && int(g.W.Isl[i]) == 0 {
					g.W.Road[i] = true
					g.W.Deco[i] = 0
				}
			}
		}
		g.dirty = true
		g.Recompute()
	}
	layRoads()
	// Blöcke (Innenfläche 4x4)
	blocks := func() []block {
		var bs []block
		for y := oy - 60; y < oy+60; y += period {
			for x := ox - 60; x < ox+60; x += period {
				if g.okFoot(x+1, y+1, 4, 4) {
					bs = append(bs, block{x + 1, y + 1})
				}
			}
		}
		kcx, kcy := k.Center()
		sort.Slice(bs, func(i, j int) bool {
			return math.Hypot(float64(bs[i].x)-kcx, float64(bs[i].y)-kcy) < math.Hypot(float64(bs[j].x)-kcx, float64(bs[j].y)-kcy)
		})
		return bs
	}
	// Lagerhäuser für mehr Fläche (wachsen nach außen)
	for l := 0; l < 10; l++ {
		bs := blocks()
		kcx, kcy := k.Center()
		best, bd := -1, 0.0
		for i, b := range bs {
			d := math.Hypot(float64(b.x)-kcx, float64(b.y)-kcy)
			// Richtung Inselzentrum bevorzugen
			d -= 0.5 * math.Hypot(float64(b.x-I.CX), float64(b.y-I.CY))
			if best < 0 || d > bd {
				best, bd = i, d
			}
		}
		if best < 0 {
			break
		}
		g.demoPlace(BLager, bs[best].x+1, bs[best].y+1)
		layRoads()
	}
	bs := blocks()
	houses := func(b block) {
		for _, d := range [][2]int{{0, 0}, {2, 0}, {0, 2}, {2, 2}} {
			g.demoPlace(BHouse, b.x+d[0], b.y+d[1])
		}
	}
	plan := []int{-1, -1, BKomm, -1, -2, -1, -3, BMedi, -1, -4, -5, -1, BLabor, -7, -1, -6, -8, -1, -1, -1, -1, -1}
	if phases < 2 {
		plan[7] = -1
	}
	if phases < 3 {
		plan[12], plan[13], plan[16] = -1, -1, -1
	}
	for i, b := range bs {
		if i >= len(plan) {
			break
		}
		switch p := plan[i]; {
		case p == -1:
			houses(b)
		case p == -2:
			g.demoPlace(BSchrott, b.x, b.y)
			g.demoPlace(BSchrott, b.x+2, b.y)
			g.demoPlace(BRecycle, b.x, b.y+2)
			g.demoPlace(BRecycle, b.x+2, b.y+2)
		case p == -3:
			g.demoPlace(BNahrung, b.x, b.y)
			g.demoPlace(BNahrung, b.x+2, b.y)
			g.demoPlace(BWind, b.x, b.y+2)
			g.demoPlace(BWind, b.x+1, b.y+3)
			g.demoPlace(BPark, b.x+2, b.y+2)
			g.demoPlace(BWind, b.x+3, b.y+3)
		case p == -4:
			g.demoPlace(BTextil, b.x, b.y)
			g.demoPlace(BRoest, b.x+2, b.y)
			g.demoPlace(BStahl, b.x, b.y+2)
		case p == -5:
			g.demoPlace(BFaserPl, b.x, b.y)
			g.demoPlace(BWind, b.x+3, b.y)
			g.demoPlace(BWind, b.x+3, b.y+2)
		case p == -6:
			if phases >= 2 {
				g.demoPlace(BSolar, b.x, b.y)
			}
		case p == -7:
			if phases >= 3 {
				g.demoPlace(BHolo, b.x, b.y)
			}
		case p == -8:
			if phases >= 3 {
				g.demoPlace(BChip, b.x, b.y)
			}
		default:
			g.demoPlace(p, b.x, b.y)
		}
	}
	// Küstenbetriebe
	coast := func(t int, n int) {
		for c := 0; c < n; c++ {
			best, bd := [2]int{}, 1e9
			for y := ky - 30; y < ky+30; y++ {
				for x := kx - 30; x < kx+30; x++ {
					bt := &BTypes[t]
					if !g.okFoot(x, y, bt.W, bt.H) {
						continue
					}
					if ok, _ := g.coastal(x, y, bt.W, bt.H); !ok {
						continue
					}
					tb := &Building{T: t, X: x, Y: y}
					if len(g.adjComps(tb)) == 0 {
						continue
					}
					d := math.Hypot(float64(x-kx), float64(y-ky))
					if d < bd {
						bd = d
						best = [2]int{x, y}
					}
				}
			}
			if bd < 1e9 {
				g.demoPlace(t, best[0], best[1])
			}
		}
	}
	coast(BAlgen, 2)
	coast(BWasser, 2)
	coast(BWerft, 1)
	// Erzminen
	if phases >= 2 {
		for y := 0; y < g.W.N; y++ {
			for x := 0; x < g.W.N; x++ {
				i := g.W.Idx(x, y)
				if g.W.Dep[i] == DepErz && g.W.Isl[i] == 0 && g.W.Infl[i] && g.W.Occ[i] == 0 && (x == 0 || g.W.Dep[i-1] != DepErz) && (y == 0 || g.W.Dep[i-g.W.N] != DepErz) {
					I.Stock[GBaustoffe] += 10
					g.Credits += 1000
					if b := g.forcePlace(BErz, x, y, nil); b != nil {
						g.connectRoad(b)
					}
				}
			}
		}
		g.Recompute()
	}
	// Restflächen begrünen
	for y := ky - 40; y < ky+40; y++ {
		for x := kx - 40; x < kx+40; x++ {
			if g.okFoot(x, y, 1, 1) && hash2(x, y, 3) < 0.25 {
				near := false
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if g.W.In(x+d[0], y+d[1]) && g.W.Road[g.W.Idx(x+d[0], y+d[1])] {
						near = true
					}
				}
				if near {
					if hash2(x, y, 4) < 0.8 {
						g.demoPlace(BPark, x, y)
					} else {
						g.demoPlace(BWind, x, y)
					}
				}
			}
		}
	}
	// Straßen ohne Nutzen entfernen? (behalten – sieht nach Stadt aus)
	// Simulation mit Handelsnachschub
	g.Credits = 60000
	g.MaxPop = [NumTiers]int{}
	minutes := []float64{0, 6, 14, 26}[phases]
	for t := 0.0; t < minutes*60; t += 0.5 {
		for _, gd := range []int{GBaustoffe, GStahl, GSynthkaffee, GTextil, GHolo, GChips, GNahrung, GWasser} {
			if phases >= 1 && I.Stock[gd] < 30 && (gd == GNahrung || gd == GWasser || gd == GBaustoffe || phases >= 2) {
				I.Stock[gd] += 1
			}
		}
		if phases == 1 {
			for _, b := range g.B {
				if b.T == BHouse && b.Tier == 0 && b.Var%3 != 0 {
					b.Lock = true
				}
			}
		}
		if phases < 3 {
			// Ingenieure verhindern
			for _, b := range g.B {
				if b.T == BHouse && b.Tier == 1 {
					b.Lock = true
				}
			}
		}
		g.step(0.5)
	}
	for _, b := range g.B {
		b.Lock = false
	}
	// zweite Kolonie + Handelsroute
	if phases >= 2 {
		I2 := g.Islands[1]
		if x, y, ok := g.findKontorSpot(1, float64(kx), float64(ky)); ok {
			I2.Stock[GBaustoffe] = 20
			g.forcePlace(BKontor, x, y, nil)
			g.Recompute()
			k2 := g.B[len(g.B)-1]
			I2.Stock[GKaffee] = 30
			s2 := &Ship{ID: g.newID(), Name: "Helios I", X: float64(k2.Dock[0]) + 0.5, Y: float64(k2.Dock[1]) + 0.5}
			for i := range s2.Cargo {
				s2.Cargo[i].G = -1
			}
			s2.Route = Route{A: k2.ID, B: k.ID, GAB: GKaffee, GBA: GBaustoffe, Active: true}
			g.Ships = append(g.Ships, s2)
		}
	}
	sh.Cargo = [4]Cargo{{GBaustoffe, 30}, {GStahl, 12}, {-1, 0}, {-1, 0}}
	g.Notifs = nil
	g.StormOn, g.Storm, g.stormT, g.Hour = false, 0, 900, 10
	g.Quest = []int{0, 6, 9, 11}[phases]
	kcx, kcy := g.CityCenter()
	g.UI.CenterOn(g, kcx, kcy)
	g.UI.SelShip = 0
	for _, b := range g.B {
		b.Constr = 0
	}
	g.Events = nil
	return g
}

func (g *Game) CityCenter() (float64, float64) {
	var sx, sy float64
	n := 0
	for _, b := range g.B {
		if b.Isl == 0 {
			x, y := b.Center()
			sx += x
			sy += y
			n++
		}
	}
	if n == 0 {
		return float64(g.Islands[0].CX), float64(g.Islands[0].CY)
	}
	return sx / float64(n), sy / float64(n)
}

// Verbindet ein Gebäude per kürzester Straße mit einem Lager-Netz (Breitensuche über freies Land)
func (g *Game) connectRoad(b *Building) {
	g.Recompute()
	w := g.W
	n := w.N
	storeComp := map[int32]bool{}
	nearStore := map[int]bool{}
	for _, o := range g.B {
		if (o.T == BKontor || o.T == BLager) && o.Isl == b.Isl {
			for _, c := range g.adjComps(o) {
				storeComp[c] = true
			}
			ot := &BTypes[o.T]
			for yy := o.Y - 1; yy <= o.Y+ot.H; yy++ {
				for xx := o.X - 1; xx <= o.X+ot.W; xx++ {
					if w.In(xx, yy) {
						nearStore[w.Idx(xx, yy)] = true
					}
				}
			}
		}
	}
	for _, c := range g.adjComps(b) {
		if storeComp[c] {
			return
		}
	}
	isGoal := func(i int) bool {
		return (w.Road[i] && storeComp[w.RoadC[i]]) || (nearStore[i] && w.Occ[i] == 0 && w.Terr[i] != TWater)
	}
	bt := &BTypes[b.T]
	prev := map[int]int{}
	var q []int
	for yy := b.Y - 1; yy <= b.Y+bt.H; yy++ {
		for xx := b.X - 1; xx <= b.X+bt.W; xx++ {
			if (xx == b.X-1 || xx == b.X+bt.W) && (yy == b.Y-1 || yy == b.Y+bt.H) {
				continue
			}
			if xx >= b.X && xx < b.X+bt.W && yy >= b.Y && yy < b.Y+bt.H {
				continue
			}
			if w.In(xx, yy) {
				i := w.Idx(xx, yy)
				if w.Terr[i] != TWater && w.Occ[i] == 0 {
					prev[i] = -1
					q = append(q, i)
				}
			}
		}
	}
	for h := 0; h < len(q) && h < 30000; h++ {
		i := q[h]
		if isGoal(i) {
			for k := i; k != -1; k = prev[k] {
				w.Road[k] = true
				w.Deco[k] = 0
			}
			g.dirty = true
			g.Recompute()
			return
		}
		x, y := i%n, i/n
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !w.In(nx, ny) {
				continue
			}
			j := w.Idx(nx, ny)
			if _, ok := prev[j]; ok {
				continue
			}
			if w.Terr[j] != TWater && w.Occ[j] == 0 && (w.Infl[j] || w.Road[j]) {
				prev[j] = i
				q = append(q, j)
			}
		}
	}
}
