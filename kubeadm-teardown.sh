kubectl drain NODENAME --delete-local-data --force --ignore-daemonsets
# Main clean-up
sudo kubeadm reset
# As stated after main clean-up, some components have to be removed manually
sudo rm -f /etc/cni/net.d/*
rm $HOME/.kube/config