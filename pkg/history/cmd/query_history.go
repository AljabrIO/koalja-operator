
package main

import (
	"flag"
	"strings"
	"fmt"
	"io/ioutil"
	"os"
	"time"
	"bufio"
//	H "github.com/AljabrIO/koalja-operator/pkg/history"
	H "history"
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
			ShowFile(args[0],file.Name())
		}
	}


}

//**************************************************************

func ShowFile(app,name string) {

	path := fmt.Sprintf("/tmp/cellibrium/%s/",app)

	file, err := os.Open(path+name)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer file.Close()
	
	scanner := bufio.NewScanner(file)

	parts := strings.Split(name,"_")

	fmt.Println("\nNew process timeline for (",app,") originally started as pid ",parts[1],"\n")

	for scanner.Scan() {
		ParseLine(app,scanner.Text(),parts[1])
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//**************************************************************

func ParseLine(app string, s string, fpid string) {

	var t,proper,exterior,prev int
	var pid string

	fmt.Sscanf(s,"%s , %d , %d , %d , %d",&pid,&t,&proper,&exterior,&prev)
	remark := strings.Split(s,";")

	if (fpid != pid) {
		fmt.Println("process forked -- ",pid," != ",fpid)
	}

	fmt.Printf("%s  | %s %s \n",Clock(t),R(prev,exterior,proper),H.ConceptName(app,remark[1]))
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

//**************************************************************

func R(previous,now,t int) string {

	if t == 1 {

		if now != previous+1 {
			return fmt.Sprintf("%d go> %d. %s",previous,now,I(t))
		} else {
			return fmt.Sprintf("%d --> %d. %s",previous,now,I(t))
		}
	} else {
		return fmt.Sprintf("   -> %d. %s",now,I(t))
	}
}
//**************************************************************

func Clock(t int) string {
	
	secs := int64(t)
	return fmt.Sprintf("%s",time.Unix(secs, 0))
}