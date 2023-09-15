package ipam

import (
	"testing"
)

func Test_ensureIPHasCIDR(t *testing.T) {
	type args struct {
		ip string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "success",
			args: args{ip: "192.168.42.1"},
			want: "192.168.42.1/32",
		},
		{
			name: "success-with-cidr",
			args: args{ip: "192.168.42.1/24"},
			want: "192.168.42.1/32",
		},
		{
			name: "success-ipv6",
			args: args{ip: "::1"},
			want: "::1/128",
		},
		{
			name: "success-ipv6-with-cidr",
			args: args{ip: "2001:aa::1/64"},
			want: "2001:aa::1/128",
		},
		{
			name:    "not an IP",
			args:    args{ip: "not an IP"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureIPHasCIDR(tt.args.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureIPHasCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ensureIPHasCIDR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ensureIPHasNoCIDR(t *testing.T) {
	type args struct {
		ip string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "success",
			args: args{ip: "192.168.101.1/31"},
			want: "192.168.101.1",
		},
		{
			name: "success-no-cidr",
			args: args{ip: "192.168.101.1"},
			want: "192.168.101.1",
		},
		{
			name:    "not an IP",
			args:    args{ip: "not an IP"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureIPHasNoCIDR(tt.args.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureIPHasNoCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ensureIPHasNoCIDR() = %v, want %v", got, tt.want)
			}
		})
	}
}
