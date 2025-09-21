## 3FS CSI Node Plugin (FUSE)

This is a Kubernetes CSI Node plugin that exposes 3FS to pods by bind-mounting subdirectories from a single node-wide 3FS FUSE mount.

### prereqs
- 3FS control plane is deployed and reachable from nodes
- A user and/or admin token is available and a secret `3fs-token` with key `token` is available
- Nodes provide `/dev/fuse` and allow mount propagation and run Linux kernel supported by FUSE
- If using RDMA endpoints, we ensure RDMA user-space libraries are present on the node or included in the image

### config 
- `clusterId` (required)
- `mgmtdAddresses` (required): array of `RDMA://<ip>:<port>`
- `globalMountBase` default `/var/lib/3fs/mnt`
- `hf3fsBinaryPath` default `/opt/3fs/bin/hf3fs_fuse_main`
- `tokenFile` default `/var/lib/3fs/token.txt`

### install via Helm
```bash
helm install 3fs-csi ./deploy/helm/3fs-csi \
  --set clusterId=stage \
  --set mgmtdAddresses='["RDMA://192.168.1.1:8000"]'
```

### static PV/PVC Example
Apply `deploy/examples/pv-pvc.yaml`, then `deploy/examples/two-pods.yaml`.

### implementation 
- Node plugin ensures exactly one FUSE mount per node at `/var/lib/3fs/mnt/<clusterId>`
- `NodePublishVolume` bind-mounts `<globalMount>/<subPath>` to the target path
- In case of error in build make sure `/etc/fuse.conf` has `user_allow_other` if required by your environment
- Check node logs for `hf3fs_fuse_main` startup issues
- Make sure `/var/lib/kubelet` is mounted with shared propagation on the node


