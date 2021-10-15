# Setting up kubernetes with Kubeadm
# Prerequisites
# Install the kubernetes components, follow the instructions for your OS

# Kube host 
sudo kubeadm init --pod-network-cidr=10.244.0.0/16 --control-plane-endpoint localhost:6443 --upload-certs

# Any preflight checks that fail should be addressed, just read them
# They may be missing dependencies / kubelet service which is not auto-started yet by systemd / swap partition which is still on / ...

# OUTPUT should look something like this
# [sudo] password for dhoogla:
# [init] Using Kubernetes version: v1.22.1
# [preflight] Running pre-flight checks
# [preflight] Pulling images required for setting up a Kubernetes cluster
# [preflight] This might take a minute or two, depending on the speed of your internet connection
# [preflight] You can also perform this action in beforehand using 'kubeadm config images pull'
# [certs] Using certificateDir folder "/etc/kubernetes/pki"
# [certs] Generating "ca" certificate and key
# [certs] Generating "apiserver" certificate and key
# [certs] apiserver serving cert is signed for DNS names [corarch kubernetes kubernetes.default kubernetes.default.svc kubernetes.default.svc.cluster.local] and IPs [10.96.0.1 192.168.0.208]
# [certs] Generating "apiserver-kubelet-client" certificate and key
# [certs] Generating "front-proxy-ca" certificate and key
# [certs] Generating "front-proxy-client" certificate and key
# [certs] Generating "etcd/ca" certificate and key
# [certs] Generating "etcd/server" certificate and key
# [certs] etcd/server serving cert is signed for DNS names [corarch localhost] and IPs [192.168.0.208 127.0.0.1 ::1]
# [certs] Generating "etcd/peer" certificate and key
# [certs] etcd/peer serving cert is signed for DNS names [corarch localhost] and IPs [192.168.0.208 127.0.0.1 ::1]
# [certs] Generating "etcd/healthcheck-client" certificate and key
# [certs] Generating "apiserver-etcd-client" certificate and key
# [certs] Generating "sa" key and public key
# [kubeconfig] Using kubeconfig folder "/etc/kubernetes"
# [kubeconfig] Writing "admin.conf" kubeconfig file
# [kubeconfig] Writing "kubelet.conf" kubeconfig file
# [kubeconfig] Writing "controller-manager.conf" kubeconfig file
# [kubeconfig] Writing "scheduler.conf" kubeconfig file
# [kubelet-start] Writing kubelet environment file with flags to file "/var/lib/kubelet/kubeadm-flags.env"
# [kubelet-start] Writing kubelet configuration to file "/var/lib/kubelet/config.yaml"
# [kubelet-start] Starting the kubelet
# [control-plane] Using manifest folder "/etc/kubernetes/manifests"
# [control-plane] Creating static Pod manifest for "kube-apiserver"
# [control-plane] Creating static Pod manifest for "kube-controller-manager"
# [control-plane] Creating static Pod manifest for "kube-scheduler"
# [etcd] Creating static Pod manifest for local etcd in "/etc/kubernetes/manifests"
# [wait-control-plane] Waiting for the kubelet to boot up the control plane as static Pods from directory "/etc/kubernetes/manifests". This can take up to 4m0s
# [apiclient] All control plane components are healthy after 6.001551 seconds
# [upload-config] Storing the configuration used in ConfigMap "kubeadm-config" in the "kube-system" Namespace
# [kubelet] Creating a ConfigMap "kubelet-config-1.22" in namespace kube-system with the configuration for the kubelets in the cluster
# [upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
# [upload-certs] Using certificate key:
# eb66ac2e56dbca7f6df08725accfd5efe7847595aacfbb4a806eb3def304ce22
# [mark-control-plane] Marking the node corarch as control-plane by adding the labels: [node-role.kubernetes.io/master(deprecated) node-role.kubernetes.io/control-plane node.kubernetes.io/exclude-from-external-load-balancers]
# [mark-control-plane] Marking the node corarch as control-plane by adding the taints [node-role.kubernetes.io/master:NoSchedule]
# [bootstrap-token] Using token: dz8zs0.nnu0iw059g4pyxsl
# [bootstrap-token] Configuring bootstrap tokens, cluster-info ConfigMap, RBAC Roles
# [bootstrap-token] configured RBAC rules to allow Node Bootstrap tokens to get nodes
# [bootstrap-token] configured RBAC rules to allow Node Bootstrap tokens to post CSRs in order for nodes to get long term certificate credentials
# [bootstrap-token] configured RBAC rules to allow the csrapprover controller automatically approve CSRs from a Node Bootstrap Token
# [bootstrap-token] configured RBAC rules to allow certificate rotation for all node client certificates in the cluster
# [bootstrap-token] Creating the "cluster-info" ConfigMap in the "kube-public" namespace
# [kubelet-finalize] Updating "/etc/kubernetes/kubelet.conf" to point to a rotatable kubelet client certificate and key
# [addons] Applied essential addon: CoreDNS
# [addons] Applied essential addon: kube-proxy

# Your Kubernetes control-plane has initialized successfully!

# create config folder in home
mkdir -p $HOME/.kube
# copy the default config here
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
# set the permissions on this file to the user, rather than root
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# NOTE: this is for one-node clusters
# Without this you would not be able to schedule pods on the master
kubectl taint nodes --all node-role.kubernetes.io/master-

# I've chosen flannel for pod networking, but there are lots of options
# WARNING: you may have to install flannel-cni separately first, before it can be applied within Kubernetes
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
# Warning: policy/v1beta1 PodSecurityPolicy is deprecated in v1.21+, unavailable in v1.25+
# podsecuritypolicy.policy/psp.flannel.unprivileged created
# clusterrole.rbac.authorization.k8s.io/flannel created
# clusterrolebinding.rbac.authorization.k8s.io/flannel created
# serviceaccount/flannel created
# configmap/kube-flannel-cfg created
# daemonset.apps/kube-flannel-ds created

# NOTE: this is for multi-node clusters
# On the new worker machine, join the cluster
#kubeadm join 192.168.0.208:6443 --token nuler0.mre2mo0ywtgxuros \
#	--discovery-token-ca-cert-hash sha256:0f8d51f551dfa4a5a45d564144455773ee629cbc215121537398251a7334e465

# [preflight] Running pre-flight checks
# [preflight] Reading configuration from the cluster...
# [preflight] FYI: You can look at this config file with 'kubectl -n kube-system get cm kubeadm-config -o yaml'
# [kubelet-start] Writing kubelet configuration to file "/var/lib/kubelet/config.yaml"
# [kubelet-start] Writing kubelet environment file with flags to file "/var/lib/kubelet/kubeadm-flags.env"
# [kubelet-start] Starting the kubelet
# [kubelet-start] Waiting for the kubelet to perform the TLS Bootstrap...

# This node has joined the cluster:
# * Certificate signing request was sent to apiserver and a response was received.
# * The Kubelet was informed of the new secure connection details.

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