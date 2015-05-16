// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"os/exec"
	"strings"

	termbox "github.com/nsf/termbox-go"
	qp "gopkg.in/alexcesaro/quotedprintable.v2"
)

// MailBuffer displays mails and allows replying to them
type MailBuffer struct {
	filename string
	mail     *Mail
	cursor   int

	buffer    []termbox.Cell
	partLines []int

	tmpDir string
}

func NewMailBuffer(filename string) *MailBuffer {
	m, err := readMail(filename)
	if err != nil {
		StatusLine = err.Error()
		m = new(Mail)
	}

	buf := NewMailBufferFromMail(m)
	buf.filename = filename
	return buf
}

func NewMailBufferFromMail(m *Mail) *MailBuffer {
	buf := new(MailBuffer)
	var err error
	buf.mail = m

	buf.partLines = make([]int, len(buf.mail.Parts))
	buf.cursor = 0
	buf.refreshBuf()

	buf.tmpDir, err = ioutil.TempDir("", "barely")
	if err != nil {
		StatusLine = "Could not open TempDir: " + err.Error()
	}

	return buf

}

const mbHeaderHeight = 5 // lines occupied by the header field

func formatPlain(buf []termbox.Cell, y, w int, text string) ([]termbox.Cell, int) {
	line := make([]termbox.Cell, w)
	x := 0
	buf = append(buf, line...)
	fg := termbox.Attribute(0)
	for _, ch := range text {
		if x == 0 {
			if ch == '>' {
				fg = termbox.Attribute(config.Theme.Quote)
			} else {
				fg = 0
			}
		}
		if ch == '\n' {
			for ; x < w; x++ {
				buf[y*w+x] = termbox.Cell{0, 0, 0}
			}
			y++
			x = 0
			buf = append(buf, line...)
			continue
		}
		if x >= w {
			y++
			buf = append(buf, line...)
			x = 0
		}

		buf[y*w+x] = termbox.Cell{ch, fg, 0}
		x++
	}

	for ; x < w; x++ {
		buf[y*w+x] = termbox.Cell{0, 0, 0}
	}
	y++
	return buf, y
}

// refreshBuf preformats the whole mail so that redrawing it while scrolling is faster.
func (b *MailBuffer) refreshBuf() {
	w, _ := termbox.Size()
	b.buffer = b.buffer[:0]
	b.partLines = make([]int, len(b.mail.Parts))
	line := make([]termbox.Cell, w)
	y := 0
	for i, part := range b.mail.Parts {
		x := 0
		b.buffer = append(b.buffer, line...)

		contentType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			continue
		}
		contentStr := contentType
		if name := attachmentName(&part); name != "" {
			contentStr += ": \"" + name + "\""
		}

		b.partLines[i] = y
		str := []rune("-- " + contentStr + " --")
		for ; x < min(w, len(str)); x++ {
			b.buffer[y*w+x] = termbox.Cell{str[x], termbox.Attribute(config.Theme.Date), 0}
		}
		for ; x < w; x++ {
			b.buffer[y*w+x] = termbox.Cell{0, 0, 0}
		}
		y++

		if contentType == "text/plain" {
			b.buffer, y = formatPlain(b.buffer, y, w, part.Body)
		}
	}
	if b.cursor >= len(b.buffer)/w {
		b.cursor = len(b.buffer)/w - 1
	}
}

func (b *MailBuffer) drawHeader() {
	getHeader := func(key string) string {
		str, _, err := qp.DecodeHeader(b.mail.Header.Get(key)) // ignore charset for now
		if err != nil {
			str = err.Error()
		}
		return str
	}
	drawField := func(y int, label, value string) {
		printLine(0, y, "| "+label+": ", config.Theme.Subject|int(termbox.AttrBold), -1)
		printLine(len(label)+4, y, value, -1, -1)
	}

	drawField(0, "Date", getHeader("Date"))
	drawField(1, "From", getHeader("From"))
	drawField(2, "To", getHeader("To"))
	drawField(3, "Subject", getHeader("Subject"))
}

func (b *MailBuffer) Draw() {
	w, h := termbox.Size()
	cbuf := termbox.CellBuffer()

	b.drawHeader()
	offset := 0
	if b.cursor >= h*3/4 {
		offset = -h*3/4 + b.cursor
	}

	y := 0
	for ; y < min(len(b.buffer)/w-offset, h-2-mbHeaderHeight); y++ {
		for x := 0; x < w; x++ {
			cbuf[(y+mbHeaderHeight)*w+x] = b.buffer[(y+offset)*w+x]
		}
	}

	for ; y < h-2-mbHeaderHeight; y++ {
		for x := 0; x < w; x++ {
			cbuf[(y+mbHeaderHeight)*w+x] = termbox.Cell{0, 0, 0}
		}
	}

	if b.cursor-offset >= 0 && b.cursor-offset < h-2-mbHeaderHeight {
		for x := 0; x < w; x++ {
			cbuf[(b.cursor-offset+mbHeaderHeight)*w+x].Bg = termbox.Attribute(config.Theme.HlBg)
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
	os.RemoveAll(b.tmpDir)
}

func attachmentName(p *Part) string {
	_, params, err := mime.ParseMediaType(p.Header.Get("Content-Type"))
	if err != nil {
		return ""
	}
	if name := params["name"]; name != "" {
		return name
	}
	_, params, err = mime.ParseMediaType(p.Header.Get("Content-Disposition"))
	if err != nil {
		return ""
	}
	return params["filename"]
}

func openAttachment(p *Part, dir string) {
	name := attachmentName(p)
	if name == "" {
		contentType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		idx := strings.Index(contentType, "/")
		if idx == -1 {
			idx = 0
		}

		name = "attachment." + contentType[min(len(contentType), idx+1):]
	}

	filename := dir + "/" + name
	file, err := os.Create(filename)
	if err != nil {
		StatusLine = err.Error()
		return
	}
	file.Write([]byte(p.Body))
	file.Close()

	cmd := exec.Command(config.Commands.Attachments, filename)
	err = cmd.Start()
	if err != nil {
		StatusLine = err.Error()
	}
	go cmd.Wait()
}

func (b *MailBuffer) HandleCommand(cmd string, args []string, stack *BufferStack) bool {
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
		if b.cursor < 0 {
			b.cursor = 0
		}
		w, _ := termbox.Size()
		if b.cursor >= len(b.buffer)/w {
			b.cursor = len(b.buffer)/w - 1
		}
		b.Draw()
	case "resize":
		b.refreshBuf()
	case "show":
		for i, l := range b.partLines {
			if b.cursor == l {
				openAttachment(&b.mail.Parts[i], b.tmpDir)
				break
			}
		}
	case "raw":
		termbox.Close()
		cmd := exec.Command(config.Commands.Editor, b.filename)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
		termbox.Init()
		stack.refresh()
	case "reply":
		reply := composeReply(b.mail)
		stack.Push(NewComposeBuffer(reply))
	default:
		return false
	}
	return true

}
