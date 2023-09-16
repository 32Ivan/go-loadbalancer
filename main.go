package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	address string
	proxy   httputil.ReverseProxy
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {

	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}

}

func newSimpleServer(address string) *simpleServer {
	serverUrl, err := url.Parse(address)

	handleErr(err)

	return &simpleServer{
		address: address,
		proxy:   *httputil.NewSingleHostReverseProxy(serverUrl),
	}

}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n ", err)
		os.Exit(1)
	}
}

func (s *simpleServer) Address() string { return s.address }

func (lb *simpleServer) IsAlive() bool { return true }

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func (lb *LoadBalancer) getNextAvaileableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {

	targetServer := lb.getNextAvaileableServer()

	fmt.Printf("forwarding request to address %q\n", targetServer.Address())

	targetServer.Serve(rw, r)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("http://www.bing.com"),
		newSimpleServer("http://www.duckduckgo.com"),
	}

	lb := NewLoadBalancer("8000", servers)

	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	http.HandleFunc("/", handleRedirect)

	fmt.Printf("server request at 'localhost: %s'\n", lb.port)

	http.ListenAndServe(":"+lb.port, nil)

}