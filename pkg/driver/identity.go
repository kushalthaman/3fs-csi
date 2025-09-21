package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"

	"github.com/3fs-csi/3fs-csi/pkg/config"
)

type identityServer struct {
	cfg *config.Config
}

func NewIdentityServer(cfg *config.Config) csi.IdentityServer {
	return &identityServer{cfg: cfg}
}

func (s *identityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	logrus.WithField("driver", s.cfg.DriverName).Debug("GetPluginInfo")
	return &csi.GetPluginInfoResponse{
		Name:          s.cfg.DriverName,
		VendorVersion: "v0.1.0",
	}, nil
}

func (s *identityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
    // Node-only plugin: no controller service advertised
    return &csi.GetPluginCapabilitiesResponse{Capabilities: []*csi.PluginCapability{}}, nil
}

func (s *identityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	ready := true
	return &csi.ProbeResponse{Ready: &csi.ProbeResponse_Ready{Value: ready}}, nil
}


