### 3FS CSI Node Plugin (FUSE)

see `docs/README.md` for full details

build image:
```bash
make image
```

deploy via Helm:
```bash
helm install 3fs-csi ./deploy/helm/3fs-csi \
  --set clusterId=stage \
  --set mgmtdAddresses='["RDMA://192.168.1.1:8000"]'
```

examples:
```bash
kubectl apply -f deploy/examples/pv-pvc.yaml
kubectl apply -f deploy/examples/two-pods.yaml
```


