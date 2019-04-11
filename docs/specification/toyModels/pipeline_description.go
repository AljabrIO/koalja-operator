
//
// This file contains a mashup of parts to avoid technicality as a POC
//
// 1. cellibrium in golang - to be a package
// 2. koalja pipeline language - POC parser, compiler
//

// ***************************************************************************
//*
//* Cellibrium v2 in golang ... 
//* experimenting ... MB
//*
// ***************************************************************************

package main

import (
	"strings"
	"unicode"
	"sync/atomic"
	"io/ioutil"
	"bufio"
	"context"
	"time"
	"runtime"
	"crypto/sha1"
	"fmt"
	"os"
)

// ***************************************************************************
// Interior timeline
// ***************************************************************************

// This is used to coordinate a forensic monotonic timelines,
// in spite of recursive context, because Context can't track time
// This will end up being irrelevant to graphDB, has only local significance,
// except for relative order

var INTERIOR_TIME int64 = 0 

// Trajectory(nowtime, event, previous time)

// The timeline is an array of tuples TUPLES[INTERMIOR_TIME]

// Should we have a code book/dictionary for compressing?

// ***************************************************************************

var logctx string

type Name string
type List []string
type BreadBoard map[string]List

type Container struct {
	alias string
	image_version string
	image string
	command string
	args []string
}

type NameAndRole struct {
	name string
	role string
	hub string
}

type Coordinates struct {
	proper int   // Proper time
	t int32
	x NameAndRole // location hub
}

type RM struct {            // Reference marker
	description NameAndRole
	xt Coordinates
	ctx context.Context
	previous int // copied from atomic int64
}

type Association struct {
 	key  int      // index
	STtype int    // oriented type, - reverses oriention
	neg bool      // negate or NOT the relation meaning
	fwd  string   // forward oriented meaning
	bwd  string   // backward " 
}

// ****************************************************************************

type LogContext struct {
	tf *os.File
	gf *os.File
	val string           // signposts
	t time.Time          // Start time
	proper int
	prefix string        // unique channel identifer
}

// ***************************************************************************
// Invariants
// ****************************************************************************

const GR_NEAR int      = 1  // approx like
const GR_FOLLOWS int   = 2  // i.e. influenced by
const GR_CONTAINS int  = 3 
const GR_EXPRESSES int = 4  // represents, etc
const GR_CONTEXT int   = 5  // approx like
const ALL_CONTEXTS string = "any"

const (
	hasrole int = 20
	expresses int = 15
	promises int = 16
	follows int = 5
	contains int = 1
	partof int = 2
	uses int = 13
	alias int = 26

	PROCESS_MARKER string = "process reference marker"
	SYS_ERR_MSG string = "system error message"
	UNSPEC_ROLE string = "unspecified role"
)

var (
	ASSOCIATIONS = [29]Association{

		{0,0, false, "unknown promise", "unknown promise"},
		{1,GR_CONTAINS, false,"contains","belongs to or is part of"},
		{2,-GR_CONTAINS, false,"is part of","contains"},
		{3,-GR_CONTAINS,true,"violates","is violated by"},
		{4,GR_CONTAINS, false, "generalizes","is a special case of"},
		{5,GR_FOLLOWS,false,"followed after","was preceded by"},
		{6,GR_FOLLOWS,false,"originates from","is the source or origin of"},
		{7,GR_FOLLOWS,false,"provided by","provides"},
		{8,GR_FOLLOWS,false,"maintained by","maintains"},
		{9,GR_FOLLOWS,false,"depends on","may determine"},
		{10,GR_FOLLOWS,false,"was started by","started"},
		{11,GR_FOLLOWS,false,"connected to","reponded to"},
		{12,GR_FOLLOWS,false,"caused by","may cause"},
		{13,GR_FOLLOWS,false,"intends to use","used by"},
		{14,GR_EXPRESSES,false,"is called","is a name for"},
		{15,GR_EXPRESSES,false,"expresses an attribute","is an attribute of"},
		{16,GR_EXPRESSES,false,"promises/intends","is intended/promised by"},
		{17,GR_EXPRESSES,false,"has an instance or particular case","is a particular case of"},
		{18,GR_EXPRESSES,false,"has value or state","is the state or value of"},
		{19,GR_EXPRESSES,false,"has argument or parameter","is a parameter or argument of"},
		{20,GR_EXPRESSES,false,"has the role of","is a role fulfilled by"},
		{21,GR_EXPRESSES,false,"has the outcome","is the outcome of"},
		{22,GR_EXPRESSES,false,"has function","is the function of"},
		{23,GR_EXPRESSES,false,"has constraint","constrains"},
		{24,GR_EXPRESSES,false,"has interpretation","is interpreted from"},
		{25,GR_NEAR,false,"seen concurrently with","seen concurrently with"},
		{26,GR_NEAR,false,"also known as","also known as"},
		{27,GR_NEAR,false,"is approximately","is approximately"},
		{28,GR_NEAR,false,"may be related to","may be related to"},

	}
)

// ****************************************************************************

var GR_DAY_TEXT = []string{
        "Monday",
        "Tuesday",
        "Wednesday",
        "Thursday",
        "Friday",
        "Saturday",
        "Sunday",
    }
        
var GR_MONTH_TEXT = []string{
        "January",
        "February",
        "March",
        "April",
        "May",
        "June",
        "July",
        "August",
        "September",
        "October",
        "November",
        "December",
}
        
var GR_SHIFT_TEXT = []string{
        "Night",
        "Morning",
        "Afternoon",
        "Evening",
    }

// ****************************************************************************

func HereAndNow() string {

	// Lookup, expand, graph

	then := time.Now()

	year := fmt.Sprintf("Yr%d",then.Year())
	month := GR_MONTH_TEXT[int(then.Month())-1]
	day := then.Day()
	hour := fmt.Sprintf("Hr%02d",then.Hour())
	mins := fmt.Sprintf("Min%02d",then.Minute())
	quarter := fmt.Sprintf("Q%d",then.Minute()/15 + 1)
	shift :=  fmt.Sprintf("%s",GR_SHIFT_TEXT[then.Hour()/6])
	//secs := then.Second()
	//nano := then.Nanosecond()
	dow := then.Weekday()

        interval_start := (then.Minute() / 5) * 5
        interval_end := (interval_start + 5) % 60
        minD := fmt.Sprintf("Min%02d_%02d",interval_start,interval_end)


	var hub string = fmt.Sprintf(" on %s %s %d %s %s at %s %s %s %s",shift,dow,day,month,year,hour,mins,quarter,minD)
	var hereandnow = Where(3) + hub
	return hereandnow
}

// ****************************************************************************

func CodeLocation() NameAndRole { // User function
	return NR(Where(1),"code position")
}

// ****************************************************************************

func Where(depth int) string {
        // Interal usage
	p,name,line, ok := runtime.Caller(depth)
	
	var location string

	if ok {
		var funcname = runtime.FuncForPC(p).Name()
		location = fmt.Sprintf("in function %s in file %s at line %d",funcname,name,line)
	} else {
		location = "unknown origin"
	}

	return location
}


// ****************************************************************************
// Explain
// ****************************************************************************

func WriteChainBlock(ctx context.Context,t int32, s string, propertime int, previoustime int) {

	var hub string 
	lctx, ok := GetLogContext(ctx)
	if ok {
		hub = fmt.Sprintf("(%s,%d,%d,%d,%s)\n",lctx.prefix,t,propertime,previoustime,s)
		lctx.tf.WriteString(hub)
	}
}

// ****************************************************************************

func WriteAddendum(ctx context.Context, s string, propertime int, previoustime int) {

	var hub string 
	lctx, ok := GetLogContext(ctx)
	if ok {
		hub = fmt.Sprintf("(%s,-,%d,%d,%s)\n",lctx.prefix,propertime,previoustime,s)
		lctx.tf.WriteString(hub)
	}
}

// ****************************************************************************

func RefMarker(ctx *context.Context, s string) RM {

	var rm RM 
	rm.description = NR(s,PROCESS_MARKER)
	AnnotateNR(*ctx,rm.description)
	rm.ctx = *ctx
	rm.previous = GetPrevious(*ctx)
	rm.xt = GetCoordinates(*ctx) 
	// Set the latest proper time
	*ctx = SetPrevious(*ctx,rm.xt.proper)
	WriteChainBlock(*ctx,rm.xt.t,rm.description.hub,rm.xt.proper,rm.previous)
	l := CodeLocation()
	Relation(*ctx,true,rm.description.hub,expresses,l.hub)
	return rm
}

// ****************************************************************************

func (rm RM) Note(ctx *context.Context,s string) RM {

// Instead of a string there should be an interface to accept an INT or a free string
// where the INT points to a list of standard strings already in the DB

	var nm NameAndRole = NR(s,PROCESS_MARKER)
	AnnotateNR(*ctx,nm)
	Relation(*ctx,true,nm.hub,follows,rm.description.hub)
	Relation(*ctx,true,rm.description.hub,contains,nm.hub)
	xt := GetCoordinates(*ctx)
	WriteChainBlock(*ctx,xt.t,nm.hub,xt.proper,rm.previous)
	return rm
}

// ****************************************************************************

func (m RM) Attributes(attr ...NameAndRole) RM {

	for i := 0; i < len(attr); i++ {
		Relation(m.ctx,true,m.description.hub,expresses,attr[i].hub)
	}
	return m
}

// ****************************************************************************

func Relation(ctx context.Context,notnot bool, n string,assoc int,subject string) {

	var hub string

	if notnot {
		hub = fmt.Sprintf("(%d,{%s}, %s, {%s})",ASSOCIATIONS[assoc].STtype,n,ASSOCIATIONS[assoc].fwd,subject)
	} else {
		hub = fmt.Sprintf("(%d,{%s}, NOT %s, {%s})",ASSOCIATIONS[assoc].STtype,n,ASSOCIATIONS[assoc].fwd,subject)
	}

	lctx, ok := GetLogContext(ctx)
	if ok {
		lctx.gf.WriteString(hub+"\n")
	}
}

// ****************************************************************************

func (m RM) Role(role string) RM {

	var logmsg string = "-in role " + role
	WriteAddendum(m.ctx,logmsg,m.xt.proper, m.previous)

	Relation(m.ctx,true,m.description.hub,hasrole,role)
	return m
}

// ****************************************************************************

func (m RM) NotRole(role string) RM {

	var logmsg string = "-NOT in role " + role
	WriteAddendum(m.ctx,logmsg,m.xt.proper, m.previous)
	Relation(m.ctx,false,m.description.hub,hasrole,role)
	return m
}

// ****************************************************************************

func (m RM) Used(nr NameAndRole) RM {

	var logmsg string = "-used " + nr.role + ": " + nr.name
	WriteAddendum(m.ctx,logmsg,m.xt.proper, m.previous)
	Relation(m.ctx,true,m.description.hub,uses,nr.hub)
	return m
}

// ****************************************************************************

func (m RM) PartOf(nr NameAndRole) RM {

	var logmsg string = "- part of " + nr.role + ": " + nr.name
	WriteAddendum(m.ctx,logmsg,m.xt.proper, m.previous)
	Relation(m.ctx,true,m.description.hub,partof,nr.hub)
	return m
}

// ****************************************************************************

func (m RM) Contains(nr NameAndRole) RM {

	//var logmsg string = nr.role + ": " + nr.name
	//WriteChainBlock(m.ctx,m.xt.t,logmsg,m.xt.proper, m.previous)

	Relation(m.ctx,true,m.description.hub,contains,nr.hub)
	return m
}

// ****************************************************************************

func (m RM) FailedToUse(nr NameAndRole) RM {

	var logmsg string = "-failure as " + nr.role + ": " + nr.name
	WriteAddendum(m.ctx,logmsg,m.xt.proper, m.previous)

	Relation(m.ctx,false,m.description.hub,uses,nr.hub)
	return m
}

// ****************************************************************************

func (m RM) Intent(s string) RM {

	var nm NameAndRole = NR(s,"intended outcome")
	AnnotateNR(m.ctx,nm)
	Relation(m.ctx,true,m.description.hub,promises,s)
	WriteChainBlock(m.ctx,m.xt.t,nm.hub,m.xt.proper,m.previous)
	return m
}

// ****************************************************************************

func (m RM) FailedIntent(s string) RM {
	return m.NotRole(s)
}

// ****************************************************************************

func Hub(name,role string) string {
	return "[" + role + ": " + name + "]"
}

// ****************************************************************************

func NR(name string, role string) NameAndRole {
	var n NameAndRole
	n.name = name
	n.role = role
	n.hub = Hub(name,role)
	return n
}

// ****************************************************************************

func AnnotateNR(ctx context.Context,n NameAndRole){

	/*if AlreadyDefined(n.hub){
	// SHOULD WE TOKENIZE?

	}*/

	Relation(ctx,true,n.hub,hasrole,n.role)
	Relation(ctx,true,n.hub,expresses,n.name)
}

// ****************************************************************************

func N(name string) NameAndRole {
	return NR(name,UNSPEC_ROLE)
}

// ****************************************************************************

func (m RM) AddError(err error) RM {

	n := NR(err.Error(),SYS_ERR_MSG)
	AnnotateNR(m.ctx,n)
	Relation(m.ctx,true,n.hub,hasrole,n.role)
	Relation(m.ctx,true,m.description.hub,expresses,n.hub)
	Relation(m.ctx,true,n.hub,follows,m.description.hub)
	
	return m
}

// ****************************************************************************

func URI(s string) NameAndRole {
	return NR(s,"object URI")
}

// ****************************************************************************

func GetLocation(ctx context.Context) string {

	val, _ := GetLogContext(ctx)
	return val.val
}

// ****************************************************************************

func SetPrevious(ctx context.Context, propertime int) context.Context {

	lctx, _ := ctx.Value(logctx).(LogContext)
	lctx.proper = propertime
	return context.WithValue(ctx, logctx, lctx)
}

// ****************************************************************************

func GetCoordinates(ctx context.Context) Coordinates {
	return Tick(ctx)
}

// ****************************************************************************

func GetPrevious(ctx context.Context) int {

	lctx, _ := ctx.Value(logctx).(LogContext)
	return lctx.proper
}

// ****************************************************************************

// Increment a proper time interval

func Tick(ctx context.Context) Coordinates {

	// Since proper time gets reset by new exec, we should keep x,t clear
	// And x should depend on the execution is possible
	var c Coordinates
	var next int64 = atomic.AddInt64(&INTERIOR_TIME,1)
	c.proper = int(next)
	c.t = int32(time.Now().Unix())
	hub := GetLocation(ctx)
 	c.x = NR(hub,PROCESS_MARKER)
	return c
}

// ****************************************************************************
// * Context
// ****************************************************************************

func LocationInfo(ctx context.Context, m map[string]string) context.Context {

	var lctx LogContext

	// If the file doesn't exist, create it, or append to the file
	var err error


	// Transaction log
	lctx.tf, err = os.OpenFile("/tmp/cellibrium_ts.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("ERROR ",err)
	}

	// Graph DB
	lctx.gf, err = os.OpenFile("/tmp/cellibrium_gr.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("ERROR ",err)
	}

	var keys,values,location string
	var ok bool = false

	keys = "("
	values = "("

	for k, v := range m {
		ok = true
		keys = keys + k + ","
		values = values + v + ","
	}
	keys = keys + ")"
	values = values + ")"
	
	if ok {
		location = keys + " = " + values
	} else {
		location = "undeclared process location"
	}


	// combine and sha1 all location markers, take first 10 chars as prefix code

	var unique string = location
	h := sha1.New()
	h.Write([]byte(unique))
	bs := h.Sum(nil)
	lctx.prefix = fmt.Sprintf("%.6x",bs)
	lctx.proper = 0
	lctx.val = location
	rctx := context.WithValue(ctx, logctx, lctx)

	Relation(rctx,true,location,alias,lctx.prefix)

	for k, v := range m {
		kv := k + ":" + v
		Relation(rctx,true,location,expresses,kv)
		Relation(rctx,true,kv,hasrole,k)
		Relation(rctx,true,kv,expresses,v)
	}

	return rctx
}

// ****************************************************************************

func GetLogContext(ctx context.Context) (LogContext, bool) {

	lctx, ok := ctx.Value(logctx).(LogContext)
	return lctx,ok
}

// ****************************************************************************

func UpdateSensorContext(ctx context.Context) context.Context {

	/* Scan sensors for 
              i) software version changes, 
             ii) new inputs, 
             iii)new targets
        */

	// Should we comput diff here, or later?

	// Build in ML for normal state of context flags

	return ctx
}

// ****************************************************************************

func CloseLog(ctx context.Context) {
	lctx, _ := GetLogContext(ctx)
	lctx.tf.Close();
	lctx.gf.Close();
}

//**************************************************************
// 1. Cellibrium application test - Non package code
//**************************************************************

func MainLoop(ctx context.Context){

	RefMarker(&ctx,"MainLoop start").
		PartOf(NR("main","function"))

	// Adaptive loop to update context by sensor/input activity
	// Context changes as a result of new environment detected

	// Start loop
	ctx = UpdateSensorContext(ctx)

        // ...other stuff happens
	mk := RefMarker(&ctx,"Beginning of test code").
		Role("Start process").
		Attributes(NR("cellibrium","go package"),N("example code"))
	// ...
	mk.Note(&ctx,"look up a name")

	// ...
	RefMarker(&ctx,"code signpost X"). // what you intended
	Intent("open file X").
		Used(NR("/etc/passed","file")).
		Used(NR("123.456.789.123","dns lookup")).
		FailedToUse(N("cc")).
		FailedIntent("xxx").
		PartOf(NR("main","function")).
		Contains(NR("Test1","test function"))

	// Pass ctx down for logging in lower levels
	Test1(ctx)

	// End loop
	RefMarker(&ctx,"The end!")
}

//**************************************************************

func Test1(ctx context.Context){

	m := RefMarker(&ctx,"TEST1---------").
		PartOf(N("Testing suite"))

	m.Intent("read whole file of data").
		Used(NR("file://URI","file"))

	_, err := ioutil.ReadFile("file://URI")

	if err != nil {
	        m.Note(&ctx,"file read failed").AddError(err)
	}
	
}

// ****************************************************************************
// SPLIT ! TOP
// 2. Koalja program starts BELOW ...
// ****************************************************************************

func main() {

	// 1. test cellibrium

	ctx := context.Background()
	ctx = LocationInfo(ctx, map[string]string{
		"Pod":     "A_pod_named_foo",
		"Process": "myApp_name2",  // insert instance data from env?
		"Version": "1.2.3",
	})

	MainLoop(ctx)

	// 2. test koalja, reads pipeline/container_description

	ParsePipeline(ctx)
}


//**************************************************************
//* Koalja PIPELINE SPEC package
//**************************************************************

func ParsePipeline(ctx context.Context){

	m := RefMarker(&ctx,"Parser").
		Intent("Parse a pipeline description file and generate YAML").
		PartOf(N("Testing suite"))

	buffer, err := ioutil.ReadFile("./pipeline_description")

	if err != nil {
		m.Note(&ctx,"file read failed").AddError(err)
	}

	namespace, name, yaml := GetPipelineDefinition(ctx,buffer)

	// Print the k8s YAML

	fmt.Println(I(0),"apiVersion: koalja.aljabr.io/v1alpha1")
	fmt.Println(I(0),"kind: Pipeline")
	fmt.Println(I(0),"metadata:")
	fmt.Println(I(1),"name: "+name)
	fmt.Println(I(1),"namespace: "+namespace)
	fmt.Println(I(0),"spec:")
	fmt.Println(I(1),"tasks:")

	for i := 0; i < len(yaml); i++ {
		fmt.Println(yaml[i])
	}
}

//**************************************************************

func GetPipelineDefinition(ctx context.Context, input []byte) (string,string,[]string) {

	var yaml []string = make([]string, 0)
	var name, namespace string

	bb := make(BreadBoard)
	i := 0

	// Begin parsing the breadboard spec

NewTask:
	for i < len(input) {

		//var in,out,task string
		var parin int
		var operator []byte
		var paren[2] []byte
		var nsn []byte

		paren[0] = make([]byte, 0)
		paren[1] = make([]byte, 0)
		operator = make([]byte, 0)
		nsn = make([]byte, 0)

		parin = 0

		// j: get a single statement, k break line into components

		for j := i; j < len(input); j++ {

			switch input[j] {

			case byte('['): 

				for k := j+1; k < len(input); k++ {
					
					if name != "" {
						fmt.Println("Name of pipeline redefined")
					}

					if input[k] == byte(']') {
						j = k+1
						if strings.Contains(string(nsn),":") {
							a := strings.Split(string(nsn),":")
							namespace = a[0]
							name = a[1]
						} else {
							name =string(nsn)
							namespace = "default"
						}
						break
					} else {
						nsn = append(nsn,input[k])
					}
				}

			case byte('('): 
				for k := j+1; k < len(input); k++ {

					if input[k] == byte(')') {
						k++
						j = k
						break
					}

					if input[k] == byte('\n') {
						continue
					}

					if parin < 2 {
						paren[parin] = append(paren[parin],input[k])
					}

					if k >= len(input){
						break NewTask
					}
				}
				parin++

			case byte('\n'): input[j] = byte(' ')

			case byte('#'): 
				for k := j+1; k < len(input); k++ {

					if input[k] == byte('\n') {
						j = k
						input[k] = byte(' ')
						break
					}

					if k >= len(input) {
						break NewTask
					}
				}
			}
			
			operator = append(operator,input[j])

			i = j+1

			if (parin == 2) {
				y := ReadOneTask(ctx,bb,paren[0],operator,paren[1])
				for i := 0; i < len(y); i++ {
					yaml = append(yaml,y[i])
				}
				break
			}
		}
	}

	// Suffix info

	yaml = append(yaml,I(2)+"links:")

	wires := AddWiring(ctx,bb)

	for i := 0; i < len(wires); i++ {
		yaml = append(yaml,wires[i])
	}
	
	// Suffix info

	yaml = append(yaml,I(2)+"types:")
	yaml = append(yaml,I(3)+"- name: singleTextFile")
	yaml = append(yaml,I(4)+  "protocol: File")
	yaml = append(yaml,I(4)+  "format: Text")

	return namespace, name, yaml
}

//**************************************************************

func LookupContainerDef(ctx context.Context, search string, in []byte, out []byte) []string {

	m := RefMarker(&ctx,"Container lookup").
		Intent("Import container description from database")

	file, err := os.Open("./container_description")
	
	if err != nil {
		m.Note(&ctx,"file read failed").AddError(err)
	}
	
	defer file.Close()

	var c Container	
	var found bool = false

	scanner := bufio.NewScanner(file)
	yaml := make([]string,0)
	c.args = make([]string,0)

	for scanner.Scan() {
		var alias, key, value string
		line := string(scanner.Text())
		
		if strings.HasPrefix(line,"(") {

			fmt.Sscanf(line,"( %s", &alias)

			if alias == search || alias == "defaults" {

				if alias == search {
					found = true
					c.args = nil
				}

				for scanner.Scan() {
					line = string(scanner.Text())

					if strings.HasPrefix(line,"#") {
						continue
					}

					if strings.HasPrefix(line,")") {
						c.alias = alias
						goto Done
					}
					
					l := strings.Split(line,":")
					key = strings.TrimSpace(l[0])
					value = strings.TrimSpace(l[1])

					if key == "command" {
						c.command = value
					}

					if key == "image" {
						c.image = value
					}

					if key == "image_version" {
						c.image_version = value
					}

					if key == "arg" {
						c.args = append(c.args,value)
					}
					
				} 
			}
		}
	Done:
	}

	if err := scanner.Err(); err != nil {
		m.Note(&ctx,"reading line of file failed").AddError(err)
	}

	if !found {
		m.Note(&ctx,"No container definition found").
			FailedToUse(NR(search,"container"))
		return append(yaml,"<No Such Container> !!")
	}

	// Get the wire endings to sub for IN and OUT, generate YAML

	var inarray []string = StripWires(in)
	var outarray []string = StripWires(out)	
	var s string
	
	s = fmt.Sprintf("image: %s:%s", c.image,c.image_version)
	yaml = append(yaml,s)
	s = fmt.Sprintf("command:")
	yaml = append(yaml,s)

	// Expand IN and OUT wire ends, might be in command string or in args

	s = fmt.Sprintf("- %s", c.command)
	yaml = append(yaml,s)

	for i := 0; i < len(c.args); i++ {

		s = c.args[i]

		// substitute IN/OUT if on separate line, then split the args one per line

		if s == "<IN>" {
			for j := 0; j < len(inarray); j++ {
				s = fmt.Sprintf("%s", "{{.inputs."+inarray[j]+".path}}")
				c.args[i] = s

			}
		} else if strings.Contains(s,"<IN>") {
			var expand string
			for j := 0; j < len(inarray); j++ {
				expand = expand + "{{.inputs."+inarray[j]+".path}} "
			}
			s = strings.Replace(s,"<IN>",expand,1)
			c.args[i] = s
		}
		
		if s == "<OUT>" {
			for j := 0; j < len(outarray); j++ {
				s = fmt.Sprintf("%s", "{{.inputs."+outarray[j]+".path}}")
				c.args[i] = s
			}
		} else if strings.Contains(s,"<OUT>") {
			var expand string
			for j := 0; j < len(outarray); j++ {
				expand = expand + "{{.inputs."+outarray[j]+".path}} "
			}
			s = strings.Replace(s,"<OUT>",expand,1)
			c.args[i] = s
		}

		s = fmt.Sprintf("- %s", s)
		yaml = append(yaml,s)
	}

	return yaml
}

//**************************************************************

func StripWires(array []byte) []string {

	var a []string = strings.FieldsFunc(string(array),InOutSplit)
	var r []string

	for i := 0; i < len(a); i++ {

		// Might need to revise these types - files already deprecated

		if strings.HasPrefix(a[i],"files") ||         // file passing
   	 	        strings.HasPrefix(a[i],"query") ||   // explicit DB/service query
			strings.HasPrefix(a[i],"implicit") { // implicit DB/service query
			continue
		}

		r = append(r,a[i])
	}

	return r
}

//**************************************************************

func ReadOneTask(ctx context.Context, bb BreadBoard, in []byte, op []byte,out []byte) []string {

	m := RefMarker(&ctx,"ReadOneTask").
		Intent("Extract inputs, outputs, operator, and policy from a declaration")

	// Generate output assembly to yaml list
	var yaml []string = make([]string, 0)

	// Split: (in) op (out) into parts

	var opa []string = strings.FieldsFunc(string(op),InOutSplit)

	if len(opa) > 1 {
		m.FailedIntent("Too many operators in the pipeline junction")
		fmt.Println("Too many operators in junction: ",opa)
		os.Exit(1)
	}
		
	operator := string(opa[0])

	// get inputs ... if there's an "in" input, then add a filedrop/ingress
	// if the name starts with "in", create a filedrop for it

	drops := CheckFileDrops(ctx,bb,in)

	for i := 0; i < len(drops); i++ {
		yaml = append(yaml,drops[i])
	}

	// Generate the output
	// Start with the NAME of the transformer/task
	
	yaml = append(yaml,I(2)+"- name: "+string(operator))
	
	// generate inputs

	var inputs []string = HandleIOPolicy(ctx,bb,operator,in)

	if (len(inputs)) > 0 {
		yaml = append(yaml,I(3)+"inputs:")
	}

	for i := 0; i < len(inputs); i++ {
		yaml = append(yaml,I(3)+inputs[i])
	}

	// generate outputs

	var outputs []string = HandleIOPolicy(ctx,bb,operator,out)

	if (len(outputs)) > 0 {
		yaml = append(yaml,I(3)+"outputs:")
	}

	for i := 0; i < len(outputs); i++ {
		yaml = append(yaml,I(3)+outputs[i])
	}

	// Append executor or filedrop ingress node
	
	yaml = append(yaml,I(2)+"executor:")

	containers := LookupContainerDef(ctx,operator,in,out)

	for i := 0; i < len(containers); i++ {
		yaml = append(yaml,I(3)+containers[i])
	}

	return yaml
}

//**************************************************************

func CheckFileDrops(ctx context.Context, bb BreadBoard, in []byte) []string {

	var array []string = strings.FieldsFunc(string(in),InOutSplit)

	var yaml []string = make([]string, 0)

	for i := 0; i < len(array); i++ {

		s := strings.Split(array[i],"[")
		name := s[0]
		fmt.Println("NAME: ",name)
		if strings.HasPrefix(name,"in") {
			yaml = append(yaml,I(2)+"- name: Drop"+name)
			yaml = append(yaml,I(3)+"outputs: ")
			yaml = append(yaml,I(3)+"- name: "+name)
			yaml = append(yaml,I(4)+"typeRef: singleTextFile")
			yaml = append(yaml,I(4)+"ready: Auto")
			yaml = append(yaml,I(3)+"type: FileDrop")

			if bb[name] == nil {
				bb[name] = make([]string,0)
			}

			bb[name] = append(bb["Drop"+name],"Drop"+name)
		}
	}

	return yaml
}

//**************************************************************

func HandleIOPolicy(ctx context.Context, bb BreadBoard, operator string, in []byte) []string {

	var array []string = strings.FieldsFunc(string(in),InOutSplit)

	m := RefMarker(&ctx,"HandleIOPolicy")

	var yaml []string = make([]string, 0)
	var window, min, max int
	var name,typename string

	// If there's a file input, add a filedrop stanza

	for i := 0; i < len(array); i++ {

		// Default type is file
		typename = "singleTextFile"
		name, min, max, window = GetInputPolicy(ctx,array[i])

		if max == 0 {
			m.Note(&ctx,"Bad separator in min-max range")
			fmt.Println("Bad separator in min-max range: ",array[i],"use - for range")
			os.Exit(1)
		}
		
		if max < min {
			m.Note(&ctx,"Policy error - max < min")
			fmt.Println("Bad separator - max < min: ",array[i])
			os.Exit(1)
		}
		
		if bb[name] == nil {
			bb[name] = make([]string,0)
		}
		
		bb[name] = append(bb[name],operator)
		
		yaml = append(yaml,"- name:"+" "+name)
		yaml = append(yaml,I(1)+"typeRef:"+" "+typename)
		
		if window > 0 {
			yaml = append(yaml,I(1)+fmt.Sprintf("slide: %d",window))
		}
		
		if min > 1 && max == min {
			yaml = append(yaml,I(1)+fmt.Sprintf("requiredSequenceLength: %d",min))
		} else if min > 1 {
			yaml = append(yaml,I(1)+fmt.Sprintf("minSequenceLength: %d",min))
		}
		
		if strings.HasPrefix(name,"in") {
			yaml = append(yaml,I(1)+"ready: Auto")
		} else {
			yaml = append(yaml,I(1)+"ready: Succeeded")
		}
	}

return yaml
}

//**************************************************************

func InOutSplit (r rune) bool {
	switch r {
	case ',': return true
	default: return unicode.IsSpace(r)
	}
}

//**************************************************************

func GetInputPolicy(ctx context.Context,in string) (string,int,int,int) {

	m := RefMarker(&ctx,"GetInputPolicy")

	var min,max int
	var ch rune
	var slide int = 0

	s := strings.Split(in,"[")

	name := s[0]

	if len(s) == 1 {

		min = 1
		max = 1

	} else {

		min = 1
		max = 0
		
		if strings.Contains(s[1],"-") && strings.Contains(s[1],"/") {
			m.Note(&ctx,"Can't combine min-max range with sliding window").
				Attributes(NR(in,"policy range"))
			os.Exit(1)
		}

		if strings.Contains(s[1],"-") {
			fmt.Sscanf(s[1],"%d%c%d]", &min,&ch,&max)
			
			// Syntax error - used a comma in the range specifier
			if ch == 0 {
				m.Note(&ctx,"Bad separator in min-max range").
					Attributes(NR(in,"policy range"))
				os.Exit(1)
			}

		} else if strings.Contains(s[1],"/") {

			fmt.Sscanf(s[1],"%d%c%d]", &min,&ch,&slide)
			max = min
		} else {
			fmt.Sscanf(s[1],"%d", &min)
			max = min
		}
	}

	return name, min, max, slide
}

//**************************************************************

func AddWiring(ctx context.Context, bb BreadBoard) []string {

	var yaml []string = make([]string, 0)

	m := RefMarker(&ctx,"AddWiring").
		Intent("Trace wires to ensure they are all connected")

	for k,v := range bb {
		if strings.HasPrefix(k,"out") {
		} else {

			if len(v) < 2 {
				fmt.Println("XXXX!!!! bad wire ("+k+") emanating from ",v)
				m.FailedIntent("XXXX!!!! bad wire ("+k+") emanating from "+v[0])
				os.Exit(1)	
			}

			var name,source,dest string
			
			for j := 1; j < len(v); j++ {
				//fmt.Println("WIRES: ",k,v)
				name = v[0]+"2"+v[j]
				source = fmt.Sprintf("%s/%s",v[0],k)
				dest = fmt.Sprintf("%s/%s",v[j],k)
				yaml = append(yaml,I(3)+"- name:"+" "+name)
				yaml = append(yaml,I(4)+"sourceRef:"+" "+source)
				yaml = append(yaml,I(4)+"destinationRef:"+" "+dest)
			}
		}
	}

	return yaml
}

//**************************************************************

func I(level int) string {
	var indent string = strings.Repeat("  ",level)
	var s string
	s = fmt.Sprintf("%.3d:%s",level,indent)
	s = indent
	return s
}