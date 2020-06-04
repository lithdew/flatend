package main

import (
	"github.com/gdamore/tcell"
	"github.com/lithdew/blanc"
	"github.com/lithdew/blanc/layout"
	"github.com/mattn/go-runewidth"
	"strconv"
)

var bodyStyle tcell.Style
var bodyRect layout.Rect

type ListItem struct {
	ID     int
	Type   string
	Status string
	Params string
}

var list struct {
	items    []ListItem
	selected int

	idRect     layout.Rect
	typeRect   layout.Rect
	statusRect layout.Rect
	paramsRect layout.Rect
}

func initBody() {
	bodyStyle = tcell.StyleDefault
	bodyStyle = bodyStyle.Background(tcell.ColorBlack)
	bodyStyle = bodyStyle.Foreground(tcell.ColorWhite)

	// example data

	list.items = []ListItem{
		{ID: 0, Type: "http", Status: "ready", Params: "9000"},
		{ID: 1, Type: "post", Status: "ready", Params: "/post/new"},
		{ID: 2, Type: "get", Status: "ready", Params: "/post/:id"},
		{ID: 3, Type: "jsonrpc", Status: "ready", Params: "hello_world"},
		{ID: 4, Type: "cron", Status: "ready", Params: "@every 10s"},
		{ID: 5, Type: "sql", Status: "ready", Params: "select * from posts where id = :id"},
		{ID: 6, Type: "csv", Status: "ready", Params: "/home/kenta/Desktop/database.csv"},
	}

	list.selected = 4
}

func resizeBody() {
	bodyRect = screenRect.PadVertical(1)

	for i := 0; i < len(list.items); i++ {
		item := list.items[i]

		if idWidth := runewidth.StringWidth(strconv.FormatInt(int64(item.ID), 10)); list.idRect.W < idWidth {
			list.idRect.W = idWidth
		}

		if typeWidth := runewidth.StringWidth(item.Type); list.typeRect.W < typeWidth {
			list.typeRect.W = typeWidth
		}

		if statusWidth := runewidth.StringWidth(item.Status); list.statusRect.W < statusWidth {
			list.statusRect.W = statusWidth
		}

		if paramsWidth := runewidth.StringWidth(item.Params); list.paramsRect.W < paramsWidth {
			list.paramsRect.W = paramsWidth
		}
	}

	switch {
	case list.idRect.W > 8:
		list.idRect.W = 8
	case list.idRect.W < 6:
		list.idRect.W = 6
	}

	switch {
	case list.typeRect.W > 16:
		list.typeRect.W = 16
	case list.typeRect.W < 10:
		list.typeRect.W = 10
	}

	switch {
	case list.statusRect.W > 16:
		list.statusRect.W = 16
	case list.statusRect.W < 10:
		list.statusRect.W = 10
	}

	switch {
	case list.paramsRect.W < 16:
		list.paramsRect.W = 16
	}

	list.idRect.X = 2
	list.typeRect = list.typeRect.AlignTo(list.idRect, layout.Right)
	list.statusRect = list.statusRect.AlignTo(list.typeRect, layout.Right)
	list.paramsRect = list.paramsRect.AlignTo(list.statusRect, layout.Right)

	list.idRect.Y = bodyRect.Y + 1
	list.typeRect.Y = bodyRect.Y + 1
	list.statusRect.Y = bodyRect.Y + 1
	list.paramsRect.Y = bodyRect.Y + 1
}

func renderBody() {
	blanc.Clear(screen, bodyStyle, bodyRect)

	var col blanc.Text
	col.SetStyle(bodyStyle.Underline(true))

	col.SetText("ID")
	col.Draw(screen, list.idRect)

	col.SetText("TYPE")
	col.Draw(screen, list.typeRect)

	col.SetText("STATUS")
	col.Draw(screen, list.statusRect)

	col.SetText("PARAMS")
	col.Draw(screen, list.paramsRect)

	for i := 0; i < len(list.items); i++ {
		item := list.items[i]

		if i == list.selected {
			selectedStyle := tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorBlack)
			blanc.Clear(screen, selectedStyle, layout.Rect{X: 0, Y: bodyRect.Y + 1 + i + 1, W: screenRect.W, H: 1})
			col.SetStyle(selectedStyle)
		} else {
			col.SetStyle(bodyStyle)
		}

		col.SetText(strconv.FormatInt(int64(item.ID), 10))
		col.Draw(screen, list.idRect.MoveDown(i+1))

		col.SetText(item.Type)
		col.Draw(screen, list.typeRect.MoveDown(i+1))

		col.SetText(item.Status)
		col.Draw(screen, list.statusRect.MoveDown(i+1))

		col.SetText(item.Params)
		col.Draw(screen, list.paramsRect.MoveDown(i+1))
	}
}
