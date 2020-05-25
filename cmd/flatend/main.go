package main

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var focused int
var widgets []widget

func currentFocus() widget {
	if focused == len(widgets) {
		return nil
	}
	return widgets[focused]
}

func eventLoop() {
	defer close(ch)

	input := &textbox{h: 2, tpx: 1, tpy: 1, fw: true, ba: alignBottom, bs: styleWhite}
	widgets = append(widgets, input)

	for {
		for _, w := range widgets {
			w.render(s)
		}

		text(s, 1, 1, styleWhite, "flatend ::")

		s.Show()

		switch e := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Clear()
			s.Sync()
		case *tcell.EventKey:
			switch e.Key() {
			case tcell.KeyTAB:
				focused = (focused + 1) % (len(widgets) + 1)
			case tcell.KeyEscape, tcell.KeyExit, tcell.KeyCtrlC:
				return
			case tcell.KeyCtrlL:
				s.Clear()
				s.Sync()
			default:
				w := currentFocus()
				if w != nil {
					w.keyPress(e)
				}
			}
		}
	}
}

var (
	styleDefault = tcell.StyleDefault.Foreground(tcell.ColorBlack)
	styleWhite   = tcell.StyleDefault.Foreground(tcell.ColorWhite)
)

var (
	s   tcell.Screen
	err error
	ch  chan struct{}
)

func main() {
	encoding.Register()

	s, err = tcell.NewScreen()
	check(err)

	check(s.Init())
	defer s.Fini()

	s.SetStyle(styleDefault)
	s.EnableMouse()
	s.Clear()

	ch = make(chan struct{})
	go eventLoop()
	<-ch
}
