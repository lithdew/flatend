package main

import (
	"github.com/gdamore/tcell"
	"github.com/lithdew/blanc"
	"github.com/lithdew/blanc/layout"
	"github.com/mattn/go-runewidth"
)

var headerRect layout.Rect
var headerTitle blanc.Text
var headerStyle tcell.Style

var tabs = []string{" API ", " Logs ", " Metrics "}
var tabsRect layout.Rect
var tabSelected = 0

func initHeader() {
	headerStyle = tcell.StyleDefault
	headerStyle = headerStyle.Background(tcell.ColorWhite)
	headerStyle = headerStyle.Foreground(tcell.ColorBlack)

	initHeaderTitle()
}

func initHeaderTitle() {
	headerTitle = blanc.NewText("flatend.")
	headerTitle.SetStyle(headerStyle)
}

func resizeHeader() {
	headerRect = screenRect.Height(1)
	resizeHeaderTabs()
}

func resizeHeaderTabs() {
	tabsWidth := 0
	for i := 0; i < len(tabs); i++ {
		tabsWidth += runewidth.StringWidth(tabs[i])
	}

	tabsRect = screenRect.Align(layout.Right)
	tabsRect = tabsRect.MoveLeft(tabsWidth)
	tabsRect = tabsRect.Width(tabsWidth)
}

func renderHeader() {
	blanc.Clear(screen, headerStyle, headerRect)

	headerTitle.Draw(screen, headerRect.PadLeft(1))
	renderHeaderTabs()
}

func renderHeaderTabs() {
	rect := tabsRect.Align(layout.Right)

	for i := len(tabs) - 1; i >= 0; i-- {
		tab := blanc.NewText(tabs[i])

		if i == tabSelected {
			tab.SetStyle(headerStyle.Reverse(true))
		} else {
			tab.SetStyle(headerStyle)
		}

		rect = rect.Width(runewidth.StringWidth(tab.Text()))
		rect = rect.MoveLeft(rect.W)

		tab.Draw(screen, rect)
	}
}
