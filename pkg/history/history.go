
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
	"sync/atomic"
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

// This is used to coordinate a forensic monotonic timeline,
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

type NameandRole struct {
	name string
	role string
	hub string
}

type Name struct {
	name string
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

func (m RM) UsingN(name string) RM {
	return UsingSlave(N(name))
}

func (m RM) UsingNR(name string, role string) RM {
	return UsingSlave(NR(name,role))
}

func (m RM) UsingSlave(nr NameAndRole) RM {

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

func (m RM) FailedBecause(name string) RM {
	return FailedBy(N(name))
}

func (m RM) FailedBecause(name string, role string) RM {
	return FailedBy(NR(name,role))
}

func (m RM) FailedBy(nr NameAndRole) RM {

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

func (m RM) FailedBecause(s string) RM {
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
