package main

import (
	"strings"

	termbox "github.com/nsf/termbox-go"
)

type Prompt struct {
	text   []rune
	cursor int
}

func (p *Prompt) Active() bool {
	return p.text != nil
}

func (p *Prompt) Draw() {
	w, h := termbox.Size()
	printLine(0, h-1, ":"+string(p.text)+" ", -1, -1)
	termbox.SetCursor(p.cursor+1, h-1)
	if p.text == nil {
		termbox.HideCursor()
		printLine(0, h-1, " ", -1, -1)
	}
	for x := len(p.text) + 1; x < w; x++ {
		termbox.CellBuffer()[(h-1)*w+x].Ch = 0
	}
}

func (p *Prompt) putChar(ch rune) {
	tail := make([]rune, len(p.text)-p.cursor)
	copy(tail, p.text[p.cursor:])

	p.text = append(p.text[:p.cursor], ch)
	p.text = append(p.text, tail...)
	p.cursor++
}

func (p *Prompt) delChar() {
	if p.cursor > 0 {
		copy(p.text[p.cursor-1:], p.text[p.cursor:])
		p.text = p.text[:len(p.text)-1]
		p.cursor--
	}
}

func (p *Prompt) HandleEvent(e *termbox.Event) (string, []string) {
	if e.Ch == 0 {
		switch e.Key {
		case termbox.KeyEsc:
			p.text = nil
			p.Draw()
		case termbox.KeyEnter:
			fields := strings.Fields(string(p.text))
			p.text = nil
			p.Draw()
			if len(fields) > 0 {
				return fields[0], fields[1:]
			}
		case termbox.KeyBackspace2:
			p.delChar()
		case termbox.KeyDelete:
			if p.cursor < len(p.text) {
				p.cursor++
				p.delChar()
			}
		case termbox.KeyArrowLeft:
			p.cursor--
		case termbox.KeyArrowRight:
			p.cursor++
		case termbox.KeySpace:
			p.putChar(' ')
		}
	} else {
		p.putChar(e.Ch)
	}

	switch {
	case p.cursor < 0:
		p.cursor = 0
	case p.cursor > len(p.text):
		p.cursor = len(p.text)
	}
	p.Draw()
	return "", nil
}

func (p *Prompt) Activate(startstr string) {
	p.text = append(p.text, []rune(startstr+" ")...)
	if startstr == "" {
		p.text = p.text[:0]
	}
	p.cursor = len(p.text)
	p.Draw()
}
