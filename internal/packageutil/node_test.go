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
			PackageInfo{Package: "@sentry/tracing", InApp: &testutil.False},
		},
		{
			"official package",
			"/node_modules/express/lib/application.js",
			PackageInfo{Package: "express", InApp: &testutil.False},
		},
		{
			"absolute path",
			"/usr/src/app/node_modules/.pnpm/@sentry+node@7.44.2/node_modules/@sentry/node/cjs/transports/http.js",
			PackageInfo{Package: "@sentry/node", InApp: &testutil.False},
		},
		{
			"internal package with just namespace",
			"node:buffer",
			PackageInfo{Package: "node:buffer", InApp: &testutil.False},
		},
		{
			"internal package with package",
			"node:buffer/some_package",
			PackageInfo{Package: "node:buffer", InApp: &testutil.False},
		},
		{
			"other official package",
			"/node_modules/pg/lib/connection.js",
			PackageInfo{Package: "pg", InApp: &testutil.False},
		},
		{
			"user package",
			"internal/stream_base_commons.js",
			PackageInfo{Package: "", InApp: &testutil.True},
		},
		{
			"absolute path for user package",
			"file:///var/runtime/index.mjs",
			PackageInfo{Package: "", InApp: &testutil.True},
		},
		{
			"absolute third party package with @",
			"/var/task/node_modules/@sentry/node/cjs/eventbuilder.js",
			PackageInfo{Package: "@sentry/node", InApp: &testutil.False},
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

func TestParseNodeModuleFromPath(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output PackageInfo
	}{
		{
			"package with @",
			"/node_modules/@sentry/tracing/cjs/integrations/node/express.js",
			PackageInfo{
				Module: "@sentry/tracing/cjs/integrations/node/express",
				InApp:  &testutil.False,
			},
		},
		{
			"official package",
			"/node_modules/express/lib/application.js",
			PackageInfo{Module: "express/lib/application", InApp: &testutil.False},
		},
		{
			"absolute path",
			"/usr/src/app/node_modules/.pnpm/@sentry+node@7.44.2/node_modules/@sentry/node/cjs/transports/http.js",
			PackageInfo{Module: "@sentry/node/cjs/transports/http", InApp: &testutil.False},
		},
		{
			"internal package with just namespace",
			"node:buffer",
			PackageInfo{Module: "node:buffer", InApp: &testutil.False},
		},
		{
			"internal package with package",
			"node:buffer/some_package",
			PackageInfo{Module: "node:buffer/some_package", InApp: &testutil.False},
		},
		{
			"other official package",
			"/node_modules/pg/lib/connection.js",
			PackageInfo{Module: "pg/lib/connection", InApp: &testutil.False},
		},
		{
			"user package",
			"internal/stream_base_commons.js",
			PackageInfo{Module: "", InApp: &testutil.True},
		},
		{
			"absolute path for user package",
			"file:///var/runtime/index.mjs",
			PackageInfo{Module: "", InApp: &testutil.True},
		},
		{
			"absolute third party package with @",
			"/var/task/node_modules/@sentry/node/cjs/eventbuilder.js",
			PackageInfo{Module: "@sentry/node/cjs/eventbuilder", InApp: &testutil.False},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageInfo := ParseNodeModuleFromPath(tt.input)
			if diff := testutil.Diff(packageInfo, tt.output); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}
