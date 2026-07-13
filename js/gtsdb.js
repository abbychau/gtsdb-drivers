/**
 * GTSDB JavaScript/Node.js client over TCP with JSON and binary protocols.
 *
 *   const { GTSDBClient } = require('gtsdb-drivers');
 *   const client = new GTSDBClient('localhost', 5555);
 *   await client.connect();
 *   await client.auth('your-token');
 *   await client.write('sensor1', 42.5);
 *   const points = await client.readBinary('sensor1', 100);
 */

const net = require('net');

class GTSDBClient {
  constructor(host, port) {
    this.host = host;
    this.port = port;
    this.socket = new net.Socket();
    this.buffer = '';
    this.pendingResolve = null;
    this.pendingReject = null;
    this.subCallbacks = new Map();
    this.connected = false;
    this._dataHandler = null;
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.socket.connect(this.port, this.host, () => {
        this.connected = true;
        resolve();
      });
      this.socket.on('error', reject);
      this.socket.on('data', (chunk) => this._onData(chunk));
    });
  }

  close() {
    return new Promise((resolve) => {
      this.socket.end(() => {
        this.connected = false;
        resolve();
      });
    });
  }

  /** Send a JSON request and return the raw response line. */
  _send(req) {
    return new Promise((resolve, reject) => {
      this.pendingResolve = (data) => {
        try { resolve(JSON.parse(data)); } catch (e) { resolve(data); }
      };
      this.pendingReject = reject;
      this.socket.write(JSON.stringify(req) + '\n');
    });
  }

  /** Send a JSON request and read a length-prefixed binary response frame. */
  _sendBinary(req) {
    return new Promise((resolve, reject) => {
      this._binaryResolve = resolve;
      this._binaryReject = reject;
      this._binaryState = 0; // 0: reading length, 1: reading data
      this._binaryLen = 0;
      this._binaryBuf = null;
      this._binaryOff = 0;
      this.socket.write(JSON.stringify(req) + '\n');
    });
  }

  _onData(chunk) {
    // If we're in binary read mode, handle binary framing
    if (this._binaryResolve) {
      this._handleBinaryChunk(chunk);
      return;
    }

    // JSON line mode
    this.buffer += chunk.toString();
    const lines = this.buffer.split('\n');
    this.buffer = lines.pop(); // keep incomplete line

    for (const line of lines) {
      if (!line.trim()) continue;
      try {
        const msg = JSON.parse(line);
        // Check if this is a subscription update
        if (msg.data && msg.data.key && this.subCallbacks.has(msg.data.key)) {
          for (const cb of this.subCallbacks.get(msg.data.key)) {
            cb(msg.data);
          }
          continue;
        }
        // Otherwise, resolve pending request
        if (this.pendingResolve) {
          this.pendingResolve(msg);
          this.pendingResolve = null;
          this.pendingReject = null;
        }
      } catch (e) {
        if (this.pendingReject) {
          this.pendingReject(e);
          this.pendingResolve = null;
          this.pendingReject = null;
        }
      }
    }
  }

  /** Handle binary protocol framing. */
  _handleBinaryChunk(chunk) {
    let off = 0;
    const data = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);

    while (off < data.length) {
      if (this._binaryState === 0) {
        // Reading 4-byte length prefix
        const need = 4 - this._binaryOff;
        const avail = data.length - off;
        const take = Math.min(need, avail);

        if (this._binaryLenBuf === undefined) this._binaryLenBuf = Buffer.alloc(4);
        data.copy(this._binaryLenBuf, this._binaryOff, off, off + take);
        this._binaryOff += take;
        off += take;

        if (this._binaryOff === 4) {
          this._binaryLen = this._binaryLenBuf.readUInt32BE(0);
          this._binaryState = 1;
          this._binaryOff = 0;
          this._binaryBuf = Buffer.alloc(this._binaryLen);
          this._binaryLenBuf = undefined;
          if (this._binaryLen === 0) {
            this._finishBinary();
            return;
          }
        }
      } else {
        // Reading frame data
        const need = this._binaryLen - this._binaryOff;
        const avail = data.length - off;
        const take = Math.min(need, avail);

        data.copy(this._binaryBuf, this._binaryOff, off, off + take);
        this._binaryOff += take;
        off += take;

        if (this._binaryOff === this._binaryLen) {
          this._finishBinary();
          return;
        }
      }
    }
  }

  _finishBinary() {
    const resolve = this._binaryResolve;
    const buf = this._binaryBuf;
    this._binaryResolve = null;
    this._binaryReject = null;
    this._binaryBuf = null;
    resolve(this._parseBinaryMulti(buf));
  }

  /** Parse a binary frame into a map of key → points. */
  _parseBinaryMulti(buf) {
    if (buf.length < 4) return {};
    let off = 0;
    const result = {};
    const numKeys = buf.readUInt32BE(off);
    off += 4;

    for (let i = 0; i < numKeys; i++) {
      if (off + 2 > buf.length) break;
      const keyLen = buf.readUInt16BE(off);
      off += 2;
      if (off + keyLen > buf.length) break;
      const key = buf.toString('utf8', off, off + keyLen);
      off += keyLen;
      if (off + 4 > buf.length) break;
      const pointCount = buf.readUInt32BE(off);
      off += 4;

      const pts = [];
      for (let j = 0; j < pointCount; j++) {
        if (off + 16 > buf.length) break;
        const ts = Number(buf.readBigInt64BE(off));
        const val = buf.readDoubleBE(off + 8);
        off += 16;
        pts.push({ key, timestamp: ts, value: val });
      }
      result[key] = pts;
    }
    return result;
  }

  // ── Public API ──────────────────────────────────

  /** Authenticate with a token. */
  async auth(token) {
    const r = await this._send({ operation: 'auth', key: token });
    if (!r.success) throw new Error('GTSDB auth failed: ' + r.message);
  }

  /** Write a single data point. */
  async write(key, value, timestamp = 0) {
    const req = { operation: 'write', key, write: { value } };
    if (timestamp > 0) req.write.timestamp = timestamp;
    return this._send(req);
  }

  /** Read data points in a time range (JSON). */
  async read(key, startTs, endTs, downsample = 0) {
    const r = await this._send({
      operation: 'read', key,
      read: { start_timestamp: startTs, end_timestamp: endTs, downsampling: downsample },
    });
    return r.data || [];
  }

  /** Read the last N data points (JSON). */
  async readLast(key, n) {
    const r = await this._send({ operation: 'read', key, read: { lastx: n } });
    return r.data || [];
  }

  /** Read the last N data points using the fast binary protocol. */
  async readBinary(key, n) {
    const multi = await this._sendBinary({
      operation: 'read', key,
      read: { lastx: n },
      response_format: 'binary',
    });
    return multi[key] || [];
  }

  /** Read from multiple keys using binary protocol. */
  async multiReadBinary(keys, lastX) {
    return this._sendBinary({
      operation: 'multi-read', keys,
      read: { lastx: lastX },
      response_format: 'binary',
    });
  }

  /** Get all known sensor IDs. */
  async ids() {
    const r = await this._send({ operation: 'ids' });
    return r.data || [];
  }

  /** Subscribe to real-time updates for a key. */
  async subscribe(key, callback) {
    if (!this.subCallbacks.has(key)) this.subCallbacks.set(key, []);
    this.subCallbacks.get(key).push(callback);
    return this._send({ operation: 'subscribe', key });
  }

  /** Unsubscribe from updates for a key. */
  async unsubscribe(key) {
    this.subCallbacks.delete(key);
    return this._send({ operation: 'unsubscribe', key });
  }

  /** Batch write multiple data points. */
  async batchWrite(points) {
    return this._send({
      operation: 'batch-write',
      points: points.map(p => ({ key: p.key, value: p.value, timestamp: p.timestamp || 0 })),
    });
  }
}

module.exports = { GTSDBClient };
