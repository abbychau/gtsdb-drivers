// a server listen to 5554 tcp print data
package main

import (
	"bufio"
	"log"
	"net"
)

func main() {
	listerner, err := net.Listen("tcp", ":5554")
	if err != nil {
		log.Fatal(err)
	}

	tsdbClient, err := NewTSDBClient("localhost:5555")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listerner.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(c net.Conn) {
			defer c.Close()
			scanner := bufio.NewScanner(c)
			for scanner.Scan() {
				log.Println(scanner.Text())
				tsdbClient.RecordMeasurement("111", 3.33)
			}
		}(conn)
	}
}
