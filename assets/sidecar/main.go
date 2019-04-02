package main

import "log"
import "net"

func sidecarServer(c net.Conn) {
	for {
		buf := make([]byte, 512)
		_, err := c.Read(buf)
		if err != nil {
			return
		}

		_, err = c.Write([]byte("Sidecar received your data"))
		if err != nil {
			log.Fatal("Write: ", err)
		}
	}
}

func main() {
	l, err := net.Listen("unix", "/tmp/sidecar.sock")
	if err != nil {
		log.Fatal("Listen error:", err)
	}

	for {
		fd, err := l.Accept()
		if err != nil {
			log.Fatal("Accept error:", err)
		}

		go sidecarServer(fd)
	}
}
