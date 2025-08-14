package frame

import (
	"hash/fnv"
	"testing"

	"github.com/getsentry/vroom/internal/platform"
)

func frameType(isApplication bool) string {
	if isApplication {
		return "application"
	}
	return "system"
}

const mockUUID = "00000000-0000-0000-0000-000000000000"

func TestIsCocoaApplicationFrame(t *testing.T) {
	tests := []struct {
		name          string
		frame         Frame
		isApplication bool
	}{
		{
			name: "main",
			frame: Frame{
				Function: "main",
				Status:   "symbolicated",
				Package:  "/Users/runner/Library/Developer/CoreSimulator/Devices/" + mockUUID + "/data/Containers/Bundle/Application/" + mockUUID + "/iOS-Swift.app/Frameworks/libclang_rt.asan_iossim_dynamic.dylib",
			},
			isApplication: false,
		},
		{
			name: "main must be symbolicated",
			frame: Frame{
				Function: "main",
				Package:  "/Users/runner/Library/Developer/CoreSimulator/Devices/" + mockUUID + "/data/Containers/Bundle/Application/" + mockUUID + "/iOS-Swift.app/Frameworks/libclang_rt.asan_iossim_dynamic.dylib",
			},
			isApplication: true,
		},
		{
			name: "__sanitizer::StackDepotNode::store(unsigned int, __sanitizer::StackTrace const&, unsigned long long)",
			frame: Frame{
				Function: "__sanitizer::StackDepotNode::store(unsigned int, __sanitizer::StackTrace const&, unsigned long long)",
				Package:  "/Users/runner/Library/Developer/CoreSimulator/Devices/" + mockUUID + "/data/Containers/Bundle/Application/" + mockUUID + "/iOS-Swift.app/Frameworks/libclang_rt.asan_iossim_dynamic.dylib",
			},
			isApplication: true,
		},
		{
			name: "symbolicate_internal",
			frame: Frame{
				Function: "symbolicate_internal",
				Package:  "/private/var/containers/Bundle/Application/00000000-0000-0000-0000-000000000000/App.app/Frameworks/Sentry.framework/Sentry",
			},
			isApplication: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isApplication := tt.frame.IsCocoaApplicationFrame(); isApplication != tt.isApplication {
				t.Fatalf(
					"Expected %s frame but got %s frame",
					frameType(tt.isApplication),
					frameType(isApplication),
				)
			}
		})
	}
}

func TestIsPythonApplicationFrame(t *testing.T) {
	tests := []struct {
		name          string
		frame         Frame
		isApplication bool
	}{
		{
			name:          "empty",
			frame:         Frame{},
			isApplication: true,
		},
		{
			name: "app",
			frame: Frame{
				Module: "app",
				File:   "app.py",
				Path:   "/home/user/app/app.py",
			},
			isApplication: true,
		},
		{
			name: "app.utils",
			frame: Frame{
				Module: "app.utils",
				File:   "app/utils.py",
				Path:   "/home/user/app/app/utils.py",
			},
			isApplication: true,
		},
		{
			name: "site-packges unix",
			frame: Frame{
				Path: "/usr/local/lib/python3.10/site-packages/urllib3/request.py",
			},
			isApplication: false,
		},
		{
			name: "site-packges dos",
			frame: Frame{
				Path: "C:\\Users\\user\\AppData\\Local\\Programs\\Python\\Python310\\lib\\site-packages\\urllib3\\request.py",
			},
			isApplication: false,
		},
		{
			name: "dist-packges unix",
			frame: Frame{
				Path: "/usr/local/lib/python3.10/dist-packages/urllib3/request.py",
			},
			isApplication: false,
		},
		{
			name: "dist-packges dos",
			frame: Frame{
				Path: "C:\\Users\\user\\AppData\\Local\\Programs\\Python\\Python310\\lib\\dist-packages\\urllib3\\request.py",
			},
			isApplication: false,
		},
		{
			name: "stdlib",
			frame: Frame{
				Module: "multiprocessing.pool",
			},
			isApplication: false,
		},
		{
			name: "sentry_sdk",
			frame: Frame{
				Module: "sentry_sdk.profiler",
			},
			isApplication: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isApplication := tt.frame.IsPythonApplicationFrame(); isApplication != tt.isApplication {
				t.Fatalf(
					"Expected %s frame but got %s frame",
					frameType(tt.isApplication),
					frameType(isApplication),
				)
			}
		})
	}
}

func TestIsNodeApplicationFrame(t *testing.T) {
	tests := []struct {
		name          string
		frame         Frame
		isApplication bool
	}{
		{
			name:          "empty",
			frame:         Frame{},
			isApplication: true,
		},
		{
			name: "app",
			frame: Frame{
				Path: "/home/user/app/app.js",
			},
			isApplication: true,
		},
		{
			name: "node_modules",
			frame: Frame{
				Path: "/home/user/app/node_modules/express/lib/express.js",
			},
			isApplication: false,
		},
		{
			name: "internal",
			frame: Frame{
				Path: "node:internal/process/task_queues",
			},
			isApplication: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isApplication := tt.frame.IsNodeApplicationFrame(); isApplication != tt.isApplication {
				t.Fatalf(
					"Expected %s frame but got %s frame",
					frameType(tt.isApplication),
					frameType(isApplication),
				)
			}
		})
	}
}

func TestIsJavaScriptApplicationFrame(t *testing.T) {
	tests := []struct {
		name          string
		frame         Frame
		isApplication bool
	}{
		{
			name:          "empty",
			frame:         Frame{},
			isApplication: true,
		},
		{
			name: "app",
			frame: Frame{
				Path: "/home/user/app/app.js",
			},
			isApplication: true,
		},
		{
			name: "node_modules",
			frame: Frame{
				Path: "/home/user/app/node_modules/express/lib/express.js",
			},
			isApplication: false,
		},
		{
			name: "app",
			frame: Frame{
				Path: "@moz-extension://00000000-0000-0000-0000-000000000000/app.js",
			},
			isApplication: false,
		},
		{
			name: "app",
			frame: Frame{
				Path: "chrome-extension://00000000-0000-0000-0000-000000000000/app.js",
			},
			isApplication: false,
		},
		{
			name: "native",
			frame: Frame{
				Function: "[Native] functionPrototypeApply",
			},
			isApplication: false,
		},
		{
			name: "host_function",
			frame: Frame{
				Function: "[HostFunction] nativeCallSyncHook",
			},
			isApplication: false,
		},
		{
			name: "gc",
			frame: Frame{
				Function: "[GC Young Gen]",
			},
			isApplication: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isApplication := tt.frame.IsJavaScriptApplicationFrame(); isApplication != tt.isApplication {
				t.Fatalf(
					"Expected %s frame but got %s frame",
					frameType(tt.isApplication),
					frameType(isApplication),
				)
			}
		})
	}
}

func TestIsPHPApplicationFrame(t *testing.T) {
	tests := []struct {
		name          string
		frame         Frame
		isApplication bool
	}{
		{
			name:          "empty",
			frame:         Frame{},
			isApplication: true,
		},
		{
			name: "file",
			frame: Frame{
				Function: "/var/www/http/webroot/index.php",
				File:     "/var/www/http/webroot/index.php",
			},
			isApplication: true,
		},
		{
			name: "src",
			frame: Frame{
				Function: "App\\Middleware\\SentryMiddleware::process",
				File:     "/var/www/http/src/Middleware/SentryMiddleware.php",
			},
			isApplication: true,
		},
		{
			name: "vendor",
			frame: Frame{
				File: "Cake\\Http\\Client::send",
				Path: "/var/www/http/vendor/cakephp/cakephp/src/Http/Client.php",
			},
			isApplication: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isApplication := tt.frame.IsPHPApplicationFrame(); isApplication != tt.isApplication {
				t.Fatalf(
					"Expected %s frame but got %s frame",
					frameType(tt.isApplication),
					frameType(isApplication),
				)
			}
		})
	}
}

func TestWriteToHash(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		frame Frame
	}{
		{
			name:  "unknown frame",
			bytes: []byte("--"),
			frame: Frame{},
		},
		{
			name:  "prefers function module over package",
			bytes: []byte("foo-"),
			frame: Frame{
				Module:  "foo",
				Package: "/bar/bar",
				File:    "baz",
			},
		},
		{
			name:  "prefers package over file",
			bytes: []byte("bar-"),
			frame: Frame{
				Package: "/bar/bar",
				File:    "baz",
			},
		},
		{
			name:  "prefers file over nothing",
			bytes: []byte("baz-"),
			frame: Frame{
				File: "baz",
			},
		},
		{
			name:  "uses function name",
			bytes: []byte("-qux"),
			frame: Frame{
				Function: "qux",
			},
		},
		{
			name:  "native unknown frame",
			bytes: []byte("--0x123456789"),
			frame: Frame{
				InstructionAddr: "0x123456789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1 := fnv.New64()
			h1.Write(tt.bytes)

			h2 := fnv.New64()
			tt.frame.WriteToHash(h2)

			s1 := h1.Sum64()
			s2 := h2.Sum64()

			if s1 != s2 {
				t.Fatalf("Expected hash %d frame but got %d", s1, s2)
			}
		})
	}
}

func TestTrimPackage(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		expected string
	}{
		{
			pkg:      "/System/Library/PrivateFrameworks/UIKitCore.framework/UIKitCore",
			expected: "UIKitCore",
		},
		{
			// strips the .dylib
			pkg:      "/usr/lib/system/libsystem_pthread.dylib",
			expected: "libsystem_pthread",
		},
		{
			pkg:      "/lib/x86_64-linux-gnu/libc.so.6",
			expected: "libc.so.6",
		},
		{
			pkg:      "/foo",
			expected: "foo",
		},
		{
			// ignore single trailing slash
			pkg:      "/foo/",
			expected: "foo",
		},
		{
			// does not ignore multiple trailing slash
			pkg:      "/foo//",
			expected: "/foo//",
		},
		{
			pkg:      "C:\\WINDOWS\\SYSTEM32\\ntdll.dll",
			expected: "ntdll",
		},
		{
			pkg:      "C:\\Program Files\\Foo 2023.3\\bin\\foo.exe",
			expected: "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := trimPackage(tt.pkg)
			if result != tt.expected {
				t.Fatalf("Expected %s but got %s", tt.expected, result)
			}
		})
	}
}

func TestFullyQualifiedName(t *testing.T) {
	tests := []struct {
		name     string
		platform platform.Platform
		frame    Frame
		expected string
	}{
		{
			name:     "nodejs no package",
			platform: platform.Node,
			frame: Frame{
				Function: "run",
			},
			expected: "run",
		},
		{
			name:     "nodejs",
			platform: platform.Node,
			frame: Frame{
				Package:  "node:events",
				Function: "emit",
			},
			expected: "node:events.emit",
		},
		{
			name:     "android",
			platform: platform.Android,
			frame: Frame{
				Package:  "java.util",
				Function: "java.util.Arrays.copyOf(byte[], int): byte[]",
			},
			expected: "java.util.Arrays.copyOf(byte[], int): byte[]",
		},
		{
			name:     "java",
			platform: platform.Java,
			frame: Frame{
				Package:  "java.util",
				Function: "java.util.Arrays.copyOf(byte[], int): byte[]",
			},
			expected: "java.util.Arrays.copyOf(byte[], int): byte[]",
		},
		{
			name:     "cocoa",
			platform: platform.Cocoa,
			frame: Frame{
				Package:  "/private/var/containers/Bundle/Application/00000000-0000-0000-0000-000000000000/iOS-Swift.app/iOS-Swift",
				Function: "Controller.doWork()",
			},
			expected: "Controller.doWork()",
		},
		{
			name:     "python",
			platform: platform.Python,
			frame: Frame{
				Module:   "threading",
				Function: "Condition.wait",
			},
			expected: "threading.Condition.wait",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.frame.FullyQualifiedName(tt.platform)
			if result != tt.expected {
				t.Fatalf("Expected %s but got %s", tt.expected, result)
			}
		})
	}
}
