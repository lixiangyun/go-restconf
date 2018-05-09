package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var (
	addr    string
	verbose bool
	help    bool
)

var (
	APPLICATION_XRD_XML   = "application/xrd+xml"
	APPLICATION_DATA_XML  = "application/yang-data+xml"
	APPLICATION_DATA_JSON = "application/yang-data+json"

	RESTCONF_PREFIX = "/restconf"

	PUBLIC_XMLNS = "urn:ietf:params:xml:ns:yang:ietf-restconf"

	YANG_LIBRARY_VERSION = "2016-06-21"

	DEFAULT_LISTEN_ADDR = ":408"

	yangLibVer = YangLibVer{Version: YANG_LIBRARY_VERSION, XmlLns: PUBLIC_XMLNS}
)

func init() {

	flag.BoolVar(&help, "h", false, "show help")
	flag.BoolVar(&verbose, "v", false, "show version")
	flag.StringVar(&addr, "addr", DEFAULT_LISTEN_ADDR, "restconf listen address")

	// 改变默认的 Usage
	flag.Usage = usage
}

func usage() {

	fmt.Fprintf(os.Stderr, ` Version: restconf/0.1.0
 Usage: resfconf [-hv] [-addr ip:port]

 Options:
`)

	flag.PrintDefaults()
}

type YangLibVer struct {
	XMLName xml.Name `json:"-" xml:"yang-library-version"`
	XmlLns  string   `json:"-" xml:"xmlns,attr"`
	Version string   `json:"yang-library-version" xml:",innerxml"`
}

type RestConf struct {
	mux map[string]http.HandlerFunc
}

func NewRestConf() *RestConf {
	server := new(RestConf)

	server.mux = make(map[string]http.HandlerFunc)

	server.Reg("/.well-known/host-meta", server.HostMeta)

	server.Reg(RESTCONF_PREFIX, server.Root)
	server.Reg(RESTCONF_PREFIX+"/data", server.Data)
	server.Reg(RESTCONF_PREFIX+"/operations", server.Operations)
	server.Reg(RESTCONF_PREFIX+"/yang-library-version", server.YangLibVer)

	return server
}

func (restconf *RestConf) Reg(url string, handler http.HandlerFunc) {
	_, b := restconf.mux[url]
	if b == false {
		restconf.mux[url] = func(rsp http.ResponseWriter, req *http.Request) {
			rsp.Header().Set("Server", "RESTCONF")
			rsp.Header().Set("Date", time.Now().Format(time.RFC1123))
			handler(rsp, req)
		}
	} else {
		log.Fatal("this handler " + url + " exist!")
	}
}

func (restconf *RestConf) HostMeta(rsp http.ResponseWriter, req *http.Request) {

	if req.Method != "GET" {
		http.Error(rsp, "method is not GET!", http.StatusBadRequest)
		return
	}

	if req.Header.Get("Accept") != APPLICATION_XRD_XML {
		http.Error(rsp, "Accept is incorrect!", http.StatusBadRequest)
		return
	}

	body := `<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
		<Link rel='restconf' href='` + RESTCONF_PREFIX + `'/>
	</XRD>`

	rsp.Header().Set("Content-Type", APPLICATION_XRD_XML)
	rsp.WriteHeader(http.StatusOK)

	fmt.Fprint(rsp, body)
}

func (restconf *RestConf) Root(rsp http.ResponseWriter, req *http.Request) {

	/*
	   {
	     "ietf-restconf:restconf" : {
	       "data" : {},
	       "operations" : {},
	       "yang-library-version" : "2016-06-21"
	     }
	   }
	*/

}

func (restconf *RestConf) Data(rsp http.ResponseWriter, req *http.Request) {

}

func (restconf *RestConf) Operations(rsp http.ResponseWriter, req *http.Request) {

}

func (restconf *RestConf) YangLibVer(rsp http.ResponseWriter, req *http.Request) {

	var body []byte
	var err error

	format := req.Header.Get("Accept")

	switch format {
	case APPLICATION_DATA_XML:
		{
			body, err = xml.Marshal(yangLibVer)
		}
	case APPLICATION_DATA_JSON:
		{
			body, err = json.Marshal(yangLibVer)
		}
	default:
		{
			http.Error(rsp, "Accept is incorrect!", http.StatusBadRequest)
			return
		}
	}

	if err != nil {
		http.Error(rsp, "Marshal failed!"+err.Error(), http.StatusExpectationFailed)
		return
	}

	rsp.Header().Set("Content-Type", format)
	rsp.WriteHeader(http.StatusOK)

	fmt.Fprint(rsp, string(body))
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

func (restconf *RestConf) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	path := cleanPath(req.URL.Path)

	fun, b := restconf.mux[path]
	if b == true {
		fun(rsp, req)
		return
	}
	for url, fun := range restconf.mux {
		if strings.HasPrefix(path, url) {
			fun(rsp, req)
			return
		}
	}

	http.NotFound(rsp, req)
}

func main() {
	flag.Parse()
	if help || verbose {
		flag.Usage()
		return
	}

	server := NewRestConf()
	log.Println("restconf start and listen ", addr)

	err := http.ListenAndServe(addr, server)
	if err != nil {
		log.Fatal(err.Error())
	}
}
