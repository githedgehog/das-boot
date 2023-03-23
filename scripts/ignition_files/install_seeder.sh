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

# now install the helm chart
helm --kubeconfig /etc/rancher/k3s/k3s.yaml install -f das-boot-seeder-values.yaml hedgehog oci://registry.local:5000/githedgehog/helm-charts/das-boot-seeder
if [ $? -ne 0 ]; then
    exit 2
fi

touch /opt/seeder/installed