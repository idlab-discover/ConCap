# Concap K3s Scenario Runbook

## Runtime contract

| Item | Value |
|---|---|
| Namespace | `concap` |
| GHCR pull secret | `concap/ghcr-creds` |
| Attacker node selector | `concap-role=attacker` |
| Target node selector | `concap-role=target` |
| Attacker host | `rgbcore` |
| Target host | `nuccore` |
| Processing pods | Default scheduler placement |

`rgbcore` hosts attackers because its extra CPU, memory, and network-generation headroom prevent attacker-side rate limiting. `nuccore` hosts target services plus target-side `tcpdump`.

Required node selection makes placement fail closed. Missing or unavailable role node leaves pod `Pending`; Kubernetes cannot silently colocate attacker and target.

Cross-node traffic path:

```text
attacker pod
  -> rgbcore cni0
  -> Flannel VXLAN (UDP 8472)
  -> tailscale0 / WireGuard
  -> physical Ethernet LAN
  -> nuccore tailscale0
  -> Flannel
  -> target pod eth0
```

Target-side `tcpdump` captures decoded traffic on pod `eth0`. PCAP does not contain outer VXLAN, WireGuard, or Ethernet encapsulation. Physical path still affects timing, loss, queueing, and throughput.

## One-time cluster setup

Create namespace and GHCR credential from working Docker login:

```sh
kubectl create namespace concap
kubectl -n concap create secret generic ghcr-creds \
  --from-file=.dockerconfigjson="$HOME/.docker/config.json" \
  --type=kubernetes.io/dockerconfigjson
```

Assign physical roles:

```sh
kubectl label node rgbcore concap-role=attacker --overwrite
kubectl label node nuccore concap-role=target --overwrite
```

Generated pods reference `ghcr-creds` directly. Patching default service account is optional, not required by Concap.

Tailnet policy must permit:

| Source | Destination | Traffic |
|---|---|---|
| `rgbcore` | `nuccore` | TCP 6443 |
| Both nodes | Both nodes | TCP 10250 |
| Both nodes | Both nodes | UDP 8472 |

TCP 6443 joins agent to K3s API. TCP 10250 supports kubelet API/metrics. UDP 8472 carries Flannel VXLAN.

## Preflight

Run before dataset batch:

```sh
kubectl get nodes -L concap-role
kubectl -n concap get secret ghcr-creds
kubectl -n concap get pods -o wide
kubectl get node rgbcore nuccore \
  -o custom-columns='NODE:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,DISK:.status.conditions[?(@.type=="DiskPressure")].status,ROLE:.metadata.labels.concap-role'
```

Expected:

```text
nuccore   Ready   DiskPressure=False   target
rgbcore   Ready   DiskPressure=False   attacker
```

Check local agent when `rgbcore` is not ready:

```sh
systemctl status k3s-agent
journalctl -u k3s-agent -n 100 --no-pager
```

Check server:

```sh
ssh nuccore systemctl status k3s
```

## Build and run one scenario

From Concap repository:

```sh
make build
./concap --dir ./example --scenario ssh-hydra-dictionary.yaml --workers 1
```

Observe placement from another terminal:

```sh
watch kubectl -n concap get pods -o wide
```

Expected scenario pod placement:

```text
*-A     -> rgbcore
*-T-0   -> nuccore
```

Multi-target runs place every target on `nuccore`. `--workers N` allows `N` scenarios concurrently; start with `1`, then raise while watching target CPU, memory, disk, and `tcpdump` logs.

## Successful completion

Concap workflow:

1. Validate `concap` namespace and usable `ghcr.io` credentials.
2. Start reusable processing pods.
3. Create attacker on `rgbcore`; target(s) on `nuccore`.
4. Wait for startup probes and pod readiness.
5. Start target-side capture.
6. Execute attacker command.
7. Normalize the raw PCAP into timestamp order, then download PCAPs, capture log, reorder log, and attacker log.
8. Run configured flow processors.
9. Write processor-native CSV outputs and completed scenario YAML.
10. Delete attacker and target pods. Processing pods remain for reuse.

Expected output directory:

```text
example/completed/<scenario-name>/
```

Typical single-target artifacts:

```text
attacker.log
dump.raw.pcap
dump.pcap
tcpdump.log
reordercap.log
scenario.yaml
<processor>.csv
<processor>.log
```

`dump.raw.pcap` is the unmodified target-side tcpdump capture. `dump.pcap` is
the timestamp-normalized capture produced by `reordercap` and is the input used
by processing pods.

Validate completion:

```sh
find example/completed/ssh-hydra-dictionary -maxdepth 1 -type f -ls
kubectl -n concap get pods -l scenario=ssh-hydra-dictionary
```

No scenario-labeled pods should remain after success. Processing pods may remain.

## Failure triage

### Pod remains `Pending`

```sh
kubectl -n concap describe pod <pod>
kubectl get nodes -L concap-role
```

Likely cause: missing role label, labeled node unavailable, insufficient resources.

### `ErrImagePull` or `ImagePullBackOff`

```sh
kubectl -n concap describe pod <pod>
kubectl -n concap get secret ghcr-creds -o jsonpath='{.type}{"\n"}'
```

Expected type: `kubernetes.io/dockerconfigjson`. Recreate secret after `docker login ghcr.io` when token expires.

### `rgbcore` is `NotReady`

Check Tailscale ACL/logs. Explicit ACL rejection looks like:

```text
flow TCP 100.75.79.80 > 100.88.208.64:6443 rejected due to acl
```

Confirm K3s API path:

```sh
curl --noproxy '*' -sk https://100.88.208.64:6443/cacerts >/dev/null
```

### Eviction or image-GC warnings

```sh
df -h /
kubectl describe node rgbcore
docker system df
```

Kubelet starts image GC near 85% filesystem usage. Keep meaningful headroom before large batches.

### Scenario completes without expected flows

Inspect:

```sh
sed -n '1,200p' example/completed/<scenario-name>/tcpdump.log
sed -n '1,200p' example/completed/<scenario-name>/reordercap.log
sed -n '1,200p' example/completed/<scenario-name>/attacker.log
```

Check startup probe, attack exit status, target port, capture filter, and packet drops. Same-node traffic should be impossible while both required role labels exist on different nodes.

## Code checks

```sh
go test ./...
go vet ./...
```
