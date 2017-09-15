package network

import (
	"log"
	"runtime/debug"
)

func CatchException() {
	if e := recover(); e != nil {
		log.Println(string(debug.Stack()))

	}
}
