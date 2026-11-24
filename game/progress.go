package game

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
)

// ================================================================ Forschung

type Tech struct {
	Name string
	Desc string
	Cost int
	Req  int // vorausgesetzte Technologie (-1 keine)
}

const (
	TechFertigung = iota
	TechLager
	TechSteuern
	TechFrachter
	TechSozial
	TechEnergie
	TechBrand
	TechLaser
	TechRaketen
	TechAutomation
	TechHandel
	TechMetropole
	NumTechs
)

var Techs = [NumTechs]Tech{
	{"Effiziente Fertigung", "+10 % Produktion in allen Betrieben", 20, -1},
	{"Kompaktlager", "+25 Lagerkapazität je Kontor und Lagerhaus", 25, -1},
	{"Steuerreform", "+10 % Steuereinnahmen", 35, -1},
	{"Leichtbau-Frachter", "Schiffe fahren 25 % schneller", 40, TechLager},
	{"Sozialprogramme", "-10 % Unterhaltskosten", 45, TechSteuern},
	{"Fusionsoptimierung", "+15 % Energie aus allen Kraftwerken", 55, TechFertigung},
	{"Brandschutz-Drohnen", "Brandgefahr halbiert", 50, TechSozial},
	{"Laserabwehr II", "+40 % Schaden für Korvetten und Abwehrtürme", 70, TechEnergie},
	{"Raketentechnik", "Raketen zum Mond starten alle 60 s statt 90 s", 80, TechFrachter},
	{"Vollautomation", "weitere +15 % Produktion", 110, TechEnergie},
	{"Interstellarer Handel", "+50 % Einnahmen aus Handelsabkommen", 90, TechSteuern},
	{"Metropolenplanung", "+15 % Steuern und Bürger steigen schneller auf", 150, TechAutomation},
}

func (g *Game) HasTech(t int) bool { return t >= 0 && t < NumTechs && g.Tech[t] }

func (g *Game) prodMul() float64 {
	m := 1.0
	if g.HasTech(TechFertigung) {
		m += 0.10
	}
	if g.HasTech(TechAutomation) {
		m += 0.15
	}
	if g.Buff > 0 {
		m *= g.BuffMul
	}
	return m
}

// Forschungspunkte pro Minute
func (g *Game) researchRate() float64 {
	r := 0.4 + float64(g.Pop[1])/120 + float64(g.Pop[2])/50
	for _, b := range g.allBuildingsAllRegions() {
		if b.Own == 0 && b.T == BLabor && b.Constr == 0 && b.Status == StOK {
			r += 2
		}
	}
	return r
}

func (g *Game) allBuildingsAllRegions() []*Building {
	if g.Other == nil {
		return g.B
	}
	return append(append([]*Building(nil), g.B...), g.Other.B...)
}

func (g *Game) Research(t int) bool {
	tc := &Techs[t]
	if g.Tech[t] || (tc.Req >= 0 && !g.Tech[tc.Req]) || g.RP < float64(tc.Cost) {
		return false
	}
	g.RP -= float64(tc.Cost)
	g.Tech[t] = true
	g.Notify(T("Erforscht: ")+T(tc.Name)+" – "+tc.Desc, hex(0x7ad0ff))
	PlaySfx(SfxQuest)
	g.dirty = true
	return true
}

func (g *Game) drawResearch(c *Canvas) {
	u := g.UI
	c.FillRect(0, 0, c.W, c.H, RGB{2, 6, 12}, 185)
	pw, ph := imin(1060, c.W-40), imin(640, c.H-30)
	px, py := c.W/2-pw/2, c.H/2-ph/2
	u.panel(c, px, py, pw, ph)
	c.Text(FSmall, px+24, py+16, "ZUKUNFTSTECHNOLOGIEN", cAccent)
	c.Text(FBig, px+24, py+34, "Forschung", cText)
	c.TextR(FBold, px+pw-150, py+26, Tf("%d Forschungspunkte", int(g.RP)), hex(0x7ad0ff))
	c.TextR(FSmall, px+pw-150, py+50, Tf("+%.1f pro Minute (Labore, Techniker, Ingenieure)", g.researchRate()), cDim)
	if u.button(c, px+pw-130, py+20, 110, 34, "Schließen", true) || In.KeyHit[KEsc] {
		u.ShowResearch = false
		In.KeyHit[KEsc] = false
	}
	cols := 4
	cw := (pw - 48 - (cols-1)*12) / cols
	chh := 150
	for t := 0; t < NumTechs; t++ {
		tc := &Techs[t]
		x := px + 24 + (t%cols)*(cw+12)
		y := py + 86 + (t/cols)*(chh+12)
		done := g.Tech[t]
		avail := tc.Req < 0 || g.Tech[tc.Req]
		bg := RGB{14, 22, 36}
		if done {
			bg = RGB{14, 48, 44}
		} else if !avail {
			bg = RGB{18, 20, 26}
		}
		c.RoundRect(x, y, cw, chh, 10, bg, 230)
		col := cText
		if !avail && !done {
			col = RGB{110, 120, 135}
		}
		c.Text(FBold, x+14, y+12, tc.Name, col)
		for i, l := range WrapText(FSmall, tc.Desc, cw-28) {
			c.Text(FSmall, x+14, y+40+i*18, l, cDim)
		}
		switch {
		case done:
			c.Text(FBold, x+14, y+chh-36, "Erforscht", cGreen)
		case !avail:
			c.Text(FSmall, x+14, y+chh-32, T("Benötigt: ")+T(Techs[tc.Req].Name), RGB{150, 120, 110})
		default:
			if u.button(c, x+14, y+chh-44, cw-28, 32, Tf("Erforschen (%d FP)", tc.Cost), g.RP >= float64(tc.Cost)) {
				g.Research(t)
			}
		}
	}
}

// ================================================================ Ereignisse mit Entscheidungen

type EventChoice struct {
	Label string
	Apply func(g *Game)
	OK    func(g *Game) bool
}

type GameEvent struct {
	Title, Text string
	Choices     []EventChoice
}

func payOK(n float64) func(g *Game) bool { return func(g *Game) bool { return g.Credits >= n } }

var gameEvents = []GameEvent{
	{"Wandernde Ingenieurin", "Eine geniale Ingenieurin bietet an, deine Fabriken zu optimieren. Ihr Honorar: 2.500 Credits.", []EventChoice{
		{"Engagieren (+25 % Produktion für 5 Minuten)", func(g *Game) { g.Credits -= 2500; g.setBuff(1.25, 300) }, payOK(2500)},
		{"Ablehnen", func(g *Game) {}, nil}}},
	{"Günstiger Stahl", "Ein Schrotthändler bietet 40 Stahl zum Sonderpreis von 1.200 Credits an.", []EventChoice{
		{"Kaufen", func(g *Game) { g.Credits -= 1200; g.addToKontor(GStahl, 40) }, payOK(1200)},
		{"Kein Interesse", func(g *Game) {}, nil}}},
	{"Streik in den Fabriken", "Die Arbeiter fordern einen Bonus. Ohne Einigung sinkt die Produktion für einige Minuten.", []EventChoice{
		{"Bonus zahlen (2.000 Credits)", func(g *Game) { g.Credits -= 2000 }, payOK(2000)},
		{"Hart bleiben (-40 % Produktion für 3 Minuten)", func(g *Game) { g.setBuff(0.6, 180) }, nil}}},
	{"Meteoritenschauer", "Astronomen warnen vor einem Meteoritenschauer. Ein Schutzschild kostet 3.000 Credits.", []EventChoice{
		{"Schild aktivieren", func(g *Game) { g.Credits -= 3000 }, payOK(3000)},
		{"Risiko eingehen", func(g *Game) {
			if len(g.B) > 0 && g.rng.Float64() < 0.7 {
				g.ignite(g.B[g.rng.Intn(len(g.B))])
			}
		}, nil}}},
	{"Forschungsdurchbruch", "Deine Labore stehen vor einem Durchbruch. Investierst du zusätzliches Geld?", []EventChoice{
		{"Investieren (2.000 Credits, +25 Forschungspunkte)", func(g *Game) { g.Credits -= 2000; g.RP += 25 }, payOK(2000)},
		{"Ergebnisse verkaufen (+3.000 Credits)", func(g *Game) { g.Credits += 3000 }, nil}}},
	{"Stadtfest", "Die Bürger wünschen sich ein großes Fest mit Drohnenshow über dem Hafen.", []EventChoice{
		{"Fest feiern (2.500 Credits, +15 % Steuern für 5 Minuten)", func(g *Game) { g.Credits -= 2500; g.TaxBuff = 300 }, payOK(2500)},
		{"Dieses Jahr nicht", func(g *Game) {}, nil}}},
	{"Treibende Frachtkapsel", "Deine Sensoren orten eine herrenlose Frachtkapsel im Meer.", []EventChoice{
		{"Bergen (+30 Baustoffe, +20 Nahrung)", func(g *Game) { g.addToKontor(GBaustoffe, 30); g.addToKontor(GNahrung, 20) }, nil},
		{"Den Behörden melden (+10 Forschungspunkte)", func(g *Game) { g.RP += 10 }, nil}}},
	{"Investoren von der Erde", "Eine Investorengruppe will in deine Kolonie einsteigen und fordert einen Teil der Einnahmen.", []EventChoice{
		{"Annehmen (+8.000 Credits, -10 % Steuern für 10 Minuten)", func(g *Game) { g.Credits += 8000; g.TaxBuff = -600 }, nil},
		{"Ablehnen", func(g *Game) {}, nil}}},
}

func (g *Game) setBuff(mul, sec float64) {
	g.BuffMul, g.Buff = mul, sec
}

func (g *Game) addToKontor(gd, n int) {
	for _, I := range g.Islands {
		if I.Owned {
			I.Stock[gd] = math.Min(float64(I.Cap), I.Stock[gd]+float64(n))
			return
		}
	}
}

func (g *Game) updateEvents(dt float64) {
	if g.Buff > 0 {
		g.Buff = math.Max(0, g.Buff-dt)
	}
	if g.TaxBuff > 0 {
		g.TaxBuff = math.Max(0, g.TaxBuff-dt)
	} else if g.TaxBuff < 0 {
		g.TaxBuff = math.Min(0, g.TaxBuff+dt)
	}
	g.RP += g.researchRate() * dt / 60
	if g.Mode == ModeTutorial || g.Event0 != nil || g.ownedIslands() == 0 {
		return
	}
	if g.EvT == 0 {
		g.EvT = 480
	}
	g.EvT -= dt
	if g.EvT <= 0 {
		g.EvT = 420 + g.rng.Float64()*420
		e := gameEvents[g.rng.Intn(len(gameEvents))]
		g.Event0 = &e
		PlaySfx(SfxNotify)
	}
}

func (g *Game) taxMul() float64 {
	m := 1.0
	if g.HasTech(TechSteuern) {
		m += 0.1
	}
	if g.HasTech(TechMetropole) {
		m += 0.15
	}
	if g.TaxBuff > 0 {
		m += 0.15
	} else if g.TaxBuff < 0 {
		m -= 0.10
	}
	return m
}

func (g *Game) drawEvent(c *Canvas) {
	e := g.Event0
	if e == nil {
		return
	}
	u := g.UI
	c.FillRect(0, 0, c.W, c.H, RGB{2, 6, 12}, 120)
	pw := 560
	lines := WrapText(FNormal, e.Text, pw-60)
	ph := 140 + len(lines)*24 + len(e.Choices)*48
	px, py := c.W/2-pw/2, c.H/2-ph/2
	u.panel(c, px, py, pw, ph)
	c.FillRect(px+14, py, pw-28, 3, cGold, 255)
	c.Text(FSmall, px+28, py+18, "EREIGNIS", cGold)
	c.TextS(FSerif, px+28, py+36, e.Title, cText)
	for i, l := range lines {
		c.Text(FNormal, px+28, py+78+i*24, l, RGB{215, 225, 240})
	}
	y := py + 96 + len(lines)*24
	for _, ch := range e.Choices {
		ok := ch.OK == nil || ch.OK(g)
		if u.button(c, px+28, y, pw-56, 38, ch.Label, ok) {
			ch.Apply(g)
			g.Event0 = nil
			PlaySfx(SfxClick)
			return
		}
		y += 48
	}
}

// ================================================================ Erfolge

type Achievement struct {
	ID, Name, Desc string
	Check          func(g *Game) bool
}

var Achievements = []Achievement{
	{"kontor", "Erster Spatenstich", "Gründe deinen ersten Kontor", func(g *Game) bool { return countB(g, BKontor) > 0 }},
	{"pop100", "Kleine Gemeinde", "Erreiche 100 Einwohner", func(g *Game) bool { return g.totalPop() >= 100 }},
	{"pop1000", "Großstadt", "Erreiche 1.000 Einwohner", func(g *Game) bool { return g.totalPop() >= 1000 }},
	{"pop5000", "Megacity", "Erreiche 5.000 Einwohner", func(g *Game) bool { return g.totalPop() >= 5000 }},
	{"tech", "Fachkräfte", "Die ersten Techniker ziehen ein", func(g *Game) bool { return g.MaxPop[TTechniker] > 0 }},
	{"ing", "Ingenieurskunst", "Die ersten Ingenieure ziehen ein", func(g *Game) bool { return g.MaxPop[TIngenieure] > 0 }},
	{"ing500", "Denkfabrik", "500 Ingenieure", func(g *Game) bool { return g.MaxPop[TIngenieure] >= 500 }},
	{"isl3", "Inselreich", "Besitze 3 Inseln", func(g *Game) bool { return g.ownedIslands() >= 3 }},
	{"rich", "Reich wie Krösus", "Besitze 250.000 Credits", func(g *Game) bool { return g.Credits >= 250000 }},
	{"moon", "Ein kleiner Schritt", "Schalte die Mondbasis frei", func(g *Game) bool { return g.Other != nil }},
	{"he3", "Helium-Zeitalter", "Fördere Helium-3 auf dem Mond", func(g *Game) bool {
		_, mb := g.regionData(1)
		for _, b := range mb {
			if b.T == BHe3 && b.Status == StOK {
				return true
			}
		}
		return false
	}},
	{"pirate", "Piratenjäger", "Versenke ein Piratenschiff", func(g *Game) bool { return g.Stats.Pirates > 0 }},
	{"war", "Sieg zur See", "Versenke 3 feindliche Kriegsschiffe", func(g *Game) bool { return g.Stats.Warships >= 3 }},
	{"ally", "Diplomat", "Schließe ein Bündnis", func(g *Game) bool {
		for _, rv := range g.Rivals {
			if rv.Treaty == TreatyAlliance {
				return true
			}
		}
		return false
	}},
	{"fire", "Feuerwehr-Held", "Lösche 5 Brände", func(g *Game) bool { return g.Stats.Fires >= 5 }},
	{"research", "Forscherdrang", "Erforsche 6 Technologien", func(g *Game) bool { return g.techCount() >= 6 }},
	{"allresearch", "Zukunft gemeistert", "Erforsche alle Technologien", func(g *Game) bool { return g.techCount() >= NumTechs }},
	{"tutorial", "Musterschüler", "Schließe das Tutorial ab", func(g *Game) bool { return g.tut != nil && g.tut.Step >= len(tutSteps)-1 }},
	{"campaign", "Held des Archipels", "Beende die Kampagne", func(g *Game) bool { return g.Mode == ModeCampaign && g.Won }},
	{"blue", "Stadtplaner", "Baue 10 Blaupausen", func(g *Game) bool { return g.Stats.Blueprints >= 10 }},
}

type GameStats struct {
	Pirates, Warships, Fires, Blueprints int
}

func (g *Game) totalPop() int {
	return g.Pop[0] + g.Pop[1] + g.Pop[2]
}

func (g *Game) techCount() int {
	n := 0
	for _, t := range g.Tech {
		if t {
			n++
		}
	}
	return n
}

var unlockedAch map[string]string // ID → Datum

func achPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	dir = filepath.Join(dir, "NovaArchipel")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "erfolge.json")
}

func loadAch() {
	if unlockedAch != nil {
		return
	}
	unlockedAch = map[string]string{}
	if b, err := os.ReadFile(achPath()); err == nil {
		json.Unmarshal(b, &unlockedAch)
	}
}

func (g *Game) checkAchievements(dt float64) {
	g.achT -= dt
	if g.achT > 0 {
		return
	}
	g.achT = 2
	loadAch()
	changed := false
	for _, a := range Achievements {
		if _, ok := unlockedAch[a.ID]; ok {
			continue
		}
		if a.Check(g) {
			unlockedAch[a.ID] = timeNowStr()
			changed = true
			g.Notify(T("Erfolg freigeschaltet: ")+T(a.Name)+" – "+a.Desc, hex(0xffd24a))
			PlaySfx(SfxUpgrade)
		}
	}
	if changed {
		if b, err := json.Marshal(unlockedAch); err == nil {
			os.WriteFile(achPath(), b, 0644)
		}
	}
}

func drawAchievements(c *Canvas, u *UIState, onClose func()) {
	loadAch()
	c.FillRect(0, 0, c.W, c.H, RGB{2, 6, 12}, 185)
	pw, ph := imin(1000, c.W-40), imin(660, c.H-30)
	px, py := c.W/2-pw/2, c.H/2-ph/2
	u.panel(c, px, py, pw, ph)
	c.Text(FSmall, px+24, py+16, Tf("%d VON %d FREIGESCHALTET", len(unlockedAch), len(Achievements)), cAccent)
	c.Text(FBig, px+24, py+34, "Erfolge", cText)
	if u.button(c, px+pw-130, py+20, 110, 34, "Schließen", true) || In.KeyHit[KEsc] {
		In.KeyHit[KEsc] = false
		onClose()
	}
	cols := 2
	cw := (pw - 60) / cols
	rh := imin(52, (ph-110)/((len(Achievements)+1)/2))
	for i, a := range Achievements {
		x := px + 24 + (i%cols)*(cw+12)
		y := py + 86 + (i/cols)*rh
		d, ok := unlockedAch[a.ID]
		bg := RGB{16, 22, 34}
		if ok {
			bg = RGB{40, 34, 14}
		}
		c.RoundRect(x, y, cw, rh-6, 8, bg, 220)
		ic := RGB{70, 80, 95}
		if ok {
			ic = hex(0xffd24a)
		}
		c.Circle(float64(x+22), float64(y+(rh-6)/2), 12, ic, 255)
		c.Circle(float64(x+22), float64(y+(rh-6)/2), 6, bg, 255)
		col := cText
		if !ok {
			col = RGB{130, 140, 155}
		}
		c.Text(FBold, x+44, y+4, a.Name, col)
		c.Text(FSmall, x+44, y+24, a.Desc, cDim)
		if ok {
			c.TextR(FSmall, x+cw-12, y+6, d, RGB{180, 160, 100})
		}
	}
}

// Für Vorschaubilder
func (g *Game) TriggerEventForShots(i int) {
	e := gameEvents[i%len(gameEvents)]
	g.Event0 = &e
}
