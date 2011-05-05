package tag

import (
	"testing"
	"fmt"
	"strings"
)


var testStructureHtml = `<html>
<body>
	<h1> A </h1>
	<p> B
		<span> C </span>
		D
	</p>
	<h2> E </h2>
	<div>
		<p> F </p>
		<p> G </p>
	</div>
</body>
</html>
`

var testXhtml = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="de" xml:lang="de">
	<head>
		<title>Some XHTML</title>
		<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	</head>
	<body>
		<h1>X-HTML Test</h1>
		<p>The Body</p>
	</bod>
</html>
`

var testBrokenHtml1 = `<!DOCTYPE html>
<html>
<body>
	<div id="div1">
		<span id="SP1">Some aaaa text</bug>
	</wrong>
	<p>Completely Skipped</p>
</body>
</html>`

var testBrokenHtml2 = `<!DOCTYPE html>
<html>
<body>
	<div id="div1"> <!-- MyComment -->
		<span id="SP1>Some aaaa text</span>
	</div>
	<p>Some Text</p>
</body>
</html>`

var testEntitiesHtml = `<html><body>
<p>a &lt; b &gt; c. A&amp;B. x=&quot;Hallo&quot;. Copy &copy;. Umlaute: äöü = &auml;&ouml;&uuml;.</p>
</body></html>`

func testStructure(doc *Node, expected []string, t *testing.T) {
	lines := strings.Split(doc.HtmlRep(0), "\n", -1)
	for i, etag := range expected {
		a, b := "<"+etag+" ", "<"+etag+">"
		got := strings.Trim(lines[i], " \t")
		if !(strings.HasPrefix(got, a) || strings.HasPrefix(got, b)) {
			t.Errorf("Expected %s on line %d but got %s.", etag, i, got)
		}
	}
}

func testHtmlParsing(html string, expected []string, t *testing.T) {
	doc, err := ParseHtml(html)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}
	testStructure(doc, expected, t)
}

//  Testcases below

func TestMostSimpleHtml(t *testing.T) {
	LogLevel = 4
	testHtmlParsing("<html><body>Hello</body></html>", []string{"html", "body"}, t)
}

func TestSimpleHtmlParsing(t *testing.T) {
	LogLevel = 3
	testHtmlParsing(testStructureHtml, []string{"html", "body", "h1", "p", "span", "h2", "div", "p", "p"}, t)
}

func TestXHtmlParsing(t *testing.T) {
	LogLevel = 3
	testHtmlParsing(testXhtml, []string{"html", "head", "title", "meta", "body", "h1", "p"}, t)
}

func TestHtmlEntitiesParsing(t *testing.T) {
	LogLevel = 3
	doc, err := ParseHtml(testEntitiesHtml)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}
	lines := strings.Split(doc.HtmlRep(0), "\n", -1)
	for i, exp := range []string{"<html>", "<body>", "<p> a < b > c. A&B. x=\"Hallo\". Copy ©. Umlaute: äöü = äöü."} {
		got := strings.Trim(lines[i], " \t")
		if !strings.HasPrefix(got, exp) {
			t.Errorf("Expected %s on line %d but got %s.", exp, i, got)
		}
	}
}


func TestBrokenClosingTagParsing(t *testing.T) {
	testHtmlParsing(testBrokenHtml1, []string{"html", "body", "div", "span"}, t)
}

func TestBrokenQuoteParsing(t *testing.T) {
	LogLevel = 2
	doc, err := ParseHtml(testBrokenHtml2)
	if err == nil {
		t.Error("No error detected on broken html 2 ")
		fmt.Printf("Resulting html structure:\n%s\n", doc.HtmlRep(0))
	}
}

func BenchmarkParsing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseHtml(testSimpleHtml)
	}
}