# EFI NOTES

## Setting variables

The kernel interface to interact with the EFI variables lies within a filesystem called `efivarfs`.
It needs to be mounted with `rw` option for you to be able to make changes to it. Like this for example:

```console
efivarfs on /sys/firmware/efi/efivars type efivarfs (rw,relatime)
```

The system expects this to be always mounted at `/sys/firmware/efi/efivars`.

You can create a new variable by creating a file in this directory.
However, the file name must adhere to the naming convention _NAME-GUID_.
The _NAME_ being the name of the variable, and _GUID_ being the vendor GUID.

The Hedgehog vendor GUID is: `d7bf196e-80c4-44ca-9cd2-26fb6a18101e`.

The files are usually set to being immutable (even with `rw` mount option).
The attribute can be changed with `chattr -i`.

Run the following commands to create a new variable:

```shell
touch /sys/firmware/efi/efivars/ONIEDisableDHCPv4-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
chattr -i /sys/firmware/efi/efivars/ONIEDisableDHCPv4-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
printf "\x07\x00\x00\x00\x01" > /sys/firmware/efi/efivars/ONIEDisableDHCPv4-d7bf196e-80c4-44ca-9cd2-26fb6a18101e
```

The `0x07` byte will set the following attributes for the variable:

```console
Attributes:
    Non-Volatile
    Boot Service Access
    Runtime Service Access
```

This will make sure that the variable survives a restart, and that both bootloaders (the DXE), as well as the OS have access to the variable.
Look at [Variable Services](https://uefi.org/specs/UEFI/2.10/08_Services_Runtime_Services.html#variable-services) for more information about EFI variables.
In general, `0x7` is exactly what we need.
This is a bit mask of the following attributes:

- EFI_VARIABLE_NON_VOLATILE 0x00000001
- EFI_VARIABLE_BOOTSERVICE_ACCESS 0x00000002
- EFI_VARIABLE_RUNTIME_ACCESS 0x00000004

Note that runtime access requires bootservice access as well.

## Reading variables

We can use the `efivar` utility from the Red Hat bootloader team to query it which can be found [here]().
ONIE includes it, so it makes it very convenient for scripting.
It uses it to detect if Secure Boot is enabled for example.
This is done by the following test:

```shell
efivar -d -n 8be4df61-93ca-11d2-aa0d-00e098032b8c-SecureBoot
```

This queries the SecureBoot non-volatile variable.
The `-d` flag requests decimal value output.
This is perfect for flags or numbers.
In this case this command will either return `1` if Secure Boot is enabled or `0` if Secure Boot is not enabled.

We are going to make use of this in exactly the same use in ONIE.
We will test if DHCPv4 can be skipped/disabled by testing our own variable like it:

```console
ONIE:/ # efivar -d -n d7bf196e-80c4-44ca-9cd2-26fb6a18101e-ONIEDisableDHCPv4
1
```
