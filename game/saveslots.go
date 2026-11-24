package game

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------- Mehrere Spielstände

type SaveMeta struct {
	Slot    string
	Name    string
	Time    time.Time
	Mode    int
	Day     int
	Pop     int
	Credits int
	Version string
	Moon    bool
	Diff    int
}

type slotInfo struct {
	SaveMeta
	thumb *Sprite
}

func saveDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	dir = filepath.Join(dir, "NovaArchipel", "Spielstaende")
	os.MkdirAll(dir, 0755)
	return dir
}

func slotPath(slot string) string { return filepath.Join(saveDir(), slot+".json") }

// Spielstand in einen Slot schreiben
func (g *Game) SaveSlot(slot, name string) error {
	data, err := g.saveBytes()
	if err != nil {
		return err
	}
	if err := os.WriteFile(slotPath(slot), data, 0644); err != nil {
		return err
	}
	isl := ""
	if len(g.Islands) > 0 {
		isl = g.Islands[0].Name
	}
	if name == "" {
		name = Tf("%s · Tag %d", isl, g.Day+1)
	}
	m := SaveMeta{Slot: slot, Name: name, Time: time.Now(), Mode: g.Mode, Day: g.Day + 1,
		Pop: g.Pop[0] + g.Pop[1] + g.Pop[2], Credits: int(g.Credits), Version: Version, Moon: g.Other != nil, Diff: g.Diff}
	mb, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(saveDir(), slot+".meta"), mb, 0644)
	g.thumbReq = filepath.Join(saveDir(), slot+".png")
	g.lastSlot = slot
	return nil
}

func LoadSlot(slot string) (*Game, error) {
	data, err := os.ReadFile(slotPath(slot))
	if err != nil {
		return nil, err
	}
	g, err := loadBytes(data)
	if err == nil {
		g.lastSlot = slot
	}
	return g, err
}

func DeleteSlot(slot string) {
	for _, ext := range []string{".json", ".meta", ".png"} {
		os.Remove(filepath.Join(saveDir(), slot+ext))
	}
}

func listSlots() []slotInfo {
	// alten Einzelspielstand übernehmen
	if _, err := os.Stat(savePath()); err == nil {
		if _, err2 := os.Stat(slotPath("alt")); err2 != nil {
			if data, err := os.ReadFile(savePath()); err == nil {
				os.WriteFile(slotPath("alt"), data, 0644)
				m := SaveMeta{Slot: "alt", Name: "Älterer Spielstand", Time: time.Now(), Version: "2.x"}
				mb, _ := json.Marshal(m)
				os.WriteFile(filepath.Join(saveDir(), "alt.meta"), mb, 0644)
			}
		}
	}
	var out []slotInfo
	files, _ := filepath.Glob(filepath.Join(saveDir(), "*.meta"))
	for _, f := range files {
		var m SaveMeta
		b, err := os.ReadFile(f)
		if err != nil || json.Unmarshal(b, &m) != nil {
			continue
		}
		if _, err := os.Stat(slotPath(m.Slot)); err != nil {
			continue
		}
		si := slotInfo{SaveMeta: m}
		si.thumb = loadThumb(filepath.Join(saveDir(), m.Slot+".png"))
		out = append(out, si)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	return out
}

func AnySave() bool {
	if SaveExists() {
		return true
	}
	files, _ := filepath.Glob(filepath.Join(saveDir(), "*.json"))
	return len(files) > 0
}

func newSlotName() string {
	return "spiel-" + time.Now().Format("20060102-150405")
}

// Vorschaubild aus dem letzten Bild schreiben (RGBA, unten links beginnend)
func writeThumb(path string, px []byte, w, h int) {
	const tw, th = 320, 180
	img := image.NewRGBA(image.Rect(0, 0, tw, th))
	// Bildausschnitt im Seitenverhältnis 16:9 aus der Mitte
	cw, ch := w, w*9/16
	if ch > h {
		ch = h
		cw = h * 16 / 9
	}
	ox, oy := (w-cw)/2, (h-ch)/2
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			var r, g, b, n int
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					X := ox + (x*2+sx)*cw/(tw*2)
					Y := oy + (y*2+sy)*ch/(th*2)
					Y = h - 1 - Y
					i := (Y*w + X) * 4
					if i >= 0 && i+2 < len(px) {
						r += int(px[i])
						g += int(px[i+1])
						b += int(px[i+2])
						n++
					}
				}
			}
			if n > 0 {
				img.Set(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255})
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	png.Encode(f, img)
	f.Close()
}

func loadThumb(path string) *Sprite {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil
	}
	b := img.Bounds()
	s := &Sprite{W: 160, H: 90, Pix: make([]uint32, 160*90)}
	for y := 0; y < 90; y++ {
		for x := 0; x < 160; x++ {
			r, gg, bb, _ := img.At(b.Min.X+x*b.Dx()/160, b.Min.Y+y*b.Dy()/90).RGBA()
			s.Pix[y*160+x] = 0xff000000 | (r>>8)<<16 | (gg>>8)<<8 | bb>>8
		}
	}
	return s
}

// ---------------------------------------------------------------- Fenster zum Speichern/Laden

type SlotDialog struct {
	Open   bool
	SaveTo bool // true = speichern, false = laden
	list   []slotInfo
	scroll int
	sel    int
	confirmDel string
}

func (d *SlotDialog) Show(save bool) {
	d.Open, d.SaveTo = true, save
	d.list = listSlots()
	d.scroll, d.sel, d.confirmDel = 0, -1, ""
}

// Liefert ein geladenes Spiel zurück, wenn der Spieler eines lädt
func (d *SlotDialog) Draw(c *Canvas, u *UIState, g *Game) *Game {
	if !d.Open {
		return nil
	}
	var loaded *Game
	c.FillRect(0, 0, c.W, c.H, RGB{2, 6, 12}, 190)
	pw, ph := imin(900, c.W-60), imin(660, c.H-40)
	px, py := c.W/2-pw/2, c.H/2-ph/2
	u.panel(c, px, py, pw, ph)
	title := "Spielstand laden"
	if d.SaveTo {
		title = "Spiel speichern"
	}
	c.Text(FSmall, px+24, py+16, "SPIELSTÄNDE", cAccent)
	c.Text(FBig, px+24, py+34, title, cText)
	if u.button(c, px+pw-130, py+20, 110, 34, "Schließen", true) || In.KeyHit[KEsc] {
		d.Open = false
		In.KeyHit[KEsc] = false
		return nil
	}
	y := py + 86
	if d.SaveTo && g != nil {
		if u.button(c, px+24, y, pw-48, 44, "+  Neuer Spielstand", true) {
			if err := g.SaveSlot(newSlotName(), ""); err == nil {
				g.Notify("Spiel gespeichert.", cGreen)
				PlaySfx(SfxQuest)
				d.Open = false
			} else {
				g.Notify(T("Speichern fehlgeschlagen: ")+err.Error(), cRed)
			}
		}
		y += 56
	}
	rowH := 106
	vis := (py + ph - 20 - y) / rowH
	if In.Wheel != 0 && u.inside(px, y, pw, ph-(y-py)) {
		d.scroll = clampi(d.scroll-int(In.Wheel), 0, imax(0, len(d.list)-vis))
		In.Wheel = 0
	}
	if len(d.list) == 0 {
		c.TextC(FNormal, c.W/2, y+40, "Noch keine Spielstände vorhanden.", cDim)
	}
	modes := []string{"Endlosspiel", "Kampagne", "Endlosspiel"}
	for i := d.scroll; i < len(d.list) && i < d.scroll+vis; i++ {
		s := d.list[i]
		ry := y + (i-d.scroll)*rowH
		hov := u.inside(px+24, ry, pw-48, rowH-8)
		bg := RGB{14, 22, 36}
		if hov {
			bg = RGB{22, 34, 54}
		}
		c.RoundRect(px+24, ry, pw-48, rowH-8, 10, bg, 220)
		if s.thumb != nil {
			c.Blit(s.thumb, px+32, ry+4)
		} else {
			c.RoundRect(px+32, ry+4, 160, 90, 6, RGB{30, 44, 66}, 255)
			c.TextC(FSmall, px+112, ry+40, "kein Bild", cDim)
		}
		tx := px + 210
		c.Text(FBold, tx, ry+10, s.Name, cText)
		md := "Endlosspiel"
		if s.Mode >= 0 && s.Mode < len(modes) {
			md = modes[s.Mode]
		}
		if s.Slot == "schnell" {
			md += " · Schnellspeicher"
		} else if s.Slot == "auto" {
			md += " · Automatisch"
		}
		c.Text(FSmall, tx, ry+36, Tf("%s  ·  %s Einwohner  ·  %s Credits", md, fmtNum(s.Pop), fmtNum(s.Credits)), cDim)
		info := s.Time.Format("02.01.2006  15:04 Uhr")
		if s.Moon {
			info += "  ·  mit Mondbasis"
		}
		c.Text(FSmall, tx, ry+58, info, RGB{120, 135, 155})
		bx := px + pw - 24 - 130
		if d.SaveTo {
			if u.button(c, bx, ry+14, 116, 32, "Überschreiben", true) && g != nil {
				if err := g.SaveSlot(s.Slot, ""); err == nil {
					g.Notify("Spiel gespeichert.", cGreen)
					PlaySfx(SfxQuest)
					d.Open = false
				}
			}
		} else if u.button(c, bx, ry+14, 116, 32, "Laden", true) {
			if ng, err := LoadSlot(s.Slot); err == nil {
				loaded = ng
				d.Open = false
				ng.Notify("Spielstand geladen.", cGreen)
			} else if g != nil {
				g.Notify(T("Laden fehlgeschlagen: ")+err.Error(), cRed)
			}
		}
		lbl := "Löschen"
		if d.confirmDel == s.Slot {
			lbl = "Wirklich?"
		}
		if u.button(c, bx, ry+52, 116, 32, lbl, true) {
			if d.confirmDel == s.Slot {
				DeleteSlot(s.Slot)
				d.list = listSlots()
				d.confirmDel = ""
			} else {
				d.confirmDel = s.Slot
			}
		}
	}
	if len(d.list) > vis {
		c.TextR(FSmall, px+pw-28, py+ph-24, Tf("%d–%d von %d  (Mausrad)", d.scroll+1, imin(len(d.list), d.scroll+vis), len(d.list)), cDim)
	}
	_ = strings.TrimSpace
	return loaded
}
