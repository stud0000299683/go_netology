package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	// Имя пользователя: USERNAME (Windows) или USER (Unix)
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	fmt.Println("User:", user)

	// Аргументы CLI (без имени программы)
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Args: (none)")
	} else {
		fmt.Println("Args:", args)
	}

	fmt.Println("Go version:", runtime.Version())
}
