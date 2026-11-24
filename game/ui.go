package game

import (
	"math"
	"strings"
)

const (
	ScrMenu = iota
	ScrGame
)

const ToolDemolish = 100

type UIState struct {
	Screen      int
	TX, TZ      float64 // Kameraziel (Kachelkoordinaten)
	Dist        float64
	Yaw         float64
	camTY       float64
	Eye         V3
	VP, InvVP   Mat4
	Proj        Mat4
	InvProj     Mat4
	world       *World
	Zoom        int
	Anim        float64
	MX, MY      int
	Tool        int
	Cat         int
	SelB        int
	SelShip     int
	HoverX      int
	HoverY      int
	RoadDrag    bool
	RSX, RSY    int
	OverUI      bool
	panels      [][4]int
	tip         []string
	consumed    bool
	PauseMenu   bool
	mini        *Sprite
	miniT       float64
	lastMX      int
	lastMY      int
	rdragged    bool
	W, H        int
	lastPlace   [2]int
	prevSpeed   int
	WinShown    bool
	ShowOptions bool
	blockBase   bool
	modalDraw   bool
	optTab      int
	dragSlider  int
	ShowHelp    bool
	ShowDiplo   bool
	ShowStats   bool
	ShowResearch bool
	ShowAch bool
	Overlay int
	tabRect [NumCats][4]int
	itemRect map[int][4]int
	statsRect [4]int
	StatTab     int
	BlueMode    bool
	SelBP       int
}

func NewUIState() *UIState {
	return &UIState{Screen: ScrGame, Zoom: 2, Dist: 30, Yaw: math.Pi / 4, Tool: -1, Cat: -1, prevSpeed: 1, lastPlace: [2]int{-99, -99}, itemRect: map[int][4]int{}}
}

// ------------------------------------------------------------ Farben/Helfer

var (
	cText   = RGB{240, 245, 250}
	cDim    = RGB{160, 176, 196}
	cAccent = RGB{94, 212, 255}
	cGold   = RGB{255, 200, 90}
	cRed    = RGB{255, 110, 120}
	cGreen  = RGB{110, 230, 160}
)

func fmtNum(v int) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := itoa(v)
	var out []byte
	for i, ch := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, ch)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func (u *UIState) inside(x, y, w, h int) bool {
	return u.MX >= x && u.MY >= y && u.MX < x+w && u.MY < y+h
}

func (u *UIState) panel(c *Canvas, x, y, w, h int) {
	c.Panel(x, y, w, h)
	u.panels = append(u.panels, [4]int{x, y, w, h})
}

func (u *UIState) clicked(x, y, w, h int) bool {
	if u.consumed || (u.blockBase && !u.modalDraw) {
		return false
	}
	if In.LReleased && u.inside(x, y, w, h) && In.PressX >= x && In.PressY >= y && In.PressX < x+w && In.PressY < y+h {
		u.consumed = true
		return true
	}
	return false
}

func (u *UIState) button(c *Canvas, x, y, w, h int, label string, enabled bool) bool {
	hov := u.inside(x, y, w, h) && enabled
	bg := uiNavyL
	if hov {
		bg = uiPurple
		if In.LDown {
			bg = uiPurpleL
		}
	}
	if !enabled {
		bg = RGB{26, 32, 44}
	}
	c.Fabric(x, y, w, h, bg, 245)
	bc := uiGoldD
	if hov {
		bc = uiGold
	}
	if !enabled {
		bc = RGB{60, 64, 74}
	}
	c.Rect(x, y, w, h, bc, 255)
	c.FillRect(x+1, y+1, w-2, 1, RGB{255, 255, 255}, 25)
	tc := cText
	if !enabled {
		tc = RGB{100, 110, 125}
	}
	f := FNormal
	if h >= 40 {
		f = FSerif
	}
	c.TextC(f, x+w/2, y+(h-Fonts[f].LineH)/2, label, tc)
	if enabled && u.clicked(x, y, w, h) {
		PlaySfx(SfxClick)
		return true
	}
	return false
}

func (u *UIState) setTip(lines ...string) { u.tip = lines }

func bar(c *Canvas, x, y, w, h int, f float64, col RGB) {
	c.FillRect(x, y, w, h, RGB{8, 14, 28}, 230)
	c.FillRect(x, y, int(float64(w)*clampf(f, 0, 1)), h, col, 255)
	c.FillRect(x, y, int(float64(w)*clampf(f, 0, 1)), 1, RGB{255, 255, 255}, 70)
	c.Rect(x, y, w, h, uiGoldD, 200)
}

// ------------------------------------------------------------ Frame

func (g *Game) Frame(c *Canvas, dt float64) {
	u := g.UI
	u.W, u.H = c.W, c.H
	u.MX, u.MY = In.MX, In.MY
	// OverUI bestimmen aus Panels des letzten Frames
	u.OverUI = false
	for _, p := range u.panels {
		if u.inside(p[0], p[1], p[2], p[3]) {
			u.OverUI = true
		}
	}
	u.panels = u.panels[:0]
	u.tip = nil
	u.consumed = false
	u.blockBase = u.ShowOptions || g.DialogOpen() || u.PauseMenu || u.ShowDiplo || u.ShowStats || u.ShowResearch || u.ShowAch || g.Event0 != nil || (Main != nil && Main.Slots.Open)
	u.modalDraw = false
	u.world = g.W
	g.cameraInput(dt)
	u.UpdateCam(g, c.W, c.H)
	wx, wy := u.ScreenToWorld(float64(u.MX), float64(u.MY), float64(c.W), float64(c.H))
	u.HoverX, u.HoverY = int(math.Floor(wx)), int(math.Floor(wy))
	g.Update(dt)
	c.Clear(RGB{})
	g.DrawWorldOverlay(c)
	g.drawHUD(c)
	if !u.consumed && !u.OverUI && !u.PauseMenu && !u.ShowOptions && !u.ShowDiplo && !u.ShowStats && !u.ShowResearch && !u.ShowAch && g.Event0 == nil && !(Main != nil && Main.Slots.Open) && !g.DialogOpen() {
		g.worldInput()
	}
	g.drawTooltip(c)
}

func (g *Game) cameraInput(dt float64) {
	u := g.UI
	fwd := V3{-math.Sin(u.Yaw), 0, -math.Cos(u.Yaw)}
	right := V3{math.Cos(u.Yaw), 0, -math.Sin(u.Yaw)}
	sp := u.Dist * 1.1 * dt * Opt.CamSpeed
	mv := V3{}
	if In.Keys[KLeft] || In.Keys[KA] {
		mv = mv.Sub(right)
	}
	if In.Keys[KRight] || In.Keys[KD] {
		mv = mv.Add(right)
	}
	if In.Keys[KUp] || In.Keys[KW] {
		mv = mv.Add(fwd)
	}
	if In.Keys[KDown] || In.Keys[KS] {
		mv = mv.Sub(fwd)
	}
	u.TX += mv.X * sp
	u.TZ += mv.Z * sp
	if In.Keys[KQ] {
		u.Yaw -= dt * 1.6
	}
	if In.Keys[KE] {
		u.Yaw += dt * 1.6
	}
	// Ziehen: rechte Taste verschiebt, mittlere Taste dreht
	if In.RPressed {
		u.rdragged = false
	}
	dx, dy := float64(In.MX-u.lastMX), float64(In.MY-u.lastMY)
	if In.RDown && (dx != 0 || dy != 0) && abs(In.MX-In.RPressX)+abs(In.MY-In.RPressY) > 6 {
		u.rdragged = true
		k := u.Dist * 0.0016
		u.TX += (fwd.X*dy - right.X*dx) * k
		u.TZ += (fwd.Z*dy - right.Z*dx) * k
	}
	if In.MDown && dx != 0 {
		u.Yaw -= dx * 0.006
	}
	u.lastMX, u.lastMY = In.MX, In.MY
	if In.Wheel != 0 && !u.OverUI {
		u.Dist *= math.Pow(0.87, float64(In.Wheel))
	}
	if In.KeyHit[KPlus] {
		u.Dist *= 0.8
	}
	if In.KeyHit[KMinus] {
		u.Dist *= 1.25
	}
	u.Dist = clampf(u.Dist, 7, []float64{70, 95, 130}[clampi(Opt.ViewDist, 0, 2)])
	n := float64(g.W.N)
	u.TX = clampf(u.TX, 0, n)
	u.TZ = clampf(u.TZ, 0, n)
	// Tastatur-Kürzel
	if In.KeyHit[KEsc] && !u.ShowOptions && !g.DialogOpen() {
		if u.Tool >= 0 {
			u.Tool = -1
			u.RoadDrag = false
		} else if u.Cat >= 0 {
			u.Cat = -1
		} else if u.SelB != 0 || u.SelShip != 0 {
			u.SelB, u.SelShip = 0, 0
		} else {
			u.PauseMenu = !u.PauseMenu
		}
	}
	if In.KeyHit[KSpace] {
		if g.Speed == 0 {
			g.Speed = imax(1, u.prevSpeed)
		} else {
			u.prevSpeed = g.Speed
			g.Speed = 0
		}
	}
	for k, s := range map[int]int{K1: 1, K2: 2, K3: 4, K4: 8} {
		if In.KeyHit[k] {
			g.Speed = s
		}
	}
	if In.KeyHit[KDel] {
		u.Tool = ToolDemolish
	}
	if In.KeyHit[KF5] {
		if err := g.SaveSlot("schnell", "Schnellspeicher"); err == nil {
			g.Notify("Schnell gespeichert (F9 lädt).", cGreen)
		} else {
			g.Notify(T("Speichern fehlgeschlagen: ")+err.Error(), cRed)
		}
	}
	if In.KeyHit[KZ] && In.Keys[KCtrl] {
		if g.Undo() {
			PlaySfx(SfxDemolish)
			g.Notify("Rückgängig gemacht.", cDim)
		}
	}
	if In.KeyHit[KO] {
		u.Overlay = (u.Overlay + 1) % len(OverlayNames)
		if u.Overlay > 0 {
			g.Notify(T("Überlagerung: ")+OverlayNames[u.Overlay]+T(" (O wechselt)"), cAccent)
		}
	}
	if In.KeyHit[KF9] && Main != nil {
		Main.LoadReq = "schnell"
	}
	if In.KeyHit[KH] {
		u.ShowHelp = !u.ShowHelp
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (u *UIState) RoadPreview() [][2]int {
	if !u.RoadDrag {
		return [][2]int{{u.HoverX, u.HoverY}}
	}
	var out [][2]int
	x, y := u.RSX, u.RSY
	dx, dy := 1, 1
	if u.HoverX < x {
		dx = -1
	}
	if u.HoverY < y {
		dy = -1
	}
	for {
		out = append(out, [2]int{x, y})
		if x == u.HoverX {
			break
		}
		x += dx
	}
	for y != u.HoverY {
		y += dy
		out = append(out, [2]int{x, y})
	}
	return out
}

func (g *Game) worldInput() {
	u := g.UI
	hx, hy := u.HoverX, u.HoverY
	switch {
	case u.Tool == BRoad:
		if In.LPressed {
			u.RoadDrag = true
			u.RSX, u.RSY = hx, hy
		}
		if In.LReleased && u.RoadDrag {
			n := 0
			for _, p := range u.RoadPreview() {
				if g.CanPlace(BRoad, p[0], p[1]).OK {
					g.forcePlace(BRoad, p[0], p[1], nil)
					n++
				}
			}
			if n > 0 {
				PlaySfx(SfxRoad)
			}
			u.RoadDrag = false
		}
	case u.Tool == ToolDemolish:
		if In.LPressed || (In.LDown && (hx != u.lastPlace[0] || hy != u.lastPlace[1])) {
			if g.Demolish(hx, hy) {
				PlaySfx(SfxDemolish)
			}
			u.lastPlace = [2]int{hx, hy}
		}
	case u.Tool >= 0:
		bt := &BTypes[u.Tool]
		x, y := hx-bt.W/2, hy-bt.H/2
		if u.BlueMode && u.Tool != BKontor && (In.LPressed || (In.LDown && (x != u.lastPlace[0] || y != u.lastPlace[1]))) {
			pc := g.PlaceBlueprint(u.Tool, x, y)
			if pc.OK {
				PlaySfx(SfxClick)
			} else if In.LPressed {
				PlaySfx(SfxError)
				g.Notify(pc.Reason, cRed)
			}
			u.lastPlace = [2]int{x, y}
		} else if In.LPressed || (In.LDown && (x != u.lastPlace[0] || y != u.lastPlace[1])) {
			pc := g.CanPlace(u.Tool, x, y)
			if pc.OK {
				g.forcePlace(u.Tool, x, y, pc.Ship)
				PlaySfx(SfxBuild)
				u.lastPlace = [2]int{x, y}
				if u.Tool == BKontor {
					u.Tool = -1
				}
			} else if In.LPressed {
				PlaySfx(SfxError)
				g.Notify(pc.Reason, cRed)
				u.lastPlace = [2]int{x, y}
			}
		}
	default:
		if In.LReleased && abs(In.MX-In.PressX)+abs(In.MY-In.PressY) < 6 {
			// Schiff?
			best, bd := (*Ship)(nil), 40.0
			for _, s := range g.Ships {
				if s.Kind != 0 {
					continue
				}
				sx, sy := u.Project(V3{s.X, 0.4, s.Y}, float64(u.W), float64(u.H))
				d := math.Hypot(sx-float64(u.MX), sy-float64(u.MY))
				if d < bd {
					bd = d
					best = s
				}
			}
			u.SelBP = 0
			if best != nil {
				u.SelShip = best.ID
				u.SelB = 0
			} else if bp := g.BlueAt(hx, hy); bp != nil {
				u.SelB, u.SelShip = 0, 0
				u.SelBP = bp.ID
			} else if b := g.BuildingAt(hx, hy); b != nil {
				u.SelB = b.ID
				u.SelShip = 0
			} else if rb := g.RivalBldAt(hx, hy); rb != nil {
				u.SelB, u.SelShip = 0, 0
				u.ShowDiplo = true
				PlaySfx(SfxClick)
			} else {
				u.SelB, u.SelShip = 0, 0
			}
		}
	}
	if !In.LDown {
		u.lastPlace = [2]int{-99, -99}
	}
	if In.RReleased && !u.rdragged {
		if u.Tool >= 0 {
			u.Tool = -1
			u.RoadDrag = false
		} else if s := g.ShipByID(u.SelShip); s != nil && s.War && g.enemyUnderCursor(s) != nil {
			e := g.enemyUnderCursor(s)
			s.AtkS = e.ID
			s.Route.Active = false
			g.ShipGoto(s, int(e.X), int(e.Y))
			PlaySfx(SfxShip)
			g.Notify("„"+T(s.Name)+T("“ greift „")+T(e.Name)+T("“ an!"), cAccent)
		} else if s := g.ShipByID(u.SelShip); s != nil {
			s.AtkS = 0
			tx, ty := hx, hy
			if b := g.BuildingAt(hx, hy); b != nil && BTypes[b.T].Coastal {
				tx, ty = b.Dock[0], b.Dock[1]
			}
			if g.W.In(tx, ty) && g.W.Terr[g.W.Idx(tx, ty)] == TWater {
				s.Route.Active = false
				if !g.ShipGoto(s, tx, ty) {
					g.Notify("Dieses Ziel ist nicht erreichbar.", cRed)
				} else {
					PlaySfx(SfxShip)
				}
			}
		} else {
			u.SelB = 0
		}
	}
}

// ------------------------------------------------------------ HUD

func (g *Game) currentIsland() *Island {
	u := g.UI
	if b := g.Bld(u.SelB); b != nil {
		return g.Islands[b.Isl]
	}
	if s := g.ShipByID(u.SelShip); s != nil {
		if k := g.ShipKontor(s); k != nil {
			return g.Islands[k.Isl]
		}
	}
	wx, wy := u.ScreenToWorld(float64(u.W)/2, float64(u.H)/2, float64(u.W), float64(u.H))
	best, bd := (*Island)(nil), 1e9
	for _, I := range g.Islands {
		if !I.Owned {
			continue
		}
		d := math.Hypot(float64(I.CX)-wx, float64(I.CY)-wy)
		if d < bd {
			bd = d
			best = I
		}
	}
	return best
}

func (g *Game) drawHUD(c *Canvas) {
	u := g.UI
	W, H := c.W, c.H
	// ---- Obere Leiste
	c.Fabric(0, 0, W, 46, uiNavy, 245)
	c.FillRect(0, 46, W, 12, RGB{12, 20, 38}, 245)
	c.FillRect(0, 46, W, 1, uiGold, 230)
	c.Meander(6, 48, W-12, uiGoldD, 255)
	c.FillRect(0, 57, W, 1, uiGold, 230)
	u.panels = append(u.panels, [4]int{0, 0, W, 58})
	x := 14
	c.Blit(UIIcon(IcoCredits, 26, cGold), x, 10)
	x += 32
	x = c.TextS(FBold, x, 12, fmtNum(int(g.Credits)), cText) + 8
	bal := int(math.Round(g.Balance))
	bc := cGreen
	bs := "+" + fmtNum(bal)
	if bal < 0 {
		bc = cRed
		bs = fmtNum(bal)
	}
	if u.inside(10, 0, x-10+TextW(FSmall, bs)+10, 46) {
		u.setTip("Credits", T("Bilanz pro Minute: ")+bs, "Steuern der Bewohner minus Unterhalt der Gebäude")
	}
	x = c.Text(FSmall, x, 16, bs+"/min", bc) + 22
	for t := 0; t < NumTiers; t++ {
		if t > 0 && g.MaxPop[t] == 0 && g.MaxPop[t-1] < Tiers[t].UnlockPop/2 {
			continue
		}
		x0 := x
		c.Blit(UIIcon(IcoPop, 24, Tiers[t].Col), x, 11)
		x += 26
		x = c.TextS(FBold, x, 12, fmtNum(g.Pop[t]), Tiers[t].Col) + 20
		if u.inside(x0, 0, x-x0, 46) {
			tip := []string{Tiers[t].Plural + ": " + fmtNum(g.Pop[t])}
			if t+1 < NumTiers {
				tip = append(tip, Tf("Aufstieg zu %s ab %d %s", Tiers[t+1].Plural, Tiers[t+1].UnlockPop, Tiers[t].Plural))
			}
			u.setTip(tip...)
		}
	}
	if I := g.currentIsland(); I != nil {
		ec := cGreen
		if I.EUse > I.EProd {
			ec = cRed
		}
		x0 := x
		c.Blit(UIIcon(IcoEnergy, 24, cGold), x, 11)
		x += 26
		x = c.TextS(FBold, x, 12, Tf("%d / %d", I.EProd, I.EUse), ec) + 20
		if u.inside(x0, 0, x-x0, 46) {
			u.setTip(T("Energie auf ")+T(I.Name), Tf("Erzeugung: %d   Verbrauch: %d", I.EProd, I.EUse), "Bei Energiemangel arbeiten Betriebe langsamer.")
		}
	}
	// Geschwindigkeit
	rx := W - 12 - 40
	if u.button(c, rx, 9, 40, 28, "", true) {
		u.PauseMenu = !u.PauseMenu
	}
	c.Blit(UIIcon(IcoMenu, 20, cText), rx+10, 13)
	speeds := []int{8, 4, 2, 1, 0}
	labels := []string{"8×", "4×", "2×", "1×", "II"}
	for i, s := range speeds {
		rx -= 44
		if u.button(c, rx, 9, 40, 28, labels[i], true) {
			if s == 0 {
				u.prevSpeed = imax(1, g.Speed)
			}
			g.Speed = s
		}
		if g.Speed == s {
			c.FillRect(rx+1, 34, 38, 2, cGold, 255)
			c.Rect(rx-1, 8, 42, 30, cGold, 255)
		}
	}
	hh := int(g.Hour)
	mm := int((g.Hour - float64(hh)) * 60)
	tx := rx - 14
	if len(g.Rivals) > 0 || g.Mode == ModeEndless {
		dw := 112
		tx -= dw + 6
		if u.button(c, tx+6, 9, dw, 28, "Diplomatie", true) {
			u.ShowDiplo = !u.ShowDiplo
		}
		for _, rv := range g.Rivals {
			if rv.Req != nil {
				c.Circle(float64(tx+6+dw-4), 12, 5, cGold, 255)
				break
			}
		}
		if u.inside(tx+6, 9, dw, 28) {
			u.setTip("Diplomatie", "Beziehungen, Abkommen und Anfragen der Konkurrenz")
		}
		tx -= 8
	}
	if g.Other != nil {
		rw := 96
		tx -= rw + 6
		lbl := "Mond"
		if g.OnMoon() {
			lbl = "Erde"
		}
		if u.button(c, tx+6, 9, rw, 28, lbl, true) {
			g.SwitchRegion()
		}
		if u.inside(tx+6, 9, rw, 28) {
			u.setTip("Region wechseln", "Zwischen Erde und Mondbasis umschalten.", "Beide Regionen laufen weiter.")
		}
		tx -= 8
	}
	{
		fw := 104
		tx -= fw + 6
		if u.button(c, tx+6, 9, fw, 28, "Forschung", true) {
			u.ShowResearch = !u.ShowResearch
		}
		for t := 0; t < NumTechs; t++ {
			if !g.Tech[t] && (Techs[t].Req < 0 || g.Tech[Techs[t].Req]) && g.RP >= float64(Techs[t].Cost) {
				c.Circle(float64(tx+6+fw-4), 12, 5, hex(0x7ad0ff), 255)
				break
			}
		}
		if u.inside(tx+6, 9, fw, 28) {
			u.setTip("Forschung", Tf("%d Forschungspunkte (+%.1f/min)", int(g.RP), g.researchRate()))
		}
		tx -= 8
	}
	{
		sw := 100
		tx -= sw + 6
		u.statsRect = [4]int{tx + 6, 9, sw, 28}
		if u.button(c, tx+6, 9, sw, 28, "Statistik", true) {
			u.ShowStats = !u.ShowStats
		}
		tx -= 8
	}
	c.TextR(FNormal, tx, 14, Tf("Jahr 2310 · Tag %d · %02d:%02d Uhr", g.Day+1, hh, mm), cDim)

	g.drawQuest(c)
	g.drawStore(c)
	g.drawBuildBar(c)
	g.drawMinimap(c, W-252, H-158, 242, 148)
	var selBP *Building
	for _, b := range g.BP {
		if b.ID == u.SelBP {
			selBP = b
		}
	}
	if b := g.Bld(u.SelB); b != nil {
		g.drawInfo(c, b)
	} else if selBP != nil {
		g.drawBlueprintPanel(c, selBP)
	} else if s := g.ShipByID(u.SelShip); s != nil {
		g.drawShipPanel(c, s)
	}
	// Meldungen
	y := 74
	for _, n := range g.Notifs {
		a := int(clampf(n.T, 0, 1) * 255)
		tw := TextW(FNormal, n.Text) + 30
		c.Fabric(W/2-tw/2, y, tw, 28, uiNavy, a*230/255)
		c.Rect(W/2-tw/2, y, tw, 28, uiGoldD, a)
		c.FillRect(W/2-tw/2, y, 4, 28, uiPurpleL, a)
		c.TextA(FNormal, W/2-tw/2+15, y+5, n.Text, n.Col, a)
		y += 32
	}
	if g.Speed == 0 && !u.PauseMenu {
		c.TextC(FBig, W/2, H/2-150, "PAUSIERT", cGold)
	}
	if u.ShowHelp {
		g.drawHelp(c)
	}
	if g.Won && !u.WinShown {
		pw, ph := 520, 220
		px, py := W/2-pw/2, H/2-ph/2
		u.panel(c, px, py, pw, ph)
		c.TextC(FBig, W/2, py+24, "Metropole der Zukunft!", cGreen)
		for i, l := range WrapText(FNormal, "Du hast Nova Archipel zu einer blühenden Metropole gemacht. Spiele gerne weiter und baue deine Inselwelt aus.", pw-60) {
			c.TextC(FNormal, W/2, py+80+i*22, l, cText)
		}
		if u.button(c, W/2-90, py+ph-60, 180, 40, "Weiterspielen", true) {
			u.WinShown = true
		}
	}
	u.modalDraw = true
	g.drawDialog(c)
	if u.PauseMenu {
		g.drawPauseMenu(c)
	}
	if u.ShowDiplo && !g.DialogOpen() && !u.PauseMenu {
		g.drawDiplomacy(c)
	}
	if u.ShowStats && !g.DialogOpen() && !u.PauseMenu {
		g.drawStats(c)
	}
	if u.ShowResearch && !g.DialogOpen() && !u.PauseMenu {
		g.drawResearch(c)
	}
	if u.ShowAch && !u.PauseMenu {
		drawAchievements(c, u, func() { u.ShowAch = false })
	}
	if g.Event0 != nil && !g.DialogOpen() && !u.PauseMenu && !u.ShowOptions && !u.ShowResearch && !u.ShowStats && !u.ShowDiplo && !u.ShowAch {
		g.drawEvent(c)
	}
	if u.ShowOptions {
		DrawOptions(c, u, func() { u.ShowOptions = false })
	}
	u.modalDraw = false
}

func (g *Game) drawQuest(c *Canvas) {
	if g.Mode == ModeTutorial {
		g.drawTutorial(c)
		return
	}
	if g.Mode == ModeCampaign {
		g.drawCampaignGoals(c)
		return
	}
	if g.Quest >= len(Quests) || !Opt.Hints {
		return
	}
	u := g.UI
	q := &Quests[g.Quest]
	lines := WrapText(FSmall, q.Desc, 270)
	h := 58 + len(lines)*17 + 18
	x, y := 12, 70
	u.panel(c, x, y, 300, h)
	c.Marble(x+4, y+4, 292, 42, 255)
	c.FillRect(x+4, y+46, 292, 1, uiGold, 220)
	c.Blit(UIIcon(IcoQuest, 20, cGold), x+12, y+13)
	c.Text(FSmall, x+40, y+7, Tf("AUFTRAG %d/%d", g.Quest+1, len(Quests)), cGold)
	c.TextS(FSerif, x+40, y+19, q.Title, cText)
	for i, l := range lines {
		c.Text(FSmall, x+14, y+52+i*17, l, cDim)
	}
	a, b := q.Check(g)
	by := y + 54 + len(lines)*17
	bar(c, x+14, by, 200, 8, float64(a)/float64(b), cAccent)
	c.Text(FSmall, x+222, by-5, Tf("%d / %d", imin(a, b), b), cText)
}

func (g *Game) drawStore(c *Canvas) {
	u := g.UI
	I := g.currentIsland()
	if I == nil {
		return
	}
	w, h := 272, 262
	x, y := 12, c.H-h-12
	u.panel(c, x, y, w, h)
	c.Marble(x+4, y+4, w-8, 28, 255)
	c.FillRect(x+4, y+32, w-8, 1, uiGold, 220)
	c.Blit(UIIcon(IcoStore, 20, cAccent), x+12, y+8)
	c.TextS(FSerif, x+40, y+5, I.Name, cText)
	c.TextR(FSmall, x+w-12, y+12, Tf("Lager %d", I.Cap), cAccent)
	fs := []string{}
	if I.Fert[FertKaffee] {
		fs = append(fs, "Kaffee")
	}
	if I.Fert[FertFasern] {
		fs = append(fs, "Fasern")
	}
	ft := "Fruchtbarkeit: keine"
	if len(fs) > 0 {
		ft = T("Fruchtbarkeit: ") + strings.Join(fs, ", ")
	}
	c.Text(FSmall, x+14, y+37, ft, cDim)
	for gd := 0; gd < NumGoods; gd++ {
		col, row := gd%2, gd/2
		gx := x + 12 + col*130
		gy := y + 58 + row*28
		c.Blit(GoodIcon(gd, 22), gx, gy)
		v := int(I.Stock[gd])
		tc := cText
		if v == 0 {
			tc = RGB{110, 125, 140}
		} else if v >= I.Cap {
			tc = cGold
		}
		c.Text(FNormal, gx+28, gy+2, fmtNum(v), tc)
		// Versorgungsbalken
		if I.Pop[0]+I.Pop[1]+I.Pop[2] > 0 && usedAsNeed(gd) {
			bar(c, gx+70, gy+9, 40, 5, I.Sat[gd], satCol(I.Sat[gd]))
		}
		if u.inside(gx, gy, 124, 24) {
			u.setTip(Goods[gd].Name, Tf("Bestand: %d / %d", v, I.Cap))
			if usedAsNeed(gd) {
				u.tip = append(u.tip, Tf("Versorgung der Bewohner: %d%%", int(I.Sat[gd]*100)))
			}
		}
	}
}

func usedAsNeed(gd int) bool {
	for _, t := range Tiers {
		for _, n := range t.Needs {
			if n.Good == gd {
				return true
			}
		}
	}
	return false
}

func satCol(f float64) RGB {
	if f > 0.85 {
		return cGreen
	}
	if f > 0.4 {
		return cGold
	}
	return cRed
}

var catIcons = [NumCats]int{BKontor, BHouse, BNahrung, BStahl, BChip, BWind}

func (g *Game) drawBuildBar(c *Canvas) {
	u := g.UI
	tabW := 66
	n := NumCats + 3
	tw := n*(tabW+6) + 10
	x := c.W/2 - tw/2
	y := c.H - 80
	u.panel(c, x, y, tw, 72)
	for i := 0; i < NumCats; i++ {
		bx := x + 8 + i*(tabW+6)
		by := y + 8
		hov := u.inside(bx, by, tabW, 56)
		bg := uiNavyL
		if u.Cat == i {
			bg = uiPurple
		} else if hov {
			bg = RGB{44, 62, 100}
		}
		c.Fabric(bx, by, tabW, 56, bg, 245)
		c.Rect(bx, by, tabW, 56, map[bool]RGB{true: uiGold, false: uiGoldD}[u.Cat == i || hov], 255)
		u.tabRect[i] = [4]int{bx, by, tabW, 56}
		s := Thumb(catIcons[i], 0)
		fw, fh := fitSize(s.W, s.H, tabW-10, 46)
		c.BlitScaled(s, bx+(tabW-fw)/2, by+(56-fh)/2, fw, fh)
		if hov {
			u.setTip(CatNames[i])
		}
		if u.clicked(bx, by, tabW, 56) {
			if u.Cat == i {
				u.Cat = -1
			} else {
				u.Cat = i
			}
		}
	}
	// Abriss
	bx := x + 8 + NumCats*(tabW+6)
	hov := u.inside(bx, y+8, tabW, 56)
	bg := RGB{60, 22, 26}
	if u.Tool == ToolDemolish {
		bg = RGB{130, 34, 38}
	} else if hov {
		bg = RGB{96, 28, 32}
	}
	c.Fabric(bx, y+8, tabW, 56, bg, 245)
	c.Rect(bx, y+8, tabW, 56, uiGoldD, 255)
	c.Blit(UIIcon(IcoDemolish, 26, RGB{255, 140, 140}), bx+20, y+23)
	if hov {
		u.setTip("Abreißen (Entf)", "Gibt die Hälfte der Baumaterialien zurück.")
	}
	if u.clicked(bx, y+8, tabW, 56) {
		if u.Tool == ToolDemolish {
			u.Tool = -1
		} else {
			u.Tool = ToolDemolish
			u.Cat = -1
		}
	}
	// Blaupausen-Modus
	bx = x + 8 + (NumCats+1)*(tabW+6)
	hov = u.inside(bx, y+8, tabW, 56)
	bg = RGB{18, 40, 72}
	if u.BlueMode {
		bg = RGB{30, 90, 170}
	} else if hov {
		bg = RGB{26, 60, 110}
	}
	c.Fabric(bx, y+8, tabW, 56, bg, 245)
	c.Rect(bx, y+8, tabW, 56, map[bool]RGB{true: hex(0x60b0ff), false: uiGoldD}[u.BlueMode], 255)
	for k := 0; k < 4; k++ {
		c.FillRect(bx+14+k*12, y+16, 1, 26, hex(0x9ad0ff), 120)
		c.FillRect(bx+14, y+16+k*8, 37, 1, hex(0x9ad0ff), 120)
	}
	c.RoundRectLine(bx+22, y+20, 22, 16, 3, RGB{255, 255, 255}, 230)
	c.TextC(FSmall, bx+tabW/2, y+44, "Plan", RGB{220, 236, 255})
	if hov {
		u.setTip("Blaupausen-Modus (B)", "Gebäude erst planen – kostenlos und ohne Material.", "Später einzeln oder alle auf einmal bauen.")
	}
	if u.clicked(bx, y+8, tabW, 56) || In.KeyHit[KB] {
		u.BlueMode = !u.BlueMode
		PlaySfx(SfxClick)
	}
	// Überlagerungen
	bx += tabW + 6
	hov = u.inside(bx, y+8, tabW, 56)
	bg = RGB{18, 40, 72}
	if u.Overlay > 0 {
		bg = RGB{20, 110, 90}
	} else if hov {
		bg = RGB{26, 60, 110}
	}
	c.Fabric(bx, y+8, tabW, 56, bg, 245)
	c.Rect(bx, y+8, tabW, 56, uiGoldD, 255)
	for k := 0; k < 3; k++ {
		c.RoundRect(bx+16+k*4, y+16+k*7, 30, 14, 3, []RGB{hex(0x4fe08a), hex(0xffc24a), hex(0xff6a6a)}[k], 200)
	}
	c.TextC(FSmall, bx+tabW/2, y+44, "Ansicht", RGB{220, 236, 255})
	if hov {
		u.setTip(T("Überlagerung (O): ")+OverlayNames[u.Overlay], "Einfluss, Energie, Versorgung, Brandschutz, Straßenanbindung")
	}
	if u.clicked(bx, y+8, tabW, 56) {
		u.Overlay = (u.Overlay + 1) % len(OverlayNames)
		PlaySfx(SfxClick)
	}
	if len(g.BP) > 0 && u.Cat < 0 {
		cr, _ := g.blueprintCost()
		lbl := Tf("Alle %d Blaupausen bauen (%s Cr)", len(g.BP), fmtNum(cr))
		bw := TextW(FNormal, lbl) + 30
		if u.button(c, c.W/2-bw/2, y-44, bw, 34, lbl, true) {
			g.BuildAllBlueprints()
		}
	}
	if u.Cat < 0 {
		return
	}
	var list []int
	for t := 0; t < NumBTypes; t++ {
		if BTypes[t].Cat == u.Cat {
			list = append(list, t)
		}
	}
	cw, ch := 92, 112
	pw := len(list)*(cw+6) + 14
	px := c.W/2 - pw/2
	py := y - ch - 24
	u.panel(c, px, py, pw, ch+16)
	ctw := TextW(FSerif, CatNames[u.Cat]) + 30
	c.Marble(px+10, py-30, ctw, 26, 255)
	c.Rect(px+10, py-30, ctw, 26, uiGold, 230)
	c.TextS(FSerif, px+25, py-30, CatNames[u.Cat], cText)
	I := g.currentIsland()
	for i, t := range list {
		bt := &BTypes[t]
		cx := px + 8 + i*(cw+6)
		cy := py + 8
		u.itemRect[t] = [4]int{cx, cy, cw, ch}
		unl := g.Unlocked(t)
		hov := u.inside(cx, cy, cw, ch)
		bg := uiNavyL
		if u.Tool == t {
			bg = uiPurple
		} else if hov {
			bg = RGB{44, 62, 100}
		}
		c.Fabric(cx, cy, cw, ch, bg, 245)
		c.Rect(cx, cy, cw, ch, map[bool]RGB{true: uiGold, false: uiGoldD}[u.Tool == t || hov], 255)
		s := Thumb(t, 0)
		fw, fh := fitSize(s.W, s.H, cw-12, 58)
		c.BlitScaled(s, cx+(cw-fw)/2, cy+4+(58-fh)/2, fw, fh)
		if !unl {
			c.FillRect(cx+1, cy+1, cw-2, ch-2, RGB{5, 10, 18}, 160)
		}
		name := bt.Name
		if TextW(FSmall, name) > cw-6 {
			for TextW(FSmall, name+".") > cw-6 {
				name = name[:len(name)-1]
			}
			name += "."
		}
		c.TextC(FSmall, cx+cw/2, cy+64, name, map[bool]RGB{true: cText, false: cDim}[unl])
		c.Blit(UIIcon(IcoCredits, 14, cGold), cx+8, cy+88)
		c.Text(FSmall, cx+24, cy+86, fmtNum(bt.Cost), cGold)
		if bt.Mat[0] > 0 {
			c.Blit(GoodIcon(GBaustoffe, 16), cx+cw-38, cy+87)
			c.Text(FSmall, cx+cw-20, cy+86, itoa(bt.Mat[0]), cText)
		}
		if hov {
			g.buildTip(t, I)
		}
		if u.clicked(cx, cy, cw, ch) && unl {
			if u.Tool == t {
				u.Tool = -1
			} else {
				u.Tool = t
				u.SelB = 0
			}
		}
	}
}

func fitSize(w, h, mw, mh int) (int, int) {
	f := math.Min(float64(mw)/float64(w), float64(mh)/float64(h))
	return int(float64(w) * f), int(float64(h) * f)
}

func (g *Game) buildTip(t int, I *Island) {
	bt := &BTypes[t]
	tip := []string{bt.Name}
	tip = append(tip, WrapText(FSmall, bt.Desc, 300)...)
	cost := T("Kosten: ") + fmtNum(bt.Cost) + T(" Credits")
	for k, m := range bt.Mat {
		if m > 0 {
			cost += ", " + itoa(m) + " " + Goods[MatGoods[k]].Name
		}
	}
	tip = append(tip, "", cost)
	if bt.Upkeep > 0 {
		tip = append(tip, T("Unterhalt: ")+itoa(bt.Upkeep)+T(" Credits/min"))
	}
	if bt.Energy > 0 {
		tip = append(tip, T("Energie: +")+itoa(bt.Energy))
	} else if bt.Energy < 0 {
		tip = append(tip, T("Energiebedarf: ")+itoa(-bt.Energy))
	}
	if bt.Out >= 0 {
		s := T("Produziert: ") + T(Goods[bt.Out].Name)
		if len(bt.In) > 0 {
			var ins []string
			for _, in := range bt.In {
				ins = append(ins, Goods[in].Name)
			}
			s += T(" aus ") + strings.Join(ins, " + ")
		}
		s += Tf(" (%.0f s)", bt.Cycle)
		tip = append(tip, s)
	}
	if bt.Radius > 0 && t != BKontor && t != BLager {
		tip = append(tip, T("Wirkungsradius: ")+itoa(bt.Radius))
	}
	if !g.Unlocked(t) {
		tip = append(tip, "", "🔒 "+g.UnlockText(t))
	}
	g.UI.tip = tip
}

func (g *Game) drawTooltip(c *Canvas) {
	u := g.UI
	if len(u.tip) == 0 {
		return
	}
	w := 0
	for i, l := range u.tip {
		f := FSmall
		if i == 0 {
			f = FBold
		}
		w = imax(w, TextW(f, l))
	}
	w += 24
	h := 14 + 22 + (len(u.tip)-1)*17
	x := u.MX + 18
	y := u.MY + 18
	if x+w > c.W-4 {
		x = u.MX - w - 10
	}
	if y+h > c.H-4 {
		y = c.H - h - 4
	}
	c.Fabric(x, y, w, h, RGB{14, 22, 42}, 248)
	c.Marble(x+1, y+1, w-2, 28, 255)
	c.Rect(x, y, w, h, uiGold, 230)
	c.FillRect(x+1, y+29, w-2, 1, uiGold, 200)
	c.TextS(FBold, x+12, y+5, u.tip[0], cText)
	for i, l := range u.tip[1:] {
		col := cDim
		if strings.HasPrefix(l, "🔒") {
			l = strings.TrimPrefix(l, "🔒 ")
			col = cRed
		}
		if strings.HasPrefix(l, "Kosten") || strings.HasPrefix(l, "Produziert") {
			col = cText
		}
		c.Text(FSmall, x+12, y+32+i*17, l, col)
	}
}

// ------------------------------------------------------------ Infopanel

func (g *Game) drawInfo(c *Canvas, b *Building) {
	u := g.UI
	bt := &BTypes[b.T]
	I := g.Islands[b.Isl]
	w := 340
	x := c.W - w - 12
	y := 58
	h := 200
	switch {
	case b.T == BHouse:
		h = 250 + len(Tiers[b.Tier].Needs)*30 + 40
	case b.T == BKontor:
		h = 130 + NumGoods*24 + 20
	case bt.Out >= 0:
		h = 300
	case b.T == BWerft:
		h = 220
	}
	h = imin(h, c.H-y-176)
	u.panel(c, x, y, w, h)
	c.Marble(x+4, y+4, w-8, 74, 255)
	c.FillRect(x+4, y+78, w-8, 1, uiGold, 220)
	spr := Thumb(b.T, b.Tier)
	fw, fh := fitSize(spr.W, spr.H, 80, 66)
	c.BlitScaled(spr, x+12, y+12, fw, fh)
	title := bt.Name
	if b.T == BHouse {
		title = T(Tiers[b.Tier].Name) + T("-Wohnmodul")
	}
	tf := FSerif
	if TextW(tf, title) > w-140 {
		tf = FBold
	}
	c.TextS(tf, x+100, y+14, title, cText)
	c.Text(FSmall, x+100, y+40, I.Name, cAccent)
	if u.button(c, x+w-34, y+10, 24, 24, "×", true) {
		u.SelB = 0
	}
	yy := y + 84
	switch {
	case b.T == BHouse:
		tier := &Tiers[b.Tier]
		c.Text(FNormal, x+16, yy, "Bewohner", cDim)
		c.TextR(FNormal, x+w-16, yy, Tf("%d / %d", int(b.Res), tier.MaxRes), tier.Col)
		yy += 22
		bar(c, x+16, yy, w-32, 8, b.Res/float64(tier.MaxRes), tier.Col)
		yy += 18
		tax := b.Res * tier.Tax
		c.Text(FSmall, x+16, yy, Tf("Steuern: bis %.0f Credits/min", tax), cGold)
		yy += 26
		c.Text(FSmall, x+16, yy, "BEDÜRFNISSE", cAccent)
		yy += 20
		for k, nd := range tier.Needs {
			f := 0.0
			if k < len(b.Fulfil) {
				f = b.Fulfil[k]
			}
			if nd.Good >= 0 {
				c.Blit(GoodIcon(nd.Good, 22), x+16, yy)
				c.Text(FNormal, x+44, yy+2, Goods[nd.Good].Name, cText)
			} else {
				spr := Thumb(nd.Service, 0)
				fw, fh := fitSize(spr.W, spr.H, 24, 22)
				c.BlitScaled(spr, x+16, yy, fw, fh)
				c.Text(FNormal, x+44, yy+2, BTypes[nd.Service].Name, cText)
			}
			bar(c, x+w-120, yy+8, 100, 7, f, satCol(f))
			yy += 30
		}
		yy += 4
		if !b.Conn {
			c.Text(FNormal, x+16, yy, "Keine Straßenanbindung an ein Lager!", cRed)
		} else if b.Tier+1 < NumTiers {
			next := &Tiers[b.Tier+1]
			msg := T("Aufstieg zu ") + T(next.Plural) + T(": alle Bedürfnisse erfüllen")
			col := cDim
			if g.MaxPop[b.Tier] < next.UnlockPop {
				msg = Tf("Aufstieg ab %d %s möglich", next.UnlockPop, tier.Plural)
			} else {
				cs := []string{}
				for k, m := range next.UpCost {
					if m > 0 {
						cs = append(cs, itoa(m)+" "+Goods[MatGoods[k]].Name)
					}
				}
				if b.UpTimer > 0 {
					msg = T("Aufstieg steht bevor (benötigt ") + strings.Join(cs, ", ") + ")"
					col = cGreen
				}
			}
			for i, l := range WrapText(FSmall, msg, w-32) {
				c.Text(FSmall, x+16, yy+i*16, l, col)
			}
			lab := "Aufstieg sperren"
			if b.Lock {
				lab = "Aufstieg erlauben"
			}
			if u.button(c, x+16, y+h-42, 150, 30, lab, true) {
				b.Lock = !b.Lock
			}
		}
		if u.button(c, x+w-136, y+h-42, 120, 30, "Abreißen", true) {
			g.Demolish(b.X, b.Y)
			u.SelB = 0
		}
	case b.T == BKontor:
		c.Text(FSmall, x+16, yy-6, "HANDEL MIT DER ORION-GILDE", cAccent)
		c.TextC(FSmall, x+219, yy+6, "Kaufen", cDim)
		c.TextC(FSmall, x+291, yy+6, "Verkaufen", cDim)
		yy += 28
		for gd := 0; gd < NumGoods; gd++ {
			c.Blit(GoodIcon(gd, 20), x+16, yy)
			c.Text(FSmall, x+42, yy+2, Goods[gd].Name, cText)
			c.TextR(FSmall, x+175, yy+2, fmtNum(int(I.Stock[gd])), cText)
			buy := int(float64(Goods[gd].Price) * 1.3 * 10)
			sell := int(float64(Goods[gd].Price) * 0.7 * 10)
			if u.button(c, x+186, yy, 66, 20, "-"+fmtNum(buy), g.Credits >= float64(buy) && I.Stock[gd]+10 <= float64(I.Cap)) {
				g.Credits -= float64(buy)
				I.Stock[gd] += 10
			}
			if u.button(c, x+258, yy, 66, 20, "+"+fmtNum(sell), I.Stock[gd] >= 10) {
				g.Credits += float64(sell)
				I.Stock[gd] -= 10
			}
			if u.inside(x+186, yy, 66, 20) {
				u.setTip("10 "+T(Goods[gd].Name)+T(" kaufen"), T("Preis: ")+fmtNum(buy)+T(" Credits"))
			}
			if u.inside(x+258, yy, 66, 20) {
				u.setTip("10 "+T(Goods[gd].Name)+T(" verkaufen"), T("Erlös: ")+fmtNum(sell)+T(" Credits"))
			}
			yy += 24
		}
	case bt.Out >= 0:
		st := StatusText[b.Status]
		sc := cGreen
		if b.Status != StOK {
			sc = cRed
		}
		if b.Status == StNoEnergy || b.Status == StPaused {
			sc = cGold
		}
		c.Text(FNormal, x+16, yy, "Status", cDim)
		c.TextR(FNormal, x+w-16, yy, st, sc)
		yy += 26
		c.Text(FNormal, x+16, yy, "Produktivität", cDim)
		c.TextR(FNormal, x+w-16, yy, Tf("%d %%", int(b.Prod*100+0.5)), cText)
		yy += 22
		bar(c, x+16, yy, w-32, 8, b.Prod, satCol(b.Prod))
		yy += 22
		// Kette
		cx := x + 16
		for _, in := range bt.In {
			c.Blit(GoodIcon(in, 32), cx, yy)
			cx += 38
		}
		if len(bt.In) > 0 {
			c.FillRect(cx+2, yy+14, 12, 4, cAccent, 255)
			for k := 0; k < 7; k++ {
				c.FillRect(cx+14+k, yy+9+k, 1, 14-2*k, cAccent, 255)
			}
			cx += 30
		}
		c.Blit(GoodIcon(bt.Out, 32), cx, yy)
		bar(c, cx+42, yy+12, w-32-(cx+42-x-16), 8, b.Prog, cAccent)
		yy += 44
		c.Text(FSmall, x+16, yy, Tf("Unterhalt: %d Credits/min   Energie: %d", bt.Upkeep, -bt.Energy), cDim)
		lab := "Pausieren"
		if b.Paused {
			lab = "Fortsetzen"
		}
		if u.button(c, x+16, y+h-42, 140, 30, lab, true) {
			b.Paused = !b.Paused
			g.dirty = true
		}
		if u.button(c, x+w-136, y+h-42, 120, 30, "Abreißen", true) {
			g.Demolish(b.X, b.Y)
			u.SelB = 0
		}
	default:
		for i, l := range WrapText(FSmall, bt.Desc, w-32) {
			c.Text(FSmall, x+16, yy+i*17, l, cDim)
		}
		yy += 60
		if bt.Energy > 0 {
			c.Text(FNormal, x+16, yy, Tf("Erzeugt %d Energie", bt.Energy), cGold)
			yy += 24
		}
		if b.T == BKomm || b.T == BMedi || b.T == BLabor {
			n := 0
			for _, h := range g.B {
				if h.T == BHouse && h.Cover[b.T] {
					n++
				}
			}
			c.Text(FNormal, x+16, yy, Tf("Versorgt %d Wohnmodule", n), cText)
			if !b.Conn {
				c.Text(FNormal, x+16, yy+22, "Keine Straßenanbindung!", cRed)
			}
		}
		if b.T == BWerft {
			if b.Building {
				nm := "Frachtgleiter"
				if b.ShipWar {
					nm = "Korvette"
				}
				c.Text(FNormal, x+16, yy, nm+T(" im Bau …"), cText)
				bar(c, x+16, yy+24, w-32, 8, b.ShipQ, cAccent)
			} else {
				ok := g.Credits >= 1500 && I.Stock[GBaustoffe] >= 15 && b.Conn && b.Constr == 0
				if u.button(c, x+16, yy, w-32, 34, "Frachtgleiter bauen (1.500 Cr, 15 Baustoffe)", ok) {
					g.Credits -= 1500
					I.Stock[GBaustoffe] -= 15
					b.Building = true
				}
				ok2 := g.Credits >= 3500 && I.Stock[GBaustoffe] >= 25 && b.Conn && b.Constr == 0
				if u.button(c, x+16, yy+40, w-32, 34, "Korvette bauen (3.500 Cr, 25 Baustoffe)", ok2) {
					g.Credits -= 3500
					I.Stock[GBaustoffe] -= 25
					b.Building = true
					b.ShipWar = true
				}
				if u.inside(x+16, yy+40, w-32, 34) {
					u.setTip("Korvette", "Kriegsschiff mit Lasergeschützen.", "Schützt Frachter vor Piraten und feindlichen Flotten.", "Rechtsklick auf ein feindliches Schiff: angreifen.")
				}
			}
		}
		if b.T != BKontor {
			if u.button(c, x+w-136, y+h-42, 120, 30, "Abreißen", true) {
				g.Demolish(b.X, b.Y)
				u.SelB = 0
			}
		}
	}
}

func (g *Game) drawShipPanel(c *Canvas, s *Ship) {
	u := g.UI
	if s.War {
		g.drawWarPanel(c, s)
		return
	}
	w, h := 340, 430
	x := c.W - w - 12
	y := 58
	u.panel(c, x, y, w, h)
	c.Marble(x+4, y+4, w-8, 74, 255)
	c.FillRect(x+4, y+78, w-8, 1, uiGold, 220)
	spr := Thumb(-1, 0)
	fw, fh := fitSize(spr.W, spr.H, 80, 60)
	c.BlitScaled(spr, x+10, y+12, fw, fh)
	c.TextS(FSerif, x+100, y+12, s.Name, cText)
	st := "Wartet auf Befehle"
	if len(s.Path) > 0 {
		st = "Unterwegs"
	}
	if s.Route.Active {
		st = "Handelsroute aktiv"
	}
	c.Text(FSmall, x+100, y+36, T("Frachtgleiter · ")+st, cDim)
	if u.button(c, x+w-34, y+10, 24, 24, "×", true) {
		u.SelShip = 0
	}
	yy := y + 84
	c.Text(FSmall, x+16, yy, "FRACHTRAUM", cAccent)
	yy += 18
	for i, cg := range s.Cargo {
		sx := x + 16 + i*78
		c.FillRect(sx, yy, 70, 44, RGB{10, 16, 30}, 230)
		c.Rect(sx, yy, 70, 44, uiGoldD, 255)
		if cg.G >= 0 && cg.Amt > 0 {
			c.Blit(GoodIcon(cg.G, 26), sx+6, yy+9)
			c.Text(FBold, sx+36, yy+12, itoa(cg.Amt), cText)
			if u.inside(sx, yy, 70, 44) {
				u.setTip(Goods[cg.G].Name, itoa(cg.Amt)+" / 50")
			}
		}
	}
	yy += 56
	k := g.ShipKontor(s)
	if k != nil {
		I := g.Islands[k.Isl]
		c.Text(FSmall, x+16, yy, T("AM KONTOR VON ")+strings.ToUpper(I.Name), cAccent)
		yy += 18
		if u.button(c, x+16, yy, w-32, 28, "Alles entladen", true) {
			for i := range s.Cargo {
				cg := &s.Cargo[i]
				if cg.G >= 0 {
					sp := imin(cg.Amt, int(float64(I.Cap)-I.Stock[cg.G]))
					I.Stock[cg.G] += float64(sp)
					cg.Amt -= sp
					if cg.Amt == 0 {
						cg.G = -1
					}
				}
			}
		}
		yy += 34
		c.Text(FSmall, x+16, yy, "Laden (Klick = 10 Einheiten):", cDim)
		yy += 18
		n := 0
		for gd := 0; gd < NumGoods; gd++ {
			if I.Stock[gd] < 1 {
				continue
			}
			gx := x + 16 + (n%10)*30
			gy := yy + (n/10)*30
			c.Blit(GoodIcon(gd, 26), gx, gy)
			if u.inside(gx, gy, 26, 26) {
				u.setTip(T(Goods[gd].Name)+T(" laden"), Tf("Im Kontor: %d", int(I.Stock[gd])))
				c.Rect(gx-1, gy-1, 28, 28, cAccent, 255)
			}
			if u.clicked(gx, gy, 26, 26) {
				put := s.Put(gd, imin(10, int(I.Stock[gd])))
				I.Stock[gd] -= float64(put)
			}
			n++
		}
		yy += 64
	} else {
		for i, l := range WrapText(FSmall, "Rechtsklick auf das Meer bewegt das Schiff. Rechtsklick auf einen Kontor steuert ihn an. Mit Baustoffen an Bord kannst du an neuen Küsten einen Kontor gründen.", w-32) {
			c.Text(FSmall, x+16, yy+i*16, l, cDim)
		}
		yy += 82
	}
	// Handelsroute
	c.Text(FSmall, x+16, yy, "HANDELSROUTE", cAccent)
	yy += 20
	ks := g.kontors()
	r := &s.Route
	kname := func(id int) string {
		if b := g.Bld(id); b != nil {
			return g.Islands[b.Isl].Name
		}
		return "—"
	}
	cycleK := func(id int) int {
		if len(ks) == 0 {
			return 0
		}
		for i, k := range ks {
			if k.ID == id {
				return ks[(i+1)%len(ks)].ID
			}
		}
		return ks[0].ID
	}
	gname := func(gd int) string {
		if gd < 0 {
			return "—"
		}
		return Goods[gd].Name
	}
	cycleG := func(gd int) int {
		gd++
		if gd >= NumGoods {
			return -1
		}
		return gd
	}
	half := (w - 40) / 2
	c.Text(FSmall, x+16, yy, "Von", cDim)
	c.Text(FSmall, x+24+half, yy, "Nach", cDim)
	yy += 16
	if u.button(c, x+16, yy, half, 26, kname(r.A), len(ks) > 0) {
		r.A = cycleK(r.A)
		r.Active = false
	}
	if u.button(c, x+24+half, yy, half, 26, kname(r.B), len(ks) > 0) {
		r.B = cycleK(r.B)
		r.Active = false
	}
	yy += 32
	c.Text(FSmall, x+16, yy, "Ware hin", cDim)
	c.Text(FSmall, x+24+half, yy, "Ware zurück", cDim)
	yy += 16
	if u.button(c, x+16, yy, half, 26, gname(r.GAB), true) {
		r.GAB = cycleG(r.GAB)
	}
	if u.button(c, x+24+half, yy, half, 26, gname(r.GBA), true) {
		r.GBA = cycleG(r.GBA)
	}
	yy += 34
	can := g.Bld(r.A) != nil && g.Bld(r.B) != nil && r.A != r.B && (r.GAB >= 0 || r.GBA >= 0)
	lab := "Route starten"
	if r.Active {
		lab = "Route stoppen"
	}
	if u.button(c, x+16, yy, w-32, 30, lab, can || r.Active) {
		r.Active = !r.Active
		s.Phase = 0
		s.Path = nil
	}
}

// ------------------------------------------------------------ Minimap

func (g *Game) drawMinimap(c *Canvas, x, y, w, h int) {
	u := g.UI
	u.panel(c, x, y, w, h)
	mw, mh := w-12, h-12
	if u.mini == nil || u.Anim-u.miniT > 1.0 || u.mini.W != mw {
		u.miniT = u.Anim
		s := &Sprite{W: mw, H: mh, Pix: make([]uint32, mw*mh)}
		n := float64(g.W.N)
		for py := 0; py < mh; py++ {
			for px := 0; px < mw; px++ {
				// Raute: px -> (x - y), py -> (x + y)
				a := (float64(px)/float64(mw)*2 - 1) * n // x - y
				b := float64(py) / float64(mh) * 2 * n   // x + y
				tx, ty := int((a+b)/2), int((b-a)/2)
				if !g.W.In(tx, ty) {
					continue
				}
				i := g.W.Idx(tx, ty)
				var col RGB
				switch g.W.Terr[i] {
				case TWater:
					col = colDeep.Mix(colShallow, clampf(1-float64(g.W.Dist[i])/5, 0, 1)*0.6)
				case TSand:
					col = colSand
				case TGrass:
					col = colGrass[g.W.Biome[i]]
				case TRock:
					col = colRock
				}
				if g.W.Moon {
					col = RGB{uint8(int(col.R)/3 + int(col.G)/3 + int(col.B)/3), 0, 0}
					v := col.R
					col = RGB{v, v, uint8(int(v) * 105 / 100 % 256)}
					if g.W.Terr[i] == TWater {
						col = hex(0x2a2a2e)
					}
				}
				if is := int(g.W.Isl[i]); is >= 0 && is < len(g.Islands) && g.Islands[is].Rival > 0 && g.W.Terr[i] != TWater {
					col = col.Mix(RivalDefs[g.Islands[is].Rival-1].Col, 0.45)
				}
				if g.W.Road[i] {
					col = hex(0x9aa4b0)
				}
				if o := g.W.Occ[i]; o != 0 {
					if b := g.Bld(int(o) - 1); b != nil {
						col = BTypes[b.T].Col
					}
				}
				s.Pix[py*mw+px] = 0xFF000000 | col.U()
			}
		}
		u.mini = s
	}
	c.Marble(x+6, y+6, mw, mh, 255)
	c.Blit(u.mini, x+6, y+6)
	n := float64(g.W.N)
	toMini := func(wx, wy float64) (float64, float64) {
		return float64(x+6) + ((wx-wy)/n+1)/2*float64(mw), float64(y+6) + (wx+wy)/(2*n)*float64(mh)
	}
	for _, s := range g.Ships {
		sx, sy := toMini(s.X, s.Y)
		col := RGB{255, 255, 255}
		if s.Kind == 1 {
			col = cGold
		} else if s.Kind >= 2 {
			col = FacCol(shipFac(s))
		}
		c.Circle(sx, sy, 2.2, col, 255)
	}
	// Sichtbereich
	var pts [4][2]float64
	for k, p := range [][2]float64{{0, 0}, {float64(c.W), 0}, {float64(c.W), float64(c.H)}, {0, float64(c.H)}} {
		wx, wy := u.ScreenToWorld(p[0], p[1], float64(c.W), float64(c.H))
		pts[k][0], pts[k][1] = toMini(wx, wy)
	}
	c.SetClip(x+6, y+6, x+6+mw, y+6+mh)
	for k := 0; k < 4; k++ {
		c.Line(pts[k][0], pts[k][1], pts[(k+1)%4][0], pts[(k+1)%4][1], RGB{255, 236, 190}, 240)
	}
	c.ResetClip()
	if In.LDown && u.inside(x+6, y+6, mw, mh) && In.PressX >= x && In.PressY >= y && In.PressX < x+w && In.PressY < y+h {
		fx := (float64(u.MX-x-6)/float64(mw))*2 - 1
		fy := float64(u.MY-y-6) / float64(mh) * 2
		a, b := fx*n, fy*n
		u.CenterOn(g, (a+b)/2, (b-a)/2)
		u.consumed = true
	}
}

// ------------------------------------------------------------ Menüs

func (g *Game) drawPauseMenu(c *Canvas) {
	u := g.UI
	c.FillRect(0, 0, c.W, c.H, RGB{0, 5, 12}, 150)
	pw, ph := 360, 478
	x, y := c.W/2-pw/2, c.H/2-ph/2
	u.panel(c, x, y, pw, ph)
	c.Marble(x+4, y+4, pw-8, 60, 255)
	c.TextC(FBig, c.W/2, y+16, "Menü", cText)
	by := y + 80
	if u.button(c, x+40, by, pw-80, 44, "Fortsetzen", true) {
		u.PauseMenu = false
	}
	by += 54
	if u.button(c, x+40, by, pw-80, 44, "Speichern", true) && Main != nil {
		Main.Slots.Show(true)
		u.PauseMenu = false
	}
	by += 54
	if u.button(c, x+40, by, pw-80, 44, "Laden", true) && Main != nil {
		Main.Slots.Show(false)
		u.PauseMenu = false
	}
	by += 54
	if u.button(c, x+40, by, pw-80, 44, "Optionen", true) {
		u.ShowOptions = true
		u.PauseMenu = false
	}
	by += 54
	if u.button(c, x+40, by, pw-80, 44, "Erfolge", true) {
		u.ShowAch = true
		u.PauseMenu = false
	}
	by += 54
	if u.button(c, x+40, by, pw-80, 44, "Hauptmenü", true) {
		u.PauseMenu = false
		Main.ToMenu = true
	}
	by += 54
	if u.button(c, x+40, by, pw-80, 44, "Beenden", true) {
		In.Quit = true
	}
}

var helpLines = []string{
	"Linksklick – Auswählen / Bauen (gedrückt halten für Reihen)",
	"Rechtsklick – Schiff bewegen / Werkzeug abbrechen",
	"Rechte Maustaste ziehen – Karte verschieben",
	"Mittlere Maustaste ziehen oder Q / E – Kamera drehen",
	"WASD / Pfeiltasten – Kamera bewegen",
	"Mausrad / + / - – Zoom (stufenlos)",
	"Leertaste – Pause,  1–4 – Spielgeschwindigkeit",
	"Entf – Abriss-Werkzeug,  Esc – Abbrechen / Menü",
	"F5 – Speichern,  F11 – Vollbild,  H – diese Hilfe",
}

func (g *Game) drawHelp(c *Canvas) {
	u := g.UI
	pw := 520
	ph := 90 + len(helpLines)*24
	x, y := c.W/2-pw/2, c.H/2-ph/2
	u.panel(c, x, y, pw, ph)
	c.Marble(x+4, y+4, pw-8, 50, 255)
	c.TextC(FBig, c.W/2, y+12, "Steuerung", cText)
	for i, l := range helpLines {
		c.Text(FNormal, x+30, y+64+i*24, l, cText)
	}
	if u.clicked(x, y, pw, ph) {
		u.ShowHelp = false
	}
}

func (u *UIState) Panels() [][4]int { return u.panels }

func (g *Game) enemyUnderCursor(s *Ship) *Ship {
	u := g.UI
	var best *Ship
	bd := 42.0
	for _, o := range g.Ships {
		if o.Dead || !g.hostile(s, o) {
			continue
		}
		sx, sy := u.Project(V3{o.X, 0.4, o.Y}, float64(u.W), float64(u.H))
		if d := math.Hypot(sx-float64(u.MX), sy-float64(u.MY)); d < bd {
			best, bd = o, d
		}
	}
	return best
}

func (g *Game) drawWarPanel(c *Canvas, s *Ship) {
	u := g.UI
	w, h := 340, 236
	x := c.W - w - 12
	y := 58
	u.panel(c, x, y, w, h)
	c.TextS(FSerif, x+18, y+14, s.Name, cText)
	st := "Bereit"
	if t := g.ShipByID(s.AtkS); t != nil {
		st = T("Greift „") + T(t.Name) + T("“ an")
	} else if len(s.Path) > 0 {
		st = "Unterwegs"
	}
	c.Text(FSmall, x+18, y+42, T("Korvette · ")+st, cDim)
	if u.button(c, x+w-34, y+10, 24, 24, "×", true) {
		u.SelShip = 0
	}
	yy := y + 72
	c.Text(FSmall, x+18, yy, "RUMPF", cAccent)
	c.TextR(FSmall, x+w-18, yy, Tf("%.0f / %.0f", s.HP, s.MaxHP), cText)
	f := s.HP / math.Max(1, s.MaxHP)
	col := cGreen
	if f < 0.35 {
		col = cRed
	} else if f < 0.65 {
		col = cGold
	}
	bar(c, x+18, yy+20, w-36, 10, f, col)
	yy += 44
	for _, l := range []string{"Rechtsklick auf feindliches Schiff: angreifen", "Rechtsklick aufs Wasser: hinfahren", "Greift Feinde in der Nähe automatisch an.", "Repariert sich am eigenen Kontor."} {
		c.Text(FSmall, x+18, yy, "· "+l, cDim)
		yy += 22
	}
}
