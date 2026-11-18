package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"time"
)

type transport struct {
	current *http.Request

	dnsStartAt time.Time
	dnsDoneAt time.Time
	startRequestAt time.Time
	gotConnAt time.Time
	gotFirstResponseByteAt time.Time
	receivedResponseAt time.Time

	localAddr string
	remoteAddr string
}

func (t *transport) DNSStart(_ httptrace.DNSStartInfo) {
	t.dnsStartAt = time.Now()
}

func (t *transport) DNSDone(_ httptrace.DNSDoneInfo) {
	t.dnsDoneAt = time.Now()
}

func (t *transport) GotConn(info httptrace.GotConnInfo) {
	t.gotConnAt = time.Now()
	t.localAddr = info.Conn.LocalAddr().String()
	t.remoteAddr = info.Conn.RemoteAddr().String()
}

func (t *transport) GotFirstResponseByte() {
	t.gotFirstResponseByteAt = time.Now()
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	return http.DefaultTransport.RoundTrip(req)
}

func get(req *http.Request, transport *transport) {
	client := &http.Client{Transport: transport}
	transport.startRequestAt = time.Now()
	res, err := client.Do(req)
	transport.receivedResponseAt = time.Now()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
	log.Printf("%+v", transport)
}

func main() {
	t := &transport{}

	url := "https://mastodon.zunda.ninja/health"
	req, _ := http.NewRequest("GET", url, nil)
	trace := &httptrace.ClientTrace{
		GotConn: t.GotConn,
		DNSStart: t.DNSStart,
		DNSDone: t.DNSDone,
		GotFirstResponseByte: t.GotFirstResponseByte,
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	get(req, t)
	time.Sleep(5 * time.Second)
	get(req, t)
}
