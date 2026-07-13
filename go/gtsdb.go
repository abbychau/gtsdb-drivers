// Package gtsdb provides a Go client for GTSDB over TCP with JSON and binary protocols.
//
//	client, _ := gtsdb.Connect("localhost:5555")
//	client.Auth("your-token")
//	client.Write("sensor1", 42.5)
//	points, _ := client.ReadBinary("sensor1", 100)
package gtsdb

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
)

// DataPoint represents a single time-series data point.
type DataPoint struct {
	Key       string  `json:"key"`
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// Client is a GTSDB TCP client.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
}

// Connect establishes a TCP connection to a GTSDB server.
func Connect(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("gtsdb: dial %s: %w", addr, err)
	}
	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// send writes a JSON-encoded request and returns the raw JSON response line.
func (c *Client) send(req interface{}) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("gtsdb: marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("gtsdb: write: %w", err)
	}
	return c.reader.ReadBytes('\n')
}

// sendBinary sends a JSON request and reads a length-prefixed binary response frame.
func (c *Client) sendBinary(req interface{}) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("gtsdb: marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("gtsdb: write: %w", err)
	}
	return readBinaryFrame(c.reader)
}

// readBinaryFrame reads a 4-byte length-prefixed binary frame.
func readBinaryFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("gtsdb: read frame len: %w", err)
	}
	n := binary.BigEndian.Uint32(lenBuf[:])
	if n == 0 {
		return nil, nil
	}
	data := make([]byte, n)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("gtsdb: read frame data: %w", err)
	}
	return data, nil
}

// parseBinaryMulti parses a binary multi-data frame into a map of key → points.
func parseBinaryMulti(data []byte) (map[string][]DataPoint, error) {
	if len(data) < 4 {
		return nil, nil
	}
	result := make(map[string][]DataPoint)
	off := 0
	numKeys := binary.BigEndian.Uint32(data[off:])
	off += 4
	for i := uint32(0); i < numKeys; i++ {
		if off+2 > len(data) {
			break
		}
		keyLen := binary.BigEndian.Uint16(data[off:])
		off += 2
		if off+int(keyLen) > len(data) {
			break
		}
		key := string(data[off : off+int(keyLen)])
		off += int(keyLen)
		if off+4 > len(data) {
			break
		}
		pointCount := binary.BigEndian.Uint32(data[off:])
		off += 4
		pts := make([]DataPoint, pointCount)
		for j := uint32(0); j < pointCount; j++ {
			if off+16 > len(data) {
				break
			}
			pts[j] = DataPoint{
				Key:       key,
				Timestamp: int64(binary.BigEndian.Uint64(data[off:])),
				Value:     math.Float64frombits(binary.BigEndian.Uint64(data[off+8:])),
			}
			off += 16
		}
		result[key] = pts
	}
	return result, nil
}

// parseBinaryKey parses a single-key binary frame.
func parseBinaryKey(data []byte) ([]DataPoint, error) {
	parsed, err := parseBinaryMulti(data)
	if err != nil {
		return nil, err
	}
	for _, pts := range parsed {
		return pts, nil
	}
	return nil, nil
}

// Auth authenticates with a token.
func (c *Client) Auth(token string) error {
	resp, err := c.send(map[string]string{
		"operation": "auth",
		"key":       token,
	})
	if err != nil {
		return err
	}
	var r jsonResponse
	json.Unmarshal(resp, &r)
	if !r.Success {
		return fmt.Errorf("gtsdb: auth failed: %s", r.Message)
	}
	return nil
}

// Write sends a single data point. Timestamp is set server-side if 0.
func (c *Client) Write(key string, value float64) error {
	return c.WriteAt(key, value, 0)
}

// WriteAt sends a single data point with a specific timestamp.
func (c *Client) WriteAt(key string, value float64, timestamp int64) error {
	req := map[string]interface{}{
		"operation": "write",
		"key":       key,
		"write": map[string]interface{}{
			"value": value,
		},
	}
	if timestamp > 0 {
		req["write"].(map[string]interface{})["timestamp"] = timestamp
	}
	_, err := c.send(req)
	return err
}

// Read reads data points in a time range.
func (c *Client) Read(key string, startTs, endTs int64, downsample int) ([]DataPoint, error) {
	resp, err := c.send(map[string]interface{}{
		"operation": "read",
		"key":       key,
		"read": map[string]interface{}{
			"start_timestamp": startTs,
			"end_timestamp":   endTs,
			"downsampling":    downsample,
		},
	})
	if err != nil {
		return nil, err
	}
	var r struct {
		Data []DataPoint `json:"data"`
	}
	json.Unmarshal(resp, &r)
	return r.Data, nil
}

// ReadLast reads the last N data points for a key.
func (c *Client) ReadLast(key string, n int) ([]DataPoint, error) {
	resp, err := c.send(map[string]interface{}{
		"operation": "read",
		"key":       key,
		"read":      map[string]interface{}{"lastx": n},
	})
	if err != nil {
		return nil, err
	}
	var r struct {
		Data []DataPoint `json:"data"`
	}
	json.Unmarshal(resp, &r)
	return r.Data, nil
}

// ReadBinary reads the last N points using the fast binary protocol.
func (c *Client) ReadBinary(key string, n int) ([]DataPoint, error) {
	data, err := c.sendBinary(map[string]interface{}{
		"operation":       "read",
		"key":             key,
		"read":            map[string]interface{}{"lastx": n},
		"response_format": "binary",
	})
	if err != nil {
		return nil, err
	}
	return parseBinaryKey(data)
}

// ReadLastBinary is an alias for ReadBinary.
func (c *Client) ReadLastBinary(key string, n int) ([]DataPoint, error) {
	return c.ReadBinary(key, n)
}

// MultiRead reads from multiple keys in one request using JSON.
func (c *Client) MultiRead(keys []string, lastX int) (map[string][]DataPoint, error) {
	resp, err := c.send(map[string]interface{}{
		"operation": "multi-read",
		"keys":      keys,
		"read":      map[string]interface{}{"lastx": lastX},
	})
	if err != nil {
		return nil, err
	}
	var r struct {
		MultiData map[string][]DataPoint `json:"multi_data"`
	}
	json.Unmarshal(resp, &r)
	return r.MultiData, nil
}

// MultiReadBinary reads from multiple keys using the fast binary protocol.
func (c *Client) MultiReadBinary(keys []string, lastX int) (map[string][]DataPoint, error) {
	data, err := c.sendBinary(map[string]interface{}{
		"operation":       "multi-read",
		"keys":            keys,
		"read":            map[string]interface{}{"lastx": lastX},
		"response_format": "binary",
	})
	if err != nil {
		return nil, err
	}
	return parseBinaryMulti(data)
}

// IDs returns all known sensor keys.
func (c *Client) IDs() ([]string, error) {
	resp, err := c.send(map[string]string{"operation": "ids"})
	if err != nil {
		return nil, err
	}
	var r struct {
		Data []string `json:"data"`
	}
	json.Unmarshal(resp, &r)
	return r.Data, nil
}

// Subscribe subscribes to real-time updates for a key.
// The callback receives data points as they arrive.
func (c *Client) Subscribe(key string, cb func(DataPoint)) error {
	_, err := c.send(map[string]string{
		"operation": "subscribe",
		"key":       key,
	})
	if err != nil {
		return err
	}
	go func() {
		for {
			resp, err := c.reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var r struct {
				Success bool      `json:"success"`
				Data    DataPoint `json:"data"`
			}
			if json.Unmarshal(resp, &r) == nil && r.Success {
				cb(r.Data)
			}
		}
	}()
	return nil
}

// Unsubscribe stops receiving updates for a key.
func (c *Client) Unsubscribe(key string) error {
	_, err := c.send(map[string]string{
		"operation": "unsubscribe",
		"key":       key,
	})
	return err
}

// BatchWrite writes multiple data points in one request.
func (c *Client) BatchWrite(points []DataPoint) error {
	batch := make([]map[string]interface{}, len(points))
	for i, p := range points {
		batch[i] = map[string]interface{}{
			"key":       p.Key,
			"value":     p.Value,
			"timestamp": p.Timestamp,
		}
	}
	_, err := c.send(map[string]interface{}{
		"operation": "batch-write",
		"points":    batch,
	})
	return err
}

type jsonResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
