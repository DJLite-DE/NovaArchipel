package game

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
)

type islandSave struct {
	Owned  bool
	Rival  int
	Stock  [NumGoods]float64
	Sat    [NumGoods]float64
	DemAcc [NumGoods]float64
}

type saveData struct {
	Version string
	Seed    int
	Credits float64
	Time    float64
	MaxPop  [NumTiers]int
	Quest   int
	Won     bool
	NextID  int
	Mode    int
	Chapter int
	Hour    float64
	Day     int
	Deco    []byte
	Road    []byte
	B       []*Building
	Ships   []*Ship
	Islands []islandSave
	CamX    float64
	CamY    float64
	Dist    float64
	Yaw     float64
	Rivals  []*Rival
	BP      []*Building
	Hist    []HistPoint
	RB      []*Building
	Moon    *moonSave
	Tech    [NumTechs]bool
	RP      float64
	Stats   GameStats
	Tut     int
	Map     MapOptions
	Diff    int
	Reg     int
	RocketT float64
}

type moonSave struct {
	Islands []islandSave
	B       []*Building
	BP      []*Building
	Ships   []*Ship
	Road    []byte
	TX, TZ  float64
	Dist    float64
	Yaw     float64
}

func roadBytes(w *World) []byte {
	r := make([]byte, len(w.Road))
	for i, v := range w.Road {
		if v {
			r[i] = 1
		}
	}
	return r
}

func islandSaves(is []*Island) []islandSave {
	var out []islandSave
	for _, I := range is {
		out = append(out, islandSave{I.Owned, I.Rival, I.Stock, I.Sat, I.DemAcc})
	}
	return out
}

func restoreIslands(is []*Island, sv []islandSave) {
	for i, s := range sv {
		if i < len(is) {
			I := is[i]
			I.Owned, I.Rival, I.Stock, I.Sat, I.DemAcc = s.Owned, s.Rival, s.Stock, s.Sat, s.DemAcc
		}
	}
}

func placeLoaded(w *World, bs []*Building) {
	for _, b := range bs {
		bt := &BTypes[b.T]
		b.BaseY = w.Flatten(b.X, b.Y, bt.W, bt.H)
		for yy := b.Y; yy < b.Y+bt.H; yy++ {
			for xx := b.X; xx < b.X+bt.W; xx++ {
				w.Occ[w.Idx(xx, yy)] = int32(b.ID + 1)
			}
		}
	}
}

func savePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	dir = filepath.Join(dir, "NovaArchipel")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "spielstand.json")
}

func SaveExists() bool {
	_, err := os.Stat(savePath())
	return err == nil
}

// Kompatibel: Autospeicher
func (g *Game) Save() error { return g.SaveSlot("auto", "Automatisch gespeichert") }

func (g *Game) saveBytes() ([]byte, error) {
	reg := g.Reg
	ecam := [4]float64{g.UI.TX, g.UI.TZ, g.UI.Dist, g.UI.Yaw}
	var mcam [4]float64
	if g.Other != nil {
		mcam = [4]float64{g.Other.TX, g.Other.TZ, g.Other.Dist, g.Other.Yaw}
	}
	if reg == 1 {
		// Erde in den Hauptfeldern speichern
		ecam, mcam = mcam, ecam
		g.swapRegion()
		defer func() {
			g.swapRegion()
			g.Other.TX, g.Other.TZ, g.Other.Dist, g.Other.Yaw = ecam[0], ecam[1], ecam[2], ecam[3]
		}()
	}
	sd := saveData{Reg: reg, RocketT: g.RocketT, Map: g.Map, Diff: g.Diff, Tech: g.Tech, RP: g.RP, Stats: g.Stats, Day: g.Day, Mode: g.Mode, Chapter: g.Chapter, Hour: g.Hour, Version: Version, Seed: g.Seed, Credits: g.Credits, Time: g.Time, MaxPop: g.MaxPop, Quest: g.Quest, Won: g.Won, NextID: g.NextID,
		Deco: g.W.Deco, B: g.B, Ships: g.Ships, CamX: g.UI.TX, CamY: g.UI.TZ, Dist: g.UI.Dist, Yaw: g.UI.Yaw, Rivals: g.Rivals, RB: g.RB, BP: g.BP, Hist: g.Hist}
	sd.Road = make([]byte, len(g.W.Road))
	for i, r := range g.W.Road {
		if r {
			sd.Road[i] = 1
		}
	}
	for _, I := range g.Islands {
		sd.Islands = append(sd.Islands, islandSave{I.Owned, I.Rival, I.Stock, I.Sat, I.DemAcc})
	}
	sd.CamX, sd.CamY, sd.Dist, sd.Yaw = ecam[0], ecam[1], ecam[2], ecam[3]
	if g.tut != nil {
		sd.Tut = g.tut.Step
	}
	if o := g.Other; o != nil {
		sd.Moon = &moonSave{Islands: islandSaves(o.Islands), B: o.B, BP: o.BP, Ships: o.Ships, Road: roadBytes(o.W), TX: mcam[0], TZ: mcam[1], Dist: mcam[2], Yaw: mcam[3]}
	}
	return json.Marshal(sd)
}

// Neuester Spielstand (für Tests und „Fortsetzen“)
func LoadGame() (*Game, error) {
	if l := listSlots(); len(l) > 0 {
		return LoadSlot(l[0].Slot)
	}
	data, err := os.ReadFile(savePath())
	if err != nil {
		return nil, err
	}
	return loadBytes(data)
}

func loadBytes(data []byte) (*Game, error) {
	var sd saveData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, err
	}
	g := &Game{Seed: sd.Seed, Speed: 1, Credits: sd.Credits, Time: sd.Time, MaxPop: sd.MaxPop, Quest: sd.Quest, Won: sd.Won, NextID: sd.NextID}
	g.rng = rand.New(rand.NewSource(int64(sd.Seed)))
	g.Mode, g.Chapter, g.Hour, g.Day, g.stormT = sd.Mode, sd.Chapter, sd.Hour, sd.Day, 400
	MapOpt = sd.Map
	g.Map, g.Diff = sd.Map, sd.Diff
	g.Tech, g.RP, g.Stats = sd.Tech, sd.RP, sd.Stats
	if sd.Mode == ModeTutorial {
		g.tut = &tutState{Step: sd.Tut}
	}
	g.W, g.Islands = GenWorld(sd.Seed)
	g.UI = NewUIState()
	g.UI.TX, g.UI.TZ = sd.CamX, sd.CamY
	if sd.Dist > 0 {
		g.UI.Dist, g.UI.Yaw = sd.Dist, sd.Yaw
	}
	g.UI.world = g.W
	g.UI.WinShown = sd.Won
	if len(sd.Deco) == len(g.W.Deco) {
		copy(g.W.Deco, sd.Deco)
	}
	for i := range g.W.Road {
		if i < len(sd.Road) {
			g.W.Road[i] = sd.Road[i] == 1
		}
	}
	for i, is := range sd.Islands {
		if i < len(g.Islands) {
			I := g.Islands[i]
			I.Owned, I.Rival, I.Stock, I.Sat, I.DemAcc = is.Owned, is.Rival, is.Stock, is.Sat, is.DemAcc
		}
	}
	g.B = sd.B
	g.RB = sd.RB
	g.BP = sd.BP
	g.Hist = sd.Hist
	g.Rivals = sd.Rivals
	for _, b := range append(g.allBuildings(), g.BP...) {
		bt := &BTypes[b.T]
		b.BaseY = g.W.Flatten(b.X, b.Y, bt.W, bt.H)
		for yy := b.Y; yy < b.Y+bt.H; yy++ {
			for xx := b.X; xx < b.X+bt.W; xx++ {
				g.W.Occ[g.W.Idx(xx, yy)] = int32(b.ID + 1)
			}
		}
	}
	g.Ships = sd.Ships
	g.RocketT = sd.RocketT
	if m := sd.Moon; m != nil {
		w, isl := GenMoon(sd.Seed)
		restoreIslands(isl, m.Islands)
		for i := range w.Road {
			if i < len(m.Road) {
				w.Road[i] = m.Road[i] == 1
			}
		}
		placeLoaded(w, append(append([]*Building(nil), m.B...), m.BP...))
		g.Other = &Region{W: w, Islands: isl, B: m.B, BP: m.BP, Ships: m.Ships, Dirty: true, TX: m.TX, TZ: m.TZ, Dist: m.Dist, Yaw: m.Yaw}
		// Mond-Region einmal durchrechnen
		g.swapRegion()
		g.Recompute()
		g.swapRegion()
		if sd.Reg == 1 {
			g.SwitchRegion()
		}
	}
	for i := range g.unlockSeen {
		g.unlockSeen[i] = g.Unlocked(i)
	}
	g.dirty = true
	g.Recompute()
	return g, nil
}

// Automatisches Speichern beim Beenden
func (g *Game) SaveAuto() { g.Save() }
