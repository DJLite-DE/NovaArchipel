//go:build !windows

package game

// Ohne Windows keine Tonausgabe (nur Mixer für Tests)
func StartAudio() { NewMixer() }
