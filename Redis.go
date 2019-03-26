package main

import (
	"crypto/tls"
	"fmt"
	"github.com/go-redis/redis"
	"reflect"
	"strings"
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
	go func() {
		args := ToInterface(StringArray(command.Args))
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
					switch strings.ToLower(command.Args[0]) {
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
			if count > 1 && delay > 0 {
				time.Sleep(delay)
			}
		}
		ch <- nil
	}()
	return ch
}

func (executor *RedisExecutor) AsyncExecute(cmd *RedisCommand) chan *RedisExecuteResult {
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
				break
			default:
				time.Sleep(time.Millisecond * 50)
				u := "/|-\\|"[counter%5]
				Write("\033[2K\033[0G" + string(u))
				break
			}
		}
	}()
	return ch1
}
