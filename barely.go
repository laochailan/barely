// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/notmuch/notmuch/bindings/go/src/notmuch"
	termbox "github.com/nsf/termbox-go"
)

const (
	Version   = "0.1"
	UserAgend = "barely/" + Version
)

func main() {
	showcfg := flag.Bool("config", false, "Print example config file.")
	flag.Parse()

	if *showcfg {
		fmt.Print(DefaultCfg)
		return
	}

	var buffers BufferStack
	var logbuf bytes.Buffer
	var err error

	log.SetOutput(&logbuf)
	rand.Seed(time.Now().Unix())

	LoadConfig()

	database, status :=
		notmuch.OpenDatabase(os.ExpandEnv(config.General.Database),
			0)
	if status != 0 {
		log.Fatal(status)
	}
	defer database.Close()

	err = termbox.Init()
	if err != nil {
		log.Fatal(err)
	}

	termbox.SetOutputMode(termbox.Output256)
	buffers.Init(database)

	for len(buffers.buffers) > 0 {
		termbox.Flush()
		event := termbox.PollEvent()
		buffers.HandleEvent(&event, database)
	}
	termbox.Close()
	if len(logbuf.Bytes()) != 0 {
		fmt.Println("Debug log:")
		fmt.Print(logbuf.String())
	}
}
