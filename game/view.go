package game

import "math"

// 2D-Hinweise über der 3D-Welt (Statussymbole, Inselnamen, Bau-Hinweis)
func (g *Game) DrawWorldOverlay(c *Canvas) {
	u := g.UI
	sw, sh := float64(c.W), float64(c.H)
	// Statussymbole
	for _, b := range g.B {
		bt := &BTypes[b.T]
		if !(b.Status == StNoRoad || b.Status == StNoInput || b.Status == StNoEnergy || (b.T == BHouse && !b.Conn)) {
			continue
		}
		cx, cy := b.Center()
		hgt := 1.6
		if b.T == BHouse {
			hgt = []float64{1.4, 2.4, 5.5}[b.Tier]
		}
		x, y := u.Project(V3{cx, b.BaseY + hgt, cy}, sw, sh)
		if x < -20 || y < -20 || x > sw+20 || y > sh+20 {
			continue
		}
		_ = bt
		pulse := 0.85 + 0.15*math.Sin(u.Anim*5)
		r := 9 * pulse
		col := hex(0xff4a5a)
		if b.Status == StNoEnergy {
			col = hex(0xffc24a)
		}
		c.Circle(x, y, r+2, RGB{10, 14, 22}, 200)
		c.Circle(x, y, r, col, 255)
		c.FillRect(int(x)-1, int(y-r*0.55), 2, int(r*0.7), RGB{255, 255, 255}, 255)
		c.FillRect(int(x)-1, int(y+r*0.3), 2, 2, RGB{255, 255, 255}, 255)
	}
	// Brände
	for _, b := range g.B {
		if b.Fire <= 0 {
			continue
		}
		cx, cy := b.Center()
		x, y := u.Project(V3{cx, b.BaseY + 2.2, cy}, sw, sh)
		if x < -20 || y < -20 || x > sw+20 || y > sh+20 {
			continue
		}
		pulse := 0.8 + 0.2*math.Sin(u.Anim*8)
		c.Circle(x, y, 13, RGB{10, 14, 22}, 210)
		c.Circle(x, y, 11*pulse, hex(0xff6a20), 255)
		c.Circle(x, y+2, 6*pulse, hex(0xffd040), 255)
		if !g.fireCovered(b) {
			k := clampf(1-b.Fire/55, 0, 1)
			c.FillRect(int(x)-18, int(y)+16, 36, 4, RGB{10, 14, 22}, 220)
			c.FillRect(int(x)-18, int(y)+16, int(36*k), 4, hex(0xff6a20), 255)
		}
	}
	// Lebensbalken von Schiffen im Kampf
	if u.Dist < 60 {
		for _, s := range g.Ships {
			if !(s.War || s.HP < s.MaxHP) || s.MaxHP == 0 {
				continue
			}
			x, y := u.Project(V3{s.X, 1.6, s.Y}, sw, sh)
			if x < -30 || y < -30 || x > sw+30 || y > sh+30 {
				continue
			}
			f := clampf(s.HP/s.MaxHP, 0, 1)
			bw := 44
			col := FacCol(shipFac(s))
			c.RoundRect(int(x)-bw/2-2, int(y)-2, bw+4, 9, 3, RGB{8, 12, 20}, 220)
			hc := cGreen
			if f < 0.35 {
				hc = cRed
			} else if f < 0.65 {
				hc = cGold
			}
			c.FillRect(int(x)-bw/2, int(y), int(float64(bw)*f), 5, hc, 255)
			c.FillRect(int(x)-bw/2-2, int(y)-2, 3, 9, col, 255)
			if s.Kind == KindPirate || g.facHostile(0, shipFac(s)) {
				c.TextC(FSmall, int(x), int(y)-20, s.Name, col)
			}
		}
	}
	// Inselnamen bei weiter Sicht
	if u.Dist > 38 {
		a := int(clampf((u.Dist-38)/15, 0, 1) * 255)
		for _, I := range g.Islands {
			x, y := u.Project(V3{float64(I.CX), 2.5, float64(I.CY)}, sw, sh)
			if x < -100 || x > sw+100 || y < -50 || y > sh+50 {
				continue
			}
			col := RGB{240, 246, 255}
			if I.Owned {
				col = cAccent
			}
			if I.Rival > 0 {
				col = RivalDefs[I.Rival-1].Col
				sub := RivalDefs[I.Rival-1].Name
				sw2 := TextW(FBold, sub)
				c.TextA(FBold, int(x)-sw2/2+1, int(y)+37, sub, RGB{0, 0, 0}, a/2)
				c.TextA(FBold, int(x)-sw2/2, int(y)+36, sub, col.Mix(RGB{255, 255, 255}, 0.3), a)
			}
			tw := TextW(FBig, I.Name)
			c.TextA(FBig, int(x)-tw/2+1, int(y)+2, I.Name, RGB{0, 0, 0}, a/2)
			c.TextA(FBig, int(x)-tw/2, int(y), I.Name, col, a)
		}
	}
	// Bau-Hinweis
	if u.Screen == ScrGame && !u.OverUI && u.Tool >= 0 && u.Tool != ToolDemolish && u.Tool != BRoad {
		bt := &BTypes[u.Tool]
		pc := g.CanPlace(u.Tool, u.HoverX-bt.W/2, u.HoverY-bt.H/2)
		if !pc.OK {
			mx, my := u.MX+20, u.MY+20
			tw := TextW(FNormal, pc.Reason) + 24
			c.RoundRect(mx, my, tw, 28, 8, RGB{60, 12, 20}, 225)
			c.Text(FNormal, mx+12, my+5, pc.Reason, RGB{255, 215, 215})
		}
	}
}
