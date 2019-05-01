
package main

import (
	"flag"
	"context"
	"strings"
	"fmt"
	"io/ioutil"
	"os"
//	"bufio"
//	H "github.com/AljabrIO/koalja-operator/pkg/history"
	H "history"
)

// ****************************************************************************
// basic query of graph structure
// ****************************************************************************

var VISITED = make(map[string]bool)  // loop avoidance

// ****************************************************************************

func main() {

	ctx := context.Background()
	ctx = H.SetLocationInfo(ctx, map[string]string{
		"Pod":     "A_pod_named_foo",
		"Process": "query_concepts",  // insert instance data from env?
		"Version": "0.1",
	})

	// 1. test cellibrium graph

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		ListProcesses();
		os.Exit(1);
	}

	H.SignPost(&ctx,"Test local concept graph").
		PartOf(H.N("Testing suite 1"))

	if len(args) == 1 || args[1] == "all" {
		ListConcepts(args[0])
	} else {
		ShowConcept(0,args[0],args[1])
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

func ListConcepts(app string) {

	path := fmt.Sprintf("/tmp/cellibrium/%s/concepts/",app)

	files, err := ioutil.ReadDir(path)

	if err != nil {
		fmt.Println("Couldn't read concepts in "+path)
		os.Exit(1)
	}

	var counter int = 0

	for _, file := range files {

		descr := fmt.Sprintf("%s/%s/description",path,file.Name())
		description, err := ioutil.ReadFile(descr)

		if err != nil {
			fmt.Println("Couldn't read concept file - "+descr)
			os.Exit(1)
		}

		fmt.Printf("%4d : %s : %.100s ...\n",counter,file.Name(),string(description))
		counter++
	}
}

//**************************************************************

func ShowConcept (level int, app,concept_hash string) {

	if VISITED[concept_hash] {
		return
	}

	VISITED[concept_hash] = true

	path := fmt.Sprintf("/tmp/cellibrium/%s/concepts/%s",app,concept_hash)
	
	descr := fmt.Sprintf("%s/description",path)
	description, err := ioutil.ReadFile(descr)

	if err != nil {
		fmt.Println("Couldn't read concept file - "+descr)
		os.Exit(1)
	}
	
	fmt.Printf("%s Stories about \"%s\" (%s)\n",I(level),string(description),concept_hash)

	links := GetLinksFrom(app,string(description),concept_hash)
	
	for i := 1; i < 5; i++ {

		if links.Bwd[i].Reltype != 0 {
			fmt.Printf("%s %s <-- type (%s)  \"%s\"\n",
				I(level),
				links.Bwd[i].Next,
				H.ASSOCIATIONS[links.Bwd[i].Reltype].Bwd,
				description)

			for next := 0; next < len(links.Bwd[i].Next); next++ {
				ShowConcept(level+1,app,links.Bwd[i].Next[next])
			}
		}

		if links.Fwd[i].Reltype != 0 {
			fmt.Printf("%s \"%s\" type (%s) --> %s\n",
				I(level),
				description,
				H.ASSOCIATIONS[links.Fwd[i].Reltype].Fwd,
				links.Fwd[i].Next)
			
			for next := 0; next < len(links.Fwd[i].Next); next++ {
				ShowConcept(level,app,links.Fwd[i].Next[next])
			}
		}
	}
}

//**************************************************************

func GetLinksFrom(app,description,concept_hash string) H.Links {

	path := fmt.Sprintf("/tmp/cellibrium/%s/concepts/%s",app,concept_hash)
	
	files, err := ioutil.ReadDir(path)
	
	if err != nil {
		fmt.Println("Couldn't read directory "+path+" for concept: "+concept_hash)
		os.Exit(1)
	}
	
	var links H.Links = H.LinkInit()
	
	for _, file := range files {
		
		if file.IsDir() {
			
			subdir := fmt.Sprintf("/tmp/cellibrium/%s/concepts/%s/%s/",app,concept_hash,file.Name())
			
			sfiles, serr := ioutil.ReadDir(subdir)
			
			if serr != nil {
				fmt.Println("Couldn't read subdirectory "+subdir)
				os.Exit(1)
			}

			for _, sfile := range sfiles {

				if sfile.IsDir() {
					ssubdir := fmt.Sprintf("/tmp/cellibrium/%s/concepts/%s/%s/%s/",app,concept_hash,file.Name(),sfile.Name())
					var sttype int
					fmt.Sscanf(file.Name(),"%d",&sttype)

					ssfiles, sserr := ioutil.ReadDir(ssubdir)
					
					if sserr != nil {
						fmt.Println("Couldn't read ssub directory "+ssubdir)
						os.Exit(1)
					}
					
					for _, ssfile := range ssfiles {

						var reltype, index int
						fmt.Sscanf(file.Name(),"%d",&reltype)

						if reltype < 0 {
							index = -2*reltype+1
						} else {
							index = 2*reltype
						}

						if sttype < 0 {
							links.Bwd[-sttype].Next = append(links.Bwd[-sttype].Next,ssfile.Name())
							links.Bwd[-sttype].Reltype = index

						} else {
							links.Fwd[sttype].Next = append(links.Fwd[sttype].Next,ssfile.Name())
							links.Fwd[sttype].Reltype = index
						}
					}
				}
			}
		}

	}

return links
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