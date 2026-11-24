package game

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"
)

type Renderer struct {
	W, H                         int
	pMesh, pDepth, pWater, pSky  *Prog
	pDown, pBlur, pComp, pFlat   *Prog
	pSSAO, pAOApply, pGlow       *Prog
	pPart, pCloud                *Prog
	reflFBO, reflTex, reflDepth  uint32
	fx                           *FX
	life                         *Life
	ground                       []*Instanced
	game                         *Game
	depthFBO, depthTex           uint32
	aoFBO, aoTex, ao2FBO, ao2Tex uint32
	OutW, OutH                   int
	curMSAA, curShadow           int
	curScale                     float64
	glows                        *GMesh
	emptyVAO                     uint32
	shadowFBO, shadowTex         uint32
	shadowSize                   int
	msFBO, msColor, msDepth      uint32
	resFBO, resTex               uint32
	halfFBO, halfTex             uint32
	qFBO, qTex, q2FBO, q2Tex     uint32
	uiTex                        uint32
	uiW, uiH                     int
	heightTex                    uint32
	OutFBO                       uint32
	Samples                      int
	models                       map[string]*GMesh
	terrain                      *GMesh
	roads                        *GMesh
	water                        *GMesh
	trees                        []*Instanced
	rotor, cart                  *GMesh
	ships                        [5]*GMesh
	flags                        [4]*GMesh
	modelH                       map[string]float64
	scaffolds                    map[int]*GMesh
	crane, craneJib              *GMesh
	warships                     [6]*GMesh
	turretHead, rocket           *GMesh
	moonF                        float64
	kenneyOn                     bool
	flatVAO, flatVBO             uint32
	world                        *World
	roadSig                      int
	decoSig                      int
	time                         float64
	Night                        float64
	thumbs                       map[int]*Sprite
	Info                         string
}

func NewRenderer() (*Renderer, error) {
	r := &Renderer{models: map[string]*GMesh{}, thumbs: map[int]*Sprite{}, shadowSize: 2048, Samples: 4}
	r.Info = glString(GL_RENDERER) + " / " + glString(GL_VERSION)
	var err error
	mk := func(vs, fs string) *Prog {
		if err != nil {
			return nil
		}
		var p *Prog
		p, err = NewProg(vs, fs)
		return p
	}
	r.pMesh = mk(vsMesh, fsMesh)
	r.pDepth = mk(vsMesh, fsDepth)
	r.pWater = mk(vsWater, fsWater)
	r.pSky = mk(vsQuad, fsSky)
	r.pDown = mk(vsQuad, fsDown)
	r.pBlur = mk(vsQuad, fsBlur)
	r.pComp = mk(vsQuad, fsComposite)
	r.pFlat = mk(vsFlat, fsFlat)
	r.pSSAO = mk(vsQuad, fsSSAO)
	r.pAOApply = mk(vsQuad, fsAOApply)
	r.pGlow = mk(vsGlow, fsGlow)
	r.pPart = mk(vsPart, fsPart)
	r.pCloud = mk(vsCloud, fsCloud)
	if err != nil {
		return nil, err
	}
	var ms int32
	gl("glGetIntegerv", GL_MAX_SAMPLES, ptr(unsafe.Pointer(&ms)))
	if int(ms) < r.Samples {
		r.Samples = int(ms)
	}
	if r.Samples < 1 {
		r.Samples = 1
	}
	r.emptyVAO = glGenVertexArray()
	r.flatVAO = glGenVertexArray()
	gl("glBindVertexArray", uintptr(r.flatVAO))
	r.flatVBO = glGenBuffer()
	gl("glBindBuffer", GL_ARRAY_BUFFER, uintptr(r.flatVBO))
	gl("glEnableVertexAttribArray", 0)
	gl("glVertexAttribPointer", 0, 3, GL_FLOAT, 0, 12, 0)
	gl("glBindVertexArray", 0)
	if err := r.makeShadow(Opt.ShadowRes); err != nil {
		return nil, err
	}
	// Modelle
	r.rotor = RotorModel().Upload()
	r.cart = CartModel().Upload()
	r.ships[0] = ShipModel(0).Upload()
	r.ships[1] = ShipModel(1).Upload()
	for k := range RivalDefs {
		r.ships[2+k] = ShipModel(2 + k).Upload()
		r.flags[1+k] = FlagModel(RivalDefs[k].Col).Upload()
	}
	r.flags[0] = FlagModel(mCyan).Upload()
	r.modelH = map[string]float64{}
	r.scaffolds = map[int]*GMesh{}
	r.crane = CraneModel().Upload()
	r.turretHead = TurretHeadModel().Upload()
	r.rocket = RocketModel().Upload()
	for f := 0; f < 6; f++ {
		switch {
		case f == 0:
			r.warships[f] = WarshipModel(mCyan, hex(0xdfe5ec)).Upload()
		case f >= 2 && f <= 4:
			c := RivalDefs[f-2].Col
			r.warships[f] = WarshipModel(c, hex(0x3a3f4a).Mix(c, 0.12)).Upload()
		case f == KindPirate:
			r.warships[f] = WarshipModel(hex(0xff3048), hex(0x1c1e24)).Upload()
		}
	}
	r.craneJib = CraneJibModel().Upload()
	for k := 0; k < 7; k++ {
		r.trees = append(r.trees, NewInstanced(TreeModel(k).Upload()))
	}
	r.buildWater()
	r.fx = newFX()
	r.life = newLife()
	for _, m := range []*MB{GrassModel(), FlowerModel(), PebbleModel()} {
		r.ground = append(r.ground, NewInstanced(m.Upload()))
	}
	r.kenneyOn = Opt.KenneyModels
	r.makeThumbs()
	ThumbProvider = func(t, tier int) *Sprite { return r.thumbs[t*4+tier] }
	return r, nil
}

func (r *Renderer) makeShadow(size int) error {
	if size < 512 {
		size = 2048
	}
	delFBO(&r.shadowFBO)
	delTex(&r.shadowTex)
	r.shadowSize = size
	r.curShadow = size
	r.shadowTex = glGenTexture()
	gl("glBindTexture", GL_TEXTURE_2D, uintptr(r.shadowTex))
	gl("glTexImage2D", GL_TEXTURE_2D, 0, GL_DEPTH_COMPONENT24, uintptr(size), uintptr(size), 0, GL_DEPTH_COMPONENT, GL_FLOAT, 0)
	texParams(GL_LINEAR, GL_CLAMP_TO_EDGE)
	gl("glTexParameteri", GL_TEXTURE_2D, GL_TEXTURE_COMPARE_MODE, GL_COMPARE_REF_TO_TEXTURE)
	gl("glTexParameteri", GL_TEXTURE_2D, GL_TEXTURE_COMPARE_FUNC, GL_LEQUAL)
	r.shadowFBO = glGenFramebuffer()
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.shadowFBO))
	gl("glFramebufferTexture2D", GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_TEXTURE_2D, uintptr(r.shadowTex), 0)
	gl("glDrawBuffer", 0)
	gl("glReadBuffer", 0)
	if st := uint32(gl("glCheckFramebufferStatus", GL_FRAMEBUFFER)); st != GL_FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("Schatten-Framebuffer unvollständig (%x)", st)
	}
	gl("glBindFramebuffer", GL_FRAMEBUFFER, 0)
	return nil
}

func texParams(filter, wrap uintptr) {
	gl("glTexParameteri", GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, filter)
	gl("glTexParameteri", GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, filter)
	gl("glTexParameteri", GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, wrap)
	gl("glTexParameteri", GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, wrap)
}

func makeTexFBO(w, h int, internal, format, typ uintptr) (uint32, uint32) {
	t := glGenTexture()
	gl("glBindTexture", GL_TEXTURE_2D, uintptr(t))
	gl("glTexImage2D", GL_TEXTURE_2D, 0, internal, uintptr(w), uintptr(h), 0, format, typ, 0)
	texParams(GL_LINEAR, GL_CLAMP_TO_EDGE)
	f := glGenFramebuffer()
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(f))
	gl("glFramebufferTexture2D", GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, uintptr(t), 0)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, 0)
	return f, t
}

func delTex(t *uint32) {
	if *t != 0 {
		gl("glDeleteTextures", 1, ptr(unsafe.Pointer(t)))
		*t = 0
	}
}
func delFBO(f *uint32) {
	if *f != 0 {
		gl("glDeleteFramebuffers", 1, ptr(unsafe.Pointer(f)))
		*f = 0
	}
}

// Ausgabegröße setzen; interne Auflösung folgt der Render-Skalierung
func (r *Renderer) Resize(ow, oh int) error {
	if Opt.ShadowRes != r.curShadow {
		if err := r.makeShadow(Opt.ShadowRes); err != nil {
			return err
		}
	}
	want := Opt.MSAA
	if want > r.maxSamples() {
		want = r.maxSamples()
	}
	if want < 1 {
		want = 1
	}
	w := imax(64, int(float64(ow)*Opt.RenderScale))
	h := imax(64, int(float64(oh)*Opt.RenderScale))
	if w == r.W && h == r.H && ow == r.OutW && oh == r.OutH && want == r.curMSAA {
		return nil
	}
	r.OutW, r.OutH = ow, oh
	r.Samples = want
	r.curMSAA = want
	r.W, r.H = w, h
	delFBO(&r.depthFBO)
	delTex(&r.depthTex)
	delFBO(&r.aoFBO)
	delTex(&r.aoTex)
	delFBO(&r.ao2FBO)
	delTex(&r.ao2Tex)
	delFBO(&r.msFBO)
	if r.msColor != 0 {
		gl("glDeleteRenderbuffers", 1, ptr(unsafe.Pointer(&r.msColor)))
		gl("glDeleteRenderbuffers", 1, ptr(unsafe.Pointer(&r.msDepth)))
	}
	delFBO(&r.resFBO)
	delTex(&r.resTex)
	delFBO(&r.halfFBO)
	delTex(&r.halfTex)
	delFBO(&r.qFBO)
	delTex(&r.qTex)
	delFBO(&r.q2FBO)
	delTex(&r.q2Tex)
	r.msColor = glGenRenderbuffer()
	gl("glBindRenderbuffer", GL_RENDERBUFFER, uintptr(r.msColor))
	gl("glRenderbufferStorageMultisample", GL_RENDERBUFFER, uintptr(r.Samples), GL_RGBA16F, uintptr(w), uintptr(h))
	r.msDepth = glGenRenderbuffer()
	gl("glBindRenderbuffer", GL_RENDERBUFFER, uintptr(r.msDepth))
	gl("glRenderbufferStorageMultisample", GL_RENDERBUFFER, uintptr(r.Samples), GL_DEPTH_COMPONENT24, uintptr(w), uintptr(h))
	r.msFBO = glGenFramebuffer()
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.msFBO))
	gl("glFramebufferRenderbuffer", GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_RENDERBUFFER, uintptr(r.msColor))
	gl("glFramebufferRenderbuffer", GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_RENDERBUFFER, uintptr(r.msDepth))
	if st := uint32(gl("glCheckFramebufferStatus", GL_FRAMEBUFFER)); st != GL_FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("Szenen-Framebuffer unvollständig (%x)", st)
	}
	r.resFBO, r.resTex = makeTexFBO(w, h, GL_RGBA16F, GL_RGBA, GL_FLOAT)
	r.halfFBO, r.halfTex = makeTexFBO(imax(1, w/2), imax(1, h/2), GL_RGBA16F, GL_RGBA, GL_FLOAT)
	r.qFBO, r.qTex = makeTexFBO(imax(1, w/4), imax(1, h/4), GL_RGBA16F, GL_RGBA, GL_FLOAT)
	r.q2FBO, r.q2Tex = makeTexFBO(imax(1, w/4), imax(1, h/4), GL_RGBA16F, GL_RGBA, GL_FLOAT)
	delFBO(&r.reflFBO)
	delTex(&r.reflTex)
	if r.reflDepth != 0 {
		gl("glDeleteRenderbuffers", 1, ptr(unsafe.Pointer(&r.reflDepth)))
	}
	r.reflFBO, r.reflTex = makeTexFBO(imax(1, w/2), imax(1, h/2), GL_RGBA16F, GL_RGBA, GL_FLOAT)
	r.reflDepth = glGenRenderbuffer()
	gl("glBindRenderbuffer", GL_RENDERBUFFER, uintptr(r.reflDepth))
	gl("glRenderbufferStorage", GL_RENDERBUFFER, GL_DEPTH_COMPONENT24, uintptr(imax(1, w/2)), uintptr(imax(1, h/2)))
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.reflFBO))
	gl("glFramebufferRenderbuffer", GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_RENDERBUFFER, uintptr(r.reflDepth))
	gl("glBindFramebuffer", GL_FRAMEBUFFER, 0)
	r.aoFBO, r.aoTex = makeTexFBO(imax(1, w/2), imax(1, h/2), GL_RGBA8, GL_RGBA, GL_UNSIGNED_BYTE)
	r.ao2FBO, r.ao2Tex = makeTexFBO(imax(1, w/2), imax(1, h/2), GL_RGBA8, GL_RGBA, GL_UNSIGNED_BYTE)
	// Tiefen-Textur für SSAO
	r.depthTex = glGenTexture()
	gl("glBindTexture", GL_TEXTURE_2D, uintptr(r.depthTex))
	gl("glTexImage2D", GL_TEXTURE_2D, 0, GL_DEPTH_COMPONENT24, uintptr(w), uintptr(h), 0, GL_DEPTH_COMPONENT, GL_FLOAT, 0)
	texParams(GL_NEAREST, GL_CLAMP_TO_EDGE)
	r.depthFBO = glGenFramebuffer()
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.depthFBO))
	gl("glFramebufferTexture2D", GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_TEXTURE_2D, uintptr(r.depthTex), 0)
	gl("glDrawBuffer", 0)
	gl("glReadBuffer", 0)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, 0)
	return nil
}

func (r *Renderer) maxSamples() int {
	var ms int32
	gl("glGetIntegerv", GL_MAX_SAMPLES, ptr(unsafe.Pointer(&ms)))
	return int(ms)
}

// ---------------------------------------------------------------- Weltgeometrie

func terrainColor3(w *World, x, y int) RGB {
	// Mischung der 4 angrenzenden Kacheln
	var r, g, b float64
	n := 0.0
	for _, d := range [][2]int{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}} {
		xx, yy := clampi(x+d[0], 0, w.N-1), clampi(y+d[1], 0, w.N-1)
		i := w.Idx(xx, yy)
		var c RGB
		if w.Moon {
			v := 0.85 + 0.3*hash2(xx/2, yy/2, 31)
			switch w.Terr[i] {
			case TWater:
				c = hex(0x6a6866)
			case TSand:
				c = hex(0xa6a39e)
			case TRock:
				c = hex(0x5a5856)
			default:
				c = hex(0x8e8b86).Mul(v)
			}
			r += float64(c.R)
			g += float64(c.G)
			b += float64(c.B)
			n++
			continue
		}
		switch w.Terr[i] {
		case TWater:
			c = hex(0xc8b88a)
		case TSand:
			c = hex(0xe6d6a8)
		case TRock:
			c = hex(0x8a8478)
		default:
			if w.Biome[i] == 1 {
				c = hex(0x7a7a44)
			} else {
				c = hex(0x587a3a)
			}
		}
		r += float64(c.R)
		g += float64(c.G)
		b += float64(c.B)
		n++
	}
	return RGB{uint8(r / n), uint8(g / n), uint8(b / n)}
}

func (r *Renderer) buildTerrain(w *World) {
	n := w.N
	b := NewMB()
	nrm := func(x, y int) V3 {
		hx := w.HC(x+1, y) - w.HC(x-1, y)
		hz := w.HC(x, y+1) - w.HC(x, y-1)
		return V3{-hx, 2, -hz}.Norm()
	}
	for y := 0; y <= n; y++ {
		for x := 0; x <= n; x++ {
			h := w.HC(x, y)
			c := terrainColor3(w, x, y)
			if h < 0.05 {
				if w.Moon {
					c = c.Mix(hex(0x77746f), 0.6)
				} else {
					c = c.Mix(hex(0xb8a878), 0.6)
				}
			}
			nn := nrm(x, y)
			b.V = append(b.V, float32(x), float32(h), float32(y), float32(nn.X), float32(nn.Y), float32(nn.Z),
				srgb(c.R), srgb(c.G), srgb(c.B), MatTerrain)
		}
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := uint32(y*(n+1) + x)
			j := i + uint32(n+1)
			b.I = append(b.I, i, j, i+1, i+1, j, j+1)
		}
	}
	// Rand nach unten (Meeresgrund ringsum)
	if r.terrain == nil {
		r.terrain = b.Upload()
	} else {
		r.terrain.Update(b)
	}
	// Höhentextur
	if r.heightTex == 0 {
		r.heightTex = glGenTexture()
	}
	gl("glBindTexture", GL_TEXTURE_2D, uintptr(r.heightTex))
	gl("glPixelStorei", GL_UNPACK_ALIGNMENT, 1)
	gl("glTexImage2D", GL_TEXTURE_2D, 0, GL_R32F, uintptr(n+1), uintptr(n+1), 0, GL_RED, GL_FLOAT, ptr(unsafe.Pointer(&w.Hgt[0])))
	runtime.KeepAlive(w.Hgt)
	texParams(GL_LINEAR, GL_CLAMP_TO_EDGE)
	w.HDirty = false
}

func (r *Renderer) buildRoads(w *World) {
	b := NewMB()
	asph := hex(0x3c434d)
	edge := hex(0xaab2bc)
	line := mCyan
	n := w.N
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := w.Idx(x, y)
			if !w.Road[i] {
				continue
			}
			h := func(a, c float64) float64 { return w.HeightAt(a, c) + 0.025 }
			fx, fy := float64(x), float64(y)
			q := func(x0, z0, x1, z1 float64, col RGB, mat int, dy float64) {
				b.Quad(V3{x0, h(x0, z0) + dy, z0}, V3{x0, h(x0, z1) + dy, z1}, V3{x1, h(x1, z1) + dy, z1}, V3{x1, h(x1, z0) + dy, z0}, col, mat)
			}
			q(fx, fy, fx+1, fy+1, edge, MatPanel, 0)
			q(fx+0.08, fy+0.08, fx+0.92, fy+0.92, asph, MatMatte, 0.004)
			conn := [4]bool{x > 0 && w.Road[i-1], x < n-1 && w.Road[i+1], y > 0 && w.Road[i-n], y < n-1 && w.Road[i+n]}
			if conn[0] {
				q(fx, fy+0.08, fx+0.08, fy+0.92, asph, MatMatte, 0.004)
				q(fx, fy+0.47, fx+0.5, fy+0.53, line, MatEmit, 0.008)
			}
			if conn[1] {
				q(fx+0.92, fy+0.08, fx+1, fy+0.92, asph, MatMatte, 0.004)
				q(fx+0.5, fy+0.47, fx+1, fy+0.53, line, MatEmit, 0.008)
			}
			if conn[2] {
				q(fx+0.08, fy, fx+0.92, fy+0.08, asph, MatMatte, 0.004)
				q(fx+0.47, fy, fx+0.53, fy+0.5, line, MatEmit, 0.008)
			}
			if conn[3] {
				q(fx+0.08, fy+0.92, fx+0.92, fy+1, asph, MatMatte, 0.004)
				q(fx+0.47, fy+0.5, fx+0.53, fy+1, line, MatEmit, 0.008)
			}
			if !conn[0] && !conn[1] && !conn[2] && !conn[3] {
				q(fx+0.44, fy+0.44, fx+0.56, fy+0.56, line, MatEmit, 0.008)
			}
		}
	}
	// Straßenlaternen + Lichtkegel
	gb := NewMB()
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := w.Idx(x, y)
			if !w.Road[i] || (x*3+y*5)%4 != 0 {
				continue
			}
			lx, lz := float64(x)+0.1, float64(y)+0.1
			hy := w.HeightAt(lx, lz)
			b.Cyl(lx, lz, 0.018, hy, hy+0.55, 5, hex(0x8a929c), MatMetal, true)
			b.Box(lx-0.02, hy+0.52, lz-0.02, lx+0.16, hy+0.56, lz+0.16, hex(0x8a929c), MatMetal)
			b.Box(lx+0.06, hy+0.505, lz+0.06, lx+0.15, hy+0.52, lz+0.15, hex(0xffd8a0), 9)
			cx, cz := lx+0.1, lz+0.1
			rad := 0.8
			gy := func(a, c float64) float64 { return w.HeightAt(a, c) + 0.04 }
			k := uint32(len(gb.V) / 10)
			for _, q := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
				px, pz := cx+q[0]*rad, cz+q[1]*rad
				gb.V = append(gb.V, float32(px), float32(gy(px, pz)), float32(pz), float32(q[0]), float32(q[1]), 0, 1.0, 0.62, 0.32, 0)
			}
			gb.I = append(gb.I, k, k+1, k+2, k, k+2, k+3)
		}
	}
	// Stadtmobiliar: Bänke, Pflanzkübel, Straßenbäume
	for y := 1; y < n-1; y++ {
		for x := 1; x < n-1; x++ {
			i := w.Idx(x, y)
			if !w.Road[i] {
				continue
			}
			for k, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				j := w.Idx(x+d[0], y+d[1])
				if w.Road[j] || w.Terr[j] == TWater || (x*5+y*7+k)%5 != 1 {
					continue
				}
				// Rand zur Nicht-Straße
				ex := float64(x) + 0.5 + float64(d[0])*0.42
				ez := float64(y) + 0.5 + float64(d[1])*0.42
				hy := w.HeightAt(ex, ez) + 0.02
				if (x+y+k)%2 == 0 {
					// Straßenbaum im Kübel
					b.Box(ex-0.07, hy, ez-0.07, ex+0.07, hy+0.06, ez+0.07, hex(0x9aa4ae), MatPanel)
					b.Cyl(ex, ez, 0.012, hy+0.06, hy+0.28, 5, mWood, MatMatte, false)
					b.Sphere(ex, hy+0.34, ez, 0.11, 0.1, 0.11, 0, 1, 5, 8, mGreen.Mul(0.85+hash2(x, y, k)*0.3), MatLeaf)
				} else {
					// Bank
					if d[0] != 0 {
						b.Box(ex-0.02, hy, ez-0.1, ex+0.02, hy+0.035, ez+0.1, mWood, MatMatte)
						b.Box(ex+float64(d[0])*0.02, hy+0.035, ez-0.1, ex+float64(d[0])*0.03, hy+0.07, ez+0.1, mWood, MatMatte)
					} else {
						b.Box(ex-0.1, hy, ez-0.02, ex+0.1, hy+0.035, ez+0.02, mWood, MatMatte)
						b.Box(ex-0.1, hy+0.035, ez+float64(d[1])*0.02, ex+0.1, hy+0.07, ez+float64(d[1])*0.03, mWood, MatMatte)
					}
				}
			}
		}
	}
	// Hafenstege an Kontor und Werft
	if r.game != nil {
		for _, bd := range r.game.B {
			if bd.T != BKontor && bd.T != BWerft {
				continue
			}
			bt := &BTypes[bd.T]
			dx, dy := bd.Dock[0], bd.Dock[1]
			var x0, z0, x1, z1 float64
			L := 2.6
			switch {
			case dx < bd.X:
				x0, x1, z0, z1 = float64(bd.X)-L, float64(bd.X), float64(dy)+0.15, float64(dy)+0.85
			case dx >= bd.X+bt.W:
				x0, x1, z0, z1 = float64(bd.X+bt.W), float64(bd.X+bt.W)+L, float64(dy)+0.15, float64(dy)+0.85
			case dy < bd.Y:
				x0, x1, z0, z1 = float64(dx)+0.15, float64(dx)+0.85, float64(bd.Y)-L, float64(bd.Y)
			default:
				x0, x1, z0, z1 = float64(dx)+0.15, float64(dx)+0.85, float64(bd.Y+bt.H), float64(bd.Y+bt.H)+L
			}
			top := bd.BaseY
			b.BoxC(x0, top-0.05, z0, x1, top+0.03, z1, hex(0x8a929c), hex(0xc4ccd4), MatPanel, MatPanel)
			// Pfeiler und Poller
			for t := 0.15; t < 1; t += 0.28 {
				px, pz := x0+(x1-x0)*t, z0+(z1-z0)*t
				if x1-x0 > z1-z0 {
					b.Cyl(px, z0+0.06, 0.04, -1.5, top-0.05, 6, hex(0x6a727c), MatMetal, false)
					b.Cyl(px, z1-0.06, 0.04, -1.5, top-0.05, 6, hex(0x6a727c), MatMetal, false)
					b.Cyl(px, z0+0.08, 0.025, top+0.03, top+0.08, 6, mGraph, MatMetal, true)
				} else {
					b.Cyl(x0+0.06, pz, 0.04, -1.5, top-0.05, 6, hex(0x6a727c), MatMetal, false)
					b.Cyl(x1-0.06, pz, 0.04, -1.5, top-0.05, 6, hex(0x6a727c), MatMetal, false)
					b.Cyl(x0+0.08, pz, 0.025, top+0.03, top+0.08, 6, mGraph, MatMetal, true)
				}
			}
			// Fracht und Bojen am Steg
			if Opt.KenneyModels {
				if x1-x0 > z1-z0 {
					b.Kenney("wc_cargo-pile-a", (x0+x1)/2-0.3, z0+0.05, 0.55, z1-z0-0.1, top+0.03, 1, 0, nil)
				} else {
					b.Kenney("wc_cargo-pile-a", x0+0.05, (z0+z1)/2-0.3, x1-x0-0.1, 0.55, top+0.03, 0, 0, nil)
				}
				bx, bz := x1+0.6, z1+0.6
				if x1-x0 <= z1-z0 {
					bx, bz = x0-0.6, z1+0.6
				}
				b.Kenney("wc_buoy", bx, bz, 0.25, 0.25, -0.05, 0, 0, nil)
			}
			// Leuchtkante
			if x1-x0 > z1-z0 {
				b.Box(x0, top+0.031, z0, x1, top+0.035, z0+0.02, mCyan, MatEmit)
				b.Box(x0, top+0.031, z1-0.02, x1, top+0.035, z1, mCyan, MatEmit)
			} else {
				b.Box(x0, top+0.031, z0, x0+0.02, top+0.035, z1, mCyan, MatEmit)
				b.Box(x1-0.02, top+0.031, z0, x1, top+0.035, z1, mCyan, MatEmit)
			}
		}
	}
	if r.roads == nil {
		r.roads = b.Upload()
	} else {
		r.roads.Update(b)
	}
	if r.glows == nil {
		r.glows = gb.Upload()
	} else {
		r.glows.Update(gb)
	}
}

func (r *Renderer) buildTrees(w *World) {
	data := make([][]float32, len(r.trees))
	add := func(k int, x, y, z, rot, s float64, tint RGB) {
		data[k] = append(data[k], float32(x), float32(y), float32(z), float32(rot), float32(s),
			float32(tint.R)/255*1.0, float32(tint.G)/255, float32(tint.B)/255)
	}
	n := w.N
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := w.Idx(x, y)
			if w.Occ[i] != 0 || w.Road[i] {
				continue
			}
			if w.Dep[i] != 0 {
				k := 5
				if w.Dep[i] == DepSil {
					k = 6
				}
				px, pz := float64(x)+0.5, float64(y)+0.5
				add(k, px, w.HeightAt(px, pz), pz, hash2(x, y, 3)*6, 1.1, RGB{255, 255, 255})
				continue
			}
			d := w.Deco[i]
			if d == 0 {
				continue
			}
			cnt := 1
			if d <= 3 {
				cnt = []int{1, 2, 3}[clampi(Opt.TreeDensity, 0, 2)] + int(hash2(x, y, 11)*2)
			}
			for c := 0; c < cnt; c++ {
				px := float64(x) + 0.2 + hash2(x*3+c, y, 21)*0.6
				pz := float64(y) + 0.2 + hash2(x, y*3+c, 22)*0.6
				hy := w.HeightAt(px, pz)
				s := 0.75 + hash2(x+c, y, 23)*0.6
				tv := 0.85 + hash2(x, y+c, 24)*0.3
				tint := RGB{uint8(255 * tv), 255, uint8(255 * tv)}
				switch d {
				case 1:
					add(0, px, hy, pz, hash2(x, y, c)*6, s, tint)
				case 2:
					add(1, px, hy, pz, hash2(x, y, c)*6, s*1.1, tint)
				case 3:
					add(2, px, hy, pz, hash2(x, y, c)*6, s, tint)
				case 4:
					add(3, px, hy-0.05, pz, hash2(x, y, c)*6, s*1.2, RGB{255, 255, 255})
				case 5:
					add(4, px, hy, pz, hash2(x, y, c)*6, s, RGB{255, 255, 255})
				}
			}
		}
	}
	for k := range r.trees {
		r.trees[k].Set(data[k])
	}
	// Bodendetails
	gd := make([][]float32, 3)
	addG := func(k int, x, z, rot, sc float64, tint RGB) {
		gd[k] = append(gd[k], float32(x), float32(w.HeightAt(x, z)), float32(z), float32(rot), float32(sc),
			float32(tint.R)/255, float32(tint.G)/255, float32(tint.B)/255)
	}
	dens := []int{2, 4, 6}[clampi(Opt.TreeDensity, 0, 2)]
	flowerCols := []RGB{hex(0xffe060), hex(0xc890ff), hex(0xffffff), hex(0xff8080)}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := w.Idx(x, y)
			if w.Occ[i] != 0 || w.Road[i] {
				continue
			}
			terr := w.Terr[i]
			if w.Moon && terr == TGrass {
				terr = TRock
			}
			switch terr {
			case TGrass:
				for c := 0; c < dens; c++ {
					px := float64(x) + hash2(x*5+c, y, 31)
					pz := float64(y) + hash2(x, y*5+c, 32)
					v := 0.8 + hash2(x+c, y, 33)*0.4
					addG(0, px, pz, hash2(x, y+c, 34)*6, 0.8+hash2(x, c, 35)*0.6, RGB{uint8(220 * v), 255, uint8(200 * v)})
				}
				if hash2(x, y, 36) < 0.22 {
					for c := 0; c < 3; c++ {
						px := float64(x) + 0.2 + hash2(x*3+c, y, 37)*0.6
						pz := float64(y) + 0.2 + hash2(x, y*3+c, 38)*0.6
						addG(1, px, pz, 0, 0.8+hash2(x, c, 39)*0.5, flowerCols[int(hash2(x, y, 40)*4)])
					}
				}
			case TSand, TRock:
				if hash2(x, y, 41) < 0.35 {
					px := float64(x) + hash2(x, y, 42)
					pz := float64(y) + hash2(x, y, 43)
					addG(2, px, pz, hash2(x, y, 44)*6, 0.6+hash2(x, y, 45)*0.8, RGB{255, 255, 255})
				}
			}
		}
	}
	for k := range r.ground {
		r.ground[k].Set(gd[k])
	}
}

func (r *Renderer) buildWater() {
	b := NewMB()
	n := MapN
	lo, hi, step := -160, n+160, 2
	cnt := (hi-lo)/step + 1
	for z := 0; z < cnt; z++ {
		for x := 0; x < cnt; x++ {
			b.V = append(b.V, float32(lo+x*step), 0, float32(lo+z*step), 0, 1, 0, 0, 0, 0, 0)
		}
	}
	for z := 0; z < cnt-1; z++ {
		for x := 0; x < cnt-1; x++ {
			i := uint32(z*cnt + x)
			j := i + uint32(cnt)
			b.I = append(b.I, i, j, i+1, i+1, j, j+1)
		}
	}
	r.water = b.Upload()
}

func (r *Renderer) model(t, v, tier int) *GMesh {
	if t != BHouse {
		tier = 0
	}
	if t != BHouse && t != BWind && t != BPark && t != BSchrott {
		v = v % 2
	}
	key := Tf("%d_%d_%d", t, v%4, tier)
	if m, ok := r.models[key]; ok {
		return m
	}
	mb := BuildModel(t, v%4, tier)
	m := mb.Upload()
	hy := 0.5
	for i := 1; i < len(mb.V); i += 10 {
		if float64(mb.V[i]) > hy {
			hy = float64(mb.V[i])
		}
	}
	r.modelH[key] = hy
	r.models[key] = m
	return m
}

func (r *Renderer) modelHeight(t, v, tier int) float64 {
	r.model(t, v, tier)
	if t != BHouse {
		tier = 0
	}
	if t != BHouse && t != BWind && t != BPark && t != BSchrott {
		v = v % 2
	}
	return r.modelH[Tf("%d_%d_%d", t, v%4, tier)]
}

func (r *Renderer) scaffold(w, h int) *GMesh {
	k := w*10 + h
	if m, ok := r.scaffolds[k]; ok {
		return m
	}
	m := ScaffoldModel(w, h).Upload()
	r.scaffolds[k] = m
	return m
}

// ---------------------------------------------------------------- Kamera

const camFov = 0.62

func (u *UIState) camPitch() float64 {
	t := clampf((u.Dist-8)/80, 0, 1)
	return 0.6 + t*0.5
}

func (u *UIState) UpdateCam(g *Game, w, h int) {
	if u.Dist == 0 {
		u.Dist = 32
		u.Yaw = math.Pi / 4
	}
	u.W, u.H = w, h
	ty := math.Max(0, g.W.HeightAt(u.TX, u.TZ))
	u.camTY += (ty - u.camTY) * 0.2
	p := u.camPitch()
	dir := V3{math.Sin(u.Yaw) * math.Cos(p), math.Sin(p), math.Cos(u.Yaw) * math.Cos(p)}
	tgt := V3{u.TX, u.camTY, u.TZ}
	u.Eye = tgt.Add(dir.Mul(u.Dist))
	view := LookAt(u.Eye, tgt, V3{0, 1, 0})
	proj := Perspective(camFov, float64(w)/float64(imax(1, h)), 0.5, 900)
	u.Proj = proj
	u.InvProj = proj.Inverse()
	u.VP = proj.Mul(view)
	u.InvVP = u.VP.Inverse()
}

// Weltposition (Kachelkoordinaten) -> Bildschirm
func (u *UIState) WorldToScreen(x, y, sw, sh float64) (float64, float64) {
	h := 0.0
	if u.world != nil {
		h = math.Max(0, u.world.HeightAt(x, y))
	}
	return u.Project(V3{x, h, y}, sw, sh)
}

func (u *UIState) Project(p V3, sw, sh float64) (float64, float64) {
	cx, cy, _, cw := u.VP.MulV(p, 1)
	if cw <= 0.01 {
		return -9999, -9999
	}
	return (cx/cw*0.5 + 0.5) * sw, (1 - (cy/cw*0.5 + 0.5)) * sh
}

func (u *UIState) ray(sx, sy, sw, sh float64) (V3, V3) {
	nx := sx/sw*2 - 1
	ny := 1 - sy/sh*2
	ax, ay, az, aw := u.InvVP.MulV(V3{nx, ny, -1}, 1)
	bx, by, bz, bw := u.InvVP.MulV(V3{nx, ny, 1}, 1)
	a := V3{ax / aw, ay / aw, az / aw}
	b := V3{bx / bw, by / bw, bz / bw}
	return a, b.Sub(a).Norm()
}

// Bildschirm -> Weltposition (Schnitt mit Gelände bzw. Wasseroberfläche)
func (u *UIState) ScreenToWorld(sx, sy, sw, sh float64) (float64, float64) {
	o, d := u.ray(sx, sy, sw, sh)
	if u.world == nil {
		if d.Y >= 0 {
			return o.X, o.Z
		}
		t := -o.Y / d.Y
		p := o.Add(d.Mul(t))
		return p.X, p.Z
	}
	ground := func(p V3) float64 { return math.Max(0, u.world.HeightAt(p.X, p.Z)) }
	t := 0.0
	step := 0.4
	prev := o
	for i := 0; i < 3000; i++ {
		p := o.Add(d.Mul(t))
		if p.Y <= ground(p) {
			// verfeinern
			lo, hi := t-step, t
			for k := 0; k < 12; k++ {
				m := (lo + hi) / 2
				q := o.Add(d.Mul(m))
				if q.Y <= ground(q) {
					hi = m
				} else {
					lo = m
				}
			}
			q := o.Add(d.Mul(hi))
			return q.X, q.Z
		}
		prev = p
		t += step
		if p.Y < -5 {
			break
		}
	}
	_ = prev
	if d.Y < 0 {
		tt := -o.Y / d.Y
		p := o.Add(d.Mul(tt))
		return p.X, p.Z
	}
	return u.TX, u.TZ
}

func (u *UIState) CenterOn(g *Game, x, y float64) {
	u.TX, u.TZ = x, y
	u.world = g.W
	u.camTY = math.Max(0, g.W.HeightAt(x, y))
}

// ---------------------------------------------------------------- Zeichnen

type sunSet struct {
	sun, sunCol, sky, gnd, horizon, fog V3
	realSun                             V3
	fogD, night, storm                  float64
	morning, sunset                     float64
}

func lerp3(a, b V3, t float64) V3 { return a.Add(b.Sub(a).Mul(clampf(t, 0, 1))) }
func smoothstepF(a, b, x float64) float64 {
	t := clampf((x-a)/(b-a), 0, 1)
	return t * t * (3 - 2*t)
}

// Licht abhängig von Uhrzeit (0..24) und Sturm (0..1)
func (r *Renderer) lighting(g *Game) sunSet {
	h, storm := 14.0, 0.0
	if g != nil {
		h, storm = g.Hour, g.Storm
	}
	// Sonnenaufgang 6 Uhr, Sonnenuntergang 20 Uhr
	var a, el float64
	if h >= 6 && h <= 20 {
		a = (h - 6) / 14 * math.Pi
		el = math.Sin(a)
	} else {
		hn := h
		if hn < 6 {
			hn += 24
		}
		t := (hn - 20) / 10
		a = math.Pi + t*math.Pi
		el = -math.Sin(t*math.Pi) * 0.7
	}
	sunDir := V3{-math.Cos(a) * 0.9, el * 1.15, 0.55}.Norm()
	day := smoothstepF(-0.06, 0.25, el)
	night := 1 - smoothstepF(-0.2, 0.04, el)
	sunset := (1 - smoothstepF(0.06, 0.38, el)) * smoothstepF(-0.14, 0.02, el)
	dayCol := V3{1.0, 0.91, 0.78}.Mul(2.4)
	setCol := V3{1.0, 0.52, 0.26}.Mul(2.1)
	moonCol := V3{0.45, 0.55, 0.85}.Mul(0.42)
	sc := lerp3(moonCol, lerp3(dayCol, setCol, sunset), day)
	ldir := sunDir
	if el < -0.02 {
		ldir = V3{math.Cos(a) * 0.9, -el*1.15 + 0.25, 0.4}.Norm()
	}
	sky := lerp3(V3{0.012, 0.02, 0.055}, lerp3(V3{0.26, 0.42, 0.72}, V3{0.32, 0.3, 0.48}, sunset), day)
	hor := lerp3(V3{0.03, 0.05, 0.11}, lerp3(V3{0.62, 0.74, 0.88}, V3{0.98, 0.56, 0.34}, sunset), day)
	gnd := lerp3(V3{0.025, 0.03, 0.05}, V3{0.20, 0.18, 0.14}, day)
	// Sturm: dunkler, grauer
	grey := V3{0.32, 0.35, 0.40}
	sc = sc.Mul(1 - 0.72*storm)
	sky = lerp3(sky, grey.Mul(0.4+0.6*day), storm*0.85)
	hor = lerp3(hor, grey.Mul(0.5+0.7*day), storm*0.85)
	fogD := []float64{0.0034, 0.0019, 0.0011}[clampi(Opt.ViewDist, 0, 2)] * (1 + storm*2.5)
	if g != nil && g.OnMoon() {
		// keine Atmosphäre: schwarzer Himmel, hartes weißes Licht
		ms := V3{1.0, 0.97, 0.93}.Mul(2.7)
		sc = lerp3(V3{0.18, 0.22, 0.32}, ms, day)
		sky = V3{0.003, 0.004, 0.009}
		hor = V3{0.012, 0.014, 0.022}
		gnd = lerp3(V3{0.02, 0.022, 0.03}, V3{0.09, 0.09, 0.1}, day)
		return sunSet{sun: ldir, realSun: sunDir, sunCol: sc, sky: sky, gnd: gnd, horizon: hor,
			fog: V3{0.01, 0.012, 0.02}, fogD: fogD * 0.35, night: math.Max(night, 0.35), storm: 0}
	}
	// Morgennebel zwischen 5 und 9 Uhr
	morning := smoothstepF(4.5, 6.3, h) * (1 - smoothstepF(7.5, 9.5, h)) * (1 - storm)
	if !Opt.HeightFog {
		morning = 0
	}
	fogD *= 1 + morning*1.4
	return sunSet{
		sun: ldir, realSun: sunDir, sunCol: sc, sky: sky, gnd: gnd, horizon: hor,
		fog: lerp3(hor.Mul(0.92), V3{0.78, 0.74, 0.7}.Mul(0.6+0.4*day), morning*0.6), fogD: fogD, night: night, storm: storm,
		morning: morning, sunset: sunset,
	}
}

func v3f(v V3) []float32 { return []float32{float32(v.X), float32(v.Y), float32(v.Z)} }

func (r *Renderer) setMeshUniforms(p *Prog, vp Mat4, lightVP Mat4, u *UIState, ls sunSet) {
	p.Use()
	glUniformMat(p.L("uVP"), &vp)
	glUniformMat(p.L("uLightVP"), &lightVP)
	glUniformV(p.L("uSun"), v3f(ls.sun)...)
	glUniformV(p.L("uSunCol"), v3f(ls.sunCol)...)
	glUniformV(p.L("uSky"), v3f(ls.sky)...)
	glUniformV(p.L("uGnd"), v3f(ls.gnd)...)
	glUniformV(p.L("uCam"), v3f(u.Eye)...)
	glUniformV(p.L("uFog"), v3f(ls.fog)...)
	glUniformV(p.L("uFogD"), float32(ls.fogD))
	glUniformV(p.L("uTime"), float32(r.time))
	glUniformV(p.L("uNight"), float32(ls.night))
	glUniformI(p.L("uPCF"), clampi(Opt.ShadowSoft, 0, 2))
	glUniformI(p.L("uClip"), 0)
	glUniformV(p.L("uHFog"), float32(b2u(Opt.HeightFog))*float32(0.6+0.4*ls.night+2.2*ls.morning))
	glUniformV(p.L("uClouds"), float32(b2u(Opt.Clouds)))
	glUniformV(p.L("uTint"), 0, 0, 0, 0)
	glUniformV(p.L("uAlpha"), 1)
	glUniformV(p.L("uAO"), 0)
	glUniformV(p.L("uMoon"), float32(r.moonF))
	glUniformI(p.L("uShadow"), 1)
	glUniformI(p.L("uInst"), 0)
	id := Ident()
	glUniformMat(p.L("uModel"), &id)
}

// zeichnet alle Schattenwerfer/Objekte mit dem aktiven Programm
func (r *Renderer) drawObjects(g *Game, p *Prog, shadow bool) {
	u := g.UI
	id := Ident()
	glUniformI(p.L("uInst"), 0)
	glUniformMat(p.L("uModel"), &id)
	glUniformV(p.L("uAO"), 0)
	r.terrain.Draw()
	if !shadow {
		r.roads.Draw()
	}
	glUniformV(p.L("uAO"), 1)
	// Sichtbarkeits-Grobtest
	vis := func(x, z, rad float64) bool {
		cx, cy, _, cw := u.VP.MulV(V3{x, 0.5, z}, 1)
		if shadow {
			return math.Hypot(x-u.TX, z-u.TZ) < u.Dist*1.6+20
		}
		if cw <= 0 {
			return false
		}
		m := 1.25 + rad*2/cw
		return math.Abs(cx/cw) < m && math.Abs(cy/cw) < m
	}
	for _, b := range g.allBuildings() {
		bt := &BTypes[b.T]
		cx, cz := b.Center()
		if !vis(cx, cz, float64(bt.W+bt.H)+3) {
			continue
		}
		if (b.T == BKontor || b.T == BLager) && b.Own < len(r.flags) && r.flags[b.Own] != nil {
			fx, fz := float64(b.X)+0.25, float64(b.Y)+0.25
			fm := Translate(fx, b.BaseY, fz).Mul(RotY(-math.Pi / 4))
			glUniformMat(p.L("uModel"), &fm)
			r.flags[b.Own].Draw()
		}
		m := Translate(float64(b.X), b.BaseY, float64(b.Y))
		if b.Constr > 0 {
			pr := 1 - b.Constr
			s := smoothstepF(0.12, 1, pr)
			hm := r.modelHeight(b.T, b.Var, b.Tier)
			if s > 0.01 {
				ms := m.Mul(ScaleM(1, s, 1))
				glUniformMat(p.L("uModel"), &ms)
				r.model(b.T, b.Var, b.Tier).Draw()
			}
			// Gerüst wächst mit, Kran dreht sich
			sh := math.Max(0.3, hm*math.Min(1, s+0.15))
			sm := Translate(float64(b.X)-0.04, b.BaseY, float64(b.Y)-0.04).Mul(ScaleM((float64(bt.W)+0.08)/float64(bt.W), sh, (float64(bt.H)+0.08)/float64(bt.H)))
			glUniformMat(p.L("uModel"), &sm)
			r.scaffold(bt.W, bt.H).Draw()
			if bt.W*bt.H >= 4 {
				ch := hm + 0.8
				cxp, czp := float64(b.X)+float64(bt.W)+0.25, float64(b.Y)+0.3
				cm := Translate(cxp, b.BaseY, czp).Mul(ScaleM(1, ch, 1))
				glUniformMat(p.L("uModel"), &cm)
				r.crane.Draw()
				jm := Translate(cxp, b.BaseY+ch, czp).Mul(RotY(math.Sin(r.time*0.4+float64(b.ID))*1.2 + 2.4))
				glUniformMat(p.L("uModel"), &jm)
				r.craneJib.Draw()
			}
			continue
		}
		glUniformMat(p.L("uModel"), &m)
		if far := math.Hypot(cx-u.TX, cz-u.TZ); far > 40+u.Dist*0.9 || (shadow && far > 22+u.Dist*0.5) {
			r.lodModel(b.T, b.Var, b.Tier).Draw()
			continue
		}
		r.model(b.T, b.Var, b.Tier).Draw()
		if b.T == BRaumhafen && r.game != nil && r.game.Other != nil {
			lt := r.game.Launch
			ry := 0.22
			if lt < 9 {
				ry += lt * lt * 0.9
			}
			if lt < 14 || lt > 30 {
				rm := Translate(float64(b.X)+2.6, b.BaseY+ry, float64(b.Y)+1.4)
				glUniformMat(p.L("uModel"), &rm)
				r.rocket.Draw()
			}
		} else if b.T == BRaumhafen {
			rm := Translate(float64(b.X)+2.6, b.BaseY+0.22, float64(b.Y)+1.4)
			glUniformMat(p.L("uModel"), &rm)
			r.rocket.Draw()
		}
		if b.T == BTurret {
			hm := Translate(float64(b.X)+0.5, b.BaseY+0.88, float64(b.Y)+0.5).Mul(RotY(-b.Aim))
			glUniformMat(p.L("uModel"), &hm)
			r.turretHead.Draw()
		}
		if b.T == BWind {
			ang := 0.0
			if !b.Paused {
				ang = b.Anim * 2.2
			}
			rm := Translate(float64(b.X)+0.5, b.BaseY+3.35, float64(b.Y)+0.5).Mul(RotY(-math.Pi / 4)).Mul(Translate(0, 0, 0.14)).Mul(RotZ(ang))
			glUniformMat(p.L("uModel"), &rm)
			r.rotor.Draw()
		}
	}
	glUniformV(p.L("uAO"), 0)
	for _, c := range g.Carts {
		x, z, a := c.WorldPos()
		if !vis(x, z, 1) {
			continue
		}
		m := Translate(x, g.W.HeightAt(x, z)+0.03, z).Mul(RotY(-a))
		glUniformMat(p.L("uModel"), &m)
		r.cart.Draw()
	}
	for _, s := range g.Ships {
		if !vis(s.X, s.Y, 3) {
			continue
		}
		bob := math.Sin(r.time*1.6+float64(s.ID))*0.04 + 0.08
		m := Translate(s.X, bob, s.Y).Mul(RotY(-s.Ang)).Mul(RotX(math.Sin(r.time*1.1+float64(s.ID)) * 0.02))
		glUniformMat(p.L("uModel"), &m)
		if s.War && r.warships[s.Kind] != nil {
			r.warships[s.Kind].Draw()
		} else {
			r.ships[s.Kind].Draw()
		}
	}
	glUniformMat(p.L("uModel"), &id)
	glUniformI(p.L("uInst"), 1)
	for _, t := range r.trees {
		t.Draw()
	}
	if r.fx != nil && Opt.Traffic {
		r.fx.carInst.Draw()
	}
	if r.life != nil {
		r.life.draw(shadow)
	}
	if !shadow && Opt.GroundDetail {
		for _, gi := range r.ground {
			gi.Draw()
		}
	}
	glUniformI(p.L("uInst"), 0)
}

func (r *Renderer) drawQuad() {
	gl("glBindVertexArray", uintptr(r.emptyVAO))
	gl("glDrawArrays", GL_TRIANGLES, 0, 3)
}

func bindTex(unit int, tex uint32) {
	gl("glActiveTexture", uintptr(GL_TEXTURE0+unit))
	gl("glBindTexture", GL_TEXTURE_2D, uintptr(tex))
}

// Vollständiger Frame: 3D-Szene + UI-Leinwand + Glas-Panels
func (r *Renderer) Render(g *Game, ui *Canvas, rects [][4]int, uiOn bool, dt float64) {
	r.time += dt
	u := g.UI
	u.world = g.W
	r.game = g
	r.moonF = float64(b2u(g.OnMoon()))
	if r.world != g.W {
		r.world = g.W
		r.terrain = nil
		r.buildTerrain(g.W)
		r.buildRoads(g.W)
		r.buildTrees(g.W)
		r.roadSig = g.W.Sig()
	}
	if g.W.HDirty {
		r.buildTerrain(g.W)
		r.buildRoads(g.W)
		r.buildTrees(g.W)
	}
	if GeomDirty {
		GeomDirty = false
		r.buildTrees(g.W)
		if r.kenneyOn != Opt.KenneyModels {
			r.kenneyOn = Opt.KenneyModels
			r.models = map[string]*GMesh{}
			r.modelH = map[string]float64{}
			r.makeThumbs()
		}
	}
	SetAudioEnv(g.Storm, r.lighting(g).night)
	{
		mood := 0
		if r.lighting(g).night > 0.6 {
			mood = 1
		}
		for _, rv := range g.Rivals {
			if rv.War {
				mood = 2
			}
		}
		if g.OnMoon() {
			mood = 3
		}
		SetMusicMood(mood)
	}
	if s := g.W.Sig(); s != r.roadSig {
		r.roadSig = s
		r.buildRoads(g.W)
		r.buildTrees(g.W)
	}
	ls := r.lighting(g)
	r.fx.update(g, dt, u)
	r.fx.uploadCars()
	r.life.update(g, u, dt)
	r.life.upload(g, r.time)
	// --- Schattenkarte
	ext := clampf(u.Dist*1.05, 14, []float64{55, 70, 95}[clampi(Opt.ViewDist, 0, 2)])
	tgt := V3{u.TX, 0, u.TZ}
	lview := LookAt(tgt.Add(ls.sun.Mul(120)), tgt, V3{0, 1, 0})
	// Texel-Snapping gegen Flimmern
	lp := Ortho(-ext, ext, -ext, ext, 10, 260)
	lvp := lp.Mul(lview)
	{
		ox, oy, _, _ := lvp.MulV(V3{0, 0, 0}, 1)
		ts := 2.0 / float64(r.shadowSize)
		dx := math.Round(ox/ts)*ts - ox
		dy := math.Round(oy/ts)*ts - oy
		lvp = Translate(dx, dy, 0).Mul(lvp)
	}
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.shadowFBO))
	gl("glViewport", 0, 0, uintptr(r.shadowSize), uintptr(r.shadowSize))
	gl("glEnable", GL_DEPTH_TEST)
	gl("glDepthFunc", GL_LEQUAL)
	gl("glDepthMask", 1)
	gl("glDisable", GL_BLEND)
	gl("glDisable", GL_CULL_FACE)
	glClearDepthBuf()
	r.pDepth.Use()
	glUniformMat(r.pDepth.L("uVP"), &lvp)
	glUniformV(r.pDepth.L("uTime"), float32(r.time))
	r.drawObjects(g, r.pDepth, true)
	if Opt.Reflections && !g.OnMoon() {
		r.reflectionPass(g, ls, lvp)
	}
	// --- Hauptszene
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.msFBO))
	gl("glViewport", 0, 0, uintptr(r.W), uintptr(r.H))
	glClearColorBuf(float32(ls.horizon.X), float32(ls.horizon.Y), float32(ls.horizon.Z), 1)
	glClearDepthBuf()
	// Himmel
	gl("glDisable", GL_DEPTH_TEST)
	gl("glDepthMask", 0)
	r.pSky.Use()
	glUniformMat(r.pSky.L("uInvVP"), &u.InvVP)
	glUniformV(r.pSky.L("uSun"), v3f(ls.sun)...)
	glUniformV(r.pSky.L("uSky"), v3f(ls.sky)...)
	glUniformV(r.pSky.L("uHorizon"), v3f(ls.horizon)...)
	glUniformV(r.pSky.L("uSunCol"), v3f(ls.sunCol)...)
	glUniformV(r.pSky.L("uRealSun"), v3f(ls.realSun)...)
	glUniformV(r.pSky.L("uNight"), float32(ls.night))
	glUniformV(r.pSky.L("uMoon"), float32(b2u(g.OnMoon())))
	glUniformV(r.pSky.L("uTime"), float32(r.time))
	r.drawQuad()
	gl("glEnable", GL_DEPTH_TEST)
	gl("glDepthMask", 1)
	bindTex(1, r.shadowTex)
	r.setMeshUniforms(r.pMesh, u.VP, lvp, u, ls)
	r.drawObjects(g, r.pMesh, false)
	if Opt.SSAO {
		r.ssao(u)
	}
	// Lichtkegel der Laternen (nachts)
	if ls.night > 0.01 && r.glows != nil && r.glows.N > 0 {
		gl("glEnable", GL_BLEND)
		gl("glBlendFunc", GL_ONE, GL_ONE)
		gl("glDepthMask", 0)
		r.pGlow.Use()
		glUniformMat(r.pGlow.L("uVP"), &u.VP)
		glUniformV(r.pGlow.L("uNight"), float32(ls.night))
		r.glows.Draw()
		gl("glDepthMask", 1)
		gl("glDisable", GL_BLEND)
	}
	// Wasser
	gl("glEnable", GL_BLEND)
	gl("glBlendFunc", GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA)
	gl("glDepthMask", 0)
	r.pWater.Use()
	glUniformMat(r.pWater.L("uVP"), &u.VP)
	glUniformMat(r.pWater.L("uLightVP"), &lvp)
	glUniformV(r.pWater.L("uSun"), v3f(ls.sun)...)
	glUniformV(r.pWater.L("uSunCol"), v3f(ls.sunCol)...)
	glUniformV(r.pWater.L("uSky"), v3f(ls.sky)...)
	glUniformV(r.pWater.L("uGnd"), v3f(ls.horizon)...)
	glUniformV(r.pWater.L("uCam"), v3f(u.Eye)...)
	glUniformV(r.pWater.L("uFog"), v3f(ls.fog)...)
	glUniformV(r.pWater.L("uFogD"), float32(ls.fogD))
	glUniformV(r.pWater.L("uTime"), float32(r.time))
	glUniformV(r.pWater.L("uNight"), float32(ls.night))
	glUniformV(r.pWater.L("uClouds"), float32(b2u(Opt.Clouds)))
	glUniformV(r.pWater.L("uStorm"), float32(ls.storm))
	glUniformV(r.pWater.L("uMoon"), float32(b2u(g.OnMoon())))
	glUniformV(r.pWater.L("uMapN"), float32(g.W.N+1))
	glUniformI(r.pWater.L("uHeight"), 2)
	glUniformI(r.pWater.L("uShadow"), 1)
	glUniformI(r.pWater.L("uRefl"), 3)
	glUniformI(r.pWater.L("uReflOn"), int(b2u(Opt.Reflections && !g.OnMoon())))
	glUniformV(r.pWater.L("uView"), float32(r.W), float32(r.H))
	glUniformV(r.pWater.L("uHFog"), float32(b2u(Opt.HeightFog))*float32(0.6+0.4*ls.night+2.2*ls.morning))
	bindTex(2, r.heightTex)
	bindTex(3, r.reflTex)
	r.water.Draw()
	gl("glDepthMask", 1)
	// Rauch und Dampf
	r.fx.drawParticles(r, u, ls)
	r.fx.drawGlowFX(r, u)
	// Wolkendecke bei weiter Sicht
	if Opt.Clouds && !g.OnMoon() {
		fade := smoothstepF(42, 85, u.Dist)
		if fade > 0.01 {
			gl("glEnable", GL_BLEND)
			gl("glBlendFunc", GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA)
			gl("glDepthMask", 0)
			pc := r.pCloud
			pc.Use()
			glUniformMat(pc.L("uVP"), &u.VP)
			glUniformV(pc.L("uY"), 16)
			glUniformV(pc.L("uTime"), float32(r.time))
			glUniformV(pc.L("uSunCol"), v3f(ls.sunCol)...)
			glUniformV(pc.L("uSky"), v3f(ls.sky.Add(ls.horizon).Mul(0.5))...)
			glUniformV(pc.L("uFade"), float32(fade))
			glUniformV(pc.L("uStorm"), float32(ls.storm))
			r.water.Draw()
			gl("glDepthMask", 1)
		}
	}
	// Überlagerungen (Bauvorschau, Auswahl, Einfluss)
	r.drawOverlays(g, ls, lvp)
	gl("glDisable", GL_BLEND)
	if Opt.DOF && !Opt.SSAO {
		gl("glBindFramebuffer", GL_READ_FRAMEBUFFER, uintptr(r.msFBO))
		gl("glBindFramebuffer", GL_DRAW_FRAMEBUFFER, uintptr(r.depthFBO))
		gl("glBlitFramebuffer", 0, 0, uintptr(r.W), uintptr(r.H), 0, 0, uintptr(r.W), uintptr(r.H), GL_DEPTH_BUFFER_BIT, GL_NEAREST)
	}
	// --- Auflösen + Weichzeichnen
	gl("glBindFramebuffer", GL_READ_FRAMEBUFFER, uintptr(r.msFBO))
	gl("glBindFramebuffer", GL_DRAW_FRAMEBUFFER, uintptr(r.resFBO))
	gl("glBlitFramebuffer", 0, 0, uintptr(r.W), uintptr(r.H), 0, 0, uintptr(r.W), uintptr(r.H), GL_COLOR_BUFFER_BIT, GL_NEAREST)
	gl("glDisable", GL_DEPTH_TEST)
	hw, hh := imax(1, r.W/2), imax(1, r.H/2)
	qw, qh := imax(1, r.W/4), imax(1, r.H/4)
	r.pDown.Use()
	glUniformI(r.pDown.L("uTex"), 0)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.halfFBO))
	gl("glViewport", 0, 0, uintptr(hw), uintptr(hh))
	bindTex(0, r.resTex)
	glUniformV(r.pDown.L("uTexel"), 0.5/float32(r.W), 0.5/float32(r.H))
	r.drawQuad()
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.qFBO))
	gl("glViewport", 0, 0, uintptr(qw), uintptr(qh))
	bindTex(0, r.halfTex)
	glUniformV(r.pDown.L("uTexel"), 0.5/float32(hw), 0.5/float32(hh))
	r.drawQuad()
	r.pBlur.Use()
	glUniformI(r.pBlur.L("uTex"), 0)
	for it := 0; it < 2; it++ {
		gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.q2FBO))
		bindTex(0, r.qTex)
		glUniformV(r.pBlur.L("uDir"), 1/float32(qw), 0)
		r.drawQuad()
		gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.qFBO))
		bindTex(0, r.q2Tex)
		glUniformV(r.pBlur.L("uDir"), 0, 1/float32(qh))
		r.drawQuad()
	}
	// --- UI-Textur
	if uiOn && ui != nil {
		if r.uiTex == 0 || r.uiW != ui.W || r.uiH != ui.H {
			delTex(&r.uiTex)
			r.uiTex = glGenTexture()
			gl("glBindTexture", GL_TEXTURE_2D, uintptr(r.uiTex))
			gl("glTexImage2D", GL_TEXTURE_2D, 0, GL_RGBA8, uintptr(ui.W), uintptr(ui.H), 0, GL_BGRA, GL_UNSIGNED_BYTE, 0)
			texParams(GL_LINEAR, GL_CLAMP_TO_EDGE)
			r.uiW, r.uiH = ui.W, ui.H
		}
		gl("glBindTexture", GL_TEXTURE_2D, uintptr(r.uiTex))
		gl("glPixelStorei", GL_UNPACK_ALIGNMENT, 4)
		gl("glTexSubImage2D", GL_TEXTURE_2D, 0, 0, 0, uintptr(ui.W), uintptr(ui.H), GL_BGRA, GL_UNSIGNED_BYTE, ptr(unsafe.Pointer(&ui.Pix[0])))
		runtime.KeepAlive(ui.Pix)
	}
	// --- Zusammensetzen
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.OutFBO))
	gl("glViewport", 0, 0, uintptr(r.OutW), uintptr(r.OutH))
	r.pComp.Use()
	glUniformV(r.pComp.L("uTime"), float32(r.time))
	rain := 0.0
	if g.Storm > 0.35 && !g.OnMoon() {
		rain = (g.Storm - 0.35) / 0.65
	}
	glUniformV(r.pComp.L("uRain"), float32(rain))
	glUniformV(r.pComp.L("uFlash"), float32(g.Flash*(1-float64(b2u(g.OnMoon())))))
	glUniformI(r.pComp.L("uScene"), 0)
	glUniformI(r.pComp.L("uBlur"), 1)
	glUniformI(r.pComp.L("uUI"), 2)
	bindTex(0, r.resTex)
	bindTex(1, r.qTex)
	bindTex(2, r.uiTex)
	glUniformV(r.pComp.L("uScreen"), float32(r.OutW), float32(r.OutH))
	bl := 0.0
	if Opt.Bloom {
		bl = 0.55
	}
	glUniformV(r.pComp.L("uBloom"), float32(bl))
	glUniformI(r.pComp.L("uDepth"), 3)
	bindTex(3, r.depthTex)
	glUniformI(r.pComp.L("uDOF"), int(b2u(Opt.DOF && (u.Screen == ScrGame || u.Screen == ScrMenu))))
	glUniformV(r.pComp.L("uFocus"), float32(u.Dist))
	{
		// Sonnenposition auf dem Bildschirm für Lichtstrahlen
		ls := r.lighting(g)
		sp := u.Eye.Add(ls.realSun.Mul(2000))
		m := func(i int) float64 { return float64(u.VP[i]) }
		cx := m(0)*sp.X + m(4)*sp.Y + m(8)*sp.Z + m(12)
		cy := m(1)*sp.X + m(5)*sp.Y + m(9)*sp.Z + m(13)
		cw := m(3)*sp.X + m(7)*sp.Y + m(11)*sp.Z + m(15)
		rays := 0.0
		if cw > 0 && Opt.GodRays && !g.OnMoon() && ls.realSun.Y > -0.05 {
			rays = (0.35 + 0.9*ls.sunset + 0.5*ls.morning) * (1 - ls.night) * (1 - ls.storm)
		}
		sx, sy := 0.5, 0.5
		if cw > 0 {
			sx, sy = (cx/cw)*0.5+0.5, (cy/cw)*0.5+0.5
		}
		glUniformV(r.pComp.L("uSunUV"), float32(sx), float32(sy))
		glUniformV(r.pComp.L("uRays"), float32(rays))
		rc := ls.sunCol.Mul(0.22)
		glUniformV(r.pComp.L("uRayCol"), float32(rc.X), float32(rc.Y), float32(rc.Z))
	}
	glUniformV(r.pComp.L("uNF"), 0.5, 900)
	gr := 0.0
	if Opt.Grain {
		gr = 0.022
	}
	glUniformV(r.pComp.L("uGrain"), float32(gr))
	glUniformI(r.pComp.L("uUIOn"), int(b2u(uiOn && ui != nil)))
	var rs []float32
	sx := float32(r.OutW) / float32(imax(1, ui.W))
	sy := float32(r.OutH) / float32(imax(1, ui.H))
	for i, q := range rects {
		if i >= 40 {
			break
		}
		rs = append(rs, float32(q[0])*sx, float32(q[1])*sy, float32(q[2])*sx, float32(q[3])*sy)
	}
	glUniformI(r.pComp.L("uNRect"), len(rs)/4)
	if len(rs) > 0 {
		glUniform4Arr(r.pComp.L("uRects[0]"), rs)
	}
	r.drawQuad()
	gl("glBindVertexArray", 0)
	gl("glEnable", GL_DEPTH_TEST)
}

// ---------------------------------------------------------------- Überlagerungen

func (r *Renderer) flatDraw(mode uintptr, pts []float32, col [4]float32) {
	if len(pts) == 0 {
		return
	}
	gl("glBindVertexArray", uintptr(r.flatVAO))
	gl("glBindBuffer", GL_ARRAY_BUFFER, uintptr(r.flatVBO))
	glBufferDataF(GL_ARRAY_BUFFER, pts, GL_DYNAMIC_DRAW)
	glUniformV(r.pFlat.L("uColor"), col[0], col[1], col[2], col[3])
	gl("glDrawArrays", mode, 0, uintptr(len(pts)/3))
}

func (r *Renderer) tileQuads(w *World, tiles [][2]int, lift float64) []float32 {
	var out []float32
	for _, t := range tiles {
		x, y := float64(t[0]), float64(t[1])
		h := func(a, b float64) float32 { return float32(math.Max(0.02, w.HeightAt(a, b)) + lift) }
		e := 0.04
		x0, y0, x1, y1 := x+e, y+e, x+1-e, y+1-e
		out = append(out, float32(x0), h(x0, y0), float32(y0), float32(x0), h(x0, y1), float32(y1), float32(x1), h(x1, y1), float32(y1))
		out = append(out, float32(x0), h(x0, y0), float32(y0), float32(x1), h(x1, y1), float32(y1), float32(x1), h(x1, y0), float32(y0))
	}
	return out
}

func rectTiles(x, y, w, h int) [][2]int {
	var t [][2]int
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			t = append(t, [2]int{xx, yy})
		}
	}
	return t
}

func (r *Renderer) outline(w *World, x, y, bw, bh int, lift float64) []float32 {
	var out []float32
	add := func(ax, ay, bx, by float64) {
		steps := int(math.Max(1, math.Hypot(bx-ax, by-ay)*2))
		for i := 0; i < steps; i++ {
			t0 := float64(i) / float64(steps)
			t1 := float64(i+1) / float64(steps)
			px, py := ax+(bx-ax)*t0, ay+(by-ay)*t0
			qx, qy := ax+(bx-ax)*t1, ay+(by-ay)*t1
			out = append(out, float32(px), float32(math.Max(0.02, w.HeightAt(px, py))+lift), float32(py),
				float32(qx), float32(math.Max(0.02, w.HeightAt(qx, qy))+lift), float32(qy))
		}
	}
	fx, fy, fw, fh := float64(x), float64(y), float64(bw), float64(bh)
	add(fx, fy, fx+fw, fy)
	add(fx+fw, fy, fx+fw, fy+fh)
	add(fx+fw, fy+fh, fx, fy+fh)
	add(fx, fy+fh, fx, fy)
	return out
}

func (r *Renderer) drawOverlays(g *Game, ls sunSet, lvp Mat4) {
	u := g.UI
	w := g.W
	if u.Screen != ScrGame {
		return
	}
	r.drawGameOverlay(g, u)
	gl("glEnable", GL_BLEND)
	gl("glBlendFunc", GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA)
	gl("glDepthMask", 0)
	r.pFlat.Use()
	glUniformMat(r.pFlat.L("uVP"), &u.VP)
	// Blaupausen als Hologramm
	if len(g.BP) > 0 {
		r.setMeshUniforms(r.pMesh, u.VP, lvp, u, ls)
		pulse := float32(0.5 + 0.08*math.Sin(r.time*3))
		glUniformV(r.pMesh.L("uTint"), 0.25, 0.62, 1.0, pulse)
		glUniformV(r.pMesh.L("uAlpha"), 0.55)
		gl("glDepthMask", 1)
		for _, b := range g.BP {
			m := Translate(float64(b.X), b.BaseY, float64(b.Y))
			glUniformMat(r.pMesh.L("uModel"), &m)
			r.model(b.T, b.Var, 0).Draw()
		}
		gl("glDepthMask", 0)
		glUniformV(r.pMesh.L("uTint"), 0, 0, 0, 0)
		glUniformV(r.pMesh.L("uAlpha"), 1)
		r.pFlat.Use()
		glUniformMat(r.pFlat.L("uVP"), &u.VP)
		for _, b := range g.BP {
			bt := &BTypes[b.T]
			col := [4]float32{0.3, 0.75, 1.3, 0.9}
			if b.ID == u.SelBP {
				col = [4]float32{0.8, 1.2, 1.6, 1}
			}
			r.flatDraw(GL_LINES, r.outline(w, b.X, b.Y, bt.W, bt.H, 0.08), col)
		}
	}
	// Auswahl
	if b := g.Bld(u.SelB); b != nil {
		bt := &BTypes[b.T]
		r.flatDraw(GL_LINES, r.outline(w, b.X, b.Y, bt.W, bt.H, 0.08), [4]float32{0.4, 1.2, 1.4, 1})
		r.flatDraw(GL_TRIANGLES, r.tileQuads(w, rectTiles(b.X, b.Y, bt.W, bt.H), 0.07), [4]float32{0.2, 0.8, 1.0, 0.18})
		if bt.Radius > 0 && b.T != BKontor && b.T != BLager {
			cx, cy := b.Center()
			r.flatDraw(GL_LINES, r.circle(w, cx, cy, float64(bt.Radius)), [4]float32{0.4, 1.2, 1.4, 0.8})
		}
	}
	if s := g.ShipByID(u.SelShip); s != nil {
		r.flatDraw(GL_LINES, r.circleY(s.X, s.Y, 1.7, 0.1), [4]float32{0.4, 1.2, 1.4, 1})
		var pts []float32
		px, pz := s.X, s.Y
		for _, p := range s.Path {
			qx, qz := float64(p[0])+0.5, float64(p[1])+0.5
			pts = append(pts, float32(px), 0.12, float32(pz), float32(qx), 0.12, float32(qz))
			px, pz = qx, qz
		}
		r.flatDraw(GL_LINES, pts, [4]float32{0.4, 1.2, 1.4, 0.7})
	}
	if !u.OverUI {
		if u.Tool >= 0 && u.Tool != ToolDemolish {
			// Einflussgrenze
			r.flatDraw(GL_LINES, r.influenceLines(w), [4]float32{0.4, 1.1, 1.3, 0.9})
		}
		switch {
		case u.Tool == ToolDemolish:
			if b := g.BuildingAt(u.HoverX, u.HoverY); b != nil {
				bt := &BTypes[b.T]
				r.flatDraw(GL_TRIANGLES, r.tileQuads(w, rectTiles(b.X, b.Y, bt.W, bt.H), 0.1), [4]float32{1.4, 0.2, 0.2, 0.45})
			} else if w.In(u.HoverX, u.HoverY) && w.Road[w.Idx(u.HoverX, u.HoverY)] {
				r.flatDraw(GL_TRIANGLES, r.tileQuads(w, [][2]int{{u.HoverX, u.HoverY}}, 0.06), [4]float32{1.4, 0.2, 0.2, 0.5})
			}
		case u.Tool == BRoad:
			var ok, bad [][2]int
			for _, p := range u.RoadPreview() {
				if g.CanPlace(BRoad, p[0], p[1]).OK {
					ok = append(ok, p)
				} else {
					bad = append(bad, p)
				}
			}
			r.flatDraw(GL_TRIANGLES, r.tileQuads(w, ok, 0.05), [4]float32{0.3, 1.3, 0.6, 0.55})
			r.flatDraw(GL_TRIANGLES, r.tileQuads(w, bad, 0.05), [4]float32{1.4, 0.2, 0.2, 0.55})
		case u.Tool >= 0:
			bt := &BTypes[u.Tool]
			x, y := u.HoverX-bt.W/2, u.HoverY-bt.H/2
			pc := g.CanPlace(u.Tool, x, y)
			col := [4]float32{0.3, 1.3, 0.6, 0.4}
			if !pc.OK {
				col = [4]float32{1.4, 0.2, 0.2, 0.45}
			}
			r.flatDraw(GL_TRIANGLES, r.tileQuads(w, rectTiles(x, y, bt.W, bt.H), 0.06), col)
			if bt.Radius > 0 {
				cx, cy := float64(x)+float64(bt.W)/2, float64(y)+float64(bt.H)/2
				r.flatDraw(GL_LINES, r.circle(w, cx, cy, float64(bt.Radius)), [4]float32{0.4, 1.2, 1.4, 0.9})
				if u.Tool == BKomm || u.Tool == BMedi || u.Tool == BLabor {
					var hs [][2]int
					for _, h := range g.B {
						if h.T == BHouse {
							hx, hy := h.Center()
							if math.Hypot(hx-cx, hy-cy) <= float64(bt.Radius) {
								hs = append(hs, rectTiles(h.X, h.Y, 2, 2)...)
							}
						}
					}
					r.flatDraw(GL_TRIANGLES, r.tileQuads(w, hs, 0.08), [4]float32{0.3, 1.0, 1.3, 0.3})
				}
			}
			// Geister-Gebäude
			s := 0.0
			for yy := y; yy <= y+bt.H; yy++ {
				for xx := x; xx <= x+bt.W; xx++ {
					s += w.HC(xx, yy)
				}
			}
			by := math.Max(0.14, s/float64((bt.W+1)*(bt.H+1)))
			r.setMeshUniforms(r.pMesh, u.VP, lvp, u, ls)
			m := Translate(float64(x), by, float64(y))
			glUniformMat(r.pMesh.L("uModel"), &m)
			tc := [4]float32{0.2, 1.0, 0.5, 0.35}
			if u.BlueMode && u.Tool != BKontor {
				g.ignoreCost = true
				pc = g.CanPlace(u.Tool, x, y)
				g.ignoreCost = false
				tc = [4]float32{0.25, 0.62, 1.0, 0.55}
			}
			if !pc.OK {
				tc = [4]float32{1.0, 0.15, 0.15, 0.45}
			}
			glUniformV(r.pMesh.L("uTint"), tc[0], tc[1], tc[2], tc[3])
			glUniformV(r.pMesh.L("uAlpha"), 0.75)
			gl("glDepthMask", 1)
			r.model(u.Tool, 0, 0).Draw()
			gl("glDepthMask", 0)
			r.pFlat.Use()
		default:
			if b := g.BuildingAt(u.HoverX, u.HoverY); b != nil && b.ID != u.SelB {
				bt := &BTypes[b.T]
				r.flatDraw(GL_LINES, r.outline(w, b.X, b.Y, bt.W, bt.H, 0.08), [4]float32{1, 1, 1, 0.6})
			}
		}
	}
	gl("glDepthMask", 1)
}

func (r *Renderer) circle(w *World, cx, cy, rad float64) []float32 {
	var out []float32
	n := int(rad * 8)
	for i := 0; i < n; i++ {
		a0 := float64(i) / float64(n) * 2 * math.Pi
		a1 := float64(i+1) / float64(n) * 2 * math.Pi
		x0, y0 := cx+math.Cos(a0)*rad, cy+math.Sin(a0)*rad
		x1, y1 := cx+math.Cos(a1)*rad, cy+math.Sin(a1)*rad
		out = append(out, float32(x0), float32(math.Max(0.03, w.HeightAt(x0, y0))+0.1), float32(y0),
			float32(x1), float32(math.Max(0.03, w.HeightAt(x1, y1))+0.1), float32(y1))
	}
	return out
}

func (r *Renderer) circleY(cx, cy, rad, yy float64) []float32 {
	var out []float32
	n := 32
	for i := 0; i < n; i++ {
		a0 := float64(i) / float64(n) * 2 * math.Pi
		a1 := float64(i+1) / float64(n) * 2 * math.Pi
		out = append(out, float32(cx+math.Cos(a0)*rad), float32(yy), float32(cy+math.Sin(a0)*rad),
			float32(cx+math.Cos(a1)*rad), float32(yy), float32(cy+math.Sin(a1)*rad))
	}
	return out
}

func (r *Renderer) influenceLines(w *World) []float32 {
	var out []float32
	n := w.N
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !w.Infl[w.Idx(x, y)] {
				continue
			}
			for k, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if w.In(nx, ny) && w.Infl[w.Idx(nx, ny)] {
					continue
				}
				var ax, ay, bx, by float64
				fx, fy := float64(x), float64(y)
				switch k {
				case 0:
					ax, ay, bx, by = fx, fy, fx, fy+1
				case 1:
					ax, ay, bx, by = fx+1, fy, fx+1, fy+1
				case 2:
					ax, ay, bx, by = fx, fy, fx+1, fy
				case 3:
					ax, ay, bx, by = fx, fy+1, fx+1, fy+1
				}
				out = append(out, float32(ax), float32(math.Max(0.03, w.HeightAt(ax, ay))+0.06), float32(ay),
					float32(bx), float32(math.Max(0.03, w.HeightAt(bx, by))+0.06), float32(by))
			}
		}
	}
	return out
}

// ---------------------------------------------------------------- Vorschaubilder für die UI

var ThumbProvider func(t, tier int) *Sprite

func Thumb(t, tier int) *Sprite {
	if ThumbProvider != nil {
		if s := ThumbProvider(t, tier); s != nil {
			return s
		}
	}
	return &Sprite{W: 1, H: 1, Pix: []uint32{0}}
}

func (r *Renderer) makeThumbs() {
	const S = 160
	fbo, tex := makeTexFBO(S, S, GL_RGBA16F, GL_RGBA, GL_FLOAT)
	db := glGenRenderbuffer()
	gl("glBindRenderbuffer", GL_RENDERBUFFER, uintptr(db))
	gl("glRenderbufferStorage", GL_RENDERBUFFER, GL_DEPTH_COMPONENT24, S, S)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(fbo))
	gl("glFramebufferRenderbuffer", GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_RENDERBUFFER, uintptr(db))
	ls := r.lighting(nil)
	tmp := &UIState{}
	buf := make([]float32, S*S*4)
	render := func(key int, m *GMesh, w, h, height float64) {
		gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(fbo))
		gl("glViewport", 0, 0, S, S)
		glClearColorBuf(0, 0, 0, 0)
		glClearDepthBuf()
		gl("glEnable", GL_DEPTH_TEST)
		gl("glDisable", GL_BLEND)
		c := V3{w / 2, height * 0.4, h / 2}
		ext := math.Max(math.Max(w, h)*0.85, height*0.75) + 0.3
		dir := V3{1, 0.95, 1}.Norm()
		view := LookAt(c.Add(dir.Mul(30)), c, V3{0, 1, 0})
		vp := Ortho(-ext, ext, -ext, ext, 1, 80).Mul(view)
		tmp.Eye = c.Add(dir.Mul(30))
		far := Translate(10, 10, 10)
		r.setMeshUniforms(r.pMesh, vp, far, tmp, ls)
		glUniformV(r.pMesh.L("uFogD"), 0)
		glUniformV(r.pMesh.L("uAO"), 1)
		bindTex(1, r.shadowTex)
		m.Draw()
		gl("glPixelStorei", GL_PACK_ALIGNMENT, 4)
		gl("glReadPixels", 0, 0, S, S, GL_RGBA, GL_FLOAT, ptr(unsafe.Pointer(&buf[0])))
		runtime.KeepAlive(buf)
		// 2x verkleinern, Tonemapping, vormultipliziert
		o := S / 2
		sp := &Sprite{W: o, H: o, Pix: make([]uint32, o*o)}
		aces := func(x float64) float64 {
			return clampf((x*(2.51*x+0.03))/(x*(2.43*x+0.59)+0.14), 0, 1)
		}
		for y := 0; y < o; y++ {
			for x := 0; x < o; x++ {
				var rr, gg, bb, aa float64
				for dy := 0; dy < 2; dy++ {
					for dx := 0; dx < 2; dx++ {
						sy := S - 1 - (y*2 + dy)
						k := (sy*S + x*2 + dx) * 4
						a := float64(buf[k+3])
						if a > 0 {
							rr += math.Pow(aces(float64(buf[k])), 1/2.2)
							gg += math.Pow(aces(float64(buf[k+1])), 1/2.2)
							bb += math.Pow(aces(float64(buf[k+2])), 1/2.2)
							aa += 1
						}
					}
				}
				if aa == 0 {
					continue
				}
				f := 255.0 / 4
				sp.Pix[y*o+x] = uint32(aa*f)<<24 | uint32(rr*f)<<16 | uint32(gg*f)<<8 | uint32(bb*f)
			}
		}
		r.thumbs[key] = sp
	}
	for t := 0; t < NumBTypes; t++ {
		bt := &BTypes[t]
		tiers := 1
		if t == BHouse {
			tiers = 3
		}
		for tier := 0; tier < tiers; tier++ {
			hgt := 1.5
			switch {
			case t == BHouse && tier == 2:
				hgt = 5
			case t == BHouse && tier == 1:
				hgt = 2.2
			case t == BWind || t == BKomm || t == BKontor || t == BWerft:
				hgt = 3.6
			}
			render(t*4+tier, r.model(t, 1, tier), float64(bt.W), float64(bt.H), hgt)
		}
	}
	render(-4, r.ships[0], 0, 0, 0.8)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, 0)
	delFBO(&fbo)
	delTex(&tex)
	gl("glDeleteRenderbuffers", 1, ptr(unsafe.Pointer(&db)))
}

// Szene in ein Bild lesen (für Screenshots)
func (r *Renderer) ReadPixels() []byte {
	buf := make([]byte, r.OutW*r.OutH*4)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.OutFBO))
	gl("glPixelStorei", GL_PACK_ALIGNMENT, 1)
	gl("glReadPixels", 0, 0, uintptr(r.OutW), uintptr(r.OutH), GL_RGBA, GL_UNSIGNED_BYTE, ptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(buf)
	return buf
}

// Ausgabe-Framebuffer für Offscreen-Rendering
func MakeOutFBO(w, h int) uint32 {
	f, _ := makeTexFBO(w, h, GL_RGBA8, GL_RGBA, GL_UNSIGNED_BYTE)
	return f
}

func (r *Renderer) ssao(u *UIState) {
	// Tiefe auflösen
	gl("glBindFramebuffer", GL_READ_FRAMEBUFFER, uintptr(r.msFBO))
	gl("glBindFramebuffer", GL_DRAW_FRAMEBUFFER, uintptr(r.depthFBO))
	gl("glBlitFramebuffer", 0, 0, uintptr(r.W), uintptr(r.H), 0, 0, uintptr(r.W), uintptr(r.H), GL_DEPTH_BUFFER_BIT, GL_NEAREST)
	hw, hh := imax(1, r.W/2), imax(1, r.H/2)
	gl("glDisable", GL_DEPTH_TEST)
	gl("glDepthMask", 0)
	gl("glDisable", GL_BLEND)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.aoFBO))
	gl("glViewport", 0, 0, uintptr(hw), uintptr(hh))
	r.pSSAO.Use()
	bindTex(0, r.depthTex)
	glUniformI(r.pSSAO.L("uDepth"), 0)
	glUniformMat(r.pSSAO.L("uInvProj"), &u.InvProj)
	glUniformMat(r.pSSAO.L("uProj"), &u.Proj)
	glUniformV(r.pSSAO.L("uTexel"), 1/float32(r.W), 1/float32(r.H))
	glUniformV(r.pSSAO.L("uRadius"), float32(clampf(u.Dist*0.03, 0.45, 2.0)))
	r.drawQuad()
	r.pBlur.Use()
	glUniformI(r.pBlur.L("uTex"), 0)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.ao2FBO))
	bindTex(0, r.aoTex)
	glUniformV(r.pBlur.L("uDir"), 0.7/float32(hw), 0)
	r.drawQuad()
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.aoFBO))
	bindTex(0, r.ao2Tex)
	glUniformV(r.pBlur.L("uDir"), 0, 0.7/float32(hh))
	r.drawQuad()
	// auf die Szene multiplizieren
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.msFBO))
	gl("glViewport", 0, 0, uintptr(r.W), uintptr(r.H))
	gl("glEnable", GL_BLEND)
	gl("glBlendFunc", GL_ZERO, 0x0300) // GL_SRC_COLOR
	r.pAOApply.Use()
	bindTex(0, r.aoTex)
	glUniformI(r.pAOApply.L("uAO"), 0)
	glUniformV(r.pAOApply.L("uStrength"), 0.85)
	r.drawQuad()
	gl("glDisable", GL_BLEND)
	gl("glEnable", GL_DEPTH_TEST)
	gl("glDepthMask", 1)
	bindTex(1, r.shadowTex)
}

// Spiegelbild der Szene für die Wasseroberfläche
func (r *Renderer) reflectionPass(g *Game, ls sunSet, lvp Mat4) {
	u := g.UI
	rw, rh := imax(1, r.W/2), imax(1, r.H/2)
	gl("glBindFramebuffer", GL_FRAMEBUFFER, uintptr(r.reflFBO))
	gl("glViewport", 0, 0, uintptr(rw), uintptr(rh))
	glClearColorBuf(float32(ls.horizon.X), float32(ls.horizon.Y), float32(ls.horizon.Z), 1)
	glClearDepthBuf()
	rvp := u.VP.Mul(ScaleM(1, -1, 1))
	inv := rvp.Inverse()
	gl("glDisable", GL_DEPTH_TEST)
	gl("glDepthMask", 0)
	r.pSky.Use()
	glUniformMat(r.pSky.L("uInvVP"), &inv)
	glUniformV(r.pSky.L("uSun"), v3f(ls.sun)...)
	glUniformV(r.pSky.L("uSky"), v3f(ls.sky)...)
	glUniformV(r.pSky.L("uHorizon"), v3f(ls.horizon)...)
	glUniformV(r.pSky.L("uSunCol"), v3f(ls.sunCol)...)
	glUniformV(r.pSky.L("uRealSun"), v3f(ls.realSun)...)
	glUniformV(r.pSky.L("uNight"), float32(ls.night))
	r.drawQuad()
	gl("glEnable", GL_DEPTH_TEST)
	gl("glDepthMask", 1)
	bindTex(1, r.shadowTex)
	eye := u.Eye
	mu := *u
	mu.Eye = V3{eye.X, -eye.Y, eye.Z}
	r.setMeshUniforms(r.pMesh, rvp, lvp, &mu, ls)
	glUniformI(r.pMesh.L("uClip"), 1)
	r.drawObjects(g, r.pMesh, false)
	glUniformI(r.pMesh.L("uClip"), 0)
}

// ---------------------------------------------------------------- Überlagerungen (Taste O)

var OverlayNames = []string{"Aus", "Einflussbereich", "Energie", "Versorgung der Wohnhäuser", "Brandschutz", "Straßenanbindung"}

func (r *Renderer) footQuad(out *[]float32, w *World, x0, z0, x1, z1 float64, lift float64) {
	y := func(x, z float64) float32 { return float32(math.Max(w.HeightAt(x, z), 0) + lift) }
	*out = append(*out,
		float32(x0), y(x0, z0), float32(z0), float32(x1), y(x1, z0), float32(z0), float32(x1), y(x1, z1), float32(z1),
		float32(x0), y(x0, z0), float32(z0), float32(x1), y(x1, z1), float32(z1), float32(x0), y(x0, z1), float32(z1))
}

func (r *Renderer) drawGameOverlay(g *Game, u *UIState) {
	if u.Overlay <= 0 || u.Overlay >= len(OverlayNames) {
		return
	}
	w := g.W
	groups := map[[4]float32][]float32{}
	add := func(col [4]float32, x0, z0, x1, z1 float64) {
		pts := groups[col]
		r.footQuad(&pts, w, x0, z0, x1, z1, 0.12)
		groups[col] = pts
	}
	bfoot := func(b *Building, col [4]float32) {
		bt := &BTypes[b.T]
		add(col, float64(b.X)+0.05, float64(b.Y)+0.05, float64(b.X+bt.W)-0.05, float64(b.Y+bt.H)-0.05)
	}
	green := [4]float32{0.2, 1.0, 0.45, 0.42}
	red := [4]float32{1.0, 0.25, 0.25, 0.5}
	yellow := [4]float32{1.0, 0.85, 0.2, 0.45}
	blue := [4]float32{0.25, 0.7, 1.0, 0.35}
	switch u.Overlay {
	case 1:
		for y := 0; y < w.N; y++ {
			for x := 0; x < w.N; x++ {
				i := w.Idx(x, y)
				if w.Infl[i] && w.Terr[i] != TWater {
					add(blue, float64(x), float64(y), float64(x+1), float64(y+1))
				}
			}
		}
	case 2:
		for _, b := range g.B {
			bt := &BTypes[b.T]
			switch {
			case bt.Energy > 0:
				bfoot(b, yellow)
			case b.Status == StNoEnergy || (g.Islands[b.Isl].Blackout > 0 && bt.Energy < 0):
				bfoot(b, red)
			case bt.Energy < 0:
				bfoot(b, green)
			}
		}
	case 3:
		for _, b := range g.B {
			if b.T != BHouse || len(b.Fulfil) == 0 {
				continue
			}
			s := 0.0
			for _, f := range b.Fulfil {
				s += f
			}
			a := s / float64(len(b.Fulfil))
			col := [4]float32{float32(1 - a), float32(0.25 + 0.75*a), 0.25, 0.5}
			bfoot(b, col)
		}
	case 4:
		for _, f := range g.B {
			if f.T != BFeuer {
				continue
			}
			fx, fy := f.Center()
			rad := float64(BTypes[BFeuer].Radius)
			for y := int(fy - rad); y <= int(fy+rad); y++ {
				for x := int(fx - rad); x <= int(fx+rad); x++ {
					if w.In(x, y) && math.Hypot(float64(x)+0.5-fx, float64(y)+0.5-fy) <= rad && w.Terr[w.Idx(x, y)] != TWater {
						add([4]float32{1, 0.55, 0.15, 0.16}, float64(x), float64(y), float64(x+1), float64(y+1))
					}
				}
			}
		}
		for _, b := range g.B {
			if b.T == BRoad || b.T == BPark {
				continue
			}
			if b.Fire > 0 || !g.fireCovered(b) {
				bfoot(b, red)
			}
		}
	case 5:
		for _, b := range g.B {
			if BTypes[b.T].Out < 0 && b.T != BHouse {
				continue
			}
			if b.Conn {
				bfoot(b, green)
			} else {
				bfoot(b, red)
			}
		}
	}
	gl("glEnable", GL_BLEND)
	gl("glBlendFunc", GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA)
	gl("glDepthMask", 0)
	r.pFlat.Use()
	glUniformMat(r.pFlat.L("uVP"), &u.VP)
	for col, pts := range groups {
		r.flatDraw(GL_TRIANGLES, pts, col)
	}
	gl("glDepthMask", 1)
}

// Vereinfachtes Modell für weit entfernte Gebäude: Hüllquader mit mittlerer Farbe
func (r *Renderer) lodModel(t, v, tier int) *GMesh {
	key := Tf("lod_%d_%d_%d", t, v%4, tier)
	if m, ok := r.models[key]; ok {
		return m
	}
	src := BuildModel(t, v%4, tier)
	mn, mx := V3{1e9, 1e9, 1e9}, V3{-1e9, -1e9, -1e9}
	var cr, cg, cb float64
	n := 0
	for i := 0; i+9 < len(src.V); i += 10 {
		p := V3{float64(src.V[i]), float64(src.V[i+1]), float64(src.V[i+2])}
		if p.Y < 0.06 {
			continue
		}
		mn = V3{math.Min(mn.X, p.X), math.Min(mn.Y, p.Y), math.Min(mn.Z, p.Z)}
		mx = V3{math.Max(mx.X, p.X), math.Max(mx.Y, p.Y), math.Max(mx.Z, p.Z)}
		cr += float64(src.V[i+6])
		cg += float64(src.V[i+7])
		cb += float64(src.V[i+8])
		n++
	}
	b := NewMB()
	bt := &BTypes[t]
	plinth(b, float64(bt.W), float64(bt.H))
	if n > 0 {
		lin := func(x float64) uint8 { return uint8(math.Min(255, math.Pow(x/float64(n), 1/2.2)*255)) }
		col := RGB{lin(cr), lin(cg), lin(cb)}
		// zwei Stufen für eine grobe Silhouette
		h := mx.Y
		b.BoxC(mn.X+0.05, 0.05, mn.Z+0.05, mx.X-0.05, h*0.6, mx.Z-0.05, col, col.Mul(0.8), MatPanel, MatPanel)
		b.BoxC(mn.X+(mx.X-mn.X)*0.2, h*0.6, mn.Z+(mx.Z-mn.Z)*0.2, mx.X-(mx.X-mn.X)*0.2, h, mx.Z-(mx.Z-mn.Z)*0.2, col, col.Mul(0.8), MatPanel, MatPanel)
		if t == BHouse {
			b.Box(mn.X+0.04, h*0.15, mn.Z+0.04, mx.X-0.04, h*0.55, mn.Z+0.06, hex(0x2a3650), MatGlass)
		}
	}
	m := b.Upload()
	r.models[key] = m
	return m
}
