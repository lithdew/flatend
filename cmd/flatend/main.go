package main

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
	"github.com/mattn/go-runewidth"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func text(s tcell.Screen, x, y int, style tcell.Style, txt string) {
	for _, c := range txt {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		s.SetContent(x, y, c, comb, style)
		x += w
	}
}

func box(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, r rune) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}
	if y1 != y2 && x1 != x2 {
		// Only add corners if we need to
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		for col := x1 + 1; col < x2; col++ {
			s.SetContent(col, row, r, nil, style)
		}
	}
}

type inputtable interface {
	keyPress(e *tcell.EventKey)
}

type renderable interface {
	render(s tcell.Screen)
}

type widget interface {
	renderable
	inputtable
}

type alignType int

const (
	alignNone alignType = iota
	alignTop
	alignBottom
)

type textbox struct {
	x, y, w, h int
	style      tcell.Style

	align alignType
	fw    bool
	fh    bool

	buf string
}

func (t *textbox) keyPress(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyBS, tcell.KeyDEL:
		if len(t.buf) > 0 {
			t.buf = t.buf[:len(t.buf)-1]
		}
	default:
		t.buf += string(e.Rune())
	}

}

func (t *textbox) render(s tcell.Screen) {
	x, y, w, h := t.x, t.y, t.w, t.h

	sw, sh := s.Size()
	if t.fw {
		w = sw
	}
	if t.fh {
		h = sh
	}
	switch t.align {
	case alignNone:
	case alignTop:
		x, y = 0, 0
	case alignBottom:
		x, y = 0, sh-1-h
	}

	box(s, x, y, x+w-1, y+h, t.style, ' ')

	text(s, x+1, y+1, t.style, t.buf)
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

	input := &textbox{h: 2, fw: true, align: alignBottom, style: styleWhite}
	widgets = append(widgets, input)

	for {
		for _, w := range widgets {
			w.render(s)
		}
		text(s, 1, 1, styleWhite, "flatend ::")

		s.Show()

		switch e := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			switch e.Key() {
			case tcell.KeyTAB:
				focused = (focused + 1) % (len(widgets) + 1)
			case tcell.KeyEscape, tcell.KeyExit, tcell.KeyCtrlC:
				return
			case tcell.KeyCtrlL:
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
