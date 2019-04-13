
package main

import (
	"strings"
	"fmt"
	"context"
	"io/ioutil"
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
		"Pod":     "A_pod_named_foo",
		"Process": "myApp_name2",  // insert instance data from env?
		"Version": "1.2.3",
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

	// End loop
	H.SignPost(&ctx,"The end!")
	fmt.Scanln()

}

//**************************************************************

func Test1(ctx context.Context){

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

func I(level int) string {
	var indent string = strings.Repeat("  ",level)
	var s string
	s = fmt.Sprintf("%.3d:%s",level,indent)
	s = indent
	return s
}