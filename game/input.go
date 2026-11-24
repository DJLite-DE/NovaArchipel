package game

// Tastencodes (plattformunabhängig)
const (
	KNone = iota
	KLeft
	KRight
	KUp
	KDown
	KW
	KA
	KS
	KD
	KEsc
	KSpace
	K1
	K2
	K3
	K4
	KF5
	KF9
	KPlus
	KMinus
	KDel
	KShift
	KTab
	KH
	KF11
	KQ
	KE
	KB
	KZ
	KCtrl
	KO
	NumKeys
)

type InputState struct {
	MX, MY     int
	LDown      bool
	RDown      bool
	MDown      bool
	LPressed   bool
	LReleased  bool
	RPressed   bool
	RReleased  bool
	PressX     int
	PressY     int
	RPressX    int
	RPressY    int
	Wheel      int
	Keys       [NumKeys]bool
	KeyHit     [NumKeys]bool
	Quit       bool
	LastVK     int // zuletzt gedrückte Taste (für Tastenbelegung)
	ToggleFull bool
}

var In InputState

func (in *InputState) EndFrame() {
	in.LPressed, in.LReleased, in.RPressed, in.RReleased = false, false, false, false
	in.Wheel = 0
	in.KeyHit = [NumKeys]bool{}
}

func (in *InputState) MouseMove(x, y int) { in.MX, in.MY = x, y }
func (in *InputState) MouseButton(btn int, down bool) {
	switch btn {
	case 0:
		if down && !in.LDown {
			in.LPressed = true
			in.PressX, in.PressY = in.MX, in.MY
		}
		if !down && in.LDown {
			in.LReleased = true
		}
		in.LDown = down
	case 1:
		if down && !in.RDown {
			in.RPressed = true
			in.RPressX, in.RPressY = in.MX, in.MY
		}
		if !down && in.RDown {
			in.RReleased = true
		}
		in.RDown = down
	case 2:
		in.MDown = down
	}
}
func (in *InputState) Key(k int, down bool) {
	if k <= 0 || k >= NumKeys {
		return
	}
	if down && !in.Keys[k] {
		in.KeyHit[k] = true
	}
	in.Keys[k] = down
}

// ---------------------------------------------------------------- Tastenbelegung

type KeyAction struct {
	K    int
	Name string
	Def  int // Windows-Tastencode
}

var KeyActions = []KeyAction{
	{KW, "Kamera vor", 'W'}, {KS, "Kamera zurück", 'S'}, {KA, "Kamera links", 'A'}, {KD, "Kamera rechts", 'D'},
	{KQ, "Drehen links", 'Q'}, {KE, "Drehen rechts", 'E'}, {KSpace, "Pause", 0x20},
	{KB, "Blaupausen-Modus", 'B'}, {KO, "Überlagerung wechseln", 'O'}, {KDel, "Abriss-Werkzeug", 0x2E},
	{KZ, "Rückgängig (mit Strg)", 'Z'}, {KF5, "Schnellspeichern", 0x74}, {KF9, "Schnellladen", 0x78}, {KH, "Hilfe", 'H'},
}

func keyBinding(k int) int {
	for _, a := range KeyActions {
		if a.K == k {
			if v, ok := Opt.Keys[a.Name]; ok && v > 0 {
				return v
			}
			return a.Def
		}
	}
	return 0
}

// Windows-Tastencode → Spieltaste (nur frei belegbare Aktionen)
func BoundKey(vk int) int {
	for _, a := range KeyActions {
		if keyBinding(a.K) == vk {
			return a.K
		}
	}
	return KNone
}

func VKName(vk int) string {
	switch {
	case vk >= 'A' && vk <= 'Z', vk >= '0' && vk <= '9':
		return string(rune(vk))
	case vk >= 0x70 && vk <= 0x7B:
		return "F" + itoa(vk-0x6F)
	case vk >= 0x60 && vk <= 0x69:
		return T("Num ") + itoa(vk-0x60)
	}
	n := map[int]string{0x20: "Leertaste", 0x2E: "Entf", 0x2D: "Einfg", 0x24: "Pos1", 0x23: "Ende", 0x21: "Bild auf", 0x22: "Bild ab",
		0x08: "Rücktaste", 0x0D: "Enter", 0xBC: ",", 0xBE: ".", 0xBA: "Ü", 0xC0: "Ö", 0xDE: "Ä", 0xDB: "ß", 0xBF: "#"}
	if s, ok := n[vk]; ok {
		return s
	}
	return T("Taste ") + itoa(vk)
}

