package term

import "fmt"

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

func WriteLnF(s string, a ...interface{}) {
	write(fmt.Sprintf(s, a...) + "\n")
	//_, _ = colorableOut.Write([]byte(fmt.Sprintf(s, a...)))
}
