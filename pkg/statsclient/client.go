package statsclient

import (
	"context"
	"fmt"

	"github.com/Alpin-A/prism/pkg/statsgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC connection to the Python stats service.
type Client struct {
	conn   *grpc.ClientConn
	client statsgrpc.StatsServiceClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connecting to stats service at %s: %w", addr, err)
	}
	return &Client{conn: conn, client: statsgrpc.NewStatsServiceClient(conn)}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) GetExperimentResult(ctx context.Context, experimentID, eventType string) (*statsgrpc.ExperimentResultResponse, error) {
	return c.client.GetExperimentResult(ctx, &statsgrpc.ExperimentResultRequest{
		ExperimentId: experimentID,
		EventType:    eventType,
	})
}
