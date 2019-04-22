package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"reflect"
	"strconv"
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
	//Cluster  bool   `short:"c" long:"cluster" description:"enable cluster mode(not finish yet.)"`
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
	FormatType EnumFormatType
	Repeat     uint
	Delay      float32
	//RunAtEachNode          bool
	//SplitResultForEachNode bool
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
				result, err := fn(executor, command)
				ch <- &RedisExecuteResult{
					Error: err,
					Value: result,
					Cmd:   command,
					Host:  executor.Option,
				}
			} else {
				result, err := executor.coreExecute(command)
				ch <- &RedisExecuteResult{
					Error: err,
					Value: result,
					Cmd:   command,
					Host:  executor.Option,
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

func _ScanAll(executor *RedisExecutor, cmd *RedisCommand) (*interface{}, error) {
	if len(cmd.Args) < 2 {
		return nil, errors.New("ScanAll argument incorrect")
	}
	count := int64(-1)
	for ix := 0; ix < len(cmd.Args); ix++ {
		value := cmd.Args[ix]
		if ToUpper(value) == "COUNT" && ix+1 < len(cmd.Args) {
			i, e := strconv.Atoi(cmd.Args[ix+1])
			if e != nil {
				return nil, e
			}
			count = int64(i)
			//args = append(args[:ix-1], args[ix+2:]...)
			break
		}
	}
	keyword := cmd.Args[1]
	for ix := 0; ix < len(cmd.Args); ix++ {
		value := cmd.Args[ix]
		if ToUpper(value) == "MATCH" && ix+1 < len(cmd.Args) {
			keyword = cmd.Args[ix+1]
			break
		}
	}
	if count <= 0 {
		i, e := executor.client.DBSize().Result()
		if e != nil {
			i = 5000
		}
		if i == 0 {
			r := interface{}([]string{})
			return &r, nil
		}
		size := i / 10
		if size < 1000 {
			size = 1000
		} else if size > 10000 {
			size = 10000
		}
		count = size
	}
	var results []string
	for cursor := uint64(0); true; {
		keys, c, err := executor.client.Scan(cursor, keyword, count).Result()
		if err != nil {
			return nil, err
		}
		results = append(results, keys...)
		if c == 0 {
			break
		}
		cursor = c
	}
	var k interface{} = results
	return &k, nil
}

func _GetMatch(executor *RedisExecutor, cmd *RedisCommand) (*interface{}, error) {
	keys, e := _ScanAll(executor, &RedisCommand{
		Args:   []string{"SCANALL", cmd.Args[1]},
		Option: cmd.Option,
	})
	if e != nil {
		return nil, e
	}
	val := (*keys).([]string)
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
	return &k, nil
}

func _DelAll(executor *RedisExecutor, cmd *RedisCommand) (*interface{}, error) {
	keys := executor.client.Keys(cmd.Args[1])
	if keys.Err() != nil {
		return nil, keys.Err()
	}
	val := keys.Val()
	//if executor.Option.Cluster {
	//	for _, value := range val {
	//		executor.client.Del(value)
	//	}
	//} else {
	for i := 0; i < len(val); i += 100 {
		o := i + 100
		if o > len(val) {
			o = len(val)
		}
		executor.client.Del(val[i:o]...)
	}
	//}
	var k interface{} = val
	return &k, nil
}

var cMap = map[string]func(*RedisExecutor, *RedisCommand) (*interface{}, error){
	"scanall":  _ScanAll,
	"getmatch": _GetMatch,
	"delall":   _DelAll,
}

func (executor *RedisExecutor) coreExecute(command *RedisCommand) (result *interface{}, err error) {
	args := ToInterface(StringArray(command.Args))
	cmd := redis.NewCmd(args...)
	err = executor.client.Process(cmd)
	if err != nil && err != redis.Nil {
		return nil, err
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
		return &result, nil
	}
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
