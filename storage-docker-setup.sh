#! /usr/bin/zsh

# Prepare the nfs server, do not expose this setup on the internet
# WARNING: you only have to do this once, you can just write this line inside /etc/exports
# the no_root_squash is a workaround so that the containers are allowed to write on the share
# NOTE: we will move towards working with POD's security-context to pre-specify which user will write
echo "/storage/nfs/kube *(rw,insecure,sync,no_subtree_check,no_root_squash)" | sudo tee /etc/exports

# Export nfs, better option is to make use of the systemd startup for nfs-server.service
# sudo exportfs -rav
sudo systemctl start nfs-server.service
# Issues can arise with rpcbind and showmount. If there are timeouts, it is most often because a service isn't running
# Additionally, you will need the nfs-utils for you distribution to be able to connect
# If you are certain that the mount location is always mounted, then you can solidify the config with an enable of the nfs-service

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

# NOTE: that dockerconfig file should look like this
# {
# 	"auths": {
# 		"gitlab.ilabt.imec.be:4567": {
# 			"auth": "[username:password]encoded in base64"
# 		}
# 	},
# 	"HttpHeaders": {
# 		"User-Agent": "Docker-Client/20.10.8 (linux)"
# 	}
# }

