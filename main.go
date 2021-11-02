package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	devicepluginv1beta1 "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const devicePluginsDir = "/var/lib/kubelet/device-plugins"
const socketName = "sample-device-plugin.sock"
const resourceName = "hardware-vendor.example/foo"
const podEnvKey = "foos"

func main() {
	if err := register(); err != nil {
		log.Fatal(fmt.Errorf("failed to register with kubelet: %s", err))
	}

	stop := make(chan struct{})
	go func() {
		if err := watchKubeletRestart(stop); err != nil {
			log.Fatal(fmt.Errorf("error watching kubelet restart: %s", err))
		}
	}()

	if err := serve(); err != nil {
		log.Fatal(fmt.Errorf("error running server: %s", err))
	}

	stop <- struct{}{}
}

func register() error {
	kubeletSocket := filepath.Join(devicePluginsDir, "kubelet.sock")
	conn, err := grpc.Dial("unix://"+kubeletSocket, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to dial %s: %s", kubeletSocket, err)
	}
	defer conn.Close()

	client := devicepluginv1beta1.NewRegistrationClient(conn)
	req := &devicepluginv1beta1.RegisterRequest{
		Version:      "v1beta1",
		Endpoint:     socketName,
		ResourceName: resourceName,
	}
	_, err = client.Register(context.Background(), req)
	return err
}

func watchKubeletRestart(stop chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %s", err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Remove == 0 {
					continue
				}
				if fileName, err := filepath.Rel(devicePluginsDir, event.Name); err == nil && fileName == socketName {
					log.Printf("socket has been removed, exiting")
					os.Exit(0)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %s", err)
			}
		}
	}()

	if err := watcher.Add(devicePluginsDir); err != nil {
		return fmt.Errorf("failed to start watching %s: %s", devicePluginsDir, err)
	}

	<-stop
	return nil
}

func serve() error {
	socket := filepath.Join(devicePluginsDir, socketName)
	_ = os.Remove(socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to listen %s: %s", socket, err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	devicepluginv1beta1.RegisterDevicePluginServer(server, &devicePluginServer{})
	return server.Serve(listener)
}

type devicePluginServer struct {
	devicepluginv1beta1.UnimplementedDevicePluginServer
}

var _ devicepluginv1beta1.DevicePluginServer = &devicePluginServer{}

func (s *devicePluginServer) GetDevicePluginOptions(ctx context.Context, req *devicepluginv1beta1.Empty) (*devicepluginv1beta1.DevicePluginOptions, error) {
	log.Printf("GetDevicePluginOptions")
	return &devicepluginv1beta1.DevicePluginOptions{}, nil
}

func (s *devicePluginServer) ListAndWatch(req *devicepluginv1beta1.Empty, stream devicepluginv1beta1.DevicePlugin_ListAndWatchServer) error {
	log.Printf("ListAndWatch")
	for {
		resp := &devicepluginv1beta1.ListAndWatchResponse{
			Devices: []*devicepluginv1beta1.Device{{
				ID:     "imma_id_1",
				Health: devicepluginv1beta1.Healthy,
			}, {
				ID:     "imma_id_2",
				Health: devicepluginv1beta1.Unhealthy,
			}, {
				ID:     "imma_id_3",
				Health: devicepluginv1beta1.Healthy,
			}},
		}
		if err := stream.Send(resp); err != nil {
			return fmt.Errorf("failed to send response: %s", err)
		}
		time.Sleep(time.Second)
	}
}

func (s *devicePluginServer) Allocate(ctx context.Context, req *devicepluginv1beta1.AllocateRequest) (*devicepluginv1beta1.AllocateResponse, error) {
	log.Printf("Allocate: %+v", req)
	var containerResps []*devicepluginv1beta1.ContainerAllocateResponse
	for _, containerReq := range req.ContainerRequests {
		containerResp := &devicepluginv1beta1.ContainerAllocateResponse{
			Envs: map[string]string{
				podEnvKey: strings.Join(containerReq.DevicesIDs, ","),
			},
		}
		containerResps = append(containerResps, containerResp)
	}
	return &devicepluginv1beta1.AllocateResponse{
		ContainerResponses: containerResps,
	}, nil
}
