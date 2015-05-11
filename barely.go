package main

import (
	"log"
	"os"

	"github.com/notmuch/notmuch/bindings/go/src/notmuch"
	termbox "github.com/nsf/termbox-go"
)

func main() {
	var buffers BufferStack

	LoadConfig()

	database, status := notmuch.OpenDatabase(os.ExpandEnv(config.General.Database), notmuch.DATABASE_MODE_READ_ONLY)
	if status != 0 {
		log.Fatal(status)
	}
	defer database.Close()

	err := termbox.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	termbox.SetOutputMode(termbox.Output256)

	buffers.Push(NewSearchBuffer("Nandita", database))
	for len(buffers.buffers) > 0 {
		termbox.Flush()
		event := termbox.PollEvent()
		buffers.HandleEvent(&event, database)
	}

}
