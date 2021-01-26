package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var portsFlag = flag.String(
	"ports",
	"8080",
	"Comma delimited list of ports, where the app will be listening to",
)

func main() {
	startTime := time.Now()
	flag.Parse()
	ports := strings.Split(*portsFlag, ",")

	wg := sync.WaitGroup{}
	for _, port := range ports {
		wg.Add(1)
		go func(wg *sync.WaitGroup, port string) {
			defer wg.Done()

			mux := http.NewServeMux()

			rootHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(port + "\n"))
			})

			uptimeHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				uptime := time.Since(startTime)
				w.Write([]byte(uptime.String()))
			})

			mux.Handle("/", rootHandler)
			mux.Handle("/uptime", uptimeHandler)

			log.Fatal(http.ListenAndServe(":"+port, mux))
		}(&wg, port)
	}
	println("Listening on ports ", strings.Join(ports, ", "))
	wg.Wait()
}
