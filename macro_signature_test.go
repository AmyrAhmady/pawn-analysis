package analysis_test

import (
	"fmt"
	"strings"
	"testing"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawn-analysis/preprocess"
	"github.com/pawnkit/pawnkit-core/source"
)

func TestIncludedMacroTagsPreserveCallableArity(t *testing.T) {
	include := []byte(`#define TAG(%0) t_%0
#define FONT: TAG(FONT):
#define MODE: TAG(MODE):
native One(FONT:value);
native Two(Text:textid, FONT:font);
native Optional(playerid, FONT:font, MODE:mode = MODE:0);
native Variadic(FONT:font, ...);
`)
	for _, test := range []struct {
		call string
		want string
	}{
		{"One(0); Two(Text:0, 1); Optional(0, 1); Optional(0, 1, 2); Variadic(0, 1, 2);", ""},
		{"One();", `"One" expects 1 arguments, got 0`},
		{"Two(Text:0);", `"Two" expects 2 arguments, got 1`},
		{"Two(Text:0, 1, 2);", `"Two" expects 2 arguments, got 3`},
		{"Optional(0);", `"Optional" expects 2 to 3 arguments, got 1`},
		{"Variadic();", `"Variadic" expects at least 1 arguments, got 0`},
	} {
		t.Run(test.call, func(t *testing.T) {
			result := analysis.Analyze([]byte(fmt.Sprintf("#include <api>\nmain() { %s }", test.call)), analysis.Options{
				Includes: preprocess.MapResolver{"api": include},
			})
			var messages []string
			for _, item := range result.Diagnostics {
				if item.Code == "pawn-analysis:sema/argument-count" {
					messages = append(messages, item.Message)
				}
			}
			if got := strings.Join(messages, "\n"); got != test.want {
				t.Fatalf("argument-count diagnostics = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMacroTaggedSignatureAfterIncrementalEdits(t *testing.T) {
	include := []byte("#define TAG(%0) t_%0\n#define FONT: TAG(FONT):\nnative Two(Text:textid, FONT:font);\n")
	var previous *analysis.Result
	for _, call := range []string{"Two(Text:0, 1)", "Two(Text:0)", "Two(Text:0, 1)"} {
		text := []byte("#include <api>\nmain() { " + call + "; }")
		opts := analysis.Options{
			URI: source.FileURI("main.pwn"), Includes: preprocess.MapResolver{"api": include},
			Previous: previous, ReuseCompatibleExpansion: true,
		}
		incremental := analysis.Analyze(text, opts)
		opts.Previous = nil
		clean := analysis.Analyze(text, opts)
		arity := func(result *analysis.Result) string {
			var messages []string
			for _, item := range result.Diagnostics {
				if item.Code == "pawn-analysis:sema/argument-count" {
					messages = append(messages, item.Message)
				}
			}
			return strings.Join(messages, "\n")
		}
		want := ""
		if call == "Two(Text:0)" {
			want = `"Two" expects 2 arguments, got 1`
		}
		if got := arity(incremental); got != want || got != arity(clean) {
			t.Fatalf("call %q: incremental=%q, clean=%q, want=%q", call, got, arity(clean), want)
		}
		previous = incremental
	}
}
