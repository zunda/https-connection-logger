package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
)

func get(req *http.Request) {
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
}

func main() {
	url := "https://mastodon.zunda.ninja/health"
	req, _ := http.NewRequest("GET", url, nil)
	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			fmt.Printf("Got Conn: %+v\n", connInfo)
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			fmt.Printf("DNS Info: %+v\n", dnsInfo)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	get(req)
	get(req)
}
