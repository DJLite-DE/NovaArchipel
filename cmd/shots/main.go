package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"time"

	g "novaarchipel/game"
)

var R *g.Renderer
var W, H = 1600, 900

func save(name string) {
	buf := R.ReadPixels()
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		copy(img.Pix[y*W*4:(y+1)*W*4], buf[(H-1-y)*W*4:(H-y)*W*4])
	}
	f, _ := os.Create(name)
	png.Encode(f, img)
	f.Close()
	fmt.Println("saved", name)
}

func frames(gm *g.Game, c *g.Canvas, n int) {
	for i := 0; i < n; i++ {
		gm.Frame(c, 0.05)
		R.Render(gm, c, gm.UI.Panels(), true, 0.05)
		g.In.EndFrame()
	}
}

func main() {
	if err := g.InitHeadlessGL(); err != nil {
		panic(err)
	}
	var err error
	t0 := time.Now()
	R, err = g.NewRenderer()
	if err != nil {
		panic(err)
	}
	fmt.Println("renderer", R.Info, time.Since(t0))
	R.Resize(W, H)
	R.OutFBO = g.MakeOutFBO(W, H)
	c := g.NewCanvas(W, H)
	which := "all"
	if len(os.Args) > 1 {
		which = os.Args[1]
	}
	out := "shots/"
	os.MkdirAll(out, 0755)
	if which == "pick" {
		gm := g.NewGame(7)
		frames(gm, c, 1)
		// gültige Kontor-Position suchen
		sh := gm.Ships[0]
		var kx, ky int
		found := false
		for r := 0; r < 14 && !found; r++ {
			for dy := -r; dy <= r && !found; dy++ {
				for dx := -r; dx <= r && !found; dx++ {
					x, y := int(sh.X)+dx, int(sh.Y)+dy
					if gm.CanPlace(g.BKontor, x, y).OK {
						kx, ky, found = x, y, true
					}
				}
			}
		}
		fmt.Println("spot", kx, ky, found)
		gm.UI.Tool = g.BKontor
		sx, sy := gm.UI.WorldToScreen(float64(kx)+1.5, float64(ky)+1.5, float64(W), float64(H))
		g.In.MX, g.In.MY = int(sx), int(sy)
		frames(gm, c, 1)
		wx, wy := gm.UI.ScreenToWorld(sx, sy, float64(W), float64(H))
		fmt.Println("hover", gm.UI.HoverX, gm.UI.HoverY, "back", wx, wy)
		save(out + "pick_preview.png")
		g.In.MouseButton(0, true)
		frames(gm, c, 1)
		g.In.MouseButton(0, false)
		frames(gm, c, 2)
		fmt.Println("kontore", len(gm.B), gm.Islands[0].Owned)
		save(out + "pick_after.png")
		return
	}
	if which == "all" || which == "day" {
		g.Opt.DayNight = false
		gm := g.BuildDemo(7, 3)
		kx, ky := gm.CityCenter()
		set := func(hour float64, storm float64, dist, yaw float64, name string) {
			g.Opt.FixedHour = hour
			gm.Storm, gm.StormOn = storm, storm > 0.5
			gm.Flash = 0
			gm.UI.CenterOn(gm, kx+3, ky+2)
			gm.UI.Dist, gm.UI.Yaw = dist, yaw
			g.In.MX, g.In.MY = 5, 450
			frames(gm, c, 2)
			gm.Storm = storm
			frames(gm, c, 1)
			save(out + name)
		}
		set(10.5, 0, 34, math.Pi/4+0.3, "10_vormittag.png")
		set(19.35, 0, 30, math.Pi/4-0.4, "11_abendrot.png")
		set(22.0, 0, 30, math.Pi/4+0.2, "12_nacht.png")
		set(15.0, 1, 30, math.Pi/4, "13_sturm.png")
		g.Opt.DayNight = true
	}
	if which == "all" || which == "camp" {
		gm := g.NewCampaign()
		g.In.MX, g.In.MY = 800, 300
		frames(gm, c, 2)
		save(out + "14_kampagne.png")
	}
	if which == "all" || which == "opt" {
		app := g.NewApp(R)
		app.OpenOptions(0)
		g.In.MX, g.In.MY = 900, 600
		for i := 0; i < 2; i++ {
			app.Frame(c, 0.05)
			app.Render(c, 0.05)
			g.In.EndFrame()
		}
		save(out + "15_optionen.png")
	}
	if which == "all" || which == "menu" {
		app := g.NewApp(R)
		g.In.MX, g.In.MY = 200, 470
		for i := 0; i < 2; i++ {
			app.Frame(c, 0.05)
			app.Render(c, 0.05)
			g.In.EndFrame()
		}
		save(out + "01_hauptmenue.png")
	}
	if which == "all" || which == "start" {
		gm := g.NewGame(7)
		g.In.MX, g.In.MY = 800, 300
		frames(gm, c, 2)
		save(out + "02_spielstart.png")
	}
	if which == "all" || which == "city" {
		gm := g.BuildDemo(7, 1)
		gm.UI.Dist = 26
		gm.UI.Cat = 1
		g.In.MX, g.In.MY = W/2-40, H-150
		frames(gm, c, 2)
		save(out + "03_stadt_baumenue.png")
	}
	if which == "all" || which == "tech" {
		gm := g.BuildDemo(7, 2)
		for _, b := range gm.B {
			if b.T == g.BHouse && b.Tier == 1 {
				gm.UI.SelB = b.ID
				x, y := b.Center()
				gm.UI.CenterOn(gm, x+2, y)
				break
			}
		}
		gm.UI.Dist = 18
		g.In.MX, g.In.MY = 700, 500
		frames(gm, c, 2)
		save(out + "04_techniker_bedarf.png")
		gm.UI.SelB = 0
		gm.UI.Dist = 80
		gm.UI.SelShip = gm.Ships[len(gm.Ships)-1].ID
		s := gm.Ships[len(gm.Ships)-1]
		gm.UI.CenterOn(gm, s.X-8, s.Y+4)
		frames(gm, c, 2)
		save(out + "06_handelsroute_karte.png")
	}
	if which == "all" || which == "eng" {
		gm := g.BuildDemo(7, 3)
		for _, b := range gm.B {
			if b.T == g.BHouse && b.Tier == 2 {
				x, y := b.Center()
				gm.UI.CenterOn(gm, x, y)
				break
			}
		}
		gm.UI.Dist = 22
		gm.UI.Yaw = math.Pi/4 + 0.5
		g.In.MX, g.In.MY = 10, 400
		frames(gm, c, 2)
		save(out + "08_ingenieure.png")
		gm.UI.Dist = 12
		gm.UI.Yaw = math.Pi/4 - 0.7
		frames(gm, c, 2)
		save(out + "05_nahaufnahme.png")
		for _, b := range gm.B {
			if b.T == g.BHouse && b.Tier == 0 {
				x, y := b.Center()
				gm.UI.CenterOn(gm, x, y)
				break
			}
		}
		gm.UI.Dist = 9
		gm.UI.Yaw = math.Pi/4 + 0.4
		frames(gm, c, 40)
		save(out + "16_strassenleben.png")
	}
	if which == "all" || which == "rival" {
		gm := g.BuildDemo(7, 3)
		gm.InitRivals(3)
		gm.FastForwardRivals(2600)
		gm.Dialogs = nil
		rv := gm.Rivals[0]
		ri := rv.Isles[0]
		gm.UI.CenterOn(gm, float64(ri.KX)+4, float64(ri.KY)+4)
		gm.UI.Dist = 26
		gm.UI.Yaw = math.Pi/4 + 2.6
		g.In.MX, g.In.MY = 10, 400
		frames(gm, c, 3)
		save(out + "17_konkurrenz.png")
		gm.UI.TX, gm.UI.TZ = 90, 90
		gm.UI.Dist = 95
		gm.UI.Yaw = math.Pi / 4
		frames(gm, c, 3)
		save(out + "18_archipel_fraktionen.png")
		gm.Rivals[1].Req = &g.RivalReq{Good: g.GStahl, Amt: 30, Reward: 3200, T: 214}
		gm.Rivals[1].Relation = 34
		gm.Rivals[1].Treaty = g.TreatyTrade
		gm.UI.ShowDiplo = true
		g.In.MX, g.In.MY = 700, 760
		frames(gm, c, 2)
		save(out + "19_diplomatie.png")
	}
	if which == "all" || which == "bau" {
		gm := g.BuildDemo(7, 2)
		kx, ky := gm.CityCenter()
		// ein paar Baustellen in verschiedenen Stadien
		n := 0
		for _, b := range gm.B {
			if b.T != g.BKontor && b.T != g.BRoad && math.Hypot(float64(b.X)-kx, float64(b.Y)-ky) < 9 && n < 6 {
				b.Constr = 0.15 + float64(n)*0.15
				n++
			}
		}
		gm.UI.CenterOn(gm, kx, ky)
		gm.UI.Dist = 14
		gm.UI.Yaw = math.Pi/4 + 0.3
		g.In.MX, g.In.MY = 10, 400
		gm.Speed = 0
		frames(gm, c, 30)
		save(out + "21_baustellen.png")
	}
	if which == "all" || which == "kampf" {
		gm := g.BuildDemo(7, 3)
		gm.InitRivals(2)
		gm.Dialogs = nil
		var werft *g.Building
		for _, b := range gm.B {
			if b.T == g.BWerft {
				werft = b
			}
		}
		gm.SpawnTestBattle(werft)
		gm.Hour = 18.6
		for i := 0; i < 12; i++ {
			gm.StepForShots(0.1)
		}
		gm.UI.Dist = 15
		gm.UI.Yaw = math.Pi/4 + 0.9
		g.In.MX, g.In.MY = 10, 400
		gm.Speed = 1
		for i := 0; i < 14; i++ {
			gm.StepForShots(0.1)
			frames(gm, c, 1)
		}
		save(out + "22_seegefecht.png")
	}
	if which == "all" || which == "plan" {
		gm := g.BuildDemo(7, 2)
		gm.UI.Screen = g.ScrGame
		gm.SimulateForShots(1800)
		gm.Notifs = nil
		kx, ky := gm.CityCenter()
		n := 0
		types := []int{g.BHouse, g.BHouse, g.BFeuer, g.BHouse, g.BTurret, g.BHouse, g.BKomm, g.BHouse}
		for r := 3; r < 22 && n < len(types); r++ {
			for y := int(ky) - r; y <= int(ky)+r && n < len(types); y++ {
				for x := int(kx) - r; x <= int(kx)+r && n < len(types); x++ {
					if gm.PlaceBlueprint(types[n], x, y).OK {
						n++
					}
				}
			}
		}
		fmt.Println("blueprints", n)
		if len(gm.BP) > 0 {
			bx, by := gm.BP[0].Center()
			gm.UI.CenterOn(gm, bx, by)
			gm.UI.SelBP = gm.BP[0].ID
		}
		gm.UI.Dist = 16
		gm.UI.Yaw = math.Pi/4 + 0.2
		gm.UI.BlueMode = true
		gm.Speed = 1
		g.In.MX, g.In.MY = 10, 400
		frames(gm, c, 3)
		save(out + "23_blaupausen.png")
		gm.UI.SelBP = 0
		gm.UI.ShowStats = true
		frames(gm, c, 2)
		save(out + "24_statistik_waren.png")
		gm.UI.StatTab = 1
		frames(gm, c, 2)
		save(out + "25_statistik_entwicklung.png")
		gm.UI.ShowStats = false
		gm.UI.BlueMode = false
		gm.UI.Cat = g.CatInfra
		g.In.MX, g.In.MY = 10, 400
		frames(gm, c, 2)
		save(out + "26_baumenue_infra.png")
	}
	if which == "all" || which == "mond" {
		gm := g.BuildDemo(7, 3)
		gm.UI.Screen = g.ScrGame
		gm.BuildMoonDemo()
		gm.Dialogs = nil
		gm.Notifs = nil
		gm.UI.WinShown = true
		gm.Won = true
		// Raketenstart auf der Erde
		for _, b := range gm.B {
			if b.T == g.BRaumhafen {
				x, y := b.Center()
				gm.UI.CenterOn(gm, x, y)
			}
		}
		gm.UI.Dist = 22
		gm.UI.Yaw = math.Pi/4 + 0.5
		gm.Launch = 1.4
		gm.Hour = 17.5
		g.In.MX, g.In.MY = 10, 400
		frames(gm, c, 12)
		save(out + "27_raketenstart.png")
		gm.SwitchRegion()
		gm.UI.WinShown = true
		gm.Won = true
		gm.Notifs = nil
		gm.Hour = 11
		gm.UI.Dist = 30
		gm.UI.Yaw = math.Pi/4 + 2.2
		frames(gm, c, 3)
		save(out + "28_mondbasis.png")
		gm.UI.Dist = 17
		gm.UI.Yaw = math.Pi/4 + 1.0
		frames(gm, c, 3)
		save(out + "29_mondbasis_nah.png")
	}
	if which == "all" || which == "upd" {
		app := g.NewApp(R)
		g.Upd.State, g.Upd.Tag, g.Upd.Notes, g.Upd.Size, g.Upd.URL = g.UpdAvailable, "v2.4", "Neu: Seegefechte, Mondbasis und Blaupausen-Modus", 3200000, "x"
		g.In.MX, g.In.MY = 10, 470
		for i := 0; i < 2; i++ {
			app.Frame(c, 0.05)
			app.Render(c, 0.05)
			g.In.EndFrame()
		}
		save(out + "20_update.png")
		g.Upd.State = g.UpdIdle
	}
	if which == "all" || which == "en" {
		g.Opt.Lang = "en"
		app := g.NewApp(R)
		g.Main = app
		step := func(n int) {
			for i := 0; i < n; i++ {
				app.Frame(c, 0.05)
				app.Render(c, 0.05)
				g.In.EndFrame()
			}
		}
		g.In.MX, g.In.MY = 10, 470
		step(2)
		save(out + "40_en_menu.png")
		gm := g.BuildDemo(7, 3)
		gm.UI.Screen = g.ScrGame
		gm.Notifs = nil
		gm.Notify("Feuer: Stahlwerk auf Neu-Aurora brennt!", g.RGB{255, 120, 60})
		app.Cur = gm
		app.InMenu = false
		for _, b := range gm.B {
			if b.T == g.BHouse {
				gm.UI.SelB = b.ID
				x, y := b.Center()
				gm.UI.CenterOn(gm, x, y)
				break
			}
		}
		gm.UI.Cat = g.CatT1
		step(3)
		save(out + "41_en_game.png")
		g.Opt.Lang = "de"
	}
	if which == "all" || which == "neu" {
		app := g.NewApp(R)
		g.Main = app
		step := func(n int) {
			for i := 0; i < n; i++ {
				app.Frame(c, 0.05)
				app.Render(c, 0.05)
				g.In.EndFrame()
			}
		}
		g.In.MX, g.In.MY = 10, 470
		step(2)
		save(out + "30_hauptmenue.png")
		app.Setup = g.SetupState{Open: true, Seed: 4711, Diff: 1, Size: 1, Islands: 9, Rivals: 2}
		step(2)
		save(out + "31_spiel_einrichten.png")
		app.Setup.Open = false
		app.Cur = g.NewTutorial()
		app.InMenu = false
		step(3)
		save(out + "32_tutorial.png")
		gm := g.BuildDemo(7, 3)
		gm.UI.Screen = g.ScrGame
		gm.SimulateForShots(600)
		gm.Notifs = nil
		gm.RP = 64
		gm.Tech[0], gm.Tech[1], gm.Tech[2] = true, true, true
		app.Cur = gm
		kx, ky := gm.CityCenter()
		gm.UI.CenterOn(gm, kx, ky)
		gm.UI.Dist = 30
		gm.UI.ShowResearch = true
		step(2)
		save(out + "33_forschung.png")
		gm.UI.ShowResearch = false
		gm.TriggerEventForShots(0)
		step(2)
		save(out + "34_ereignis.png")
		gm.Event0 = nil
		gm.UI.Overlay = 3
		step(2)
		save(out + "35_ueberlagerung.png")
		gm.UI.Overlay = 0
		if err := gm.SaveSlot("test", "Testspiel"); err != nil {
			fmt.Println("save err", err)
		}
		step(2)
		app.Slots.Show(false)
		step(2)
		save(out + "36_spielstaende.png")
		app.Slots.Open = false
		gm.UI.ShowAch = true
		step(2)
		save(out + "37_erfolge.png")
		gm.UI.ShowAch = false
		gm.UI.ShowOptions = true
		gm.UI.OptTabForShots(3)
		step(2)
		save(out + "38_tasten.png")
	}
}

// Interaktionstest: Kontor per Mausklick setzen
func init() {
	if len(os.Args) > 1 && os.Args[1] == "pick" {
		pickTest = true
	}
}

var pickTest bool
