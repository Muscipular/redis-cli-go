// +build windows

package main

import (
	"errors"
	"fmt"
	"github.com/Azure/go-ansiterm"
	"github.com/Azure/go-ansiterm/winterm"
	"os"
	"runtime"
	"syscall"
)

var f *ansiterm.AnsiParser = nil
var isWindows = runtime.GOOS == "windows"
var isInitial = false

func write(s string) {
	if isWindows {
		parser, e := getAnsiParser()
		if e != nil {
			fmt.Println(e)
			os.Exit(-1)
		}
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
		f = ansiterm.CreateParser("Ground", handler)
	}
	return f, nil
}
