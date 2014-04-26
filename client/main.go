package main

import (
	"flag"
	"fmt"

	"github.com/rsesek/go-crash-reporting/crashhost"
)

var doPanic = flag.Bool("panic", false, "Panic after launch")

func main() {
	flag.Parse()

	fmt.Println("client main()")

	crashhost.EnableCrashReporting()

	fmt.Println("crash reporting enabled")

	if *doPanic {
		panic("omg")
	}

	for {}
}
