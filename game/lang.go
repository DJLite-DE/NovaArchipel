package game

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ---------------------------------------------------------------- Sprachen (Deutsch / Englisch)

var (
	tMu     sync.Mutex
	tCache  = map[string]string{}
	tNames  []string
	tLangOn string
)

// Übersetzt einen deutschen Text in die eingestellte Sprache
func T(s string) string {
	if Opt.Lang != "en" || s == "" {
		return s
	}
	if v, ok := enDict[s]; ok {
		return v
	}
	tMu.Lock()
	defer tMu.Unlock()
	if v, ok := tCache[s]; ok {
		return v
	}
	if tNames == nil {
		set := map[string]bool{}
		add := func(n string) {
			if _, ok := enDict[n]; ok && len(n) >= 4 {
				set[n] = true
			}
		}
		for _, b := range BTypes {
			add(b.Name)
		}
		for _, g := range Goods {
			add(g.Name)
		}
		for _, t := range Tiers {
			add(t.Name)
			add(t.Plural)
		}
		for _, c := range CatNames {
			add(c)
		}
		for _, k := range []string{"Obsidian Syndikat", "Korvette", "Frachtgleiter", "Kontor", "Wohnmodule", "Einwohner", "Credits"} {
			add(k)
		}
		for k := range set {
			tNames = append(tNames, k)
		}
		sort.Slice(tNames, func(i, j int) bool { return len(tNames[i]) > len(tNames[j]) })
	}
	out := s
	for _, k := range tNames {
		if strings.Contains(out, k) {
			out = strings.ReplaceAll(out, k, enDict[k])
		}
	}
	if len(tCache) > 4000 {
		tCache = map[string]string{}
	}
	tCache[s] = out
	return out
}

// Sprintf mit übersetztem Formattext
func Tf(format string, a ...interface{}) string {
	return fmt.Sprintf(T(format), a...)
}
