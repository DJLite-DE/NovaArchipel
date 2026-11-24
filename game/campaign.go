package game

import (
	"math"
)

const (
	ModeEndless = iota
	ModeCampaign
	ModeTutorial
)

type Speaker struct {
	Name string
	Role string
	Col  RGB
}

var (
	SpVoss  = Speaker{"Admiralin Mara Voss", "Flottenkommando Nova", hex(0x5ed4ff)}
	SpKern  = Speaker{"Dr. Elias Kern", "Chefingenieur", hex(0xffb84a)}
	SpLin   = Speaker{"Lin Okafor", "Sprecherin der Siedler", hex(0x8ae68a)}
	SpOrion = Speaker{"Kapitän Orion", "Händlergilde", hex(0xffd24a)}
)

type Dialog struct {
	Sp   Speaker
	Text string
}

type Objective struct {
	Text  string
	Check func(g *Game) (int, int)
}

type Chapter struct {
	Title  string
	Sub    string
	Intro  []Dialog
	Goals  []Objective
	Outro  []Dialog
	Reward int
	Start  func(g *Game)
}

func countConn(g *Game, t int) int {
	n := 0
	for _, b := range g.B {
		if b.T == t {
			n++
		}
	}
	return n
}

func energySurplus(g *Game) int {
	best := -9999
	for _, I := range g.Islands {
		if I.Owned && I.EProd-I.EUse > best {
			best = I.EProd - I.EUse
		}
	}
	return best
}

var Chapters = []Chapter{
	{
		Title: "Kapitel I", Sub: "Ankunft",
		Intro: []Dialog{
			{SpVoss, "Gouverneur, willkommen im Nova-Archipel. Die alten Kontinente sind überflutet – diese Inseln sind unsere neue Chance."},
			{SpVoss, "Ihr Frachtgleiter „Aurora I“ liegt vor Neu-Aurora. Gründen Sie an der Küste einen Kontor, damit die ersten Siedler landen können."},
			{SpLin, "Wir sind nach der langen Reise erschöpft. Ein Dach über dem Kopf, Nahrung und sauberes Wasser – mehr brauchen wir zunächst nicht."},
		},
		Goals: []Objective{
			{"Kontor auf Neu-Aurora gründen", func(g *Game) (int, int) { return imin(1, countConn(g, BKontor)), 1 }},
			{"10 Wohnmodule bauen", func(g *Game) (int, int) { return countConn(g, BHouse), 10 }},
			{"Windturbinen: Energieüberschuss von 5", func(g *Game) (int, int) { return imax(0, energySurplus(g)), 5 }},
			{"Nahrungsfabrik und Wasserwerk errichten", func(g *Game) (int, int) {
				return imin(1, countConn(g, BNahrung)) + imin(1, countConn(g, BWasser)), 2
			}},
			{"80 Arbeiter ansiedeln", func(g *Game) (int, int) { return g.MaxPop[TArbeiter], 80 }},
		},
		Outro: []Dialog{
			{SpLin, "Die Siedlung lebt! Die Kinder spielen am Strand und die Algenbecken leuchten. Danke, Gouverneur."},
			{SpVoss, "Ein guter Anfang. Aber eine Kolonie, die nur überlebt, ist noch keine Zukunft."},
		},
		Reward: 4000,
	},
	{
		Title: "Kapitel II", Sub: "Funken des Fortschritts",
		Intro: []Dialog{
			{SpKern, "Elias Kern, Chefingenieur. Ich habe Pläne für Erzminen und Stahlwerke – aber dafür brauche ich ausgebildete Techniker."},
			{SpKern, "Versorgen Sie die Arbeiter vollständig, dann steigen sie auf. Ein Komm-Zentrum vernetzt die Stadt, ein Medizentrum hält die Techniker gesund."},
		},
		Goals: []Objective{
			{"Komm-Zentrum bauen", func(g *Game) (int, int) { return imin(1, countConn(g, BKomm)), 1 }},
			{"150 Arbeiter erreichen", func(g *Game) (int, int) { return g.MaxPop[TArbeiter], 150 }},
			{"40 Techniker erreichen", func(g *Game) (int, int) { return g.MaxPop[TTechniker], 40 }},
			{"Medizentrum bauen", func(g *Game) (int, int) { return imin(1, countConn(g, BMedi)), 1 }},
			{"Stahlwerk in Betrieb nehmen", func(g *Game) (int, int) { return imin(1, countConn(g, BStahl)), 1 }},
		},
		Outro: []Dialog{
			{SpKern, "Hören Sie das? Das Summen der Fabriken. Wir bauen wieder, Gouverneur – mit unseren eigenen Händen."},
		},
		Reward: 6000,
	},
	{
		Title: "Kapitel III", Sub: "Jenseits des Horizonts",
		Intro: []Dialog{
			{SpOrion, "Kapitän Orion von der Händlergilde. Ihre Techniker wollen Synthkaffee – und der wächst nicht auf Neu-Aurora."},
			{SpOrion, "Korallenspitze im Nordosten hat fruchtbaren Boden für Kaffee. Bauen Sie eine Werft, schicken Sie einen Gleiter hinüber und verbinden Sie die Inseln."},
		},
		Goals: []Objective{
			{"Werft errichten", func(g *Game) (int, int) { return imin(1, countConn(g, BWerft)), 1 }},
			{"Zweiten Kontor auf einer anderen Insel gründen", func(g *Game) (int, int) { return countConn(g, BKontor), 2 }},
			{"Kaffeeplantage bauen", func(g *Game) (int, int) { return imin(1, countConn(g, BKaffeePl)), 1 }},
			{"Aktive Handelsroute einrichten", func(g *Game) (int, int) {
				for _, s := range g.Ships {
					if s.Route.Active {
						return 1, 1
					}
				}
				return 0, 1
			}},
		},
		Outro: []Dialog{
			{SpOrion, "Die Frachtgleiter fliegen, die Lager füllen sich. Die Gilde wird Ihren Namen nicht vergessen."},
			{SpVoss, "Gouverneur … unsere Wetterbojen melden etwas Großes. Bereiten Sie sich vor."},
		},
		Reward: 8000,
	},
	{
		Title: "Kapitel IV", Sub: "Der Sturm",
		Intro: []Dialog{
			{SpVoss, "Ein Superzellen-Sturm trifft das Archipel! Die Windturbinen laufen am Limit, aber Blitze und Regen setzen der Stadt zu."},
			{SpKern, "Wir brauchen eine stabile Energieversorgung, die nicht vom Wetter abhängt. Solarparks und eine solide Reserve – und die Kasse muss stimmen."},
		},
		Goals: []Objective{
			{"Solarpark bauen", func(g *Game) (int, int) { return imin(1, countConn(g, BSolar)), 1 }},
			{"Energieüberschuss von 30", func(g *Game) (int, int) { return imax(0, energySurplus(g)), 30 }},
			{"120 Techniker erreichen", func(g *Game) (int, int) { return g.MaxPop[TTechniker], 120 }},
			{"Positive Bilanz von +200/min", func(g *Game) (int, int) { return imax(0, int(g.Balance)), 200 }},
		},
		Outro: []Dialog{
			{SpLin, "Der Sturm ist vorbei und kein Licht ist ausgegangen. Die Leute nennen Sie schon „Sturmbrecher“."},
		},
		Reward: 10000,
		Start: func(g *Game) {
			g.chapterT = 0
			g.Credits -= 2000
			g.Notify("Sturmschäden: −2.000 Credits", cRed)
		},
	},
	{
		Title: "Kapitel V", Sub: "Stadt der Zukunft",
		Intro: []Dialog{
			{SpKern, "Die ersten Ingenieure sind da! Mit Mikrochips und Holo-Pads können wir endlich die Fusion zünden."},
			{SpVoss, "Machen Sie Neu-Aurora zum Leuchtturm des Archipels: Forschung, Fusionsenergie und eine Metropole aus Glas und Licht."},
		},
		Goals: []Objective{
			{"Forschungslabor bauen", func(g *Game) (int, int) { return imin(1, countConn(g, BLabor)), 1 }},
			{"Fusionsreaktor in Betrieb nehmen", func(g *Game) (int, int) { return imin(1, countConn(g, BFusion)), 1 }},
			{"150 Ingenieure erreichen", func(g *Game) (int, int) { return g.MaxPop[TIngenieure], 150 }},
		},
		Outro: []Dialog{
			{SpVoss, "Gouverneur … sehen Sie hinaus. Die Lichter von Neu-Aurora sieht man bis zum Horizont."},
			{SpKern, "Wir haben aus dem Nichts eine Zukunft gebaut."},
			{SpLin, "Und sie gehört uns allen. Danke."},
		},
		Reward: 25000,
	},
}

// ---------------------------------------------------------------- Ablauf

func NewCampaign() *Game {
	MapOpt = MapOptions{Size: 1, Islands: 9}
	g := NewGame(7)
	g.Mode = ModeCampaign
	g.Notifs = nil
	g.startChapter(0)
	return g
}

func (g *Game) startChapter(i int) {
	g.Chapter = i
	g.chapterT = 0
	if i >= len(Chapters) {
		g.Won = true
		return
	}
	ch := &Chapters[i]
	g.Dialogs = append(g.Dialogs, ch.Intro...)
	if i == 2 && len(g.Rivals) == 0 {
		g.Dialogs = append(g.Dialogs, Dialog{SpVoss, "Achtung, Gouverneur: Unsere Sensoren melden fremde Kolonien im Archipel. Wir sind nicht mehr allein – behalte die Konkurrenz im Auge."})
		g.InitRivals(imax(2, Opt.Rivals))
	}
	g.Banner = 5
	if ch.Start != nil {
		ch.Start(g)
	}
}

func (g *Game) DialogOpen() bool { return len(g.Dialogs) > 0 }

func (g *Game) checkCampaign(dt float64) {
	if g.Mode != ModeCampaign || g.Chapter >= len(Chapters) || g.DialogOpen() {
		return
	}
	g.chapterT += dt
	ch := &Chapters[g.Chapter]
	for _, o := range ch.Goals {
		a, b := o.Check(g)
		if a < b {
			return
		}
	}
	g.Credits += float64(ch.Reward)
	PlaySfx(SfxQuest)
	g.Notify(Tf("%s abgeschlossen! (+%s Credits)", ch.Title, fmtNum(ch.Reward)), cGold)
	g.Dialogs = append(g.Dialogs, ch.Outro...)
	g.startChapter(g.Chapter + 1)
}

// Kampagnen-Zielpanel (ersetzt die Aufträge)
func (g *Game) drawCampaignGoals(c *Canvas) {
	u := g.UI
	if g.Chapter >= len(Chapters) {
		return
	}
	ch := &Chapters[g.Chapter]
	x, y, w := 12, 70, 320
	h := 66 + len(ch.Goals)*26
	u.panel(c, x, y, w, h)
	c.Text(FSmall, x+16, y+10, T(ch.Title)+T(" · KAMPAGNE"), cAccent)
	c.TextS(FSerif, x+16, y+26, ch.Sub, cText)
	yy := y + 60
	for _, o := range ch.Goals {
		a, b := o.Check(g)
		done := a >= b
		col := cText
		if done {
			col = cGreen
			c.Circle(float64(x+24), float64(yy+9), 7, cGreen, 255)
			c.Line(float64(x+20), float64(yy+9), float64(x+23), float64(yy+12), RGB{10, 40, 20}, 255)
			c.Line(float64(x+23), float64(yy+12), float64(x+28), float64(yy+5), RGB{10, 40, 20}, 255)
		} else {
			c.RoundRectLine(x+17, yy+2, 14, 14, 7, cDim, 200)
		}
		t := o.Text
		for TextW(FSmall, t) > w-110 && len(t) > 4 {
			t = t[:len(t)-1]
		}
		c.Text(FSmall, x+40, yy+1, t, col)
		c.TextR(FSmall, x+w-14, yy+1, Tf("%d/%d", imin(a, b), b), cDim)
		yy += 26
	}
}

// Dialogfenster mit Porträt
func (g *Game) drawDialog(c *Canvas) {
	if !g.DialogOpen() {
		return
	}
	u := g.UI
	d := g.Dialogs[0]
	c.FillRect(0, 0, c.W, c.H, RGB{0, 0, 0}, 70)
	w := imin(820, c.W-80)
	h := 190
	x := c.W/2 - w/2
	y := c.H - h - 110
	u.panel(c, x, y, w, h)
	// Porträt
	px, py := float64(x+90), float64(y+95)
	c.Circle(px, py, 62, RGB{8, 12, 20}, 230)
	c.Circle(px, py, 58, d.Sp.Col.Mul(0.35), 255)
	c.Circle(px, py-14, 20, d.Sp.Col.Mix(RGB{255, 255, 255}, 0.4), 255)
	for yy := 0; yy < 34; yy++ {
		hw := 18 + float64(yy)*0.9
		c.FillRect(int(px-hw), int(py+10)+yy, int(hw*2), 1, d.Sp.Col.Mix(RGB{255, 255, 255}, 0.25), 255)
	}
	c.SetClip(x, y, x+w, y+h)
	c.ResetClip()
	for a := 0; a < 360; a += 2 {
		rad := float64(a) * math.Pi / 180
		c.Plot(int(px+math.Cos(rad)*60), int(py+math.Sin(rad)*60), d.Sp.Col.U(), 255)
	}
	c.TextS(FSerif, x+175, y+22, d.Sp.Name, d.Sp.Col)
	c.Text(FSmall, x+175, y+48, d.Sp.Role, cDim)
	for i, l := range WrapText(FNormal, d.Text, w-210) {
		c.Text(FNormal, x+175, y+76+i*22, l, cText)
	}
	if u.button(c, x+w-150, y+h-48, 130, 34, "Weiter", true) || In.KeyHit[KSpace] {
		g.Dialogs = g.Dialogs[1:]
		PlaySfx(SfxClick)
	}
	if g.Banner > 0 && g.Chapter < len(Chapters) {
		ch := &Chapters[g.Chapter]
		a := int(clampf(g.Banner, 0, 1) * 255)
		c.TextA(FBig, c.W/2-TextW(FBig, ch.Title)/2, 130, ch.Title, cAccent, a)
		c.TextA(FTitle, c.W/2-TextW(FTitle, ch.Sub)/2, 160, ch.Sub, RGB{250, 252, 255}, a)
	}
}
