package packageutil

import (
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

type output struct {
	Name  string
	InApp bool
}

func TestParseNodePackageFromPath(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output output
	}{
		{
			"package with @",
			"/node_modules/@sentry/tracing/cjs/integrations/node/express.js",
			output{"@sentry/tracing", false},
		},
		{"official package", "/node_modules/express/lib/application.js", output{"express", false}},
		{
			"absolute path",
			"/usr/src/app/node_modules/.pnpm/@sentry+node@7.44.2/node_modules/@sentry/node/cjs/transports/http.js",
			output{"@sentry/node", false},
		},
		{"internal package with just namespace", "node:buffer", output{"node:buffer", false}},
		{"internal package with package", "node:buffer/some_package", output{"node:buffer", false}},
		{"other official package", "/node_modules/pg/lib/connection.js", output{"pg", false}},
		{"user package", "internal/stream_base_commons.js", output{"", true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageName, inApp := ParseNodePackageFromPath(tt.input)
			if diff := testutil.Diff(output{packageName, inApp}, tt.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
