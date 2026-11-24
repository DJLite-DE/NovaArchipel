package game

import (
	"math"
)

// ---------------------------------------------------------------- Seegefechte

const KindPirate = 5

// Fraktion: 0 Spieler, -1 neutraler Händler, -2 Piraten, k+1 KI-Gegner
func shipFac(s *Ship) int {
	switch {
	case s.Kind == 0:
		return 0
	case s.Kind == 1:
		return -1
	case s.Kind == KindPirate:
		return -2
	}
	return s.Kind - 1
}

func FacCol(f int) RGB {
	switch {
	case f == 0:
		return mCyan
	case f == -1:
		return hex(0xffc24a)
	case f == -2:
		return hex(0xff3048)
	}
	return RivalDefs[f-1].Col
}

func (g *Game) facHostile(a, b int) bool {
	if a == b {
		return false
	}
	if a == -2 || b == -2 {
		return true
	}
	if a == -1 || b == -1 {
		return false
	}
	if a == 0 || b == 0 {
		k := a + b // der jeweils andere
		if rv := g.rivalByIdx(k - 1); rv != nil {
			return rv.War
		}
	}
	return false
}

func (g *Game) hostile(a, b *Ship) bool { return g.facHostile(shipFac(a), shipFac(b)) }

func (s *Ship) ensureHP() {
	if s.MaxHP == 0 {
		s.MaxHP = 60
		if s.War {
			s.MaxHP = 150
			if s.Kind == KindPirate {
				s.MaxHP = 120
			}
		}
		s.HP = s.MaxHP
	}
}

func (g *Game) nearCam(x, y float64) bool {
	u := g.UI
	return math.Hypot(x-u.TX, y-u.TZ) < u.Dist*1.1+8
}

func (g *Game) sfxAt(id int, x, y float64) {
	if g.nearCam(x, y) {
		PlaySfx(id)
	}
}

// Treffer auf ein Schiff
func (g *Game) hitShip(t *Ship, dmg float64, by int) {
	t.ensureHP()
	t.HP -= dmg
	if t.HP > 0 || t.Dead {
		return
	}
	t.Dead = true
	g.Event(GEvent{Kind: EvExplode, X: t.X, Y: 0.5, Z: t.Y, S: 1.3})
	g.Event(GEvent{Kind: EvSplash, X: t.X, Z: t.Y})
	g.sfxAt(SfxBoom, t.X, t.Y)
	f := shipFac(t)
	switch {
	case f == 0:
		g.Notify("„"+T(t.Name)+T("“ wurde versenkt!"), cRed)
	case f == -2 && by == 0:
		g.Stats.Pirates++
		g.Credits += 800
		g.Notify("Piraten versenkt! Kopfgeld: +800 Credits", cGreen)
	case f > 0 && t.War:
		if rv := g.rivalByIdx(f - 1); rv != nil {
			rv.Losses++
			if by == 0 {
				g.Stats.Warships++
				g.Notify(T("Kriegsschiff von ")+rv.Def().Name+T(" versenkt!"), cGreen)
			}
		}
	}
	if g.UI.SelShip == t.ID {
		g.UI.SelShip = 0
	}
}

func (g *Game) fireAt(s *Ship, tx, ty, th float64) {
	g.Event(GEvent{Kind: EvLaser, X: s.X, Y: 0.75, Z: s.Y, X2: tx, Y2: th, Z2: ty, Col: FacCol(shipFac(s))})
	g.sfxAt(SfxLaser, s.X, s.Y)
}

func (g *Game) nearestHostile(s *Ship, rng float64, cargoOnly bool) *Ship {
	var best *Ship
	bd := rng
	for _, o := range g.Ships {
		if o == s || o.Dead || !g.hostile(s, o) || (cargoOnly && o.War) {
			continue
		}
		d := math.Hypot(o.X-s.X, o.Y-s.Y)
		if d < bd {
			best, bd = o, d
		}
	}
	return best
}

// Kampflogik eines bewaffneten Schiffs. Gibt true zurück, wenn es gerade kämpft.
func (g *Game) combatAI(s *Ship, dt float64) bool {
	s.Cool -= dt
	s.Rep -= dt
	tgt := g.ShipByID(s.AtkS)
	if tgt != nil && (tgt.Dead || !g.hostile(s, tgt)) {
		tgt = nil
	}
	if tgt == nil {
		s.AtkS = 0
		tgt = g.nearestHostile(s, 6, false)
	}
	if tgt == nil {
		return false
	}
	d := math.Hypot(tgt.X-s.X, tgt.Y-s.Y)
	if d <= 4.6 {
		// Breitseite: kurz stehen bleiben ist nicht nötig, einfach feuern
		if s.Cool <= 0 {
			dmg := 14.0
			if s.Kind == KindPirate {
				dmg = 11
			}
			s.Cool = 1.5
			g.fireAt(s, tgt.X, tgt.Y, 0.45)
			if shipFac(s) == 0 && g.HasTech(TechLaser) {
				dmg *= 1.4
			}
			g.hitShip(tgt, dmg, shipFac(s))
		}
		if s.AtkS == tgt.ID {
			s.Path = nil
		}
		return true
	}
	if s.AtkS == tgt.ID && s.Rep <= 0 {
		s.Rep = 1.5
		g.ShipGoto(s, int(tgt.X), int(tgt.Y))
	}
	return s.AtkS == tgt.ID
}

// Reparatur in der Nähe eines eigenen Kontors
func (g *Game) repairShip(s *Ship, dt float64) {
	if s.HP >= s.MaxHP {
		return
	}
	f := shipFac(s)
	var ks []*Building
	if f == 0 {
		ks = g.kontors()
	} else if f > 0 {
		for _, b := range g.RB {
			if b.T == BKontor && b.Own == f {
				ks = append(ks, b)
			}
		}
	}
	for _, k := range ks {
		if g.shipNear(s, k) || math.Hypot(s.X-float64(k.Dock[0]), s.Y-float64(k.Dock[1])) < 4 {
			s.HP = math.Min(s.MaxHP, s.HP+5*dt)
			return
		}
	}
}

func (g *Game) updateCombat(dt float64) {
	for _, s := range g.Ships {
		s.ensureHP()
		if s.Dead {
			continue
		}
		g.repairShip(s, dt)
		if !s.War {
			continue
		}
		fighting := g.combatAI(s, dt)
		if fighting {
			continue
		}
		switch {
		case s.Kind == KindPirate:
			g.pirateAI(s, dt)
		case s.Kind >= 2:
			g.rivalWarAI(s, dt)
		}
	}
	// Abwehrtürme
	for _, b := range g.B {
		if b.T != BTurret || b.Constr > 0 || b.Paused || b.Fire > 0 {
			continue
		}
		b.Cool -= dt
		cx, cy := b.Center()
		var tgt *Ship
		bd := float64(BTypes[BTurret].Radius)
		for _, s := range g.Ships {
			if s.Dead || !g.facHostile(0, shipFac(s)) {
				continue
			}
			if d := math.Hypot(s.X-cx, s.Y-cy); d < bd {
				tgt, bd = s, d
			}
		}
		if tgt == nil {
			continue
		}
		ta := math.Atan2(tgt.Y-cy, tgt.X-cx)
		b.Aim += math.Remainder(ta-b.Aim, 2*math.Pi) * math.Min(1, dt*6)
		if b.Cool <= 0 {
			b.Cool = 1.3
			g.Event(GEvent{Kind: EvLaser, X: cx + math.Cos(b.Aim)*0.3, Y: b.BaseY + 1.05, Z: cy + math.Sin(b.Aim)*0.3, X2: tgt.X, Y2: 0.45, Z2: tgt.Y, Col: hex(0x60f0ff)})
			g.sfxAt(SfxLaser, cx, cy)
			g.hitShip(tgt, 16*map[bool]float64{true: 1.4, false: 1}[g.HasTech(TechLaser)], 0)
		}
	}
	// Versenkte Schiffe entfernen
	k := g.Ships[:0]
	for _, s := range g.Ships {
		if !s.Dead {
			k = append(k, s)
		}
	}
	g.Ships = k
}

// ---------------------------------------------------------------- Piraten

func (g *Game) updatePirates(dt float64) {
	n := 0
	for _, s := range g.Ships {
		if s.Kind == KindPirate {
			n++
		}
	}
	if !Opt.Pirates || g.Mode == ModeTutorial {
		return
	}
	if g.PirT == 0 {
		g.PirT = 1500
	}
	g.PirT -= dt
	if g.PirT > 0 || n >= 2 {
		return
	}
	g.PirT = (600 + g.rng.Float64()*400) * g.dm(1.8, 1, 0.65)
	// nur wenn es etwas zu holen gibt
	cargo := 0
	for _, s := range g.Ships {
		if s.Kind == 0 && !s.War {
			cargo++
		}
	}
	if cargo == 0 {
		return
	}
	w := g.W
	for tries := 0; tries < 200; tries++ {
		var x, y int
		e := g.rng.Intn(w.N - 6)
		switch g.rng.Intn(4) {
		case 0:
			x, y = 3, e+3
		case 1:
			x, y = w.N-4, e+3
		case 2:
			x, y = e+3, 3
		default:
			x, y = e+3, w.N-4
		}
		if w.Terr[w.Idx(x, y)] != TWater {
			continue
		}
		s := &Ship{ID: g.newID(), Name: []string{"Schwarzer Komet", "Nachtfalke", "Rostiger Hai", "Sturmkrähe"}[g.rng.Intn(4)], X: float64(x) + 0.5, Y: float64(y) + 0.5, Kind: KindPirate, War: true, Life: 420, Target: -1}
		for i := range s.Cargo {
			s.Cargo[i].G = -1
		}
		s.Route = Route{GAB: -1, GBA: -1}
		s.ensureHP()
		g.Ships = append(g.Ships, s)
		g.Notify("Piraten gesichtet! Schütze deine Frachter mit Korvetten oder Abwehrtürmen.", hex(0xff4a5a))
		PlaySfx(SfxNotify)
		return
	}
}

func (g *Game) pirateAI(s *Ship, dt float64) {
	s.Life -= dt
	if s.Life <= 0 {
		s.Dead = true
		g.Notify(T("Die Piraten „")+T(s.Name)+T("“ sind abgezogen."), cDim)
		return
	}
	// Beute jagen
	var best *Ship
	bd := 45.0
	for _, o := range g.Ships {
		if o.Dead || o.War || o.Kind == KindPirate {
			continue
		}
		d := math.Hypot(o.X-s.X, o.Y-s.Y)
		if o.Kind == 0 {
			d *= 0.6 // Spieler-Frachter bevorzugt
		}
		if d < bd {
			best, bd = o, d
		}
	}
	if best != nil {
		s.AtkS = best.ID
		if s.Rep <= 0 {
			s.Rep = 2
			g.ShipGoto(s, int(best.X), int(best.Y))
		}
		return
	}
	if len(s.Path) == 0 {
		ks := g.kontors()
		if len(ks) > 0 {
			k := ks[g.rng.Intn(len(ks))]
			g.ShipGoto(s, k.Dock[0]+g.rng.Intn(9)-4, k.Dock[1]+g.rng.Intn(9)-4)
		}
	}
}

// ---------------------------------------------------------------- Krieg mit KI-Gegnern

func (g *Game) rivalWarships(rv *Rival) int {
	n := 0
	for _, s := range g.Ships {
		if s.War && s.Kind == 2+rv.Idx {
			n++
		}
	}
	return n
}

func (g *Game) updateRivalWar(rv *Rival, dt float64) {
	d := rv.Def()
	if !rv.War && rv.Relation < -60 && rv.Age > 1200 {
		g.DeclareWar(rv, false)
	}
	if !rv.War {
		return
	}
	rv.WarT -= dt
	if rv.WarT <= 0 && g.rivalWarships(rv) < imin(1+len(rv.Isles), 3) && len(rv.Isles) > 0 {
		rv.WarT = 110 / d.Expand
		if s := g.rivalShip(rv, g.rng.Intn(len(rv.Isles))); s != nil {
			s.War = true
			s.Name = T(d.Name) + T(" Korvette")
			s.ensureHP()
			g.Ships = append(g.Ships, s)
		}
	}
	if rv.Losses >= 3 {
		rv.War = false
		rv.Losses = 0
		rv.Bonus += 25
		g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), "Genug! Dieser Krieg kostet uns zu viel. Lassen Sie uns Frieden schließen."})
		g.Notify(T("Frieden mit ")+T(d.Name)+"!", cGreen)
	}
}

func (g *Game) DeclareWar(rv *Rival, byPlayer bool) {
	if rv.War {
		return
	}
	rv.War = true
	rv.Treaty = TreatyNone
	rv.WarT = 20
	rv.Losses = 0
	if byPlayer {
		rv.Bonus -= 40
		g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), "Sie wollen Krieg? Den sollen Sie bekommen."})
	} else {
		g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), T("Ihre Arroganz ist unerträglich. ") + rv.Def().Name + T(" erklärt Ihnen hiermit den Krieg!")})
	}
	g.Notify(T("KRIEG mit ")+rv.Def().Name+T("! Baue Korvetten in der Werft und Abwehrtürme an deinen Küsten."), hex(0xff4a5a))
	PlaySfx(SfxThunder)
}

const PeaceCost = 5000

func (g *Game) OfferPeace(rv *Rival) {
	if !rv.War || g.Credits < PeaceCost {
		return
	}
	if rv.Relation < -75 && rv.Losses == 0 {
		g.Notify(rv.Def().Name+T(" lehnt den Frieden ab."), cRed)
		PlaySfx(SfxError)
		return
	}
	g.Credits -= PeaceCost
	rv.War = false
	rv.Losses = 0
	rv.Bonus += 20
	g.Dialogs = append(g.Dialogs, Dialog{rv.Speaker(), "Einverstanden. Die Waffen schweigen – vorerst."})
}

func (g *Game) rivalWarAI(s *Ship, dt float64) {
	rv := g.rivalByIdx(s.Kind - 2)
	if rv == nil {
		return
	}
	s.Rep -= 0
	if !rv.War {
		// Friedenszeit: zurück und patrouillieren
		if len(s.Path) == 0 && s.Wait <= 0 && len(rv.Isles) > 0 {
			ri := rv.Isles[g.rng.Intn(len(rv.Isles))]
			if k := g.rivalKontor(&ri); k != nil {
				g.ShipGoto(s, k.Dock[0]+g.rng.Intn(11)-5, k.Dock[1]+g.rng.Intn(11)-5)
			}
			s.Wait = 10
		}
		return
	}
	// Spielerschiffe jagen
	if t := g.nearestHostile(s, 14, false); t != nil {
		s.AtkS = t.ID
		return
	}
	// Kontore beschießen
	ks := g.kontors()
	if len(ks) == 0 {
		return
	}
	var near *Building
	for _, k := range ks {
		if math.Hypot(s.X-float64(k.Dock[0]), s.Y-float64(k.Dock[1])) < 5 {
			near = k
		}
	}
	if near != nil {
		if s.Cool <= 0 {
			s.Cool = 2.2
			cx, cy := near.Center()
			g.fireAt(s, cx, cy, near.BaseY+0.8)
			g.Event(GEvent{Kind: EvExplode, X: cx, Y: near.BaseY + 0.6, Z: cy, S: 0.5})
			if g.rng.Float64() < 0.3 {
				g.igniteNear(near, 4)
			}
			if g.rng.Float64() < 0.1 {
				g.Credits -= math.Max(0, math.Min(g.Credits, 50))
			}
			if g.rng.Float64() < 0.15 {
				g.Notify(Tf("%s beschießt %s!", rv.Def().Name, g.Islands[near.Isl].Name), hex(0xff4a5a))
			}
		}
		return
	}
	if len(s.Path) == 0 {
		k := ks[g.rng.Intn(len(ks))]
		g.ShipGoto(s, k.Dock[0], k.Dock[1])
	}
}

// ---------------------------------------------------------------- Werft: Korvette

func (g *Game) spawnWarship(b *Building) {
	names := []string{"Wächter", "Bastion", "Aegis", "Falke", "Speer", "Komet"}
	n := 0
	for _, s := range g.Ships {
		if s.Kind == 0 && s.War {
			n++
		}
	}
	s := &Ship{ID: g.newID(), Name: T("Korvette ") + names[n%len(names)], X: float64(b.Dock[0]) + 0.5, Y: float64(b.Dock[1]) + 0.5, War: true}
	for i := range s.Cargo {
		s.Cargo[i].G = -1
	}
	s.Route = Route{GAB: -1, GBA: -1}
	s.ensureHP()
	g.Ships = append(g.Ships, s)
	g.Notify(T("Die „")+T(s.Name)+T("“ ist einsatzbereit!"), hex(0x96d678))
}

// ---------------------------------------------------------------- Brände & Stromausfälle

func (g *Game) fireCovered(b *Building) bool {
	cx, cy := b.Center()
	for _, f := range g.B {
		if f.T == BFeuer && f.Isl == b.Isl && f.Constr == 0 && f.Conn && !f.Paused && f.Fire == 0 {
			fx, fy := f.Center()
			if math.Hypot(cx-fx, cy-fy) <= float64(BTypes[BFeuer].Radius) {
				return true
			}
		}
	}
	return false
}

func (g *Game) ignite(b *Building) {
	if b.Fire > 0 || b.Constr > 0 || b.T == BRoad || b.T == BPark {
		return
	}
	b.Fire = 0.01
	g.Notify(T("Feuer: ")+T(BTypes[b.T].Name)+T(" auf ")+T(g.Islands[b.Isl].Name)+T(" brennt!"), hex(0xff7a3a))
}

func (g *Game) igniteNear(k *Building, r float64) {
	kx, ky := k.Center()
	var cand []*Building
	for _, b := range g.B {
		bx, by := b.Center()
		if b.Isl == k.Isl && math.Hypot(bx-kx, by-ky) < r && b.Fire == 0 && b.Constr == 0 {
			cand = append(cand, b)
		}
	}
	if len(cand) > 0 {
		g.ignite(cand[g.rng.Intn(len(cand))])
	}
}

func (g *Game) updateDisasters(dt float64) {
	// laufende Brände
	for _, b := range g.B {
		if b.Fire <= 0 {
			continue
		}
		b.Fire += dt
		bt := &BTypes[b.T]
		cx, cy := b.Center()
		b.Cool -= dt
		if b.Cool <= 0 {
			b.Cool = 0.12
			g.Event(GEvent{Kind: EvFire, X: cx, Y: b.BaseY + 0.3, Z: cy, S: float64(bt.W) * 0.7})
		}
		covered := g.fireCovered(b)
		switch {
		case covered && b.Fire > 7:
			b.Fire = 0
			g.Stats.Fires++
			g.Notify("Brand gelöscht – die Feuerwache war schnell zur Stelle.", cGreen)
		case b.Fire > 55:
			// ausgebrannt: wird wieder aufgebaut
			b.Fire = 0
			cost := bt.Cost * 3 / 10
			g.Credits -= float64(cost)
			b.Constr = 1
			g.Notify(Tf("%s ist ausgebrannt und wird neu errichtet (-%s Credits). Eine Feuerwache hätte geholfen!", bt.Name, fmtNum(cost)), cRed)
		case int(b.Fire)%15 == 14 && int(b.Fire-dt)%15 != 14 && g.rng.Float64() < 0.35:
			g.igniteNear(b, 3)
		}
	}
	for _, I := range g.Islands {
		if I.Blackout > 0 {
			I.Blackout -= dt
			if I.Blackout <= 0 {
				I.Blackout = 0
				g.Notify(T("Die Stromversorgung auf ")+T(I.Name)+T(" läuft wieder."), cGreen)
			}
		}
	}
	if !Opt.Disasters {
		return
	}
	pop := g.Pop[0] + g.Pop[1] + g.Pop[2]
	if pop < 150 || g.OnMoon() || g.Mode == ModeTutorial {
		return
	}
	if g.FireT == 0 {
		g.FireT = 500
	}
	g.FireT -= dt
	if g.FireT <= 0 {
		g.FireT = (420 + g.rng.Float64()*420) * g.dm(1.8, 1, 0.7) * map[bool]float64{true: 2, false: 1}[g.HasTech(TechBrand)]
		var cand []*Building
		for _, b := range g.B {
			if b.T != BRoad && b.T != BPark && b.T != BKontor && b.Constr == 0 && b.Fire == 0 {
				cand = append(cand, b)
				if BTypes[b.T].Out >= 0 {
					cand = append(cand, b) // Fabriken brennen häufiger
				}
			}
		}
		if len(cand) > 0 {
			b := cand[g.rng.Intn(len(cand))]
			if !(g.fireCovered(b) && g.rng.Float64() < 0.65) {
				g.ignite(b)
				PlaySfx(SfxNotify)
			}
		}
	}
	// Stromausfall bei Sturm
	if g.Storm > 0.6 && g.rng.Float64() < dt/240 {
		var cand []*Island
		for _, I := range g.Islands {
			if I.Owned && I.EUse > 0 && I.Blackout == 0 && float64(I.EProd) < float64(I.EUse)*1.25 {
				cand = append(cand, I)
			}
		}
		if len(cand) > 0 {
			I := cand[g.rng.Intn(len(cand))]
			I.Blackout = 40
			g.Notify(T("Stromausfall auf ")+T(I.Name)+T(" durch den Sturm! Mehr Energiereserven helfen."), hex(0xffc24a))
			PlaySfx(SfxThunder)
		}
	}
}

// Für Vorschaubilder: kleines Gefecht vor der Werft aufstellen
func (g *Game) SpawnTestBattle(werft *Building) {
	if werft == nil {
		return
	}
	g.spawnWarship(werft)
	g.spawnWarship(werft)
	rv := g.Rivals[0]
	rv.War = true
	dx, dy := float64(werft.Dock[0])+0.5, float64(werft.Dock[1])+0.5
	// Feindschiffe etwas weiter draußen auf dem Wasser
	n := 0
	for r := 3.0; r < 9 && n < 2; r += 0.5 {
		for a := 0.0; a < 6.28 && n < 2; a += 0.4 {
			x, y := int(dx+r*math.Cos(a)), int(dy+r*math.Sin(a))
			if g.W.In(x, y) && g.W.Terr[g.W.Idx(x, y)] == TWater && g.W.Dist[g.W.Idx(x, y)] >= 2 {
				s := g.rivalShip(rv, 0)
				if s == nil {
					return
				}
				s.War, s.X, s.Y = true, float64(x)+0.5, float64(y)+0.5
				s.Name = rv.Def().Name + T(" Korvette")
				s.ensureHP()
				s.HP = s.MaxHP * (0.8 + 0.2*float64(n))
				g.Ships = append(g.Ships, s)
				n++
				a += 1.2
			}
		}
	}
	p := &Ship{ID: g.newID(), Name: "Schwarzer Komet", X: dx + 3, Y: dy + 3, Kind: KindPirate, War: true, Life: 400, Target: -1}
	for i := range p.Cargo {
		p.Cargo[i].G = -1
	}
	p.ensureHP()
	if g.W.In(int(p.X), int(p.Y)) && g.W.Terr[g.W.Idx(int(p.X), int(p.Y))] == TWater {
		g.Ships = append(g.Ships, p)
	}
	g.UI.CenterOn(g, dx+2, dy+2)
	// Ein Brand in der Stadt
	for _, b := range g.B {
		if BTypes[b.T].Out >= 0 && b.Fire == 0 {
			bx, by := b.Center()
			if math.Hypot(bx-dx, by-dy) < 10 {
				b.Fire = 5
				break
			}
		}
	}
}

func (g *Game) StepForShots(dt float64) {
	g.updateShips(dt)
	g.updateCombat(dt)
	g.updateDisasters(dt)
}
