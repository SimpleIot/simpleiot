package client

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"golang.org/x/exp/slices"
)

// NTPConfigPath points to the systemd-timesyncd configuration file
const NTPConfigPath = "/etc/systemd/timesyncd.conf.d/simpleiot.conf"

// NTPClient is a SimpleIoT client that synchronizes NTP servers to local
// systemd-timesync configuration
type NTPClient struct {
	log      *log.Logger
	nc       *nats.Conn
	config   NTP
	stopCh   chan struct{}
	pointsCh chan NewPoints
}

// NTP client configuration
type NTP struct {
	ID              string   `node:"id"`
	Parent          string   `node:"parent"`
	Servers         []string `point:"server"`
	FallbackServers []string `point:"fallbackServer"`
}

// NewNTPClient returns a new NTPClient using its
// configuration read from the Client Manager
func NewNTPClient(nc *nats.Conn, config NTP) Client {
	// TODO: Ensure only one NTP client exists
	return &NTPClient{
		log:      log.New(os.Stderr, "NTP: ", log.LstdFlags|log.Lmsgprefix),
		nc:       nc,
		config:   config,
		stopCh:   make(chan struct{}),
		pointsCh: make(chan NewPoints),
	}
}

// Run starts the NTP Client
func (c *NTPClient) Run() error {
	c.log.Println("Starting NTP client")
	err := c.UpdateConfig()
	if err != nil {
		c.log.Println("error updating systemd-timesyncd config:", err)
	}
loop:
	for {
		select {
		case <-c.stopCh:
			break loop
		case points := <-c.pointsCh:
			// Update local configuration
			err := data.MergePoints(points.ID, points.Points, &c.config)
			if err != nil {
				return fmt.Errorf("merging points: %w", err)
			}
			err = c.UpdateConfig()
			if err != nil {
				c.log.Println("error updating systemd-timesyncd config:", err)
				break // select
			}
		}
	}
	return nil
}

// UpdateConfig writes the NTP configuration to NTPConfigPath and restarts
// system-timesyncd
func (c *NTPClient) UpdateConfig() error {
	data := []byte(`
# This file is auto-generated by SimpleIoT
# DO NOT EDIT OR REMOVE!
NTP=` + strings.Join(c.config.Servers, " ") + `
FallbackNTP=` + strings.Join(c.config.FallbackServers, " ") + `
`)
	f, err := os.OpenFile(NTPConfigPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	currData, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if slices.Equal(currData, data) {
		// No changes to file
		return nil
	}
	n, err := f.WriteAt(data, 0)
	if err != nil {
		return err
	}
	err = f.Truncate(int64(n))
	if err != nil {
		return err
	}
	// Restart NTP
	return exec.Command("/usr/bin/systemctl", "restart", "systemd-timesyncd").Run()
}

// Stop stops the NTP Client
func (c *NTPClient) Stop(error) {
	close(c.stopCh)
}

// Points is called when the client's node points are updated
func (c *NTPClient) Points(nodeID string, points []data.Point) {
	c.pointsCh <- NewPoints{
		ID:     nodeID,
		Points: points,
	}
}

// EdgePoints is called when the client's node edge points are updated
func (c *NTPClient) EdgePoints(
	_ string, _ string, _ []data.Point,
) {
	// Do nothing
}
