package network

import (
	"fmt"
	"os"
	"strings"
)

// Ethernet implements the Interface interface
type Ethernet struct {
	iface   string
	enabled bool
}

// NewEthernet contructor
func NewEthernet(iface string) *Ethernet {
	return &Ethernet{
		iface: iface,
	}
}

// Desc returns a description of the interface
func (e *Ethernet) Desc() string {
	return fmt.Sprintf("Eth(%v)", e.iface)
}

// Configure the interface
func (e *Ethernet) Configure() (InterfaceConfig, error) {
	return InterfaceConfig{}, nil
}

// Connect network interface
func (e *Ethernet) Connect() error {
	// this is handled by system so no-op
	return nil
}

func (e *Ethernet) detected() bool {
	cnt, err := os.ReadFile("/sys/class/net/" + e.iface + "/carrier")
	if err != nil {
		return false
	}

	if !strings.Contains(string(cnt), "1") {
		return false
	}

	cnt, err = os.ReadFile("/sys/class/net/" + e.iface + "/operstate")
	if err != nil {
		return false
	}

	if !strings.Contains(string(cnt), "up") {
		return false
	}

	return true
}

// Connected returns true if connected
func (e *Ethernet) connected() bool {
	if !e.detected() {
		return false
	}

	_, err := GetIP(e.iface)
	return err == nil
}

// GetStatus returns ethernet interface status
func (e *Ethernet) GetStatus() (InterfaceStatus, error) {
	ip, _ := GetIP(e.iface)
	return InterfaceStatus{
		Detected:  e.detected(),
		Connected: e.connected(),
		IP:        ip,
	}, nil
}

// Reset interface. Currently no-op for ethernet
func (e *Ethernet) Reset() error {
	return nil
}

// Enable or disable interface
func (e *Ethernet) Enable(en bool) error {
	e.enabled = en
	return nil
}
