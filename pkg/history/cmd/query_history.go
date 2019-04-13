
package main

import (
	"flag"
	"strings"
	"fmt"
	"io/ioutil"
	"os"
	"bufio"
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
	
	path := fmt.Sprintf("/tmp/cellibrium/%s/",args[0])

	files, err := ioutil.ReadDir(path)

	if err != nil {
		os.Exit(1)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(),"transaction") {
			ShowFile(path,file.Name())
		}
	}


}

//**************************************************************

func ShowFile(path,name string) {

	file, err := os.Open(path+name)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer file.Close()
	
	scanner := bufio.NewScanner(file)

	parts := strings.Split(name,"_")

	fmt.Println("SCANNING process originally started as pid ",parts[1])

	for scanner.Scan() {
		ParseLine(scanner.Text(),parts[1])
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//**************************************************************

func ParseLine(s string, fpid string) {

	var t,proper,exterior,prev int
	var pid string

	fmt.Sscanf(s,"%s , %d , %d , %d , %d",&pid,&t,&proper,&exterior,&prev)
	remark := strings.Split(s,";")

	if (fpid != pid) {
		fmt.Println("process forked -- ",pid," != ",fpid)
	}

	if exterior != prev+1 {
		fmt.Printf("go> %d <- root = %d+d%d  (utc %d)  | %s \n",prev,exterior,proper,t,remark[1])
	} else {
		fmt.Printf("%d <- root = %d+d%d  (utc %d)      | %s \n",prev,exterior,proper,t,remark[1])
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