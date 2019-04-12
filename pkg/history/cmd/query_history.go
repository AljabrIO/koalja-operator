
package main

import (
	"flag"
	"strings"
	"fmt"
	"io/ioutil"
	"os"
//	H "github.com/AljabrIO/koalja-operator/pkg/history"
//	H "history"
)

// ****************************************************************************
// SPLIT ! TOP
// 2. Koalja program starts BELOW ...
// ****************************************************************************

func main() {

	// 1. test cellibrium - need an invariant name (non trivial in cloud)

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Input process missing.");
		os.Exit(1);
	}

	fmt.Printf("opening /tmp/cellibrium/%s\n", args[0]);
	
	path := fmt.Sprintf("/tmp/cellibrium/%s",args[0])

	files, err := ioutil.ReadDir(path)

	if err != nil {
		os.Exit(1)
	}

	for _, file := range files {
		fmt.Println(file.Name())
	}

	if err != nil {
		fmt.Println("ERROR ",err)
	}


}

//**************************************************************

func usage() {
    fmt.Fprintf(os.Stderr, "usage: query_history [process]\n")
    flag.PrintDefaults()
    os.Exit(2)
}

//**************************************************************

func I(level int) string {
	var indent string = strings.Repeat("  ",level)
	var s string
	s = fmt.Sprintf("%.3d:%s",level,indent)
	s = indent
	return s
}