package main

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
	"github.com/lithdew/blanc/layout"
	"log"
	"time"
)

var err error

var screen tcell.Screen
var screenRect layout.Rect

func check(err error) {
	if err != nil {
		if screen != nil {
			screen.Fini()
		}
		log.Panic(err)
	}
}

func init() {
	encoding.Register()

	initScreen()
	initHeader()
	initBody()
	initFooter()
}

func initScreen() {
	screen, err = tcell.NewScreen()
	check(err)
}

func resizeScreen() {
	screen.Sync()

	w, h := screen.Size()
	screenRect = screenRect.Width(w)
	screenRect = screenRect.Height(h)

	resizeHeader()
	resizeBody()
	resizeFooter()
}

func renderScreen() {
	renderHeader()
	renderBody()
	renderFooter()

	screen.Show()
}

var ch chan struct{}

func eventLoop() {
	defer close(ch)
	for {
		e := screen.PollEvent()
		switch e := e.(type) {
		case *tcell.EventKey:
			switch e.Key() {
			case tcell.KeyCtrlC:
				return
			case tcell.KeyCtrlL:
				resizeScreen()
			}
		case *tcell.EventResize:
			resizeScreen()
		}

		if footerInput.HandleEvent(e) {
			//if footerInput.Text() == "http" {
			//	fmt.Println("ready")
			//}
		}
	}
}

func main() {
	check(screen.Init())
	defer screen.Fini()

	resizeScreen()

	ch = make(chan struct{})
	go eventLoop()

	for {
		select {
		case <-ch:
			return
		case <-time.After(40 * time.Millisecond):
		}

		renderScreen()
	}
}
