package term

import (
	"fmt"
	"github.com/eiannone/keyboard"
)

type Term struct {
	isDisposed  bool
	write       func(...interface{})
	platformObj *interface{}
}

func NewTerm() *Term {
	return &Term{
		isDisposed: false,
	}
}

func (tm *Term) Init() error {
	err := keyboard.Open()
	if err != nil {
		return err
	}
	err = tm._platformInit()
	if err != nil {
		return err
	}
	return nil
}

func (tm *Term) Dispose() error {
	keyboard.Close()
	return nil
}

func (tm *Term) GetKey() (rune rune, key keyboard.Key, err error) {
	r, i, err := keyboard.GetKey()
	return r, i, err
}

func (tm *Term) HideCursor() {

}

func (tm *Term) ShowCursor() {

}

func (tm *Term) Move(i int) {

}

func (tm *Term) Write(a ...interface{}) {
	tm.write(fmt.Sprint(a...))
}

func (tm *Term) WriteLn(a ...interface{}) {
	tm.write(fmt.Sprint(a...) + "\n")
}

func (tm *Term) WriteF(s string, a ...interface{}) {
	tm.write(fmt.Sprintf(s, a...))
}

func (tm *Term) WriteLnF(s string, a ...interface{}) {
	tm.write(fmt.Sprintf(s, a...) + "\n")
}
