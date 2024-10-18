library(lubridate)

TSDBClient <- R6::R6Class("TSDBClient",
  public = list(
    host = NULL,
    port = NULL,
    conn = NULL,

    initialize = function(host, port) {
      self$host <- host
      self$port <- port
    },

    connect = function() {
      self$conn <- socketConnection(host = self$host, port = self$port, open = "r+")
    },

    close = function() {
      close(self$conn)
    },

    writeData = function(key, timestamp, value) {
      writeLines(paste(key, timestamp, value, sep = ","), self$conn)
    },

    readData = function(key, startTime, endTime, downsampling) {
      query <- paste(key, startTime, endTime, downsampling, sep = ",")
      writeLines(query, self$conn)
      response <- readLines(self$conn, n = 1)
      strsplit(response, "\\|")[[1]]
    },

    recordMeasurement = function(sensorID, value) {
      timestamp <- as.integer(Sys.time())
      self$writeData(sensorID, timestamp, value)
    },

    getLatestMeasurement = function(sensorID) {
      endTime <- as.integer(Sys.time())
      startTime <- endTime - 3600 # Look back 1 hour

      data <- self$readData(sensorID, startTime, endTime, 0)
      if (length(data) == 0) {
        stop(paste("No data found for sensor", sensorID))
      }

      latestMeasurement <- strsplit(data[length(data)], ",")[[1]]
      list(
        value = as.numeric(latestMeasurement[3]),
        timestamp = as_datetime(as.numeric(latestMeasurement[2]))
      )
    },

    getAverageMeasurement = function(sensorID, durationSeconds) {
      endTime <- as.integer(Sys.time())
      startTime <- endTime - durationSeconds

      data <- self$readData(sensorID, startTime, endTime, 0)
      if (length(data) == 0) {
        stop(paste("No data found for sensor", sensorID, "in the specified time range"))
      }

      values <- sapply(strsplit(data, ","), function(x) as.numeric(x[3]))
      mean(values, na.rm = TRUE)
    },

    getMeasurementHistory = function(sensorID, startTime, endTime, intervalSeconds) {
      downsamplingSeconds <- max(1, intervalSeconds)
      startTimestamp <- as.integer(startTime)
      endTimestamp <- as.integer(endTime)

      data <- self$readData(sensorID, startTimestamp, endTimestamp, downsamplingSeconds)

      history <- lapply(strsplit(data, ","), function(x) {
        list(
          timestamp = as_datetime(as.numeric(x[2])),
          value = as.numeric(x[3])
        )
      })

      do.call(rbind, lapply(history, as.data.frame))
    }
  )
)

example = function(){
  # Example usage
  tryCatch({
    client <- TSDBClient$new("localhost", 8080)
    client$connect()

    # Record a measurement
    client$recordMeasurement("sensor1", 25.5)
    cat("Measurement recorded\n")

    # Get the latest measurement
    latestMeasurement <- client$getLatestMeasurement("sensor1")
    cat(sprintf("Latest measurement for sensor1: %.2f at %s\n", 
                latestMeasurement$value, 
                format(latestMeasurement$timestamp, "%Y-%m-%d %H:%M:%S")))

    # Get the average measurement over the last hour
    avgValue <- client$getAverageMeasurement("sensor1", 3600) # 1 hour in seconds
    cat(sprintf("Average measurement for sensor1 over the last hour: %.2f\n", avgValue))

    # Get measurement history for the last 24 hours, with 1-hour intervals
    endTime <- Sys.time()
    startTime <- endTime - hours(24)
    history <- client$getMeasurementHistory("sensor1", startTime, endTime, 3600) # 1 hour in seconds
    cat("Measurement history for sensor1:\n")
    print(history)

    }, error = function(e) {
    cat("Error:", conditionMessage(e), "\n")
    }, finally = {
    if (exists("client")) {
        client$close()
    }
  })
}