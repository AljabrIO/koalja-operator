
// 1. cellibrium in golang
// 2. path history
//

// ***************************************************************************
//*
//* Cellibrium v2 in golang ... 
//*
// ***************************************************************************

package history

import (
//	"strings"
	"sync/atomic"
	"context"
	"time"
	"runtime"
//	"crypto/sha1"
	"fmt"
	"os"
)

// ***************************************************************************
// Interior timeline
// ***************************************************************************

// This is used to coordinate a forensic monotonic timeline,
// in spite of recursive context, because Context can't track time
// This will end up being irrelevant to graphDB, has only local significance,
// except for relative order

type Name string
type List []string
type Neighbours []int
type SparseGraph map[int]Neighbours
type BreadBoard map[string]List

var INTERIOR_TIME int64 = 0 
var PROPER_PATHS SparseGraph
var PROCESS_CTX string

// ****************************************************************************

type NameAndRole struct {
	name string
	role string
	hub string
}

// ****************************************************************************

type Association struct {
 	key     int      // index
	STtype  int      // oriented type, - reverses oriention
	fwd     string   // forward oriented meaning
	bwd     string   // backward " 
}

// ****************************************************************************

type PTime struct {

	proper   int        // monotonic thread clock
	exterior int        // monotonic exterior clock
	previous int        // exterior ancestor of current time
	utc      int64  // Unix time
}

// ****************************************************************************

type ProcessContext struct {  // Embed this in ctx

	// Process invariants

	// Streams for dropping outcomes(t)
	tf *os.File
	gf *os.File

	tick PTime

	prefix     string    // unique process channel name declared in LocationInfo()
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
	uses int = 13
	alias int = 26
	determines int = 9

	PROCESS_MARKER string = "process reference marker"
	SYS_ERR_MSG string = "system error message"
	UNSPEC_ROLE string = "unspecified role"
)

var (
	ASSOCIATIONS = [99]Association{
		{0,0, "unknown promise", "unknown promise"},

		{1,GR_CONTAINS,"contains","belongs to or is part of"},
		{-1,GR_CONTAINS,"does not contain","is not part of"},

		// blue satisfies colour, colour is satisfied by blue
		{3,-GR_CONTAINS,"satisfies","is satisfied by"},
		{-3,-GR_CONTAINS,"does not satisfy","is not satisfied by"},

		// colour generalizes blue
		{4,GR_CONTAINS,"generalizes","is a special case of"},
		{-4,GR_CONTAINS,"is not a generalization of","is not a special case of"},

		{5,GR_FOLLOWS,"followed after","is preceded by"},
		{-5,GR_FOLLOWS,"does not follow","is not preceded by"},

		{6,GR_FOLLOWS,"originates from","is the source/origin of"},
		{-6,GR_FOLLOWS,"does not originate from","is not the source/origin of"},

		{7,GR_FOLLOWS,"provided by","provides"},
		{-7,GR_FOLLOWS,"is not provided by","does not provide"},

		{8,GR_FOLLOWS,"maintained by","maintains"},
		{-8,GR_FOLLOWS,"is not maintained by","doesn't maintain"},

		{9,GR_FOLLOWS,"may depend on","may determine"},
		{-9,GR_FOLLOWS,"doesn't depend on","doesn't determine"},

		{10,GR_FOLLOWS,"was created by","created"},
		{-10,GR_FOLLOWS,"was not created by","did not creat"},

		{11,GR_FOLLOWS,"reached to","reponded to"},
		{-11,GR_FOLLOWS,"did not reach to","did not repond to"},

		{12,GR_FOLLOWS,"caused by","may cause"},
		{-12,GR_FOLLOWS,"was not caused by","probably didn't cause"},

		{13,GR_FOLLOWS,"seeks to use","is used by"},
		{-13,GR_FOLLOWS,"does not seek to use","is not used by"},

		{14,GR_EXPRESSES,"is called","is a name for"},
		{-14,GR_EXPRESSES,"is not called","is not a name for"},

		{15,GR_EXPRESSES,"expresses an attribute","is an attribute of"},
		{-15,GR_EXPRESSES,"has no attribute","is not an attribute of"},

		{16,GR_EXPRESSES,"promises/intends","is intended/promised by"},
		{-16,GR_EXPRESSES,"rejects/promises to not","is rejected by"},

		{17,GR_EXPRESSES,"has an instance or particular case","is a particular case of"},
		{-17,GR_EXPRESSES,"has no instance/case of","is not a particular case of"},

		{18,GR_EXPRESSES,"has value or state","is the state or value of"},
		{-18,GR_EXPRESSES,"hasn't any value or state","is not the state or value of"},

		{19,GR_EXPRESSES,"has argument or parameter","is a parameter or argument of"},
		{-19,GR_EXPRESSES,"has no argument or parameter","isn't a parameter or argument of"},

		{20,GR_EXPRESSES,"has the role of","is a role fulfilled by"},
		{-20,GR_EXPRESSES,"has no role","is not a role fulfilled by"},

		{21,GR_EXPRESSES,"has outcome","is an outcome of"},
		{-21,GR_EXPRESSES,"has no outcome","is not an outcome of"},

		{22,GR_EXPRESSES,"has function","is the function of"},
		{-22,GR_EXPRESSES,"doesn't have function","is not the function of"},

		{24,GR_EXPRESSES,"infers","is inferred from"},
		{-24,GR_EXPRESSES,"does not infer","cannot be inferred from"},

		{25,GR_NEAR,"concurrent with","not concurrent with"},
		{-25,GR_NEAR,"not concurrent with","not concurrent with"},

		{26,GR_NEAR,"also known as","also known as"},
		{-26,GR_NEAR,"not known as","not known as"},

		{27,GR_NEAR,"is approximately","is approximately"},
		{-27,GR_NEAR,"is far from","is far from"},

		{28,GR_NEAR,"may be related to","may be related to"},
		{-28,GR_NEAR,"likely unrelated to","likely unrelated to"},

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
// Transactions
// ****************************************************************************

func SignPost(ctx *context.Context, remark string) ProcessContext {

	// Retrieve the cookie
	cctx := *ctx

	pc, _ := cctx.Value(PROCESS_CTX).(ProcessContext)

	// Update the cookie
	// time ...

	pc.tick = BigTick(pc.tick)

	// location ...

	// Update this local copy of context, each time we erect a signpost
	// to hand down to the next layer

	*ctx = context.WithValue(cctx, PROCESS_CTX, pc)

	WriteChainBlock(pc, remark)

	// Pass on local data relative to current context
	return pc
}

// ****************************************************************************

func (pc ProcessContext) Note(s string) ProcessContext {

	pc.tick = SmallTick(pc.tick)
	WriteChainBlock(pc,s)
	return pc
}

// ****************************************************************************

func (pc ProcessContext) Attributes(attr ...NameAndRole) ProcessContext {

	for i := 0; i < len(attr); i++ {
		//Relation(m.ctx,true,m.description.hub,expresses,attr[i].hub)

		s := "(" + attr[i].name + "," + attr[i].role + ")"
		pc.tick = SmallTick(pc.tick)
		WriteChainBlock(pc,s)
	}
	return pc
}

// ****************************************************************************

func (pc ProcessContext) ReliesOn(nr NameAndRole) ProcessContext {

// SRC uses DEST
// uses
	//var logmsg string = "-used " + nr.role + ": " + nr.name

	pc.tick = SmallTick(pc.tick)
	WriteChainBlock(pc,nr.hub)
	return pc
}

// ****************************************************************************

func (pc ProcessContext) Determines(nr NameAndRole) ProcessContext {

	//var logmsg string = "-used by " + nr.role + ": " + nr.name

// determines
// DEST uses SRC
	//Relation(m.ctx,true,m.description.hub,uses,nr.hub)

	pc.tick = SmallTick(pc.tick)
	WriteChainBlock(pc,nr.hub)
	return pc
}

// ****************************************************************************

func (pc ProcessContext) PartOf(nr NameAndRole) ProcessContext {

	//var logmsg string = "- part of " + nr.role + ": " + nr.name

	pc.tick = SmallTick(pc.tick)
	WriteChainBlock(pc,nr.hub)
	return pc
}

// This could be folded into something else

func (pc ProcessContext) Contains(nr NameAndRole) ProcessContext {

	pc.tick = SmallTick(pc.tick)
	WriteChainBlock(pc,nr.hub)
	return pc
}

// ****************************************************************************

func (pc ProcessContext) FailedSlave(nr NameAndRole) ProcessContext {

	pc.tick = SmallTick(pc.tick)
	WriteChainBlock(pc,nr.hub)
	return pc
}

func (pc ProcessContext) FailedBecause(name string) ProcessContext {
	return pc.FailedSlave(N(name))
}

// ****************************************************************************

func (pc ProcessContext) Intent(s string) ProcessContext {

	pc.tick = SmallTick(pc.tick)
	WriteChainBlock(pc,s)
	return pc
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

func N(name string) NameAndRole {
	return NR(name,UNSPEC_ROLE)
}

// ****************************************************************************

func BigTick(t PTime) PTime {

	// Since proper time gets reset by new exec, we should keep x,t clear
	// And x should depend on the execution is possible

	var next int64 = atomic.AddInt64(&INTERIOR_TIME,1)
	// record the ancestry
	t.previous = t.exterior
	// and add to unique timeline
	t.exterior = int(next)
	t.proper = 1
	t.utc = time.Now().Unix()

	if PROPER_PATHS[t.previous] == nil {
		PROPER_PATHS[t.previous] = make(Neighbours,0)
	}

	PROPER_PATHS[t.previous] = append(PROPER_PATHS[t.previous], t.exterior)

	// Check for discontinuous time

	return t
}

// ****************************************************************************

func SmallTick(t PTime) PTime {

	t.proper += 1
	t.utc = time.Now().Unix()

	// Check for discontinuous time

	return t
}

// ****************************************************************************
// * Context
// ****************************************************************************

func SetLocationInfo(ctx context.Context, m map[string]string) context.Context {

	var pc ProcessContext

	// If the file doesn't exist, create it, or append to the file
	var err error
	var pid int

	// Make a unique filename for the application instance, using pid and executable

	path := m["Process"] + m["Version"]

	pid = os.Getpid()

	// Put the dir in /tmp for now, assuming effectively private in cloud

	err = os.MkdirAll("/tmp/cellibrium/"+path, 0755)

	tpath := fmt.Sprintf("/tmp/cellibrium/%s/transaction_%d",path,pid)

	// Transaction log
	pc.tf, err = os.OpenFile(tpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("ERROR ",err)
	}

	// Graph DB
	gpath := fmt.Sprintf("/tmp/cellibrium/%s/graph_%d",path,pid)
	pc.gf, err = os.OpenFile(gpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("ERROR ",err)
	}

	// Initialize process time
	pc.tick.proper = 0
	pc.tick.previous = 0
	pc.tick.exterior = 0
	pc.tick.utc = time.Now().Unix()

	PROPER_PATHS = make(SparseGraph)

	ext_ctx := context.WithValue(ctx, PROCESS_CTX, pc)

	// Explain context in graphical terms

//	Relation(ext_ctx,true,location,alias,pc.prefix)

	//for k, v := range m {
		//kv := k + ":" + v
		//Relation(ext_ctx,true,location,expresses,kv)
		//Relation(ext_ctx,true,kv,hasrole,k)
		//Relation(ext_ctx,true,kv,expresses,v)
	//}

	return ext_ctx
}

// ****************************************************************************

func WriteChainBlock(pc ProcessContext, remark string) {

	pid := os.Getpid()
	entry := fmt.Sprintf("%d , %d , %d , %d , %d ;%s\n",pid,time.Now().Unix(),pc.tick.proper,pc.tick.exterior,pc.tick.previous,remark)
	pc.tf.WriteString(entry)
	//fmt.Print(entry)
}

// ****************************************************************************

func (pc ProcessContext) AddError(err error) ProcessContext {

	n := NR(err.Error(),SYS_ERR_MSG)
	pc.FailedSlave(n)

/*	AnnotateNR(m.ctx,n)
	Relation(m.ctx,true,n.hub,hasrole,n.role)
	Relation(m.ctx,true,m.description.hub,expresses,n.hub)
	Relation(m.ctx,true,n.hub,follows,m.description.hub)
	*/
	return pc
}

// ****************************************************************************

func CloseProcess(ctx context.Context) {
//	pc, _ := GetProcessContext(ctx)
//	pc.tf.Close();
//	pc.gf.Close();
}
