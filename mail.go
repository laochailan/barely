package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"strings"
)

type Part struct {
	Header textproto.MIMEHeader
	Body   string
}

type Mail struct {
	Header mail.Header
	Parts  []Part
}

// reads a mail and parses it into decoded parts
func readMail(filename string) *Mail {
	m := new(Mail)

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}

	msg, err := mail.ReadMessage(file)
	if err != nil {
		log.Fatalln(err)
	}

	m.Header = msg.Header

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(msg.Body, params["boundary"])
		fmt.Println(params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				log.Fatal(err)
			}
			m.Parts = append(m.Parts, Part{p.Header, string(slurp)})
		}
	} else {
		log.Println("handle this!")
	}

	return m
}
