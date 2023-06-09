package tools

import (
	"net/http"
	"strconv"
	"strings"
)

// PaginationOptions options for pagination
type PaginationOptions struct {
	After, Size int64
	Ascend      bool
}

// Unique distinct slice
func Unique(slice []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for _, v := range slice {
		if !encountered[v] {
			encountered[v] = true
			result = append(result, v)
		}
	}

	return result
}

// Contains 函数用于判断一个字符串切片中是否存在某个字符串
func Contains(strSlice []string, searchString string) bool {
	for _, str := range strSlice {
		if strings.Contains(str, searchString) {
			return true
		}
	}
	return false
}

func Reverse[T any](slice []T) {
	length := len(slice)
	for i := 0; i < length/2; i++ {
		j := length - i - 1
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func URLQueryInt64Default(r *http.Request, key string, defaultValue int64) (value int64, err error) {
	value = defaultValue
	str := r.URL.Query().Get(key)
	if len(str) > 0 {
		value, err = strconv.ParseInt(str, 10, 64)
	}
	return
}

func URLQueryInt64(r *http.Request, key string) (value int64, err error) {
	return URLQueryInt64Default(r, key, 0)
}

func URLPaginationOptions(r *http.Request) (*PaginationOptions, error) {
	size, err := URLQueryInt64Default(r, "size", 20)
	if err != nil {
		return nil, err
	}
	after, err := URLQueryInt64(r, "after")
	if err != nil {
		return nil, err
	}
	return &PaginationOptions{
		After: after, Size: size, Ascend: r.URL.Query().Get("order") == "asc",
	}, nil
}
