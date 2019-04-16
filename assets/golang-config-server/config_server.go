package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/config/", config)
	fmt.Println("listening...")
	err := http.ListenAndServe(":"+os.Getenv("CONFIG_SERVER_PORT"), nil)
	if err != nil {
		panic(err)
	}
}

type Config struct {
	Scope    string
	Password string
}

func hello(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "Example config server")
}

func config(res http.ResponseWriter, req *http.Request) {
	config := Config{"dora.admin", "not-a-real-p4$$w0rd"}

	js, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}

	fmt.Fprintln(res, string(js))
}
