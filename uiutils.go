// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

// printLine prints text at the position (x, y).
func printLine(x, y int, text string, fg, bg int) {
	w, h := termbox.Size()

	if y < 0 || y >= h {
		return
	}

	cbuf := termbox.CellBuffer()
	i := 0
	for _, c := range text {
		if !strconv.IsPrint(c) {
			continue
		}
		if x+i < 0 || x+i >= w {
			continue
		}
		cbuf[y*w+x+i].Ch = c
		if fg >= 0 {
			cbuf[y*w+x+i].Fg = termbox.Attribute(fg)
		}
		if bg >= 0 {
			cbuf[y*w+x+i].Bg = termbox.Attribute(bg)
		}
		runeWidth := runewidth.RuneWidth(c)
		if runeWidth == 0 || (runeWidth == 2 && runewidth.IsAmbiguousWidth(c)) {
			runeWidth = 1
		}
		i += runeWidth
	}
}

func shortFrom(from string) string {
	fields := strings.Fields(from)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "<>\"'")
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
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
		return fmt.Sprintf("yest %02dh", date.Hour())
	case now.YearDay()-date.YearDay() < 7:
		return fmt.Sprintf("%s %dh", date.Weekday().String()[:2], date.Hour())
	}
	return date.Format("Jan 02")
}

//expandEnvHome expands environment viriables as well as ~/ in front of paths
func expandEnvHome(str string) string {
	if str[:2] == "~/" {
		str = strings.Replace(str, "~", "${HOME}", 1)
	}
	return os.ExpandEnv(str)
}
