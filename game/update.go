package game

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------- Auto-Updater über GitHub-Releases

const UpdateRepo = "DJLite-DE/NovaArchipel"

const (
	UpdIdle = iota
	UpdChecking
	UpdLatest
	UpdAvailable
	UpdDownloading
	UpdReady
	UpdError
	UpdNoChannel
)

type updateState struct {
	mu      sync.Mutex
	State   int
	Tag     string
	Notes   string
	URL     string
	Size    int64
	Done    int64
	Err     string
	newPath string
}

var Upd updateState

func (u *updateState) get() updateState {
	u.mu.Lock()
	defer u.mu.Unlock()
	return updateState{State: u.State, Tag: u.Tag, Notes: u.Notes, URL: u.URL, Size: u.Size, Done: u.Done, Err: u.Err}
}

func (u *updateState) set(f func(*updateState)) {
	u.mu.Lock()
	f(u)
	u.mu.Unlock()
}

// Versionsvergleich "v2.10" > "2.9"
func versionNewer(tag, cur string) bool {
	parse := func(s string) []int {
		s = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(s)), "v")
		var out []int
		for _, p := range strings.Split(s, ".") {
			n, _ := strconv.Atoi(strings.TrimFunc(p, func(r rune) bool { return r < '0' || r > '9' }))
			out = append(out, n)
		}
		return out
	}
	a, b := parse(tag), parse(cur)
	for i := 0; i < len(a) || i < len(b); i++ {
		x, y := 0, 0
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// Reste eines früheren Updates entfernen
func CleanupUpdate() {
	if exe, err := os.Executable(); err == nil {
		os.Remove(exe + ".old")
		os.Remove(filepath.Join(filepath.Dir(exe), "NovaArchipel.new.exe"))
	}
}

func CheckUpdate() {
	Upd.set(func(u *updateState) { u.State = UpdChecking; u.Err = "" })
	go func() {
		cl := &http.Client{Timeout: 15 * time.Second}
		req, _ := http.NewRequest("GET", T("https://api.github.com/repos/")+UpdateRepo+"/releases/latest", nil)
		req.Header.Set("User-Agent", T("NovaArchipel/")+Version)
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := cl.Do(req)
		if err != nil {
			Upd.set(func(u *updateState) { u.State = UpdError; u.Err = "Keine Verbindung zu GitHub" })
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			Upd.set(func(u *updateState) { u.State = UpdNoChannel })
			return
		}
		if resp.StatusCode != 200 {
			Upd.set(func(u *updateState) {
				u.State = UpdError
				u.Err = Tf("GitHub antwortet mit %d", resp.StatusCode)
			})
			return
		}
		var rel struct {
			Tag    string `json:"tag_name"`
			Body   string `json:"body"`
			Assets []struct {
				Name string `json:"name"`
				URL  string `json:"browser_download_url"`
				Size int64  `json:"size"`
			} `json:"assets"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			Upd.set(func(u *updateState) { u.State = UpdError; u.Err = "Antwort unlesbar" })
			return
		}
		if !versionNewer(rel.Tag, Version) {
			Upd.set(func(u *updateState) { u.State = UpdLatest })
			return
		}
		for _, a := range rel.Assets {
			if strings.EqualFold(a.Name, "NovaArchipel.exe") {
				Upd.set(func(u *updateState) {
					u.State, u.Tag, u.Notes, u.URL, u.Size = UpdAvailable, rel.Tag, rel.Body, a.URL, a.Size
				})
				return
			}
		}
		Upd.set(func(u *updateState) { u.State = UpdLatest })
	}()
}

type countWriter struct {
	w io.Writer
}

func (c countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	Upd.set(func(u *updateState) { u.Done += int64(n) })
	return n, err
}

func StartUpdateDownload() {
	st := Upd.get()
	if st.State != UpdAvailable {
		return
	}
	Upd.set(func(u *updateState) { u.State = UpdDownloading; u.Done = 0 })
	go func() {
		fail := func(msg string) {
			Upd.set(func(u *updateState) { u.State = UpdError; u.Err = msg })
		}
		exe, err := os.Executable()
		if err != nil {
			fail("Programmpfad unbekannt")
			return
		}
		np := filepath.Join(filepath.Dir(exe), "NovaArchipel.new.exe")
		cl := &http.Client{Timeout: 10 * time.Minute}
		req, _ := http.NewRequest("GET", st.URL, nil)
		req.Header.Set("User-Agent", T("NovaArchipel/")+Version)
		resp, err := cl.Do(req)
		if err != nil || resp.StatusCode != 200 {
			fail("Download fehlgeschlagen")
			return
		}
		defer resp.Body.Close()
		f, err := os.Create(np)
		if err != nil {
			fail("Datei kann nicht geschrieben werden")
			return
		}
		n, err := io.Copy(countWriter{f}, resp.Body)
		f.Close()
		if err != nil || (st.Size > 0 && n != st.Size) || n < 1<<20 {
			os.Remove(np)
			fail("Download unvollständig")
			return
		}
		Upd.set(func(u *updateState) { u.State = UpdReady; u.newPath = np })
	}()
}

// Neue Version einsetzen und neu starten
func ApplyUpdateAndRestart() error {
	Upd.mu.Lock()
	np := Upd.newPath
	Upd.mu.Unlock()
	exe, err := os.Executable()
	if err != nil || np == "" {
		return fmt.Errorf("kein Update bereit")
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(np, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	In.Quit = true
	return nil
}

// Update-Hinweis im Hauptmenü
func (a *App) drawUpdateCard(c *Canvas, x, y int) {
	st := Upd.get()
	u := a.mu
	switch st.State {
	case UpdAvailable, UpdDownloading, UpdReady:
	case UpdChecking:
		c.Text(FSmall, x, y, "Suche nach Updates …", cDim)
		return
	case UpdLatest:
		c.Text(FSmall, x, y, "Du hast die neueste Version.", cDim)
		return
	case UpdError:
		c.Text(FSmall, x, y, T("Update: ")+st.Err, cDim)
		return
	default:
		return
	}
	w, h := 420, 128
	u.panels = append(u.panels, [4]int{x, y, w, h})
	c.RoundRect(x, y, w, h, 12, RGB{12, 30, 40}, 220)
	c.RoundRectLine(x, y, w, h, 12, cAccent, 160)
	c.Text(FSmall, x+18, y+12, "UPDATE VERFÜGBAR", cAccent)
	c.Text(FSerif, x+18, y+30, T("Version ")+strings.TrimPrefix(st.Tag, "v")+T(" ist da"), cText)
	notes := strings.TrimSpace(strings.SplitN(st.Notes, "\n", 2)[0])
	if notes != "" {
		for TextW(FSmall, notes) > w-36 && len(notes) > 4 {
			notes = notes[:len(notes)-4] + "…"
		}
		c.Text(FSmall, x+18, y+60, notes, cDim)
	}
	switch st.State {
	case UpdAvailable:
		if !u.ShowOptions && u.button(c, x+18, y+h-42, 200, 32, "Jetzt aktualisieren", true) {
			if a.Cur != nil {
				a.Cur.SaveAuto()
			}
			StartUpdateDownload()
		}
	case UpdDownloading:
		f := 0.0
		if st.Size > 0 {
			f = float64(st.Done) / float64(st.Size)
		}
		bar(c, x+18, y+h-32, w-36, 10, f, cAccent)
		c.TextR(FSmall, x+w-18, y+h-52, Tf("%.0f %%", f*100), cText)
	case UpdReady:
		if !u.ShowOptions && u.button(c, x+18, y+h-42, 220, 32, "Neu starten & installieren", true) {
			if err := ApplyUpdateAndRestart(); err != nil {
				Upd.set(func(s *updateState) { s.State = UpdError; s.Err = T("Installation fehlgeschlagen: ") + err.Error() })
			}
		}
	}
}
