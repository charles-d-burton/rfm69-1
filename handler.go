package rfm69

import (
	"log"

	"github.com/davecheney/gpio"
)

// Router manages sending and receiving of commands / data
type Router struct {
	handlers map[byte]Handle

	rfm *Device

	tx chan *Data
}

// Handle is a function that can be registered to handle an rfm69 message
type Handle func(Data)

// RawHandle registers a generic new event handler for a specific node
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
			log.Println("got data from", data.FromAddress, ", RSSI", data.Rssi)
			if data.ToAddress != 255 && data.RequestAck {
				r.tx <- data.ToAck()
			}

			if h, ok := r.handlers[data.FromAddress]; ok {
				h(data)
			}
		}
	}
}

// Send data to a node
func (r *Router) Send(nodeID byte, payload []byte, ack bool) error {
	r.tx <- &Data{
		ToAddress:  nodeID,
		Data:       payload,
		RequestAck: ack,
	}

	// TODO: Handle ack

	return nil
}

func (r *Router) Close() error {
	return r.rfm.Close()
}
