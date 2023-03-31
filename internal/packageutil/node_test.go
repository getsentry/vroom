package packageutil

import (
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

func TestParseNodePackageFromPath(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output PackageInfo
	}{
		{
			"package with @",
			"/node_modules/@sentry/tracing/cjs/integrations/node/express.js",
			PackageInfo{"@sentry/tracing", &testutil.False},
		},
		{
			"official package",
			"/node_modules/express/lib/application.js",
			PackageInfo{"express", &testutil.False},
		},
		{
			"absolute path",
			"/usr/src/app/node_modules/.pnpm/@sentry+node@7.44.2/node_modules/@sentry/node/cjs/transports/http.js",
			PackageInfo{"@sentry/node", &testutil.False},
		},
		{
			"internal package with just namespace",
			"node:buffer",
			PackageInfo{"node:buffer", &testutil.False},
		},
		{
			"internal package with package",
			"node:buffer/some_package",
			PackageInfo{"node:buffer", &testutil.False},
		},
		{
			"other official package",
			"/node_modules/pg/lib/connection.js",
			PackageInfo{"pg", &testutil.False},
		},
		{"user package", "internal/stream_base_commons.js", PackageInfo{"", &testutil.True}},
		{
			"absolute path for user package",
			"file:///var/runtime/index.mjs",
			PackageInfo{"", &testutil.True},
		},
		{
			"absolute third party package with @",
			"/var/task/node_modules/@sentry/node/cjs/eventbuilder.js",
			PackageInfo{"@sentry/node", &testutil.False},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageInfo := ParseNodePackageFromPath(tt.input)
			if diff := testutil.Diff(packageInfo, tt.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
