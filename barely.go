// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/notmuch/notmuch/bindings/go/src/notmuch"
	termbox "github.com/nsf/termbox-go"
)

func main() {
	var buffers BufferStack
	logfile, err := os.Create("barely.log")
	if err != nil {
		log.Printf("Warning: Could not open log file")
	} else {
		log.SetOutput(logfile)
	}
	rand.Seed(time.Now().Unix())

	LoadConfig()

	database, status := notmuch.OpenDatabase(os.ExpandEnv(config.General.Database), notmuch.DATABASE_MODE_READ_ONLY)
	if status != 0 {
		log.Fatal(status)
	}
	defer database.Close()

	err = termbox.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	termbox.SetOutputMode(termbox.Output256)
	buffers.Init(database)

	for len(buffers.buffers) > 0 {
		termbox.Flush()
		event := termbox.PollEvent()
		buffers.HandleEvent(&event, database)
	}

}
