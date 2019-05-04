
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

		app := args[0]
		concept_hash := args[1]

		var visited = make(map[string]bool)  // loop avoidance

		description := ConceptName(app,concept_hash)
		links := GetLinksFrom(app,concept_hash,visited)
	
		fmt.Printf("\nStories about \"%s\" (%s) in app %s \n\n",description,concept_hash,app)
		DescribeConcept(app,1,description,links)

		causal_set := GetGeneralizationCone(app,concept_hash)

		GetCausationCone(app,causal_set)
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

func DescribeConcept (app string, level int, name string, links H.Links) {
	
	fmt.Println(I(level),"<begin describe>")

	for l := range links.Bwd[H.GR_EXPRESSES] {

		for next := 0; next < len(links.Bwd[H.GR_EXPRESSES][l]); next++ {
			
			fmt.Printf("%s \"%s\" -- (%s) --> \"%s\" (%s)\n",
				I(level),
				"topic",
				H.ASSOCIATIONS[l].Bwd,
				ConceptName(app,links.Bwd[H.GR_EXPRESSES][l][next]),
				links.Bwd[H.GR_EXPRESSES][l][next])
		}
	}
	
	for l := range links.Fwd[H.GR_EXPRESSES] {
		
		for next := 0; next < len(links.Fwd[H.GR_EXPRESSES][l]); next++ {
			
			fmt.Printf("%s \"%s\" -- (%s) -->  \"%s\" (%s)\n",
				I(level),
				"topic",
				H.ASSOCIATIONS[l].Fwd,
				ConceptName(app,links.Fwd[H.GR_EXPRESSES][l][next]),
				links.Fwd[H.GR_EXPRESSES][l][next])
		}
	}

	fmt.Println(I(level),"<end describe>")
}

//**************************************************************

func GetGeneralizationCone (app string, concept_hash string) []string {
	
	// First establish how many different up/down CONTAINS relations attach to the anchor concept
	// These form separate graphs

	var visited = make(map[string]bool)  // loop avoidance
	var level int = 1

	nodelinks := GetLinksFrom(app,concept_hash,visited)

	retarded_directions := nodelinks.Fwd[H.GR_CONTAINS]
	advanced_directions := nodelinks.Bwd[H.GR_CONTAINS]

	fwd_cone := RetardedCone(app,retarded_directions,visited)
	bwd_cone := AdvancedCone(app,advanced_directions,visited)

	fmt.Println("")
	fmt.Println(I(level),"<begin generalization cone>")
	region := ShowCone(app,concept_hash,fwd_cone,bwd_cone)
	fmt.Println(I(level),"<end generalization cone>")

return region
}

//**************************************************************

func GetCausationCone (app string, cset []string) {
	
	var visited = make(map[string]bool)  // loop avoidance

	for c := range cset {
		concept_hash := cset[c]
		visited[concept_hash] = true

		nodelinks := GetLinksFrom(app,concept_hash,visited)

		retarded_directions := nodelinks.Bwd[H.GR_FOLLOWS]
		advanced_directions := nodelinks.Fwd[H.GR_FOLLOWS]

		fmt.Println("")		
		fmt.Println(I(1),"<begin causal propagation cone>")
		for  i := range advanced_directions {
			for n := range advanced_directions[i] {
				fmt.Println(I(2),"-", H.ASSOCIATIONS[i].Fwd,ConceptName(app,advanced_directions[i][n]))
			}
		}

		for  i := range retarded_directions {
			for n := range retarded_directions[i] {
				fmt.Println(I(2),"-",H.ASSOCIATIONS[i].Bwd,ConceptName(app,retarded_directions[i][n]))
			}
		}

		fmt.Println(I(1),"<begin causal propagation cone>")
	}
}


//**************************************************************

/* retarded wave, for each direction, push the wave and accumulate 
   the wavefront, which will be added to the total region

    directions 1 **
        |      2 **** locations ->
        V      3 *

   // RetardedCone (init H.NeighourConcepts)
   // for maplist = init; maplist not empty; maplist = nextlist
   //   
   //   for each channel in maplist[]
   //       for each node in maplist[channel]
   //         scan node -> fwd(channel, nextnode) 
   //         nextlist[channel] = nextnode
   // fwd_cone[] += maplist[]
   // return fwd_cone
		
   // total_region = init + fwd_cone + bwd_cone

   // we are only doing ascent, so don't need to mix fwd/bwd channels in same process
   // no antiparticles in reasoning...?

*/

//**************************************************************

func RetardedCone(app string, init H.NeighbourConcepts, visited map[string]bool) H.NeighbourConcepts {

	var maplist, nextlist, fwd_cone H.NeighbourConcepts

	fwd_cone = make(H.NeighbourConcepts,0)

	for maplist = init; Neighbours(maplist); maplist = nextlist {

		fwd_cone = JoinNeighbours(fwd_cone,maplist)
		nextlist = make(H.NeighbourConcepts,0)

		for linktype := range maplist {			
			if nextlist[linktype] == nil {
				nextlist[linktype] = make([]string,0)
			}
			
			for node := 0; node < len(maplist[linktype]); node++ {
				neighbours := GetLinksFrom(app,maplist[linktype][node],visited)
				nextlist = JoinNeighbours(nextlist,neighbours.Fwd[H.GR_CONTAINS])
			}
		}
	}

return fwd_cone
}

//**************************************************************

func AdvancedCone(app string, init H.NeighbourConcepts, visited map[string]bool) H.NeighbourConcepts {

	var maplist, nextlist, bwd_cone H.NeighbourConcepts

	bwd_cone = make(H.NeighbourConcepts,0)

	for maplist = init; Neighbours(maplist); maplist = nextlist {

		bwd_cone = JoinNeighbours(bwd_cone,nextlist)
		nextlist = make(H.NeighbourConcepts,0)
		
		for linktype := range maplist {
			
			if nextlist[linktype] == nil {
				nextlist[linktype] = make([]string,0)
			}
			
			for node := 0; node < len(maplist[linktype]); node++ {
				
				neighbours := GetLinksFrom(app,maplist[linktype][node],visited)
				nextlist = JoinNeighbours(nextlist,neighbours.Bwd[H.GR_CONTAINS])
			}
		}
	}

return bwd_cone
}

// ************************************************************************

func ShowCone(app string,concept_hash string, fcone, bcone H.NeighbourConcepts) []string {

	var kinds = make(map[int]bool)
	var region = make([]string,0)

	// merge the region subtypes from the forward/backward propagation cones

	for linktype := range fcone {
		kinds[linktype] = true
	}

	for linktype := range bcone {
		kinds[linktype] = true
	}

	// print and merge into a single list

	for linktype := range kinds {
		if fcone[linktype] != nil {

			for fnode := 0; fnode < len(fcone[linktype]); fnode++ {
				fmt.Printf("%s%s --(%s)--> %s\n",
					I(3),
					ConceptName(app,concept_hash),
					H.ASSOCIATIONS[linktype].Fwd,
					ConceptName(app,fcone[linktype][fnode]))

				region = append(region,fcone[linktype][fnode])
			}
		}
		if bcone[linktype] != nil {
			for bnode := 0; bnode < len(bcone[linktype]); bnode++ {
				fmt.Printf("%s%s --(%s)--> %s\n",
					I(3),
					ConceptName(app,concept_hash),
					H.ASSOCIATIONS[linktype].Bwd,
					ConceptName(app,bcone[linktype][bnode]))

				region = append(region,bcone[linktype][bnode])
			}
		}
		
	}

	// don't forget the focal point!
	
	region = append(region,concept_hash)
	return region
}

// ************************************************************************

func Neighbours(x H.NeighbourConcepts) bool {

	for t := range x {
		if x[t] != nil && len(x[t]) > 0 {
			return true
		}
	}
	
	return false
}

// ************************************************************************

func JoinNeighbours(master H.NeighbourConcepts, delta H.NeighbourConcepts) H.NeighbourConcepts {

	var result H.NeighbourConcepts = make(H.NeighbourConcepts)

	for t := range master {
		
		result[t] = make([]string,0)
	
		for n := 0; n < len(master[t]); n++ {

			result[t] = append(result[t],master[t][n])
		}
	}

	for t := range delta {
		
		if result[t] == nil {
			result[t] = make([]string,0)
		}
	
		for n := 0; n < len(delta[t]); n++ {
			result[t] = append(result[t],delta[t][n])
		}
	}

return result
}

//************************************
// advanced wave

func ExploreBwdCone(app string, nextlinks H.Links, region H.NeighbourConcepts) {

	for direction := range nextlinks.Fwd[H.GR_CONTAINS] {

		if region[direction] == nil {
			region[direction] = make([]string,0)
		}

		// iterate over the links of same type

		for next_location := 0; next_location < len(nextlinks.Bwd[H.GR_CONTAINS][direction]); next_location++ {
			region[direction] = append(region[direction],nextlinks.Bwd[H.GR_CONTAINS][direction][next_location])
		}
	}
}


//**************************************************************

func GetLinksFrom(app,concept_hash string, visited map[string]bool) H.Links {

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
					
					for _, ssfile := range ssfiles {

						// Loop prevention

						if visited[ssfile.Name()] {
							continue
						}
						
						visited[ssfile.Name()] = true

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