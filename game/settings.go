package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Einstellungen (werden in %APPDATA%\NovaArchipel\optionen.json gespeichert)
type Settings struct {
	Preset       int     // 0 Niedrig, 1 Mittel, 2 Hoch, 3 Ultra, 4 Benutzerdefiniert
	ShadowRes    int     // 1024/2048/4096
	ShadowSoft   int     // 0 hart, 1 weich, 2 sehr weich
	MSAA         int     // 0,2,4,8
	RenderScale  float64 // 0.5 .. 2.0
	SSAO         bool
	Bloom        bool
	Clouds       bool
	TreeDensity  int // 0 gering, 1 normal, 2 dicht
	ViewDist     int // 0 nah, 1 normal, 2 weit
	VSync        bool
	Fullscreen   bool
	UIScale      float64
	DayNight     bool
	DayLength    int // Minuten Echtzeit pro Tag: 4, 8, 12, 20
	FixedHour    float64
	Weather      bool
	MasterVol    float64
	MusicVol     float64
	SfxVol       float64
	AmbVol       float64
	CamSpeed     float64
	EdgeScroll   bool
	AutoSave     int // Minuten, 0 = aus
	Hints        bool
	Rivals       int
	Pirates      bool
	Disasters    bool
	AutoUpdate   bool
	Reflections  bool
	Particles    bool
	Traffic      bool
	DOF          bool
	Grain        bool
	HeightFog    bool
	GodRays      bool
	KenneyModels bool
	Keys map[string]int
	Lang string
	Life         bool
	GroundDetail bool
}

var Opt = DefaultSettings()

func DefaultSettings() Settings {
	s := Settings{VSync: true, Fullscreen: true, UIScale: 1, DayNight: true, DayLength: 12, FixedHour: 14,
		Weather: true, MasterVol: 0.8, MusicVol: 0.55, SfxVol: 0.8, AmbVol: 0.6, CamSpeed: 1, AutoSave: 10, Hints: true, Rivals: 2, AutoUpdate: true, Pirates: true, Disasters: true, KenneyModels: true}
	s.ApplyPreset(3)
	return s
}

var PresetNames = []string{"Niedrig", "Mittel", "Hoch", "Ultra", "Benutzerdefiniert"}

func (s *Settings) ApplyPreset(p int) {
	s.Preset = p
	switch p {
	case 0:
		s.GodRays = false
		s.Life, s.GroundDetail = false, false
		s.Reflections, s.Particles, s.Traffic, s.DOF, s.Grain, s.HeightFog = false, false, false, false, false, false
		s.ShadowRes, s.ShadowSoft, s.MSAA, s.RenderScale, s.SSAO, s.Bloom, s.Clouds, s.TreeDensity, s.ViewDist = 1024, 0, 0, 0.75, false, false, false, 0, 0
	case 1:
		s.GodRays = false
		s.Life, s.GroundDetail = true, false
		s.Reflections, s.Particles, s.Traffic, s.DOF, s.Grain, s.HeightFog = false, true, true, false, false, true
		s.ShadowRes, s.ShadowSoft, s.MSAA, s.RenderScale, s.SSAO, s.Bloom, s.Clouds, s.TreeDensity, s.ViewDist = 2048, 1, 2, 1, false, true, true, 1, 1
	case 2:
		s.GodRays = true
		s.Life, s.GroundDetail = true, true
		s.Reflections, s.Particles, s.Traffic, s.DOF, s.Grain, s.HeightFog = true, true, true, false, false, true
		s.ShadowRes, s.ShadowSoft, s.MSAA, s.RenderScale, s.SSAO, s.Bloom, s.Clouds, s.TreeDensity, s.ViewDist = 2048, 1, 4, 1, true, true, true, 1, 1
	case 3:
		s.GodRays = true
		s.Life, s.GroundDetail = true, true
		s.Reflections, s.Particles, s.Traffic, s.DOF, s.Grain, s.HeightFog = true, true, true, true, true, true
		s.ShadowRes, s.ShadowSoft, s.MSAA, s.RenderScale, s.SSAO, s.Bloom, s.Clouds, s.TreeDensity, s.ViewDist = 4096, 2, 8, 1.25, true, true, true, 2, 2
	}
}

func optPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	dir = filepath.Join(dir, "NovaArchipel")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "optionen.json")
}

func LoadSettings() {
	data, err := os.ReadFile(optPath())
	if err != nil {
		return
	}
	s := DefaultSettings()
	if json.Unmarshal(data, &s) == nil {
		if s.UIScale < 0.75 || s.UIScale > 2.5 {
			s.UIScale = 1
		}
		if s.RenderScale < 0.5 || s.RenderScale > 2 {
			s.RenderScale = 1
		}
		Opt = s
	}
}

func SaveSettings() {
	data, _ := json.MarshalIndent(Opt, "", "  ")
	os.WriteFile(optPath(), data, 0644)
}
