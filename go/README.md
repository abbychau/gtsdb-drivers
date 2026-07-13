# GTSDB Go Driver

Go client for [GTSDB](https://github.com/abbychau/gtsdb) with JSON and binary protocol support.

## Install

```bash
go get github.com/abbychau/gtsdb-drivers/go@latest
```

## Quick Start

```go
import "github.com/abbychau/gtsdb-drivers/go"

client, _ := gtsdb.Connect("localhost:5555")
defer client.Close()
client.Auth("your-token")

// Write
client.Write("sensor1", 42.5)
client.WriteAt("sensor1", 99.9, 1717965210)

// JSON reads
points, _ := client.ReadLast("sensor1", 100)
points, _ := client.Read("sensor1", startTs, endTs, 0)

// 🚀 Binary reads — zero-alloc, 100x faster for bulk
points, _ := client.ReadBinary("sensor1", 5000)
multi, _ := client.MultiReadBinary([]string{"s1", "s2"}, 5000)

// Batch write
client.BatchWrite([]gtsdb.DataPoint{
    {Key: "s1", Value: 1.0},
    {Key: "s2", Value: 2.0, Timestamp: 1717965210},
})

// Subscribe
client.Subscribe("sensor1", func(dp gtsdb.DataPoint) {
    fmt.Printf("%s: %f\n", dp.Key, dp.Value)
})
```

## API

| Method | Description |
|---|---|
| `Connect(addr)` | TCP connection |
| `Close()` | Disconnect |
| `Auth(token)` | Authenticate |
| `Write(key, value)` | Write (server timestamp) |
| `WriteAt(key, value, ts)` | Write with timestamp |
| `Read(key, startTs, endTs, downsample)` | Read time range (JSON) |
| `ReadLast(key, n)` | Read last N (JSON) |
| `ReadBinary(key, n)` | Read last N (binary 🚀) |
| `MultiRead(keys, lastX)` | Multi-key read (JSON) |
| `MultiReadBinary(keys, lastX)` | Multi-key read (binary 🚀) |
| `BatchWrite(points)` | Batch insert |
| `IDs()` | List all keys |
| `Subscribe(key, callback)` | Real-time stream |
| `Unsubscribe(key)` | Stop streaming |

## Run Example

```bash
go run example/main.go [host:port]
```

## Test

```bash
go test ./...
```

## License

MIT
