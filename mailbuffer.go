package main

import (
	"mime"

	termbox "github.com/nsf/termbox-go"
)

// MailBuffer displays mails and allows replying to them
type MailBuffer struct {
	mail   *Mail
	cursor int
}

func NewMailBuffer(filename string) *MailBuffer {
	buf := new(MailBuffer)
	var err error
	buf.mail, err = readMail(filename)
	if err != nil {
		StatusLine = err.Error()
		buf.mail = new(Mail)
	}
	buf.cursor = mbHeaderHeight

	return buf
}

const mbHeaderHeight = 5 // lines occupied by the header field

func (b *MailBuffer) drawPlain(y *int, text string) {
	x := 0
	w, h := termbox.Size()
	cbuf := termbox.CellBuffer()
	for _, ch := range text {
		if ch == '\n' {
			for ; *y >= mbHeaderHeight && *y < h-2 && x < w; x++ {
				cbuf[*y*w+x] = termbox.Cell{0, 0, 0}
			}
			(*y)++
			x = 0
			continue
		}
		if x >= w {
			(*y)++
			x = 0
		}
		if *y < mbHeaderHeight {
			continue
		}
		if *y >= h-2 {
			break
		}
		cbuf[*y*w+x].Ch = ch
		cbuf[*y*w+x].Fg = 0
		cbuf[*y*w+x].Bg = 0
		x++
	}

	cursor := b.cursor
	if cursor >= h*3/4 {
		cursor = h * 3 / 4
	}

	if cursor >= mbHeaderHeight && cursor < h-2 {
		for x := 0; x < w; x++ {
			cbuf[cursor*w+x].Bg = termbox.Attribute(config.Theme.HlBg)
		}
	}
}

func (b *MailBuffer) drawHeader() {
	printLine(0, 0, "| Date:", config.Theme.Subject|int(termbox.AttrBold), -1)
	printLine(0, 1, "| From:", config.Theme.Subject|int(termbox.AttrBold), -1)
	printLine(0, 2, "| To:", config.Theme.Subject|int(termbox.AttrBold), -1)
	printLine(0, 3, "| Subject:", config.Theme.Subject|int(termbox.AttrBold), -1)
	printLine(0, 0, "| Date: "+b.mail.Header.Get("Date"), -1, -1)
	printLine(0, 1, "| From: "+b.mail.Header.Get("From"), -1, -1)
	printLine(0, 2, "| To: "+b.mail.Header.Get("To"), -1, -1)
	printLine(0, 3, "| Subject: "+b.mail.Header.Get("Subject"), -1, -1)
}

func (b *MailBuffer) Draw() {
	w, h := termbox.Size()
	cbuf := termbox.CellBuffer()

	b.drawHeader()
	offset := 0
	if b.cursor >= h*3/4 {
		offset = -h*3/4 + b.cursor
	}

	y := mbHeaderHeight - offset

	for _, part := range b.mail.Parts {
		contentType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			continue
		}
		if y >= h-2 {
			break
		}
		if y >= mbHeaderHeight {
			printLine(0, y, "-- "+contentType+" --", config.Theme.Date, 0)
			for x := len(contentType) + 6; x < w; x++ {
				cbuf[y*w+x] = termbox.Cell{0, 0, 0}
			}
		}
		y++
		if contentType[:4] == "text" {
			b.drawPlain(&y, part.Body)
		}
	}

	if y < mbHeaderHeight {
		y = mbHeaderHeight
	}
	for ; y < h-2; y++ {
		for x := 0; x < w; x++ {
			cbuf[y*w+x] = termbox.Cell{0, 0, 0}
		}
	}

}

func (b *MailBuffer) Title() string {
	return "reading " + b.mail.Header.Get("Message-ID")
}

func (b *MailBuffer) Name() string {
	return "mail"
}

func (b *MailBuffer) Close() {
}

func (b *MailBuffer) HandleCommand(cmd string, args []string, stack *BufferStack) {
	switch cmd {
	case "move":
		if len(args) == 0 {
			break
		}
		_, h := termbox.Size()
		switch args[0] {
		case "up":
			b.cursor--
		case "down":
			b.cursor++
		case "pageup":
			b.cursor -= h / 2
		case "pagedown":
			b.cursor += h / 2
		}
		if b.cursor < mbHeaderHeight {
			b.cursor = mbHeaderHeight
		}
		b.Draw()
	}

}
