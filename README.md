# Nova Archipel 🏝️🚀

Aufbau-Strategiespiel im Jahr 2310 – Inseln besiedeln, Produktionsketten verknüpfen, Bürger vom Arbeiter bis zum Ingenieur führen, KI-Rivalen, Seeschlachten und eine Mondbasis.

## Download
Die neueste Version gibt es unter **[Releases](../../releases/latest)**:
- `NovaArchipel-Setup.exe` – Installer (Startmenü + Desktop-Verknüpfung, Deinstallation über Windows-Einstellungen)
- `NovaArchipel.exe` – direkt startbar, ohne Installation

Das Spiel aktualisiert sich selbst: beim Start wird hier nach einer neuen Version gesucht.

> Hinweis: Die exe ist nicht signiert. Windows SmartScreen zeigt deshalb beim ersten Start eine Warnung → „Weitere Informationen“ → „Trotzdem ausführen“.

## Bauen
Go 1.22+, keine externen Abhängigkeiten:
```
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o NovaArchipel.exe ./cmd/nova
```
Installer: `NovaArchipel.exe` nach `cmd/setup/payload/` kopieren, dann `go build ... -o NovaArchipel-Setup.exe ./cmd/setup`.

## Credits
Teile der 3D-Modelle: [Kenney](https://www.kenney.nl) (CC0, siehe `tools/KENNEY_LICENSE.txt`).
