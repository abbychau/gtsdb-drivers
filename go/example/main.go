// Example: Go GTSDB driver usage.
//
//	go run example/main.go
package main

import (
	"fmt"
	"os"

	"github.com/abbychau/gtsdb-drivers/go"
)

func main() {
	addr := "localhost:5555"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	fmt.Println("Connecting to", addr, "...")
	client, err := gtsdb.Connect(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("✓ Connected")

	// Auth (skip if no-auth mode)
	_ = client.Auth("")

	// Write
	fmt.Print("Write... ")
	if err := client.Write("test.sensor", 42.5); err != nil {
		fmt.Println("FAIL:", err)
	} else {
		fmt.Println("✓")
	}

	// Write with timestamp
	fmt.Print("WriteAt... ")
	if err := client.WriteAt("test.sensor", 99.9, 1717965210); err != nil {
		fmt.Println("FAIL:", err)
	} else {
		fmt.Println("✓")
	}

	// Read JSON
	fmt.Print("ReadLast JSON... ")
	pts, err := client.ReadLast("test.sensor", 10)
	if err != nil {
		fmt.Println("FAIL:", err)
	} else {
		fmt.Printf("✓ (%d points)\n", len(pts))
		if len(pts) > 0 {
			fmt.Printf("  latest: ts=%d val=%f\n", pts[len(pts)-1].Timestamp, pts[len(pts)-1].Value)
		}
	}

	// Read binary 🚀
	fmt.Print("ReadLast Binary... ")
	pts, err = client.ReadBinary("test.sensor", 10)
	if err != nil {
		fmt.Println("FAIL:", err)
	} else {
		fmt.Printf("✓ (%d points)\n", len(pts))
		if len(pts) > 0 {
			fmt.Printf("  latest: ts=%d val=%f\n", pts[len(pts)-1].Timestamp, pts[len(pts)-1].Value)
		}
	}

	// Batch write
	fmt.Print("BatchWrite... ")
	err = client.BatchWrite([]gtsdb.DataPoint{
		{Key: "test.batch1", Value: 1.0},
		{Key: "test.batch2", Value: 2.0},
		{Key: "test.batch3", Value: 3.0},
	})
	if err != nil {
		fmt.Println("FAIL:", err)
	} else {
		fmt.Println("✓")
	}

	// IDs
	fmt.Print("IDs... ")
	ids, err := client.IDs()
	if err != nil {
		fmt.Println("FAIL:", err)
	} else {
		fmt.Printf("✓ (%d keys: %v)\n", len(ids), ids[:min(3, len(ids))])
	}

	fmt.Println("\nAll tests passed!")
}
