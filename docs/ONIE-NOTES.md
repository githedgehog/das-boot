# ONIE NOTES

Manual network setup steps to reach the control node in ONIE:

```shell
ip link add link eth0 name control type vlan id 42
ip addr add 192.168.42.200/24 dev control
ip link set control up
ip route add 10.42.0.0/16 via 192.168.42.1 dev control
ip route add 10.43.0.0/16 via 192.168.42.1 dev control
```
