package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

// printLineClr prints text at the position (x, y).
func printLine(x, y int, text string, fg, bg int) {
	w, h := termbox.Size()

	if y < 0 || y >= h {
		return
	}

	cbuf := termbox.CellBuffer()
	i := 0
	for _, c := range text {
		if x+i < 0 || x+i >= w {
			return
		}
		cbuf[y*w+x+i].Ch = c
		if fg >= 0 {
			cbuf[y*w+x+i].Fg = termbox.Attribute(fg)
		}
		if bg >= 0 {
			cbuf[y*w+x+i].Bg = termbox.Attribute(bg)
		}
		i++
	}
}

func printStatus(text string) {
	_, h := termbox.Size()
	printLine(1, h-2, text, -1, -1)

}

func shortFrom(from string) string {
	fields := strings.Fields(from)
	return strings.Trim(fields[0], "<>\"'")
}

func shortTime(date time.Time) string {
	now := time.Now()
	dur := time.Since(date)
	switch {
	case dur < time.Minute:
		return "just now"
	case dur < time.Hour:
		return fmt.Sprintf("%dmin", int(dur.Minutes()))
	case date.Year() != now.Year():
		return date.Format("Jan 2006")
	case date.YearDay() == now.YearDay():
		return date.Format("15:04")
	case now.YearDay()-date.YearDay() == 1:
		return fmt.Sprintf("yest %2dh", date.Hour())
	case now.YearDay()-date.YearDay() < 7:
		return date.Format("Mo 15h")
	}
	return date.Format("Jan 02")
}
