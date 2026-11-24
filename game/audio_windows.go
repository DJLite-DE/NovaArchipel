//go:build windows

package game

import (
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	winmmA          = syscall.NewLazyDLL("winmm.dll")
	pWaveOutOpen    = winmmA.NewProc("waveOutOpen")
	pWaveOutPrepare = winmmA.NewProc("waveOutPrepareHeader")
	pWaveOutWrite   = winmmA.NewProc("waveOutWrite")
)

type waveFormat struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	CbSize         uint16
}

type waveHdr struct {
	Data          uintptr
	BufferLength  uint32
	BytesRecorded uint32
	User          uintptr
	Flags         uint32
	Loops         uint32
	Next          uintptr
	Reserved      uintptr
}

var audioBufs [][]int16
var audioHdrs []*waveHdr

// Startet die Tonausgabe in einem eigenen Thread
func StartAudio() {
	m := NewMixer()
	go func() {
		defer func() { recover() }()
		var hwo uintptr
		wf := waveFormat{FormatTag: 1, Channels: 2, SamplesPerSec: SR, AvgBytesPerSec: SR * 4, BlockAlign: 4, BitsPerSample: 16}
		r, _, _ := pWaveOutOpen.Call(uintptr(unsafe.Pointer(&hwo)), 0xFFFFFFFF, uintptr(unsafe.Pointer(&wf)), 0, 0, 0)
		if r != 0 {
			return
		}
		const frames = 1024
		const nb = 4
		for i := 0; i < nb; i++ {
			b := make([]int16, frames*2)
			h := &waveHdr{Data: uintptr(unsafe.Pointer(&b[0])), BufferLength: frames * 4}
			audioBufs = append(audioBufs, b)
			audioHdrs = append(audioHdrs, h)
			pWaveOutPrepare.Call(hwo, uintptr(unsafe.Pointer(h)), unsafe.Sizeof(*h))
			m.Fill(b)
			pWaveOutWrite.Call(hwo, uintptr(unsafe.Pointer(h)), unsafe.Sizeof(*h))
		}
		cur := 0
		for {
			h := audioHdrs[cur]
			for atomic.LoadUint32(&h.Flags)&1 == 0 { // WHDR_DONE
				time.Sleep(2 * time.Millisecond)
			}
			atomic.StoreUint32(&h.Flags, atomic.LoadUint32(&h.Flags)&^1)
			m.Fill(audioBufs[cur])
			pWaveOutWrite.Call(hwo, uintptr(unsafe.Pointer(h)), unsafe.Sizeof(*h))
			cur = (cur + 1) % nb
		}
	}()
}
