package main

import (
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	"reflect"
	"sort"
	"strconv"
	"time"
	//"runtime/debug"

	//"os"
	"strings"
)

//type PrintOutput func(string)
type AnyType interface{}

func If(condition bool, true AnyType, false AnyType) AnyType {
	if condition {
		return true
	}
	return false
}

type Collection interface {
	Len() int
	Get(i int) interface{}
}

type StringArray []string

func (f StringArray) Len() int {
	return len(f)
}
func (f StringArray) Get(i int) interface{} {
	return f[i]
}

func ToInterface(s Collection) []interface{} {
	mx := s.Len()
	newSls := make([]interface{}, mx)
	for i := 0; i < mx; i++ {
		newSls[i] = s.Get(i)
	}
	return newSls
}

//var colorableOut = colorable.NewColorableStdout()

func Write(a ...interface{}) {
	write(fmt.Sprint(a...))
	//_, _ = colorableOut.Write([]byte(fmt.Sprint(a...)))
}

func WriteLn(a ...interface{}) {
	write(fmt.Sprint(a...) + "\n")
	//debug.PrintStack()
	//_, _ = colorableOut.Write([]byte(fmt.Sprint(a...) + "\n"))
}

func WriteF(s string, a ...interface{}) {
	write(fmt.Sprintf(s, a...))
	//_, _ = colorableOut.Write([]byte(fmt.Sprintf(s, a...)))
}
func WriteFLn(s string, a ...interface{}) {
	write(fmt.Sprintf(s, a...) + "\n")
	//_, _ = colorableOut.Write([]byte(fmt.Sprintf(s, a...)))
}

func parseArg(input string) []string {
	var cur []rune = []rune{}
	var list []string = []string{}
	escape := false
	group := false
	groupCh := '"'
	runes := []rune(input)
	for _, char := range runes {
		switch char {
		case ' ':
			if group {
				cur = append(cur, ' ')
			} else if len(cur) > 0 {
				list = append(list, string(cur))
				cur = []rune{}
			}
			escape = false
			break
		case '\'', '"':
			//Debug("p", char)
			if escape {
				cur = append(cur, char)
				escape = false
			} else if group {
				if groupCh != char {
					cur = append(cur, char)
				} else {
					group = false
					list = append(list, string(cur))
					cur = []rune{}
				}
			} else if len(cur) > 0 {
				cur = append(cur, char)
			} else {
				group = true
				groupCh = char
			}
			break
		case '\\':
			if escape {
				cur = append(cur, char)
				escape = false
			} else {
				escape = true
			}
			break
		default:
			cur = append(cur, char)
			escape = false
			break
		}
	}
	if len(cur) > 0 {
		list = append(list, string(cur))
	}
	//WriteLn(input)
	//for _, value := range list {
	//	WriteLn("[" + value + "]")
	//}
	return list
}

func GetHostOpt(args []string) (error, *RedisHostOption, []string) {
	option := RedisHostOption{
		Host: "localhost",
		Port: 6379,
	}
	parser := flags.NewParser(&option, flags.PassDoubleDash|flags.IgnoreUnknown)
	cmd, e := parser.ParseArgs(args)
	return e, &option, cmd
}

func GetCmdOpt(args []string) (*RedisCommandOption, []string) {
	commandOption := RedisCommandOption{}
	option := struct {
		FormatType func(string)  `short:"f" long:"format" alias:"as" description:"format type: support: json, normal, raw"`
		Repeat     func(uint)    `short:"r" long:"repeat" description:"repeat time"`
		Delay      func(float32) `short:"d" long:"delay" description:"delay in sec. (float)"`
		//RunAtEachNode          bool          `short:"e" long:"each-node" description:"run at each node"`
		//SplitResultForEachNode bool          `short:"s" long:"split-node" description:"split result for each node"`
		NoColor bool `long:"no-color" description:"no color output"`
	}{
		FormatType: func(s string) {
			switch strings.ToLower(s) {
			case "1", "json":
				commandOption.FormatType = FormatJson
				break
			case "2", "raw", "rawstring":
				commandOption.FormatType = FormatRawString
				break
			}
		},
		Repeat: func(u uint) {
			commandOption.Repeat = u
		},
		Delay: func(u float32) {
			commandOption.Delay = u
		},
	}
	parser := flags.NewParser(&option, flags.PassDoubleDash|flags.IgnoreUnknown)
	parser.Name = "command [args[]]"
	argsx, _ := parser.ParseArgs(args)
	//commandOption.RunAtEachNode = option.RunAtEachNode
	//commandOption.SplitResultForEachNode = option.SplitResultForEachNode
	//WriteLn(commandOption)
	//WriteLn(argsx)
	//parser.WriteHelp(os.Stdout)
	return &commandOption, argsx
}

var _DEBUG_ = true

func Debug(tag string, a ...interface{}) {
	if _DEBUG_ {
		var k = append(append([]interface{}{interface{}("\033[32mDEBUG: " + time.Now().Format("2006-01-02 15:04:05") + " [" + tag + "] ")}, a...), "\033[0m")
		WriteLn(k...)
	}
}

func (result *RedisExecuteResult) Format(f EnumFormatType, nc bool) {
	switch f {
	case FormatJson:
		result.FormatNormal(f)
	case FormatNormal:
		result.FormatNormal(f)
	case FormatRawString:
		result.FormatNormal(f)
	}
}

func (result *RedisExecuteResult) FormatNormal(ft EnumFormatType) {
	//Debug("format", *result.Value)
	if result.Value == nil || *result.Value == nil {
		WriteLn(If(ft == FormatJson, "null", "(nil)"))
		return
	}
	value := reflect.ValueOf(*result.Value)
	formatNormal(value, "", 0, 0, ft)
}

func makePrefix(prefix string, ix int, count int) string {
	//if len(prefix) > 0 {
	//	prefix
	//}
	if count == 0 {
		return prefix
	}
	sIx := len(strconv.Itoa(ix))
	sCount := len(strconv.Itoa(count))
	//defer func() {
	//	if r := recover(); r != nil {
	//		WriteLn(fmt.Sprintf("%d:%d, %d,%d", ix, sIx, count, sCount))
	//	}
	//}()
	if ix == 0 {
		//Debug("Prefix", ix, count, "[", prefix, "]")
		return prefix + strings.Repeat(" ", sCount-sIx) + strconv.Itoa(ix) + ") "
	}
	return strings.Repeat(" ", sCount-sIx+len(prefix)) + strconv.Itoa(ix) + ") "
}

func formatNormal(value reflect.Value, prefix string, ix int, count int, ft EnumFormatType) {
	kind := value.Kind()
	//Debug("BG", dep, kind, value.Interface())
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		WriteLn(makePrefix(prefix, ix, count), value)
	case reflect.String:
		_s := valueToString(ft, value)
		WriteLn(makePrefix(prefix, ix, count), _s)
	case reflect.Array, reflect.Slice:
		_len := value.Len()
		//Debug("Array", ix, count, _len, " [", prefix, "]")
		px := makePrefix(prefix, ix, count)
		if _len == 0 {
			WriteLn(px, "(empty)")
			break
		}
		for i := 0; i < _len; i++ {
			v2 := value.Index(i)
			formatNormal(v2, px, i, _len, ft)
		}
	case reflect.Map:
		iter := Keys(value.MapKeys())
		sort.Sort(iter)
		for _, v := range iter {
			WriteLn(makePrefix(prefix, ix, count) + v.Interface().(string) + " : " + valueToString(ft, value.MapIndex(v)))
		}
	case reflect.Interface:
		formatNormal(value.Elem(), prefix, ix, count, ft)
	case reflect.Invalid:
		WriteLn(makePrefix(prefix, ix, count), "(nil)")
	default:
		if value.IsNil() {
			WriteLn(makePrefix(prefix, ix, count), "(nil)")
		} else {
			WriteLn(makePrefix(prefix, ix, count), value)
		}
	}
}

type Keys []reflect.Value

func (a Keys) Len() int {
	return len(a)
}

func (a Keys) Less(i, j int) bool {
	iv := a[i].Interface().(string)
	jv := a[j].Interface().(string)
	return iv < jv
}

func (a Keys) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func valueToString(formatType EnumFormatType, value reflect.Value) string {
	var ifv interface{}
	if value.Kind() == reflect.Interface {
		ifv = value.Interface()
	}
	Debug("vts", formatType, value.Kind(), value.Type(), reflect.ValueOf(ifv).Kind(), value)
	if value.Kind() == reflect.String || reflect.ValueOf(ifv).Kind() == reflect.String {
		if formatType == FormatRawString {
			return value.Interface().(string)
		}
		if formatType == FormatJson {
			var data interface{}
			sss := value.Interface().(string)
			e := json.Unmarshal([]byte(sss), &data)
			if e != nil {
				//Debug("json", sss, " - ", e)
				bytes, _ := json.Marshal(sss)
				return string(bytes)
			}

			s, _ := json.MarshalIndent(&data, "", "  ")
			//Debug("json", data, reflect.TypeOf(data).String(), " - ", string(s))
			return string(s)
		}
	}
	//if formatType == FormatNormal {
	s, _ := json.MarshalIndent(value.Interface(), "", "  ")
	return string(s)
	//}
	//return fmt.Sprint(value)
}
