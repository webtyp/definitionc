package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"webtyp.com/definitionc"
)

func main() {
	flag.Parse()

	o := definitionc.New()
	o.SetLog(func(messages ...any) {
		fmt.Fprintln(os.Stderr, messages...)
	})
	if err := o.Run(); err != nil {
		log.Fatalf("definitionc: %v", err)
	}
}
