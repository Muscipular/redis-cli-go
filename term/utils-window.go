// +build windows

package term

import (
	"errors"
	"github.com/Azure/go-ansiterm"
	"github.com/Azure/go-ansiterm/winterm"
	"os"
	"syscall"
)

type termWin struct {
	ansiParser *ansiterm.AnsiParser
	handler    ansiterm.AnsiEventHandler
}

func (tm *Term) _platformInit() error {
	handler := winterm.CreateWinEventHandler(os.Stdout.Fd(), os.Stdout)
	if handler == nil {
		lastErr := syscall.GetLastError()
		if lastErr == nil {
			lastErr = errors.New("terminal initial error")
		}
		return lastErr
	}
	ansiParser := ansiterm.CreateParser("Ground", handler)
	var win interface{} = termWin{
		ansiParser: ansiParser,
		handler:    handler,
	}
	tm.platformObj = &win
	return nil
}
