package game

import (
	"math"
	"math/rand"
)

const MapN = 168

// Terrain
const (
	TWater = iota
	TSand
	TGrass
	TRock
)

type World struct {
	N      int
	Terr   []uint8
	Isl    []int16 // Insel-ID oder -1
	Deco   []uint8 // 0 nichts, 1..3 Bäume, 4 Felsen, 5 Kristallgewächs
	Dep    []uint8 // Vorkommen
	Dist   []uint8 // Wasser: Abstand zum Land
	Elev   []uint8 // Terrain-Klasse für Darstellung 0..5
	Occ    []int32 // Gebäude-ID+1
	Road   []bool
	Infl   []bool // Einfluss des Spielers
	RoadC  []int32
	Biome  []uint8
	LDist  []uint8   // Land: Abstand zum Wasser
	Hgt    []float32 // Eckhöhen (N+1)^2 für 3D
	HDirty bool
	Light  []float32 // Eck-Helligkeit (N+1)^2 für Hügel-Schattierung
	Seed   int
	Moon   bool
}

type Island struct {
	ID       int
	Name     string
	CX, CY   int
	Tiles    int
	Fert     [3]bool
	Biome    int
	Owned    bool
	Rival    int
	Blackout float64
	Stock    [NumGoods]float64
	Cap      int
	EProd    int
	EUse     int
	Pop      [NumTiers]int
	Sat      [NumGoods]float64
	DemAcc   [NumGoods]float64
	Produced [NumGoods]float64 // gleitende Rate pro Minute
	Consumed [NumGoods]float64
}

func (w *World) Idx(x, y int) int { return y*w.N + x }
func (w *World) In(x, y int) bool { return x >= 0 && y >= 0 && x < w.N && y < w.N }
func (w *World) Land(x, y int) bool {
	return w.In(x, y) && w.Terr[w.Idx(x, y)] != TWater
}
func (w *World) Water(x, y int) bool {
	return !w.In(x, y) || w.Terr[w.Idx(x, y)] == TWater
}

// ------------------------------------------------------------ Rauschen

func hash2(x, y int, seed int) float64 {
	h := uint32(x*374761393 + y*668265263 + seed*1442695041)
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xFFFFFF) / float64(0xFFFFFF)
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

func vnoise(x, y float64, seed int) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := smooth(x-float64(xi)), smooth(y-float64(yi))
	a := hash2(xi, yi, seed)
	b := hash2(xi+1, yi, seed)
	c := hash2(xi, yi+1, seed)
	d := hash2(xi+1, yi+1, seed)
	return a + (b-a)*fx + (c-a)*fy + (a-b-c+d)*fx*fy
}

func fbm(x, y float64, seed int) float64 {
	v, amp, f := 0.0, 0.5, 1.0
	for i := 0; i < 4; i++ {
		v += amp * vnoise(x*f, y*f, seed+i*31)
		amp *= 0.5
		f *= 2
	}
	return v
}

// ------------------------------------------------------------ Generator

type islandDef struct {
	name       string
	x, y, r    float64
	fert       []int
	erz, sil   int
	biome      int
	elong, rot float64
	he3        int
}

var islandDefs = []islandDef{
	{"Neu-Aurora", 62, 84, 25, []int{FertFasern}, 2, 0, 0, 1.25, 0.6, 0},
	{"Korallenspitze", 104, 52, 15, []int{FertKaffee}, 0, 0, 1, 1.4, -0.4, 0},
	{"Silberklippe", 114, 112, 17, []int{FertKaffee}, 0, 2, 1, 1.2, 1.1, 0},
	{"Nebelhorn", 28, 36, 13, []int{FertFasern}, 1, 0, 0, 1.3, 0.2, 0},
	{"Glutfels", 140, 28, 10, nil, 1, 1, 1, 1.1, 0, 0},
	{"Smaragdriff", 36, 136, 14, []int{FertKaffee, FertFasern}, 0, 1, 0, 1.5, 2.0, 0},
	{"Sturmwacht", 70, 22, 8, []int{FertFasern}, 0, 0, 0, 1.2, 0.8, 0},
	{"Kap Nova", 145, 78, 9, []int{FertKaffee}, 1, 0, 1, 1.3, 1.6, 0},
	{"Leuchtfeuer", 92, 150, 7, nil, 0, 0, 0, 1.2, 0.3, 0},
}

// Kartenoptionen für neue Spiele
type MapOptions struct {
	Size    float64 // Inselgröße (0.85 … 1.15)
	Rich    int     // Rohstoffe: -1 knapp, 0 normal, +1 reich
	Islands int     // Anzahl Inseln (5 … 9)
}

var MapOpt = MapOptions{Size: 1, Islands: 9}

func GenWorld(seed int) (*World, []*Island) {
	o := MapOpt
	if o.Size == 0 {
		o.Size = 1
	}
	n := clampi(o.Islands, 4, len(islandDefs))
	if o.Islands == 0 {
		n = len(islandDefs)
	}
	defs := make([]islandDef, 0, n)
	for i := 0; i < n; i++ {
		d := islandDefs[i]
		d.r *= o.Size
		if i > 0 {
			d.erz = imax(0, d.erz+o.Rich)
			d.sil = imax(0, d.sil+o.Rich)
			if o.Rich > 0 && d.erz+d.sil == 0 {
				d.erz = 1
			}
		}
		defs = append(defs, d)
	}
	return genWorldDefs(seed, defs, false)
}

var moonDefs = []islandDef{
	{name: "Tranquility-Plateau", x: 74, y: 82, r: 24, erz: 1, sil: 1, he3: 3, elong: 1.3, rot: 0.4},
	{name: "Shackleton-Rand", x: 118, y: 54, r: 14, erz: 1, he3: 2, elong: 1.2, rot: -0.6},
	{name: "Kopernikus", x: 48, y: 128, r: 13, sil: 1, he3: 2, elong: 1.3, rot: 1.2},
	{name: "Mare Nova", x: 116, y: 118, r: 11, he3: 2, elong: 1.4, rot: 0.2},
}

func GenMoon(seed int) (*World, []*Island) { return genWorldDefs(seed+777, moonDefs, true) }

func genWorldDefs(seed int, defs []islandDef, moon bool) (*World, []*Island) {
	n := MapN
	w := &World{N: n, Seed: seed, Moon: moon}
	sz := n * n
	w.Terr = make([]uint8, sz)
	w.Isl = make([]int16, sz)
	w.Deco = make([]uint8, sz)
	w.Dep = make([]uint8, sz)
	w.Dist = make([]uint8, sz)
	w.Elev = make([]uint8, sz)
	w.Occ = make([]int32, sz)
	w.Road = make([]bool, sz)
	w.Infl = make([]bool, sz)
	w.RoadC = make([]int32, sz)
	w.Biome = make([]uint8, sz)
	rng := rand.New(rand.NewSource(int64(seed)))
	var isl []*Island
	owner := make([]int16, sz)
	for i := range owner {
		owner[i] = -1
		w.Isl[i] = -1
	}
	for id, d := range defs {
		ca, sa := math.Cos(d.rot), math.Sin(d.rot)
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				dx, dy := float64(x)-d.x, float64(y)-d.y
				u := (dx*ca + dy*sa) / d.elong
				v := (-dx*sa + dy*ca) * 1.0
				dist := math.Hypot(u, v) / d.r
				nv := fbm(float64(x)*0.09, float64(y)*0.09, seed+id*7)
				val := 1 - dist + (nv-0.5)*0.75
				if val > 0.12 {
					i := w.Idx(x, y)
					if owner[i] == -1 {
						owner[i] = int16(id)
						w.Terr[i] = TGrass
					}
				}
			}
		}
		isl = append(isl, &Island{ID: id, Name: d.name, CX: int(d.x), CY: int(d.y), Biome: d.biome})
		for _, f := range d.fert {
			isl[id].Fert[f] = true
		}
	}
	// Kleine Landflecken entfernen, Ränder freihalten
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := w.Idx(x, y)
			if x < 3 || y < 3 || x >= n-3 || y >= n-3 {
				w.Terr[i] = TWater
				owner[i] = -1
			}
		}
	}
	for pass := 0; pass < 2; pass++ {
		for y := 1; y < n-1; y++ {
			for x := 1; x < n-1; x++ {
				i := w.Idx(x, y)
				if w.Terr[i] == TWater {
					continue
				}
				c := 0
				for _, dd := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if w.Land(x+dd[0], y+dd[1]) {
						c++
					}
				}
				if c <= 1 {
					w.Terr[i] = TWater
					owner[i] = -1
				}
			}
		}
	}
	for i := range owner {
		w.Isl[i] = owner[i]
		if owner[i] >= 0 {
			isl[owner[i]].Tiles++
			w.Biome[i] = uint8(isl[owner[i]].Biome)
		}
	}
	// Strände
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := w.Idx(x, y)
			if w.Terr[i] != TGrass {
				continue
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if w.Water(x+dx, y+dy) {
						w.Terr[i] = TSand
					}
				}
			}
		}
	}
	// Gebirge: Inselinneres mit Rauschen
	for id, d := range defs {
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				i := w.Idx(x, y)
				if int(w.Isl[i]) != id || w.Terr[i] != TGrass {
					continue
				}
				m := fbm(float64(x)*0.13, float64(y)*0.13, seed+500+id)
				// Gebirge eher am Rand des Inneren, nicht im Zentrum
				dc := math.Hypot(float64(x)-d.x, float64(y)-d.y) / d.r
				if m > 0.63 && dc > 0.25 && !nearType(w, x, y, TSand, 2) {
					w.Terr[i] = TRock
				}
			}
		}
		// Vorkommen
		place := func(kind uint8, count int) {
			for c := 0; c < count; c++ {
				best := -1
				bestD := 1e9
				for try := 0; try < 4000; try++ {
					x := int(d.x) + rng.Intn(int(d.r*2)) - int(d.r)
					y := int(d.y) + rng.Intn(int(d.r*2)) - int(d.r)
					if !w.In(x+1, y+1) || !w.In(x, y) {
						continue
					}
					ok := true
					for yy := y - 1; yy <= y+2 && ok; yy++ {
						for xx := x - 1; xx <= x+2; xx++ {
							j := w.Idx(xx, yy)
							if !w.In(xx, yy) || int(w.Isl[j]) != id || w.Terr[j] == TWater || w.Terr[j] == TSand || w.Dep[j] != 0 {
								ok = false
								break
							}
						}
					}
					if !ok {
						continue
					}
					dd := math.Abs(math.Hypot(float64(x)-d.x, float64(y)-d.y) - d.r*0.45)
					if dd < bestD {
						bestD = dd
						best = w.Idx(x, y)
					}
					if try > 600 && best >= 0 {
						break
					}
				}
				if best >= 0 {
					bx, by := best%n, best/n
					for yy := by - 1; yy <= by+2; yy++ {
						for xx := bx - 1; xx <= bx+2; xx++ {
							j := w.Idx(xx, yy)
							w.Terr[j] = TRock
						}
					}
					for yy := by; yy <= by+1; yy++ {
						for xx := bx; xx <= bx+1; xx++ {
							w.Dep[w.Idx(xx, yy)] = kind
						}
					}
				}
			}
		}
		place(DepErz, d.erz)
		place(DepSil, d.sil)
		place(DepHe3, d.he3)
	}
	// Vegetation
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := w.Idx(x, y)
			if moon {
				// Krater-Geröll statt Pflanzen
				if (w.Terr[i] == TRock && w.Dep[i] == 0 && hash2(x, y, seed+6) < 0.5) || (w.Terr[i] == TGrass && hash2(x, y, seed+7) < 0.04) {
					w.Deco[i] = 4
				}
				continue
			}
			if w.Terr[i] == TGrass {
				f := fbm(float64(x)*0.11, float64(y)*0.11, seed+900)
				if f > 0.52 && hash2(x, y, seed+3) < 0.75 {
					w.Deco[i] = uint8(1 + int(hash2(x, y, seed+4)*3))
				} else if hash2(x, y, seed+5) < 0.05 {
					w.Deco[i] = 5
				}
			} else if w.Terr[i] == TRock && w.Dep[i] == 0 {
				if hash2(x, y, seed+6) < 0.45 {
					w.Deco[i] = 4
				}
			}
		}
	}
	w.ComputeDist()
	w.ComputeHeights()
	return w, isl
}

func nearType(w *World, x, y int, t uint8, r int) bool {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if w.In(x+dx, y+dy) && w.Terr[w.Idx(x+dx, y+dy)] == t {
				return true
			}
		}
	}
	return false
}

func (w *World) ComputeDist() {
	n := w.N
	q := make([]int, 0, n*n)
	for i := range w.Dist {
		if w.Terr[i] != TWater {
			w.Dist[i] = 0
			q = append(q, i)
		} else {
			w.Dist[i] = 255
		}
	}
	for h := 0; h < len(q); h++ {
		i := q[h]
		x, y := i%n, i/n
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !w.In(nx, ny) {
				continue
			}
			j := w.Idx(nx, ny)
			if w.Dist[j] > w.Dist[i]+1 {
				w.Dist[j] = w.Dist[i] + 1
				q = append(q, j)
			}
		}
	}
	// Abstand Land -> Wasser
	w.LDist = make([]uint8, n*n)
	q = q[:0]
	for i := range w.LDist {
		if w.Terr[i] == TWater {
			w.LDist[i] = 0
			q = append(q, i)
		} else {
			w.LDist[i] = 255
		}
	}
	for h := 0; h < len(q); h++ {
		i := q[h]
		x, y := i%n, i/n
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !w.In(nx, ny) {
				continue
			}
			j := w.Idx(nx, ny)
			if w.LDist[j] > w.LDist[i]+1 {
				w.LDist[j] = w.LDist[i] + 1
				q = append(q, j)
			}
		}
	}
	// Hügel: Höhenfeld an den Ecken, daraus Lichtwerte
	hgt := func(x, y int) float64 {
		x, y = clampi(x, 0, n-1), clampi(y, 0, n-1)
		i := w.Idx(x, y)
		if w.Terr[i] == TWater {
			return 0
		}
		f := math.Min(1, float64(w.LDist[i])/5)
		h := (0.3 + fbm(float64(x)*0.07, float64(y)*0.07, w.Seed+1234)) * 3.2 * f
		if w.Terr[i] == TRock {
			h += 1.6 + fbm(float64(x)*0.3, float64(y)*0.3, w.Seed+77)*1.5
		}
		return h
	}
	hc := make([]float64, (n+1)*(n+1))
	for y := 0; y <= n; y++ {
		for x := 0; x <= n; x++ {
			hc[y*(n+1)+x] = (hgt(x-1, y-1) + hgt(x, y-1) + hgt(x-1, y) + hgt(x, y)) / 4
		}
	}
	w.Light = make([]float32, (n+1)*(n+1))
	for y := 0; y <= n; y++ {
		for x := 0; x <= n; x++ {
			at := func(a, b int) float64 { return hc[clampi(b, 0, n)*(n+1)+clampi(a, 0, n)] }
			gx := at(x-1, y) - at(x+1, y)
			gy := at(x, y-1) - at(x, y+1)
			l := 1 + 0.16*gx + 0.10*gy
			w.Light[y*(n+1)+x] = float32(clampf(l, 0.72, 1.25))
		}
	}
	for i := range w.Elev {
		switch w.Terr[i] {
		case TWater:
			d := w.Dist[i]
			if d <= 1 {
				w.Elev[i] = 2
			} else if d <= 3 {
				w.Elev[i] = 1
			} else {
				w.Elev[i] = 0
			}
		case TSand:
			w.Elev[i] = 3
		case TGrass:
			w.Elev[i] = 4
		case TRock:
			w.Elev[i] = 5
		}
	}
}

// Eckwert (für weiche Übergänge): Mittel der 4 angrenzenden Kacheln, Viertelschritte
func (w *World) LightAt(x, y int) float32 {
	if x < 0 || y < 0 || x > w.N || y > w.N {
		return 1
	}
	return w.Light[y*(w.N+1)+x]
}

func (w *World) Corner(x, y int) int {
	s := 0
	for _, d := range [][2]int{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}} {
		xx, yy := clampi(x+d[0], 0, w.N-1), clampi(y+d[1], 0, w.N-1)
		s += int(w.Elev[w.Idx(xx, yy)])
	}
	return s // 0..20
}

// ------------------------------------------------------------ 3D-Höhen

func (w *World) tileHeight(x, y int) float64 {
	n := w.N
	if x < 0 || y < 0 || x >= n || y >= n {
		return -2.2
	}
	i := w.Idx(x, y)
	switch w.Terr[i] {
	case TWater:
		d := float64(w.Dist[i])
		return -0.25 - 0.32*math.Min(d, 7)
	case TSand:
		return 0.14
	case TRock:
		f := math.Min(1, float64(w.LDist[i])/4)
		return 0.45 + (0.35+fbm(float64(x)*0.18, float64(y)*0.18, w.Seed+77)*1.3)*f
	default:
		f := math.Min(1, float64(w.LDist[i])/7)
		return 0.24 + (fbm(float64(x)*0.06, float64(y)*0.06, w.Seed+1234)*1.1)*f
	}
}

func (w *World) ComputeHeights() {
	n := w.N
	w.Hgt = make([]float32, (n+1)*(n+1))
	for y := 0; y <= n; y++ {
		for x := 0; x <= n; x++ {
			h := (w.tileHeight(x-1, y-1) + w.tileHeight(x, y-1) + w.tileHeight(x-1, y) + w.tileHeight(x, y)) / 4
			w.Hgt[y*(n+1)+x] = float32(h)
		}
	}
	// leichte Glättung an Land
	for pass := 0; pass < 2; pass++ {
		nh := make([]float32, len(w.Hgt))
		copy(nh, w.Hgt)
		for y := 1; y < n; y++ {
			for x := 1; x < n; x++ {
				i := y*(n+1) + x
				if w.Hgt[i] > 0.2 {
					nh[i] = (w.Hgt[i]*4 + w.Hgt[i-1] + w.Hgt[i+1] + w.Hgt[i-n-1] + w.Hgt[i+n+1]) / 8
				}
			}
		}
		w.Hgt = nh
	}
	w.HDirty = true
}

func (w *World) HC(x, y int) float64 {
	x, y = clampi(x, 0, w.N), clampi(y, 0, w.N)
	return float64(w.Hgt[y*(w.N+1)+x])
}

// Bilineare Geländehöhe an Weltposition
func (w *World) HeightAt(x, z float64) float64 {
	if x < 0 || z < 0 || x >= float64(w.N) || z >= float64(w.N) {
		return -2.2
	}
	xi, zi := int(x), int(z)
	fx, fz := x-float64(xi), z-float64(zi)
	a, b := w.HC(xi, zi), w.HC(xi+1, zi)
	c, d := w.HC(xi, zi+1), w.HC(xi+1, zi+1)
	return a*(1-fx)*(1-fz) + b*fx*(1-fz) + c*(1-fx)*fz + d*fx*fz
}

// Ebnet die Fläche unter einem Gebäude ein und liefert die Bauhöhe
func (w *World) Flatten(x, y, bw, bh int) float64 {
	s, c := 0.0, 0
	for yy := y; yy <= y+bh; yy++ {
		for xx := x; xx <= x+bw; xx++ {
			s += w.HC(xx, yy)
			c++
		}
	}
	h := math.Max(0.14, s/float64(c))
	for yy := y; yy <= y+bh; yy++ {
		for xx := x; xx <= x+bw; xx++ {
			if xx >= 0 && yy >= 0 && xx <= w.N && yy <= w.N {
				w.Hgt[yy*(w.N+1)+xx] = float32(h)
			}
		}
	}
	w.HDirty = true
	return h
}

// Prüfsumme über Straßen/Bebauung/Deko – ändert sich, wenn 3D-Geometrie neu gebaut werden muss
func (w *World) Sig() int {
	s := 0
	for i := range w.Road {
		if w.Road[i] {
			s += i*7 + 1
		}
		if w.Occ[i] != 0 {
			s += i * 3
		}
		s += int(w.Deco[i]) * (i & 15)
	}
	return s
}
