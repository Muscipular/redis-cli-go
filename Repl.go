package main

import (
	//"fmt"
	"github.com/c-bata/go-prompt"
	"os"
	"reflect"
	"unsafe"
	//. "strings"
	//"time"
	//"github.com/briandowns/spinner"
)

type REPL struct {
	redis   *RedisExecutor
	canExit bool
	prompt  string
	history *prompt.History
}

func NewREPL(redis *RedisExecutor) *REPL {
	return &REPL{
		canExit: false,
		redis:   redis,
		prompt:  "> ",
	}
}

func (repl *REPL) execute(input string) {
	repl.canExit = false
	if len(input) == 0 {
		return
	}
	ss := parseArg(input)
	opt, ss := GetCmdOpt(ss)
	//Debug("Repl", "cmd: ", ss, opt)
	cmd := &RedisCommand{
		Args:   ss,
		Option: opt,
	}
	executor := repl.redis
	ch := executor.AsyncExecute(cmd)
	for true {
		resp := <-ch
		if resp == nil {
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
	prompt.New(func(s string) { repl.execute(s) }, func(document prompt.Document) []prompt.Suggest { return repl.suggest(document) },
		prompt.OptionLivePrefix(repl.getPromptText),
		prompt.OptionTitle("redis cli"),
		prompt.OptionAddKeyBind(
			bindKey(prompt.ControlC, repl.handleControlC),
			bindKey(prompt.Escape, repl.handleEsc),
		),
		func(p *prompt.Prompt) error {
			pointer := reflect.ValueOf(*p).Field(4).Pointer()
			repl.history = (*prompt.History)(unsafe.Pointer(pointer))
			repl.history.Clear()
			return nil
		},
	).Run()
	return nil
}
