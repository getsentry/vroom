package frame

import (
	"hash/fnv"
	"testing"
)

func frameType(isApplication bool) string {
	if isApplication {
		return "application"
	}
	return "system"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isApplication := tt.frame.IsPythonApplicationFrame(); isApplication != tt.isApplication {
				t.Fatalf("Expected %s frame but got %s frame", frameType(tt.isApplication), frameType(isApplication))
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
				t.Fatalf("Expected %s frame but got %s frame", frameType(tt.isApplication), frameType(isApplication))
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
				t.Fatalf("Expected %s frame but got %s frame", frameType(tt.isApplication), frameType(isApplication))
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
			name:  "empty frame",
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
