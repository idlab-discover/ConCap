## Get a Local or Remote K8s Cluster

### Launch Kubernetes Cluster on CloudNativeLab

1. Browse to [CloudNativeLab](https://practicum.cloudnativelab.ilabt.imec.be/CreateCluster) and choose number of nodes and their resources.
2. Download Kube config and VPN config, and configure them on your working device.
   1. Install VPN [source](https://github.com/idlab-discover/CloudNativeLab/wiki/Retrieve-and-install-VPN-config)
      - Download OpenVPN Client
      - Import downloaded vpn file
      - Connect the VPN
   2. Install Kubectl ([source](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/))
    ```
    # Download Kubectl
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

    # Download checksum file and validate kubectl download: output should be "kubectl: OK"
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
    echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check

    # Install Kubectl
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

    # Validate correct install
    kubectl version --client
    ```

    3. Configure Kubectl
       - Move downloaded `kubeconfig.txt` to `~/.kube` and rename to `config` (merge if needed with existing config). 
       - Verify if you can access the cluster with `kubectl get all`.
    4. Access a node
       - Install kubectl plugin [`node-shell`](https://github.com/kvaps/kubectl-node-shell#installation)
        ```
        curl -LO https://github.com/kvaps/kubectl-node-shell/raw/master/kubectl-node_shell
        chmod +x ./kubectl-node_shell
        sudo mv ./kubectl-node_shell /usr/local/bin/kubectl-node_shell
        ```
       - Start terminal on node
        ```
        # Get node id
        kubectl get nodes

        # Spawn terminal on node
        kubectl node-shell <id>
        ```  

### Run Local K8s cluster using MicroK8s

1. Install MicroK8s by following the [official documentation](https://microk8s.io/#install-microk8s) for Linux, MacOS or Windows 10/11.

2. Next, you need to add the kubeconfig for the MicroK8s cluster to `kubectl` since `ConCap` uses the default `kubectl` command on your host to communicate with the K8s cluster. 

```sh
# If this is your first K8s cluster
microk8s config > ~/.kube/config

# Otherwise, you need to merge the config with your existing config
cp ~/.kube/config ~/.kube/config-backup
microk8s config > ~/.kube/microk8s
export KUBECONFIG=~/.kube/config-backup:~/.kube/microk8s
kubectl config view --flatten > ~/.kube/config
kubectl config use-context microk8s

# Test if you can use kubectl, change default context in ~/.kube/config if needed
kubectl config get-clusters
kubectl get nodes
```

## Install K9s [Optional]
  
    Check latest release [here](https://github.com/derailed/k9s/releases)
  ```
  curl -s -L https://github.com/derailed/k9s/releases/download/v0.31.9/k9s_Linux_amd64.tar.gz -o k9s
  tar -xvf k9s 
  chmod 755 k9s 
  rm LICENSE README.md  
  sudo mv k9s /usr/local/bin
  ``` 

## Install ContainerCap

1. Download Go [source](https://go.dev/doc/install)
```
# Check latest version before downloading
curl -s -L https://golang.org/dl/go1.22.0.linux-amd64.tar.gz -o go
sudo tar -C /usr/local -xzf go
# Add go to path (edit .bashrc)
export PATH=$PATH:/usr/local/go/bin
# Test go
go version
```
2. Clone ContainerCap Git Repo [source](https://gitlab.ilabt.imec.be/lpdhooge/containercap/-/tree/master?ref_type=heads)
3. go.mod > delete all require, change version to 1.22
4. sudo apt install libpcap-dev
