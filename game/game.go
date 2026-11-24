package game

import (
	"time"
	"math"
	"math/rand"
	"strconv"
)

func itoa(i int) string { return strconv.Itoa(i) }

type Building struct {
	ID     int
	T      int
	X, Y   int
	Isl    int
	Var    int
	Prog   float64
	Prod   float64 // Produktivität 0..1 (gleitend)
	Status int
	Paused bool
	Conn   bool // Straßenanbindung an Lager
	// Wohnhaus
	Tier    int
	Res     float64
	Fulfil  []float64
	Cover   [NumBTypes]bool
	UpTimer float64
	Lock    bool
	// Werft
	ShipQ    float64
	Building bool
	Dock     [2]int
	Anim     float64
	BaseY    float64
	Own      int     // 0 = Spieler, sonst KI-Gegner+1
	Constr   float64 `json:",omitempty"` // verbleibender Bauanteil (0 = fertig)
	Fire     float64 `json:",omitempty"` // brennt seit … Sekunden
	Aim      float64 `json:",omitempty"`
	ShipWar  bool    `json:",omitempty"`
	Cool     float64 `json:"-"`
}

const (
	StOK = iota
	StNoRoad
	StNoInput
	StFull
	StNoEnergy
	StPaused
	StBuilding
	StFire
)

var StatusText = []string{"Arbeitet", "Keine Straßenanbindung", "Rohstoffe fehlen", "Lager voll", "Energiemangel", "Pausiert", "Im Bau", "Brennt!"}

type Cargo struct {
	G   int
	Amt int
}

type Route struct {
	A, B     int // Kontor-Gebäude-IDs
	GAB, GBA int
	Active   bool
}

type Ship struct {
	ID     int
	Name   string
	X, Y   float64
	Ang    float64
	Path   [][2]int
	Cargo  [4]Cargo
	Kind   int // 0 Spieler, 1 Händler
	Route  Route
	Phase  int // 0 -> A, 1 -> B
	Wait   float64
	Target int     // Kontor-ID für Händler
	War    bool    `json:",omitempty"` // Kriegsschiff
	HP     float64 `json:",omitempty"`
	MaxHP  float64 `json:",omitempty"`
	AtkS   int     `json:",omitempty"` // Angriffsziel (Schiffs-ID)
	Life   float64 `json:",omitempty"` // Piraten: Verweildauer
	Cool   float64 `json:"-"`
	Rep    float64 `json:"-"`
	Dead   bool    `json:"-"`
}

type Cart struct {
	Path [][2]int
	Pos  float64
	G    int
}

type Notif struct {
	Text string
	T    float64
	Col  RGB
}

type Game struct {
	W          *World
	Islands    []*Island
	B          []*Building
	Ships      []*Ship
	Carts      []*Cart
	NextID     int
	Credits    float64
	Balance    float64
	Time       float64
	Speed      int
	Pop        [NumTiers]int
	MaxPop     [NumTiers]int
	Quest      int
	Notifs     []Notif
	Won        bool
	Hour       float64
	Day        int
	Mode       int
	Chapter    int
	chapterT   float64
	Dialogs    []Dialog
	Banner     float64
	Storm      float64
	StormOn    bool
	stormT     float64
	Flash      float64
	saveT      float64
	Seed       int
	rng        *rand.Rand
	dirty      bool
	unlockSeen [NumBTypes]bool
	Rivals     []*Rival
	Events     []GEvent `json:"-"`
	PirT       float64
	BP         []*Building
	Hist       []HistPoint
	histT      float64
	ignoreCost bool
	thumbReq   string
	lastSlot   string
	Diff       int // 0 Entspannt, 1 Normal, 2 Profi
	tut        *tutState
	undo       []UndoAct
	Tech       [NumTechs]bool
	RP         float64
	Buff       float64
	BuffMul    float64
	TaxBuff    float64
	EvT        float64
	Event0     *GameEvent `json:"-"`
	Stats      GameStats
	achT       float64
	noUndo     bool
	Map        MapOptions
	FireT      float64
	RB         []*Building // Gebäude der KI-Gegner
	Reg        int         // 0 Erde, 1 Mond
	Other      *Region     // die gerade nicht angezeigte Region
	RocketT    float64
	Launch     float64    `json:"-"`
	LaunchPos  [2]float64 `json:"-"`
	LaunchY    float64    `json:"-"`

	// Darstellung / Eingabe
	UI *UIState
}

func NewGame(seed int) *Game {
	g := &Game{Seed: seed, Speed: 1, Credits: 15000, NextID: 1, Hour: 9, stormT: 400, Diff: 1, Map: MapOpt}
	g.rng = rand.New(rand.NewSource(int64(seed)))
	g.W, g.Islands = GenWorld(seed)
	g.UI = NewUIState()
	// Startschiff vor der Startinsel
	sx, sy := g.startSpot()
	sh := &Ship{ID: g.newID(), Name: "Aurora I", X: float64(sx) + 0.5, Y: float64(sy) + 0.5}
	sh.Cargo[0] = Cargo{GBaustoffe, 50}
	sh.Cargo[1] = Cargo{GBaustoffe, 20}
	sh.Cargo[2] = Cargo{GNahrung, 20}
	sh.Cargo[3] = Cargo{GWasser, 20}
	sh.Route = Route{GAB: -1, GBA: -1}
	g.Ships = append(g.Ships, sh)
	// Händler
	tr := &Ship{ID: g.newID(), Name: "Orion-Händler", X: 10, Y: 10, Kind: 1, Target: -1}
	g.Ships = append(g.Ships, tr)
	for i := range g.unlockSeen {
		g.unlockSeen[i] = g.Unlocked(i)
	}
	g.UI.CenterOn(g, sh.X, sh.Y)
	g.UI.SelShip = sh.ID
	g.Notify("Willkommen, Gouverneur! Steuere deinen Frachtgleiter an die Küste von Neu-Aurora und gründe einen Kontor.", hex(0xf0d28a))
	return g
}

// Startposition: Wasser vor einer breiten, flachen Küste der Startinsel
func (g *Game) startSpot() (int, int) {
	w := g.W
	d := islandDefs[0]
	best, bs := [2]int{5, 5}, -1
	for y := int(d.y) - 45; y < int(d.y)+45; y++ {
		for x := int(d.x) - 45; x < int(d.x)+45; x++ {
			if !w.In(x, y) || w.Terr[w.Idx(x, y)] != TWater || w.Dist[w.Idx(x, y)] != 3 {
				continue
			}
			if (x+y)%2 == 1 {
				continue
			}
			score := 0
			for yy := y - 12; yy <= y+12; yy++ {
				for xx := x - 12; xx <= x+12; xx++ {
					if w.In(xx, yy) {
						i := w.Idx(xx, yy)
						if w.Isl[i] == 0 && (w.Terr[i] == TGrass || w.Terr[i] == TSand) {
							score++
						}
					}
				}
			}
			if score > bs {
				bs = score
				best = [2]int{x, y}
			}
		}
	}
	return best[0], best[1]
}

func (g *Game) allBuildings() []*Building {
	if len(g.RB) == 0 {
		return g.B
	}
	out := make([]*Building, 0, len(g.B)+len(g.RB))
	return append(append(out, g.B...), g.RB...)
}

func (g *Game) newID() int { g.NextID++; return g.NextID }

func (g *Game) findWaterNear(fx, fy float64) (int, int) {
	best, bd := [2]int{5, 5}, 1e9
	for y := 0; y < g.W.N; y++ {
		for x := 0; x < g.W.N; x++ {
			i := g.W.Idx(x, y)
			if g.W.Terr[i] == TWater && g.W.Dist[i] >= 2 {
				d := math.Hypot(float64(x)-fx, float64(y)-fy)
				if d < bd {
					bd = d
					best = [2]int{x, y}
				}
			}
		}
	}
	return best[0], best[1]
}

func (g *Game) Notify(s string, c RGB) {
	if g.UI != nil && g.UI.Screen == ScrGame {
		PlaySfx(SfxNotify)
	}
	g.Notifs = append(g.Notifs, Notif{s, 7, c})
	if len(g.Notifs) > 4 {
		g.Notifs = g.Notifs[1:]
	}
}

func (g *Game) Bld(id int) *Building {
	for _, b := range g.B {
		if b.ID == id {
			return b
		}
	}
	return nil
}

func (g *Game) ShipByID(id int) *Ship {
	for _, s := range g.Ships {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (g *Game) BuildingAt(x, y int) *Building {
	if !g.W.In(x, y) {
		return nil
	}
	o := g.W.Occ[g.W.Idx(x, y)]
	if o == 0 {
		return nil
	}
	return g.Bld(int(o) - 1)
}

func (b *Building) Center() (float64, float64) {
	t := &BTypes[b.T]
	return float64(b.X) + float64(t.W)/2, float64(b.Y) + float64(t.H)/2
}

func (b *Building) MaxRes() int { return Tiers[b.Tier].MaxRes }

// ------------------------------------------------------------ Platzierung

type PlaceCheck struct {
	OK     bool
	Reason string
	Ship   *Ship // Material vom Schiff
}

func (g *Game) islandAtFoot(x, y, w, h int) int {
	isl := -2
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if !g.W.In(xx, yy) {
				return -1
			}
			i := int(g.W.Isl[g.W.Idx(xx, yy)])
			if isl == -2 {
				isl = i
			} else if isl != i {
				return -1
			}
		}
	}
	return isl
}

func (g *Game) coastal(x, y, w, h int) (bool, [2]int) {
	for yy := y - 1; yy <= y+h; yy++ {
		for xx := x - 1; xx <= x+w; xx++ {
			if xx >= x && xx < x+w && yy >= y && yy < y+h {
				continue
			}
			if (xx == x-1 || xx == x+w) && (yy == y-1 || yy == y+h) {
				continue
			}
			if g.W.In(xx, yy) && g.W.Terr[g.W.Idx(xx, yy)] == TWater {
				return true, [2]int{xx, yy}
			}
		}
	}
	return false, [2]int{}
}

func (g *Game) CanPlace(t, x, y int) PlaceCheck {
	bt := &BTypes[t]
	w := g.W
	if !g.Unlocked(t) {
		return PlaceCheck{Reason: g.UnlockText(t)}
	}
	isl := g.islandAtFoot(x, y, bt.W, bt.H)
	if isl < 0 {
		return PlaceCheck{Reason: "Nur auf einer Insel baubar"}
	}
	hasDep := false
	for yy := y; yy < y+bt.H; yy++ {
		for xx := x; xx < x+bt.W; xx++ {
			i := w.Idx(xx, yy)
			if w.Terr[i] == TWater {
				return PlaceCheck{Reason: "Nicht auf Wasser"}
			}
			if w.Occ[i] != 0 || w.Road[i] {
				return PlaceCheck{Reason: "Fläche ist belegt"}
			}
			if w.Terr[i] == TRock && bt.Deposit == DepNone && t != BRoad {
				return PlaceCheck{Reason: "Gebirge ist nicht bebaubar"}
			}
			if bt.Deposit != DepNone && int(w.Dep[i]) == bt.Deposit {
				hasDep = true
			}
			if bt.Deposit != DepNone && w.Terr[i] != TRock {
				return PlaceCheck{Reason: "Muss auf einem Vorkommen im Gebirge stehen"}
			}
		}
	}
	if bt.Deposit != DepNone && !hasDep {
		return PlaceCheck{Reason: "Kein passendes Vorkommen"}
	}
	I := g.Islands[isl]
	if why := g.moonRule(t); why != "" {
		return PlaceCheck{Reason: why}
	}
	if I.Rival > 0 {
		return PlaceCheck{Reason: T("Diese Insel gehört ") + T(RivalDefs[I.Rival-1].Name)}
	}
	if bt.Fert != FertNone && !I.Fert[bt.Fert] {
		return PlaceCheck{Reason: "Insel hat keine passende Fruchtbarkeit"}
	}
	if bt.Coastal {
		if ok, _ := g.coastal(x, y, bt.W, bt.H); !ok {
			return PlaceCheck{Reason: "Muss an der Küste stehen"}
		}
	}
	var ship *Ship
	if t == BKontor && !I.Owned {
		// Schiff in der Nähe mit Material?
		cx, cy := float64(x)+1.5, float64(y)+1.5
		for _, s := range g.Ships {
			if s.Kind != 0 || math.Hypot(s.X-cx, s.Y-cy) > 12 {
				continue
			}
			ok := true
			for k, m := range bt.Mat {
				if m > 0 && s.Has(MatGoods[k]) < m {
					ok = false
				}
			}
			if ok {
				ship = s
				break
			}
		}
		if ship == nil {
			return PlaceCheck{Reason: "Ein Frachtgleiter mit Baustoffen muss in der Nähe sein"}
		}
	} else {
		if !I.Owned {
			return PlaceCheck{Reason: "Gründe zuerst einen Kontor auf dieser Insel"}
		}
		for yy := y; yy < y+bt.H; yy++ {
			for xx := x; xx < x+bt.W; xx++ {
				if !w.Infl[w.Idx(xx, yy)] {
					return PlaceCheck{Reason: "Außerhalb des Einflussbereichs"}
				}
			}
		}
		for k, m := range bt.Mat {
			if m > 0 && int(I.Stock[MatGoods[k]]) < m && !g.ignoreCost {
				return PlaceCheck{Reason: T("Nicht genug ") + Goods[MatGoods[k]].Name}
			}
		}
	}
	if g.Credits < float64(bt.Cost) && !g.ignoreCost {
		return PlaceCheck{Reason: "Nicht genug Credits"}
	}
	return PlaceCheck{OK: true, Ship: ship}
}

func (s *Ship) Has(good int) int {
	n := 0
	for _, c := range s.Cargo {
		if c.G == good {
			n += c.Amt
		}
	}
	return n
}

func (s *Ship) Take(good, n int) int {
	got := 0
	for i := range s.Cargo {
		if s.Cargo[i].G == good && got < n {
			k := imin(n-got, s.Cargo[i].Amt)
			s.Cargo[i].Amt -= k
			got += k
			if s.Cargo[i].Amt == 0 {
				s.Cargo[i].G = -1
			}
		}
	}
	return got
}

// Lädt bis zu n Einheiten; gibt Menge zurück
func (s *Ship) Put(good, n int) int {
	put := 0
	for i := range s.Cargo {
		if s.Cargo[i].G == good && s.Cargo[i].Amt < 50 {
			k := imin(n-put, 50-s.Cargo[i].Amt)
			s.Cargo[i].Amt += k
			put += k
		}
	}
	for i := range s.Cargo {
		if put >= n {
			break
		}
		if s.Cargo[i].G == -1 || s.Cargo[i].Amt == 0 {
			k := imin(n-put, 50)
			s.Cargo[i] = Cargo{good, k}
			put += k
		}
	}
	return put
}

func (g *Game) Place(t, x, y int) *Building {
	pc := g.CanPlace(t, x, y)
	if !pc.OK {
		return nil
	}
	return g.forcePlace(t, x, y, pc.Ship)
}

func (g *Game) forcePlace(t, x, y int, ship *Ship) *Building {
	bt := &BTypes[t]
	w := g.W
	isl := int(w.Isl[w.Idx(x, y)])
	I := g.Islands[isl]
	g.Credits -= float64(bt.Cost)
	for k, m := range bt.Mat {
		if m == 0 {
			continue
		}
		if ship != nil {
			ship.Take(MatGoods[k], m)
		} else {
			I.Stock[MatGoods[k]] -= float64(m)
		}
	}
	if t != BKontor && !g.noUndo {
		g.pushUndo(UndoAct{Kind: 0, T: t, X: x, Y: y, Isl: isl})
	}
	if t == BRoad {
		i := w.Idx(x, y)
		w.Road[i] = true
		w.Deco[i] = 0
		g.dirty = true
		return nil
	}
	b := &Building{ID: g.newID(), T: t, X: x, Y: y, Isl: isl, Var: int(hash2(x, y, 77) * 12), Prod: 0, Constr: 1}
	b.BaseY = w.Flatten(x, y, bt.W, bt.H)
	for yy := y; yy < y+bt.H; yy++ {
		for xx := x; xx < x+bt.W; xx++ {
			i := w.Idx(xx, yy)
			w.Occ[i] = int32(b.ID + 1)
			w.Deco[i] = 0
		}
	}
	if t == BHouse {
		b.Res = 1
	}
	if bt.Coastal {
		_, d := g.coastal(x, y, bt.W, bt.H)
		b.Dock = d
	}
	if t == BKontor && !I.Owned {
		I.Owned = true
		if !g.OnMoon() {
			g.onPlayerColony(isl)
		}
		I.Cap = 50
		g.Notify(T("Kolonie auf ")+T(I.Name)+T(" gegründet!"), hex(0x96d678))
	}
	g.B = append(g.B, b)
	g.dirty = true
	return b
}

func (g *Game) Demolish(x, y int) bool {
	w := g.W
	if !w.In(x, y) {
		return false
	}
	i := w.Idx(x, y)
	if bp := g.BlueAt(x, y); bp != nil {
		g.removeBlueprint(bp)
		return true
	}
	if w.Road[i] {
		if w.Isl[i] >= 0 && g.Islands[w.Isl[i]].Rival > 0 {
			return false
		}
		w.Road[i] = false
		g.dirty = true
		g.pushUndo(UndoAct{Kind: 1, T: BRoad, X: x, Y: y, Isl: int(w.Isl[i])})
		return true
	}
	b := g.BuildingAt(x, y)
	if b == nil {
		return false
	}
	if b.T == BKontor {
		g.Notify("Ein Kontor kann nicht abgerissen werden.", hex(0xff8080))
		return false
	}
	bt := &BTypes[b.T]
	I := g.Islands[b.Isl]
	ua := UndoAct{Kind: 1, T: b.T, X: b.X, Y: b.Y, Isl: b.Isl, Tier: b.Tier}
	for k, m := range bt.Mat {
		before := I.Stock[MatGoods[k]]
		I.Stock[MatGoods[k]] = math.Min(float64(I.Cap), I.Stock[MatGoods[k]]+float64(m/2))
		ua.Ref[k] = I.Stock[MatGoods[k]] - before
	}
	g.pushUndo(ua)
	for yy := b.Y; yy < b.Y+bt.H; yy++ {
		for xx := b.X; xx < b.X+bt.W; xx++ {
			w.Occ[w.Idx(xx, yy)] = 0
		}
	}
	for k, o := range g.B {
		if o == b {
			g.B = append(g.B[:k], g.B[k+1:]...)
			break
		}
	}
	g.dirty = true
	return true
}

// ------------------------------------------------------------ Neuberechnung

func (g *Game) Recompute() {
	w := g.W
	n := w.N
	// Einfluss
	for i := range w.Infl {
		w.Infl[i] = false
	}
	for _, I := range g.Islands {
		I.Cap = 0
	}
	for _, b := range g.B {
		bt := &BTypes[b.T]
		if b.T == BKontor || b.T == BLager {
			I := g.Islands[b.Isl]
			if b.T == BKontor {
				I.Cap += 50
			} else {
				I.Cap += 30
			}
			if g.HasTech(TechLager) {
				I.Cap += 25
			}
			cx, cy := b.Center()
			r := float64(bt.Radius)
			for y := int(cy - r); y <= int(cy+r); y++ {
				for x := int(cx - r); x <= int(cx+r); x++ {
					if !w.In(x, y) {
						continue
					}
					if math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) <= r && int(w.Isl[w.Idx(x, y)]) == b.Isl {
						w.Infl[w.Idx(x, y)] = true
					}
				}
			}
		}
	}
	// Straßennetze
	for i := range w.RoadC {
		w.RoadC[i] = 0
	}
	comp := int32(0)
	for i := 0; i < n*n; i++ {
		if !w.Road[i] || w.RoadC[i] != 0 {
			continue
		}
		comp++
		q := []int{i}
		w.RoadC[i] = comp
		for h := 0; h < len(q); h++ {
			x, y := q[h]%n, q[h]/n
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if w.In(nx, ny) {
					j := w.Idx(nx, ny)
					if w.Road[j] && w.RoadC[j] == 0 {
						w.RoadC[j] = comp
						q = append(q, j)
					}
				}
			}
		}
	}
	// Netze mit Lager
	storeComp := map[int32]bool{}
	for _, b := range g.B {
		if b.T == BKontor || b.T == BLager {
			for _, c := range g.adjComps(b) {
				storeComp[c] = true
			}
		}
	}
	for _, b := range g.B {
		if b.T == BKontor || b.T == BLager || b.T == BWind || b.T == BPark || b.T == BSolar || b.T == BFusion {
			b.Conn = true
			continue
		}
		b.Conn = false
		for _, c := range g.adjComps(b) {
			if storeComp[c] {
				b.Conn = true
			}
		}
	}
	// Dienstleistungs-Abdeckung
	for _, h := range g.B {
		if h.T != BHouse {
			continue
		}
		h.Cover = [NumBTypes]bool{}
		hx, hy := h.Center()
		for _, s := range g.B {
			if (s.T == BKomm || s.T == BMedi || s.T == BLabor) && s.Isl == h.Isl && s.Conn && !s.Paused {
				sx, sy := s.Center()
				if math.Hypot(hx-sx, hy-sy) <= float64(BTypes[s.T].Radius) {
					h.Cover[s.T] = true
				}
			}
		}
	}
	g.dirty = false
}

func (g *Game) adjComps(b *Building) []int32 {
	bt := &BTypes[b.T]
	var out []int32
	w := g.W
	for yy := b.Y - 1; yy <= b.Y+bt.H; yy++ {
		for xx := b.X - 1; xx <= b.X+bt.W; xx++ {
			if (xx == b.X-1 || xx == b.X+bt.W) && (yy == b.Y-1 || yy == b.Y+bt.H) {
				continue
			}
			if w.In(xx, yy) {
				c := w.RoadC[w.Idx(xx, yy)]
				if c != 0 {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

// ------------------------------------------------------------ Simulation

func (g *Game) Update(dt float64) {
	g.UI.Anim += dt
	if g.Banner > 0 && !g.DialogOpen() {
		g.Banner -= dt
	}
	for i := range g.Notifs {
		g.Notifs[i].T -= dt
	}
	for len(g.Notifs) > 0 && g.Notifs[0].T <= 0 {
		g.Notifs = g.Notifs[1:]
	}
	if g.dirty {
		g.Recompute()
	}
	if g.UI.Screen == ScrMenu {
		g.Hour = math.Mod(g.Hour+dt*24/(4*60), 24)
		return
	}
	if g.Speed == 0 || g.UI.Screen != ScrGame || g.UI.PauseMenu || g.DialogOpen() {
		return
	}
	sdt := dt * float64(g.Speed)
	// Unterteilen für Stabilität
	for sdt > 0 {
		step := math.Min(sdt, 0.25)
		g.step(step)
		sdt -= step
	}
}

func (g *Game) weather(dt float64) {
	if Opt.DayNight {
		g.Hour += dt * 24 / (float64(imax(1, Opt.DayLength)) * 60)
		if g.Hour >= 24 {
			g.Hour -= 24
			g.Day++
		}
	} else {
		g.Hour = Opt.FixedHour
	}
	if Opt.Weather || g.forceStorm() {
		g.stormT -= dt
		if g.stormT <= 0 {
			g.StormOn = !g.StormOn
			if g.StormOn {
				g.stormT = 90 + g.rng.Float64()*120
				g.Notify("Ein Sturm zieht auf – Windturbinen liefern mehr Energie!", hex(0x9fc8ff))
			} else {
				g.stormT = 420 + g.rng.Float64()*600
			}
		}
	} else {
		g.StormOn = false
	}
	if g.forceStorm() {
		g.StormOn = true
	}
	tgt := 0.0
	if g.StormOn {
		tgt = 1
	}
	g.Storm += (tgt - g.Storm) * math.Min(1, dt/18)
	g.Flash = math.Max(0, g.Flash-dt*3.5)
	if g.Storm > 0.6 && g.rng.Float64() < dt/9 {
		g.Flash = 1
		PlaySfx(SfxThunder)
	}
}

func (g *Game) forceStorm() bool { return g.Mode == ModeCampaign && g.Chapter == 3 && g.chapterT < 240 }

func (g *Game) step(dt float64) {
	g.Time += dt
	g.weather(dt)
	g.recordHistory()
	g.updateEvents(dt)
	g.stepRegion(dt)
	pop, bal := g.Pop, g.Balance
	if g.Other != nil {
		g.swapRegion()
		g.stepRegion(dt)
		for t := range pop {
			pop[t] += g.Pop[t]
		}
		bal += g.Balance
		g.swapRegion()
	}
	g.Pop, g.Balance = pop, bal
	g.updateRockets(dt)
	for t := 0; t < NumTiers; t++ {
		if g.Pop[t] > g.MaxPop[t] {
			g.MaxPop[t] = g.Pop[t]
		}
	}
	for t := 0; t < NumBTypes; t++ {
		if !g.unlockSeen[t] && g.Unlocked(t) {
			g.unlockSeen[t] = true
			g.Notify(T("Freigeschaltet: ")+T(BTypes[t].Name), hex(0xffd24a))
		}
	}
	onMoon := g.Reg == 1
	if onMoon {
		g.swapRegion()
	}
	switch g.Mode {
	case ModeCampaign:
		g.checkCampaign(dt)
	case ModeTutorial:
		g.updateTutorial(dt)
	default:
		g.checkQuests()
	}
	g.checkAchievements(dt)
	if onMoon {
		g.swapRegion()
	}
	if Opt.AutoSave > 0 && g.UI.Screen == ScrGame {
		g.saveT += dt / float64(imax(1, g.Speed))
		if g.saveT > float64(Opt.AutoSave)*60 {
			g.saveT = 0
			if g.Save() == nil {
				g.Notify("Automatisch gespeichert.", cDim)
			}
		}
	}
}

// Simulation der gerade eingesetzten Region
func (g *Game) stepRegion(dt float64) {
	if !g.OnMoon() {
		g.updateRivals(dt)
		g.updateCombat(dt)
		g.updatePirates(dt)
	}
	g.updateDisasters(dt)
	if g.dirty {
		g.Recompute()
	}
	// Energie
	for _, I := range g.Islands {
		I.EProd, I.EUse = 0, 0
		I.Pop = [NumTiers]int{}
	}
	upkeep := 0.0
	for _, b := range g.B {
		bt := &BTypes[b.T]
		I := g.Islands[b.Isl]
		u := float64(bt.Upkeep) * g.dm(0.75, 1, 1.25)
		if g.HasTech(TechSozial) {
			u *= 0.9
		}
		if b.Paused {
			u *= 0.5
		} else {
			if bt.Energy > 0 {
				e := float64(bt.Energy)
				if b.T == BWind {
					e *= 1 + 0.5*g.Storm
				}
				if g.HasTech(TechEnergie) {
					e *= 1.15
				}
				if b.T == BFusion && I.Stock[GHelium3] > 0 {
					e *= 1.6
					I.Stock[GHelium3] = math.Max(0, I.Stock[GHelium3]-dt/30)
				}
				I.EProd += int(e)
			} else {
				I.EUse -= bt.Energy
			}
		}
		upkeep += u
		if b.T == BHouse {
			I.Pop[b.Tier] += int(b.Res)
		}
	}
	income := 0.0
	// Produktion
	for _, b := range g.B {
		bt := &BTypes[b.T]
		b.Anim += dt
		if b.Constr > 0 {
			b.Constr -= dt / BuildTime(b.T)
			if b.Constr <= 0 {
				b.Constr = 0
				g.Event(GEvent{Kind: EvBuilt, X: float64(b.X) + float64(bt.W)/2, Y: b.BaseY, Z: float64(b.Y) + float64(bt.H)/2, S: float64(bt.W)})
			} else if bt.Out >= 0 {
				b.Status = StBuilding
				b.Prod = 0
				continue
			}
		}
		if bt.Out < 0 {
			if b.T == BWerft && b.Building {
				b.ShipQ += dt / 40
				if b.ShipQ >= 1 {
					b.Building = false
					b.ShipQ = 0
					if b.ShipWar {
						b.ShipWar = false
						g.spawnWarship(b)
					} else {
						g.spawnShip(b)
					}
				}
			}
			continue
		}
		I := g.Islands[b.Isl]
		ef := 1.0
		if I.EUse > 0 && I.EProd < I.EUse {
			ef = float64(I.EProd) / float64(I.EUse)
		}
		if I.Blackout > 0 && bt.Energy < 0 {
			ef = 0
		}
		working := false
		switch {
		case b.Fire > 0:
			b.Status = StFire
		case b.Paused:
			b.Status = StPaused
		case !b.Conn:
			b.Status = StNoRoad
		default:
			if b.Prog == 0 {
				if I.Stock[bt.Out] >= float64(I.Cap) {
					b.Status = StFull
					break
				}
				ok := true
				for _, in := range bt.In {
					if I.Stock[in] < 1 {
						ok = false
					}
				}
				if !ok {
					b.Status = StNoInput
					break
				}
				for _, in := range bt.In {
					I.Stock[in]--
					I.Consumed[in] += 1
				}
				b.Prog = 0.0001
			}
			f := 1.0
			if bt.Energy < 0 {
				f = ef
			}
			if f < 0.999 {
				b.Status = StNoEnergy
			} else {
				b.Status = StOK
			}
			b.Prog += dt / bt.Cycle * f * g.prodMul()
			working = f > 0.01
			if b.Prog >= 1 {
				b.Prog = 0
				I.Stock[bt.Out] = math.Min(float64(I.Cap), I.Stock[bt.Out]+1)
				I.Produced[bt.Out] += 1
				g.spawnCart(b)
			}
		}
		target := 0.0
		if working {
			target = 1
			if b.Status == StNoEnergy {
				target = ef
			}
		}
		b.Prod += (target - b.Prod) * math.Min(1, dt/15)
	}
	// Bedarf der Bewohner
	var demand [32][NumGoods]float64
	for _, b := range g.B {
		if b.T != BHouse || !b.Conn {
			continue
		}
		for _, nd := range Tiers[b.Tier].Needs {
			if nd.Good >= 0 {
				demand[b.Isl][nd.Good] += b.Res * nd.Rate * dt / 60
			}
		}
	}
	for ii, I := range g.Islands {
		if !I.Owned {
			continue
		}
		for gd := 0; gd < NumGoods; gd++ {
			d := demand[ii][gd]
			if d <= 0 {
				I.Sat[gd] += (1 - I.Sat[gd]) * math.Min(1, dt/10)
				continue
			}
			I.DemAcc[gd] += d
			for I.DemAcc[gd] >= 1 && I.Stock[gd] >= 1 {
				I.Stock[gd]--
				I.DemAcc[gd]--
				I.Consumed[gd] += 1
			}
			tgt := 0.0
			if I.Stock[gd] >= 1 || I.DemAcc[gd] < 1 {
				tgt = 1
			}
			if I.DemAcc[gd] > 3 {
				I.DemAcc[gd] = 3
			}
			I.Sat[gd] += (tgt - I.Sat[gd]) * math.Min(1, dt/12)
		}
		// Gleitende Raten (pro Minute)
		for gd := 0; gd < NumGoods; gd++ {
			_ = gd
		}
	}
	// Häuser
	for _, b := range g.B {
		if b.T != BHouse {
			continue
		}
		I := g.Islands[b.Isl]
		tier := &Tiers[b.Tier]
		if len(b.Fulfil) != len(tier.Needs) {
			b.Fulfil = make([]float64, len(tier.Needs))
		}
		sum := 0.0
		allMet := true
		for k, nd := range tier.Needs {
			f := 0.0
			if b.Conn {
				if nd.Good >= 0 {
					f = I.Sat[nd.Good]
				} else if b.Cover[nd.Service] {
					f = 1
				}
			}
			b.Fulfil[k] = f
			sum += f
			if f < 0.9 {
				allMet = false
			}
		}
		avg := sum / float64(len(tier.Needs))
		tgt := float64(tier.MaxRes) * (0.3 + 0.7*avg)
		if !b.Conn {
			tgt = 1
		}
		rate := dt / 3
		if b.Res < tgt {
			b.Res = math.Min(tgt, b.Res+rate)
		} else {
			b.Res = math.Max(tgt, b.Res-rate*0.5)
		}
		income += b.Res * tier.Tax * (0.5 + 0.5*avg) * g.taxMul()
		// Aufstieg
		if allMet && b.Tier+1 < NumTiers && !b.Lock && b.Res >= float64(tier.MaxRes)-0.5 {
			next := &Tiers[b.Tier+1]
			if g.MaxPop[b.Tier] >= next.UnlockPop {
				b.UpTimer += dt
				if b.UpTimer > 8+float64(b.Var) {
					ok := true
					for k, m := range next.UpCost {
						if m > 0 && I.Stock[MatGoods[k]] < float64(m) {
							ok = false
						}
					}
					if ok {
						for k, m := range next.UpCost {
							I.Stock[MatGoods[k]] -= float64(m)
						}
						b.Tier++
						b.Constr = 0.45
						b.UpTimer = 0
						b.Fulfil = nil
						PlaySfx(SfxUpgrade)
						if g.Pop[b.Tier] == 0 && g.MaxPop[b.Tier] == 0 {
							g.Notify(T("Die ersten ")+T(next.Plural)+T(" sind eingezogen!"), next.Col)
						}
					}
				}
			} else {
				b.UpTimer = 0
			}
		} else {
			b.UpTimer = 0
		}
	}
	g.Balance = income - upkeep
	if !g.OnMoon() {
		g.Balance += g.treatyIncome()
	}
	g.Credits += g.Balance * dt / 60
	// Bevölkerung
	g.Pop = [NumTiers]int{}
	for _, I := range g.Islands {
		I.Pop = [NumTiers]int{}
	}
	for _, b := range g.B {
		if b.T == BHouse {
			g.Pop[b.Tier] += int(b.Res + 0.001)
			g.Islands[b.Isl].Pop[b.Tier] += int(b.Res + 0.001)
		}
	}
	g.updateShips(dt)
	g.updateCarts(dt)
}

// ------------------------------------------------------------ Gleiter (Warentransport)

func (g *Game) spawnCart(b *Building) {
	if len(g.Carts) > 150 {
		return
	}
	// Straßenkachel neben dem Gebäude
	w := g.W
	bt := &BTypes[b.T]
	var start [2]int
	found := false
	for yy := b.Y - 1; yy <= b.Y+bt.H && !found; yy++ {
		for xx := b.X - 1; xx <= b.X+bt.W; xx++ {
			if w.In(xx, yy) && w.Road[w.Idx(xx, yy)] {
				start = [2]int{xx, yy}
				found = true
				break
			}
		}
	}
	if !found {
		return
	}
	// BFS zu Straße neben Lager
	n := w.N
	prev := map[int]int{}
	si := w.Idx(start[0], start[1])
	prev[si] = -1
	q := []int{si}
	goal := -1
	for h := 0; h < len(q) && h < 4000; h++ {
		i := q[h]
		x, y := i%n, i/n
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			ob := g.BuildingAt(x+d[0], y+d[1])
			if ob != nil && (ob.T == BKontor || ob.T == BLager) && ob.Isl == b.Isl {
				goal = i
				break
			}
		}
		if goal >= 0 {
			break
		}
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if w.In(nx, ny) {
				j := w.Idx(nx, ny)
				if _, ok := prev[j]; !ok && w.Road[j] {
					prev[j] = i
					q = append(q, j)
				}
			}
		}
	}
	if goal < 0 {
		return
	}
	var path [][2]int
	for i := goal; i != -1; i = prev[i] {
		path = append([][2]int{{i % n, i / n}}, path...)
	}
	g.Carts = append(g.Carts, &Cart{Path: path, G: bt.Out})
}

func (g *Game) updateCarts(dt float64) {
	out := g.Carts[:0]
	for _, c := range g.Carts {
		c.Pos += dt * 2.2
		if int(c.Pos) < len(c.Path)-1 {
			out = append(out, c)
		}
	}
	g.Carts = out
}

func (c *Cart) WorldPos() (float64, float64, float64) {
	i := int(c.Pos)
	f := c.Pos - float64(i)
	a := c.Path[i]
	b := c.Path[imin(i+1, len(c.Path)-1)]
	x := float64(a[0]) + float64(b[0]-a[0])*f + 0.5
	y := float64(a[1]) + float64(b[1]-a[1])*f + 0.5
	ang := math.Atan2(float64(b[1]-a[1]), float64(b[0]-a[0]))
	return x, y, ang
}

// ------------------------------------------------------------ Schiffe

func (g *Game) spawnShip(b *Building) {
	names := []string{"Aurora", "Helios", "Nova", "Polaris", "Vega", "Lyra", "Orion", "Kepler"}
	n := 0
	for _, s := range g.Ships {
		if s.Kind == 0 {
			n++
		}
	}
	sh := &Ship{ID: g.newID(), Name: names[n%len(names)] + " " + roman(n/len(names)+1), X: float64(b.Dock[0]) + 0.5, Y: float64(b.Dock[1]) + 0.5}
	if n%len(names) == 0 {
		sh.Name = names[0] + " " + roman(n/len(names)+2)
	}
	for i := range sh.Cargo {
		sh.Cargo[i].G = -1
	}
	sh.Route = Route{GAB: -1, GBA: -1}
	g.Ships = append(g.Ships, sh)
	g.Notify(T("Neuer Frachtgleiter „")+T(sh.Name)+T("“ ist vom Stapel gelaufen!"), hex(0x96d678))
}

func roman(n int) string {
	r := []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"}
	if n-1 < len(r) && n > 0 {
		return r[n-1]
	}
	return itoa(n)
}

// A* auf Wasserkacheln
func (g *Game) WaterPath(sx, sy, tx, ty int) [][2]int {
	w := g.W
	n := w.N
	if !w.In(tx, ty) || w.Terr[w.Idx(tx, ty)] != TWater {
		return nil
	}
	type node struct {
		i int
		f float64
	}
	gs := make([]float32, n*n)
	for i := range gs {
		gs[i] = 1e9
	}
	prev := make([]int32, n*n)
	open := []node{}
	push := func(nd node) {
		open = append(open, nd)
		for k := len(open) - 1; k > 0; {
			p := (k - 1) / 2
			if open[p].f <= open[k].f {
				break
			}
			open[p], open[k] = open[k], open[p]
			k = p
		}
	}
	pop := func() node {
		top := open[0]
		last := len(open) - 1
		open[0] = open[last]
		open = open[:last]
		k := 0
		for {
			l, r := 2*k+1, 2*k+2
			m := k
			if l < len(open) && open[l].f < open[m].f {
				m = l
			}
			if r < len(open) && open[r].f < open[m].f {
				m = r
			}
			if m == k {
				break
			}
			open[m], open[k] = open[k], open[m]
			k = m
		}
		return top
	}
	si := w.Idx(sx, sy)
	ti := w.Idx(tx, ty)
	gs[si] = 0
	prev[si] = -1
	push(node{si, 0})
	dirs := [][3]float64{{1, 0, 1}, {-1, 0, 1}, {0, 1, 1}, {0, -1, 1}, {1, 1, 1.414}, {1, -1, 1.414}, {-1, 1, 1.414}, {-1, -1, 1.414}}
	iter := 0
	for len(open) > 0 && iter < 60000 {
		iter++
		cur := pop()
		if cur.i == ti {
			break
		}
		x, y := cur.i%n, cur.i/n
		for _, d := range dirs {
			nx, ny := x+int(d[0]), y+int(d[1])
			if !w.In(nx, ny) {
				continue
			}
			j := w.Idx(nx, ny)
			if w.Terr[j] != TWater {
				continue
			}
			if d[2] > 1 && (w.Terr[w.Idx(x+int(d[0]), y)] != TWater || w.Terr[w.Idx(x, y+int(d[1]))] != TWater) {
				continue
			}
			cost := d[2]
			if w.Dist[j] <= 1 {
				cost += 0.6 // Küste meiden
			}
			ng := gs[cur.i] + float32(cost)
			if ng < gs[j] {
				gs[j] = ng
				prev[j] = int32(cur.i)
				h := math.Hypot(float64(nx-tx), float64(ny-ty))
				push(node{j, float64(ng) + h})
			}
		}
	}
	if gs[ti] >= 1e9 {
		return nil
	}
	var path [][2]int
	for i := ti; i != si && i >= 0; i = int(prev[i]) {
		path = append(path, [2]int{i % n, i / n})
	}
	for a, b := 0, len(path)-1; a < b; a, b = a+1, b-1 {
		path[a], path[b] = path[b], path[a]
	}
	return path
}

func (g *Game) ShipGoto(s *Ship, tx, ty int) bool {
	sx, sy := int(s.X), int(s.Y)
	if !g.W.In(sx, sy) || g.W.Terr[g.W.Idx(sx, sy)] != TWater {
		sx, sy = g.findWaterNear(s.X, s.Y)
	}
	p := g.WaterPath(sx, sy, tx, ty)
	if p == nil {
		return false
	}
	s.Path = p
	return true
}

func (g *Game) shipNear(s *Ship, b *Building) bool {
	return math.Hypot(s.X-float64(b.Dock[0])-0.5, s.Y-float64(b.Dock[1])-0.5) < 2.5
}

func (g *Game) updateShips(dt float64) {
	if dt <= 0 {
		return
	}
	for _, s := range g.Ships {
		if len(s.Path) > 0 {
			t := s.Path[0]
			tx, ty := float64(t[0])+0.5, float64(t[1])+0.5
			dx, dy := tx-s.X, ty-s.Y
			d := math.Hypot(dx, dy)
			sp := 3.2 * dt * g.shipMul()
			if d <= sp {
				s.X, s.Y = tx, ty
				s.Path = s.Path[1:]
			} else {
				s.X += dx / d * sp
				s.Y += dy / d * sp
			}
			ta := math.Atan2(dy, dx)
			da := math.Remainder(ta-s.Ang, 2*math.Pi)
			s.Ang += da * math.Min(1, dt*5)
			continue
		}
		if s.Wait > 0 {
			s.Wait -= dt
			continue
		}
		if s.War {
			continue
		}
		if s.Kind == 1 {
			g.traderAI(s)
			continue
		}
		if s.Kind >= 2 {
			g.rivalShipAI(s)
			continue
		}
		if s.Route.Active {
			g.routeAI(s)
		}
	}
}

func (g *Game) kontors() []*Building {
	var out []*Building
	for _, b := range g.B {
		if b.T == BKontor {
			out = append(out, b)
		}
	}
	return out
}

func (g *Game) traderAI(s *Ship) {
	ks := g.kontors()
	if s.Target == -1 {
		if len(ks) > 0 && g.rng.Float64() < 0.7 {
			k := ks[g.rng.Intn(len(ks))]
			if g.ShipGoto(s, k.Dock[0], k.Dock[1]) {
				s.Target = k.ID
				return
			}
		}
		// Rand der Karte
		corners := [][2]int{{4, 4}, {g.W.N - 5, 4}, {4, g.W.N - 5}, {g.W.N - 5, g.W.N - 5}}
		c := corners[g.rng.Intn(4)]
		g.ShipGoto(s, c[0], c[1])
		s.Target = -2
		s.Wait = 5
		return
	}
	if s.Target >= 0 {
		s.Wait = 12
	}
	s.Target = -1
}

func (g *Game) routeAI(s *Ship) {
	r := &s.Route
	A, B := g.Bld(r.A), g.Bld(r.B)
	if A == nil || B == nil {
		r.Active = false
		return
	}
	cur, other := A, B
	load, unload := r.GAB, r.GBA
	if s.Phase == 1 {
		cur, other = B, A
		load, unload = r.GBA, r.GAB
	}
	if !g.shipNear(s, cur) {
		if !g.ShipGoto(s, cur.Dock[0], cur.Dock[1]) {
			r.Active = false
			g.Notify(T(s.Name)+T(": Route nicht erreichbar."), hex(0xff8080))
		}
		return
	}
	I := g.Islands[cur.Isl]
	if unload >= 0 {
		n := s.Has(unload)
		space := float64(I.Cap) - I.Stock[unload]
		k := imin(n, int(space))
		s.Take(unload, k)
		I.Stock[unload] += float64(k)
	}
	if load >= 0 {
		avail := int(I.Stock[load])
		k := s.Put(load, imin(avail, 100))
		I.Stock[load] -= float64(k)
	}
	s.Wait = 3
	s.Phase = 1 - s.Phase
	_ = other
}

// Kontor in Reichweite des Schiffs
func (g *Game) ShipKontor(s *Ship) *Building {
	for _, b := range g.B {
		if b.T == BKontor && g.shipNear(s, b) {
			return b
		}
	}
	return nil
}

// ------------------------------------------------------------ Aufträge

type QuestDef struct {
	Title, Desc string
	Check       func(g *Game) (int, int)
	Reward      int
}

func (g *Game) count(t int) int {
	n := 0
	for _, b := range g.B {
		if b.T == t {
			n++
		}
	}
	return n
}

var Quests = []QuestDef{
	{"Neues Land", "Bringe deinen Frachtgleiter an die Küste von Neu-Aurora und baue einen Kontor.", func(g *Game) (int, int) { return g.count(BKontor), 1 }, 2000},
	{"Material entladen", "Wähle deinen Frachtgleiter und entlade die Fracht im Kontor.", func(g *Game) (int, int) {
		for _, I := range g.Islands {
			if I.Owned && I.Stock[GBaustoffe] >= 20 {
				return 1, 1
			}
		}
		return 0, 1
	}, 500},
	{"Erste Siedler", "Baue 8 Wohnmodule und verbinde sie per Magnetstraße mit dem Kontor.", func(g *Game) (int, int) { return g.count(BHouse), 8 }, 1500},
	{"Bausteine der Zukunft", "Baue Schrottsammler, Recyclingwerk und eine Windturbine – ohne Energie stehen die Fabriken still.", func(g *Game) (int, int) {
		return imin(1, g.count(BSchrott)) + imin(1, g.count(BRecycle)) + imin(1, g.count(BWind)), 3
	}, 1500},
	{"Versorgung", "Baue Algenfarm, Nahrungsfabrik und Wasserwerk.", func(g *Game) (int, int) {
		return imin(1, g.count(BAlgen)) + imin(1, g.count(BNahrung)) + imin(1, g.count(BWasser)), 3
	}, 2000},
	{"Vernetzt", "Errichte ein Komm-Zentrum in der Nähe deiner Wohnmodule.", func(g *Game) (int, int) { return g.count(BKomm), 1 }, 2000},
	{"Wachsende Kolonie", "Erreiche 60 Arbeiter – dann können Bewohner zu Technikern aufsteigen.", func(g *Game) (int, int) { return g.MaxPop[TArbeiter], 60 }, 3000},
	{"Die Techniker kommen", "Erfülle alle Bedürfnisse der Arbeiter, damit 40 Techniker einziehen.", func(g *Game) (int, int) { return g.MaxPop[TTechniker], 40 }, 5000},
	{"Expansion", "Gründe mit einem Frachtgleiter einen zweiten Kontor auf einer anderen Insel.", func(g *Game) (int, int) { return g.count(BKontor), 2 }, 4000},
	{"Seehandel", "Richte eine aktive Handelsroute zwischen zwei Kontoren ein.", func(g *Game) (int, int) {
		for _, s := range g.Ships {
			if s.Route.Active {
				return 1, 1
			}
		}
		return 0, 1
	}, 4000},
	{"Hochtechnologie", "Erreiche 120 Techniker – die Ingenieure warten.", func(g *Game) (int, int) { return g.MaxPop[TTechniker], 120 }, 8000},
	{"Ingenieurskunst", "Erreiche 60 Ingenieure.", func(g *Game) (int, int) { return g.MaxPop[TIngenieure], 60 }, 15000},
	{"Metropole der Zukunft", "Erreiche 300 Ingenieure und mache Nova Archipel zur Metropole.", func(g *Game) (int, int) { return g.Pop[TIngenieure], 300 }, 50000},
}

func (g *Game) checkQuests() {
	if g.Quest >= len(Quests) {
		return
	}
	q := &Quests[g.Quest]
	a, b := q.Check(g)
	if a >= b {
		g.Credits += float64(q.Reward)
		PlaySfx(SfxQuest)
		g.Notify(T("Auftrag erfüllt: ")+T(q.Title)+"  (+"+itoa(q.Reward)+T(" Credits)"), hex(0xffd24a))
		g.Quest++
		if g.Quest >= len(Quests) {
			g.Won = true
			g.Notify("Glückwunsch! Nova Archipel ist eine Metropole der Zukunft!", hex(0x96d678))
		}
	}
}

// Bauzeit in Sekunden je nach Größe
func BuildTime(t int) float64 {
	bt := &BTypes[t]
	switch t {
	case BRoad:
		return 0.5
	case BPark, BWind:
		return 4
	}
	return 5 + float64(bt.W*bt.H)*0.9
}

// Ereignisse für Effekte im Renderer (Staub, Explosionen, Laser …)
const (
	EvBuilt = iota
	EvExplode
	EvLaser
	EvSplash
	EvFire
)

type GEvent struct {
	Kind       int
	X, Y, Z    float64
	X2, Y2, Z2 float64
	S          float64
	Col        RGB
}

func (g *Game) Event(e GEvent) {
	if len(g.Events) < 400 {
		g.Events = append(g.Events, e)
	}
}

// Schwierigkeit
var DiffNames = []string{"Entspannt", "Normal", "Profi"}

func (g *Game) dm(easy, normal, hard float64) float64 {
	return []float64{easy, normal, hard}[clampi(g.Diff, 0, 2)]
}

// Neues Endlosspiel mit Einstellungen aus dem Setup-Bildschirm
func NewCustomGame(seed int, mo MapOptions, diff, rivals int) *Game {
	MapOpt = mo
	g := NewGame(seed)
	g.Diff = diff
	g.Map = mo
	g.Credits = g.dm(24000, 15000, 10000)
	g.InitRivals(rivals)
	for _, rv := range g.Rivals {
		rv.Bonus += g.dm(15, 0, -12)
	}
	return g
}

// ---------------------------------------------------------------- Rückgängig (Strg+Z)

type UndoAct struct {
	Kind       int // 0 gebaut, 1 abgerissen
	T, X, Y    int
	Isl, Tier  int
	Ref        [3]float64
	At         float64
}

func (g *Game) pushUndo(a UndoAct) {
	a.At = g.Time
	g.undo = append(g.undo, a)
	if len(g.undo) > 60 {
		g.undo = g.undo[len(g.undo)-60:]
	}
}

// Letzte Aktion zurücknehmen; zusammenhängende Straßen-Züge (innerhalb 1,5 s) gemeinsam
func (g *Game) Undo() bool {
	if len(g.undo) == 0 {
		return false
	}
	last := g.undo[len(g.undo)-1]
	n := 0
	for len(g.undo) > 0 {
		a := g.undo[len(g.undo)-1]
		if a.Kind != last.Kind || a.T != last.T || math.Abs(a.At-last.At) > 1.5 || (n > 0 && a.T != BRoad) {
			break
		}
		g.undo = g.undo[:len(g.undo)-1]
		g.undoOne(a)
		n++
	}
	g.dirty = true
	return n > 0
}

func (g *Game) undoOne(a UndoAct) {
	bt := &BTypes[a.T]
	w := g.W
	if a.Isl < 0 || a.Isl >= len(g.Islands) {
		return
	}
	I := g.Islands[a.Isl]
	if a.Kind == 0 {
		// Bau zurücknehmen: alles erstatten
		if a.T == BRoad {
			i := w.Idx(a.X, a.Y)
			if !w.Road[i] {
				return
			}
			w.Road[i] = false
		} else {
			b := g.BuildingAt(a.X, a.Y)
			if b == nil || b.T != a.T || b.X != a.X || b.Y != a.Y {
				return
			}
			for yy := b.Y; yy < b.Y+bt.H; yy++ {
				for xx := b.X; xx < b.X+bt.W; xx++ {
					w.Occ[w.Idx(xx, yy)] = 0
				}
			}
			for k, o := range g.B {
				if o == b {
					g.B = append(g.B[:k], g.B[k+1:]...)
					break
				}
			}
		}
		g.Credits += float64(bt.Cost)
		for k, m := range bt.Mat {
			I.Stock[MatGoods[k]] = math.Min(float64(I.Cap), I.Stock[MatGoods[k]]+float64(m))
		}
		return
	}
	// Abriss zurücknehmen: Gebäude wieder hinstellen, Erstattung abziehen
	if a.T == BRoad {
		w.Road[w.Idx(a.X, a.Y)] = true
		return
	}
	for yy := a.Y; yy < a.Y+bt.H; yy++ {
		for xx := a.X; xx < a.X+bt.W; xx++ {
			if !w.In(xx, yy) || w.Occ[w.Idx(xx, yy)] != 0 || w.Road[w.Idx(xx, yy)] {
				return
			}
		}
	}
	g.Credits += float64(bt.Cost)
	for k, m := range bt.Mat {
		I.Stock[MatGoods[k]] += float64(m) - a.Ref[k]
	}
	g.noUndo = true
	b := g.forcePlace(a.T, a.X, a.Y, nil)
	g.noUndo = false
	if b != nil {
		b.Constr = 0
		b.Tier = a.Tier
	}
}

func timeNowStr() string { return time.Now().Format("02.01.2006") }

func (g *Game) shipMul() float64 {
	if g.HasTech(TechFrachter) {
		return 1.25
	}
	return 1
}
