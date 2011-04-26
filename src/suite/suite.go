package suite

import (
	"fmt"
	// "bufio"
	"os"
	"http"
	"log"
	"strings"
	"strconv"
	"./../tag/tag"
	"io"
	"bytes"
)

var logLevel int = 5   // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace

func error(f string, m... interface{}) { if logLevel >= 1 { log.Print("*ERROR* " + fmt.Sprintf(f, m...)) } }
func warn(f string, m... interface{})  { if logLevel >= 2 { log.Print("*WARN * " + fmt.Sprintf(f, m...)) } }
func info(f string, m... interface{})  { if logLevel >= 3 { log.Print("*INFO * " + fmt.Sprintf(f, m...)) } }
func debug(f string, m... interface{}) { if logLevel >= 4 { log.Print("*DEBUG* " + fmt.Sprintf(f, m...)) } }
func trace(f string, m... interface{}) { if logLevel >= 5 { log.Print("*TRACE* " + fmt.Sprintf(f, m...)) } }

// Represent a condition like "!Content-Type ~= "text/html" where Key="Content-Type"
// Op="~=", Val="text/html" and Neg=true.  For tags Op contains the number of
// occurences of the tag. Key is "Text", "Bin" or "Tag" for body-testing.
// Line contains the line number in the source
type Condition struct {
	Key string
	Op	string
	Val string
	Neg bool
	Line int
}

func atoi(a string, n int) int {
	i, err := strconv.Atoi(a)
	if err != nil {
		error("Cannot convert '%s' to integer (line %d).", a, n)
		i = -99999	}
	return i
}

func (cond *Condition) Fullfilled (v string) bool {
	ans := false
	switch cond.Op {
	case "==": ans = (cond.Val == v)
	case "_=": ans = strings.HasPrefix(v, cond.Val)
	case "=_": ans = strings.HasSuffix(v, cond.Val)
	case "~=": ans = strings.Contains(v, cond.Val)
	case ">": ans = (atoi(v, cond.Line) > atoi(cond.Val, cond.Line))
	case ">=": ans = (atoi(v, cond.Line) >= atoi(cond.Val, cond.Line))
	case "<": ans = (atoi(v, cond.Line) < atoi(cond.Val, cond.Line))
	case "<=": ans = (atoi(v, cond.Line) >= atoi(cond.Val, cond.Line))
	default:
		warn("Condition operator '%s' (line %d) not implemented.", cond.Op, cond.Line)
	}
	if cond.Neg {
		ans = !ans
	}
	return ans
}

func (c *Condition) String() (s string) {
	if c.Neg { s = "!" }
	s += c.Key + " " + c.Op + " " + c.Val
	return
}

func (c *Condition) Copy() (n *Condition) {
	n = new(Condition)
	n.Key, n.Op, n.Val, n.Neg, n.Line = c.Key, c.Op, c.Val, c.Neg, c.Line
	return	
}

type Test struct {
	Title  string
	Method string
	Url    string
	Header map[string] string
	RespCond []*Condition
	BodyCond []*Condition
	Pre  []string
	MaxTime int // -1: unset, 0=no limit, >0: limit in ms
	Sleep int   // -1: unset, >=0: sleep after in ms
	Repeat int  // -1: unset, 0=disabled, >0: count
	Param map[string] string
	Const  map[string] []string
	Random map[string] []string
	Seq map[string] []string
	SeqCnt map[string] []int
	run bool
	passed bool
}

func condCopy(src []*Condition) (dest []*Condition) {
	trace("Copying %d conditions", len(src))
	dest = make([]*Condition, len(src))
	for i, c := range src { 
		dest[i] = c.Copy() 
	}  
	return
}

func mapCopy(src map[string] string) (dest map[string] string) {
	dest = make(map[string] string, len(src))
	for k, v := range src { 
		dest[k] = v 
	}
	return
}

func (src *Test) Copy() (dest *Test) {
	trace("Copying test '%s'", src.Title)
	dest = new(Test)
	dest.Title = src.Title
	dest.Method = src.Method
	dest.Url = src.Url
	dest.Pre = src.Pre
	dest.MaxTime = src.MaxTime
	dest.Repeat = src.Repeat
	dest.Param = mapCopy(src.Param)
	dest.RespCond = condCopy(src.RespCond)
	dest.BodyCond = condCopy(src.BodyCond)
	return
}

type TestError struct {
	os.ErrorString
}

var (
	ErrTimeout	= &TestError{"Connection timed out."}
	ErrSystem   = &TestError{"Underlying system failed."}
	ErrTest     = &TestError{"Failed Test."}
)


// TODO: GLobal is just Test[0]. Test shall have own Random and Sequenz and Const
type Suite struct {
	Global Test
	Const  map[string] string
	Random map[string] string
	Sequenz map[string] string
	Test []*Test
	Result  map[string] int  // 0: not run jet, 1: pass, 2: fail, 3: err
}

func readBody(r io.ReadCloser) string {
	var bb bytes.Buffer
	if r != nil {
		io.Copy(&bb, r)
		r.Close()
	}
	return bb.String()
}

func shouldRedirect(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		return true
	}
	return false
}

func postWrapper(c *http.Client, t *Test) (r *http.Response, finalURL string, err os.Error) {
	return

}

func addHeaders(req *http.Request, t *Test) {
	for k, v:= range t.Header {
		req.Header.Set(k, v)
	}
}



func testHeader(resp *http.Response, t *Test) (err os.Error) {
	info("Testing Header")
	for i, c := range t.RespCond {
		if c == nil {
			trace("Skipping nil response condition %d.", i)
			continue
		}
		trace("Response condition %d: %s", i, c.String())
		v := resp.Header.Get(c.Key)
		if !c.Fullfilled(v) {
			error("Failed header condition '%s' (line %d): Got '%s'", c.String(), c.Line, v)
			err = ErrTest
		} else {
			debug("Okay")
		}
	}
	return
}

func testBody(body string, t *Test, doc *tag.Node) (err os.Error) {
	info("Testing Body")
	for i, c := range t.BodyCond {
		trace("Body Condition %d: '%s'", i, c.String())
		switch c.Key {
		case "Text":
			trace("Text Matching '%s'", c.String())
			if !c.Fullfilled(body) {
				error("Failed body text condition '%s' (line %d).", c.String(), c.Line)
				err = ErrTest
			} else {
				debug("Okay")
			}
		case "Bin":
			error("Unimplemented")
			err = ErrSystem
		case "Tag":
			if doc == nil {
				error("Faild body tag condition (line %d): Document unparsable.", c.Line)
				err = ErrSystem
				continue
			}
			ts := tag.ParseTagSpec(c.Val)
			if c.Op == "" { // no counting
				n := tag.FindTag(ts, doc)
				if n == nil && !c.Neg {
					error("Failed body tag condition '%s' (line %d): Not found.", ts.String(), c.Line)
					err = ErrTest
				} else if n != nil && c.Neg {
					error("Failed body tag condition '%s' (line %d): Found forbidden.", ts.String(), c.Line)
					err = ErrTest
				} else {
					debug("Okay")
				}
			} else {
				warn("Tag counting not implemented jet (line %d).", c.Line)
			}
		}
	}
	return
}



func Get(t *Test) (r *http.Response, finalURL string, err os.Error) {
	var url = t.Url  // <-- Patched
	// TODO: if/when we add cookie support, the redirected request shouldn't
	// necessarily supply the same cookies as the original.
	// TODO: set referrer header on redirects.
	var base *http.URL
	// TODO: remove this hard-coded 10 and use the Client's policy
	// (ClientConfig) instead.
	for redirect := 0; ; redirect++ {
		if redirect >= 10 {
			err = os.ErrorString("stopped after 10 redirects")
			break
		}

		var req http.Request
		req.Method = "GET"
		req.ProtoMajor = 1
		req.ProtoMinor = 1
		if base == nil {
			req.URL, err = http.ParseURL(url)
		} else {
			req.URL, err = base.ParseURL(url)
		}
		if err != nil {
			break
		}
		// vvvv Patched vvvv
		addHeaders(&req, t) 
		// ^^^^ Patched ^^^^
		url = req.URL.String()
		if r, err = http.DefaultClient.Do(&req); err != nil {
			break
		}
		if shouldRedirect(r.StatusCode) {
			r.Body.Close()
			if url = r.Header.Get("Location"); url == "" {
				err = os.ErrorString(fmt.Sprintf("%d response missing Location header", r.StatusCode))
				break
			}
			base = req.URL
			continue
		}
		finalURL = url
		return
	}

	err = &http.URLError{"Get", url, err}
	return
}

func fillRespCond(src, dest []*Condition) {
	a := len(dest)
	for _, cond := range src {
		f := false
		for i:=0; i<a; i++ {
			if dest[i].Key == cond.Key { 
				dest[i] = cond.Copy() true
				f = true
				trace("Overwriting global response condition '%s' with '%s'", dest[i].String(), cond.String())
				break
			}
		}
		if !f {
			dest = append(dest, cond.Copy())
			trace("Adding response condition '%s'", cond.String())
		}
	}
}

func fillBodyCond(src, dest []*Condition) { 
	// TODO: how to delete a test inherited from global?
	for _, c := range src {
		trace("Adding body condition '%s'", c.String())
		dest = append(dest, c.Copy())
	}
}

// Prepare the test: Use global as template and overwrite everiting set
// TODO: Method and Url might be set in global?
func prepareTest(s *Suite, n int) (test *Test) {
	trace("Preparing test no %d.", n)
	src := s.Test[n]
	test = s.Test[0].Copy()  // Use Global-Test (index 0) as template
	test.Title, test.Method, test.Url = src.Title, src.Method, src.Url
	if src.MaxTime >=0 { test.MaxTime = src.MaxTime }
	if src.Sleep >=0 { test.Sleep = src.Sleep }
	if src.Repeat >= 0 { test.Repeat = src.Repeat }
	fillRespCond(src.RespCond, test.RespCond)
	fillBodyCond(src.BodyCond, test.BodyCond)
	substituteVariables(test, 
	return
}

func parsableBody(resp *http.Response) bool {
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return true
	} else if strings.Contains(resp.Header.Get("Content-Type"), "text/xml") {
		return true
	}
	info("Response body is not considered parsable")
	return false
}

func (s *Suite) RunTest(n int) (err os.Error) {
	
	t := prepareTest(s, n)
	info("Running test %d: '%s'", n, t.Title)
	
	
	
	if t.Method != "GET" {
		error("Post not jet implemented")
		return ErrSystem
	} 
	
	response, url, err := Get(t)
	
	if err != nil {
		error(err.String())
		return ErrSystem
	}
	
	// Add special fields to header
	response.Header.Set("StatusCode", fmt.Sprintf("%d", response.StatusCode))
	response.Header.Set("Url", url)
	herr := testHeader(response, t)
	
	body := readBody(response.Body)
	var doc *tag.Node
	if parsableBody(response) {
		doc = tag.ParseHtml(body)
	} 
	
	berr := testBody(body, t, doc)

	if herr!=nil || berr!=nil {
		err = ErrTest
	}
	
	return
}