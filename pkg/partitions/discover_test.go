package partitions

import (
	"reflect"
	"testing"
)

func TestDiscover(t *testing.T) {
	tests := []struct {
		name    string
		want    interface{}
		wantErr bool
	}{
		{
			name: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Discover()
			if (err != nil) != tt.wantErr {
				t.Errorf("Discover() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Discover() = %v, want %v", got, tt.want)
			}
		})
	}
}
