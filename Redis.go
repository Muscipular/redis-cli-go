package main

import (
	"crypto/tls"
	"fmt"
	"github.com/go-redis/redis"
	"reflect"
	. "strings"
	"time"
)

type RedisHostOption struct {
	Host     string `short:"h" long:"host" description:"redis host"`
	Port     int    `short:"p" long:"port" description:"redis port"`
	Socket   string `long:"socket" description:"unix socket file"`
	Password string `short:"a" long:"password" description:"password"`
	Database int    `short:"n" long:"database" description:"default db"`
	Ssl      bool   `long:"ssl" description:"enable ssl"`
	SslHost  string `long:"ssl-host" description:"ssl host"`
	Cluster  bool   `short:"c" long:"cluster" description:"enable cluster mode"`
}

type RedisExecutor struct {
	Option *RedisHostOption
	client *redis.Client
}

type EnumFormatType int8

const (
	FormatNormal    EnumFormatType = 0
	FormatJson      EnumFormatType = 1
	FormatRawString EnumFormatType = 2
)

func (f EnumFormatType) String() string {
	switch f {
	case FormatRawString:
		return "raw"
	case FormatJson:
		return "json"
	default:
		return "normal"
	}
}

type RedisCommand struct {
	Args   []string
	Option *RedisCommandOption
}

type RedisCommandOption struct {
	FormatType             EnumFormatType
	Repeat                 uint
	Delay                  float32
	RunAtEachNode          bool
	SplitResultForEachNode bool
}

func NewRedisExecutor(opt *RedisHostOption) *RedisExecutor {
	var tlsOption *tls.Config = nil
	if opt.Ssl {
		tlsOption = &tls.Config{
			ServerName: If(len(opt.SslHost) > 0, opt.SslHost, "").(string),
		}
	}
	client := redis.NewClient(&redis.Options{
		Addr:        If(len(opt.Socket) > 0, opt.Socket, fmt.Sprintf("%s:%d", opt.Host, If(opt.Port <= 0, 6379, opt.Port))).(string),
		Network:     If(len(opt.Socket) > 0, "unix", "tcp").(string),
		DialTimeout: time.Second * 5,
		ReadTimeout: time.Second * 10,
		TLSConfig:   tlsOption,
		Password:    opt.Password,
		DB:          opt.Database,
	})
	return &RedisExecutor{
		Option: opt,
		client: client,
	}
}

type RedisExecuteResult struct {
	Value *interface{}
	Host  *RedisHostOption
	Cmd   *RedisCommand
	Error error
}

func (executor *RedisExecutor) Execute(command *RedisCommand) chan *RedisExecuteResult {
	ch := make(chan *RedisExecuteResult)
	if ToLower(command.Args[0]) == "info" {
		command.Option.FormatType = FormatRawString
	}
	go func() {
		count := 1
		if command.Option != nil && command.Option.Repeat > 1 {
			count = int(command.Option.Repeat)
		}
		delay := time.Second * 0
		if count > 1 && (command.Option == nil || command.Option.Delay > 0) {
			i := int64(command.Option.Delay * 1000)
			delay = time.Duration(int64(time.Millisecond) * i)
		}
		for ; count > 0; count-- {
			fn, ok := cMap[ToLower(command.Args[0])]
			if ok {
				fn(executor, ch, command)
			} else {
				executor.coreExecute(ch, command)
			}
			if count > 1 && delay > 0 {
				time.Sleep(delay)
			}
		}
		ch <- nil
	}()
	return ch
}

var cMap = map[string]func(*RedisExecutor, chan *RedisExecuteResult, *RedisCommand){
	"getmatch": func(executor *RedisExecutor, ch chan *RedisExecuteResult, cmd *RedisCommand) {
		keys := executor.client.Keys(cmd.Args[1])
		val := keys.Val()
		mmm := map[string]interface{}{}
		for _, value := range val {
			t := executor.client.Type(value).Val()
			switch t {
			case "hash":
				mmm[value] = executor.client.HGetAll(value).Val()
			case "list":
				mmm[value] = executor.client.LRange(value, 0, -1).Val()
			case "string":
				mmm[value] = executor.client.Get(value).Val()
			default:
				mmm[value] = t
			}
		}
		var k interface{} = mmm
		ch <- &RedisExecuteResult{
			Error: nil,
			Value: &k,
			Host:  executor.Option,
			Cmd:   cmd,
		}
	},
	"delall": func(executor *RedisExecutor, ch chan *RedisExecuteResult, cmd *RedisCommand) {
		keys := executor.client.Keys(cmd.Args[1])
		val := keys.Val()
		if executor.Option.Cluster {
			for _, value := range val {
				executor.client.Del(value)
			}
		} else {
			for i := 0; i < len(val); i += 100 {
				o := i + 100
				if o > len(val) {
					o = len(val)
				}
				executor.client.Del(val[i:o]...)
			}
		}
		var k interface{} = val
		ch <- &RedisExecuteResult{
			Error: nil,
			Value: &k,
			Host:  executor.Option,
			Cmd:   cmd,
		}
	},
}

func (executor *RedisExecutor) coreExecute(ch chan *RedisExecuteResult, command *RedisCommand) {
	args := ToInterface(StringArray(command.Args))
	cmd := redis.NewCmd(args...)
	err := executor.client.Process(cmd)
	if err != nil && err != redis.Nil {
		ch <- &RedisExecuteResult{
			Value: nil,
			Error: err,
			Host:  executor.Option,
			Cmd:   command,
		}
	} else {
		result := cmd.Val()
		if result != nil {
			switch ToLower(command.Args[0]) {
			case "hgetall":
				list := reflect.ValueOf(result)
				_len := list.Len()
				m := make(map[string]interface{})
				for i := 0; i < _len; i += 2 {
					m[list.Index(i).Interface().(string)] = list.Index(i + 1).Interface()
				}
				result = m
			}
		}
		//WriteLn(result, err)
		ch <- &RedisExecuteResult{
			Error: nil,
			Value: &result,
			Host:  executor.Option,
			Cmd:   command,
		}
	}
}

func (executor *RedisExecutor) AsyncExecute(cmd *RedisCommand, handleCancel func() bool) chan *RedisExecuteResult {
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
