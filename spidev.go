package rfm69

import (
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
)

/*const (
	spiMode  = uint8(0)
	spiBits  = uint8(8)
	spiSpeed = uint32(1000000)
	spiDelay = uint16(8)

	spiIOCWrMode        = 0x40016B01
	spiIOCWrBitsPerWord = 0x40016B03
	spiIOCWrMaxSpeedHz  = 0x40046B04

	spiIOCRdMode        = 0x80016B01
	spiIOCRdBitsPerWord = 0x80016B03
	spiIOCRdMaxSpeedHz  = 0x80046B04

	spiIOCMessage0    = 1073769216 // 0x40006B00
	spiIOCIncrementor = 2097152    // 0x200000
)

/*type spiIOCTransfer struct {
	txBuf uint64
	rxBuf uint64

	length      uint32
	speedHz     uint32
	delayus     uint16
	bitsPerWord uint8

	csChange uint8
	pad      uint32
}*/

/*func spiIOCMessageN(n uint32) uint32 {
	return (spiIOCMessage0 + (n * spiIOCIncrementor))
}*/

// spiDevice device
/*type spiDevice struct {
	file            *os.File
	spiTransferData spiIOCTransfer
}*/

type spiDevice struct {
	port spi.Port
}

// newSPIDevice opens the device
func newSPIDevice(devPath string) (*spiDevice, error) {
	var spiDev spiDevice
	// Use spireg SPI port registry to find the first available SPI bus.
	p, err := spireg.Open("")
	if err != nil {
		return nil, err
	}
	spiDev.port = p

	return &spiDev, nil
}

// Not yet implemented
func (s *spiDevice) Close() {
	// s.port.Close()
	// s.spi.Close()
}

/*func (s *spiDevice) spiClose() {
	s.file.Close()
}*/
