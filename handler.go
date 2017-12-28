package rfm69

import (
	"errors"
	"log"
	"time"

	"github.com/davecheney/gpio"
)

// Router manages sending and receiving of commands / data
type Router struct {
	handlers  map[byte]Handle
	responses map[byte]chan Data

	rfm *Device

	tx chan *Data
}

// Handle is a function that can be registered to handle an rfm69 message
type Handle func(Data)

// Handle registers a generic new event handler for a specific node
func (r *Router) Handle(node byte, handle Handle) {
	if r.handlers == nil {
		r.handlers = make(map[byte]Handle)
	}

	r.handlers[node] = handle
}

// Init initializes the connection to the rfm69 module
func Init(encryptionKey string, nodeID byte, networkID byte, isRfm69Hw bool) (*Router, error) {
	var err error

	r := new(Router)

	r.rfm, err = NewDevice(nodeID, networkID, isRfm69Hw)

	if err != nil {
		return nil, err
	}

	err = r.rfm.Encrypt([]byte(encryptionKey))
	if err != nil {
		return nil, err
	}

	r.tx = make(chan *Data, 5)

	return r, nil
}

// Run the rfm98 event watcher
func (r *Router) Run() {
	irq := make(chan int)
	r.rfm.gpio.BeginWatch(gpio.EdgeRising, func() {
		irq <- 1
	})
	defer r.rfm.gpio.EndWatch()

	err := r.rfm.SetMode(RF_OPMODE_RECEIVER)
	if err != nil {
		log.Print(err)
		return
	}
	defer r.rfm.SetMode(RF_OPMODE_STANDBY)

	for {
		select {
		case dataToTransmit := <-r.tx:
			// TODO: can send?
			r.rfm.readWriteReg(REG_PACKETCONFIG2, 0xFB, RF_PACKET2_RXRESTART) // avoid RX deadlocks
			err = r.rfm.SetModeAndWait(RF_OPMODE_STANDBY)
			if err != nil {
				log.Fatal(err)
			}
			err = r.rfm.writeReg(REG_DIOMAPPING1, RF_DIOMAPPING1_DIO0_00)
			if err != nil {
				log.Fatal(err)
			}
			err = r.rfm.writeFifo(dataToTransmit)
			if err != nil {
				log.Fatal(err)
			}
			err = r.rfm.SetMode(RF_OPMODE_TRANSMITTER)
			if err != nil {
				log.Fatal(err)
			}

			<-irq

			err = r.rfm.SetModeAndWait(RF_OPMODE_STANDBY)
			if err != nil {
				log.Fatal(err)
			}
			err = r.rfm.writeReg(REG_DIOMAPPING1, RF_DIOMAPPING1_DIO0_01)
			if err != nil {
				log.Fatal(err)
			}
			err = r.rfm.SetMode(RF_OPMODE_RECEIVER)
			if err != nil {
				log.Fatal(err)
			}
		case <-irq:
			if r.rfm.mode != RF_OPMODE_RECEIVER {
				continue
			}
			flags, err := r.rfm.readReg(REG_IRQFLAGS2)
			if err != nil {
				return
			}
			if flags&RF_IRQFLAGS2_PAYLOADREADY == 0 {
				continue
			}
			data, err := r.rfm.readFifo()
			if err != nil {
				log.Print(err)
				return
			}
			err = r.rfm.SetMode(RF_OPMODE_RECEIVER)
			if err != nil {
				log.Fatal(err)
			}
			if data.ToAddress != r.rfm.address {
				break
			}
			if data.ToAddress != 255 && data.RequestAck {
				r.tx <- data.ToAck()
			}

			// check if
			// 1. we are waiting for a response from this node
			// 2. we have a handler for this node otherwise

			if c, ok := r.responses[data.FromAddress]; ok {
				c <- data
			} else if h, ok := r.handlers[data.FromAddress]; ok {
				h(data)
			}
		}
	}
}

// Send data to a node
func (r *Router) Send(nodeID byte, payload []byte) error {
	_, err := r.request(nodeID, payload, false, 0, 0, false, 0)
	return err
}

// SendWithAck sends data to a node with ack
func (r *Router) SendWithAck(nodeID byte, payload []byte) error {
	_, err := r.request(nodeID, payload, true, 3, 40, false, 0)
	return err
}

// Get data from a node (send request with ack and wait for response)
func (r *Router) Get(nodeID byte, payload []byte) (Data, error) {
	return r.request(nodeID, payload, true, 3, 40, true, 3000)
}

// Internal function to send data and handle responses
// acktime and datatime are in milliseconds
func (r *Router) request(nodeID byte, payload []byte, ack bool, retries int, acktime uint16, getdata bool, datatime uint16) (Data, error) {

	resp := make(chan Data, 1)

	if r.responses == nil {
		r.responses = make(map[byte]chan Data)
	}
	r.responses[nodeID] = resp

	if ack {
	loop:
		for i := 1; i <= retries; i++ {
			r.tx <- &Data{
				ToAddress:  nodeID,
				Data:       payload,
				RequestAck: ack,
			}
			if ack == true {
				select {
				case d := <-resp:
					if len(d.Data) > 0 {
						return Data{}, errors.New("invalid ack")
					}
					break loop
				case <-time.After(time.Millisecond * time.Duration(acktime)):
					if i == retries {
						return Data{}, errors.New("no ack response")
					}
				}
			}
		}
	} else {
		r.tx <- &Data{
			ToAddress:  nodeID,
			Data:       payload,
			RequestAck: ack,
		}
	}

	select {
	case d := <-resp:
		delete(r.responses, nodeID)
		if getdata {
			return d, nil
		} else {
			return Data{}, nil
		}
	case <-time.After(time.Millisecond * time.Duration(datatime)):
		delete(r.responses, nodeID)
		if getdata {
			return Data{}, errors.New("no data response")
		} else {
			return Data{}, nil
		}
	}

}

// Close connection to the rfm69 module
func (r *Router) Close() error {
	return r.rfm.Close()
}
