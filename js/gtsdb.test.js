const { GTSDBClient } = require('./gtsdb.js');
const assert = require('assert');

// Helper: build a binary frame identical to Go's binary.go
function encodeBinaryFrame(data) {
  const parts = [];
  const keys = Object.keys(data);

  // Key count
  const keyCountBuf = Buffer.alloc(4);
  keyCountBuf.writeUInt32BE(keys.length, 0);
  parts.push(keyCountBuf);

  for (const key of keys) {
    const pts = data[key];
    // Key length + key
    const keyLenBuf = Buffer.alloc(2);
    keyLenBuf.writeUInt16BE(Buffer.byteLength(key), 0);
    parts.push(keyLenBuf);
    parts.push(Buffer.from(key, 'utf8'));

    // Point count
    const ptCountBuf = Buffer.alloc(4);
    ptCountBuf.writeUInt32BE(pts.length, 0);
    parts.push(ptCountBuf);

    for (const pt of pts) {
      const tsBuf = Buffer.alloc(8);
      tsBuf.writeBigInt64BE(BigInt(pt.timestamp), 0);
      parts.push(tsBuf);

      const valBuf = Buffer.alloc(8);
      valBuf.writeDoubleBE(pt.value, 0);
      parts.push(valBuf);
    }
  }

  const dataBuf = Buffer.concat(parts);
  const lenBuf = Buffer.alloc(4);
  lenBuf.writeUInt32BE(dataBuf.length, 0);
  return Buffer.concat([lenBuf, dataBuf]);
}

describe('GTSDBClient', () => {
  describe('_parseBinaryMulti', () => {
    const client = new GTSDBClient('localhost', 5555);

    it('parses multiple keys with multiple points', () => {
      const input = {
        sensor1: [
          { key: 'sensor1', timestamp: 1700000000, value: 1.5 },
          { key: 'sensor1', timestamp: 1700000001, value: 3.0 },
        ],
        sensor2: [
          { key: 'sensor2', timestamp: 1700000100, value: -42.75 },
        ],
      };
      const frame = encodeBinaryFrame(input);
      const result = client._parseBinaryMulti(frame.slice(4)); // skip len prefix
      assert.strictEqual(Object.keys(result).length, 2);
      assert.strictEqual(result.sensor1.length, 2);
      assert.strictEqual(result.sensor1[0].timestamp, 1700000000);
      assert.strictEqual(result.sensor1[0].value, 1.5);
      assert.strictEqual(result.sensor1[1].timestamp, 1700000001);
      assert.strictEqual(result.sensor1[1].value, 3.0);
      assert.strictEqual(result.sensor2.length, 1);
      assert.strictEqual(result.sensor2[0].timestamp, 1700000100);
      assert.strictEqual(result.sensor2[0].value, -42.75);
    });

    it('parses empty data', () => {
      const frame = encodeBinaryFrame({});
      const result = client._parseBinaryMulti(frame.slice(4));
      assert.strictEqual(Object.keys(result).length, 0);
    });

    it('handles special float values', () => {
      const input = {
        k: [
          { key: 'k', timestamp: 0, value: 0 },
          { key: 'k', timestamp: -1, value: Infinity },
          { key: 'k', timestamp: Number.MAX_SAFE_INTEGER, value: NaN },
        ],
      };
      const frame = encodeBinaryFrame(input);
      const result = client._parseBinaryMulti(frame.slice(4));
      const pts = result.k;
      assert.strictEqual(pts.length, 3);
      assert.strictEqual(pts[0].timestamp, 0);
      assert.strictEqual(pts[0].value, 0);
      assert.strictEqual(pts[1].timestamp, -1);
      assert(pts[1].value === Infinity);
      assert(Number.isNaN(pts[2].value));
    });

    it('returns empty object for short buffers', () => {
      assert.deepStrictEqual(client._parseBinaryMulti(Buffer.alloc(0)), {});
    });
  });

  describe('constructor', () => {
    it('creates client with default state', () => {
      const c = new GTSDBClient('localhost', 5555);
      assert.strictEqual(c.host, 'localhost');
      assert.strictEqual(c.port, 5555);
      assert.strictEqual(c.connected, false);
      assert.strictEqual(c.buffer, '');
    });
  });
});
