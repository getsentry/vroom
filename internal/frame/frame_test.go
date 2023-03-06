package frame

import (
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
