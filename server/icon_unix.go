//go:build !windows

package server

import (
	_ "embed"
)

//go:embed icon.png
var iconData []byte
