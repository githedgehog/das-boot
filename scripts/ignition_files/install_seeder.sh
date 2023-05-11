#!/bin/bash

cd /opt/seeder

# wait until the k3s cluster is really available

i=1
while true; do
    echo "Testing if k3s kubeconfig is available now ($i)..."
    if [ -f /etc/rancher/k3s/k3s.yaml ]; then
        echo "k3s kubeconfig available now"
        break
    fi
    sleep 5
    ((i=i+1))
done

i=1
while true; do
    echo "Testing if k3s cluster is usable now ($i)..."
    kubectl get node &>/dev/null
    if [ $? -eq 0 ]; then
        echo "k3s cluster is usable now"
        break
    fi
    sleep 5
    ((i=i+1))
done

# create all secrets
kubectl apply -f "*-secret.yaml"
if [ $? -ne 0 ]; then
    exit 1
fi

# hack: our special one for DAS BOOT seeder
yq -M '.clusters[0].cluster.server="https://192.168.42.11:6443"' /etc/rancher/k3s/k3s.yaml > /opt/seeder/k3s.yaml
kubectl create secret generic das-boot-kubeconfig --from-file=k3s.yaml=/opt/seeder/k3s.yaml

# install syslog
helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f rsyslog-server-values.yaml hedgehog-syslog oci://registry.local:5000/githedgehog/helm-charts/rsyslog
if [ $? -ne 0 ]; then
    exit 2
fi

# install ntp
helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f ntp-values.yaml hedgehog-ntp oci://registry.local:5000/githedgehog/helm-charts/ntp
if [ $? -ne 0 ]; then
    exit 3
fi

# install our CRDs
helm --kubeconfig /etc/rancher/k3s/k3s.yaml upgrade --install --force --version=0.3 hedgehog-agent-crds oci://registry.local:5000/githedgehog/helm-charts/agent-crd
if [ $? -ne 0 ]; then
    exit 4
fi

helm --kubeconfig /etc/rancher/k3s/k3s.yaml upgrade --install --force --version=0.3.0 hedgehog-wiring-crds oci://registry.local:5000/githedgehog/helm-charts/wiring-crd
if [ $? -ne 0 ]; then
    exit 5
fi

helm --kubeconfig /etc/rancher/k3s/k3s.yaml upgrade --install --force --version=0.2.0 hedgehog-fabric oci://registry.local:5000/githedgehog/helm-charts/fabric-helm
if [ $? -ne 0 ]; then
    exit 6
fi

helm --kubeconfig /etc/rancher/k3s/k3s.yaml upgrade --install hedgehog-das-boot-crds oci://registry.local:5000/githedgehog/helm-charts/das-boot-crds
if [ $? -ne 0 ]; then
    exit 7
fi

helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f das-boot-registration-controller-values.yaml hedgehog-registration-controller oci://registry.local:5000/githedgehog/helm-charts/das-boot-registration-controller
if [ $? -ne 0 ]; then
    exit 8
fi

helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f das-boot-seeder-values.yaml hedgehog-seeder oci://registry.local:5000/githedgehog/helm-charts/das-boot-seeder
if [ $? -ne 0 ]; then
    exit 9
fi

touch /opt/seeder/installed