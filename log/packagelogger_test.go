package log

import (
	"log"
	"reflect"
	"testing"

	"github.com/domonda/golog"
)

func TestNewNamedPackageLogger(t *testing.T) {
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
			if got := NewNamedPackageLogger(tt.args.pkgName, tt.args.filters...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNamedPackageLogger() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

var testPackageLogger = NewPackageLogger()

func TestNewPackageLogger(t *testing.T) {
	t.Run("var testPackageLogger", func(t *testing.T) {
		t.Cleanup(PackageRegistry.Clear)
		if got := testPackageLogger.Prefix(); got != "log" {
			log.Fatalf(`testPackageLogger.Prefix() = %q, want "log"`, got)
		}
	})
	t.Run("NewPackageLogger()", func(t *testing.T) {
		t.Cleanup(PackageRegistry.Clear)
		PackageRegistry.Clear() // Clear state from var testPackageLogger = NewPackageLogger()
		l := NewPackageLogger()
		defer l.Clone() // Prevent inlining
		if got := l.Prefix(); got != "log" {
			log.Fatalf(`NewPackageLogger().Prefix() = %#v, want "log"`, got)
		}
	})
}
