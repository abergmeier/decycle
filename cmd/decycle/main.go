package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/abergmeier/decycle/internal"
)

var filename = flag.String("filename", "", "Single filename to scan")

func main() {
	flag.Parse()

	if *filename != "" {

		ids, err := internal.ParseFile(*filename)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%v\n", ids)
		return
	}

	n := flag.NArg()

	if n == 0 {
		fmt.Fprintf(os.Stderr, "Missing directory to parse")
		flag.PrintDefaults()
		os.Exit(1)
	}

	fs := map[string][]internal.Import{}

	c := internal.ParseDir(flag.Arg(0))
	for imp := range c {
		fs[imp.Filename] = imp.Imps
	}
	fmt.Println(fs)
}
