// Command blockstore-server exposes a read-only, loopback development store.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"shadowdevice/internal/store"
)

func main() {
	root := flag.String("store", "", "ciphertext directory")
	addr := flag.String("listen", "127.0.0.1:8080", "loopback listen address")
	flag.Parse()
	if *root == "" {
		log.Fatal("-store required")
	}
	host, _, e := net.SplitHostPort(*addr)
	if e != nil {
		log.Fatal(e)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		log.Fatal("development server requires a numeric loopback address")
	}
	s, e := store.Open(*root)
	if e != nil {
		log.Fatal(e)
	}
	server := http.Server{Addr: *addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	fmt.Printf("Read-only development ciphertext server: http://%s\n", *addr)
	log.Fatal(server.ListenAndServe())
}
