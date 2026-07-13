# GTSDB Drivers

Client libraries for [GTSDB](https://github.com/abbychau/gtsdb) with JSON and binary protocol support.

## Go (`go/`)

### Install

```bash
go get github.com/abbychau/gtsdb-drivers/go@latest
```

### Quick Start

```go
import "github.com/abbychau/gtsdb-drivers/go"

client, _ := gtsdb.Connect("localhost:5555")
defer client.Close()
client.Auth("your-token")

// Write
client.Write("sensor1", 42.5)
client.WriteAt("sensor1", 99.9, 1717965210) // with timestamp

// JSON reads
points, _ := client.ReadLast("sensor1", 100)
points, _ := client.Read("sensor1", startTs, endTs, 0)
ids, _ := client.IDs()

// 🚀 Binary reads — zero-alloc, 100x faster for bulk
points, _ := client.ReadBinary("sensor1", 5000)
multi, _ := client.MultiReadBinary([]string{"s1", "s2"}, 5000)

// Batch write
client.BatchWrite([]gtsdb.DataPoint{
    {Key: "s1", Value: 1.0, Timestamp: 1717965210},
    {Key: "s2", Value: 2.0},
})

// Subscribe to real-time updates
client.Subscribe("sensor1", func(dp gtsdb.DataPoint) {
    fmt.Printf("%s: %f\n", dp.Key, dp.Value)
})
```

### API

| Method | Description |
|---|---|
| `Connect(addr)` | TCP connection |
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
| `Close()` | Disconnect |

## Node.js (`js/`)

### Install

```bash
npm install github:abbychau/gtsdb-drivers
```

### Quick Start

```js
const { GTSDBClient } = require('gtsdb-drivers/js/gtsdb');
const client = new GTSDBClient('localhost', 5555);

await client.connect();
await client.auth('your-token');

// Write
await client.write('sensor1', 42.5);
await client.write('sensor1', 99.9, 1717965210); // with timestamp

// JSON reads
const points = await client.readLast('sensor1', 100);
const ids = await client.ids();

// 🚀 Binary reads
const points = await client.readBinary('sensor1', 5000);
const multi = await client.multiReadBinary(['s1', 's2'], 5000);

// Batch write
await client.batchWrite([
    { key: 's1', value: 1.0, timestamp: 1717965210 },
    { key: 's2', value: 2.0 },
]);

// Subscribe
await client.subscribe('sensor1', (dp) => {
    console.log(dp.key, dp.value);
});
```

### API

| Method | Description |
|---|---|
| `new GTSDBClient(host, port)` | Create client |
| `connect()` | TCP connection |
| `auth(token)` | Authenticate |
| `write(key, value, ts?)` | Write point |
| `read(key, startTs, endTs, downsample?)` | Read time range (JSON) |
| `readLast(key, n)` | Read last N (JSON) |
| `readBinary(key, n)` | Read last N (binary 🚀) |
| `multiReadBinary(keys, lastX)` | Multi-key read (binary 🚀) |
| `batchWrite(points)` | Batch insert |
| `ids()` | List all keys |
| `subscribe(key, callback)` | Real-time stream |
| `unsubscribe(key)` | Stop streaming |
| `close()` | Disconnect |

## Testing

```bash
cd go && go test ./...     # Go tests (7)
cd js && npm test           # Node.js tests (5)
```

## Protocol

Default: JSON. Opt into binary with `"response_format": "binary"` for read operations.

Binary frame: `[4B length] [4B key_count] [2B key_len] [key_bytes] [4B pt_count] [8B ts] [8B val]...`
