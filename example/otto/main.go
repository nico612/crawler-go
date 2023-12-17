package main

import (
	"fmt"
	"github.com/robertkrimen/otto"
)

// otto 是用 Go 编写的 JavaScript 虚拟机，用于在 Go 中执行 Javascript 语法。

func main() {
	vm := otto.New()

	script := `
		var n = 100;
		console.log("hello-" + n);
		n;
	`
	value, _ := vm.Run(script)
	fmt.Println("value: ", value.String())

}
