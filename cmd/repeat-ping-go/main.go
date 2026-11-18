package main

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"
)

type transport struct {
	current *http.Request

	dnsStartAt time.Time
	dnsDoneAt time.Time
	startedAt time.Time
	gotConnAt time.Time
	gotFirstResponseByteAt time.Time
	completedAt time.Time

	url string
	errmsg string
	localIp string
	localPort string
	remoteIp string
	remotePort string
	responseCode int
	srcIp string
}

func (t *transport) DNSStart(_ httptrace.DNSStartInfo) {
	t.dnsStartAt = time.Now()
}

func (t *transport) DNSDone(_ httptrace.DNSDoneInfo) {
	t.dnsDoneAt = time.Now()
}

func (t *transport) GotConn(info httptrace.GotConnInfo) {
	t.gotConnAt = time.Now()
	t.localIp, t.localPort, _ = net.SplitHostPort(info.Conn.LocalAddr().String())
	t.remoteIp, t.remotePort, _ = net.SplitHostPort(info.Conn.RemoteAddr().String())
}

func (t *transport) GotFirstResponseByte() {
	t.gotFirstResponseByteAt = time.Now()
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	return http.DefaultTransport.RoundTrip(req)
}

func get(req *http.Request, transport *transport) {
	transport.url = req.URL.String()
	transport.startedAt = time.Now()
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	transport.completedAt = time.Now()
	if err != nil {
		transport.errmsg = err.Error()
		goto Log
	}
	defer res.Body.Close()
	transport.responseCode = res.StatusCode
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		transport.errmsg = err.Error()
		goto Log
	}
Log:
	log.Printf("%+v", transport)
}

func srcIp() string {
	res, err := http.Get("https://4.ipaddr.io/ip")
	if err != nil {
		return err.Error()
	}
	defer res.Body.Close()
	srcIp, err := io.ReadAll(res.Body)
	if err != nil {
		return err.Error()
	}
	return string(srcIp)
}

func main() {
	t := &transport{}
	t.srcIp = srcIp()

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
