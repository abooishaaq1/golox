package main

import (
	"bufio"
	"fmt"
	"golox/vm"
	"golox/vm/interpretresult"
	"os"
)

func repl(vm *vm.VM) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, _, err := reader.ReadLine()
		if err != nil {
			fmt.Printf("an error while reading input: %s\n", err.Error())
			return
		} else {
			fmt.Println()
		}
		// appending "\x00" so that currChar() does not give runtime error
		vm.Interpret(string(line) + "\x00")
	}
}

func runFile(path string, vm *vm.VM) {
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("an error occurred while reading the file: %s", err.Error())
		os.Exit(74)
	}
	result := vm.Interpret(string(source) + "\x00")
	if result == interpretresult.INTERPRET_COMPILE_ERROR {
		os.Exit(65)
	}
	if result == interpretresult.INTERPRET_RUNTIME_ERROR {
		os.Exit(75)
	}
}

func main() {
	vm := new(vm.VM)
	vm.Init()

	if len(os.Args) == 1 {
		repl(vm)
	} else if len(os.Args) == 2 {
		runFile(os.Args[1], vm)
	} else {
		fmt.Fprint(os.Stderr, "Usage: clox [path]\n")
		os.Exit(64)
	}
}
