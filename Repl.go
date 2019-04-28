package main

import (
	. "./term"
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
		prompt.OptionSuggestionBGColor(prompt.LightGray),
		prompt.OptionPreviewSuggestionTextColor(prompt.LightGray),
		prompt.OptionDescriptionBGColor(prompt.LightGray),
		prompt.OptionDescriptionTextColor(prompt.Black),
		prompt.OptionPreviewSuggestionTextColor(prompt.Black),
		prompt.OptionSelectedSuggestionBGColor(prompt.DarkGray),
		prompt.OptionSelectedSuggestionTextColor(prompt.Black),
		prompt.OptionSelectedDescriptionBGColor(prompt.DarkGray),
		prompt.OptionSelectedDescriptionTextColor(prompt.Black),
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
	Debug("Suggest", document.Text, document.GetWordBeforeCursor(), document.FindStartOfPreviousWord())

	if document.GetWordBeforeCursor() == "-" {
		return []prompt.Suggest{
			{
				Text:        "-f ",
				Description: "format",
			},
			{
				Text:        "-r ",
				Description: "repeat",
			},
			{
				Text:        "-d ",
				Description: "delay",
			},
		}
	}
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

func (executor *RedisExecutor) asyncExecute(cmd *RedisCommand, handleCancel func() bool) chan *RedisExecuteResult {
	ch1 := make(chan *RedisExecuteResult)
	go func() {
		ch := executor.Execute(cmd)
		running := true
		for counter := 0; running; counter++ {
			select {
			case resp := <-ch:
				Write("\033[2K\033[0G")
				if resp == nil {
					running = false
					ch1 <- nil
					return
				}
				ch1 <- resp
				<-ch1
				time.Sleep(time.Millisecond * 50)
			default:
				if handleCancel != nil && handleCancel() {
					running = false
					ch1 <- nil
					return
				}
				time.Sleep(time.Millisecond * 50)
				u := "/|-\\|"[counter%5]
				Write("\033[2K\033[0G" + string(u))
			}
		}
	}()
	return ch1
}
