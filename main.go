package main

import (
	. "./term"
	"fmt"
	"github.com/eiannone/keyboard"

	//"github.com/gdamore/tcell"
	"os"
)

var _redis *RedisExecutor = nil

func main() {
	//fmt.Println("Please select table.")
	//repl()
	//WriteLn("redis cli")
	eee := keyboard.Open()
	if eee != nil {
		fmt.Println(eee)
	}
	defer keyboard.Close()
	for {
		ch, key, eee := keyboard.GetKey()
		switch key {
		case keyboard.KeyCtrlC:
			fmt.Println(ch, key, eee, "End")
			os.Exit(0)
		}
		fmt.Println(ch, key, eee)
	}
	return

	args := os.Args[1:]
	Debug("main", args)
	e, opt, cmds := GetHostOpt(args)
	if e != nil {
		WriteLn(e)
		os.Exit(1)
		return
	}
	_redis = NewRedisExecutor(opt)
	if len(cmds) > 0 {
		option, cmds := GetCmdOpt(cmds)
		Debug("main", opt, option, cmds)
		command := &RedisCommand{
			Args:   cmds,
			Option: option,
		}
		ch := _redis.asyncExecute(command, nil)

		for true {
			resp := <-ch
			if resp == nil {
				break
			}
			if resp.Error != nil {
				Debug("main", resp.Error)
				WriteLn(resp.Error)
			} else {
				resp.Format(command.Option.FormatType, false)
			}
			ch <- nil
		}
	} else {
		repl()
	}
	//GetOpt(parseArg("-d 11 ee aaaaa  bbb ccc 哈哈哈 欧克，asd \"aaa\\\\ bbb\"bb -f json -d 11 -r 10 -s -e -kk -es a ccccccc"))
}

func repl() {
	repl := NewREPL(_redis)
	err := repl.run()
	if err != nil {
		fmt.Println(err)
		os.Exit(-2)
	}
}
