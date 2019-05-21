
package main

import (
	"context"
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

	ctx := context.Background()
	ctx = H.SetLocationInfo(ctx, map[string]string{
		"Pod":        "Koalja_empty_pod",
		"Deployment": "Koalja query_history",  // insert instance data from env?
		"Version":    "0.1",
	})

	m := H.SignPost(&ctx,"Show process history").
		PartOf(H.N("Koalja"))

	// 1. test cellibrium - need an invariant name (non trivial in cloud)

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		ListProcesses();
		os.Exit(1);
	}

	fmt.Printf("opening /tmp/cellibrium/%s\n", args[0]);
	
	path := fmt.Sprintf("/tmp/cellibrium/%s/",args[0])

	files, err := ioutil.ReadDir(path)

	if err != nil {
		os.Exit(1)
	}

	m.Note("For each file, open")

	for _, file := range files {
		if strings.HasPrefix(file.Name(),"transaction") {
			ShowFile(ctx,args[0],file.Name())
		}
	}
}

//**************************************************************

func ListProcesses() {

	path := fmt.Sprintf("/tmp/cellibrium/")

	files, err := ioutil.ReadDir(path)

	if err != nil {
		fmt.Println("Couldn't read concepts in "+path)
		os.Exit(1)
	}

	fmt.Println("Available processes:")

	for _, file := range files {

		fmt.Println(" - "+file.Name())
	}
}

//**************************************************************

func ShowFile(ctx context.Context,app,name string) {

	m := H.SignPost(&ctx,"ShowFile")

	path := fmt.Sprintf("/tmp/cellibrium/%s/",app)

	file, err := os.Open(path+name)

	if err != nil {
		m.FailedBecause("Opening "+path+" to show history").AddError(err)
		fmt.Println(err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(file)

	parts := strings.Split(name,"_")

	m.Note("Start history dump")

	fmt.Println("\nNew process timeline for (",app,") originally started as pid ",parts[1],"\n")
	fmt.Println("Unix clock context              | root --> NOW,delta  Comment indented by subtime")
	fmt.Println("------------------------------------------------------------------------------------------")

	for scanner.Scan() {
		ParseLine(m,app,scanner.Text(),parts[1])
	}

	m.Note("End history dump")

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	file.Close()
}

//**************************************************************

func ParseLine(m H.ProcessContext, app string, s string, fpid string) {

	var t,proper,exterior,prev int
	var pid string

	fmt.Sscanf(s,"%s , %d , %d , %d , %d",&pid,&t,&proper,&exterior,&prev)
	remark := strings.Split(s,"; ")

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

func R(previous,now,proper int) string {

	if proper == 1 {

		if now != previous+1 {
			return fmt.Sprintf("%4d go> %3d,%-3d %s",previous,now,proper,I(proper))
		} else {
			return fmt.Sprintf("%4d --> %3d,%-3d %s",previous,now,proper,I(proper))
		}
	} else {
		return fmt.Sprintf("      -> %3d,%-3d %s",now,proper,I(proper))
	}
}
//**************************************************************

func Clock(t int) string {
	
	secs := int64(t)
	return fmt.Sprintf("%s",time.Unix(secs, 0))
}