#! /usr/bin/zsh

# Prepare the nfs server, do not expose this setup on the internet
# WARNING: you only have to do this once, you can just write this line inside /etc/exports
# the no_root_squash is a workaround so that the containers are allowed to write on the share
# NOTE: we will move towards working with POD's security-context to pre-specify which user will write
echo "/storage/nfs/L/kube *(rw,insecure,sync,no_subtree_check,no_root_squash)" | sudo tee /etc/exports

# Export nfs, better option is to make use of the systemd startup for nfs-server.service
# sudo exportfs -rav
sudo systemctl start nfs-server.service
# Issues can arise with rpcbind and showmount. If there are timeouts, it is most often because a service isn't running
# Additionally, you will need the nfs-utils for you distribution to be able to connect
# If you are certain that the mount location is always mounted, then you can solidify the config with an enable of the nfs-service

# Create PersistentVolume and PersistentVolumeClaim
kubectl create -f pv-nfs.yaml
kubectl create -f pvc-nfs.yaml

# Create the IDLab secret
kubectl create secret docker-registry idlab-gitlab --docker-server=gitlab.ilabt.imec.be:4567 --docker-username=lpdhooge --docker-password=$(head $HOME/.docker/containercap-gitlab-token)
