
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
// SPLIT ! TOP
// 2. Koalja program starts BELOW ...
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
		ShowConcept(args[0],args[1])
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

func ShowConcept (app,concept_hash string) {

//	c := H.CreateConcept(concept)
	
	path := fmt.Sprintf("/tmp/cellibrium/%s/concepts/%s",app,concept_hash)
	
	files, err := ioutil.ReadDir(path)

	if err != nil {
		fmt.Println("Couldn't read directory "+path+" for concept: "+concept_hash)
		os.Exit(1)
	}

	descr := fmt.Sprintf("%s/description",path)
	description, err := ioutil.ReadFile(descr)

	if err != nil {
		fmt.Println("Couldn't read concept file - "+descr)
		os.Exit(1)
	}
	
	fmt.Printf("Stories about \"%s\" (%s)\n",string(description),concept_hash)

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

						var reltype,index int
						fmt.Sscanf(file.Name(),"%d",&reltype)

						//fmt.Printf("RAW of type (%s,%s) = %s\n",file.Name(),sfile.Name(),ssfile.Name())

						// The POSITIVE and NEGATIVE relations are stored in pairs, so...
						
						if reltype < 0 {
							index = -2*reltype+1
						} else {
							index = 2*reltype
						}

						if sttype < 0 {
							links.Bwd[-sttype].Name = append(links.Bwd[-sttype].Name,ssfile.Name())
							links.Bwd[-sttype].Reltype = reltype
							fmt.Printf("BWD \"%s\" type (%s,%d) = %s\n",
								string(description),
								H.ASSOCIATIONS[index].Bwd,
								reltype,
								H.ConceptName(ssfile.Name()))

						} else {
							links.Fwd[sttype].Name = append(links.Fwd[-sttype].Name,ssfile.Name())
							links.Fwd[sttype].Reltype = reltype
							fmt.Printf("FWD \"%s\" type (%s,%d) = %s\n",
								string(description),
								H.ASSOCIATIONS[index].Fwd,
								reltype,
								H.ConceptName(ssfile.Name()))
						}
					}
				}
			}
		}

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