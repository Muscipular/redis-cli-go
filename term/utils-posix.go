// +build !windows

package term

import "fmt"

func write(s string) {
	fmt.Print(s)
}
