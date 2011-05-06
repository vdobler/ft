package suite

import (
	"fmt"
	"strings"
	"os"
	"http"
	"io"
	"bytes"
	"strconv"
)


func readBody(r io.ReadCloser) string {
	var bb bytes.Buffer
	if r != nil {
		io.Copy(&bb, r)
		r.Close()
	}
	body := bb.String()
	trace("Read body with len = %d:\n%s\n", len(body), body)
	return body
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
	for k, v := range t.Header {
		trace("req.Header = %v", req.Header)
		req.Header.Set(k, v)
	}
}

func DoAndFollow(req *http.Request) (r *http.Response, finalUrl string, err os.Error) {
	trace("DoAndFollow: %s %s", req.Method, req.URL.String())
	
	r, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	finalUrl = req.URL.String()

	if !shouldRedirect(r.StatusCode) {
		return
	}
	
	// Start redirecting to final destination
	r.Body.Close()
	var base = req.URL
	
	// Following the redirect chain is done with a clean/empty new GET request
	req = new(http.Request)
	req.Method = "GET"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
		
	for redirect:=0; redirect<10; redirect++ {
		var url string
		if url = r.Header.Get("Location"); url == "" {
			fmt.Printf("Header:\n%v", r.Header)
			err = os.ErrorString(fmt.Sprintf("%d response missing Location header", r.StatusCode))
			return
		}
		if base == nil {
			req.URL, err = http.ParseURL(url)
		} else {
			req.URL, err = base.ParseURL(url)
		}
		if err != nil {
			return
		}

		url = req.URL.String()
		info("GET %s", url)
		if r, err = http.DefaultClient.Do(req); err != nil {
			return
		}
		finalUrl = url

		if !shouldRedirect(r.StatusCode) {
			return
		}
		r.Body.Close()
		base = req.URL

	}
	err = os.ErrorString("stopped after 10 redirects")
	return
}

func Get(t *Test) (r *http.Response, finalURL string, err os.Error) {
	var url = t.Url // <-- Patched
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
		// vvvv Patched vvvv
		req.Header = http.Header{}
		if len(t.Param) > 0 {
			ep := http.EncodeQuery(t.Param)
			// TODO handle #-case
			if strings.Contains(url, "?") {
				url = url + "&" + ep
			} else {
				url = url + "?" + ep
			}
		}
		// ^^^^ Patched ^^^^
		if base == nil {
			req.URL, err = http.ParseURL(url)
		} else {
			req.URL, err = base.ParseURL(url)
		}
		if err != nil {
			break
		}
		addHeaders(&req, t)  // <-- Patched
		url = req.URL.String()
		info("GET %s", url)
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


// PostForm issues a POST to the specified URL, 
// with data's keys and values urlencoded as the request body.
//
// Caller should close r.Body when done reading from it.
func Post(t *Test) (r *http.Response, finalUrl string, err os.Error) {
	var req http.Request
	var url = t.Url  //  <-- Patched
	req.Method = "POST"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Close = true
	// vvvvvv Patched vvvvv
	// body := urlencode(data)
	bodystr := http.EncodeQuery(t.Param)
	debug("Body:\n%s", bodystr)
	body := bytes.NewBuffer([]byte(bodystr))
	// ^^^^^^^Patched ^^^^^^
	req.Body = nopCloser{body}
	req.Header = http.Header{
		"Content-Type":   {"application/x-www-form-urlencoded"},
		"Content-Length": {strconv.Itoa(body.Len())},
	}
	addHeaders(&req, t)  // <-- Patched

	req.ContentLength = int64(body.Len())

	req.URL, err = http.ParseURL(url)
	if err != nil {
		return nil, url, err
	}
	debug("Will post to %s", req.URL.String())
	r, finalUrl, err = DoAndFollow(&req)
	return 
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() os.Error { return nil }

