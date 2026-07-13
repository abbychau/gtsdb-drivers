# GTSDB Node.js Driver

Node.js client for [GTSDB](https://github.com/abbychau/gtsdb) with JSON and binary protocol support.

## Install

```bash
npm install github:abbychau/gtsdb-drivers
```

## Quick Start

```js
const { GTSDBClient } = require('gtsdb-drivers');
const client = new GTSDBClient('localhost', 5555);

await client.connect();
await client.auth('your-token');

// Write
await client.write('sensor1', 42.5);
await client.write('sensor1', 99.9, 1717965210);

// JSON reads
const points = await client.readLast('sensor1', 100);
const ids = await client.ids();

// 🚀 Binary reads — 100x faster for bulk
const points = await client.readBinary('sensor1', 5000);
const multi = await client.multiReadBinary(['s1', 's2'], 5000);

// Batch write
await client.batchWrite([
    { key: 's1', value: 1.0 },
    { key: 's2', value: 2.0, timestamp: 1717965210 },
]);

// Subscribe
await client.subscribe('sensor1', (dp) => {
    console.log(dp.key, dp.value);
});
```

## API

| Method | Description |
|---|---|
| `new GTSDBClient(host, port)` | Create client |
| `connect()` | TCP connection |
| `close()` | Disconnect |
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

## Run Example

```bash
node example.js [host:port]
```

## Test

```bash
npm test
```

## License

MIT
