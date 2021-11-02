package main

import (
	"context"
	"log"
	"net"
	"testing"

	assert "github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	devicepluginv1beta1 "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func Test_DevicePluginServer_GetDevicePluginOptions(t *testing.T) {
	stop := make(chan struct{})
	client, err := startDevicePluginServer(stop)
	assert.NoError(t, err)
	req := &devicepluginv1beta1.Empty{}
	resp, err := client.GetDevicePluginOptions(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	stop <- struct{}{}
}

func Test_DevicePluginServer_ListAndWatch(t *testing.T) {
	stop := make(chan struct{})
	client, err := startDevicePluginServer(stop)
	assert.NoError(t, err)
	req := &devicepluginv1beta1.Empty{}
	stream, err := client.ListAndWatch(context.Background(), req)
	assert.NoError(t, err)
	for i := 0; i < 3; i++ {
		resp, err := stream.Recv()
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	}
	stop <- struct{}{}
}

func Test_DevicePluginServer_Allocate(t *testing.T) {
	stop := make(chan struct{})
	client, err := startDevicePluginServer(stop)
	assert.NoError(t, err)
	req := &devicepluginv1beta1.AllocateRequest{}
	resp, err := client.Allocate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	stop <- struct{}{}
}

func startDevicePluginServer(stop chan struct{}) (devicepluginv1beta1.DevicePluginClient, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	devicepluginv1beta1.RegisterDevicePluginServer(server, &devicePluginServer{})
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("error running server: %s", err)
		}
	}()

	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	go func() {
		<-stop
		conn.Close()
		server.Stop()
	}()

	return devicepluginv1beta1.NewDevicePluginClient(conn), nil
}
