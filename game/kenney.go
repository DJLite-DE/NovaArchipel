package game

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/binary"
	"io"
	"math"
)

// ---------------------------------------------------------------- Kenney-Modelle (CC0, www.kenney.nl)
//
// Die Modelle stammen aus den kostenlosen Kenney-Paketen City Kit (Commercial,
// Industrial, Suburban), Space Kit und Watercraft Pack (Lizenz: CC0). Sie werden
// mit tools/glb2nova.py in ein kompaktes Format übersetzt und hier eingebettet.

//go:embed kenney.bin
var kenneyBin []byte

type KModel struct {
	P        []float32 // xyz
	N        []float32 // xyz
	C        []RGB
	M        []uint8
	I        []uint32
	Min, Max V3
}

var kenney map[string]*KModel

func loadKenney() {
	if kenney != nil {
		return
	}
	kenney = map[string]*KModel{}
	zr, err := gzip.NewReader(bytes.NewReader(kenneyBin))
	if err != nil {
		return
	}
	data, err := io.ReadAll(zr)
	if err != nil {
		return
	}
	r := bytes.NewReader(data)
	var n uint32
	binary.Read(r, binary.LittleEndian, &n)
	for k := uint32(0); k < n; k++ {
		var nl uint16
		binary.Read(r, binary.LittleEndian, &nl)
		name := make([]byte, nl)
		io.ReadFull(r, name)
		var nv, ni uint32
		binary.Read(r, binary.LittleEndian, &nv)
		binary.Read(r, binary.LittleEndian, &ni)
		m := &KModel{Min: V3{1e9, 1e9, 1e9}, Max: V3{-1e9, -1e9, -1e9}}
		for i := uint32(0); i < nv; i++ {
			var v struct {
				X, Y, Z    float32
				NX, NY, NZ int8
				R, G, B    uint8
				Mat        uint8
			}
			binary.Read(r, binary.LittleEndian, &v)
			m.P = append(m.P, v.X, v.Y, v.Z)
			nn := V3{float64(v.NX), float64(v.NY), float64(v.NZ)}
			if nn.Len() < 1e-3 {
				nn = V3{0, 1, 0}
			}
			nn = nn.Norm()
			m.N = append(m.N, float32(nn.X), float32(nn.Y), float32(nn.Z))
			m.C = append(m.C, RGB{v.R, v.G, v.B})
			m.M = append(m.M, v.Mat)
			p := V3{float64(v.X), float64(v.Y), float64(v.Z)}
			m.Min = V3{math.Min(m.Min.X, p.X), math.Min(m.Min.Y, p.Y), math.Min(m.Min.Z, p.Z)}
			m.Max = V3{math.Max(m.Max.X, p.X), math.Max(m.Max.Y, p.Y), math.Max(m.Max.Z, p.Z)}
		}
		m.I = make([]uint32, ni)
		binary.Read(r, binary.LittleEndian, m.I)
		kenney[string(name)] = m
	}
}

func KM(key string) *KModel {
	loadKenney()
	return kenney[key]
}

// Farbumfärbung: orange/gelbe Akzente der Vorlage in eine Fraktionsfarbe tauschen
type Recolor func(c RGB, mat int) (RGB, int)

func accentTo(col RGB, glow bool) Recolor {
	return func(c RGB, mat int) (RGB, int) {
		r, g, b := int(c.R), int(c.G), int(c.B)
		if r > 170 && g > 80 && g < 200 && b < 110 && r-b > 110 {
			if glow {
				return col, MatEmit
			}
			return col, mat
		}
		return c, mat
	}
}

// Modell einsetzen: in die Grundfläche (x0,z0)-(x0+w,z0+d) eingepasst, Boden auf y0.
// rot = Vierteldrehungen, maxH > 0 begrenzt die Höhe. Gibt die erreichte Höhe zurück.
func (b *MB) Kenney(key string, x0, z0, w, d, y0 float64, rot int, maxH float64, rc Recolor) float64 {
	m := KM(key)
	if m == nil {
		return 0
	}
	sx, sz := m.Max.X-m.Min.X, m.Max.Z-m.Min.Z
	sy := m.Max.Y - m.Min.Y
	if rot%2 == 1 {
		sx, sz = sz, sx
	}
	s := math.Min(w/math.Max(sx, 0.01), d/math.Max(sz, 0.01))
	if maxH > 0 && sy*s > maxH {
		s = maxH / sy
	}
	cx, cz := (m.Min.X+m.Max.X)/2, (m.Min.Z+m.Max.Z)/2
	tf := Translate(x0+w/2, y0, z0+d/2).Mul(RotY(float64(rot) * math.Pi / 2)).Mul(ScaleM(s, s, s)).Mul(Translate(-cx, -m.Min.Y, -cz))
	b.Push(tf)
	base := uint32(len(b.V) / 10)
	for i := range m.C {
		p := V3{float64(m.P[i*3]), float64(m.P[i*3+1]), float64(m.P[i*3+2])}
		n := V3{float64(m.N[i*3]), float64(m.N[i*3+1]), float64(m.N[i*3+2])}
		c, mt := m.C[i], int(m.M[i])
		if rc != nil {
			c, mt = rc(c, mt)
		}
		b.vert(p, n, c, mt)
	}
	for _, i := range m.I {
		b.I = append(b.I, base+i)
	}
	b.Pop()
	return sy * s
}

// Ganzes Kenney-Modell als eigenes Mesh mit Ursprung in der Mitte (für Schiffe, Fahrzeuge)
func KenneyCentered(key string, length float64, rot int, rc Recolor) *MB {
	b := NewMB()
	m := KM(key)
	if m == nil {
		return b
	}
	sx, sz := m.Max.X-m.Min.X, m.Max.Z-m.Min.Z
	if rot%2 == 1 {
		sx, sz = sz, sx
	}
	s := length / math.Max(sx, sz)
	b.Kenney(key, -sx*s/2, -sz*s/2, sx*s, sz*s, 0, rot, 0, rc)
	return b
}
