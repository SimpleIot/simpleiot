package client

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// Companion file in api/send-file.go

type fileDownload struct {
	name string
	data []byte
	seq  int32
}

// ListenForFile listens for a file sent from server. dir is the directly to place
// downloaded files.
func ListenForFile(nc *nats.Conn, dir, deviceID string, callback func(path string)) error {
	dl := fileDownload{}
	_, err := nc.Subscribe(fmt.Sprintf("device.%v.file", deviceID), func(m *nats.Msg) {
		chunk := &pb.FileChunk{}

		err := proto.Unmarshal(m.Data, chunk)

		if err != nil {
			log.Println("Error decoding file chunk: ", err)
			err := nc.Publish(m.Reply, []byte("error decoding"))
			if err != nil {
				log.Println("Error replying to file download: ", err)
				return
			}
		}

		if chunk.Seq == 0 {
			// we are starting a new stream
			dl.name = chunk.FileName
			dl.data = []byte{}
			dl.seq = 0
		} else if chunk.Seq != dl.seq+1 {
			log.Println("Seq # error in file download: ", dl.seq, chunk.Seq)
			err := nc.Publish(m.Reply, []byte("seq error"))
			if err != nil {
				log.Println("Error replying to file download: ", err)
				return
			}
		}

		// process data from server
		dl.data = append(dl.data, chunk.Data...)
		dl.seq = chunk.Seq

		switch chunk.State {
		case pb.FileChunk_ERROR:
			log.Println("Server error getting chunk")
			// reset download
			dl = fileDownload{}
		case pb.FileChunk_DONE:
			filePath := path.Join(dir, dl.name)
			err := os.WriteFile(filePath, dl.data, 0644)
			if err != nil {
				log.Println("Error writing dl file: ", err)
				err := nc.Publish(m.Reply, []byte("error writing"))
				if err != nil {
					log.Println("Error replying to file download: ", err)
					return
				}
			}

			callback(filePath)
		}

		err = nc.Publish(m.Reply, []byte("OK"))
		if err != nil {
			log.Println("Error replying to file download: ", err)
		}
	})

	return err
}

// SendFile can be used to send a file to a device. Callback provides bytes transferred.
func SendFile(nc *nats.Conn, deviceID string, reader io.Reader, name string, callback func(int)) error {
	done := false
	seq := int32(0)

	bytesTx := 0

	// send file in chunks
	for {
		var err error
		data := make([]byte, 50*1024)
		count, err := reader.Read(data)
		data = data[:count]

		chunk := &pb.FileChunk{Seq: seq, Data: data}

		if seq == 0 {
			chunk.FileName = name
		}

		if err != nil {
			if err != io.EOF {
				chunk.State = pb.FileChunk_ERROR
			} else {
				chunk.State = pb.FileChunk_DONE
			}
			done = true
		}

		out, err := proto.Marshal(chunk)

		if err != nil {
			return err
		}

		subject := fmt.Sprintf("device.%v.file", deviceID)

		retry := 0
		for ; retry < 3; retry++ {
			msg, err := nc.Request(subject, out, time.Minute)

			if err != nil {
				log.Println("Error sending file, retrying: ", retry, err)
				continue
			}

			msgS := string(msg.Data)

			if msgS != "OK" {
				log.Println("Error from device when sending file: ", retry, msgS)
				continue
			}

			// we must have sent OK, break out of loop
			break
		}

		if retry >= 3 {
			return errors.New("Error sending file to device")
		}

		bytesTx += count
		callback(bytesTx)

		if done {
			break
		}

		seq++
	}

	return nil
}
