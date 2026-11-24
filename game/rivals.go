package game

import (
	"fmt"
	"math"
	"sort"
)

// ---------------------------------------------------------------- KI-Gegenspieler

type RivalDef struct {
	Name, Leader, Role string
	Col                RGB
	Greet              string
	Claim              string
	Angry              string
	Happy              string
	Expand             float64 // Wachstumstempo
	Friendly           float64 // Grundhaltung -50..50
	MaxIsles           int
	Start              string // Startinsel
}

var RivalDefs = []RivalDef{
	{Name: "Helion Dynamics", Leader: "Dara Helion", Role: "CEO, Helion Dynamics", Col: hex(0xff7a3d),
		Greet:  "Willkommen im Archipel, Gouverneur. Helion Dynamics war zuerst hier – und wir haben nicht vor, zu teilen.",
		Claim:  "Helion Dynamics sichert sich %s. Fortschritt wartet auf niemanden.",
		Angry:  "Sie kommen uns in die Quere. Das wird Helion Dynamics nicht vergessen.",
		Happy:  "Ich gebe zu: Mit Ihnen lassen sich Geschäfte machen.",
		Expand: 1.35, Friendly: -15, MaxIsles: 3, Start: "Silberklippe"},
	{Name: "Aqua Sola", Leader: "Ilka Nordström", Role: "Sprecherin, Aqua Sola", Col: hex(0x4fe08a),
		Greet:  "Schön, Nachbarn zu haben! Aqua Sola baut im Einklang mit dem Meer. Lasst uns zusammenarbeiten.",
		Claim:  "Aqua Sola hat auf %s eine neue Gemeinschaft gegründet.",
		Angry:  "Wir sind enttäuscht. So behandelt man keine Nachbarn.",
		Happy:  "Gemeinsam machen wir dieses Archipel zu etwas Besonderem!",
		Expand: 0.85, Friendly: 20, MaxIsles: 2, Start: "Nebelhorn"},
	{Name: "Obsidian Syndikat", Leader: "Marek Vane", Role: "Handelsfürst, Obsidian Syndikat", Col: hex(0xb57aff),
		Greet:  "Ah, frische Kundschaft. Das Obsidian Syndikat handelt mit allem – zum richtigen Preis.",
		Claim:  "Das Obsidian Syndikat übernimmt %s. Gute Lage, gute Rendite.",
		Angry:  "Schlechte Geschäfte haben Folgen, Gouverneur.",
		Happy:  "Ein Vergnügen, mit Ihnen zu handeln. Wirklich.",
		Expand: 1.0, Friendly: 0, MaxIsles: 2, Start: "Smaragdriff"},
}

const (
	TreatyNone = iota
	TreatyTrade
	TreatyAlliance
)

var TreatyNames = []string{"Kein Abkommen", "Handelsabkommen", "Bündnis"}

type RivalIsle struct {
	Isl    int
	KX, KY int
	R      float64
	Blocks int
	Full   bool
}

type RivalReq struct {
	Good   int
	Amt    int
	Reward int
	T      float64
}

type Rival struct {
	Idx      int
	Relation float64
	Bonus    float64
	Treaty   int
	Isles    []RivalIsle
	Tier     int
	Age      float64
	GrowT    float64
	ExpT     float64
	Req      *RivalReq
	ReqT     float64
	Pop      int
	Mood     int // letzte Stimmungsstufe für Meldungen
	War      bool
	WarT     float64
	Losses   int
	cur      int
}

func (rv *Rival) Def() *RivalDef { return &RivalDefs[rv.Idx] }

func (rv *Rival) Speaker() Speaker {
	d := rv.Def()
	return Speaker{d.Leader, d.Role, d.Col}
}

func RelationText(v float64) (string, RGB) {
	switch {
	case v < -50:
		return "Feindselig", hex(0xff4a5a)
	case v < -15:
		return "Angespannt", hex(0xff9a4a)
	case v < 15:
		return "Neutral", hex(0xd0d8e4)
	case v < 50:
		return "Freundlich", hex(0x8ae68a)
	}
	return "Eng verbunden", hex(0x46f0a0)
}

func (g *Game) RivalOf(isl int) *Rival {
	if isl < 0 || isl >= len(g.Islands) || g.Islands[isl].Rival == 0 {
		return nil
	}
	k := g.Islands[isl].Rival - 1
	for _, r := range g.Rivals {
		if r.Idx == k {
			return r
		}
	}
	return nil
}

func (g *Game) islandByName(n string) int {
	for i, I := range g.Islands {
		if I.Name == n {
			return i
		}
	}
	return -1
}

// KI-Gegner ins Spiel bringen
func (g *Game) InitRivals(n int) {
	n = clampi(n, 0, len(RivalDefs))
	for k := 0; k < n; k++ {
		d := &RivalDefs[k]
		isl := g.islandByName(d.Start)
		if isl < 0 || g.Islands[isl].Owned || g.Islands[isl].Rival != 0 {
			isl = g.freeIsland(g.Islands[0].CX, g.Islands[0].CY, true)
		}
		if isl < 0 {
			continue
		}
		rv := &Rival{Idx: k, Relation: d.Friendly, ReqT: 240 + float64(k)*90, ExpT: 1200 / d.Expand, cur: 0}
		g.Rivals = append(g.Rivals, rv)
		if !g.rivalSettle(rv, isl, false) {
			g.Rivals = g.Rivals[:len(g.Rivals)-1]
			continue
		}
		// Startausbau, damit die Stadt schon sichtbar ist
		for i := 0; i < 7; i++ {
			g.rivalGrow(rv, &rv.Isles[0])
		}
		g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), d.Greet})
	}
	for _, b := range g.RB {
		b.Constr = 0
	}
	g.dirty = true
}

// freie Insel suchen: far=true → möglichst weit weg vom Punkt, sonst möglichst nah
func (g *Game) freeIsland(px, py int, far bool) int {
	best, bd := -1, 0.0
	for i, I := range g.Islands {
		if i == 0 || I.Owned || I.Rival != 0 || I.Tiles < 120 {
			continue
		}
		d := math.Hypot(float64(I.CX-px), float64(I.CY-py))
		if !far {
			d = -d
		}
		if best < 0 || d > bd {
			best, bd = i, d
		}
	}
	return best
}

func (g *Game) rivalSettle(rv *Rival, isl int, announce bool) bool {
	I := g.Islands[isl]
	kx, ky, ok := g.findKontorSpot(isl, float64(I.CX), float64(I.CY))
	if !ok {
		return false
	}
	I.Rival = rv.Idx + 1
	ri := RivalIsle{Isl: isl, KX: kx, KY: ky, R: 9}
	if g.rivalPlace(rv, isl, BKontor, kx, ky, 0) == nil {
		I.Rival = 0
		return false
	}
	rv.Isles = append(rv.Isles, ri)
	// Frachter
	if s := g.rivalShip(rv, len(rv.Isles)-1); s != nil {
		g.Ships = append(g.Ships, s)
	}
	if announce {
		d := rv.Def()
		g.Notify(fmt.Sprintf(d.Claim, I.Name), d.Col)
	}
	return true
}

func (g *Game) rivalShip(rv *Rival, isle int) *Ship {
	ri := &rv.Isles[isle]
	k := g.rivalKontor(ri)
	if k == nil {
		return nil
	}
	names := []string{"Frachter", "Kurier", "Transporter"}
	s := &Ship{ID: g.newID(), Name: rv.Def().Name + " " + names[isle%3], X: float64(k.Dock[0]) + 0.5, Y: float64(k.Dock[1]) + 0.5, Kind: 2 + rv.Idx, Target: -1}
	for i := range s.Cargo {
		s.Cargo[i].G = -1
	}
	s.Route = Route{GAB: -1, GBA: -1}
	return s
}

func (g *Game) rivalKontor(ri *RivalIsle) *Building {
	for _, b := range g.RB {
		if b.T == BKontor && b.Isl == ri.Isl {
			return b
		}
	}
	return nil
}

func (g *Game) rivalFoot(isl, x, y, w, h int) bool {
	W := g.W
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if !W.In(xx, yy) {
				return false
			}
			i := W.Idx(xx, yy)
			if int(W.Isl[i]) != isl || W.Terr[i] == TWater || W.Terr[i] == TRock || W.Occ[i] != 0 || W.Road[i] {
				return false
			}
		}
	}
	return true
}

func (g *Game) rivalPlace(rv *Rival, isl, t, x, y, tier int) *Building {
	bt := &BTypes[t]
	if !g.rivalFoot(isl, x, y, bt.W, bt.H) {
		return nil
	}
	w := g.W
	b := &Building{ID: g.newID(), T: t, X: x, Y: y, Isl: isl, Var: int(hash2(x, y, 77) * 12), Prod: 1, Tier: tier, Own: rv.Idx + 1, Conn: true, Constr: 1}
	b.BaseY = w.Flatten(x, y, bt.W, bt.H)
	for yy := y; yy < y+bt.H; yy++ {
		for xx := x; xx < x+bt.W; xx++ {
			i := w.Idx(xx, yy)
			w.Occ[i] = int32(b.ID + 1)
			w.Deco[i] = 0
		}
	}
	if bt.Coastal {
		_, d := g.coastal(x, y, bt.W, bt.H)
		b.Dock = d
	}
	g.RB = append(g.RB, b)
	return b
}

// Straßenraster der KI-Stadt im aktuellen Radius
func (g *Game) rivalRoads(ri *RivalIsle) {
	w := g.W
	ox, oy := ri.KX+3, ri.KY+3
	cx, cy := float64(ri.KX)+1.5, float64(ri.KY)+1.5
	r := int(ri.R) + 1
	for y := ri.KY - r; y <= ri.KY+r+3; y++ {
		for x := ri.KX - r; x <= ri.KX+r+3; x++ {
			if !w.In(x, y) || math.Hypot(float64(x)-cx, float64(y)-cy) > ri.R+1 {
				continue
			}
			if ((x-ox)%5+5)%5 != 0 && ((y-oy)%5+5)%5 != 0 {
				continue
			}
			i := w.Idx(x, y)
			if int(w.Isl[i]) == ri.Isl && w.Terr[i] != TWater && w.Terr[i] != TRock && w.Occ[i] == 0 && !w.Road[i] {
				w.Road[i] = true
				w.Deco[i] = 0
			}
		}
	}
}

// Ein Wachstumsschritt: ein Block (4×4) wird bebaut
func (g *Game) rivalGrow(rv *Rival, ri *RivalIsle) {
	if ri.Full {
		return
	}
	g.rivalRoads(ri)
	ox, oy := ri.KX+3, ri.KY+3
	cx, cy := float64(ri.KX)+1.5, float64(ri.KY)+1.5
	type blk struct {
		x, y int
		d    float64
	}
	var bs []blk
	r := int(ri.R) + 5
	for y := oy - r - 5; y < oy+r; y += 5 {
		for x := ox - r - 5; x < ox+r; x += 5 {
			bx := ox + ((x-ox)/5)*5 + 1
			by := oy + ((y-oy)/5)*5 + 1
			d := math.Hypot(float64(bx)+2-cx, float64(by)+2-cy)
			if d > ri.R+1 {
				continue
			}
			// Block muss ganz frei sein oder darf noch Lücken haben
			free := 0
			for yy := by; yy < by+4; yy++ {
				for xx := bx; xx < bx+4; xx++ {
					if g.rivalFoot(ri.Isl, xx, yy, 1, 1) {
						free++
					}
				}
			}
			if free >= 4 {
				bs = append(bs, blk{bx, by, d})
			}
		}
	}
	sort.Slice(bs, func(i, j int) bool {
		if bs[i].d != bs[j].d {
			return bs[i].d < bs[j].d
		}
		return bs[i].x*1000+bs[i].y < bs[j].x*1000+bs[j].y
	})
	for _, b := range bs {
		if g.rivalFillBlock(rv, ri, b.x, b.y) {
			ri.Blocks++
			g.dirty = true
			return
		}
	}
	// kein Platz mehr → Radius wachsen lassen
	ri.R += 3
	I := g.Islands[ri.Isl]
	if ri.R > math.Sqrt(float64(I.Tiles))*0.9+6 {
		ri.Full = true
	}
}

func (g *Game) rivalFillBlock(rv *Rival, ri *RivalIsle, bx, by int) bool {
	n := ri.Blocks
	placed := 0
	houseTier := func() int {
		t := rv.Tier
		if t > 0 && hash2(bx, by+placed, 11) < 0.3 {
			t--
		}
		return t
	}
	tryHouses := func() {
		for _, d := range [][2]int{{0, 0}, {2, 0}, {0, 2}, {2, 2}} {
			if g.rivalPlace(rv, ri.Isl, BHouse, bx+d[0], by+d[1], houseTier()) != nil {
				placed++
			}
		}
	}
	if n%4 == 3 {
		// Infrastruktur / Produktion je nach Entwicklungsstufe
		pool := []int{BKomm, BRecycle, BNahrung, BLager, BWind, BPark}
		if rv.Tier >= 1 {
			pool = append(pool, BMedi, BSolar, BStahl, BTextil, BRoest)
		}
		if rv.Tier >= 2 {
			pool = append(pool, BLabor, BChip, BHolo, BFusion)
		}
		t := pool[(n/4+ri.Isl*3+rv.Idx)%len(pool)]
		if rv.Tier >= 1 && n/4 < 2 {
			t = BKomm
		}
		bt := &BTypes[t]
		switch {
		case bt.W == 1:
			for _, d := range [][2]int{{0, 0}, {3, 0}, {0, 3}, {3, 3}, {1, 1}, {2, 2}} {
				if g.rivalPlace(rv, ri.Isl, t, bx+d[0], by+d[1], 0) != nil {
					placed++
				}
			}
		case bt.W == 2 && bt.H == 2:
			for _, d := range [][2]int{{0, 0}, {2, 2}} {
				if g.rivalPlace(rv, ri.Isl, t, bx+d[0], by+d[1], 0) != nil {
					placed++
				}
			}
			for _, d := range [][2]int{{2, 0}, {0, 2}} {
				if g.rivalPlace(rv, ri.Isl, BHouse, bx+d[0], by+d[1], houseTier()) != nil {
					placed++
				}
			}
		default:
			if g.rivalPlace(rv, ri.Isl, t, bx, by, 0) != nil {
				placed++
				for _, p := range [][2]int{{3, 0}, {3, 1}, {3, 2}, {3, 3}, {0, 3}, {1, 3}, {2, 3}} {
					g.rivalPlace(rv, ri.Isl, BPark, bx+p[0], by+p[1], 0)
				}
			}
		}
		if placed == 0 {
			tryHouses()
		}
	} else {
		tryHouses()
		if placed > 0 && placed < 4 {
			for _, d := range [][2]int{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {0, 3}, {3, 3}} {
				g.rivalPlace(rv, ri.Isl, BPark, bx+d[0], by+d[1], 0)
			}
		}
	}
	return placed > 0
}

// --------------------------------------------------------------- Simulation

func (g *Game) updateRivals(dt float64) {
	if len(g.Rivals) == 0 {
		return
	}
	for _, b := range g.RB {
		if b.Constr > 0 {
			b.Constr -= dt / BuildTime(b.T)
			if b.Constr <= 0 {
				b.Constr = 0
				bt := &BTypes[b.T]
				g.Event(GEvent{Kind: EvBuilt, X: float64(b.X) + float64(bt.W)/2, Y: b.BaseY, Z: float64(b.Y) + float64(bt.H)/2, S: float64(bt.W)})
			}
		}
	}
	for _, rv := range g.Rivals {
		d := rv.Def()
		rv.Age += dt
		// Entwicklungsstufe
		if rv.Tier == 0 && rv.Age > 900/d.Expand {
			rv.Tier = 1
		} else if rv.Tier == 1 && rv.Age > 2400/d.Expand {
			rv.Tier = 2
		}
		// Wachstum
		rv.GrowT -= dt
		if rv.GrowT <= 0 && len(rv.Isles) > 0 {
			rv.GrowT = 34 / d.Expand * g.dm(1.4, 1, 0.75)
			ri := &rv.Isles[rv.cur%len(rv.Isles)]
			rv.cur++
			g.rivalGrow(rv, ri)
			g.rivalUpgrade(rv)
		}
		// Expansion auf weitere Inseln
		rv.ExpT -= dt
		if rv.ExpT <= 0 {
			rv.ExpT = 600 / d.Expand * g.dm(1.5, 1, 0.75)
			blocks := 0
			for _, ri := range rv.Isles {
				blocks += ri.Blocks
			}
			if len(rv.Isles) < d.MaxIsles && blocks >= 9*len(rv.Isles) {
				I0 := g.Islands[rv.Isles[0].Isl]
				// Verbündete greifen nicht nach Inseln nahe deiner Kolonien
				isl := g.freeIsland(I0.CX, I0.CY, false)
				if isl >= 0 && !(rv.Treaty == TreatyAlliance && g.nearPlayer(isl)) {
					if g.rivalSettle(rv, isl, true) {
						for i := 0; i < 3; i++ {
							g.rivalGrow(rv, &rv.Isles[len(rv.Isles)-1])
						}
					}
				}
			}
		}
		// Beziehung
		rv.Bonus -= rv.Bonus * dt / 1500
		base := d.Friendly + rv.Bonus + []float64{0, 10, 25}[rv.Treaty]
		if rv.War {
			base -= 25
		}
		g.updateRivalWar(rv, dt)
		base = clampf(base, -100, 100)
		rv.Relation += (base - rv.Relation) * math.Min(1, dt/90)
		// Abkommen kündigen bei schlechter Stimmung
		if rv.Treaty == TreatyAlliance && rv.Relation < 30 {
			rv.Treaty = TreatyTrade
			g.Notify(T(d.Name)+T(" hat das Bündnis aufgekündigt."), hex(0xff9a4a))
		}
		if rv.Treaty == TreatyTrade && rv.Relation < -5 {
			rv.Treaty = TreatyNone
			g.Notify(T(d.Name)+T(" hat das Handelsabkommen aufgekündigt."), hex(0xff9a4a))
		}
		mood := 0
		if rv.Relation < -40 {
			mood = -1
		} else if rv.Relation > 55 {
			mood = 1
		}
		if mood != rv.Mood {
			if mood == -1 {
				g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), d.Angry})
			} else if mood == 1 {
				g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), d.Happy})
			}
			rv.Mood = mood
		}
		// Anfragen
		if rv.War {
			rv.Req = nil
		} else if rv.Req != nil {
			rv.Req.T -= dt
			if rv.Req.T <= 0 {
				rv.Bonus -= 8
				g.Notify(T(d.Name)+T(": Anfrage nicht erfüllt – die Beziehung leidet."), hex(0xff9a4a))
				rv.Req = nil
				rv.ReqT = 300 + g.rng.Float64()*300
			}
		} else {
			rv.ReqT -= dt
			if rv.ReqT <= 0 {
				rv.Req = g.newRequest(rv)
				if rv.Req != nil {
					g.Notify(Tf("%s bittet um %d %s – öffne die Diplomatie.", d.Name, rv.Req.Amt, Goods[rv.Req.Good].Name), d.Col)
					PlaySfx(SfxNotify)
				}
			}
		}
		// Einwohner
		pop := 0
		for _, b := range g.RB {
			if b.Own == rv.Idx+1 && b.T == BHouse {
				pop += Tiers[b.Tier].MaxRes
			}
		}
		rv.Pop = pop
	}
}

func (g *Game) nearPlayer(isl int) bool {
	I := g.Islands[isl]
	for _, J := range g.Islands {
		if J.Owned && math.Hypot(float64(I.CX-J.CX), float64(I.CY-J.CY)) < 45 {
			return true
		}
	}
	return false
}

// Häuser nach und nach auf die aktuelle Stufe heben
func (g *Game) rivalUpgrade(rv *Rival) {
	n := 0
	for _, b := range g.RB {
		if b.Own == rv.Idx+1 && b.T == BHouse && b.Tier < rv.Tier && hash2(b.X, b.Y, int(rv.Age/30)) < 0.5 {
			b.Tier++
			n++
			if n >= 3 {
				break
			}
		}
	}
}

func (g *Game) newRequest(rv *Rival) *RivalReq {
	pool := []int{GBaustoffe, GNahrung, GWasser}
	if rv.Tier >= 1 || g.MaxPop[TTechniker] > 0 {
		pool = append(pool, GStahl, GSynthkaffee, GTextil)
	}
	if rv.Tier >= 2 && g.MaxPop[TIngenieure] > 0 {
		pool = append(pool, GChips, GHolo)
	}
	gd := pool[g.rng.Intn(len(pool))]
	amt := 20 + g.rng.Intn(4)*10
	if Goods[gd].Price >= 100 {
		amt = 10 + g.rng.Intn(2)*5
	}
	rew := int(float64(amt*Goods[gd].Price)*1.8/50+0.5) * 50
	return &RivalReq{Good: gd, Amt: amt, Reward: rew, T: 300}
}

func (g *Game) playerStock(gd int) int {
	n := 0.0
	for _, I := range g.Islands {
		if I.Owned {
			n += I.Stock[gd]
		}
	}
	return int(n)
}

func (g *Game) takeStock(gd, amt int) {
	for _, I := range g.Islands {
		if !I.Owned || amt <= 0 {
			continue
		}
		k := math.Min(I.Stock[gd], float64(amt))
		I.Stock[gd] -= k
		amt -= int(k)
	}
}

func (g *Game) FulfilRequest(rv *Rival) {
	r := rv.Req
	if r == nil || g.playerStock(r.Good) < r.Amt {
		return
	}
	g.takeStock(r.Good, r.Amt)
	g.Credits += float64(r.Reward)
	rv.Bonus += 14
	rv.Req = nil
	rv.ReqT = 360 + g.rng.Float64()*300
	PlaySfx(SfxQuest)
	g.Notify(Tf("%s ist dankbar! +%s Credits", rv.Def().Name, fmtNum(r.Reward)), cGreen)
}

func (g *Game) DeclineRequest(rv *Rival) {
	if rv.Req == nil {
		return
	}
	rv.Req = nil
	rv.Bonus -= 5
	rv.ReqT = 400 + g.rng.Float64()*300
}

const GiftCost = 2500

func (g *Game) SendGift(rv *Rival) {
	if g.Credits < GiftCost {
		return
	}
	g.Credits -= GiftCost
	rv.Bonus += 10
	PlaySfx(SfxQuest)
	g.Notify(rv.Def().Name+T(" nimmt dein Geschenk gerne an."), cGreen)
}

func (g *Game) ProposeTreaty(rv *Rival, t int) {
	need := []float64{0, 20, 60}[t]
	if rv.War || rv.Relation < need {
		g.Notify(rv.Def().Name+T(" lehnt ab – die Beziehung ist nicht gut genug."), cRed)
		PlaySfx(SfxError)
		return
	}
	rv.Treaty = t
	PlaySfx(SfxQuest)
	g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), []string{"", "Abgemacht. Unsere Frachter werden Ihre Kontore anlaufen – zum beiderseitigen Vorteil.", "Ein Bündnis also. Gemeinsam sind wir in diesem Archipel unschlagbar."}[t]})
}

func (g *Game) CancelTreaty(rv *Rival) {
	if rv.Treaty == TreatyNone {
		return
	}
	rv.Treaty--
	rv.Bonus -= 12
	g.Notify(T("Abkommen mit ")+rv.Def().Name+T(" gekündigt."), hex(0xff9a4a))
}

// Einnahmen aus Abkommen pro Minute
func (g *Game) treatyIncome() float64 {
	s := 0.0
	for _, rv := range g.Rivals {
		s += []float64{0, 60, 180}[rv.Treaty] * map[bool]float64{true: 1.5, false: 1}[g.HasTech(TechHandel)]
	}
	return s
}

// Spieler gründet eine Kolonie → Konkurrenz reagiert
func (g *Game) onPlayerColony(isl int) {
	for _, rv := range g.Rivals {
		pen := 5.0
		if rv.Idx == 0 {
			pen = 10
		}
		for _, ri := range rv.Isles {
			I, J := g.Islands[isl], g.Islands[ri.Isl]
			if math.Hypot(float64(I.CX-J.CX), float64(I.CY-J.CY)) < 50 {
				pen *= 1.6
			}
		}
		if rv.Treaty == TreatyAlliance {
			pen *= 0.3
		}
		rv.Bonus -= pen
	}
}

// Frachter der Konkurrenz
func (g *Game) rivalShipAI(s *Ship) {
	k := s.Kind - 2
	var rv *Rival
	for _, r := range g.Rivals {
		if r.Idx == k {
			rv = r
		}
	}
	if rv == nil {
		return
	}
	// am Ziel angekommen?
	if s.Target > 0 {
		if b := g.Bld(s.Target); b != nil && b.T == BKontor && rv.Treaty >= TreatyTrade {
			g.rivalTrade(rv, s, b)
		}
		s.Target = -1
		s.Wait = 8
		return
	}
	// Ziel wählen
	if rv.Treaty >= TreatyTrade && g.rng.Float64() < 0.5 {
		ks := g.kontors()
		if len(ks) > 0 {
			b := ks[g.rng.Intn(len(ks))]
			if g.ShipGoto(s, b.Dock[0], b.Dock[1]) {
				s.Target = b.ID
				return
			}
		}
	}
	if len(rv.Isles) > 0 {
		ri := &rv.Isles[g.rng.Intn(len(rv.Isles))]
		if kb := g.rivalKontor(ri); kb != nil {
			g.ShipGoto(s, kb.Dock[0], kb.Dock[1])
		}
	}
	s.Target = -2
	s.Wait = 6
}

func (g *Game) rivalTrade(rv *Rival, s *Ship, k *Building) {
	I := g.Islands[k.Isl]
	got := 0
	var parts []string
	// Überschüsse aufkaufen
	for gd := 0; gd < NumGoods; gd++ {
		if I.Cap > 0 && I.Stock[gd] > float64(I.Cap)*0.75 {
			n := imin(20, int(I.Stock[gd]-float64(I.Cap)*0.6))
			if n <= 0 {
				continue
			}
			I.Stock[gd] -= float64(n)
			v := n * Goods[gd].Price * 9 / 10
			got += v
			parts = append(parts, Tf("%d %s", n, Goods[gd].Name))
		}
	}
	// Engpässe bei Grundwaren ausgleichen
	for _, gd := range []int{GBaustoffe, GNahrung, GWasser} {
		if I.Stock[gd] < 8 && g.Credits > 2000 {
			n := imin(15, I.Cap-int(I.Stock[gd]))
			if n > 0 {
				I.Stock[gd] += float64(n)
				got -= n * Goods[gd].Price
				parts = append(parts, Tf("+%d %s", n, Goods[gd].Name))
			}
		}
	}
	if len(parts) == 0 {
		return
	}
	g.Credits += float64(got)
	sign := "+"
	if got < 0 {
		sign = ""
	}
	g.Notify(Tf("%s handelt in %s (%s Credits)", rv.Def().Name, I.Name, sign+fmtNum(got)), rv.Def().Col)
	rv.Bonus += 1
}

// KI-Gebäude unter einer Kachel
func (g *Game) RivalBldAt(x, y int) *Building {
	if !g.W.In(x, y) {
		return nil
	}
	o := int(g.W.Occ[g.W.Idx(x, y)]) - 1
	if o < 0 {
		return nil
	}
	for _, b := range g.RB {
		if b.ID == o {
			return b
		}
	}
	return nil
}

func (g *Game) rivalByIdx(k int) *Rival {
	for _, r := range g.Rivals {
		if r.Idx == k {
			return r
		}
	}
	return nil
}

// --------------------------------------------------------------- Diplomatie-Fenster

func (g *Game) drawDiplomacy(c *Canvas) {
	u := g.UI
	W, H := c.W, c.H
	c.FillRect(0, 0, W, H, RGB{2, 6, 12}, 185)
	n := len(g.Rivals)
	cw := 340
	pw := imax(560, n*cw+(n+1)*16)
	ph := imin(650, H-30)
	px, py := W/2-pw/2, H/2-ph/2
	u.panel(c, px, py, pw, ph)
	c.Text(FSmall, px+24, py+16, "BEZIEHUNGEN IM ARCHIPEL", cAccent)
	c.Text(FBig, px+24, py+34, "Diplomatie", cText)
	if u.button(c, px+pw-130, py+20, 110, 34, "Schließen", true) || In.KeyHit[KEsc] {
		u.ShowDiplo = false
		In.KeyHit[KEsc] = false
	}
	if n == 0 {
		for i, l := range WrapText(FNormal, "In diesem Spiel gibt es keine Konkurrenten. Du kannst die Anzahl der KI-Gegner für neue Endlosspiele in den Optionen unter „Spiel“ einstellen.", pw-80) {
			c.Text(FNormal, px+40, py+120+i*24, l, cDim)
		}
		return
	}
	for k, rv := range g.Rivals {
		d := rv.Def()
		x := px + 16 + k*(cw+16)
		y := py + 84
		h := ph - 100
		c.RoundRect(x, y, cw, h, 12, RGB{14, 22, 36}, 210)
		c.FillRect(x+12, y, cw-24, 3, d.Col, 255)
		// Porträt
		c.Circle(float64(x+46), float64(y+50), 30, d.Col.Mul(0.35), 255)
		c.Circle(float64(x+46), float64(y+50), 27, d.Col.Mul(0.75), 255)
		ini := string([]rune(d.Leader)[0:1])
		for _, p := range []rune(d.Leader) {
			_ = p
		}
		if sp := []rune(d.Leader); len(sp) > 0 {
			for i, ch := range sp {
				if ch == ' ' && i+1 < len(sp) {
					ini += string(sp[i+1])
					break
				}
			}
		}
		c.TextC(FBold, x+46, y+40, ini, RGB{255, 255, 255})
		c.TextS(FSerif, x+86, y+24, d.Name, d.Col)
		c.Text(FSmall, x+86, y+54, d.Leader, cDim)
		// Beziehung
		yy := y + 100
		lab, lc := RelationText(rv.Relation)
		c.Text(FSmall, x+16, yy, "Beziehung", cDim)
		c.TextR(FBold, x+cw-16, yy-2, lab, lc)
		yy += 22
		bw := cw - 32
		c.FillRect(x+16, yy, bw, 10, RGB{8, 14, 28}, 230)
		mid := x + 16 + bw/2
		f := clampf(rv.Relation/100, -1, 1)
		if f >= 0 {
			c.FillRect(mid, yy, int(float64(bw/2)*f), 10, lc, 255)
		} else {
			w := int(float64(bw/2) * -f)
			c.FillRect(mid-w, yy, w, 10, lc, 255)
		}
		c.FillRect(mid, yy-2, 1, 14, RGB{255, 255, 255}, 160)
		yy += 24
		// Kennzahlen
		var names []string
		for _, ri := range rv.Isles {
			names = append(names, g.Islands[ri.Isl].Name)
		}
		row := func(a, b string) {
			c.Text(FSmall, x+16, yy, a, cDim)
			c.TextR(FSmall, x+cw-16, yy, b, cText)
			yy += 22
		}
		row("Einwohner", fmtNum(rv.Pop))
		row("Entwicklung", Tiers[rv.Tier].Plural)
		row("Inseln", fmt.Sprint(len(rv.Isles)))
		for _, l := range WrapText(FSmall, joinNames(names), cw-32) {
			c.Text(FSmall, x+16, yy, l, d.Col.Mix(cText, 0.4))
			yy += 18
		}
		yy += 4
		if rv.War {
			c.Text(FSmall, x+16, yy, "Status", cDim)
			c.TextR(FBold, x+cw-16, yy-2, "KRIEG", hex(0xff4a5a))
			yy += 22
		} else {
			row("Abkommen", TreatyNames[rv.Treaty])
		}
		yy += 6
		// Anfrage
		if r := rv.Req; r != nil {
			c.RoundRect(x+12, yy, cw-24, 92, 8, RGB{40, 34, 18}, 220)
			c.Text(FSmall, x+22, yy+8, Tf("ANFRAGE · noch %d:%02d", int(r.T)/60, int(r.T)%60), cGold)
			c.Text(FNormal, x+22, yy+26, Tf("%d %s  für  %s Credits", r.Amt, Goods[r.Good].Name, fmtNum(r.Reward)), cText)
			have := g.playerStock(r.Good)
			if u.button(c, x+22, yy+54, 140, 30, Tf("Liefern (%d)", have), have >= r.Amt) {
				g.FulfilRequest(rv)
			}
			if u.button(c, x+cw-142, yy+54, 120, 30, "Ablehnen", true) {
				g.DeclineRequest(rv)
			}
		}
		// Aktionen
		if u.button(c, x+cw-82, y+14, 66, 26, "Zeigen", len(rv.Isles) > 0) {
			ri := rv.Isles[0]
			u.CenterOn(g, float64(ri.KX)+1.5, float64(ri.KY)+1.5)
			u.ShowDiplo = false
		}
		by := y + h - 4*38 - 10
		bwid := cw - 32
		if u.button(c, x+16, by, bwid, 32, T("Geschenk senden (")+fmtNum(GiftCost)+" C)", g.Credits >= GiftCost) {
			g.SendGift(rv)
		}
		by += 38
		switch {
		case rv.War:
			c.RoundRect(x+16, by, bwid, 32, 8, RGB{70, 20, 26}, 220)
			c.TextC(FNormal, x+cw/2, by+6, "Im Krieg", hex(0xff6a7a))
		case rv.Treaty == TreatyNone:
			if u.button(c, x+16, by, bwid, 32, "Handelsabkommen vorschlagen", true) {
				g.ProposeTreaty(rv, TreatyTrade)
			}
			if u.inside(x+16, by, bwid, 32) {
				u.setTip("Handelsabkommen", "Benötigt: Beziehung „Freundlich“ (20+)", "+60 Credits/min, ihre Frachter kaufen deine Überschüsse", "und liefern Grundwaren bei Engpässen.")
			}
		case rv.Treaty == TreatyTrade:
			if u.button(c, x+16, by, bwid, 32, "Bündnis vorschlagen", true) {
				g.ProposeTreaty(rv, TreatyAlliance)
			}
			if u.inside(x+16, by, bwid, 32) {
				u.setTip("Bündnis", "Benötigt: Beziehung 60+", "+180 Credits/min statt +60", "Verbündete besiedeln keine Inseln nahe deiner Kolonien.")
			}
		default:
			c.RoundRect(x+16, by, bwid, 32, 8, RGB{20, 60, 40}, 220)
			c.TextC(FNormal, x+cw/2, by+6, "Verbündet", cGreen)
		}
		by += 38
		if u.button(c, x+16, by, bwid, 32, "Abkommen kündigen", rv.Treaty > 0) {
			g.CancelTreaty(rv)
		}
		by += 38
		if rv.War {
			if u.button(c, x+16, by, bwid, 32, T("Frieden anbieten (")+fmtNum(PeaceCost)+" C)", g.Credits >= PeaceCost) {
				g.OfferPeace(rv)
			}
		} else if u.button(c, x+16, by, bwid, 32, "Krieg erklären", true) {
			g.DeclareWar(rv, true)
		}
	}
}

func joinNames(n []string) string {
	s := ""
	for i, x := range n {
		if i > 0 {
			s += ", "
		}
		s += x
	}
	return s
}

// Für Vorschaubilder: Konkurrenz im Zeitraffer wachsen lassen
func (g *Game) FastForwardRivals(sec float64) {
	for t := 0.0; t < sec; t += 1 {
		g.updateRivals(1)
	}
	g.Recompute()
}
