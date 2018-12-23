package gobot

import (
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"time"
)

func dispJPEG(picImg []byte) (err error) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return err
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, 320, 320, sdl.WINDOW_SHOWN)
	if err != nil {
		return err
	}
	defer window.Destroy()

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
	png.BlitScaled(nil, surface, nil)
	window.UpdateSurface()

	//timeo := time.NewTimer(time.Minute * 1)
	timeo := time.NewTimer(time.Second * 20)
	running := true
	for running {
		select {
		case <-timeo.C:
			running = false
		default:
		}
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			}
		}
	}
	return nil
}
