package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// The Server interface defines the requirements that servers must implement for load balancer management.
type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

// simpleServer is an implementation of the Server interface representing an individual server.
type simpleServer struct {
	address string
	proxy   httputil.ReverseProxy
}

// LoadBalancer represents a load balancer that distributes traffic among multiple servers.
type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

// NewLoadBalancer is a constructor for the LoadBalancer.
func NewLoadBalancer(port string, servers []Server) *LoadBalancer {

	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}

}

// newSimpleServer is a constructor for simpleServer.
func newSimpleServer(address string, isHTTPS bool) *simpleServer {
	serverUrl, err := url.Parse(address)

	handleErr(err)

	proxy := httputil.NewSingleHostReverseProxy(serverUrl)

	if isHTTPS {

		certFile := "///myserver.cert" // Replace with the path to your certificate
		keyFile := "///myserver.key"   //  Replace with the path to your private key

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			handleErr(err)
		}

		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert}},
		}
	}

	return &simpleServer{
		address: address,
		proxy:   *proxy,
	}

}

// handleErr is an error handling function.
func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n ", err)
		os.Exit(1)
	}
}

// Address returns the server's address.
func (s *simpleServer) Address() string {
	return s.address
}

// IsAlive checks if the server is active.
func (lb *simpleServer) IsAlive() bool {
	return true
}

// Serve forwards requests through the proxy..
func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

// getNextAvailableServer returns the next available server using a round-robin approach.
func (lb *LoadBalancer) getNextAvaileableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

// serveProxy handles client requests and forwards them to the appropriate server.
func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {

	targetServer := lb.getNextAvaileableServer()

	fmt.Printf("forwarding request to address %q\n", targetServer.Address())

	targetServer.Serve(rw, r)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com", true),
		newSimpleServer("https://www.bing.com", true),
		newSimpleServer("https://www.duckduckgo.com", true),
	}

	lb := NewLoadBalancer("8000", servers)

	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	http.HandleFunc("/", handleRedirect)

	fmt.Printf("server request at 'localhost: %s'\n", lb.port)

	certFile := "/myserver.cert" //  Replace with the path to your certificate
	keyFile := "/myserver.key"   //  Replace with the path to your private key

	http.ListenAndServeTLS(":"+lb.port, certFile, keyFile, nil)

}
