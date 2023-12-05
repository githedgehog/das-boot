# Development Scratchpad

## DAS BOOT Development with hhfab/vlab

Upgrading the seeder helm chart in a running vlab, and adjusting some settings:

```shell
helm upgrade das-boot-seeder oci://registry.local:31000/githedgehog/helm-charts/das-boot-seeder --insecure-skip-tls-verify --reuse-values --set image.tag=latest --set settings.secure_server_name=172.30.1.1
```

Adding iptables rule to allow 443 to our control VIP through

```shell
sudo iptables -t nat -I PREROUTING 1 -4 -d 172.30.1.1/32 -p tcp --dport 443 -j ACCEPT
```

## Creating a new ONIE kvm image

- build it first with the usual `make MACHINE="kvm_x86_64" all`
- then prepare the VM by going into the `emulation/` folder
- wipe any previous `emulation-files/` or `oras/` which might be in here (just remove the whole folders)

```shell
./onie-vm.sh --m-bios-uefi --m-embed-onie --m-onie-iso ../build/images/onie-recovery-x86_64-kvm_x86_64-r0.iso
```

- in a separate terminal connect to it with `telnet localhost 9300`
- select the `Embed ONIE` option
- let it install, and reboot
- on reboot, select `Rescue` to enter ONIE
- then prep the VM image by disabling DHCP and IPv4 link-local addressing (this speeds up things for us)

```shell
# ONIEDisableDHCPv4 variable
touch /sys/firmware/efi/efivars/ONIEDisableDHCPv4-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
chattr -i /sys/firmware/efi/efivars/ONIEDisableDHCPv4-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
printf "\x07\x00\x00\x00\x01" > /sys/firmware/efi/efivars/ONIEDisableDHCPv4-d7bf196e-80c4-44ca-9cd2-26fb6a18101e

# ONIEDisableIPv4LL variable
touch /sys/firmware/efi/efivars/ONIEDisableIPv4LL-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
chattr -i /sys/firmware/efi/efivars/ONIEDisableIPv4LL-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
printf "\x07\x00\x00\x00\x01" > /sys/firmware/efi/efivars/ONIEDisableIPv4LL-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
```

- reboot again, and observe that the EFI variables are taking effect (there are log messages showing that both are skipped)
- power off the VM gracefully
- next you can prepare the OCI image with `oras`:

```shell
mkdir oras
cd oras/
cp ../emulation-files/onie-kvm_x86_64-demo.qcow2 onie-kvm_x86_64.qcow2
xz onie-kvm_x86_64.qcow2 
cp ../emulation-files/uefi-bios/x86/OVMF_CODE.fd onie_efi_code.fd
cp ../emulation-files/uefi-bios/x86/OVMF_VARS.fd onie_efi_vars.fd
oras push ghcr.io/githedgehog/honie:latest *
```

## Reboot with SysRq key

Issue a reboot with sysrq through the console

```shell
sudo sh -c "echo b > /proc/sysrq-trigger"
```

# scp into ONIE

```shell
scp -O onie-updater-x86_64-accton_as7726_32x-r0-marcus root@172.30.20.7:
```
