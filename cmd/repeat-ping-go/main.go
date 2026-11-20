/*
Usage:

  INTERVAL=60 TARGET=https://zunda.ninja repeat-ping-go

repeatedly sends requests to the TARGET with the INTERVAL
and prints metrics
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"time"
)

const (
	defaultInterval = 600
	defaultUrl      = "https://zunda.ninja"
)

type transport struct {
	current *http.Request

	StartedAt   time.Time
	CompletedAt time.Time
	TimeTotal   float64

	Url            string
	Errmsg         string
	LocalIp        string
	LocalPort      string
	RemoteIp       string
	RemotePort     string
	ResponseCode   int
	SrcIp          string
	ReqOnConnCount int
}

func (t *transport) GotConn(info httptrace.GotConnInfo) {
	t.LocalIp, t.LocalPort, _ = net.SplitHostPort(info.Conn.LocalAddr().String())
	t.RemoteIp, t.RemotePort, _ = net.SplitHostPort(info.Conn.RemoteAddr().String())
	if info.Reused {
		t.ReqOnConnCount += 1
	} else {
		t.ReqOnConnCount = 1
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

var optInterval float64
var optUrl string

func init() {
	flag.Float64Var(&optInterval, "i", defaultInterval, "interval between requests (sec)")
	flag.StringVar(&optUrl, "t", defaultUrl, "target URL")
}

func main() {
	flag.Parse()

	var interval float64
	var url string

	i := os.Getenv("INTERVAL")
	if i != "" {
		interval, _ = strconv.ParseFloat(i, 64)
	}
	if interval == 0 {
		interval = optInterval
	}

	url = os.Getenv("TARGET")
	if url == "" {
		url = optUrl
	}

	t := &transport{}
	t.SrcIp = srcIp()
	req, _ := http.NewRequest("GET", url, nil)
	trace := &httptrace.ClientTrace{
		GotConn: t.GotConn,
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	repeatGet(time.Duration(interval)*time.Second, req, t)
}
