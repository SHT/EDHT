package main

import (
	"fmt"
	bt "github.com/ecc1/ble"
	"net"
	"time"
)

// Connect initializes a TCP socket connection to the destination address
// It blocks until a connection is established
func Connect() net.Conn {
	// Format address string
	address := fmt.Sprintf("%s%s%d", "192.168.1.22", ":", 4198)
	// Check for connection until one is established
	for {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			time.Sleep(1 * time.Second)
		} else {
			fmt.Println("Connection established.")
			return conn
		}
	}
}

// Send sends the ambilight data to the destination
func Send(conn net.Conn, data []byte) error {
	_, err := conn.Write(data)
	return err
}

// Disconnect closes an existing TCP connection
func Disconnect(conn net.Conn) error {
	err := conn.Close()
	return err
}

func main() {
	sock, err := bt.Open()
	if err != nil {
		fmt.Println(err)
		return
	}
	d, err := sock.GetDevice("00001801-0000-1000-8000-00805f9b34fb")
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		err = d.Connect()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		_, err = sock.GetService("0000fe55-0000-1000-8000-00805f9b34fb")
		if err != nil {
			fmt.Println(err)
			return
		}
		c, err := sock.GetCharacteristic("00000001-1000-1000-8000-00805f9b34fb")
		if err != nil {
			fmt.Println(err)
			return
		}
		// No need to check for error
		_ = c.StartNotify()
		// Try to connect indefinitely
		for {
			conn := Connect()
			connected := true
			for {
				c.HandleNotify(func(stream []byte) {
					// Attempt to send the data to the server
					err := Send(conn, stream)
					if err != nil {
						// Close the connection, don't check for errors
						_ = Disconnect(conn)
						connected = false
					}
				})
				// If connection is closed then break the notify loop
				if !connected {
					break
				}
			}
			// Try to reconnect every second
			time.Sleep(1 * time.Second)
		}
	}
}
