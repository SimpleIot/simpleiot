package client

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dim13/cobs"
)

type ioSim struct {
	out  *bytes.Buffer
	in   *bytes.Buffer
	m    *sync.Mutex
	stop chan struct{}
}

// newIoSim creates a new IO sim and returns the A and B side of an IO simulator
// that implements a ReadWriteCloser
func newIoSim() (*ioSim, *ioSim) {
	var a2b bytes.Buffer
	var b2a bytes.Buffer
	var m sync.Mutex

	a := ioSim{&a2b, &b2a, &m, make(chan struct{})}
	b := ioSim{&b2a, &a2b, &m, make(chan struct{})}

	return &a, &b
}

func (ios *ioSim) Write(d []byte) (int, error) {
	ios.m.Lock()
	defer ios.m.Unlock()
	return ios.in.Write(d)
}

// Read blocks until there is some data in the out buffer or the ioSim is closed.
func (ios *ioSim) Read(d []byte) (int, error) {
	ret := make(chan struct{})

	go func() {
		for {
			ios.m.Lock()
			if ios.out.Len() > 0 {
				close(ret)
				ios.m.Unlock()
				return
			}
			ios.m.Unlock()
			select {
			case <-time.After(time.Millisecond):
				// continue
			case <-ios.stop:
				close(ret)
				return
			}
		}
	}()

	// block until we have data
	<-ret
	ios.m.Lock()
	defer ios.m.Unlock()
	return ios.out.Read(d)
}

func (ios *ioSim) Close() error {
	close(ios.stop)
	return nil
}

func TestCobsRead(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := newIoSim()

	cw := newCobsWrapper(b)

	a.Write(append([]byte{0}, cobs.Encode(d)...))

	buf := make([]byte, 500)

	c, err := cw.Read(buf)
	if err != nil {
		t.Fatal("Error reading cw: ", err)
	}
	buf = buf[0:c]

	if !reflect.DeepEqual(buf, d) {
		t.Fatal("Read data does not match")
	}
}

func TestCobsWrite(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := newIoSim()

	cw := newCobsWrapper(b)

	_, err := cw.Write(d)
	if err != nil {
		t.Fatal("Error write: ", err)
	}

	buf := make([]byte, 500)

	c, err := a.Read(buf)
	if err != nil {
		t.Fatal("Error read: ", err)
	}
	buf = buf[0:c]

	if buf[0] != 0 {
		t.Fatal("COBS encoded packet must start with 0")
	}

	if !reflect.DeepEqual(cobs.Decode(buf[1:]), d) {
		t.Fatal("cw.Write, buf is not same")
	}
}

func TestCobsWrapperPartialPacket(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}

	a, b := newIoSim()

	cw := newCobsWrapper(b)

	de := append([]byte{0}, cobs.Encode(d)...)

	// write part of packet
	a.Write(de[0:4])

	// start reader
	readData := make(chan []byte)
	errCh := make(chan error)

	go func() {
		buf := make([]byte, 500)
		c, err := cw.Read(buf)
		if err != nil {
			errCh <- err
		}
		buf = buf[0:c]
		readData <- buf
	}()

	// should time out as we don't have entire packet to decode yet
	select {
	case <-readData:
		t.Fatal("should not have read data yet")
	case err := <-errCh:
		t.Fatal("Read failed when it should have blocked: ", err)
	case <-time.After(time.Millisecond * 10):
		// all is well
	}

	// write the rest of the packet
	a.Write(de[4:])

	// now look for packet
	select {
	case ret := <-readData:
		if !reflect.DeepEqual(ret, d) {
			t.Fatal("Read data does not match")
		}
	case err := <-errCh:
		t.Fatal("Read failed: ", err)
	case <-time.After(time.Millisecond * 10):
		t.Fatal("Timeout reading packet")
	}
}

func TestCobsTwoLeadingZeros(t *testing.T) {
	d := []byte{1, 2, 3, 0, 4, 5, 6}
	a, b := newIoSim()

	cw := newCobsWrapper(b)

	a.Write(append([]byte{0, 0}, cobs.Encode(d)...))

	buf := make([]byte, 500)

	c, err := cw.Read(buf)
	if err != nil {
		t.Fatal("Error reading cw: ", err)
	}
	buf = buf[0:c]

	if !reflect.DeepEqual(buf, d) {
		t.Fatal("Read data does not match")
	}
}