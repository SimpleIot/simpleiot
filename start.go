package simpleiot

import (
	"fmt"
	"log"
	"os"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
	"github.com/simpleiot/simpleiot/natsserver"
	"github.com/simpleiot/simpleiot/node"
	"github.com/simpleiot/simpleiot/particle"
	"github.com/simpleiot/simpleiot/store"
)

// Options used for starting SIOT
type Options struct {
	StoreType         store.Type
	DataDir           string
	DisableAuth       bool
	NatsServer        string
	NatsDisableServer bool
	NatsPort          int
	NatsHTTPPort      int
	NatsTLSCert       string
	NatsTLSKey        string
	NatsTLSTimeout    float64
	AuthToken         string
	DebugHTTP         bool
}

// Start Simple IoT data store
func Start(o Options) (*natsgo.Conn, error) {
	// =============================================
	// Start server, default action
	// =============================================
	dbInst, err := store.NewDb(o.StoreType, o.DataDir)
	if err != nil {
		return nil, fmt.Errorf("Error opening db: %v", err)
	}
	defer dbInst.Close()

	// finally, start web server
	port := os.Getenv("SIOT_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	var auth api.Authorizer

	if o.DisableAuth {
		auth = api.AlwaysValid{}
	} else {
		auth, err = api.NewKey(20)
		if err != nil {
			log.Println("Error generating key: ", err)
		}
	}

	if !o.NatsDisableServer {
		go natsserver.StartNatsServer(o.NatsPort, o.NatsHTTPPort, o.AuthToken,
			o.NatsTLSCert, o.NatsTLSKey, o.NatsTLSTimeout)
	}

	natsHandler := store.NewNatsHandler(dbInst, o.AuthToken, o.NatsServer)

	var nc *natsgo.Conn

	// this is a bit of a hack, but we're not sure when the NATS
	// server will be started, so try several times
	for i := 0; i < 10; i++ {
		// FIXME should we get nc with edgeConnect here?
		nc, err = natsHandler.Connect()
		if err != nil {
			log.Println("NATS local connect retry: ", i)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		break
	}

	if err != nil || nc == nil {
		log.Fatal("Error connecting to NATs server: ", err)
	}

	nodeManager := node.NewManger(nc)
	err = nodeManager.Init()
	if err != nil {
		log.Fatal("Error initializing node manager: ", err)
	}
	go nodeManager.Run()

	rootNode, err := nats.GetNode(nc, "root", "")

	if err != nil {
		log.Println("Error getting root id for metrics: ", err)
	} else {

		err = natsHandler.StartMetrics(rootNode.ID)
		if err != nil {
			log.Println("Error starting nats metrics: ", err)
		}
	}

	// set up particle connection if configured
	// todo -- move this to a node
	particleAPIKey := os.Getenv("SIOT_PARTICLE_API_KEY")

	if particleAPIKey != "" {
		go func() {
			err := particle.PointReader("sample", particleAPIKey,
				func(id string, points data.Points) {
					err := nats.SendNodePoints(nc, id, points, false)
					if err != nil {
						log.Println("Error getting particle sample: ", err)
					}
				})

			if err != nil {
				fmt.Println("Get returned error: ", err)
			}
		}()
	}

	err = api.Server(api.ServerArgs{
		Port:       port,
		DbInst:     dbInst,
		GetAsset:   frontend.Asset,
		Filesystem: frontend.FileSystem(),
		Debug:      o.DebugHTTP,
		JwtAuth:    auth,
		AuthToken:  o.AuthToken,
		Nc:         nc,
	})

	return nc, err
}