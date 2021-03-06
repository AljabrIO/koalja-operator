//
// Copyright © 2019 Aljabr, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"strings"
	"fmt"
	"context"
	"io/ioutil"
	"os/exec"
	"sort"
	"time"

//	H "github.com/AljabrIO/koalja-operator/pkg/history"
	H "history"
)

// ****************************************************************************
// SPLIT ! TOP
// 2. Koalja program starts BELOW ...
// ****************************************************************************

func main() {

	// 1. test cellibrium - need an invariant name (non trivial in cloud)

	ctx := context.Background()
	ctx = H.SetLocationInfo(ctx, map[string]string{
		"Pod":        "A_pod_named_foo",
		"Deployment": "myApp_name2",  // insert instance data from env?
		"Version":    "1.2.3",
	})

	MainLoop(ctx)

	// 2. test koalja, reads pipeline/container_description

   // go routine ...several parallel with same name

}

//**************************************************************
// 1. Cellibrium application test - Non package code
//**************************************************************

func MainLoop(ctx context.Context){

	H.SignPost(&ctx,"MainLoop start").
		PartOf(H.NR("main","function"))

	// Adaptive loop to update context by sensor/input activity
	// Context changes as a result of new environment detected

        // ...other stuff happens
	mk := H.SignPost(&ctx,"Beginning of test code").
		Note("Start process").
		Attributes(H.NR("cellibrium","go package"),H.N("example code"))
	// ...
	mk.Note("look up a name")

	// ...
	H.SignPost(&ctx,"code signpost X"). // what you intended
	Intent("open file X").
		ReliesOn(H.NR("/etc/passed","file")).
		Determines(H.NR("123.456.789.123","dns lookup")).
		FailedBecause("xxx").
		PartOf(H.NR("main","coroutine"))

	// Pass ctx down for logging in lower levels
	go Test1(ctx)
	
	ScanSystem(ctx)

	H.SignPost(&ctx,"Commence testing")

	ConceptConeGeneralizations(ctx)

	DataPipelineExample(ctx)
	
	// End loop
	H.SignPost(&ctx,"The end!")

	H.SignPost(&ctx,"Show the signposts")
	ShowMap()

	time.Sleep(3 * time.Second)
}

//**************************************************************

func ConceptConeGeneralizations(ctx context.Context) {

	H.SignPost(&ctx,"A sideline to test some raw concept mapping").
		PartOf(H.N("Commence testing"))

	H.ConeTest()

	H.SignPost(&ctx,"End of sideline concept test").
		PartOf(H.N("Commence testing"))
}

//**************************************************************

func DataPipelineExample(ctx context.Context){

	m := H.SignPost(&ctx,"Starting Kubernetes deployment").
		PartOf(H.N("Commence testing"))

	time.Sleep(3 * time.Second)

	// Initiate pod for the description

	m.Note("Starting kubernetes pod")
	time.Sleep(3 * time.Second)

	// for each input signal an arrival from sensor

	m.Note("File drop in pipeline")
	time.Sleep(3 * time.Second)

	// Look up data in a model service

	m.Note("Querying data model")
	time.Sleep(3 * time.Second)

	// trigger an output result

	m.Note("Submit transformation result")
	time.Sleep(3 * time.Second)

	// request to build or get from cache
}

//**************************************************************

func Test1(ctx context.Context) {

	m := H.SignPost(&ctx,"TEST1---------").
		PartOf(H.N("Testing suite 1"))

	m.Intent("read whole file of data").
		Determines(H.NR("file://URI","file"))

	_, err := ioutil.ReadFile("file://URI")

	if err != nil {
	        m.Note("file read failed").AddError(err)
	}
	
}

//**************************************************************

func ScanSystem(ctx context.Context) string {
	
	mk := H.SignPost(&ctx,"Run ps command").
		Attributes(H.N("/bin/ps -eo user,pcpu,pmem,vsz,stime,etime,time,args"))
	
	lsCmd := exec.Command("/bin/ps", "-eo", "user,pcpu,pmem,vsz,stime,etime,time,args")
	lsOut, err := lsCmd.Output()
	
	mk.Note("Finished ps command")
	
	if err != nil {
		panic(err)
	}
	return string(lsOut)
}

//**************************************************************

func ShowMap() {

	fmt.Println("This signature of the execution can be compared for interferometry of changes in testing w fixed inputs")

	// Need to sort the keys because map is non-deterministic

	var keys []int
	for k := range H.PROPER_PATHS  {
		keys = append(keys, k)
	}
	
	sort.Ints(keys)
	
	for k := range keys {
		fmt.Println(k," ",H.PROPER_PATHS[k])
		}
}

//**************************************************************

func I(level int) string {
	var indent string = strings.Repeat("  ",level)
	var s string
	s = fmt.Sprintf("%.3d:%s",level,indent)
	s = indent
	return s
}