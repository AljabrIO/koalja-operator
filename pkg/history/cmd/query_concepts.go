
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
		"Pod":        "Koalja_empty_pod",
		"Deployment": "Koalja query_concepts",  // insert instance data from env?
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
		PartOf(H.N("Koalja"))

	if len(args) == 1 || args[1] == "all" {
		ListConcepts(args[0])
	} else {

		app := args[0]
		concept_hash := args[1]


		description := H.ConceptName(app,concept_hash)
		fmt.Printf("\nStories about \"%s\" (%s) in app %s \n\n",description,concept_hash,app)

		DescribeConcept(app,1,concept_hash,description)

		causal_set := GetGeneralizationCone(app,concept_hash)

		// Single path

		GetCausationCone(app,causal_set)

		// apply iteration here too?
		// Downward generalization (specialization) is harmless sample completion
		// Upward generalization (renormalization) has causal implications

		GetSuperCausationCone (app, causal_set)
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

func DescribeConcept (app string, level int, concept_hash string, name string) {
	
	var visited = make(map[string]int)  // loop avoidance
	
	links := GetLinksFrom(app,concept_hash,visited,0)
	
	fmt.Println(I(level),"<begin describe>")

	for l := range links.Bwd[H.GR_EXPRESSES] {

		for next := 0; next < len(links.Bwd[H.GR_EXPRESSES][l]); next++ {
			
			fmt.Printf("%s   -- (%s) --> \"%s\" (%s)\n",
				I(level),
				H.ASSOCIATIONS[l].Bwd,
				H.ConceptName(app,links.Bwd[H.GR_EXPRESSES][l][next].Name),
				links.Bwd[H.GR_EXPRESSES][l][next].Name)
		}
	}
	
	for l := range links.Fwd[H.GR_EXPRESSES] {
		
		for next := 0; next < len(links.Fwd[H.GR_EXPRESSES][l]); next++ {
			
			fmt.Printf("%s   -- (%s) -->  \"%s\" (%s)\n",
				I(level),
				H.ASSOCIATIONS[l].Fwd,
				H.ConceptName(app,links.Fwd[H.GR_EXPRESSES][l][next].Name),
				links.Fwd[H.GR_EXPRESSES][l][next].Name)
		}
	}

	fmt.Println(I(level),"<end describe>")
}

//**************************************************************

func GetGeneralizationCone (app string, concept_hash string) []string {
	
	// First establish how many different up/down CONTAINS relations attach to the anchor concept
	// These form separate graphs

	var visited = make(map[string]int)  // loop avoidance
	var level int = 1
	visited[concept_hash] = 1

	nodelinks := GetLinksFrom(app,concept_hash,visited,0)

	// Detail resolution
	fwd_cone := RetardedCone(app,H.GR_CONTAINS,nodelinks,visited)

	// Increasing entropy
	bwd_cone := AdvancedCone(app,H.GR_CONTAINS,nodelinks,visited)

	fmt.Println("")
	fmt.Println(I(level),"<begin generalization cone>")
	region := ShowCone(app,concept_hash,fwd_cone,bwd_cone)
	fmt.Println(I(level),"<end generalization cone>")

return region
}

//**************************************************************

func GetCausationCone (app string, cset []string) {
	
	var visited = make(map[string]int)  // loop avoidance

	fmt.Println("")
	fmt.Println(I(1),"<begin CAUSE>")

	for c := range cset {
		concept_hash := cset[c]

		nodelinks := GetLinksFrom(app,concept_hash,visited,0)

		fwd_cone := RetardedCone(app,H.GR_FOLLOWS,nodelinks,visited)
		bwd_cone := AdvancedCone(app,H.GR_FOLLOWS,nodelinks,visited)

		ShowCone(app,concept_hash,fwd_cone,bwd_cone)		
	}
	fmt.Println(I(1),"<end CAUSE>")
	
}

//**************************************************************

func GetSuperCausationCone (app string, cset []string) {
	
	var visited = make(map[string]int)  // loop avoidance

	fmt.Println("")
	fmt.Println(I(1),"<begin GENERALIZED CAUSE>")

	for c := range cset {
		concept_hash := cset[c]
		nodelinks := GetLinksFrom(app,concept_hash,visited,0)
		fwd_cone := RetardedHistories(app,nodelinks,visited)
		bwd_cone := AdvancedHistories(app,nodelinks,visited)

		ShowCone(app,concept_hash,fwd_cone,bwd_cone)		
	}
	fmt.Println(I(1),"<end GENERALIZED CAUSE>")
	
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
Neighbou
   // we are only doing ascent, so don't need to mix fwd/bwd channels in same process
   // no antiparticles in reasoning...?

*/

//**************************************************************

func RetardedCone(app string, sttype int, links H.Links, visited map[string]int) H.NeighbourConcepts {

	var maplist, nextlist, fwd_cone H.NeighbourConcepts
	var depth int = 0
	
	init := links.Fwd[sttype]
	nextlist = make(H.NeighbourConcepts,0)

	for maplist = init; Neighbours(maplist); maplist = nextlist {

		fwd_cone = JoinNeighbours(fwd_cone,maplist)
		nextlist = make(H.NeighbourConcepts,0)
		depth++
		
		for linktype := range maplist {			
			for node := 0; node < len(maplist[linktype]); node++ {
				neighbours := GetLinksFrom(app,maplist[linktype][node].Name,visited,depth)
				nextlist = JoinNeighbours(nextlist,neighbours.Fwd[sttype])
			}
		}
	}

	return fwd_cone
}

//**************************************************************

func AdvancedCone(app string, sttype int, links H.Links, visited map[string]int) H.NeighbourConcepts {

	var maplist, nextlist, bwd_cone H.NeighbourConcepts
	var depth int = 0

	init := links.Bwd[sttype]
	nextlist = make(H.NeighbourConcepts,0)

	for maplist = init; Neighbours(maplist); maplist = nextlist {

		bwd_cone = JoinNeighbours(bwd_cone,maplist)
		nextlist = make(H.NeighbourConcepts,0)
		depth++

		for linktype := range maplist {
			for node := 0; node < len(maplist[linktype]); node++ {
				neighbours := GetLinksFrom(app,maplist[linktype][node].Name,visited,depth)
				nextlist = JoinNeighbours(nextlist,neighbours.Bwd[sttype])
			}
		}
	}

	return bwd_cone
}

//**************************************************************

func RetardedHistories(app string,links H.Links, visited map[string]int) H.NeighbourConcepts {

	var maplist, nextlist, genlist, speclist, wavefront, fwd_cone H.NeighbourConcepts
	var depth int = 0
	
	init := links.Fwd[H.GR_FOLLOWS]

	nextlist = make(H.NeighbourConcepts,0)

	for maplist = init; Neighbours(maplist); maplist = wavefront {

		fwd_cone = JoinNeighbours(fwd_cone,maplist)
		nextlist = make(H.NeighbourConcepts,0)
		genlist = make(H.NeighbourConcepts,0)
		speclist = make(H.NeighbourConcepts,0)
		wavefront = make(H.NeighbourConcepts,0)
		depth++

		for linktype := range maplist {			
			for node := 0; node < len(maplist[linktype]); node++ {
				neighbours := GetLinksFrom(app,maplist[linktype][node].Name,visited,depth)

				nextlist = JoinNeighbours(nextlist,neighbours.Fwd[H.GR_FOLLOWS])
				speclist = JoinNeighbours(speclist,neighbours.Fwd[H.GR_CONTAINS])
				genlist = JoinNeighbours(genlist,neighbours.Bwd[H.GR_CONTAINS])
			}

			wavefront = JoinNeighbours(wavefront,genlist)
			wavefront = JoinNeighbours(wavefront,speclist)
			wavefront = JoinNeighbours(wavefront,nextlist)
		}
	}

	return fwd_cone
}

//**************************************************************

func AdvancedHistories(app string,links H.Links, visited map[string]int) H.NeighbourConcepts {

	var maplist, nextlist, genlist, speclist, wavefront, fwd_cone H.NeighbourConcepts
	var depth int = 0
	
	init := links.Bwd[H.GR_FOLLOWS]

	nextlist = make(H.NeighbourConcepts,0)

	for maplist = init; Neighbours(maplist); maplist = wavefront {

		fwd_cone = JoinNeighbours(fwd_cone,maplist)
		nextlist = make(H.NeighbourConcepts,0)
		genlist = make(H.NeighbourConcepts,0)
		speclist = make(H.NeighbourConcepts,0)
		wavefront = make(H.NeighbourConcepts,0)
		depth++

		for linktype := range maplist {			
			for node := 0; node < len(maplist[linktype]); node++ {
				neighbours := GetLinksFrom(app,maplist[linktype][node].Name,visited,depth)
				nextlist = JoinNeighbours(nextlist,neighbours.Bwd[H.GR_FOLLOWS])
				speclist = JoinNeighbours(speclist,neighbours.Fwd[H.GR_CONTAINS])
				genlist = JoinNeighbours(genlist,neighbours.Bwd[H.GR_CONTAINS])
			}

			wavefront = JoinNeighbours(wavefront,genlist)
			wavefront = JoinNeighbours(wavefront,speclist)
			wavefront = JoinNeighbours(wavefront,nextlist)
		}
	}

	return fwd_cone
}

// ************************************************************************

func ShowCone(app string,concept_hash string, fcone, bcone H.NeighbourConcepts) []string {

	// Show and return the complete propagation cone

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

				fmt.Printf("\n%s (%s) --f(%s)--> \"%s\" (%s)\n",
					I(3+2*fcone[linktype][fnode].Depth),
					H.ConceptName(app,fcone[linktype][fnode].Prev),
					H.ASSOCIATIONS[linktype].Fwd,
					H.ConceptName(app,fcone[linktype][fnode].Name),
					fcone[linktype][fnode].Name)

				region = append(region,fcone[linktype][fnode].Name)
			}
		}

		if bcone[linktype] != nil {
			for bnode := 0; bnode < len(bcone[linktype]); bnode++ {

				fmt.Printf("\n%s (%s) --b(%s)--> \"%s\" (%s)\n",
					I(3+2*bcone[linktype][bnode].Depth),
					H.ConceptName(app,bcone[linktype][bnode].Prev),
					H.ASSOCIATIONS[linktype].Bwd,
					H.ConceptName(app,bcone[linktype][bnode].Name),
					bcone[linktype][bnode].Name)

				region = append(region,bcone[linktype][bnode].Name)
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

	// not currently idempotent, so multiset union (inefficient) -- this allows superposition

	var result H.NeighbourConcepts = make(H.NeighbourConcepts)

	for t := range master {
		
		result[t] = make([]H.Pair,0)
	
		for n := 0; n < len(master[t]); n++ {

			result[t] = append(result[t],master[t][n])
		}
	}

	for t := range delta {
		
		if result[t] == nil {
			result[t] = make([]H.Pair,0)
		}
	
		for n := 0; n < len(delta[t]); n++ {
			result[t] = append(result[t],delta[t][n])
		}
	}

return result
}

//**************************************************************

func GetLinksFrom(app,concept_hash string, visited map[string]int, depth int) H.Links {

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

			var sttype int = 0
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

						/* Loop prevention */

						if visited[ssfile.Name()] > 0 {
							if visited[ssfile.Name()] < 2 {
							} else {
								//fmt.Println("    ----- LOOP!",H.ConceptName(app,ssfile.Name()))
								continue
							}
						}
						
						visited[ssfile.Name()]++

						var a H.Pair
						a.Name = ssfile.Name()
						a.Prev = concept_hash
						a.Depth = depth

						if sttype < 0 {
							links.Bwd[-sttype][index] = append(links.Bwd[-sttype][index],a)
						} else {
							links.Fwd[sttype][index] = append(links.Fwd[sttype][index],a)
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