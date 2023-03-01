package location

import (
	"reflect"
	"testing"
)

func TestInfo_MetadataDecoded(t *testing.T) {
	tests := []struct {
		name string
		info *Info
		want Metadata
	}{
		{
			name: "success",
			info: &Info{
				Metadata: `{"a":"aa","b":"bb","c":"cc"}`,
			},
			want: Metadata{
				"a": "aa",
				"b": "bb",
				"c": "cc",
			},
		},
		{
			name: "invalid metadata",
			info: &Info{
				Metadata: `{"invalid":"json`,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.MetadataDecoded(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Info.MetadataDecoded() = %v, want %v", got, tt.want)
			}
		})
	}
}
