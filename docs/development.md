# Development Scratchpad

Upgrading the seeder helm chart in a running vlab, and adjusting some settings:

```shell
helm upgrade das-boot-seeder oci://registry.local:31000/githedgehog/helm-charts/das-boot-seeder --insecure-skip-tls-verify --reuse-values --set image.tag=latest --set settings.secure_server_name=172.30.1.1
```

Adding iptables rule to allow 443 to our control VIP through

```shell
sudo iptables -t nat -I PREROUTING 1 -4 -d 172.30.1.1/32 -p tcp --dport 443 -j ACCEPT
```
