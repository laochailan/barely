package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"strings"
)

func readMail() {
	filename := "/home/laochailan/mail/nkirou1/INBOX/cur/1431188304_0.573.youmu,U=2072,FMD5=7e33429f656f1e6e9d79b29c3f82c57e:2,S"

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}

	msg, err := mail.ReadMessage(file)
	if err != nil {
		log.Fatalln(err)
	}

	for k, _ := range msg.Header {
		fmt.Println(k, "\t", msg.Header.Get(k))
	}
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
			fmt.Printf("Part: %q, %s\n", p.Header, slurp)
		}
	}

}
