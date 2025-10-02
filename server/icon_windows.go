//go:build windows

package server

import (
	_ "embed"
)

//go:embed icon.ico
var iconData []byte
