package game

import "fmt"

var waitKey = -1

var GeomDirty bool // z. B. Vegetationsdichte geändert

var optTabs = []string{"Grafik", "Audio", "Spiel", "Steuerung"}

// Optionsmenü (Hauptmenü und Pausemenü)
func DrawOptions(c *Canvas, u *UIState, onClose func()) {
	c.FillRect(0, 0, c.W, c.H, RGB{0, 0, 0}, 110)
	w, h := imin(860, c.W-40), imin(700, c.H-40)
	x, y := c.W/2-w/2, c.H/2-h/2
	u.panel(c, x, y, w, h)
	c.TextS(FBig, x+28, y+20, "Optionen", cText)
	// Tabs
	tx := x + 28
	for i, t := range optTabs {
		tw := TextW(FBold, t) + 34
		sel := u.optTab == i
		hov := u.inside(tx, y+66, tw, 34)
		if sel {
			c.RoundRect(tx, y+66, tw, 34, 8, uiPurple, 220)
		} else if hov {
			c.RoundRect(tx, y+66, tw, 34, 8, RGB{40, 60, 85}, 200)
		}
		c.Text(FBold, tx+17, y+72, t, cText)
		if u.clicked(tx, y+66, tw, 34) {
			u.optTab = i
			PlaySfx(SfxClick)
		}
		tx += tw + 8
	}
	c.FillRect(x+28, y+108, w-56, 1, RGB{255, 255, 255}, 40)
	row := y + 124
	lx := x + 36
	cx := x + w - 330
	cw := 290
	changedGfx := false
	rowH := 42
	label := func(s string) {
		c.Text(FNormal, lx, row+6, s, cText)
	}
	choice := func(name string, opts []string, idx int) int {
		label(name)
		ni := idx
		if u.button(c, cx, row, 36, 32, "<", true) {
			ni = (idx - 1 + len(opts)) % len(opts)
		}
		c.RoundRect(cx+42, row, cw-84, 32, 6, RGB{20, 30, 44}, 200)
		c.TextC(FBold, cx+cw/2, row+6, opts[clampi(idx, 0, len(opts)-1)], cAccent)
		if u.button(c, cx+cw-36, row, 36, 32, ">", true) {
			ni = (idx + 1) % len(opts)
		}
		row += rowH
		if ni != idx {
			PlaySfx(SfxClick)
		}
		return ni
	}
	toggle := func(name string, v bool) bool {
		label(name)
		bx := cx + cw - 64
		on := v
		col := RGB{60, 70, 84}
		if on {
			col = uiPurpleL
		}
		c.RoundRect(bx, row+4, 64, 26, 13, col, 255)
		kx := bx + 4
		if on {
			kx = bx + 64 - 26
		}
		c.Circle(float64(kx+11), float64(row+17), 10, RGB{245, 248, 252}, 255)
		c.Text(FSmall, cx, row+8, map[bool]string{true: "An", false: "Aus"}[on], cDim)
		if u.clicked(bx-10, row, 84, 34) {
			on = !on
			PlaySfx(SfxClick)
		}
		row += rowH
		return on
	}
	slider := func(id int, name string, v, lo, hi float64, f string) float64 {
		label(name)
		sx, sw := cx, cw-60
		t := (v - lo) / (hi - lo)
		c.RoundRect(sx, row+13, sw, 6, 3, RGB{40, 52, 68}, 255)
		c.RoundRect(sx, row+13, int(float64(sw)*t), 6, 3, uiPurpleL, 255)
		c.Circle(float64(sx)+float64(sw)*t, float64(row+16), 9, RGB{245, 248, 252}, 255)
		c.TextR(FBold, cx+cw, row+6, fmt.Sprintf(f, v), cAccent)
		if In.LPressed && u.inside(sx-10, row, sw+20, 32) {
			u.dragSlider = id
		}
		if !In.LDown {
			u.dragSlider = 0
		}
		if u.dragSlider == id {
			t = clampf(float64(In.MX-sx)/float64(sw), 0, 1)
			v = lo + t*(hi-lo)
			u.consumed = true
		}
		row += rowH
		return v
	}
	idxOf := func(v int, list []int) int {
		for i, x := range list {
			if x == v {
				return i
			}
		}
		return 0
	}
	idxOfF := func(v float64, list []float64) int {
		best, bd := 0, 1e9
		for i, x := range list {
			if d := (x - v) * (x - v); d < bd {
				best, bd = i, d
			}
		}
		return best
	}
	switch u.optTab {
	case 0:
		rowH = 33
		lx, cx, cw = x+36, x+226, 180
		p := choice("Qualitätsstufe", PresetNames, Opt.Preset)
		if p != Opt.Preset {
			if p == 4 {
				Opt.Preset = 4
			} else {
				Opt.ApplyPreset(p)
				GeomDirty = true
			}
		}
		scales := []float64{0.5, 0.75, 1, 1.25, 1.5, 2}
		si := choice("Auflösungsskalierung", []string{"50 %", "75 %", "100 %", "125 %", "150 %", "200 %"}, idxOfF(Opt.RenderScale, scales))
		if scales[si] != Opt.RenderScale {
			Opt.RenderScale = scales[si]
			changedGfx = true
		}
		ms := []int{0, 2, 4, 8}
		mi := choice("Kantenglättung (MSAA)", []string{"Aus", "2×", "4×", "8×"}, idxOf(Opt.MSAA, ms))
		if ms[mi] != Opt.MSAA {
			Opt.MSAA = ms[mi]
			changedGfx = true
		}
		sr := []int{1024, 2048, 4096}
		ri := choice("Schattenauflösung", []string{"Niedrig (1K)", "Hoch (2K)", "Ultra (4K)"}, idxOf(Opt.ShadowRes, sr))
		if sr[ri] != Opt.ShadowRes {
			Opt.ShadowRes = sr[ri]
			changedGfx = true
		}
		if v := choice("Schattenkanten", []string{"Hart", "Weich", "Sehr weich"}, Opt.ShadowSoft); v != Opt.ShadowSoft {
			Opt.ShadowSoft = v
			changedGfx = true
		}
		if v := choice("Vegetationsdichte", []string{"Gering", "Normal", "Dicht"}, Opt.TreeDensity); v != Opt.TreeDensity {
			Opt.TreeDensity = v
			GeomDirty = true
			changedGfx = true
		}
		if v := choice("Sichtweite", []string{"Nah", "Normal", "Weit"}, Opt.ViewDist); v != Opt.ViewDist {
			Opt.ViewDist = v
			changedGfx = true
		}
		row = y + 124
		lx, cx, cw = x+w/2+24, x+w-216, 180
		if v := toggle("SSAO (Verdeckung)", Opt.SSAO); v != Opt.SSAO {
			Opt.SSAO = v
			changedGfx = true
		}
		if v := toggle("Bloom-Leuchten", Opt.Bloom); v != Opt.Bloom {
			Opt.Bloom = v
			changedGfx = true
		}
		if v := toggle("Wolkenschatten", Opt.Clouds); v != Opt.Clouds {
			Opt.Clouds = v
			changedGfx = true
		}
		tg := func(name string, v *bool) {
			if nv := toggle(name, *v); nv != *v {
				*v = nv
				changedGfx = true
			}
		}
		tg("Wasserspiegelungen", &Opt.Reflections)
		tg("Rauch & Dampf", &Opt.Particles)
		tg("Gleiter-Verkehr", &Opt.Traffic)
		tg("Tiefenschärfe", &Opt.DOF)
		tg("Höhennebel", &Opt.HeightFog)
		tg("Filmkorn", &Opt.Grain)
		tg("Bewohner & Tiere", &Opt.Life)
		tg("Gras & Blumen", &Opt.GroundDetail)
		tg("Lichtstrahlen", &Opt.GodRays)
		if nv := toggle("Detailmodelle (Kenney)", Opt.KenneyModels); nv != Opt.KenneyModels {
			Opt.KenneyModels = nv
			GeomDirty = true
		}
		Opt.VSync = toggle("VSync", Opt.VSync)
		Opt.Fullscreen = toggle("Vollbild", Opt.Fullscreen)
		row = y + 124 + 7*rowH
		lx, cx, cw = x+36, x+226, 180
		uis := []float64{0.75, 1, 1.25, 1.5, 2}
		ui := choice("UI-Skalierung", []string{"75 %", "100 %", "125 %", "150 %", "200 %"}, idxOfF(Opt.UIScale, uis))
		Opt.UIScale = uis[ui]
		if changedGfx {
			Opt.Preset = 4
		}
	case 1:
		Opt.MasterVol = slider(1, "Gesamtlautstärke", Opt.MasterVol*100, 0, 100, "%.0f %%") / 100
		Opt.MusicVol = slider(2, "Musik", Opt.MusicVol*100, 0, 100, "%.0f %%") / 100
		Opt.SfxVol = slider(3, "Effekte", Opt.SfxVol*100, 0, 100, "%.0f %%") / 100
		Opt.AmbVol = slider(4, "Umgebung (Meer, Regen)", Opt.AmbVol*100, 0, 100, "%.0f %%") / 100
		c.Text(FSmall, lx, row+10, "Musik und Klänge werden live vom Spiel erzeugt – kein Download nötig.", cDim)
	case 2:
		Opt.DayNight = toggle("Tag-Nacht-Wechsel", Opt.DayNight)
		dl := []int{4, 8, 12, 20, 40}
		di := choice("Länge eines Tages", []string{"4 Minuten", "8 Minuten", "12 Minuten", "20 Minuten", "40 Minuten"}, idxOf(Opt.DayLength, dl))
		Opt.DayLength = dl[di]
		if !Opt.DayNight {
			Opt.FixedHour = slider(5, "Feste Uhrzeit", Opt.FixedHour, 0, 23.99, "%.1f Uhr")
		}
		Opt.Weather = toggle("Wetter und Stürme", Opt.Weather)
		Opt.CamSpeed = slider(6, "Kamerageschwindigkeit", Opt.CamSpeed, 0.4, 2.5, "%.1f×")
		as := []int{0, 5, 10, 15, 30}
		ai := choice("Automatisch speichern", []string{"Aus", "alle 5 Min.", "alle 10 Min.", "alle 15 Min.", "alle 30 Min."}, idxOf(Opt.AutoSave, as))
		Opt.AutoSave = as[ai]
		Opt.Hints = toggle("Hinweise anzeigen", Opt.Hints)
		Opt.Rivals = choice("KI-Gegner (neues Endlosspiel)", []string{"Keine", "1 Gegner", "2 Gegner", "3 Gegner"}, clampi(Opt.Rivals, 0, 3))
		Opt.Pirates = toggle("Piraten", Opt.Pirates)
		Opt.Disasters = toggle("Brände & Stromausfälle", Opt.Disasters)
		if li := choice("Sprache / Language", []string{"Deutsch", "English"}, map[bool]int{true: 1, false: 0}[Opt.Lang == "en"]); (li == 1) != (Opt.Lang == "en") {
			Opt.Lang = []string{"de", "en"}[li]
			tMu.Lock()
			tCache = map[string]string{}
			tMu.Unlock()
		}
		Opt.AutoUpdate = toggle("Beim Start nach Updates suchen", Opt.AutoUpdate)
	case 3:
		// frei belegbare Tasten in zwei Spalten
		for i, a := range KeyActions {
			col, r := i/7, i%7
			kx := lx + col*430
			ky := row + r*42
			c.Text(FNormal, kx, ky+6, a.Name, cText)
			lbl := VKName(keyBinding(a.K))
			if waitKey == a.K {
				lbl = "Taste drücken …"
				if In.LastVK != 0 {
					if In.LastVK != 0x1B {
						if Opt.Keys == nil {
							Opt.Keys = map[string]int{}
						}
						// Doppelbelegung auflösen
						for _, o := range KeyActions {
							if o.K != a.K && keyBinding(o.K) == In.LastVK {
								Opt.Keys[o.Name] = keyBinding(a.K)
							}
						}
						Opt.Keys[a.Name] = In.LastVK
					}
					waitKey = -1
					In.KeyHit = [NumKeys]bool{}
				}
			}
			if u.button(c, kx+230, ky, 160, 34, lbl, true) {
				waitKey = a.K
				In.LastVK = 0
			}
		}
		yy := row + 7*42 + 16
		for i, l := range []string{"Rechte Maustaste ziehen: Kamera schieben  ·  Mittlere Maustaste: drehen  ·  Mausrad: zoomen",
			"Linksklick: auswählen/bauen  ·  Rechtsklick: Schiff schicken / Werkzeug abbrechen  ·  Strg + Rückgängig-Taste",
			"Esc: Menü  ·  1–4: Geschwindigkeit  ·  F11: Vollbild"} {
			c.Text(FSmall, lx, yy+i*24, l, cDim)
		}
	}
	if u.button(c, x+28, y+h-58, 160, 40, "Standard", true) {
		keep := Opt.Fullscreen
		Opt = DefaultSettings()
		Opt.Fullscreen = keep
		GeomDirty = true
	}
	if u.button(c, x+w-188, y+h-58, 160, 40, "Fertig", true) || In.KeyHit[KEsc] {
		SaveSettings()
		onClose()
	}
	u.consumed = true
}

func (u *UIState) OptTabForShots(t int) { u.optTab = t }
