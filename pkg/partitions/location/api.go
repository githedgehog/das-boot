package location

type LocationPartition interface{}

type Info struct {
	UUID        string
	UUIDSig     []byte
	Metadata    string
	MetadataSig []byte
}

type Metadata map[string]string
