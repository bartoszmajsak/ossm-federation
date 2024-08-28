package adsc

import (
	"context"
	"errors"

	"math"
	"time"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	istiolog "istio.io/istio/pkg/log"
)

const (
	defaultClientMaxReceiveMessageSize = math.MaxInt32
	defaultInitialConnWindowSize       = 1024 * 1024 // default gRPC InitialWindowSize
	defaultInitialWindowSize           = 1024 * 1024 // default gRPC ConnWindowSize
)

var log = istiolog.RegisterScope("adsc", "Aggregated Discovery Service Client")

type ADSCConfig struct {
	DiscoveryAddr            string
	InitialDiscoveryRequests []*discovery.DiscoveryRequest
	Handlers                 map[string]ResponseHandler
}

type ADSC struct {
	stream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	// xds client used to create a stream
	client discovery.AggregatedDiscoveryServiceClient
	conn   *grpc.ClientConn
	cfg    *ADSCConfig
}

func New(opts *ADSCConfig) (*ADSC, error) {
	if opts == nil {
		return nil, errors.New("adsc: opts is nil")
	}
	adsc := &ADSC{cfg: opts}
	if err := adsc.dial(); err != nil {
		return nil, err
	}

	return adsc, nil
}

func (a *ADSC) Run() error {
	var err error
	a.client = discovery.NewAggregatedDiscoveryServiceClient(a.conn)
	a.stream, err = a.client.StreamAggregatedResources(context.Background())
	if err != nil {
		return err
	}
	// Send the initial requests
	for _, r := range a.cfg.InitialDiscoveryRequests {
		_ = a.send(r)
	}

	go a.handleRecv()
	return nil
}

func (a *ADSC) send(req *discovery.DiscoveryRequest) error {
	req.ResponseNonce = time.Now().String()
	log.Infof("Sending Discovery Request to ADS server: %s", req.String())
	return a.stream.Send(req)
}

func (a *ADSC) dial() error {
	var err error
	a.conn, err = grpc.NewClient(
		a.cfg.DiscoveryAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithInitialWindowSize(int32(defaultInitialWindowSize)),
		grpc.WithInitialConnWindowSize(int32(defaultInitialConnWindowSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize)),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: 5 * time.Second,
		}),
	)
	if err != nil {
		return err
	}
	return nil
}

func (a *ADSC) handleRecv() {
	for {
		var err error
		msg, err := a.stream.Recv()
		if err != nil {
			log.Infof("connection closed with err: %v", err)
			return
		}
		log.Infof("received response of type %s", msg.TypeUrl)
		log.Infof("received response body %v", msg.Resources)
		if handler, found := a.cfg.Handlers[msg.TypeUrl]; found {
			log.Infof("ResponseHandler found for type %s", msg.TypeUrl)
			if err := handler.Handle(msg.Resources); err != nil {
				log.Infof("error handling resource %s: %v", msg.TypeUrl, err)
			}
		} else {
			log.Infof("no handler found for type: %s", msg.TypeUrl)
		}
	}
}