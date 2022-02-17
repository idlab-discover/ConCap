# Setting up kubernetes with Kubeadm
# Prerequisites
# Install the kubernetes components, follow the instructions for your OS

# Kube host 
sudo kubeadm init --pod-network-cidr=10.1.1.0/24 --control-plane-endpoint 1.1.1.1:6443 --upload-certs # Calico

# Any preflight checks that fail should be addressed, just read them
# They may be missing dependencies / kubelet service which is not auto-started yet by systemd / swap partition which is still on / ...

# create config folder in home
mkdir -p $HOME/.kube
# copy the default config here
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
# set the permissions on this file to the user, rather than root
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# Weave cni
kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"

# NOTE: this is for one-node clusters
# Without this you would not be able to schedule pods on the master
kubectl taint nodes --all node-role.kubernetes.io/master-


# Run 'kubectl get nodes' on the control-plane to see this node join the cluster.

# NOTE: Some useful commands
# kubectl get nodes
# kubectl get pods --all-namespaces
# kubectl -n kube-system describe pod ...
# Many more here: https://kubernetes.io/docs/reference/kubectl/cheatsheet/
# There are smart auto-complete options for kubectl, just look it up to to install them for your shell, I'm using oh-my-zsh

# NOTE: If you get stuck, there are many articles on how to set up kubernetes, though not that many that start straight from kubeadm
# For the configuration of Docker, I have used https://dnaeon.github.io/install-and-configure-k8s-on-arch-linux/
# Also, don't skip on the official documentation