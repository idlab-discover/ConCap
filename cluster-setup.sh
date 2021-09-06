#! /usr/bin/zsh

# Prepare the nfs server, do not expose this setup on the internet
# the no_root_squash is a workaround so that the containers are allowed to write on the share
echo "/storage/nfs/kube *(rw,insecure,sync,no_subtree_check,no_root_squash)" | sudo tee /etc/exports

# Export nfs
sudo exportfs -rav

# WARNING: this is a legacy line, it is here together with kind-example-config.yaml as an ultimate fallback to get going
# Create Kubernetes cluster through kubernetes in docker
# kind create cluster --config kind-example-config.yaml
# Make sure that the tools to mount properly are installed in the nodes
# docker exec kind-worker bash -c "apt update && apt install -y nfs-common" &>/dev/null &
# docker exec kind-worker2 bash -c "apt update && apt install -y nfs-common" &>/dev/null &
# docker exec kind-worker3 bash -c "apt update && apt install -y nfs-common" &>/dev/null &


# Create PersistentVolume and PersistentVolumeClaim
kubectl create -f pv-nfs.yaml
kubectl create -f pvc-nfs.yaml

# Create the IDLab secret
kubectl create secret generic idlab-gitlab --from-file=.dockerconfigjson=/home/dhoogla/.docker/config.json --type=kubernetes.io/dockerconfigjson


