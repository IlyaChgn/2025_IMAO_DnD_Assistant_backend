package utils

import (
	"runtime"
	"strings"
)

func GetFunctionName() string {
	pc, _, _, _ := runtime.Caller(1)

	fullName := runtime.FuncForPC(pc).Name()
	fullNameArr := strings.Split(fullName, ".")

	return fullNameArr[len(fullNameArr)-1]
}
