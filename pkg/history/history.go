
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
	"io/ioutil"
//	"crypto/sha1"
	"fmt"
	"os"

	// Try this for local string -> int
	"hash/fnv"
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

var BASEDIR string = "/tmp/cellibrium"

// ****************************************************************************

type Concept struct {
	Name string
	Hash string
	Key uint64
}

// ****************************************************************************

type NameAndRole struct {
	name string
	role string
	hub string
}

// ****************************************************************************

type Association struct {
 	Key     int      // index
	STtype  int      // oriented type, - reverses oriention
	Fwd     string   // forward oriented meaning
	Bwd     string   // backward " 
}

// ****************************************************************************

type PTime struct {

	proper   int        // monotonic thread clock
	exterior int        // monotonic exterior clock
	previous int        // exterior ancestor of current time
	utc      int64  // Unix time
}

// ****************************************************************************

type ProcessContext struct {  // Embed this in ctx as stigmergic memory

	// Process invariants

	previous_concept Concept
	previous_event Concept

	// Streams for dropping outcomes(t)
	tf *os.File
	gf *os.File

	// Process paths
	tick PTime

	prefix     string    // unique process channel name declared in LocationInfo()
}

// ****************************************************************************

type NeighbourConcepts map[int][]string // a list of concept hashes reachable by int type of relation

type Links struct {
	Fwd [5]NeighbourConcepts  // The association links, classified by direction and ST type
	Bwd [5]NeighbourConcepts
}

// ****************************************************************************

/* From each concept, there may be links of different ST types, and these may
   have subtly different interpretations, we collect a list here */

func LinkInit() Links {
	var links Links
 	links.Fwd[GR_NEAR] = make(NeighbourConcepts,0)
	links.Fwd[GR_FOLLOWS] = make(NeighbourConcepts,0)
	links.Fwd[GR_CONTAINS] = make(NeighbourConcepts,0)
	links.Fwd[GR_EXPRESSES] = make(NeighbourConcepts,0)  
	links.Bwd[GR_NEAR] = make(NeighbourConcepts,0)
	links.Bwd[GR_FOLLOWS] = make(NeighbourConcepts,0)
	links.Bwd[GR_CONTAINS] = make(NeighbourConcepts,0)
	links.Bwd[GR_EXPRESSES] = make(NeighbourConcepts,0)  
	return links
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
	has_role int = 19
	originates_from int = 5
	expresses int = 14
	promises int = 15
	follows int = 4
	contains int = 1
	generalizes int = 3
	uses int = 12
	alias int = 24

	PROCESS_MARKER string = "process reference marker"
	SYS_ERR_MSG string = "system error message"
	UNSPEC_ROLE string = "unspecified role"
)

// ****************************************************************************

var (
	ASSOCIATIONS = [99]Association{
		{0,0, "", ""},

		{1,GR_CONTAINS,"contains","belongs to or is part of"},
		{-1,GR_CONTAINS,"does not contain","is not part of"},

		// blue satisfies colour, colour is satisfied by blue
		{2,-GR_CONTAINS,"satisfies","is satisfied by"},
		{-2,-GR_CONTAINS,"does not satisfy","is not satisfied by"},

		// colour generalizes blue
		{3,GR_CONTAINS,"generalizes","is a special case of"},
		{-3,GR_CONTAINS,"is not a generalization of","is not a special case of"},

		{4,GR_FOLLOWS,"follows after","precedes"},
		{-4,GR_FOLLOWS,"does not follow","does not precede"},

		{5,GR_FOLLOWS,"originates from","is the source/origin of"},
		{-5,GR_FOLLOWS,"does not originate from","is not the source/origin of"},

		{6,GR_FOLLOWS,"provided by","provides"},
		{-6,GR_FOLLOWS,"is not provided by","does not provide"},

		{7,GR_FOLLOWS,"maintained by","maintains"},
		{-7,GR_FOLLOWS,"is not maintained by","doesn't maintain"},

		{8,GR_FOLLOWS,"may depend on","may determine"},
		{-8,GR_FOLLOWS,"doesn't depend on","doesn't determine"},

		{9,GR_FOLLOWS,"was created by","created"},
		{-9,GR_FOLLOWS,"was not created by","did not creat"},

		{10,GR_FOLLOWS,"reached to","reponded to"},
		{-10,GR_FOLLOWS,"did not reach to","did not repond to"},

		{11,GR_FOLLOWS,"caused by","may cause"},
		{-11,GR_FOLLOWS,"was not caused by","probably didn't cause"},

		{12,GR_FOLLOWS,"seeks to use","is used by"},
		{-12,GR_FOLLOWS,"does not seek to use","is not used by"},

		{13,GR_EXPRESSES,"is called","is a name for"},
		{-13,GR_EXPRESSES,"is not called","is not a name for"},

		{14,GR_EXPRESSES,"expresses an attribute","is an attribute of"},
		{-14,GR_EXPRESSES,"has no attribute","is not an attribute of"},

		{15,GR_EXPRESSES,"promises/intends","is intended/promised by"},
		{-15,GR_EXPRESSES,"rejects/promises to not","is rejected by"},

		{16,GR_EXPRESSES,"has an instance or particular case","is a particular case of"},
		{-16,GR_EXPRESSES,"has no instance/case of","is not a particular case of"},

		{17,GR_EXPRESSES,"has value or state","is the state or value of"},
		{-17,GR_EXPRESSES,"hasn't any value or state","is not the state or value of"},

		{18,GR_EXPRESSES,"has argument or parameter","is a parameter or argument of"},
		{-18,GR_EXPRESSES,"has no argument or parameter","isn't a parameter or argument of"},

		{19,GR_EXPRESSES,"has the role of","is a role fulfilled by"},
		{-19,GR_EXPRESSES,"has no role","is not a role fulfilled by"},

		{20,GR_EXPRESSES,"occurred at","was marked by event"},
		{-20,GR_EXPRESSES,"did not occur at","was not marked by an event"},

		{21,GR_EXPRESSES,"has function","is the function of"},
		{-21,GR_EXPRESSES,"doesn't have function","is not the function of"},

		{22,GR_EXPRESSES,"infers","is inferred from"},
		{-22,GR_EXPRESSES,"does not infer","cannot be inferred from"},

		{23,GR_NEAR,"concurrent with","not concurrent with"},
		{-23,GR_NEAR,"not concurrent with","not concurrent with"},

		{24,GR_NEAR,"also known as","also known as"},
		{-24,GR_NEAR,"not known as","not known as"},

		{25,GR_NEAR,"is approximately","is approximately"},
		{-25,GR_NEAR,"is far from","is far from"},

		{26,GR_NEAR,"may be related to","may be related to"},
		{-26,GR_NEAR,"likely unrelated to","likely unrelated to"},

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

func HereAndNow() (string, string) {

	// Time

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
	dayname := fmt.Sprintf("Day%d",day)

        interval_start := (then.Minute() / 5) * 5
        interval_end := (interval_start + 5) % 60
        minD := fmt.Sprintf("Min%02d_%02d",interval_start,interval_end)

	var when string = fmt.Sprintf(" on %s %s %s %s %s at %s %s %s %s",shift,dow,dayname,month,year,hour,mins,quarter,minD)

	// Space

	var where = Where(3)

	// Build the invariant concept subgraphs

	cwhen := CreateConcept(when)
	cmins := CreateConcept(mins)
	chour := CreateConcept(hour)
	cyear := CreateConcept(year)
	cday := CreateConcept(dayname)
	cquart := CreateConcept(quarter)
	cminD := CreateConcept(minD)
	cshift := CreateConcept(shift)
	cmonth := CreateConcept(month)

	// variant times labels are only expressed by special case "when"

	ConceptLink(cwhen,expresses,cmins)
	ConceptLink(cwhen,expresses,chour)
	ConceptLink(cwhen,expresses,cday)
	ConceptLink(cwhen,expresses,cyear)
	ConceptLink(cwhen,expresses,cquart)
	ConceptLink(cwhen,expresses,cminD)
	ConceptLink(cwhen,expresses,cshift)
	ConceptLink(cwhen,expresses,cyear)
	ConceptLink(cwhen,expresses,cshift)
	ConceptLink(cwhen,expresses,cmonth)

	return where, when
}

// ****************************************************************************

func Where(depth int) string {

        // Interal usage
	p,fname,line, ok := runtime.Caller(depth)
	
	var location string

	if ok {
		var funcname = runtime.FuncForPC(p).Name()
		fn := "function "+funcname
		file := "file "+fname
		lnr := fmt.Sprintf("line %d",line)
		location = fmt.Sprintf(" in %s of %s at %s",fn,file,lnr)

		call := CreateConcept("code position")
		cwhere := CreateConcept(location)
		cfunc := CreateConcept(fn)
		cname := CreateConcept(file)
		cline := CreateConcept(lnr)

		ConceptLink(call,generalizes,cwhere)
		ConceptLink(cwhere,expresses,cfunc)
		ConceptLink(cwhere,expresses,cname)
		ConceptLink(cwhere,expresses,cline)

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

	// Pick up the stigmergic process memory
	pc, _ := cctx.Value(PROCESS_CTX).(ProcessContext)

	// Part1. Update the stigmergic cookie
	// Build the timelike change interaction picture

	pc.tick = BigTick(pc.tick)

	// Part 2. Build invariant picture
	// location ... and metric space of concepts

	cloc := CreateConcept("locations")
	csigns  := CreateConcept("signpost")
	cevent := CreateConcept("events")

	here, now := HereAndNow()

	// prefix contains the deployment
	hereandnow := here + " of " + pc.prefix + now

	chn  := CreateConcept(hereandnow)      // disambiguator
	cwhere := CreateConcept(here)
	cwhen := CreateConcept(now)

	// the remark sets semantics of this disambiguated region

	signpost := remark + hereandnow        // unique semantic event + spacetime

	cthissign := CreateConcept(signpost)       // specific combinatoric instance
	cremark  := CreateConcept(remark)      // possibly used elsewhere/when

	ConceptLink(csigns,contains,cthissign)

	ConceptLink(cthissign,expresses,cremark)
	ConceptLink(cthissign,expresses,chn)
	ConceptLink(cthissign,expresses,cwhere)

	// invariants CONTAIN when as a special case

	ConceptLink(cevent,generalizes,chn)
	ConceptLink(cloc,generalizes,cwhere)
	ConceptLink(cevent,generalizes,chn)

	ConceptLink(chn,expresses,cwhere)
	ConceptLink(chn,expresses,cwhen)

	// Graph causality - must be idempotent/invariant, so no specific coordinates

	ConceptLink(cremark,follows,pc.previous_concept)

	// This is variant, but orthogonal (testing, as this doesn't converge)
	ConceptLink(cthissign,follows,pc.previous_event)

	// Update this local copy of context, each time we erect a signpost
	// to hand down to the next layer

	pc.previous_concept = cremark
	pc.previous_event = cthissign

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

	if BASEDIR != "/tmp/cellibrium" {
		fmt.Println("Second call to SetLocationInfo is not allowed")
		os.Exit(1)
	}

	// Make a unique filename for the application instance, using pid and executable

	path := m["Deployment"] + m["Version"]

	pid = os.Getpid()

	// Put the dir in /tmp for now, assuming effectively private in cloud

	BASEDIR = BASEDIR+"/"+path
	err = os.MkdirAll(BASEDIR, 0755)

	tpath := fmt.Sprintf("%s/transaction_%d",BASEDIR,pid)

	// Transaction log
	pc.tf, err = os.OpenFile(tpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("ERROR ",err)
	}

	// Initialize process time
	pc.tick.proper = 0
	pc.tick.previous = 0
	pc.tick.exterior = 0
	pc.tick.utc = time.Now().Unix()
	pc.prefix = path

	pc.previous_concept = CreateConcept("program start")

	here, now := HereAndNow()

	// prefix contains the deployment
	hereandnow := here + " of " + pc.prefix + now

	chn  := CreateConcept(hereandnow)      // disambiguator
	CreateConcept(here)
	CreateConcept(now)

	pc.previous_event = chn

	PROPER_PATHS = make(SparseGraph)

	ext_ctx := context.WithValue(ctx, PROCESS_CTX, pc)

	kpod := CreateConcept("kubernetes pods")
	k8s := CreateConcept("kubernetes")
	pod := CreateConcept("pods")
	thispod := CreateConcept(m["Pod"])
	thisdeploy := CreateConcept(path)
	thisversion := CreateConcept(m["Version"])

	kdeploy := CreateConcept("kubernetes deployments")
	deploy := CreateConcept("deployments")

	// invariants are contained 
	ConceptLink(pod,generalizes,kpod)
	ConceptLink(deploy,generalizes,kdeploy)
	ConceptLink(kpod,generalizes,thispod)

	// Variants are expressed
	ConceptLink(kpod,expresses,k8s)
	ConceptLink(kdeploy,expresses,k8s)
	ConceptLink(kdeploy,generalizes,thisdeploy)
	ConceptLink(thisdeploy,expresses,thispod)
	ConceptLink(thisdeploy,expresses,thisversion)

	return ext_ctx
}

// ****************************************************************************

func WriteChainBlock(pc ProcessContext, remark string) {

	pid := os.Getpid()

	c := CreateConcept(remark)

	entry := fmt.Sprintf("%d , %d , %d , %d , %d ;%s\n",pid,time.Now().Unix(),
		pc.tick.proper,pc.tick.exterior,pc.tick.previous,c.Hash)

	pc.tf.WriteString(entry)
}

// ****************************************************************************

func fnvhash(b []byte) uint64 { // Currently trusting this to have no collisions
	hash := fnv.New64a()
	hash.Write(b)
	return hash.Sum64()
}

// ****************************************************************************

func FileExists(path string) bool {
    _, err := os.Stat(path)
    if err == nil { return true }
    if os.IsNotExist(err) { return false }
    return true
}

// ****************************************************************************

func (pc ProcessContext) AddError(err error) ProcessContext {

	n := NR(err.Error(),SYS_ERR_MSG)
	pc.FailedSlave(n)

/*	AnnotateNR(m.ctx,n)
	Relation(m.ctx,true,n.hub,has_role,n.role)
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

// ****************************************************************************
//  Graph invariants
// ****************************************************************************

func ConceptLink(c1 Concept, rel int, c2 Concept) {

	// Does these concepts already exist? If so, don't do these heavyweight ops again!

	var path string
	var err error
	var index int

	if rel < 0 {
		index = -2*rel
	} else {
		index = 2*rel-1
	}

	//fmt.Printf("LINKS of type (%s,%d)\n",ASSOCIATIONS[index].Fwd,rel)

	path = fmt.Sprintf("%s/concepts/%s/%d/%d/",BASEDIR,c1.Hash,ASSOCIATIONS[index].STtype,ASSOCIATIONS[index].Key)
	err = os.MkdirAll(path, 0755)

	if err != nil {
		fmt.Println("Couldn't make directory "+path)
		os.Exit(1)
	}

	// inode's name is concept hash for the link of type STtype
	os.Create(path + c2.Hash)

	path = fmt.Sprintf("%s/concepts/%s/%d/%d/",BASEDIR,c2.Hash,-ASSOCIATIONS[index].STtype,ASSOCIATIONS[index].Key)
	err = os.MkdirAll(path, 0755)

	if err != nil {
		fmt.Println("Couldn't make directory "+path)
		os.Exit(1)
	}

	// inode's name is concept hash for the INVERSE link of type -STtype
	os.Create(path + c1.Hash)
}

// ****************************************************************************

func CreateConcept(description string) Concept {

	// Using a hash to avoid overthinking this unique key problem for now

	var concept Concept
	var err error

	concept.Name = description
	concept.Key = fnvhash([]byte(description))
	concept.Hash = fmt.Sprintf("%d",concept.Key)

	path := fmt.Sprintf("%s/concepts/%s",BASEDIR,concept.Hash)

	err = os.MkdirAll(path, 0755)
	if err != nil {
		fmt.Println("Couldn't make directory "+path)
		os.Exit(1)
	}

	cpath := path + "/description"	

	if !FileExists(cpath) {
		content := []byte(concept.Name)
		err = ioutil.WriteFile(cpath, content, 0644)
	}

	return concept
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

// ****************************************************************************

func ConeTest() {

	country := CreateConcept("country")
	town := CreateConcept("town")
	city := CreateConcept("city")
	metro := CreateConcept("metropolis")
	district := CreateConcept("district")
	house := CreateConcept("house")
	apartment := CreateConcept("apartment")
	flat := CreateConcept("flat")
	home := CreateConcept("home")
	dwelling := CreateConcept("dwelling")
	oslo := CreateConcept("Oslo")

	ConceptLink(country,contains,city)
	ConceptLink(city,contains,district)
	ConceptLink(district,contains,home)

	ConceptLink(home,generalizes,house)
	ConceptLink(home,generalizes,apartment)
	ConceptLink(apartment,generalizes,flat)
	ConceptLink(dwelling,generalizes,home)
	ConceptLink(metro,generalizes,city)

	ConceptLink(city,generalizes,town)
	ConceptLink(town,generalizes,oslo)

	realnum := CreateConcept("real number")
	complexnum := CreateConcept("complex number")
	integer := CreateConcept("integer")
	one := CreateConcept("1")
	two := CreateConcept("2")
	three := CreateConcept("3")
	four := CreateConcept("4")

	ConceptLink(complexnum,generalizes,realnum)
	ConceptLink(realnum,generalizes,integer)
	ConceptLink(integer,generalizes,one)
	ConceptLink(integer,generalizes,two)
	ConceptLink(integer,generalizes,three)
	ConceptLink(integer,generalizes,four)
	ConceptLink(two,follows,one)
	ConceptLink(three,follows,two)
	ConceptLink(four,follows,three)

}