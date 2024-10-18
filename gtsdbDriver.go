package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// TSDBClient struct remains the same as in the previous example
type TSDBClient struct {
	conn net.Conn
}

// NewTSDBClient creates a new TSDB client
func NewTSDBClient(address string) (*TSDBClient, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &TSDBClient{conn: conn}, nil
}

// Close closes the connection to the TSDB
func (c *TSDBClient) Close() error {
	return c.conn.Close()
}

// WriteData writes a single data point to the TSDB
func (c *TSDBClient) WriteData(key string, timestamp int64, value float64) error {
	_, err := fmt.Fprintf(c.conn, "%s,%d,%.2f\n", key, timestamp, value)
	return err
}

// ReadData reads data from the TSDB for a given key, time range, and downsampling
func (c *TSDBClient) ReadData(key string, startTime, endTime int64, downsampling int) ([]string, error) {
	_, err := fmt.Fprintf(c.conn, "%s,%d,%d,%d\n", key, startTime, endTime, downsampling)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(c.conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(response), "|"), nil
}

// Subscribe subscribes to updates for a given key
func (c *TSDBClient) Subscribe(key string) error {
	_, err := fmt.Fprintf(c.conn, "subscribe,%s\n", key)
	return err
}

// Unsubscribe unsubscribes from updates for a given key
func (c *TSDBClient) Unsubscribe(key string) error {
	_, err := fmt.Fprintf(c.conn, "unsubscribe,%s\n", key)
	return err
}

// RecordMeasurement records a single measurement for a given sensor
func (c *TSDBClient) RecordMeasurement(sensorID string, value float64) error {
	timestamp := time.Now().Unix()
	return c.WriteData(sensorID, timestamp, value)
}

// GetLatestMeasurement retrieves the most recent measurement for a given sensor
func (c *TSDBClient) GetLatestMeasurement(sensorID string) (float64, time.Time, error) {
	endTime := time.Now().Unix()
	startTime := endTime - 3600 // Look back 1 hour to find the latest measurement

	data, err := c.ReadData(sensorID, startTime, endTime, 0)
	if err != nil {
		return 0, time.Time{}, err
	}

	if len(data) == 0 {
		return 0, time.Time{}, fmt.Errorf("no data found for sensor %s", sensorID)
	}

	// Parse the most recent measurement
	parts := strings.Split(data[len(data)-1], ",")
	if len(parts) != 3 {
		return 0, time.Time{}, fmt.Errorf("invalid data format")
	}

	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, time.Time{}, err
	}

	value, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, time.Time{}, err
	}

	return value, time.Unix(timestamp, 0), nil
}

// GetAverageMeasurement calculates the average measurement over a specified time period
func (c *TSDBClient) GetAverageMeasurement(sensorID string, duration time.Duration) (float64, error) {
	endTime := time.Now().Unix()
	startTime := endTime - int64(duration.Seconds())

	data, err := c.ReadData(sensorID, startTime, endTime, 0)
	if err != nil {
		return 0, err
	}

	if len(data) == 0 {
		return 0, fmt.Errorf("no data found for sensor %s in the specified time range", sensorID)
	}

	var sum float64
	var count int

	for _, measurement := range data {
		parts := strings.Split(measurement, ",")
		if len(parts) != 3 {
			continue
		}

		value, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			continue
		}

		sum += value
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no valid measurements found")
	}

	return sum / float64(count), nil
}

// GetMeasurementHistory retrieves the measurement history for a given sensor and time range
func (c *TSDBClient) GetMeasurementHistory(sensorID string, startTime, endTime time.Time, interval time.Duration) ([]struct {
	Timestamp time.Time
	Value     float64
}, error) {
	downsamplingSeconds := int(interval.Seconds())
	if downsamplingSeconds < 1 {
		downsamplingSeconds = 1
	}

	data, err := c.ReadData(sensorID, startTime.Unix(), endTime.Unix(), downsamplingSeconds)
	if err != nil {
		return nil, err
	}

	var history []struct {
		Timestamp time.Time
		Value     float64
	}

	for _, measurement := range data {
		parts := strings.Split(measurement, ",")
		if len(parts) != 3 {
			continue
		}

		timestamp, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		value, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			continue
		}

		history = append(history, struct {
			Timestamp time.Time
			Value     float64
		}{
			Timestamp: time.Unix(timestamp, 0),
			Value:     value,
		})
	}

	return history, nil
}

// Example usage
func example() {
	client, err := NewTSDBClient("localhost:8080")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Record a measurement
	err = client.RecordMeasurement("sensor1", 25.5)
	if err != nil {
		panic(err)
	}

	// Get the latest measurement
	value, timestamp, err := client.GetLatestMeasurement("sensor1")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Latest measurement for sensor1: %.2f at %s\n", value, timestamp)

	// Get the average measurement over the last hour
	avgValue, err := client.GetAverageMeasurement("sensor1", time.Hour)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Average measurement for sensor1 over the last hour: %.2f\n", avgValue)

	// Get measurement history for the last 24 hours, with 1-hour intervals
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	history, err := client.GetMeasurementHistory("sensor1", startTime, endTime, time.Hour)
	if err != nil {
		panic(err)
	}
	fmt.Println("Measurement history for sensor1:")
	for _, entry := range history {
		fmt.Printf("  %s: %.2f\n", entry.Timestamp, entry.Value)
	}
}
