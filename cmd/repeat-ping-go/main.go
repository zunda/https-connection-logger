package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"
)

type transport struct {
	current *http.Request

	StartedAt   time.Time
	CompletedAt time.Time
	TimeTotal   float64

	Url          string
	Errmsg       string
	LocalIp      string
	LocalPort    string
	RemoteIp     string
	RemotePort   string
	ResponseCode int
	SrcIp        string
	RequestCount int
}

func (t *transport) GotConn(info httptrace.GotConnInfo) {
	t.LocalIp, t.LocalPort, _ = net.SplitHostPort(info.Conn.LocalAddr().String())
	t.RemoteIp, t.RemotePort, _ = net.SplitHostPort(info.Conn.RemoteAddr().String())
	if info.Reused {
		t.RequestCount += 1
	} else {
		t.RequestCount = 1
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	return http.DefaultTransport.RoundTrip(req)
}

func (t *transport) log() {
	e := json.NewEncoder(os.Stdout)
	t.TimeTotal = t.CompletedAt.Sub(t.StartedAt).Seconds()
	err := e.Encode(t)
	if err != nil {
		fmt.Println(os.Stderr, err.Error())
	}
	os.Stdout.Sync()
}

func get(req *http.Request, transport *transport) {
	transport.Url = req.URL.String()
	transport.StartedAt = time.Now()
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	transport.CompletedAt = time.Now()
	if err != nil {
		transport.Errmsg = err.Error()
		goto Log
	}
	defer res.Body.Close()
	transport.ResponseCode = res.StatusCode
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		transport.Errmsg = err.Error()
		goto Log
	}
Log:
	transport.log()
}

func repeatGet(interval time.Duration, req *http.Request, transport *transport) {
	for {
		get(req, transport)
		time.Sleep(interval)
	}
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
	t.SrcIp = srcIp()

	url := "https://mastodon.zunda.ninja/health"
	req, _ := http.NewRequest("GET", url, nil)
	trace := &httptrace.ClientTrace{
		GotConn: t.GotConn,
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	repeatGet(5 * time.Second, req, t)
}
