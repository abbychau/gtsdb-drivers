<?php

class TSDBClient
{
    private $socket;
    private $host;
    private $port;

    public function __construct($host, $port)
    {
        $this->host = $host;
        $this->port = $port;
    }

    public function connect()
    {
        $this->socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
        if ($this->socket === false) {
            throw new Exception("Socket creation failed: " . socket_strerror(socket_last_error()));
        }
        $result = socket_connect($this->socket, $this->host, $this->port);
        if ($result === false) {
            throw new Exception("Connection failed: " . socket_strerror(socket_last_error($this->socket)));
        }
    }

    public function close()
    {
        socket_close($this->socket);
    }

    private function writeData($key, $timestamp, $value)
    {
        $data = "{$key},{$timestamp},{$value}\n";
        socket_write($this->socket, $data, strlen($data));
    }

    private function readData($key, $startTime, $endTime, $downsampling)
    {
        $query = "{$key},{$startTime},{$endTime},{$downsampling}\n";
        socket_write($this->socket, $query, strlen($query));
        $response = socket_read($this->socket, 4096);
        return explode("|", trim($response));
    }

    public function recordMeasurement($sensorID, $value)
    {
        $timestamp = time();
        $this->writeData($sensorID, $timestamp, $value);
    }

    public function getLatestMeasurement($sensorID)
    {
        $endTime = time();
        $startTime = $endTime - 3600; // Look back 1 hour

        $data = $this->readData($sensorID, $startTime, $endTime, 0);
        if (empty($data)) {
            throw new Exception("No data found for sensor {$sensorID}");
        }

        $latestMeasurement = explode(',', end($data));
        return [
            'value' => floatval($latestMeasurement[2]),
            'timestamp' => new DateTime("@{$latestMeasurement[1]}")
        ];
    }

    public function getAverageMeasurement($sensorID, $durationSeconds)
    {
        $endTime = time();
        $startTime = $endTime - $durationSeconds;

        $data = $this->readData($sensorID, $startTime, $endTime, 0);
        if (empty($data)) {
            throw new Exception("No data found for sensor {$sensorID} in the specified time range");
        }

        $sum = 0;
        $count = 0;

        foreach ($data as $measurement) {
            $parts = explode(',', $measurement);
            if (count($parts) === 3) {
                $sum += floatval($parts[2]);
                $count++;
            }
        }

        if ($count === 0) {
            throw new Exception("No valid measurements found");
        }

        return $sum / $count;
    }

    public function getMeasurementHistory($sensorID, $startTime, $endTime, $intervalSeconds)
    {
        $downsamplingSeconds = max(1, $intervalSeconds);
        $startTimestamp = $startTime->getTimestamp();
        $endTimestamp = $endTime->getTimestamp();

        $data = $this->readData($sensorID, $startTimestamp, $endTimestamp, $downsamplingSeconds);

        $history = [];
        foreach ($data as $measurement) {
            $parts = explode(',', $measurement);
            if (count($parts) === 3) {
                $history[] = [
                    'timestamp' => new DateTime("@{$parts[1]}"),
                    'value' => floatval($parts[2])
                ];
            }
        }

        return $history;
    }

    public function example()
    {
        // Example usage
        try {
            $client = new TSDBClient('localhost', 8080);
            $client->connect();

            // Record a measurement
            $client->recordMeasurement('sensor1', 25.5);
            echo "Measurement recorded\n";

            // Get the latest measurement
            $latestMeasurement = $client->getLatestMeasurement('sensor1');
            echo "Latest measurement for sensor1: {$latestMeasurement['value']} at {$latestMeasurement['timestamp']->format('Y-m-d H:i:s')}\n";

            // Get the average measurement over the last hour
            $avgValue = $client->getAverageMeasurement('sensor1', 3600); // 1 hour in seconds
            echo "Average measurement for sensor1 over the last hour: {$avgValue}\n";

            // Get measurement history for the last 24 hours, with 1-hour intervals
            $endTime = new DateTime();
            $startTime = (new DateTime())->modify('-24 hours');
            $history = $client->getMeasurementHistory('sensor1', $startTime, $endTime, 3600); // 1 hour in seconds
            echo "Measurement history for sensor1:\n";
            foreach ($history as $entry) {
                echo "  {$entry['timestamp']->format('Y-m-d H:i:s')}: {$entry['value']}\n";
            }
        } catch (Exception $e) {
            echo "Error: " . $e->getMessage() . "\n";
        } finally {
            if (isset($client)) {
                $client->close();
            }
        }
    }
}
