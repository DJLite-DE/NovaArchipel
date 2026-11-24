package game

import (
	"fmt"
	"math"
	"time"
)

const Version = "3.0"

type App struct {
	Cur     *Game
	Demo    *Game
	InMenu  bool
	ToMenu  bool
	mu      *UIState
	t       float64
	msg     string
	R       *Renderer
	updNote bool
	Slots   SlotDialog
	Setup   SetupState
	LoadReq string
}

var Main *App

func NewApp(r *Renderer) *App {
	a := &App{InMenu: true, mu: NewUIState(), R: r}
	a.Demo = BuildDemo(7, 3)
	a.Demo.UI.Screen = ScrMenu
	Main = a
	return a
}

// aktuell darzustellendes Spiel
func (a *App) Active() *Game {
	if !a.InMenu && a.Cur != nil {
		return a.Cur
	}
	return a.Demo
}

// Glasflächen (Panels) des aktuellen Frames
func (a *App) Rects() [][4]int {
	if !a.InMenu && a.Cur != nil {
		return a.Cur.UI.panels
	}
	return a.mu.panels
}

// UI zeichnen und Spiel fortschreiben; danach Render() aufrufen
func (a *App) Frame(c *Canvas, dt float64) {
	a.t += dt
	if a.ToMenu {
		a.ToMenu = false
		a.InMenu = true
	}
	if !a.InMenu && a.Cur != nil {
		if !a.updNote && Upd.get().State == UpdAvailable {
			a.updNote = true
			a.Cur.Notify("Ein Update ist verfügbar – installiere es im Hauptmenü.", cAccent)
		}
		if a.LoadReq != "" {
			slot := a.LoadReq
			a.LoadReq = ""
			if ng, err := LoadSlot(slot); err == nil {
				a.Cur = ng
				ng.Notify("Schnellspeicher geladen.", cGreen)
			} else {
				a.Cur.Notify("Kein Schnellspeicher vorhanden (F5 speichert).", cRed)
			}
		}
		a.Cur.Frame(c, dt)
		if a.Slots.Open {
			u := a.Cur.UI
			u.modalDraw = true
			if ng := a.Slots.Draw(c, u, a.Cur); ng != nil {
				a.Cur = ng
			}
			u.modalDraw = false
		}
		return
	}
	a.menuFrame(c, dt)
}

// 3D-Szene mit UI-Ebene ausgeben
func (a *App) Render(c *Canvas, dt float64) {
	if a.R == nil {
		return
	}
	a.R.Render(a.Active(), c, a.Rects(), true, dt)
	if g := a.Cur; g != nil && g.thumbReq != "" && !a.InMenu {
		writeThumb(g.thumbReq, a.R.ReadPixels(), a.R.OutW, a.R.OutH)
		g.thumbReq = ""
	}
}

func (a *App) menuFrame(c *Canvas, dt float64) {
	d := a.Demo
	d.UI.Screen = ScrGame
	d.Update(dt)
	d.UI.Screen = ScrMenu
	// langsame Kamerafahrt um die Stadt
	kx, ky := d.CityCenter()
	d.UI.TX, d.UI.TZ = kx+2, ky+2
	d.UI.Yaw = math.Pi/4 + a.t*0.03
	d.UI.Dist = 30
	d.UI.world = d.W
	d.UI.UpdateCam(d, c.W, c.H)
	c.Clear(RGB{})
	// Verlauf links für Lesbarkeit
	for x := 0; x < imin(c.W, 760); x++ {
		al := int(200 * math.Pow(1-float64(x)/760, 1.6))
		c.FillRect(x, 0, 1, c.H, RGB{6, 10, 18}, al)
	}
	u := a.mu
	u.W, u.H = c.W, c.H
	u.MX, u.MY = In.MX, In.MY
	u.consumed = false
	u.blockBase = u.ShowOptions
	u.modalDraw = false
	u.panels = u.panels[:0]
	x := 90
	y := c.H/2 - 290
	c.Text(FBold, x+2, y, "S T R A T E G I E   ·   A U F B A U", cAccent)
	c.TextA(FTitle, x+2, y+26, "NOVA ARCHIPEL", RGB{0, 0, 0}, 90)
	c.Text(FTitle, x, y+24, "NOVA ARCHIPEL", RGB{250, 252, 255})
	for i := 0; i < 480; i++ {
		t := float64(i) / 480
		c.FillRect(x+2+i, y+118, 1, 3, cAccent, int(255*(1-t)))
	}
	c.Text(FLight, x+2, y+128, "Kolonien der Zukunft · 2310", RGB{210, 225, 240})
	by := y + 200
	bw := 320
	btn := func(label string, ok bool) bool {
		u.panels = append(u.panels, [4]int{x, by, bw, 52})
		hov := u.inside(x, by, bw, 52) && ok
		c.RoundRect(x, by, bw, 52, 10, RGB{10, 16, 26}, 90)
		if hov {
			c.RoundRect(x, by, bw, 52, 10, uiPurple, 150)
			c.FillRect(x+10, by+50, bw-20, 2, cAccent, 255)
		}
		c.RoundRectLine(x, by, bw, 52, 10, RGB{255, 255, 255}, 50)
		col := cText
		if !ok {
			col = RGB{110, 120, 135}
		}
		c.Text(FSerif, x+26, by+13, label, col)
		c.Text(FBold, x+bw-36, by+14, ">", col)
		r := ok && !u.ShowOptions && u.clicked(x, by, bw, 52)
		if r {
			PlaySfx(SfxClick)
		}
		by += 64
		return r
	}
	if a.Cur != nil {
		if btn("Fortsetzen", true) {
			a.InMenu = false
		}
	}
	if btn("Kampagne", true) {
		a.Cur = NewCampaign()
		a.InMenu = false
	}
	if btn("Endlosspiel", true) {
		a.Setup.Open = true
		if a.Setup.Seed == 0 {
			a.Setup = SetupState{Open: true, Seed: 1000 + int(a.t*7919)%90000, Diff: 1, Size: 1, Islands: 9, Rivals: Opt.Rivals}
		}
	}
	if btn("Tutorial", true) {
		a.Cur = NewTutorial()
		a.InMenu = false
	}
	if btn("Spielstand laden", AnySave()) {
		a.Slots.Show(false)
	}
	if btn("Optionen", true) {
		u.ShowOptions = true
	}
	if btn("Erfolge", true) {
		u.ShowAch = true
	}
	if btn("Beenden", true) {
		In.Quit = true
	}
	if a.msg != "" {
		c.Text(FNormal, x, by+10, a.msg, cRed)
		by += 30
	}
	a.drawUpdateCard(c, x, by+14)
	if u.ShowOptions {
		u.modalDraw = true
		DrawOptions(c, u, func() { u.ShowOptions = false })
		u.modalDraw = false
	}
	if u.ShowAch {
		u.modalDraw = true
		drawAchievements(c, u, func() { u.ShowAch = false })
		u.modalDraw = false
	}
	if a.Setup.Open {
		u.modalDraw = true
		if ng := a.Setup.Draw(c, u); ng != nil {
			a.Cur = ng
			a.InMenu = false
		}
		u.modalDraw = false
	}
	if a.Slots.Open {
		u.modalDraw = true
		if ng := a.Slots.Draw(c, u, nil); ng != nil {
			a.Cur = ng
			a.InMenu = false
		}
		u.modalDraw = false
	}
	c.Text(FSmall, x, c.H-60, "Baue Kolonien, verknüpfe Produktionsketten und führe deine Bürger", cDim)
	c.Text(FSmall, x, c.H-44, T("vom Arbeiter bis zum Ingenieur.   Version ")+Version+T("  ·  F11 Vollbild"), cDim)
	c.Text(FSmall, x, c.H-28, "Teile der 3D-Modelle: Kenney (www.kenney.nl, CC0)", RGB{110, 125, 145})
}

// Ladebildschirm
func DrawLoading(c *Canvas) {
	c.GradRect(0, 0, c.W, c.H, RGB{10, 18, 30}, RGB{4, 8, 14}, 255)
	c.TextC(FTitle, c.W/2, c.H/2-70, "NOVA ARCHIPEL", RGB{250, 252, 255})
	for i := 0; i < 400; i++ {
		t := math.Abs(float64(i)/400*2 - 1)
		c.FillRect(c.W/2-200+i, c.H/2+20, 1, 2, cAccent, int(255*(1-t)))
	}
	c.TextC(FLight, c.W/2, c.H/2+36, "Die Inselwelt wird erschaffen …", RGB{210, 225, 240})
}

// Fenster-Icon
func AppIcon(n int) []uint32 {
	p := NewPainter(n, n, 4)
	f := float64(n)
	p.Ellipse(f/2, f/2, f*0.48, f*0.48, hex(0x0e1a2a), 1)
	p.Ellipse(f/2, f*0.66, f*0.34, f*0.14, hex(0x1e8fb8), 1)
	p.Poly([]P{{f * .38, f * .64}, {f * .38, f * .3}, {f * .5, f * .16}, {f * .62, f * .3}, {f * .62, f * .64}}, hex(0xeef2f5), 1)
	p.Poly([]P{{f * .41, f * .34}, {f * .59, f * .34}, {f * .59, f * .6}, {f * .41, f * .6}}, hex(0x2a4a68), 1)
	p.Poly([]P{{f * .38, f * .44}, {f * .62, f * .44}, {f * .62, f * .47}, {f * .38, f * .47}}, hex(0x5ef2ff), 1)
	s := p.ToSprite(0, 0)
	return s.Pix
}

func (a *App) OpenOptions(tab int) { a.mu.ShowOptions = true; a.mu.optTab = tab }

// ---------------------------------------------------------------- Neues Spiel einrichten

type SetupState struct {
	Open      bool
	Seed      int
	Diff      int
	Size      int // 0 klein, 1 normal, 2 groß
	Rich      int // 0 knapp, 1 normal, 2 reich
	Islands   int
	Rivals    int
	NoPirates bool
	NoDisast  bool
}

func (s *SetupState) Draw(c *Canvas, u *UIState) *Game {
	c.FillRect(0, 0, c.W, c.H, RGB{2, 6, 12}, 170)
	pw, ph := 640, 600
	px, py := c.W/2-pw/2, c.H/2-ph/2
	u.panel(c, px, py, pw, ph)
	c.Text(FSmall, px+28, py+18, "ENDLOSSPIEL", cAccent)
	c.Text(FBig, px+28, py+36, "Neues Spiel einrichten", cText)
	y := py + 100
	row := func(label string, val string, dec, inc func(), tip string) {
		c.Text(FNormal, px+28, y+6, label, cText)
		if u.button(c, px+300, y, 34, 34, "<", true) {
			dec()
		}
		c.RoundRect(px+340, y, 210, 34, 8, RGB{10, 18, 32}, 220)
		c.TextC(FBold, px+445, y+6, val, cAccent)
		if u.button(c, px+556, y, 34, 34, ">", true) {
			inc()
		}
		if tip != "" && u.inside(px+28, y, 260, 34) {
			u.setTip(label, tip)
		}
		y += 48
	}
	cyc := func(v *int, n, d int) func() { return func() { *v = (*v + d + n) % n } }
	row("Schwierigkeit", DiffNames[s.Diff], cyc(&s.Diff, 3, -1), cyc(&s.Diff, 3, 1),
		"Entspannt: mehr Startgeld, weniger Unterhalt, friedlichere Konkurrenz. Profi: das Gegenteil.")
	row("Inselgröße", []string{"Klein", "Normal", "Groß"}[s.Size], cyc(&s.Size, 3, -1), cyc(&s.Size, 3, 1), "")
	row("Rohstoffe", []string{"Knapp", "Normal", "Reich"}[s.Rich], cyc(&s.Rich, 3, -1), cyc(&s.Rich, 3, 1), "Anzahl der Erz- und Siliziumvorkommen")
	row("Anzahl Inseln", fmt.Sprint(s.Islands), func() { s.Islands = imax(5, s.Islands-1) }, func() { s.Islands = imin(9, s.Islands+1) }, "")
	row("KI-Gegner", []string{"Keine", "1", "2", "3"}[s.Rivals], cyc(&s.Rivals, 4, -1), cyc(&s.Rivals, 4, 1), "")
	onoff := func(b bool) string {
		if b {
			return "Aus"
		}
		return "An"
	}
	row("Piraten", onoff(s.NoPirates), func() { s.NoPirates = !s.NoPirates }, func() { s.NoPirates = !s.NoPirates }, "")
	row("Brände & Stromausfälle", onoff(s.NoDisast), func() { s.NoDisast = !s.NoDisast }, func() { s.NoDisast = !s.NoDisast }, "")
	row("Karten-Seed", fmt.Sprint(s.Seed), func() { s.Seed = imax(1, s.Seed-1) }, func() { s.Seed++ }, "Gleicher Seed = gleiche Karte")
	if u.button(c, px+28, py+ph-60, 140, 40, "Zufällig", true) {
		s.Seed = 1000 + int(time.Now().UnixNano()/1e6)%90000
	}
	if u.button(c, px+pw-330, py+ph-60, 140, 40, "Zurück", true) || In.KeyHit[KEsc] {
		s.Open = false
		In.KeyHit[KEsc] = false
		return nil
	}
	if u.button(c, px+pw-180, py+ph-60, 152, 40, "Spiel starten", true) {
		s.Open = false
		Opt.Pirates, Opt.Disasters = !s.NoPirates, !s.NoDisast
		mo := MapOptions{Size: []float64{0.85, 1, 1.15}[s.Size], Rich: s.Rich - 1, Islands: s.Islands}
		PlaySfx(SfxQuest)
		return NewCustomGame(s.Seed, mo, s.Diff, s.Rivals)
	}
	return nil
}
