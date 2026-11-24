package game

import (
	"math"
)

// ---------------------------------------------------------------- Tutorial

type TutStep struct {
	Title string
	Text  string
	Build int                    // Gebäude, auf das gezeigt wird (-1: keins)
	Done  func(g *Game) bool     // Bedingung für den nächsten Schritt
	Ship  bool                   // Pfeil auf das eigene Schiff
	UI    func(u *UIState) [4]int // anderes UI-Element
}

func countB(g *Game, t int) int {
	n := 0
	for _, b := range g.B {
		if b.T == t && b.Constr == 0 {
			n++
		}
	}
	return n
}

func countRoads(g *Game) int {
	n := 0
	for i, r := range g.W.Road {
		if r && g.W.Isl[i] >= 0 && g.Islands[g.W.Isl[i]].Owned {
			n++
		}
	}
	return n
}

var tutSteps = []TutStep{
	{Title: "Willkommen!", Text: "Schön, dass du da bist, Gouverneur! Bewege die Kamera mit W A S D oder halte die rechte Maustaste gedrückt und ziehe. Mit dem Mausrad zoomst du, mit Q und E drehst du die Ansicht.", Build: -1,
		Done: func(g *Game) bool { return g.tut.camMoved > 12 }},
	{Title: "Dein Frachtgleiter", Text: "Das blinkende Schiff ist dein Frachtgleiter mit Baumaterial. Er ist bereits ausgewählt. Klicke mit der RECHTEN Maustaste aufs Wasser nahe der Küste, um ihn dorthin zu schicken.", Build: -1, Ship: true,
		Done: func(g *Game) bool { return g.tut.shipMoved > 2 }},
	{Title: "Der Kontor", Text: "Jede Kolonie beginnt mit einem Kontor. Öffne unten das Baumenü „Infrastruktur“, wähle den Kontor und setze ihn an die Küste in die Nähe deines Schiffs.", Build: BKontor,
		Done: func(g *Game) bool { return countB(g, BKontor) > 0 }},
	{Title: "Straßen", Text: "Gebäude müssen mit Straßen an den Kontor angeschlossen sein. Wähle im Menü „Infrastruktur“ die Straße und ziehe mit gedrückter linker Maustaste ein paar Straßen vom Kontor weg.", Build: BRoad,
		Done: func(g *Game) bool { return countRoads(g) >= 10 }},
	{Title: "Wohnmodule", Text: "Jetzt ziehen die ersten Arbeiter ein. Baue im Menü „Wohnen & Öffentlich“ mindestens 4 Wohnmodule direkt an die Straße.", Build: BHouse,
		Done: func(g *Game) bool { return countB(g, BHouse) >= 4 }},
	{Title: "Nahrung", Text: "Deine Bürger brauchen Nahrung. Baue eine Algenfarm an die Küste und eine Nahrungsfabrik an die Straße (Menü „Arbeiter-Produktion“).", Build: BAlgen,
		Done: func(g *Game) bool { return countB(g, BAlgen) > 0 && countB(g, BNahrung) > 0 }},
	{Title: "Wasser", Text: "Auch Trinkwasser ist wichtig. Baue einen Wasseraufbereiter an die Küste.", Build: BWasser,
		Done: func(g *Game) bool { return countB(g, BWasser) > 0 }},
	{Title: "Energie", Text: "Fabriken brauchen Strom. Baue im Menü „Energie“ ein Windrad. Oben in der Leiste siehst du beim Blitz, wie viel Energie erzeugt und verbraucht wird.", Build: BWind,
		Done: func(g *Game) bool { return countB(g, BWind) > 0 }},
	{Title: "Baustoffe", Text: "Baustoffe brauchst du für fast alles. Baue einen Schrottsammler und ein Recyclingwerk – zusammen stellen sie Baustoffe her.", Build: BSchrott,
		Done: func(g *Game) bool { return countB(g, BSchrott) > 0 && countB(g, BRecycle) > 0 }},
	{Title: "Bedürfnisse", Text: "Klicke ein Wohnmodul mit der linken Maustaste an. Im Fenster rechts siehst du, welche Bedürfnisse erfüllt sind. Zufriedene Bürger steigen später zu Technikern auf.", Build: -1,
		Done: func(g *Game) bool {
			b := g.Bld(g.UI.SelB)
			return b != nil && b.T == BHouse
		}},
	{Title: "Überblick", Text: "Öffne oben die „Statistik“. Dort siehst du Produktion und Verbrauch jeder Ware und deine Finanzen.", Build: -1,
		UI:   func(u *UIState) [4]int { return u.statsRect },
		Done: func(g *Game) bool { return g.UI.ShowStats }},
	{Title: "Zeit & Speichern", Text: "Mit den Knöpfen 1× bis 8× oben rechts läuft die Zeit schneller. F5 speichert schnell, F9 lädt. Ändere jetzt einmal die Geschwindigkeit.", Build: -1,
		Done: func(g *Game) bool { return g.Speed > 1 }},
	{Title: "Geschafft!", Text: "Super, du kennst jetzt die Grundlagen! Baue weiter, erfülle die Bedürfnisse und erreiche die Techniker. Viel Spaß – im Hauptmenü wartet die Kampagne und das Endlosspiel.", Build: -1,
		Done: func(g *Game) bool { return false }},
}

type tutState struct {
	Step     int
	camMoved float64
	lastCam  [3]float64
	shipX    float64
	shipY    float64
	shipMoved float64
	t        float64
}

func NewTutorial() *Game {
	MapOpt = MapOptions{Size: 1, Islands: 9}
	g := NewGame(7)
	g.Mode = ModeTutorial
	g.Notifs = nil
	g.Credits = 30000
	g.tut = &tutState{}
	if len(g.Ships) > 0 {
		g.tut.shipX, g.tut.shipY = g.Ships[0].X, g.Ships[0].Y
	}
	return g
}

func (g *Game) updateTutorial(dt float64) {
	t := g.tut
	if t == nil {
		g.tut = &tutState{}
		t = g.tut
	}
	u := g.UI
	cam := [3]float64{u.TX, u.TZ, u.Dist}
	if t.lastCam != [3]float64{} {
		t.camMoved += math.Abs(cam[0]-t.lastCam[0]) + math.Abs(cam[1]-t.lastCam[1]) + math.Abs(cam[2]-t.lastCam[2])*0.5
	}
	t.lastCam = cam
	for _, s := range g.Ships {
		if s.Kind == 0 && !s.War {
			t.shipMoved = math.Max(t.shipMoved, math.Hypot(s.X-t.shipX, s.Y-t.shipY))
			if len(s.Path) > 0 {
				t.shipMoved = math.Max(t.shipMoved, 3)
			}
			break
		}
	}
	t.t += dt
	if t.Step < len(tutSteps)-1 && tutSteps[t.Step].Done(g) {
		t.Step++
		t.t = 0
		PlaySfx(SfxQuest)
		g.Credits += 500
	}
}

func (g *Game) drawTutorial(c *Canvas) {
	u := g.UI
	t := g.tut
	if t == nil {
		return
	}
	st := tutSteps[t.Step]
	w := 380
	x, y := 12, 58
	lines := WrapText(FNormal, st.Text, w-36)
	h := 100 + len(lines)*22
	u.panel(c, x, y, w, h)
	c.FillRect(x+12, y, w-24, 3, cAccent, 255)
	c.Text(FSmall, x+18, y+14, Tf("TUTORIAL · SCHRITT %d VON %d", t.Step+1, len(tutSteps)), cAccent)
	c.TextS(FSerif, x+18, y+32, st.Title, cText)
	for i, l := range lines {
		c.Text(FNormal, x+18, y+66+i*22, l, RGB{215, 225, 240})
	}
	bar(c, x+18, y+h-20, w-36, 6, float64(t.Step)/float64(len(tutSteps)-1), cAccent)
	if t.Step == len(tutSteps)-1 {
		if u.button(c, x+w-150, y+h-56, 132, 30, "Hauptmenü", true) && Main != nil {
			Main.ToMenu = true
		}
	}
	// Hinweis-Pfeile
	pulse := 0.5 + 0.5*math.Sin(t.t*5)
	hl := func(r [4]int) {
		if r[2] == 0 {
			return
		}
		for k := 0; k < 3; k++ {
			c.RoundRectLine(r[0]-3-k, r[1]-3-k, r[2]+6+2*k, r[3]+6+2*k, 8, hex(0xffd24a), int(220-60*float64(k)))
		}
		ax, ay := r[0]+r[2]/2, r[1]-14-int(pulse*10)
		if r[1] < 120 {
			ay = r[1] + r[3] + 14 + int(pulse*10)
			c.Tri(float64(ax), float64(ay-10), float64(ax-10), float64(ay+8), float64(ax+10), float64(ay+8), hex(0xffd24a), 255)
		} else {
			c.Tri(float64(ax), float64(ay+10), float64(ax-10), float64(ay-8), float64(ax+10), float64(ay-8), hex(0xffd24a), 255)
		}
	}
	if st.UI != nil {
		hl(st.UI(u))
	}
	if st.Build >= 0 {
		cat := BTypes[st.Build].Cat
		if st.Build == BRoad {
			cat = CatInfra
		}
		if u.Cat != cat {
			hl(u.tabRect[cat])
		} else if r, ok := u.itemRect[st.Build]; ok && u.Tool != st.Build {
			hl(r)
		}
	}
	if st.Ship {
		for _, s := range g.Ships {
			if s.Kind == 0 {
				sx, sy := u.Project(V3{s.X, 1.5, s.Y}, float64(c.W), float64(c.H))
				ay := sy - 30 - pulse*12
				c.Tri(sx, ay+16, sx-12, ay-4, sx+12, ay-4, hex(0xffd24a), 255)
				c.Circle(sx, sy-4, 26+pulse*6, hex(0xffd24a), 50)
				break
			}
		}
	}
}
