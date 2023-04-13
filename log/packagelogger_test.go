package log

import (
	"reflect"
	"testing"

	"github.com/domonda/golog"
)

func TestNewPackageLogger(t *testing.T) {
	type args struct {
		pkgName string
		filters []golog.LevelFilter
	}
	tests := []struct {
		name string
		args args
		want *golog.Logger
	}{
		{
			name: "pkgName",
			args: args{pkgName: "mypkg", filters: nil},
			want: golog.NewLoggerWithPrefix(golog.NewDerivedConfig(&Config), "mypkg"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPackageLogger(tt.args.pkgName, tt.args.filters...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPackageLogger() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
