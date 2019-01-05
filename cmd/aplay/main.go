package main

import (
	"io/ioutil"
	"os"

	"github.com/anisse/alsa"
)

func main() {
	p, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		panic(err.Error())
	}
	defer p.Close()

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err.Error())
	}
	_, err = p.Write(b)
	if err != nil {
		panic(err.Error())
	}
}
