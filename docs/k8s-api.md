# API Reference

## Packages
- [fabric.githedgehog.com/v1alpha1](#fabricgithedgehogcomv1alpha1)


## fabric.githedgehog.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the fabric v1alpha1 API group

### Resource Types
- [DeviceRegistration](#deviceregistration)



#### DeviceRegistration



DeviceRegistration is the Schema for the racks API



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `fabric.githedgehog.com/v1alpha1`
| `kind` _string_ | `DeviceRegistration`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[DeviceRegistrationSpec](#deviceregistrationspec)_ |  |
| `status` _[DeviceRegistrationStatus](#deviceregistrationstatus)_ |  |


#### DeviceRegistrationSpec



DeviceSpec defines the properties of a rack which we are modelling

_Appears in:_
- [DeviceRegistration](#deviceregistration)

| Field | Description |
| --- | --- |
| `locationUUID` _string_ |  |
| `csr` _string_ |  |




