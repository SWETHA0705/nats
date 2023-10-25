package main

import (
	"log"
	"net"
	"time"
)

func main() {
	serverAddress := "192.168.1.31" // Replace with your server's IP address

	for i:=1;i<=3;i++ {
		err := ping(serverAddress)
		if err != nil {
			log.Printf("Server is down: %s", err)

		} else {
			log.Println("Server is up and running.")
		}
		time.Sleep(10 * time.Second) // Adjust the time interval for health checks
	}
}

func ping(address string) error {
	conn, err := net.DialTimeout("tcp", address+ ":50052", 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
