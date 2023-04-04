# SONiC NOTES

Getting the control VLAN setup in SONiC on eth1 (wich maps to Ethernet0):

```shell
config interface ip remove Ethernet0 10.0.0.0/31
config vlan add 42
config vlan member add 42 Ethernet0
config interface ip add Vlan42 192.168.42.188/24
show vlan brief
```
