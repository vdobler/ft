package suite

import (
	"testing"
	"time"
	"fmt"
	"http"
	"os"
	"strconv"
	"strings"
	"html"
)

var (
	port       = ":54123"
	host       = "http://localhost"
	theSuite   *Suite
	skipStress bool
)

// keep server a life for n seconds after last testcase to allow manual testt the test server...
var testserverStayAlive int64 = 0


var suiteTmpl = `
----------------------
Global
----------------------
GET http://wont.use
RESPONSE
	Status-Code	== 200
CONST
	URL	 %s%s

# Test no 1
----------------------
Basic Test
----------------------
GET ${URL}/html.html
RESPONSE
	 Fancy-Header  == Important Value
	!Fancy-Header  == Wrong
	 Fancy-Header  ~= Important Value
	 Fancy-Header  ~= Value
	 Fancy-Header  _= Important
	 Fancy-Header  _= Important Value
	!Fancy-Header  _= Value
	!Fancy-Header  =_ Important
	 Fancy-Header  =_ Important Value
	 Fancy-Header  =_ Value
	 Fancy-Header  /= Important.*
	 Fancy-Header  /= ^Imp.*lue$
	!Fancy-Header  /= Wrong
	!Fancy-Header  /= ^Wro.*ng$
BODY
	 Txt  ~= Braunschweig
	 # 5765696c6572 = hex(Weiler)
	 Bin  ~= 5765696c6572
TAG
	 title == Dummy HTML 1
	 p class=a
	!p class=c
	!p == Wrong.*
	
# Test no 2
----------------------
Binary Test 1
----------------------
GET ${URL}/bin.bin
RESPONSE
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!
	Bin  ==  010748616c6c6f2057656c74210aFFFE
	Bin  _=  01074861
	Bin  =_  FFFE
	Bin  ~=  48616c

# Test no 3
----------------------
Binary Test 2
----------------------
GET ${URL}/bin.bin
PARAM
	sc  401
RESPONSE
	Status-Code    ==  401
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!

# Test no 4
----------------------
Sequence Test
----------------------
GET ${URL}/html.html
SEQ
	name   Anna Berta Claudia "Doris Dagmar" Emmely
PARAM
	text ${name}
BODY
	Txt  ~=  ${name}
SETTING
	Repeat	3

# Test no 5
----------------------
Random Test
----------------------
GET ${URL}/html.html
RAND
	name   Anna Berta Claudia "Doris Dagmar" Emmely
PARAM
	text ${name}
BODY
	Txt  ~=  ${name}
SETTING
	Repeat	3
	
# Test no 6
-------------------------
Plain Post (no Redirect)
-------------------------
POST ${URL}/post
RESPONSE
	Final-Url	==	${URL}/post
BODY
	Txt  ~=  Post Page 

# Test no 7
-------------------------
Post (with Redirect)
-------------------------
POST ${URL}/post
PARAM
	q		html.html
RESPONSE
	Final-Url	==	${URL}/html.html
TAG
	h1 == Dummy Document *
	p class=a == *Braunschweig Weiler
SETTING
	Dump 1

	
# Test no 8
------------------------
Too slow
------------------------
GET ${URL}/html.html
PARAM
	sleep	110
SETTING
	Max-Time	100
	
	
# Test no 9
----------------------
Variable Subst
----------------------
GET ${URL}/html.html
PARAM
	text ${SomeGlobal}
BODY
	Txt  ~=  GlobalValue
	
	
# Test no 10
----------------------
Multiple Parameters
----------------------
GET ${URL}/html.html
PARAM
	text 	Hund Katze "Anna Berta"
	
	
# Test no 11
-------------------------
Filepost
-------------------------
POST ${URL}/post
PARAM
	datei	@file:condition.go
RESPONSE
	Final-Url	==	${URL}/post
BODY
	Txt  ~=  Post Page 

# Test no 12
----------------------
Encoding 1
----------------------
GET ${URL}/html.html
PARAM
	text	"€ & <ÜÖÄ> = üöa" 
BODY
	Txt  ~=  UTF-8 Umlaute äöüÄÖÜ Euro €
	Txt  ~=  € &amp; &lt;ÜÖÄ&gt; = üöa
	
# Test no 13
-------------------------
Encoding 2
-------------------------
POST ${URL}/post
PARAM
	datei	@file:condition.go
	text	"€ & <ÜÖÄ> = üöa"
RESPONSE
	Final-Url	==	${URL}/post
BODY
	Txt  ~=  Post Page 
	Txt  ~=  € &amp; &lt;ÜÖÄ&gt; = üöa

# Test no 14
-------------------------
Multipart Post
-------------------------
POST:mp ${URL}/post
RESPONSE
	Final-Url	==	${URL}/post
PARAM
	text ABCDwxyz1234
BODY
	Txt  ~=  Post Page
	Txt  ~= ABCDwxyz1234
SETTING
	Dump 1
`

var cookieSuite = fmt.Sprintf(`
----------------------
Global
----------------------
GET http://wont.use
CONST
	URL	 %s%s

----------------------
Display Cookies
----------------------
GET ${URL}/cookie.html
SEND-COOKIE
	MyFirst     MyFirstCookieValue
	Sessionid   abc123XYZ
	JSESSIONID  5AE613FC082DEB79484C774677651164
	
TAG
	li == Sessionid :: abc123XYZ
	li == MyFirst :: MyFirstCookieValue
	li == JSESSIONID :: 5AE613FC082DEB79484C774677651164
	
----------------------
Login
----------------------
POST ${URL}/post
PARAM
	# cookie parameter is added to response header by /post handler
	cookie  TheSession=randomsessionid
RESPONSE
	Final-Url	==	${URL}/post
SET-COOKIE
	TheSession         ~=  randomsessionid
	TheSession:Value   ==  randomsessionid
	TheSession:Path    _=  /de/
	TheSession:Secure  ==  true
	TheSession:Domain  =_  .org
	TheSession:Expires  <  Sat, 21 May 2011 12:00:00 GMT
	TheSession:Expires  >  Thu, 19 May 2011 12:00:00 GMT
	
BODY
	Txt  ~=  Post Page 
SETTING
	# Store recieved Cookies in Global
	Keep-Cookies  1
	Dump          1
	
	
---------------------
Access
---------------------
GET ${URL}/cookie.html
TAG
	li == Thesession :: randomsessionid
`,
	host, port)


func TestServer(t *testing.T) {
	go StartHandlers(port, t)
}


func StartHandlers(addr string, t *testing.T) (err os.Error) {
	http.Handle("/html.html", http.HandlerFunc(htmlHandler))
	http.Handle("/bin.bin", http.HandlerFunc(binHandler))
	http.Handle("/post", http.HandlerFunc(postHandler))
	http.Handle("/404.html", http.NotFoundHandler())
	http.Handle("/cookie.html", http.HandlerFunc(cookieHandler))
	fmt.Printf("\n\nRunning test server on %s\n\n", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Printf("Cannot run test server on port %s.\nFAIL\n", addr)
		os.Exit(1)
	}
	return
}

var htmlPat = `<!DOCTYPE html>
<html><head><title>Dummy HTML 1</title></head>
<body>
	<h1>Dummy Document for html based testst</h1>
	<p class="a">Some fancy text Braunschweig Weiler</p>
	%s
	<div>
		<form action="/post" method="post" enctype="multipart/form-data">
			Go to: <input type="text" name="q" value="Search" />
			File <input type="file" name="file-upload">
			<input type="submit" />
		</form>
	</div>
	<p class="b">Stupid stuff here.</p>
	<span>UTF-8 Umlaute äöüÄÖÜ Euro €</span>
</body>
`

func htmlHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Fancy-Header", "Important Value")
	w.WriteHeader(200)
	t := req.FormValue("text")
	s := req.FormValue("sleep")
	if ms, err := strconv.Atoi(s); err == nil {
		time.Sleep(1000000 * int64(ms))
	}
	if len(req.Cookie) > 0 {
		t += "\n<a href=\"/bin.bin\" title=\"TheCookieValue\">" + req.Cookie[0].Name + " = " + req.Cookie[0].Value + "</a>"
	}
	body := fmt.Sprintf(htmlPat, html.EscapeString(t))
	w.Write([]byte(body))
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if cv := req.FormValue("cookie"); cv != "" {
		trace("postHandler recieved param cookie %s.", cv)
		cp := strings.Split(cv, "=", 2)
		w.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Path=/de/index; expires=Fri, 20 May 2011 12:34:55 GMT; Domain=my.domain.org; Secure;", cp[0], cp[1]))
	}
	t := req.FormValue("q")
	if req.Method != "POST" {
		fmt.Printf("========= called /post with GET! ========\n")
	}

	_, header, err := req.FormFile("datei")
	if err == nil {
		info("Recieved datei: %s. %v", header.Filename, header.Filename == "file äöü 1.txt")
	}

	if t != "" {
		w.Header().Set("Location", host+port+"/"+t)
		w.WriteHeader(302)
	} else {
		text := req.FormValue("text")
		w.WriteHeader(200)
		body := "<html><body><h1>Post Page</h1><p>t = " + html.EscapeString(text) + "</p></body></html>"
		w.Write([]byte(body))
	}
}


func binHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/data")
	w.Header().Set("Fancy-Header", "Arbitary Value")
	var c int = 200
	if sc := req.FormValue("sc"); sc != "" {
		var e os.Error
		c, e = strconv.Atoi(sc)
		if e != nil {
			c = 500
		}
	}
	w.WriteHeader(c)
	w.Write([]byte("\001\007Hallo Welt!\n\377\376"))
}


func code(s string) string {
	return "<code>" + s + "</code>" // TODO: escape html
}

func cookieHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if cn := req.FormValue("set"); cn != "" {
		cv, cp := req.FormValue("val"), req.FormValue("pat")
		trace("cookieHandler recieved cookie %s=%s; path=%s.", cn, cv, cp)
		w.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Path=/de/index; Domain=my.domain.org; Secure;", cn, cv))
	}

	if t := req.FormValue("goto"); t != "" {
		w.Header().Set("Location", host+port+"/"+t)
		w.WriteHeader(302)
	} else {
		w.WriteHeader(200)
		body := "<html><head><title>Cookies</title></head>\n<body><h1>All Submitted Cookies</h1>"
		for _, cookie := range req.Cookie {
			body += "<div class=\"cookies\">\n"
			body += "  <ul>\n"
			body += "   <li>" + cookie.Name + " :: " + cookie.Value + "</li>\n"
			body += "  </ul>\n"
			body += "</div>\n"
		}
		body += "</body></html>"
		w.Write([]byte(body))
	}
}


func passed(test *Test, t *testing.T) bool {
	if !strings.HasPrefix(test.Status(), "PASSED") {
		t.Logf("Result from test %s:\n%s", test.Title, test.Result)
		t.Fail()
		return false
	}
	return true
}


func failed(test *Test, t *testing.T) bool {
	if strings.HasPrefix(test.Status(), "PASSED") {
		t.Logf("Test %s expected to fail but passed.\n%s", test.Title, test.Result)
		t.Fail()
		return false
	}
	return true
}

func TestParsing(t *testing.T) {
	suiteText := fmt.Sprintf(suiteTmpl, host, port)
	p := NewParser(strings.NewReader(suiteText), "suiteText")
	var err os.Error
	theSuite, err = p.ReadSuite()
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}
}

func TestTagStructParsing(t *testing.T) {
	var tagSuite = `
---------------------
Tag Spec
---------------------
GET x
TAG
	[
		div
			h2
			p
				span
			h3
	]
`

	p := NewParser(strings.NewReader(tagSuite), "tagSuite")
	s, err := p.ReadSuite()
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}
	erg := s.Test[0].String()
	if !strings.Contains(erg,
		"\t\t[\n\t\t\tdiv\n\t\t\t  h2\n\t\t\t  p\n\t\t\t    span\n\t\t\t  h3\n\t\t]\n") {
		t.Error("Nested tags parsed wrong")

	}
}


func TestBasic(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(0)
	passed(&theSuite.Test[0], t)
}

func TestBin(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(1)
	passed(&theSuite.Test[1], t)
	theSuite.RunTest(2)
	passed(&theSuite.Test[2], t)
}


func TestSequence(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(3)
	passed(&theSuite.Test[3], t)
}

func TestRandom(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(4)
	passed(&theSuite.Test[4], t)
}

func TestPlainPost(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(5)
	passed(&theSuite.Test[5], t)
}

func TestPost(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(6)
	passed(&theSuite.Test[6], t)
}

func TestTooSlow(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(7)
	failed(&theSuite.Test[7], t)
}

func TestGlobalSubst(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	Const["SomeGlobal"] = "GlobalValue"
	theSuite.RunTest(8)
	passed(&theSuite.Test[8], t)
}

func TestParameters(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(9)
	passed(&theSuite.Test[9], t)
}

func TestFileUpload(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(10)
	passed(&theSuite.Test[10], t)
}

func TestEncoding(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(11)
	passed(&theSuite.Test[11], t)
	theSuite.RunTest(12)
	passed(&theSuite.Test[12], t)
}

func TestMultipartPost(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(13)
	passed(&theSuite.Test[13], t)
}


func TestCookies(t *testing.T) {
	LogLevel = 4
	p := NewParser(strings.NewReader(cookieSuite), "cookieSuite")
	cs, err := p.ReadSuite()
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}

	cs.RunTest(0) // Display Cookies
	if !passed(&cs.Test[0], t) {
		t.FailNow()
	}

	cs.RunTest(1) // Login
	passed(&cs.Test[1], t)
	if !passed(&cs.Test[1], t) {
		t.FailNow()
	}

	cs.RunTest(2) // Access
	passed(&cs.Test[1], t)
}


func TestStayAlife(t *testing.T) {
	fmt.Printf("Will stay alive for %d seconds.\n", testserverStayAlive)
	time.Sleep(1000000000 * testserverStayAlive)
}


var backgroundSuite = `
----------------------
html
----------------------
GET ${URL}/html.html
SETTING
	Repeat 5
	Sleep  1
	
----------------------
bin
----------------------
GET ${URL}/bin.bin
SETTING
	Repeat 5
	Sleep  1
`

var stressSuite = `
----------------------
Basic
----------------------
GET ${URL}/html.html
RESPONSE
	Status-Code  ==  200
BODY
	 Txt  ~= Braunschweig
TAG
	 title == Dummy HTML 1
	 p class=a
	
----------------------
Binary
----------------------
GET ${URL}/bin.bin
RESPONSE
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!
`

func testPrintStResult(txt string, result StressResult) {
	fmt.Printf("%s: Response Time %5d / %5d / %5d (min/avg/max). Status %2d / %2d / %2d (err/pass/fail). %2d / %2d (tests/checks).\n",
		txt, result.MinRT, result.AvgRT, result.MaxRT, result.Err, result.Pass, result.Fail, result.N, result.Total)
}

func TestStresstest(t *testing.T) {
	if skipStress {
		return
	}
	LogLevel = 1
	bgText := strings.Replace(backgroundSuite, "${URL}", host+port, -1)
	suiteText := strings.Replace(stressSuite, "${URL}", host+port, -1)
	p := NewParser(strings.NewReader(bgText), "background")
	var background *Suite
	var err os.Error
	background, err = p.ReadSuite()
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}
	p = NewParser(strings.NewReader(suiteText), "suite")
	var suite *Suite
	suite, err = p.ReadSuite()
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}

	r0 := suite.Stresstest(background, 0, 3, 100)
	r10 := suite.Stresstest(background, 10, 3, 100)
	r30 := suite.Stresstest(background, 30, 2, 100)
	time.Sleep(100000000)
	r60 := suite.Stresstest(background, 60, 1, 100)
	time.Sleep(100000000)
	r100 := suite.Stresstest(background, 100, 1, 100)
	time.Sleep(200000000)
	r150 := suite.Stresstest(background, 150, 1, 100)
	time.Sleep(200000000)
	r200 := suite.Stresstest(background, 200, 5, 100)
	time.Sleep(500000000)

	testPrintStResult("Load   0", r0)
	testPrintStResult("Load  10", r10)
	testPrintStResult("Load  30", r30)
	testPrintStResult("Load  60", r60)
	testPrintStResult("Load 100", r100)
	testPrintStResult("Load 150", r150)
	testPrintStResult("Load 200", r200)
	if r0.Total <= 0 || r0.N <= 0 {
		t.Error("No tests run without load")
		t.FailNow()
	}
	if r0.Fail > 0 || r0.Err > 0 {
		t.Error("Failures without load")
		t.FailNow()
	}

	// There will be failures in the 200 run....
	if r200.Total <= 0 || r200.N <= 0 {
		t.Error("No tests run with load 200")
		t.FailNow()
	}
	if r200.Fail == 0 || r200.Err == 0 {
		t.Error("Expected Failures at load of 200!")
	}

}
