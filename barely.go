// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

// barely is a notmuch-frontend inspired by alot.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	termbox "github.com/nsf/termbox-go"
)

const (
	// Version is the version number of barely.
	Version = "0.1"

	// UserAgent is the User Agent string attached to mail messages.
	UserAgent = "barely/" + Version

	// StderrLogFile is a file in the temporary files directory
	// where Xapians messy stderr output gets redirected to.
	StderrLogFile = "barely.log"
)

var logbuf bytes.Buffer

// redirect panics to stdout
func recoverPanic() {
	if r := recover(); r != nil {
		termbox.Close()
		fmt.Println(r)
		buf := make([]byte, 2048)
		l := runtime.Stack(buf, true)
		fmt.Println(string(buf[:l]))
	}
	if len(logbuf.Bytes()) != 0 {
		fmt.Println("Debug log:")
		fmt.Print(logbuf.String())
	}
}

func main() {
	defer recoverPanic()

	showcfg := flag.Bool("config", false, "Print example config file.")
	flag.Parse()

	if *showcfg {
		fmt.Print(DefaultCfg)
		return
	}

	var buffers BufferStack
	var err error

	stderrFile, err := os.Create(os.TempDir() + "/" + StderrLogFile)
	if err == nil {
		os.Stderr = stderrFile
	} else {
		fmt.Println(err)
	}

	log.SetOutput(&logbuf)
	rand.Seed(time.Now().Unix())

	LoadConfig()

	err = termbox.Init()
	if err != nil {
		log.Fatal(err)
	}

	termbox.SetOutputMode(termbox.Output256)
	buffers.Init()

	for len(buffers.buffers) > 0 {
		termbox.Flush()
		event := termbox.PollEvent()
		buffers.HandleEvent(&event)
	}
	termbox.Close()

	// remove the stderrFile if it is empty
	if stderrFile != nil {
		stderrFile.Sync()
		info, err := stderrFile.Stat()
		stderrFile.Close()
		if err == nil && info.Size() == 0 {
			os.Remove(stderrFile.Name())
		}
	}

}
