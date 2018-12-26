package gobot

import (
	"image"

	"github.com/kjx98/go-sdl2/sdl"
)

var window *sdl.Window

func SetJpegWindow(w *sdl.Window) {
	if window != nil {
		return
	}
	window = w
}

func initJpegWin(w, h int) (err error) {
	if window != nil {
		return
	}
	if w == 0 {
		w = 320
	}
	if h == 0 {
		h = 320
	}
	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return
	}
	window, err = sdl.CreateWindow("请扫描二维码登录", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, int32(w), int32(h), sdl.WINDOW_SHOWN)
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

func dispImage(jpImg image.Image) error {
	if window == nil {
		w := jpImg.Bounds().Max.X - jpImg.Bounds().Min.X
		h := jpImg.Bounds().Max.Y - jpImg.Bounds().Min.Y
		if err := initJpegWin(w, h); err != nil {
			return err
		}
	}
	surface, err := window.GetSurface()
	if err != nil {
		return err
	}

	for i := jpImg.Bounds().Min.X; i < jpImg.Bounds().Max.X; i++ {
		for j := jpImg.Bounds().Min.Y; j < jpImg.Bounds().Max.Y; j++ {
			surface.Set(i, j, jpImg.At(i, j))
		}
	}

	window.UpdateSurface()

	return nil
}
