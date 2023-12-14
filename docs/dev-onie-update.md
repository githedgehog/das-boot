# Dev Notes for ONIE Updates

We originally had it planned that ONIE updates, like regular NOS installations, need to go through stage0 and stage1 first before they can download the ONIE update image.
That would mean that the device must perform device registration first before it can download the ONIE updater - exactly the same way as it needs to be registered first before it can download a NOS installer image.
That said, technically neither the NOS installer nor the ONIE update *need* to be under that type of protection.
Downloading the device configuration and credentials need to be under that type of protection, however, just downloading the NOS or ONIE updater do not necessarily need this.

There is this function in ONIE which checks if the image type is an ONIE updater:

```shell
# Return the type of installer image, either a firmware update or NOS
# installer.
get_image_type()
{
    # ONIE updater images *must* contain the string
    # "ONIE-UPDATER-COOKIE" within the first 100 lines of the image.
    if head -n 100 $1 | grep -q "$onie_updater_cookie" ; then
        echo -n $onie_image_type_update
    else
        echo -n $onie_image_type_nos
    fi
}
```

While possible, this is rather difficult to handle in stage0 as we are shipping a golang binary as installer.
We cannot change the `get_image_type()` function as the big use-case here is that we can potentially easily upgrade standard ONIE installations with HONIE before attempting a regular fabric/NOS installation.

So we are going to offer routes on the insecure handler which allow ONIE images to be directly downloaded.
In this case we are going to use the most platform specific route to build the right handler.

Here an output from the waterfall queries which are running:

```log
ONIE: Starting ONIE Service Discovery
Info: Attempting file://dev/vda4/onie-updater-x86_64-kvm_x86_64-r0 ...
Info: Attempting file://dev/vda4/onie-updater-x86_64-kvm_x86_64-r0.bin ...
Info: Attempting file://dev/vda4/onie-updater-x86_64-kvm_x86_64 ...
Info: Attempting file://dev/vda4/onie-updater-x86_64-kvm_x86_64.bin ...
Info: Attempting file://dev/vda4/onie-updater-kvm_x86_64 ...
Info: Attempting file://dev/vda4/onie-updater-kvm_x86_64.bin ...
Info: Attempting file://dev/vda4/onie-updater-x86_64-qemu ...
Info: Attempting file://dev/vda4/onie-updater-x86_64-qemu.bin ...
Info: Attempting file://dev/vda4/onie-updater-x86_64 ...
Info: Attempting file://dev/vda4/onie-updater-x86_64.bin ...
Info: Attempting file://dev/vda4/onie-updater ...
Info: Attempting file://dev/vda4/onie-updater.bin ...
Info: Attempting file://dev/vda3/onie-updater-x86_64-kvm_x86_64-r0 ...
Info: Attempting file://dev/vda3/onie-updater-x86_64-kvm_x86_64-r0.bin ...
Info: Attempting file://dev/vda3/onie-updater-x86_64-kvm_x86_64 ...
Info: Attempting file://dev/vda3/onie-updater-x86_64-kvm_x86_64.bin ...
Info: Attempting file://dev/vda3/onie-updater-kvm_x86_64 ...
Info: Attempting file://dev/vda3/onie-updater-kvm_x86_64.bin ...
Info: Attempting file://dev/vda3/onie-updater-x86_64-qemu ...
Info: Attempting file://dev/vda3/onie-updater-x86_64-qemu.bin ...
Info: Attempting file://dev/vda3/onie-updater-x86_64 ...
Info: Attempting file://dev/vda3/onie-updater-x86_64.bin ...
Info: Attempting file://dev/vda3/onie-updater ...
Info: Attempting file://dev/vda3/onie-updater.bin ...
Info: Discovered servers through LLDP network configuration: 172.30.1.1
Info: Skipping Neighbor Discovery, as we found servers through LLDP...
Info: Attempting http://172.30.1.1/onie-updater-x86_64-kvm_x86_64-r0 ...
Info: Attempting http://172.30.1.1/onie-updater-x86_64-kvm_x86_64-r0.bin ...
Info: Attempting http://172.30.1.1/onie-updater-x86_64-kvm_x86_64 ...
Info: Attempting http://172.30.1.1/onie-updater-x86_64-kvm_x86_64.bin ...
Info: Attempting http://172.30.1.1/onie-updater-kvm_x86_64 ...
Info: Attempting http://172.30.1.1/onie-updater-kvm_x86_64.bin ...
Info: Attempting http://172.30.1.1/onie-updater-x86_64-qemu ...
Info: Attempting http://172.30.1.1/onie-updater-x86_64-qemu.bin ...
Info: Attempting http://172.30.1.1/onie-updater-x86_64 ...
ONIE: Executing installer: http://172.30.1.1/onie-updater-x86_64
Failure: ONIE Update: Invalid ONIE update image format.
Info: Attempting http://172.30.1.1/onie-updater-x86_64.bin ...
Info: Attempting http://172.30.1.1/onie-updater ...
ONIE: Executing installer: http://172.30.1.1/onie-updater
Failure: ONIE Update: Invalid ONIE update image format.
Info: Attempting http://172.30.1.1/onie-updater.bin ...
Info: Attempting tftp://onie-server/0c-20-12-fe-07-00/onie-updater-x86_64-kvm_x86_64-r0 ...
Info: Attempting tftp://onie-server/onie-updater-x86_64-kvm_x86_64-r0 ...
Info: Attempting tftp://onie-server/onie-updater-x86_64-kvm_x86_64-r0.bin ...
Info: Attempting tftp://onie-server/onie-updater-x86_64-kvm_x86_64 ...
Info: Attempting tftp://onie-server/onie-updater-x86_64-kvm_x86_64.bin ...
Info: Attempting tftp://onie-server/onie-updater-kvm_x86_64 ...
Info: Attempting tftp://onie-server/onie-updater-kvm_x86_64.bin ...
Info: Attempting tftp://onie-server/onie-updater-x86_64-qemu ...
Info: Attempting tftp://onie-server/onie-updater-x86_64-qemu.bin ...
Info: Attempting tftp://onie-server/onie-updater-x86_64 ...
Info: Attempting tftp://onie-server/onie-updater-x86_64.bin ...
Info: Attempting tftp://onie-server/onie-updater ...
Info: Attempting tftp://onie-server/onie-updater.bin ...
```

In the docs the waterfall is described as:

```text
onie-installer-<arch>-<vendor>_<machine>-r<machine_revision>
```