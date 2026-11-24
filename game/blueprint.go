package game

import (
	"fmt"
	"math"
)

// ---------------------------------------------------------------- Blaupausen-Modus

// Blaupause setzen: belegt die Fläche, kostet nichts, produziert nichts
func (g *Game) PlaceBlueprint(t, x, y int) PlaceCheck {
	if t == BKontor || t == BRoad {
		return PlaceCheck{Reason: "Dafür gibt es keine Blaupause"}
	}
	g.ignoreCost = true
	pc := g.CanPlace(t, x, y)
	g.ignoreCost = false
	if !pc.OK {
		return pc
	}
	bt := &BTypes[t]
	w := g.W
	isl := int(w.Isl[w.Idx(x, y)])
	b := &Building{ID: g.newID(), T: t, X: x, Y: y, Isl: isl, Var: int(hash2(x, y, 77) * 12)}
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
	g.BP = append(g.BP, b)
	return PlaceCheck{OK: true}
}

func (g *Game) BlueAt(x, y int) *Building {
	if !g.W.In(x, y) {
		return nil
	}
	o := int(g.W.Occ[g.W.Idx(x, y)]) - 1
	if o < 0 {
		return nil
	}
	for _, b := range g.BP {
		if b.ID == o {
			return b
		}
	}
	return nil
}

func (g *Game) removeBlueprint(b *Building) {
	bt := &BTypes[b.T]
	for yy := b.Y; yy < b.Y+bt.H; yy++ {
		for xx := b.X; xx < b.X+bt.W; xx++ {
			g.W.Occ[g.W.Idx(xx, yy)] = 0
		}
	}
	k := g.BP[:0]
	for _, o := range g.BP {
		if o != b {
			k = append(k, o)
		}
	}
	g.BP = k
}

// Blaupause in echtes Gebäude umwandeln
func (g *Game) BuildBlueprint(b *Building) (bool, string) {
	g.removeBlueprint(b)
	pc := g.CanPlace(b.T, b.X, b.Y)
	if !pc.OK {
		// zurücklegen
		g.ignoreCost = true
		g.PlaceBlueprint(b.T, b.X, b.Y)
		g.ignoreCost = false
		return false, pc.Reason
	}
	g.forcePlace(b.T, b.X, b.Y, pc.Ship)
	g.Stats.Blueprints++
	return true, ""
}

func (g *Game) BuildAllBlueprints() {
	n, fail := 0, ""
	list := append([]*Building(nil), g.BP...)
	for _, b := range list {
		if ok, why := g.BuildBlueprint(b); ok {
			n++
		} else if fail == "" {
			fail = why
		}
	}
	if n > 0 {
		PlaySfx(SfxBuild)
		g.Notify(Tf("%d Blaupausen werden gebaut.", n), cGreen)
	}
	if len(g.BP) > 0 {
		g.Notify(Tf("%d Blaupausen noch offen: %s", len(g.BP), fail), cGold)
	}
}

func (g *Game) blueprintCost() (int, [3]int) {
	cr := 0
	var m [3]int
	for _, b := range g.BP {
		bt := &BTypes[b.T]
		cr += bt.Cost
		for k := range m {
			m[k] += bt.Mat[k]
		}
	}
	return cr, m
}

// Info-Fenster einer Blaupause
func (g *Game) drawBlueprintPanel(c *Canvas, b *Building) {
	u := g.UI
	bt := &BTypes[b.T]
	w, h := 340, 190
	x := c.W - w - 12
	y := 58
	u.panel(c, x, y, w, h)
	c.Text(FSmall, x+18, y+14, "BLAUPAUSE", hex(0x60b0ff))
	c.TextS(FSerif, x+18, y+30, bt.Name, cText)
	if u.button(c, x+w-34, y+10, 24, 24, "×", true) {
		u.SelBP = 0
	}
	cost := fmtNum(bt.Cost) + T(" Credits")
	for k, m := range bt.Mat {
		if m > 0 {
			cost += Tf(", %d %s", m, Goods[MatGoods[k]].Name)
		}
	}
	c.Text(FSmall, x+18, y+70, T("Kosten: ")+cost, cDim)
	pc := g.CanPlace(b.T, b.X, b.Y)
	_ = pc
	if u.button(c, x+18, y+h-52, 150, 36, "Jetzt bauen", true) {
		if ok, why := g.BuildBlueprint(b); ok {
			PlaySfx(SfxBuild)
			u.SelBP = 0
		} else {
			PlaySfx(SfxError)
			g.Notify(why, cRed)
		}
	}
	if u.button(c, x+w-148, y+h-52, 130, 36, "Verwerfen", true) {
		g.removeBlueprint(b)
		u.SelBP = 0
		PlaySfx(SfxDemolish)
	}
}

// ---------------------------------------------------------------- Statistik

type HistPoint struct {
	T       float64
	Credits float64
	Pop     [NumTiers]int
	Bal     float64
}

func (g *Game) recordHistory() {
	if g.Time-g.histT < 30 && len(g.Hist) > 0 {
		return
	}
	g.histT = g.Time
	g.Hist = append(g.Hist, HistPoint{T: g.Time, Credits: g.Credits, Pop: g.Pop, Bal: g.Balance})
	if len(g.Hist) > 240 {
		g.Hist = g.Hist[len(g.Hist)-240:]
	}
}

// Produktion und Verbrauch pro Minute über alle eigenen Inseln
func (g *Game) goodRates() (prod, cons [NumGoods]float64) {
	for _, b := range g.B {
		bt := &BTypes[b.T]
		if b.Constr > 0 || b.Fire > 0 {
			continue
		}
		if b.T == BHouse {
			for _, nd := range Tiers[b.Tier].Needs {
				if nd.Good >= 0 {
					cons[nd.Good] += b.Res * nd.Rate
				}
			}
			continue
		}
		if bt.Out < 0 || bt.Cycle <= 0 || b.Status != StOK {
			continue
		}
		r := 60 / bt.Cycle
		prod[bt.Out] += r
		for _, in := range bt.In {
			cons[in] += r
		}
	}
	return
}

func (g *Game) drawStats(c *Canvas) {
	u := g.UI
	W, H := c.W, c.H
	c.FillRect(0, 0, W, H, RGB{2, 6, 12}, 185)
	pw, ph := imin(1000, W-60), imin(640, H-40)
	px, py := W/2-pw/2, H/2-ph/2
	u.panel(c, px, py, pw, ph)
	c.Text(FSmall, px+24, py+16, "ÜBERBLICK", cAccent)
	c.Text(FBig, px+24, py+34, "Statistik", cText)
	if u.button(c, px+pw-130, py+20, 110, 34, "Schließen", true) || In.KeyHit[KEsc] {
		u.ShowStats = false
		In.KeyHit[KEsc] = false
	}
	tabs := []string{"Waren", "Entwicklung", "Finanzen"}
	for i, t := range tabs {
		bx := px + 220 + i*130
		on := u.StatTab == i
		if u.button(c, bx, py+24, 122, 30, t, true) {
			u.StatTab = i
		}
		if on {
			c.FillRect(bx+10, py+56, 102, 3, cAccent, 255)
		}
	}
	x0, y0 := px+24, py+84
	switch u.StatTab {
	case 0:
		prod, cons := g.goodRates()
		cols := []int{0, 230, 330, 440, 560}
		hdr := []string{"Ware", "Bestand", "Produktion", "Verbrauch", "Bilanz /min"}
		for i, h := range hdr {
			c.Text(FSmall, x0+cols[i], y0, h, cDim)
		}
		y := y0 + 26
		rowH := imin(32, (ph-140)/NumGoods)
		for gd := 0; gd < NumGoods; gd++ {
			if gd%2 == 0 {
				c.FillRect(x0-8, y-4, pw-32, rowH, RGB{20, 30, 48}, 120)
			}
			c.Blit(GoodIcon(gd, 20), x0, y)
			c.Text(FNormal, x0+28, y, Goods[gd].Name, cText)
			c.Text(FNormal, x0+cols[1], y, fmtNum(g.playerStock(gd)), cText)
			c.Text(FNormal, x0+cols[2], y, Tf("+%.1f", prod[gd]), cGreen)
			c.Text(FNormal, x0+cols[3], y, Tf("-%.1f", cons[gd]), hex(0xff8a8a))
			d := prod[gd] - cons[gd]
			col := cDim
			if d > 0.05 {
				col = cGreen
			} else if d < -0.05 {
				col = cRed
			}
			c.Text(FBold, x0+cols[4], y, Tf("%+.1f", d), col)
			// Balken
			bw := pw - cols[4] - 160
			mx := math.Max(prod[gd], cons[gd])
			if mx > 0 {
				bx := x0 + cols[4] + 90
				c.FillRect(bx, y+4, int(float64(bw)*prod[gd]/mx), 6, cGreen, 200)
				c.FillRect(bx, y+12, int(float64(bw)*cons[gd]/mx), 6, hex(0xff8a8a), 200)
			}
			y += rowH
		}
	case 1:
		cw := pw - 48
		chH := (ph - 150) / 2
		g.drawChart(c, x0, y0, cw, chH, "Einwohner", func(h HistPoint) [3]float64 {
			return [3]float64{float64(h.Pop[0]), float64(h.Pop[1]), float64(h.Pop[2])}
		}, []RGB{Tiers[0].Col, Tiers[1].Col, Tiers[2].Col}, []string{Tiers[0].Plural, Tiers[1].Plural, Tiers[2].Plural})
		g.drawChart(c, x0, y0+chH+40, cw, chH, "Credits", func(h HistPoint) [3]float64 {
			return [3]float64{h.Credits, -1, -1}
		}, []RGB{cGold}, []string{"Credits"})
	case 2:
		upkeep := 0.0
		for _, b := range g.B {
			upkeep += float64(BTypes[b.T].Upkeep)
		}
		tr := g.treatyIncome()
		income := g.Balance - tr + upkeep
		rows := [][2]string{
			{"Steuern der Bürger", Tf("+%s /min", fmtNum(int(income)))},
			{"Handelsabkommen", Tf("+%s /min", fmtNum(int(tr)))},
			{"Unterhalt der Gebäude", Tf("-%s /min", fmtNum(int(upkeep)))},
			{"Bilanz", Tf("%+d /min", int(g.Balance))},
		}
		y := y0 + 10
		for i, r := range rows {
			f := FNormal
			col := cText
			if i == len(rows)-1 {
				f = FBold
				c.FillRect(x0, y-8, 520, 1, cDim, 160)
				col = cGreen
				if g.Balance < 0 {
					col = cRed
				}
			}
			c.Text(f, x0, y, r[0], cText)
			c.TextR(f, x0+520, y, r[1], col)
			y += 40
		}
		y += 20
		ships, war := 0, 0
		for _, s := range g.Ships {
			if s.Kind == 0 {
				ships++
				if s.War {
					war++
				}
			}
		}
		info := [][2]string{
			{"Kolonien", fmt.Sprint(g.ownedIslands())},
			{"Gebäude", fmtNum(len(g.B))},
			{"Frachtgleiter", fmt.Sprint(ships - war)},
			{"Korvetten", fmt.Sprint(war)},
			{"Offene Blaupausen", fmt.Sprint(len(g.BP))},
		}
		for _, r := range info {
			c.Text(FNormal, x0, y, r[0], cDim)
			c.TextR(FNormal, x0+520, y, r[1], cText)
			y += 32
		}
	}
}

func (g *Game) ownedIslands() int {
	n := 0
	for _, I := range g.Islands {
		if I.Owned {
			n++
		}
	}
	return n
}

func (g *Game) drawChart(c *Canvas, x, y, w, h int, title string, val func(HistPoint) [3]float64, cols []RGB, names []string) {
	c.Text(FBold, x, y, title, cText)
	lx := x + TextW(FBold, title) + 24
	for i, n := range names {
		c.FillRect(lx, y+6, 14, 4, cols[i], 255)
		c.Text(FSmall, lx+20, y+1, n, cDim)
		lx += TextW(FSmall, n) + 44
	}
	y += 26
	h -= 26
	c.RoundRect(x, y, w, h, 8, RGB{10, 16, 28}, 200)
	hs := g.Hist
	if len(hs) < 2 {
		c.TextC(FSmall, x+w/2, y+h/2-8, "Noch zu wenig Daten – spiel ein paar Minuten weiter.", cDim)
		return
	}
	mx := 1.0
	for _, p := range hs {
		v := val(p)
		for i := range cols {
			mx = math.Max(mx, v[i])
		}
	}
	for k := 1; k < 4; k++ {
		gy := y + h - h*k/4
		c.FillRect(x+8, gy, w-16, 1, RGB{40, 56, 80}, 160)
		c.Text(FSmall, x+10, gy+2, fmtNum(int(mx*float64(k)/4)), RGB{110, 125, 145})
	}
	for i := range cols {
		for j := 1; j < len(hs); j++ {
			a, b := val(hs[j-1])[i], val(hs[j])[i]
			x0 := float64(x+8) + float64(j-1)/float64(len(hs)-1)*float64(w-16)
			x1 := float64(x+8) + float64(j)/float64(len(hs)-1)*float64(w-16)
			y0 := float64(y+h-6) - a/mx*float64(h-16)
			y1 := float64(y+h-6) - b/mx*float64(h-16)
			c.Line(x0, y0, x1, y1, cols[i], 255)
			c.Line(x0, y0+1, x1, y1+1, cols[i], 120)
		}
	}
	mins := int((hs[len(hs)-1].T - hs[0].T) / 60)
	c.TextR(FSmall, x+w-10, y+h-20, Tf("letzte %d Min.", mins), RGB{110, 125, 145})
}

// Für Vorschaubilder
func (g *Game) SimulateForShots(sec float64) {
	for t := 0.0; t < sec; t += 0.5 {
		g.step(0.5)
		g.Dialogs = nil
	}
}
