#! /usr/bin/zsh

# Export nfs?
# sudo export exportfs -rav

# Create Kubernetes cluster through kubernetes in docker
kind create cluster --config kind-example-config.yaml

# Create PersistentVolume and PersistentVolumeClaim
kubectl create -f pv-nfs.yaml
kubectl create -f pvc-nfs.yaml

# Create the IDLab secret
kubectl create secret generic idlab-gitlab --from-file=.dockerconfigjson=/home/dhoogla/.docker/config.json --type=kubernetes.io/dockerconfigjson

# Make sure that the tools to mount properly are installed in the nodes
docker exec kind-worker bash -c "apt update && apt install -y nfs-common" &>/dev/null &
docker exec kind-worker2 bash -c "apt update && apt install -y nfs-common" &>/dev/null &
docker exec kind-worker3 bash -c "apt update && apt install -y nfs-common" &>/dev/null &

