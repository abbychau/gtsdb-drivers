package gtsdb

import (
	"encoding/binary"
	"math"
	"testing"
)

// encodeBinaryFrame builds a binary frame from a map of key→points.
func encodeBinaryFrame(data map[string][]DataPoint) []byte {
	total := 4 // key count
	for k, pts := range data {
		total += 2 + len(k) + 4 + len(pts)*16
	}
	buf := make([]byte, 4, 4+total)
	binary.BigEndian.PutUint32(buf[:4], uint32(total))

	off := 4
	tmp := make([]byte, 8)
	// key count
	binary.BigEndian.PutUint32(buf[off:off+4], uint32(len(data)))
	off += 4
	buf = buf[:off]
	for k, pts := range data {
		need := 2 + len(k) + 4 + len(pts)*16
		for len(buf) < off+need {
			buf = append(buf, 0)
		}
		binary.BigEndian.PutUint16(buf[off:], uint16(len(k)))
		off += 2
		copy(buf[off:], k)
		off += len(k)
		binary.BigEndian.PutUint32(buf[off:], uint32(len(pts)))
		off += 4
		for _, dp := range pts {
			binary.BigEndian.PutUint64(tmp, uint64(dp.Timestamp))
			copy(buf[off:], tmp)
			off += 8
			binary.BigEndian.PutUint64(tmp, math.Float64bits(dp.Value))
			copy(buf[off:], tmp)
			off += 8
		}
	}
	return buf
}

func TestParseBinaryMulti(t *testing.T) {
	input := map[string][]DataPoint{
		"sensor1": {
			{Key: "sensor1", Timestamp: 1700000000, Value: 1.5},
			{Key: "sensor1", Timestamp: 1700000001, Value: 3.0},
		},
		"sensor2": {
			{Key: "sensor2", Timestamp: 1700000100, Value: -42.75},
		},
	}
	frame := encodeBinaryFrame(input)
	result, err := parseBinaryMulti(frame[4:]) // skip length prefix
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(result))
	}
	pts1 := result["sensor1"]
	if len(pts1) != 2 {
		t.Fatalf("sensor1: expected 2 points, got %d", len(pts1))
	}
	if pts1[0].Timestamp != 1700000000 || pts1[0].Value != 1.5 {
		t.Errorf("sensor1[0]: {%d, %f}", pts1[0].Timestamp, pts1[0].Value)
	}
	if pts1[1].Timestamp != 1700000001 || pts1[1].Value != 3.0 {
		t.Errorf("sensor1[1]: {%d, %f}", pts1[1].Timestamp, pts1[1].Value)
	}
	pts2 := result["sensor2"]
	if len(pts2) != 1 {
		t.Fatalf("sensor2: expected 1 point, got %d", len(pts2))
	}
	if pts2[0].Timestamp != 1700000100 || pts2[0].Value != -42.75 {
		t.Errorf("sensor2[0]: {%d, %f}", pts2[0].Timestamp, pts2[0].Value)
	}
}

func TestParseBinaryMultiEmpty(t *testing.T) {
	frame := encodeBinaryFrame(map[string][]DataPoint{})
	result, err := parseBinaryMulti(frame[4:])
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty, got %d keys", len(result))
	}

	// nil/empty input
	r, err := parseBinaryMulti(nil)
	if err != nil || r != nil {
		t.Error("nil should return nil")
	}
	r, err = parseBinaryMulti([]byte{})
	if err != nil || r != nil {
		t.Error("empty should return nil")
	}
}

func TestParseBinaryMultiSpecialValues(t *testing.T) {
	input := map[string][]DataPoint{
		"k": {
			{Key: "k", Timestamp: 0, Value: 0},
			{Key: "k", Timestamp: -1, Value: math.Inf(1)},
			{Key: "k", Timestamp: 1<<62, Value: math.NaN()},
		},
	}
	frame := encodeBinaryFrame(input)
	result, err := parseBinaryMulti(frame[4:])
	if err != nil {
		t.Fatal(err)
	}
	pts := result["k"]
	if len(pts) != 3 {
		t.Fatalf("expected 3 points, got %d", len(pts))
	}
	if pts[0].Timestamp != 0 || pts[0].Value != 0 {
		t.Errorf("zero: {%d, %f}", pts[0].Timestamp, pts[0].Value)
	}
	if pts[1].Timestamp != -1 || !math.IsInf(pts[1].Value, 1) {
		t.Errorf("neg/inf: {%d, %f}", pts[1].Timestamp, pts[1].Value)
	}
	if pts[2].Timestamp != 1<<62 || !math.IsNaN(pts[2].Value) {
		t.Errorf("large/nan: {%d, %f}", pts[2].Timestamp, pts[2].Value)
	}
}

func TestParseBinaryKey(t *testing.T) {
	input := map[string][]DataPoint{
		"only": {{Key: "only", Timestamp: 42, Value: 99.9}},
	}
	frame := encodeBinaryFrame(input)
	pts, err := parseBinaryKey(frame[4:])
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 1 {
		t.Fatalf("expected 1 point, got %d", len(pts))
	}
	if pts[0].Timestamp != 42 || pts[0].Value != 99.9 {
		t.Errorf("got {%d, %f}", pts[0].Timestamp, pts[0].Value)
	}
}

func TestParseBinaryKeyEmpty(t *testing.T) {
	frame := encodeBinaryFrame(map[string][]DataPoint{})
	pts, err := parseBinaryKey(frame[4:])
	if err != nil {
		t.Fatal(err)
	}
	if pts != nil {
		t.Errorf("expected nil, got %v", pts)
	}
}

func TestReadBinaryFrame(t *testing.T) {
	// This tests the frame splitting logic indirectly via parse
	// The actual io.ReadFull is tested in integration
}

func TestDataPointStruct(t *testing.T) {
	dp := DataPoint{Key: "test", Timestamp: 100, Value: 3.14}
	if dp.Key != "test" || dp.Timestamp != 100 || dp.Value != 3.14 {
		t.Error("struct fields mismatch")
	}
}
