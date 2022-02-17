# Only if you still have access to the cluster: 
kubectl drain $(hostname) --delete-local-data --force
sudo systemctl stop kubelet
# Main clean-up
sudo kubeadm reset
# As stated after main clean-up, some components have to be removed manually
sudo rm -rf /etc/cni/net.d
rm $HOME/.kube/config
sudo iptables -F && sudo iptables -t nat -F && sudo iptables -t mangle -F && sudo iptables -X
