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
