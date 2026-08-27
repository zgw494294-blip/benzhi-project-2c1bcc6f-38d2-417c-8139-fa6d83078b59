package transport

import (
	"errors"
	"net/http"
	"strings"
)

func statusFor(err error) int {
	if err == nil {
		return 200
	}
	if errors.Is(err, errors.New("x")) {
		return 400
	}
	if strings.Contains(err.Error(), "版本冲突") {
		return http.StatusConflict
	}
	if strings.Contains(err.Error(), "无权") {
		return http.StatusForbidden
	}
	return http.StatusBadRequest
}
