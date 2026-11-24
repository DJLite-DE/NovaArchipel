// Nova Archipel – Installer / Deinstaller (Windows)
package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload/NovaArchipel.exe
var gameExe []byte

const (
	appName = "Nova Archipel"
	version = "3.0"
	regKey  = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\NovaArchipel`
)

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	msgBoxW = user32.NewProc("MessageBoxW")
)

func box(text string, flags uintptr) int {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(appName + " " + version)
	r, _, _ := msgBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags|0x40000)
	return int(r)
}

const (
	mbOK    = 0x40
	mbYesNo = 0x4 | 0x20
	mbErr   = 0x10
	idYes   = 6
)

func hidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd.Run()
}

func installDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "NovaArchipel")
}

func shortcutPaths() []string {
	return []string{
		filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`, appName+".lnk"),
		filepath.Join(os.Getenv("USERPROFILE"), "Desktop", appName+".lnk"),
	}
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func makeShortcut(lnk, target, dir string) error {
	script := "$s=(New-Object -ComObject WScript.Shell).CreateShortcut(" + psQuote(lnk) + ");" +
		"$s.TargetPath=" + psQuote(target) + ";$s.WorkingDirectory=" + psQuote(dir) + ";" +
		"$s.IconLocation=" + psQuote(target+",0") + ";$s.Description='Nova Archipel – Kolonien der Zukunft';$s.Save()"
	return hidden("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
}

func install() {
	dir := installDir()
	if box("Nova Archipel "+version+" wird installiert nach:\n\n"+dir+
		"\n\nEs werden Verknüpfungen im Startmenü und auf dem Desktop angelegt.\nSpielstände und Einstellungen bleiben erhalten.\n\nJetzt installieren?", mbYesNo) != idYes {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		box("Ordner konnte nicht angelegt werden:\n"+err.Error(), mbErr)
		return
	}
	exe := filepath.Join(dir, "NovaArchipel.exe")
	os.Remove(exe + ".old")
	if err := os.WriteFile(exe, gameExe, 0755); err != nil {
		box("Das Spiel konnte nicht kopiert werden – läuft es vielleicht noch?\n\n"+err.Error(), mbErr)
		return
	}
	// Deinstaller = Kopie dieses Setups
	uninst := filepath.Join(dir, "Deinstallieren.exe")
	if self, err := os.Executable(); err == nil && !strings.EqualFold(self, uninst) {
		if b, err := os.ReadFile(self); err == nil {
			os.WriteFile(uninst, b, 0755)
		}
	}
	for _, l := range shortcutPaths() {
		os.MkdirAll(filepath.Dir(l), 0755)
		makeShortcut(l, exe, dir)
	}
	size := (len(gameExe) * 2) / 1024
	reg := func(name, typ, val string) {
		hidden("reg.exe", "add", regKey, "/v", name, "/t", typ, "/d", val, "/f")
	}
	reg("DisplayName", "REG_SZ", appName)
	reg("DisplayVersion", "REG_SZ", version)
	reg("Publisher", "REG_SZ", "DJLite-DE")
	reg("DisplayIcon", "REG_SZ", exe)
	reg("InstallLocation", "REG_SZ", dir)
	reg("UninstallString", "REG_SZ", `"`+uninst+`" --uninstall`)
	reg("EstimatedSize", "REG_DWORD", itoa(size))
	reg("NoModify", "REG_DWORD", "1")
	reg("NoRepair", "REG_DWORD", "1")

	if box("Installation abgeschlossen! 🎉\n\nNova Archipel jetzt starten?", mbYesNo) == idYes {
		cmd := exec.Command(exe)
		cmd.Dir = dir
		cmd.Start()
	}
}

func uninstall() {
	if box("Nova Archipel wirklich deinstallieren?\n\nDeine Spielstände in %APPDATA%\\NovaArchipel bleiben erhalten.", mbYesNo) != idYes {
		return
	}
	dir := installDir()
	for _, l := range shortcutPaths() {
		os.Remove(l)
	}
	hidden("reg.exe", "delete", regKey, "/f")
	os.Remove(filepath.Join(dir, "NovaArchipel.exe"))
	os.Remove(filepath.Join(dir, "NovaArchipel.exe.old"))
	os.Remove(filepath.Join(dir, "NovaArchipel.new.exe"))
	box("Nova Archipel wurde entfernt. Bis bald! 👋", mbOK)
	// Ordner samt Deinstaller nach dem Beenden löschen
	cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 3 >nul & rmdir /s /q \""+dir+"\"")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	cmd.Start()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func main() {
	for _, a := range os.Args[1:] {
		if a == "--uninstall" || a == "/uninstall" {
			uninstall()
			return
		}
	}
	install()
}
