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

# install syslog
helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f rsyslog-server-values.yaml hh-syslog oci://registry.local:5000/githedgehog/helm-charts/rsyslog
if [ $? -ne 0 ]; then
    exit 2
fi

# install ntp
helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f ntp-values.yaml hh-ntp oci://registry.local:5000/githedgehog/helm-charts/ntp
if [ $? -ne 0 ]; then
    exit 3
fi

# install our CRDs
helm --kubeconfig /etc/rancher/k3s/k3s.yaml upgrade --install hh-fabric-api --force --version=v0.15.2 oci://registry.local:5000/githedgehog/helm-charts/fabric-api
if [ $? -ne 0 ]; then
    exit 4
fi

helm --kubeconfig /etc/rancher/k3s/k3s.yaml upgrade --install hh-das-boot-crds oci://registry.local:5000/githedgehog/helm-charts/das-boot-crds
if [ $? -ne 0 ]; then
    exit 6
fi

# apply wiring yaml
kubectl apply -f wiring.yaml
if [ $? -ne 0 ]; then
    exit 7
fi

# install the fabric controller
# TODO: currently broken with the new wiring
#helm --kubeconfig /etc/rancher/k3s/k3s.yaml upgrade --install --force --version=0.2.0 hh-fabric oci://registry.local:5000/githedgehog/helm-charts/fabric-helm
#if [ $? -ne 0 ]; then
#    exit 8
#fi

# install DAS BOOT - registration controller
helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f das-boot-registration-controller-values.yaml hh-rc oci://registry.local:5000/githedgehog/helm-charts/das-boot-registration-controller
if [ $? -ne 0 ]; then
    exit 9
fi

# install DAS BOOT - seeder
helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f das-boot-seeder-values.yaml hh-seeder oci://registry.local:5000/githedgehog/helm-charts/das-boot-seeder
if [ $? -ne 0 ]; then
    exit 10
fi

touch /opt/seeder/installed
