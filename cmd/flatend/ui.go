package main

import (
	"github.com/gdamore/tcell"
)

type inputtable interface {
	keyPress(e *tcell.EventKey)
}

type renderable interface {
	render(s tcell.Screen)
}

type widget interface {
	inputtable
	renderable
}

type alignType int

const (
	alignNone alignType = 1 << iota
	alignTop
	alignBottom
	alignCenter
	alignLeft
	alignRight
)

type cell struct {
	sw int // container width
	sh int // container height

	x     int       // position x
	y     int       // position y
	w     int       // size width
	h     int       // size height
	px    int       // padding x
	py    int       // padding y
	fw    bool      // full width
	fh    bool      // full height
	align alignType // alignment type
}

func (c cell) pos() {
	x, y, w, h := c.x, c.y, c.w, c.h

	// apply full width/full height
	// also constrain max width/height

	if c.fw || w > c.w {
		w = c.sw
	}
	if c.fh || h > c.h {
		h = c.sh
	}

	// apply padding

	x += c.px
	y += c.py
	w -= c.px
	h -= c.py

	switch {
	case c.align&alignTop == alignTop:

	case c.align&alignBottom == alignBottom:
	}
}

type textbox struct {
	x, y, w, h int // x, y, width, and height

	tpx, tpy int // text padding x, text padding y

	bs tcell.Style // box style
	ts tcell.Style // text style

	ba alignType // box alignment (top, bottom)
	ta alignType // text alignment (top, bottom)

	fw bool // full width?
	fh bool // full height?

	buf string // text buffer
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
	switch t.ba {
	case alignNone:
		fallthrough
	case alignTop:
		x, y = 0, 0
	case alignBottom:
		x, y = 0, sh-1-h
	}

	box(s, x, y, x+w-1, y+h, t.bs, ' ')

	tx, ty := x+1, y+1

	//switch t.ta {
	//case alignNone:
	//	fallthrough
	//case alignTop:
	//	tx, ty = x, y
	//case alignBottom:
	//	tx, ty = x+w-1, y+h-1
	//case alignCenter:
	//	tx, ty = x+w/2-1, y+h/2-1
	//}

	tx += t.tpx
	ty += t.tpy

	text(s, tx, ty, t.bs, t.buf)
}
