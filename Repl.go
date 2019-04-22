package main

import (
	//"fmt"
	"github.com/c-bata/go-prompt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
	"unsafe"
	//. "strings"
	//"time"
	//"github.com/briandowns/spinner"
)

type REPL struct {
	redis          *RedisExecutor
	canExit        bool
	prompt         string
	history        *prompt.History
	promptInstance *prompt.Prompt
	in             prompt.ConsoleParser
}

func NewREPL(redis *RedisExecutor) *REPL {
	i := &REPL{
		canExit: false,
		redis:   redis,
		prompt:  "> ",
	}
	i.promptInstance = prompt.New(func(s string) { i.execute(s) }, func(document prompt.Document) []prompt.Suggest { return i.suggest(document) },
		prompt.OptionLivePrefix(i.getPromptText),
		prompt.OptionTitle("redis cli"),
		prompt.OptionAddKeyBind(
			bindKey(prompt.ControlC, i.handleControlC),
			bindKey(prompt.Escape, i.handleEsc),
		),
		func(p *prompt.Prompt) error {
			pointer := reflect.ValueOf(*p).Field(4).Pointer()
			i.history = (*prompt.History)(unsafe.Pointer(pointer))
			i.history.Clear()
			return nil
		},
	)
	return i
}

func (repl *REPL) execute(input string) {
	repl.canExit = false
	if len(input) == 0 {
		return
	}
	ss := parseArg(input)
	opt, ss := GetCmdOpt(ss)
	Debug("Repl", "cmd: ", ss, opt)
	cmd := &RedisCommand{
		Args:   ss,
		Option: opt,
	}
	executor := repl.redis
	sch := make(chan os.Signal, 1)
	signal.Notify(sch,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	ch := executor.asyncExecute(cmd, func() bool {
		select {
		case <-sch:
			signal.Stop(sch)
			WriteLn("\033[2K\033[0GCancel")
			return true
		default:
		}
		return false
	})
	for true {
		resp := <-ch
		if resp == nil {
			time.Sleep(time.Millisecond * 50)
			break
		}
		if resp.Error != nil {
			WriteLn(resp.Error)
		} else {
			resp.Format(opt.FormatType, false)
		}
		ch <- nil
	}
	repl.history.Add(input)
}

func (repl *REPL) suggest(document prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{}
}

func bindKey(key prompt.Key, fn func(buffer *prompt.Buffer)) prompt.KeyBind {
	return prompt.KeyBind{Key: key, Fn: fn}
}

func (repl *REPL) handleControlC(buffer *prompt.Buffer) {
	if repl.canExit {
		WriteLn("\033[2K\033[0GExit, bye.")
		os.Exit(0)
	} else {
		WriteLn("\033[2K\033[0GTo exit, press \033[31mCtrl + C\033[0m again.")
		repl.canExit = true
	}
}

func (repl *REPL) handleEsc(buffer *prompt.Buffer) {
	buffer.DeleteBeforeCursor(buffer.DisplayCursorPosition())
	buffer.Delete(len(buffer.Text()))
}

func (repl *REPL) getPromptText() (string, bool) {
	return repl.prompt, true
}

func (repl *REPL) run() error {
	c, e := repl.redis.client.Ping().Result()
	if e != nil || c != "PONG" {
		//WriteLn(e, c)
		return e
	}
	repl.promptInstance.Run()
	return nil
}
