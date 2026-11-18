package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
)

type transport struct {
	current *http.Request
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	return http.DefaultTransport.RoundTrip(req)
}

func (t *transport) GotConn(info httptrace.GotConnInfo) {
	log.Printf("%+v: %+v", t.current, info)
}

func get(req *http.Request, transport *transport) {
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
}

func main() {
	t := &transport{}

	url := "https://mastodon.zunda.ninja/health"
	req, _ := http.NewRequest("GET", url, nil)
	trace := &httptrace.ClientTrace{
		GotConn: t.GotConn,
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	get(req, t)
	get(req, t)
}
