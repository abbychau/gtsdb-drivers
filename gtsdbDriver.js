const net = require('net');

class TSDBClient {
  constructor(host, port) {
    this.host = host;
    this.port = port;
    this.socket = new net.Socket();
    this.connected = false;
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.socket.connect(this.port, this.host, () => {
        this.connected = true;
        resolve();
      });
      this.socket.on('error', (err) => {
        reject(err);
      });
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

  writeData(key, timestamp, value) {
    return new Promise((resolve, reject) => {
      this.socket.write(`${key},${timestamp},${value}\n`, (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  readData(key, startTime, endTime, downsampling) {
    return new Promise((resolve, reject) => {
      this.socket.write(`${key},${startTime},${endTime},${downsampling}\n`, (err) => {
        if (err) reject(err);
      });

      this.socket.once('data', (data) => {
        resolve(data.toString().trim().split('|'));
      });
    });
  }

  async recordMeasurement(sensorID, value) {
    const timestamp = Math.floor(Date.now() / 1000);
    await this.writeData(sensorID, timestamp, value);
  }

  async getLatestMeasurement(sensorID) {
    const endTime = Math.floor(Date.now() / 1000);
    const startTime = endTime - 3600; // Look back 1 hour

    const data = await this.readData(sensorID, startTime, endTime, 0);
    if (data.length === 0) {
      throw new Error(`No data found for sensor ${sensorID}`);
    }

    const latestMeasurement = data[data.length - 1].split(',');
    return {
      value: parseFloat(latestMeasurement[2]),
      timestamp: new Date(parseInt(latestMeasurement[1]) * 1000)
    };
  }

  async getAverageMeasurement(sensorID, durationMs) {
    const endTime = Math.floor(Date.now() / 1000);
    const startTime = endTime - Math.floor(durationMs / 1000);

    const data = await this.readData(sensorID, startTime, endTime, 0);
    if (data.length === 0) {
      throw new Error(`No data found for sensor ${sensorID} in the specified time range`);
    }

    let sum = 0;
    let count = 0;

    for (const measurement of data) {
      const parts = measurement.split(',');
      if (parts.length === 3) {
        sum += parseFloat(parts[2]);
        count++;
      }
    }

    if (count === 0) {
      throw new Error('No valid measurements found');
    }

    return sum / count;
  }

  async getMeasurementHistory(sensorID, startTime, endTime, intervalMs) {
    const downsamplingSeconds = Math.max(1, Math.floor(intervalMs / 1000));
    const startTimestamp = Math.floor(startTime.getTime() / 1000);
    const endTimestamp = Math.floor(endTime.getTime() / 1000);

    const data = await this.readData(sensorID, startTimestamp, endTimestamp, downsamplingSeconds);

    return data.map(measurement => {
      const parts = measurement.split(',');
      return {
        timestamp: new Date(parseInt(parts[1]) * 1000),
        value: parseFloat(parts[2])
      };
    });
  }

  subscribe(key) {
    return new Promise((resolve, reject) => {
      this.socket.write(`subscribe,${key}\n`, (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  unsubscribe(key) {
    return new Promise((resolve, reject) => {
      this.socket.write(`unsubscribe,${key}\n`, (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  onSubscriptionData(callback) {
    this.socket.on('data', (data) => {
      const messages = data.toString().trim().split('\n');
      messages.forEach(message => {
        const [key, timestamp, value] = message.split(',');
        callback({
          key,
          timestamp: new Date(parseInt(timestamp) * 1000),
          value: parseFloat(value)
        });
      });
    });
  }
}

// Example usage
async function example() {
  const client = new TSDBClient('localhost', 8080);

  try {
    await client.connect();

    // Subscribe to updates for sensor1
    await client.subscribe('sensor1');
    console.log('Subscribed to sensor1');

    // Set up a callback for subscription data
    client.onSubscriptionData((data) => {
      console.log(`Received update for ${data.key}: ${data.value} at ${data.timestamp}`);
    });

    // Wait for some time to receive updates
    await new Promise(resolve => setTimeout(resolve, 30000)); // Wait for 30 seconds

    // Unsubscribe from updates for sensor1
    await client.unsubscribe('sensor1');
    console.log('Unsubscribed from sensor1');

    // Record a measurement
    await client.recordMeasurement('sensor1', 25.5);
    console.log('Measurement recorded');

    // Get the latest measurement
    const latestMeasurement = await client.getLatestMeasurement('sensor1');
    console.log(`Latest measurement for sensor1: ${latestMeasurement.value} at ${latestMeasurement.timestamp}`);

    // Get the average measurement over the last hour
    const avgValue = await client.getAverageMeasurement('sensor1', 3600000); // 1 hour in milliseconds
    console.log(`Average measurement for sensor1 over the last hour: ${avgValue}`);

    // Get measurement history for the last 24 hours, with 1-hour intervals
    const endTime = new Date();
    const startTime = new Date(endTime.getTime() - 24 * 60 * 60 * 1000);
    const history = await client.getMeasurementHistory('sensor1', startTime, endTime, 3600000); // 1 hour in milliseconds
    console.log('Measurement history for sensor1:');
    history.forEach(entry => {
      console.log(`  ${entry.timestamp}: ${entry.value}`);
    });

  } catch (error) {
    console.error('Error:', error);
  } finally {
    await client.close();
  }
}
