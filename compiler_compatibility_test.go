package analysis_test

import (
	"strings"
	"testing"

	"github.com/pawnkit/pawn-analysis"
)

func TestCompilerAcceptedMacroDeclarations(t *testing.T) {
	for _, test := range []struct {
		name, source string
	}{
		{"callback", `#define Callback:%0(%1) forward %0(%1); public %0(%1)
stock bool:Result() { return true; }
Callback:Run() return Result();
main() {}
`},
		{"command alias", `#define CMD:%1(%2) forward cmd_%1(%2); public cmd_%1(%2)
#define Link:%1->%2; CMD:%1() return cmd_%2();
CMD:flip() { return 1; }
Link:fl->flip;
main() {}
`},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := analysis.Analyze([]byte(test.source), analysis.Options{})
			if len(result.Diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
			}
		})
	}
}

func TestMalformedMacroExpansionReportsInvocation(t *testing.T) {
	const invocation = "Link:fl->flip;"
	text := "#define Link:%1->%2; new = ;\n" + invocation + "\nmain() {}\n"
	start := strings.Index(text, invocation)
	result := analysis.Analyze([]byte(text), analysis.Options{})
	for _, item := range result.Diagnostics {
		if item.Source == "pawn-parser" && int(item.Primary.Start) == start &&
			int(item.Primary.End) == start+len(invocation) {
			if len(item.SafeFixes)+len(item.ReviewFixes) != 0 {
				t.Fatal("suggested a source edit for a synthetic token")
			}
			return
		}
	}
	t.Fatalf("missing expanded syntax diagnostic: %+v", result.Diagnostics)
}

func TestMacroAnalysisRetainsRealDiagnostics(t *testing.T) {
	text := `#define Header:%0() forward Float:%0(); public Float:%0()
stock bool:Value() { return true; }
Header:Result() return Value();
main() { new broken = ; }
`
	result := analysis.Analyze([]byte(text), analysis.Options{})
	var syntaxError, tagError bool
	for _, item := range result.Diagnostics {
		syntaxError = syntaxError || (item.Source == "pawn-parser" &&
			int(item.Primary.Start) >= strings.Index(text, "new broken"))
		tagError = tagError || strings.Contains(item.Message, "return value expects tag Float, got bool")
	}
	if !syntaxError || !tagError {
		t.Fatalf("missing genuine diagnostics: %+v", result.Diagnostics)
	}
}
