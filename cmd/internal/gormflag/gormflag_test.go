//spellchecker:words gormflag
package gormflag_test

//spellchecker:words flag testing github quickpid internal gormflag gorm logger Logger
import (
	"flag"
	"testing"

	"github.com/tkw1536/quickpid/cmd/internal/gormflag"
	gormLogger "gorm.io/gorm/logger"
)

func TestLogLevel_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    gormLogger.LogLevel
		wantErr bool
	}{
		{name: "silent", input: "silent", want: gormLogger.Silent},
		{name: "error", input: "error", want: gormLogger.Error},
		{name: "warn", input: "warn", want: gormLogger.Warn},
		{name: "warning", input: "warning", want: gormLogger.Warn},
		{name: "info", input: "info", want: gormLogger.Info},
		{name: "uppercase", input: "INFO", want: gormLogger.Info},
		{name: "invalid", input: "debug", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var level gormflag.LogLevel
			err := level.Set(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Set(%q) = %v", tt.input, err)
			}
			if got := level.Level(); got != tt.want {
				t.Fatalf("Level() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogLevel_flagValue(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	level := gormflag.DefaultLogLevel
	fs.Var(&level, "gorm-log-level", gormflag.FlagUsage)

	if err := fs.Parse([]string{"-gorm-log-level", "info"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := level.Level(); got != gormLogger.Info {
		t.Fatalf("Level() = %v, want %v", got, gormLogger.Info)
	}
	if got := level.String(); got != "info" {
		t.Fatalf("String() = %q, want %q", got, "info")
	}
}
