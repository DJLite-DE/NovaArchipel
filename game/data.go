package game

// ---------------------------------------------------------------- Waren

const (
	GSchrott = iota
	GBaustoffe
	GAlgen
	GNahrung
	GWasser
	GErz
	GStahl
	GKaffee
	GSynthkaffee
	GFasern
	GTextil
	GSilizium
	GChips
	GHolo
	GHelium3
	NumGoods
)

type GoodInfo struct {
	Name  string
	Col   RGB
	Price int // Basispreis beim Händler
}

var Goods = [NumGoods]GoodInfo{
	{"Schrott", hex(0x8a8f98), 8},
	{"Baustoffe", hex(0xd9dee6), 30},
	{"Algen", hex(0x3fcf8e), 10},
	{"Nahrung", hex(0xf2b84b), 30},
	{"Reinwasser", hex(0x4fc3ff), 20},
	{"Erz", hex(0xc0703a), 20},
	{"Stahl", hex(0x9fb3c8), 60},
	{"Kaffeebohnen", hex(0x7a4a2a), 25},
	{"Synthkaffee", hex(0xb5651d), 70},
	{"Biofasern", hex(0xc8e06b), 20},
	{"Smart-Textil", hex(0xc061ff), 80},
	{"Silizium", hex(0x7fe3e0), 40},
	{"Mikrochips", hex(0x46f0a0), 140},
	{"Holo-Pads", hex(0x5aa8ff), 260},
	{"Helium-3", hex(0xe8f4ff), 320},
}

// Baumaterialien: Index in Materialkosten
var MatGoods = [3]int{GBaustoffe, GStahl, GChips}

// ---------------------------------------------------------------- Bevölkerung

const (
	TArbeiter = iota
	TTechniker
	TIngenieure
	NumTiers
)

type Need struct {
	Good    int // -1 = Dienstleistung
	Service int // Gebäudetyp des Dienstes
	Rate    float64
}

type TierInfo struct {
	Name, Plural string
	MaxRes       int
	Tax          float64 // Credits pro Einwohner und Minute
	Col          RGB
	Needs        []Need
	UpCost       [3]int // Materialkosten für Aufstieg in diese Stufe
	UnlockPop    int    // benötigte Einwohner der vorigen Stufe
}

var Tiers [NumTiers]TierInfo

// ---------------------------------------------------------------- Gebäude

const (
	BRoad = iota
	BKontor
	BLager
	BWerft
	BHouse
	BPark
	BKomm
	BMedi
	BLabor
	BWind
	BSolar
	BFusion
	BSchrott
	BRecycle
	BAlgen
	BNahrung
	BWasser
	BErz
	BStahl
	BKaffeePl
	BRoest
	BFaserPl
	BTextil
	BSilMine
	BChip
	BHolo
	BTurret
	BFeuer
	BRaumhafen
	BHe3
	NumBTypes
)

const (
	CatInfra = iota
	CatPublic
	CatT1
	CatT2
	CatT3
	CatEnergy
	NumCats
)

var CatNames = [NumCats]string{"Infrastruktur", "Wohnen & Öffentlich", "Arbeiter-Produktion", "Techniker-Produktion", "Ingenieurs-Produktion", "Energie"}

const (
	DepNone = iota
	DepErz
	DepSil
	DepHe3
)
const (
	FertNone = iota
	FertKaffee
	FertFasern
)

type BType struct {
	Name    string
	Cat     int
	W, H    int
	Cost    int
	Mat     [3]int
	Upkeep  int
	Energy  int // + Erzeugung, - Verbrauch
	In      []int
	Out     int
	Cycle   float64 // Sekunden pro Einheit
	Tier    int     // Stufe, deren Bevölkerung zählt
	Unlock  int     // benötigte Einwohnerzahl dieser Stufe
	Radius  int
	Coastal bool
	Deposit int
	Fert    int
	Desc    string
	Col     RGB // Minimap-Farbe
}

var BTypes [NumBTypes]BType

func init() {
	none := []int(nil)
	B := func(id int, b BType) { BTypes[id] = b }
	B(BRoad, BType{Name: "Magnetstraße", Cat: CatInfra, W: 1, H: 1, Cost: 5, Out: -1, Desc: "Verbindet Gebäude mit Kontor und Lagerhäusern. Gleiter transportieren darauf die Waren.", Col: hex(0x6b7380)})
	B(BKontor, BType{Name: "Kontor", Cat: CatInfra, W: 3, H: 3, Cost: 800, Mat: [3]int{8, 0, 0}, Out: -1, Radius: 15, Coastal: true, Desc: "Herz jeder Kolonie. Muss an der Küste stehen. Gründet die Kolonie auf einer Insel – auf neuen Inseln mit Material aus einem nahen Frachtgleiter.", Col: hex(0x30d0ff)})
	B(BLager, BType{Name: "Lagerhaus", Cat: CatInfra, W: 2, H: 2, Cost: 400, Mat: [3]int{4, 0, 0}, Upkeep: 10, Out: -1, Radius: 10, Desc: "Erweitert den Einflussbereich und erhöht die Lagerkapazität der Insel um 30 je Ware.", Col: hex(0x30a0d0)})
	B(BWerft, BType{Name: "Werft", Cat: CatInfra, W: 3, H: 3, Cost: 1500, Mat: [3]int{12, 0, 0}, Upkeep: 40, Energy: -4, Out: -1, Tier: TArbeiter, Unlock: 30, Coastal: true, Desc: "Baut Frachtgleiter für den Handel zwischen den Inseln.", Col: hex(0x5080a0)})
	B(BHouse, BType{Name: "Wohnmodul", Cat: CatPublic, W: 2, H: 2, Cost: 50, Mat: [3]int{2, 0, 0}, Out: -1, Desc: "Heim deiner Bürger. Sind alle Bedürfnisse erfüllt, steigen die Bewohner zur nächsten Stufe auf.", Col: hex(0xe8eef5)})
	B(BPark, BType{Name: "Grünanlage", Cat: CatPublic, W: 1, H: 1, Cost: 30, Out: -1, Desc: "Hängende Gärten und Wasserspiele – rein zur Verschönerung deiner Stadt.", Col: hex(0x40c060)})
	B(BKomm, BType{Name: "Komm-Zentrum", Cat: CatPublic, W: 3, H: 3, Cost: 1000, Mat: [3]int{10, 0, 0}, Upkeep: 30, Energy: -2, Out: -1, Radius: 12, Desc: "Netzwerk-Knoten und Treffpunkt. Dienstleistung für Arbeiter, Techniker und Ingenieure.", Col: hex(0x3fe0ff)})
	B(BMedi, BType{Name: "Medizentrum", Cat: CatPublic, W: 3, H: 3, Cost: 2500, Mat: [3]int{12, 4, 0}, Upkeep: 70, Energy: -4, Out: -1, Radius: 14, Tier: TTechniker, Unlock: 1, Desc: "Gesundheitsversorgung – Bedürfnis der Techniker und Ingenieure.", Col: hex(0xff6070)})
	B(BLabor, BType{Name: "Forschungslabor", Cat: CatPublic, W: 3, H: 3, Cost: 6000, Mat: [3]int{15, 8, 4}, Upkeep: 140, Energy: -8, Out: -1, Radius: 16, Tier: TIngenieure, Unlock: 1, Desc: "Innovationszentrum – Bedürfnis der Ingenieure.", Col: hex(0xa070ff)})
	B(BWind, BType{Name: "Windturbine", Cat: CatEnergy, W: 1, H: 1, Cost: 200, Mat: [3]int{2, 0, 0}, Upkeep: 5, Energy: 6, Out: -1, Desc: "Liefert 6 Energie für die Insel.", Col: hex(0xf0f0f0)})
	B(BSolar, BType{Name: "Solarpark", Cat: CatEnergy, W: 3, H: 3, Cost: 1800, Mat: [3]int{8, 4, 0}, Upkeep: 25, Energy: 28, Out: -1, Tier: TTechniker, Unlock: 1, Desc: "Liefert 28 Energie für die Insel.", Col: hex(0x2050a0)})
	B(BFusion, BType{Name: "Fusionsreaktor", Cat: CatEnergy, W: 4, H: 4, Cost: 12000, Mat: [3]int{20, 15, 8}, Upkeep: 180, Energy: 140, Out: -1, Tier: TIngenieure, Unlock: 1, Desc: "Liefert 140 Energie für die Insel, mit Helium-3 vom Mond sogar 60 % mehr.", Col: hex(0xffa040)})
	B(BSchrott, BType{Name: "Schrottsammler", Cat: CatT1, W: 2, H: 2, Cost: 100, Mat: [3]int{2, 0, 0}, Upkeep: 5, In: none, Out: GSchrott, Cycle: 20, Desc: "Sammelt Altmaterial aus der Umgebung.", Col: hex(0x8a8f98)})
	B(BRecycle, BType{Name: "Recyclingwerk", Cat: CatT1, W: 2, H: 2, Cost: 300, Mat: [3]int{3, 0, 0}, Upkeep: 15, Energy: -2, In: []int{GSchrott}, Out: GBaustoffe, Cycle: 20, Desc: "Verarbeitet Schrott zu Baustoffen.", Col: hex(0xb0b8c0)})
	B(BAlgen, BType{Name: "Algenfarm", Cat: CatT1, W: 3, H: 2, Cost: 300, Mat: [3]int{3, 0, 0}, Upkeep: 10, Energy: -1, In: none, Out: GAlgen, Cycle: 30, Coastal: true, Desc: "Züchtet Algen in Becken an der Küste.", Col: hex(0x3fcf8e)})
	B(BNahrung, BType{Name: "Nahrungsfabrik", Cat: CatT1, W: 2, H: 2, Cost: 500, Mat: [3]int{5, 0, 0}, Upkeep: 20, Energy: -3, In: []int{GAlgen}, Out: GNahrung, Cycle: 30, Desc: "Stellt aus Algen nahrhafte Rationen her.", Col: hex(0xf2b84b)})
	B(BWasser, BType{Name: "Wasserwerk", Cat: CatT1, W: 2, H: 2, Cost: 400, Mat: [3]int{4, 0, 0}, Upkeep: 15, Energy: -2, In: none, Out: GWasser, Cycle: 30, Coastal: true, Desc: "Entsalzt Meerwasser zu Reinwasser.", Col: hex(0x4fc3ff)})
	B(BErz, BType{Name: "Erzmine", Cat: CatT2, W: 2, H: 2, Cost: 800, Mat: [3]int{6, 0, 0}, Upkeep: 25, Energy: -4, In: none, Out: GErz, Cycle: 30, Deposit: DepErz, Tier: TTechniker, Unlock: 1, Desc: "Muss auf einem Erzvorkommen im Gebirge stehen.", Col: hex(0xc0703a)})
	B(BStahl, BType{Name: "Stahlwerk", Cat: CatT2, W: 3, H: 2, Cost: 1500, Mat: [3]int{8, 0, 0}, Upkeep: 45, Energy: -8, In: []int{GErz}, Out: GStahl, Cycle: 30, Tier: TTechniker, Unlock: 1, Desc: "Schmilzt Erz zu Stahl – Baumaterial der Techniker.", Col: hex(0x9fb3c8)})
	B(BKaffeePl, BType{Name: "Kaffeeplantage", Cat: CatT2, W: 3, H: 3, Cost: 600, Mat: [3]int{4, 0, 0}, Upkeep: 15, Energy: -1, In: none, Out: GKaffee, Cycle: 30, Fert: FertKaffee, Tier: TTechniker, Unlock: 1, Desc: "Benötigt eine Insel mit Kaffee-Fruchtbarkeit.", Col: hex(0x507a30)})
	B(BRoest, BType{Name: "Kaffeerösterei", Cat: CatT2, W: 2, H: 2, Cost: 1200, Mat: [3]int{6, 2, 0}, Upkeep: 35, Energy: -4, In: []int{GKaffee}, Out: GSynthkaffee, Cycle: 30, Tier: TTechniker, Unlock: 1, Desc: "Veredelt Kaffeebohnen zu Synthkaffee.", Col: hex(0xb5651d)})
	B(BFaserPl, BType{Name: "Faserplantage", Cat: CatT2, W: 3, H: 3, Cost: 600, Mat: [3]int{4, 0, 0}, Upkeep: 15, Energy: -1, In: none, Out: GFasern, Cycle: 30, Fert: FertFasern, Tier: TTechniker, Unlock: 1, Desc: "Benötigt eine Insel mit Fasern-Fruchtbarkeit.", Col: hex(0xc8e06b)})
	B(BTextil, BType{Name: "Textilfabrik", Cat: CatT2, W: 2, H: 2, Cost: 1300, Mat: [3]int{6, 2, 0}, Upkeep: 40, Energy: -5, In: []int{GFasern}, Out: GTextil, Cycle: 30, Tier: TTechniker, Unlock: 1, Desc: "Webt Biofasern zu Smart-Textil.", Col: hex(0xc061ff)})
	B(BSilMine, BType{Name: "Siliziummine", Cat: CatT3, W: 2, H: 2, Cost: 2500, Mat: [3]int{8, 6, 0}, Upkeep: 50, Energy: -6, In: none, Out: GSilizium, Cycle: 30, Deposit: DepSil, Tier: TIngenieure, Unlock: 1, Desc: "Muss auf einem Siliziumvorkommen stehen.", Col: hex(0x7fe3e0)})
	B(BChip, BType{Name: "Chipfabrik", Cat: CatT3, W: 3, H: 2, Cost: 4000, Mat: [3]int{10, 8, 0}, Upkeep: 90, Energy: -12, In: []int{GSilizium}, Out: GChips, Cycle: 30, Tier: TTechniker, Unlock: 100, Desc: "Belichtet Silizium zu Mikrochips – Baumaterial der Ingenieure.", Col: hex(0x46f0a0)})
	B(BHolo, BType{Name: "Holo-Fabrik", Cat: CatT3, W: 3, H: 3, Cost: 6000, Mat: [3]int{12, 10, 4}, Upkeep: 130, Energy: -15, In: []int{GChips, GStahl}, Out: GHolo, Cycle: 60, Tier: TIngenieure, Unlock: 1, Desc: "Fertigt Holo-Pads aus Mikrochips und Stahl.", Col: hex(0x5aa8ff)})
	B(BTurret, BType{Name: "Abwehrturm", Cat: CatInfra, W: 1, H: 1, Cost: 1500, Mat: [3]int{10, 0, 0}, Upkeep: 25, Energy: -2, Out: -1, Tier: TArbeiter, Unlock: 120, Radius: 7, Desc: "Laser-Abwehrturm. Beschießt Piraten und feindliche Schiffe in Reichweite.", Col: hex(0xd04a4a)})
	B(BFeuer, BType{Name: "Feuerwache", Cat: CatPublic, W: 2, H: 2, Cost: 1400, Mat: [3]int{10, 0, 0}, Upkeep: 30, Energy: -2, Out: -1, Tier: TArbeiter, Unlock: 100, Radius: 14, Desc: "Löschdrohnen bekämpfen Brände im Umkreis und senken die Brandgefahr.", Col: hex(0xff5a3a)})
	B(BRaumhafen, BType{Name: "Raumhafen", Cat: CatInfra, W: 4, H: 4, Cost: 30000, Mat: [3]int{40, 30, 15}, Upkeep: 250, Energy: -20, Out: -1, Tier: TIngenieure, Unlock: 60, Desc: "Startrampe zum Mond. Schaltet die Mondbasis frei und versorgt sie per Rakete.", Col: hex(0xe8f4ff)})
	B(BHe3, BType{Name: "Helium-3-Extraktor", Cat: CatT3, W: 2, H: 2, Cost: 4000, Mat: [3]int{10, 10, 4}, Upkeep: 60, Energy: -6, In: none, Out: GHelium3, Cycle: 40, Deposit: DepHe3, Tier: TIngenieure, Unlock: 1, Desc: "Nur auf dem Mond: fördert Helium-3 aus dem Regolith. Treibt Fusionsreaktoren auf der Erde an.", Col: hex(0xe8f4ff)})
	// Siliziummine wird bereits mit 100 Technikern freigeschaltet, damit Chips vor den Ingenieuren möglich sind
	BTypes[BSilMine].Tier = TTechniker
	BTypes[BSilMine].Unlock = 100

	Tiers[TArbeiter] = TierInfo{Name: "Arbeiter", Plural: "Arbeiter", MaxRes: 10, Tax: 1.4, Col: hex(0xc8d8ee),
		Needs: []Need{{GNahrung, 0, 1.0 / 150}, {GWasser, 0, 1.0 / 200}, {-1, BKomm, 0}}}
	Tiers[TTechniker] = TierInfo{Name: "Techniker", Plural: "Techniker", MaxRes: 20, Tax: 3.0, Col: hex(0xeab46a),
		Needs:  []Need{{GNahrung, 0, 1.0 / 120}, {GWasser, 0, 1.0 / 160}, {GSynthkaffee, 0, 1.0 / 150}, {GTextil, 0, 1.0 / 200}, {-1, BKomm, 0}, {-1, BMedi, 0}},
		UpCost: [3]int{3, 0, 0}, UnlockPop: 60}
	Tiers[TIngenieure] = TierInfo{Name: "Ingenieur", Plural: "Ingenieure", MaxRes: 30, Tax: 5.5, Col: hex(0xc49ae8),
		Needs:  []Need{{GNahrung, 0, 1.0 / 100}, {GSynthkaffee, 0, 1.0 / 120}, {GTextil, 0, 1.0 / 160}, {GHolo, 0, 1.0 / 250}, {-1, BKomm, 0}, {-1, BMedi, 0}, {-1, BLabor, 0}},
		UpCost: [3]int{2, 3, 0}, UnlockPop: 120}
}

// Gebäude freigeschaltet?
func (g *Game) Unlocked(t int) bool {
	b := &BTypes[t]
	if b.Unlock == 0 {
		return true
	}
	return g.MaxPop[b.Tier] >= b.Unlock
}

func (g *Game) UnlockText(t int) string {
	b := &BTypes[t]
	return T("Benötigt ") + itoa(b.Unlock) + " " + T(Tiers[b.Tier].Plural)
}
