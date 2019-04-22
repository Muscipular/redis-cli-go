// +build windows

package main

import (
	"errors"
	"fmt"
	"github.com/Azure/go-ansiterm"
	"github.com/Azure/go-ansiterm/winterm"
	"os"
	"syscall"
)

var ansiParser *ansiterm.AnsiParser = nil
var isInitial = false

func write(s string) {
	parser, e := getAnsiParser()
	if e != nil {
		fmt.Println(e)
		os.Exit(-1)
	}
	if parser != nil {
		_, _ = parser.Parse([]byte(s))
	} else {
		fmt.Print(s)
	}
}

func getAnsiParser() (*ansiterm.AnsiParser, error) {
	if !isInitial {
		isInitial = true
		handler := winterm.CreateWinEventHandler(os.Stdout.Fd(), os.Stdout)
		if handler == nil {
			lasterr := syscall.GetLastError()
			if lasterr == nil {
				lasterr = errors.New("terminal initial error")
			}
			return nil, lasterr
		}
		ansiParser = ansiterm.CreateParser("Ground", handler)
	}
	return ansiParser, nil
}
