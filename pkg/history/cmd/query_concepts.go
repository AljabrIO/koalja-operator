
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
		"Pod":        "A_pod_named_foo",
		"Deployment": "query_concepts",  // insert instance data from env?
		"Version":    "0.1",
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

		description := ConceptName(args[0],args[1])
		links := GetLinksFrom(args[0],description,args[1])
	
		fmt.Printf("\nStories about \"%s\" (%s) in app %s \n\n",description,args[1],args[0])
		DescribeConcept(args[0],1,description,links)
		GeneralizeConcept(args[0],1,description,links)

		fmt.Println("1. Following ---------\n")
		ShowNeighbourConcepts(0,H.GR_FOLLOWS,args[0],args[1],description,links)
		fmt.Println("2. Is followed by ---------\n")
		ShowNeighbourConcepts(0,-H.GR_FOLLOWS,args[0],args[1],description,links)

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

func ConceptName(app,concept_hash string) string {

	path := fmt.Sprintf("/tmp/cellibrium/%s/concepts/%s",app,concept_hash)	
	descr := fmt.Sprintf("%s/description",path)
	description, err := ioutil.ReadFile(descr)

	if err != nil {
		fmt.Println("Couldn't read concept file - "+descr)
		os.Exit(1)
	}

	return string(description)
}

//**************************************************************

func ShowNeighbourConcepts (level int, sttype int, app string, concept_hash string, description string, links H.Links) {

//	fmt.Println("DEB",sttype,links)

	if sttype < 0 {
		for l := range links.Bwd[-sttype] {
		
			for next := 0; next < len(links.Bwd[-sttype][l]); next++ {
				
				fmt.Printf("%s %s <-- type (%s) -- \"%s\"\n",
					I(level),
					links.Bwd[-sttype][l][next],
					H.ASSOCIATIONS[l].Bwd,
					description)
				
				//ShowNeighbourConcepts(level+1,sttype,app,links.Bwd[sttype].Next[next])
			}
		}
		
	} else {
		for l := range links.Fwd[sttype] {

			for next := 0; next < len(links.Fwd[sttype][l]); next++ {
				
				fmt.Printf("%s %s -- type (%s) -->  \"%s\"\n",
					I(level),
					description,
					H.ASSOCIATIONS[l].Fwd,
					links.Fwd[sttype][l][next])
				
				//ShowNeighbourConcepts(level+1,sttype,app,links.Bwd[sttype].Next[next])
			}
		}
		
	}
}


//**************************************************************

func DescribeConcept (app string, level int, name string, links H.Links) {
	
	fmt.Println(I(level),"<begin descr>")

	for l := range links.Bwd[H.GR_EXPRESSES] {

		for next := 0; next < len(links.Bwd[H.GR_EXPRESSES][l]); next++ {
			
			fmt.Printf("%s \"%s\" -- (%s) --> \"%s\" (%s)\n",
				I(level),
				name,
				H.ASSOCIATIONS[l].Bwd,
				ConceptName(app,links.Bwd[H.GR_EXPRESSES][l][next]),
				links.Bwd[H.GR_EXPRESSES][l][next])
		}
	}
	
	for l := range links.Fwd[H.GR_EXPRESSES] {
		
		for next := 0; next < len(links.Fwd[H.GR_EXPRESSES][l]); next++ {
			
			fmt.Printf("%s \"%s\" -- (%s) -->  \"%s\" (%s)\n",
				I(level),
				name,
				H.ASSOCIATIONS[l].Fwd,
				ConceptName(app,links.Fwd[H.GR_EXPRESSES][l][next]),
				links.Fwd[H.GR_EXPRESSES][l][next])
		}
	}

	fmt.Println(I(level),"<end descr>")
}

//**************************************************************

func GeneralizeConcept (app string, level int, name string, links H.Links) {
	
	fmt.Println(I(level),"<begin general>")

	for l := range links.Bwd[H.GR_CONTAINS] {

		for next := 0; next < len(links.Bwd[H.GR_CONTAINS][l]); next++ {
			
			description := ConceptName(app,links.Bwd[H.GR_CONTAINS][l][next])

			fmt.Printf("%s %s -- (%s) --> \"%s\"\n",
				I(level),
				name,
				H.ASSOCIATIONS[l].Bwd,
				description)

			nextlinks := GetLinksFrom(app,description,links.Bwd[H.GR_CONTAINS][l][next])

			for l2 := range nextlinks.Bwd[H.GR_CONTAINS] {
				for next := 0; next < len(nextlinks.Bwd[H.GR_CONTAINS][l2]); next++ {
					
					description := ConceptName(app,nextlinks.Bwd[H.GR_CONTAINS][l2][next])
					
					fmt.Printf("%s %s -- (%s) --> \"%s\"\n",
						I(level+1),
						name,
						H.ASSOCIATIONS[l2].Bwd,
						description)
				}
				
			}
		}
	}
	

	// **

	for l := range links.Fwd[H.GR_CONTAINS] {

		for next := 0; next < len(links.Fwd[H.GR_CONTAINS][l]); next++ {
			
			description := ConceptName(app,links.Fwd[H.GR_CONTAINS][l][next])

			fmt.Printf("%s %s -- (%s) --> \"%s\"\n",
				I(level),
				name,
				H.ASSOCIATIONS[l].Fwd,
				description)

			nextlinks := GetLinksFrom(app,description,links.Fwd[H.GR_CONTAINS][l][next])

			for l2 := range nextlinks.Fwd[H.GR_CONTAINS] {
				for next := 0; next < len(nextlinks.Fwd[H.GR_CONTAINS][l2]); next++ {
					
					description := ConceptName(app,nextlinks.Fwd[H.GR_CONTAINS][l2][next])
					
					fmt.Printf("%s %s -- (%s) --> \"%s\"\n",
						I(level+1),
						name,
						H.ASSOCIATIONS[l2].Fwd,
						description)
				}
				
			}
		}
	}


	fmt.Println(I(level),"<end general>")
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

			var sttype int
			fmt.Sscanf(file.Name(),"%d",&sttype)

			for _, sfile := range sfiles {

				if sfile.IsDir() {
					ssubdir := fmt.Sprintf("/tmp/cellibrium/%s/concepts/%s/%s/%s/",app,concept_hash,file.Name(),sfile.Name())

					ssfiles, sserr := ioutil.ReadDir(ssubdir)
					
					if sserr != nil {
						fmt.Println("Couldn't read ssub directory "+ssubdir)
						os.Exit(1)
					}

					var reltype, index int
					reltype = 0
					index = 0
					fmt.Sscanf(sfile.Name(),"%d",&reltype)
					
					if reltype < 0 {
						index = -2*reltype-1
					} else {
						index = 2*reltype-1
					}
					
					//fmt.Printf("ST=%d, rel=%d, index=%d\n",sttype,reltype,index)
					
					for _, ssfile := range ssfiles {

						// Loop prevention

						if VISITED[ssfile.Name()] {
							continue
						}
						
						VISITED[ssfile.Name()] = true

						if sttype < 0 {
							links.Bwd[-sttype][index] = append(links.Bwd[-sttype][index],ssfile.Name())
						} else {
							links.Fwd[sttype][index] = append(links.Fwd[sttype][index],ssfile.Name())
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