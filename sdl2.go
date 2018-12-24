package gobot

import (
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

var window *sdl.Window

func SetJpegWindow(w *sdl.Window) {
	if window != nil {
		return
	}
	window = w
}

func initJpegWin() (err error) {
	if window != nil {
		return
	}
	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return
	}
	window, err = sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, 320, 320, sdl.WINDOW_SHOWN)
	if err != nil {
		return
	}
	return
}

func jpegLoop() bool {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			println("dispJPEG Quit")
			return false
		}
	}
	return true
}

func shutJpegWin() {
	defer sdl.Quit()
	// first remove all event
	jpegLoop()
	window.Destroy()
}

func dispJPEG(picImg []byte) (err error) {
	if window == nil {
		if err = initJpegWin(); err != nil {
			return
		}
	}
	surface, err := window.GetSurface()
	if err != nil {
		return err
	}

	rwops, err := sdl.RWFromMem(picImg)
	if err != nil {
		return err
	}
	png, err := img.LoadRW(rwops, false)
	if err != nil {
		return err
	}
	//println("w/h:", png.W, png.H)
	//png.BlitScaled(nil, surface, nil)
	png.Blit(nil, surface, nil)
	window.UpdateSurface()

	return nil
}
