package main

import (
	"github.com/gdamore/tcell"
	"github.com/lithdew/blanc"
	"github.com/lithdew/blanc/layout"
)

var footerRect layout.Rect
var footerInputRect layout.Rect

var footerInput *blanc.Textbox

var footerStyle tcell.Style

func resizeFooter() {
	footerRect = screenRect.Position(layout.Bottom | layout.Left)
	footerRect = footerRect.WidthOf(screenRect)
	footerRect = footerRect.Height(1)

	footerInputRect = footerRect.PadLeft(1)
}

func initFooter() {
	footerStyle = tcell.StyleDefault
	footerStyle = footerStyle.Background(tcell.ColorWhite)
	footerStyle = footerStyle.Foreground(tcell.ColorBlack)

	initFooterInput()
}

func initFooterInput() {
	selectedStyle := tcell.StyleDefault
	selectedStyle = selectedStyle.Background(tcell.ColorDarkGray)
	selectedStyle = selectedStyle.Foreground(tcell.ColorWhite)

	footerInput = blanc.NewTextbox()
	footerInput.SetLabel(">>> ")
	footerInput.SetTextStyle(footerStyle)
	footerInput.SetLabelStyle(footerStyle)
	footerInput.SetSelectedStyle(selectedStyle)
}

func renderFooter() {
	blanc.Clear(screen, footerStyle, footerRect)
	footerInput.Draw(screen, footerInputRect)
}
