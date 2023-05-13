# API Reference

## Packages
- [dasboot.githedgehog.com/v1alpha1](#dasbootgithedgehogcomv1alpha1)


## dasboot.githedgehog.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the fabric v1alpha1 API group

### Resource Types
- [DeviceRegistration](#deviceregistration)





#### DeviceRegistration



DeviceRegistration is the Schema for the device registration within DAS BOOT



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `dasboot.githedgehog.com/v1alpha1`
| `kind` _string_ | `DeviceRegistration`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[DeviceRegistrationSpec](#deviceregistrationspec)_ |  |
| `status` _[DeviceRegistrationStatus](#deviceregistrationstatus)_ |  |


#### DeviceRegistrationSpec



DeviceRegistrationSpec defines the properties of a device registration process

_Appears in:_
- [DeviceRegistration](#deviceregistration)

| Field | Description |
| --- | --- |
| `locationUUID` _string_ |  |
| `csr` _integer array_ |  |


#### DeviceRegistrationStatus



DeviceRegistrationStatus defines the observed state of the device registration process

_Appears in:_
- [DeviceRegistration](#deviceregistration)

| Field | Description |
| --- | --- |
| `certificate` _integer array_ |  |


