package rfm69

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	spiMode  = uint8(0)
	spiBits  = uint8(8)
	spiSpeed = uint32(5000000)
	spiDelay = uint16(8)

	spiIOCWrMode        = 0x40016B01
	spiIOCWrBitsPerWord = 0x40016B03
	spiIOCWrMaxSpeedHz  = 0x40046B04

	spiIOCRdMode        = 0x80016B01
	spiIOCRdBitsPerWord = 0x80016B03
	spiIOCRdMaxSpeedHz  = 0x80046B04

	spiIOCMessage0    = 1073769216 //0x40006B00
	spiIOCIncrementor = 2097152    //0x200000
)

type spiIOCTransfer struct {
	txBuf uint64
	rxBuf uint64

	length      uint32
	speedHz     uint32
	delayus     uint16
	bitsPerWord uint8

	csChange uint8
	pad      uint32
}

func spiIOCMessageN(n uint32) uint32 {
	return (spiIOCMessage0 + (n * spiIOCIncrementor))
}

// spiDevice device
type spiDevice struct {
	file            *os.File
	spiTransferData spiIOCTransfer
}

// newSPIDevice opens the device
func newSPIDevice(devPath string) (*spiDevice, error) {
	s := new(spiDevice)
	s.spiTransferData = spiIOCTransfer{}

	var err error
	if s.file, err = os.OpenFile(devPath, os.O_RDWR, os.ModeExclusive); err != nil {
		return nil, err
	}

	var mode = uint8(spiMode)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, s.file.Fd(), spiIOCWrMode, uintptr(unsafe.Pointer(&mode)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return nil, err
	}

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, s.file.Fd(), spiIOCRdMode, uintptr(unsafe.Pointer(&mode)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return nil, err
	}

	var bpw uint8 = spiBits
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, s.file.Fd(), spiIOCWrBitsPerWord, uintptr(unsafe.Pointer(&bpw)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return nil, err
	}

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, s.file.Fd(), spiIOCRdBitsPerWord, uintptr(unsafe.Pointer(&bpw)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return nil, err
	}
	s.spiTransferData.bitsPerWord = bpw

	var speed uint32 = spiSpeed
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, s.file.Fd(), spiIOCWrMaxSpeedHz, uintptr(unsafe.Pointer(&speed)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return nil, err
	}

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, s.file.Fd(), spiIOCRdMaxSpeedHz, uintptr(unsafe.Pointer(&speed)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return nil, err
	}
	s.spiTransferData.speedHz = speed

	var delay uint16 = spiDelay
	s.spiTransferData.delayus = delay

	return s, nil
}

func (s *spiDevice) spiXfer(tx []byte) ([]byte, error) {
	length := len(tx)
	rx := make([]byte, length)

	dataCarrier := s.spiTransferData
	dataCarrier.length = uint32(length)
	dataCarrier.txBuf = uint64(uintptr(unsafe.Pointer(&tx[0])))
	dataCarrier.rxBuf = uint64(uintptr(unsafe.Pointer(&rx[0])))

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, s.file.Fd(), uintptr(spiIOCMessageN(1)), uintptr(unsafe.Pointer(&dataCarrier)))
	if errno != 0 {
		err := syscall.Errno(errno)
		return nil, err
	}
	return rx, nil
}

func (s *spiDevice) spiClose() {
	s.file.Close()
}
