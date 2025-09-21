package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/3fs-csi/3fs-csi/pkg/config"
	"github.com/3fs-csi/3fs-csi/pkg/mount"
)

type nodeServer struct {
	cfg     *config.Config
	mounter mount.Manager
}

func NewNodeServer(cfg *config.Config, mounter mount.Manager) csi.NodeServer {
	return &nodeServer{cfg: cfg, mounter: mounter}
}

func (n *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, fmt.Errorf("volume_id is required")
	}
	target := req.GetTargetPath()
	if target == "" {
		return nil, fmt.Errorf("target_path is required")
	}
	vc := req.GetVolumeContext()
	subPath, ok := vc["subPath"]
	if !ok || subPath == "" {
		return nil, fmt.Errorf("volume_context.subPath is required")
	}

	if err := n.mounter.EnsureNodeMount(ctx); err != nil {
		return nil, err
	}

	_ = ensureKubeletShared(target)

	source := filepath.Join(n.cfg.GlobalMountPoint(), filepath.Clean("/"+subPath))
	if err := n.mounter.BindMountIfNeeded(source, target); err != nil {
		return nil, err
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, fmt.Errorf("volume_id is required")
	}
	target := req.GetTargetPath()
	if target == "" {
		return nil, fmt.Errorf("target_path is required")
	}
	if err := n.mounter.Unmount(target); err != nil {
		return nil, err
	}
	_ = os.Remove(target)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{Type: &csi.NodeServiceCapability_Rpc{Rpc: &csi.NodeServiceCapability_RPC{Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS}}},
		},
	}, nil
}

func (n *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{NodeId: n.cfg.NodeID}, nil
}

func (n *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (n *nodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	path := req.GetVolumePath()
	if path == "" {
		return nil, fmt.Errorf("volume_path is required")
	}
	var s unix.Statfs_t
	if err := unix.Statfs(path, &s); err != nil {
		return nil, err
	}
	// Sizes reported in bytes
	cap := s.Blocks * uint64(s.Bsize)
	free := s.Bavail * uint64(s.Bsize)
	used := cap - free
	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{Unit: csi.VolumeUsage_BYTES, Total: int64(cap), Used: int64(used), Available: int64(free)},
		},
	}, nil
}

func (n *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, fmt.Errorf("unsupported")
}

func (n *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, fmt.Errorf("unsupported")
}

func (n *nodeServer) NodeGetPreferredAllocationUnits(ctx context.Context, req *csi.NodeGetPreferredAllocationUnitsRequest) (*csi.NodeGetPreferredAllocationUnitsResponse, error) {
	return &csi.NodeGetPreferredAllocationUnitsResponse{}, nil
}

func ensureKubeletShared(targetPath string) error {
	var candidates []string
	candidates = append(candidates, "/var/lib/kubelet")
	if targetPath != "" {
		candidates = append(candidates, filepath.Dir(targetPath))
	}
	for _, c := range candidates {
		if c == "/" || c == "" {
			continue
		}
		if err := tryMakeRShared(c); err == nil {
			return nil
		}
	}
	return nil
}

func tryMakeRShared(path string) error {
	cmd := unix.Mount("none", path, "", uintptr(unix.MS_SHARED|unix.MS_REC), "")
	if cmd == nil {
		return nil
	}
	return nil
}


