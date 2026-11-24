package game

import (
	"math"
	"sync"
)

const SR = 44100

const (
	SfxClick = iota
	SfxBuild
	SfxRoad
	SfxDemolish
	SfxNotify
	SfxQuest
	SfxUpgrade
	SfxError
	SfxThunder
	SfxShip
	SfxLaser
	SfxBoom
	NumSfx
)

type voice struct {
	buf []float32
	pos int
	vol float32
	pan float32
}

type Mixer struct {
	mu     sync.Mutex
	voices []voice
	bank   [NumSfx][]float32
	n      int64
	// Musik
	padPh  [6]float64
	padLP  [2]float64
	arpEnv float64
	arpPh  float64
	arpF   float64
	bassPh float64
	bassF  float64
	bassE  float64
	dl     []float32
	dr     []float32
	dpos   int
	chord  int
	chordX float64
	// Ambiente
	brown  float64
	lp1    float64
	rainLP float64
	rng    uint32
	Storm  float64
	Night  float64
	Muted  bool
	Mood   int
}

var mixer *Mixer

func NewMixer() *Mixer {
	m := &Mixer{rng: 12345}
	d := SR * 3 / 4 * 60 / 84 // punktierte Achtel bei 84 BPM
	m.dl = make([]float32, d)
	m.dr = make([]float32, d)
	m.genBank()
	mixer = m
	return m
}

func PlaySfx(s int) {
	if mixer == nil || s < 0 || s >= NumSfx {
		return
	}
	mixer.mu.Lock()
	if len(mixer.voices) < 24 {
		mixer.voices = append(mixer.voices, voice{buf: mixer.bank[s], vol: 1, pan: 0})
	}
	mixer.mu.Unlock()
}

// Musikstimmung: 0 Tag, 1 Nacht, 2 Krieg, 3 Mond
func SetMusicMood(m int) {
	if mixer == nil {
		return
	}
	mixer.mu.Lock()
	mixer.Mood = m
	mixer.mu.Unlock()
}

type musicMood struct {
	chords    [][]float64
	arp       float64
	bassEvery int
	beatMul   float64
}

var moods = []musicMood{
	{chordNotes, 0.55, 4, 1},
	{[][]float64{{57, 60, 64, 67}, {53, 57, 60, 64}, {50, 53, 57, 60}, {52, 55, 59, 62}}, 0.3, 8, 1.35},
	{[][]float64{{52, 55, 59, 64}, {48, 52, 55, 59}, {50, 53, 57, 62}, {47, 50, 54, 59}}, 0.8, 2, 0.8},
	{[][]float64{{50, 54, 57, 64}, {55, 59, 62, 69}, {52, 56, 59, 66}, {57, 61, 64, 71}}, 0.35, 8, 1.6},
}

func SetAudioEnv(storm, night float64) {
	if mixer == nil {
		return
	}
	mixer.mu.Lock()
	mixer.Storm, mixer.Night = storm, night
	mixer.mu.Unlock()
}

func (m *Mixer) rnd() float64 {
	m.rng ^= m.rng << 13
	m.rng ^= m.rng >> 17
	m.rng ^= m.rng << 5
	return float64(m.rng)/float64(1<<32)*2 - 1
}

// ---------------------------------------------------------------- Effekte erzeugen

func env(t, a, d float64) float64 {
	if t < a {
		return t / a
	}
	return math.Exp(-(t - a) / d)
}

func (m *Mixer) genBank() {
	mk := func(sec float64, f func(t float64) float64) []float32 {
		n := int(sec * SR)
		b := make([]float32, n)
		for i := range b {
			b[i] = float32(f(float64(i) / SR))
		}
		return b
	}
	bell := func(t, f, d float64) float64 {
		return (math.Sin(2*math.Pi*f*t) + 0.4*math.Sin(2*math.Pi*f*2.01*t)*math.Exp(-t/d*2) + 0.15*math.Sin(2*math.Pi*f*3.02*t)) * env(t, 0.004, d)
	}
	var nz float64
	noise := func() float64 { return m.rnd() }
	m.bank[SfxClick] = mk(0.06, func(t float64) float64 {
		return 0.25 * math.Sin(2*math.Pi*1700*t) * env(t, 0.001, 0.012)
	})
	m.bank[SfxBuild] = mk(0.9, func(t float64) float64 {
		f := 95 - 45*math.Min(1, t*6)
		thump := math.Sin(2*math.Pi*f*t) * env(t, 0.003, 0.12) * 0.7
		ch := 0.0
		if t > 0.06 {
			ch = (bell(t-0.06, 660, 0.35) + bell(t-0.06, 990, 0.3)*0.6) * 0.18
		}
		return thump + ch + noise()*env(t, 0.001, 0.02)*0.15
	})
	m.bank[SfxRoad] = mk(0.2, func(t float64) float64 {
		return math.Sin(2*math.Pi*(140-60*t*5)*t)*env(t, 0.002, 0.05)*0.35 + noise()*env(t, 0.001, 0.01)*0.08
	})
	m.bank[SfxDemolish] = mk(1.0, func(t float64) float64 {
		nz += (noise() - nz) * 0.08
		return nz*env(t, 0.005, 0.25)*1.6 + math.Sin(2*math.Pi*55*t)*env(t, 0.01, 0.3)*0.4
	})
	m.bank[SfxNotify] = mk(0.9, func(t float64) float64 {
		v := bell(t, 880, 0.25) * 0.15
		if t > 0.12 {
			v += bell(t-0.12, 1318.5, 0.3) * 0.15
		}
		return v
	})
	m.bank[SfxQuest] = mk(1.8, func(t float64) float64 {
		v := 0.0
		for i, f := range []float64{523.25, 659.25, 783.99, 1046.5} {
			st := float64(i) * 0.11
			if t > st {
				v += bell(t-st, f, 0.6) * 0.13
			}
		}
		return v
	})
	m.bank[SfxUpgrade] = mk(1.2, func(t float64) float64 {
		f := 600 + 900*t
		return math.Sin(2*math.Pi*f*t)*env(t, 0.02, 0.35)*0.12 + bell(t, 1567.98, 0.4)*0.06
	})
	m.bank[SfxError] = mk(0.22, func(t float64) float64 {
		s := math.Sin(2*math.Pi*160*t) + 0.3*math.Sin(2*math.Pi*320*t)
		return s * env(t, 0.005, 0.07) * 0.22
	})
	m.bank[SfxThunder] = mk(4.0, func(t float64) float64 {
		nz += (noise() - nz) * 0.02
		crack := noise() * env(t, 0.001, 0.05) * 0.5
		rum := nz * (env(t, 0.08, 1.1) + 0.4*env(t, 0.6, 0.8)*math.Max(0, math.Sin(t*7))) * 3.2
		return crack + rum
	})
	m.bank[SfxShip] = mk(1.4, func(t float64) float64 {
		e := math.Min(1, t*8) * math.Exp(-math.Max(0, t-0.7)*5)
		s := 0.0
		for _, f := range []float64{110, 165} {
			for h := 1; h <= 5; h++ {
				s += math.Sin(2*math.Pi*f*float64(h)*t) / float64(h)
			}
		}
		return s * e * 0.09
	})
	m.bank[SfxLaser] = mk(0.35, func(t float64) float64 {
		f := 1800*math.Exp(-t*9) + 220
		return math.Sin(2*math.Pi*f*t) * env(t, 0.002, 0.25) * 0.22
	})
	var bz float64
	m.bank[SfxBoom] = mk(1.6, func(t float64) float64 {
		bz += (noise() - bz) * 0.05
		thump := math.Sin(2*math.Pi*(55+90*math.Exp(-t*12))*t) * env(t, 0.002, 0.45)
		return bz*env(t, 0.003, 0.9)*2.6 + thump*0.7
	})
}

// ---------------------------------------------------------------- Musik

var chordNotes = [][]float64{
	{57, 60, 64, 69}, // Am
	{53, 57, 60, 65}, // F
	{48, 52, 55, 60}, // C
	{55, 59, 62, 67}, // G
	{57, 60, 64, 71}, // Am(add9)
	{50, 53, 57, 62}, // Dm
	{53, 57, 60, 64}, // Fmaj7
	{52, 55, 59, 64}, // Em
}

func midiF(n float64) float64 { return 440 * math.Pow(2, (n-69)/12) }

func hashI(a, b int) float64 { return hash2(a, b, 991) }

// Füllt einen Stereo-Puffer (interleaved int16)
func (m *Mixer) Fill(out []int16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mv := float32(Opt.MasterVol)
	mus := Opt.MusicVol
	sfx := float32(Opt.SfxVol)
	amb := Opt.AmbVol
	md := moods[clampi(m.Mood, 0, len(moods)-1)]
	beat := float64(SR) * 60 / 84 / 2 * md.beatMul // Achtel
	for i := 0; i < len(out)/2; i++ {
		var l, r float64
		// --- Musik
		if mus > 0.001 {
			step := int(float64(m.n) / beat)
			bar := step / 16
			ci := (bar / 2) % len(md.chords)
			if ci != m.chord {
				m.chord = ci
				m.chordX = 0
			}
			m.chordX = math.Min(1, m.chordX+1.0/(SR*1.5))
			ch := md.chords[ci%len(md.chords)]
			pad := 0.0
			for k := 0; k < 3; k++ {
				f := midiF(ch[k+1] - 12)
				for d := 0; d < 2; d++ {
					idx := k*2 + d
					det := 1.0 + float64(d*2-1)*0.0035
					m.padPh[idx] += f * det / SR
					if m.padPh[idx] > 1 {
						m.padPh[idx]--
					}
					pad += m.padPh[idx]*2 - 1
				}
			}
			cut := 0.02 + 0.012*math.Sin(float64(m.n)/SR*0.21) + 0.01*(1-m.Night)
			m.padLP[0] += (pad - m.padLP[0]) * cut
			m.padLP[1] += (m.padLP[0] - m.padLP[1]) * cut
			pv := m.padLP[1] * 0.05 * (0.4 + 0.6*m.chordX)
			// Arpeggio
			if float64(m.n) >= float64(step)*beat && float64(m.n) < float64(step)*beat+1 {
				hv := hashI(step, bar)
				if hv < md.arp {
					note := ch[int(hashI(step, 7)*4)] + 12
					if hashI(step, 9) < 0.3 {
						note += 12
					}
					m.arpF = midiF(note)
					m.arpEnv = 0.5 + hv*0.5
				}
				if step%md.bassEvery == 0 {
					m.bassF = midiF(ch[0] - 12)
					m.bassE = 1
				}
			}
			m.arpPh += m.arpF / SR
			m.arpEnv *= 0.99985
			arp := (math.Sin(2*math.Pi*m.arpPh) + 0.25*math.Sin(4*math.Pi*m.arpPh)) * m.arpEnv * m.arpEnv * 0.07
			m.bassPh += m.bassF / SR
			m.bassE *= 0.99993
			bass := math.Sin(2*math.Pi*m.bassPh) * m.bassE * 0.11
			// Ping-Pong-Echo
			dl := float64(m.dl[m.dpos])
			dr := float64(m.dr[m.dpos])
			m.dl[m.dpos] = float32(arp + dr*0.42)
			m.dr[m.dpos] = float32(dl * 0.42)
			m.dpos = (m.dpos + 1) % len(m.dl)
			ml := pv + arp*0.8 + bass + dl*0.6
			mr := pv + arp*0.6 + bass + dr*0.6
			l += ml * mus
			r += mr * mus
		}
		// --- Ambiente: Meer + Regen
		if amb > 0.001 {
			w := m.rnd()
			m.brown = (m.brown + w*0.02) * 0.998
			t := float64(m.n) / SR
			swell := 0.55 + 0.45*math.Sin(t*0.45)*math.Sin(t*0.17+1)
			m.lp1 += (m.brown - m.lp1) * 0.06
			sea := m.lp1 * swell * (1.6 + m.Storm*2)
			rain := 0.0
			if m.Storm > 0.05 {
				m.rainLP += (w - m.rainLP) * 0.5
				rain = (w - m.rainLP) * 0.12 * m.Storm
			}
			l += (sea + rain) * amb
			r += (sea*0.9 + rain*1.1) * amb
		}
		// --- Effekte
		for vi := range m.voices {
			v := &m.voices[vi]
			if v.pos < len(v.buf) {
				s := v.buf[v.pos] * v.vol * sfx
				l += float64(s)
				r += float64(s)
				v.pos++
			}
		}
		l *= float64(mv)
		r *= float64(mv)
		if m.Muted {
			l, r = 0, 0
		}
		out[i*2] = int16(clampf(math.Tanh(l)*30000, -32767, 32767))
		out[i*2+1] = int16(clampf(math.Tanh(r)*30000, -32767, 32767))
		m.n++
	}
	k := m.voices[:0]
	for _, v := range m.voices {
		if v.pos < len(v.buf) {
			k = append(k, v)
		}
	}
	m.voices = k
}
