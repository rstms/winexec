package cmd

import (
	"github.com/rstms/go-common"
)

func ViperGetString(key string) string {
	return common.ViperGetString(key)
}

func ViperGetInt(key string) int {
	return common.ViperGetInt(key)
}

func ViperGetBool(key string) bool {
	return common.ViperGetBool(key)
}
