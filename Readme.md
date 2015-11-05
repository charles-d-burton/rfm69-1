

RFM69 for Go
============

A Go port of the Arduino library (https://github.com/LowPowerLab/RFM69) by Felix Rusu, LowPowerLab.com

The Go port was done by Friedl Ulrich (https://github.com/fulr/rfm69)

I forked the library to turn it into a more generic library with an event
handler for all packets received by the RFM69 module

> Warning: This library uses a non-standard bitrate of 19.2 KBPS
>
> You may revert this in device.go - see commit [d093a65](https://github.com/chbmuc/rfm69/commit/d093a65137f74539dd07e5fc7c16b79fa1b1482a)

### Hardware

Here is how to wire the RFM69 module to the GPIO pins of your Pi2:

![Wiring diagram](PI2-RFM69W.png)

### Usage example
```go
package main

import (
        "github.com/chbmuc/rfm69"
        "log"
)

const (
        encryptionKey = "abcdefghijklmnop"
        nodeID        = 1
        networkID     = 100
        isRfm69Hw     = false
)

var rfm *rfm69.Router

func logData(data rfm69.Data) {
        log.Printf("message received from %d: %v\n", data.FromAddress, data.Data)
}

func main() {
        var err error

        rfm, err = rfm69.Init(encryptionKey, nodeID, networkID, isRfm69Hw)
        if err != nil {
                log.Fatal(err)
        }

        defer rfm.Close()

        // Handle message received from node 10 with function logData
        rfm.Handle(10, logData)

        rfm.Run()
}
```

### License

GPL v3 http://opensource.org/licenses/GPL-3.0
