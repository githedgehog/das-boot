# QEMU Notes

Just as a reference, this is how my current ONIE test device is running in GNS3:

```shell
/usr/bin/qemu-system-x86_64 \
  -name ONIEx86_64latest-4 -m 4096M \
  -smp cpus=1,sockets=1 -enable-kvm -machine smm=off \
  -boot order=d -drive file=/home/mheese/GNS3/projects/hedgehog/project-files/qemu/a56aec4d-100e-4af0-8206-02a50f5e96f4/hda_disk.qcow2,if=pflash,index=0,media=disk,id=drive0 \
  -drive file=/home/mheese/GNS3/projects/hedgehog/project-files/qemu/a56aec4d-100e-4af0-8206-02a50f5e96f4/hdb_disk.qcow2,if=pflash,index=1,media=disk,id=drive1 \
  -drive file=/home/mheese/GNS3/projects/hedgehog/project-files/qemu/a56aec4d-100e-4af0-8206-02a50f5e96f4/hdc_disk.qcow2,if=virtio,index=2,media=disk,id=drive2 \
  -uuid a56aec4d-100e-4af0-8206-02a50f5e96f4 \
  -serial telnet:127.0.0.1:5000,server,nowait \
  -monitor tcp:127.0.0.1:34077,server,nowait \
  -net none \
  -device e1000,mac=0c:6a:ec:4d:00:00,netdev=gns3-0 \
  -netdev socket,id=gns3-0,udp=127.0.0.1:10005,localaddr=127.0.0.1:10004 \
  -display none \
  -chardev socket,id=chrtpm,path=/tmp/tmp4lsmf5r1/swtpm.sock \
  -tpmdev emulator,id=tpm0,chardev=chrtpm \
  -device tpm-tis,tpmdev=tpm0 \
  -machine q35,smm=on \
  -global ICH9-LPC.disable_s3=1
```

And this is how my current control node on a standard Ubuntu 22 VM is running in GNS3:

```shell
/usr/bin/qemu-system-x86_64 \
  -name UbuntuCloudGuest22.04(LTS)-1 -m 4096M \
  -smp cpus=1,sockets=1 -enable-kvm -machine smm=off \
  -boot order=c -cdrom /home/mheese/GNS3/images/QEMU/ubuntu-cloud-init-data.iso \
  -drive file=/home/mheese/GNS3/projects/hedgehog/project-files/qemu/339bbe6f-e85a-450c-9b1e-89d836f2a4ea/hda_disk.qcow2,if=virtio,index=0,media=disk,id=drive0 \
  -uuid 339bbe6f-e85a-450c-9b1e-89d836f2a4ea \
  -serial telnet:127.0.0.1:5002,server,nowait \
  -monitor tcp:127.0.0.1:36825,server,nowait \
  -net none \
  -device virtio-net-pci,mac=0c:9b:be:6f:00:00,netdev=gns3-0 \
  -netdev socket,id=gns3-0,udp=127.0.0.1:10007,localaddr=127.0.0.1:10006 \
  -device virtio-net-pci,mac=0c:9b:be:6f:00:01,netdev=gns3-1 \
  -netdev socket,id=gns3-1,udp=127.0.0.1:10009,localaddr=127.0.0.1:10008 \
  -nographic
```

## On device hot plugging

Sources are from here:

- [QEMU on PCIE](https://github.com/qemu/qemu/blob/master/docs/pcie.txt)
- [QEMU on PCIE to PCI Bridge](https://github.com/qemu/qemu/blob/master/docs/pcie_pci_bridge.txt)

Technically, it is possible, but I'm not sure we want to go through all that trouble at the moment.
However, here are my notes on it:

Adding the following to the qemu commandline:

```console
-device pcie-root-port,bus=pcie.0,id=rp1,slot=1 -device pcie-pci-bridge,id=br1,bus=rp1
```

allowed me then to hotplug the device by running the following commands (one can only hotplug PCI devices into PCI Express to PCI and PCI-PCI Bridges):

```console
(qemu) netdev_add socket,id=eth3,udp=127.0.0.1:22222,localaddr=127.0.0.1:22221
netdev_add socket,id=eth3,udp=127.0.0.1:22222,localaddr=127.0.0.1:22221
(qemu) device_add virtio-net-pci,id=eth3pci,netdev=eth3,bus=br1
device_add virtio-net-pci,id=eth3pci,netdev=eth3,bus=br1```

and I can see the device with `info pci` in the qemu monitor:

```console
Bus  2, device   0, function 0:
    Ethernet controller: PCI device 1af4:1000
      PCI subsystem 1af4:0001
      IRQ 0, pin A
      BAR0: I/O at 0xffffffffffffffff [0x001e].
      BAR1: 32 bit memory at 0xffffffffffffffff [0x00000ffe].
      BAR4: 64 bit prefetchable memory at 0xffffffffffffffff [0x00003ffe].
      BAR6: 32 bit memory at 0xffffffffffffffff [0x0003fffe].
      id "eth3pci"
```

but unfortunately the device isn't being picked up by the guest

```console
ONIE:/sys/bus/pci/devices # lspci
00:1f.2 Class 0106: 8086:2922
00:1f.0 Class 0601: 8086:2918
00:01.0 Class 0604: 1b36:000c
00:04.0 Class 0200: 1af4:1000
01:00.0 Class 0604: 1b36:000e
00:1f.3 Class 0c05: 8086:2930
00:00.0 Class 0600: 8086:29c0
00:03.0 Class 0200: 1af4:1000
00:06.0 Class 0100: 1af4:1001
00:02.0 Class 0200: 1af4:1000
00:05.0 Class 00ff: 1af4:1005
```

This might be just a limitation in ONIE though because apparently one needs a Linux module to be loaded (`shpchp`) as the hotplug is SHPC based.
There might also be an easier solution: the hotplug into a PCI-PCI bridge is ACPI based which I think would mean that we wouldn't need an additional module.

## On OVMF firmware debugging

See OVMF notes, however, if the firmware has debugging enabled, but not the `DEBUG_ON_SERIAL_PORT` option during build (which you should avoid),
then the output is accessible through the following options to QEMU:

```shell
-debugcon file:ovmf-debug.log -global isa-debugcon.iobase=0x402
```