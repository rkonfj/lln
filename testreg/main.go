package main

import (
	"fmt"
	"regexp"
)

var imageRegex = regexp.MustCompile(`\[img\](https://[^\s\[\]]+)\[/img\]`)

func main() {
	r := "[img]https://baidu.com?dsds=dsds.jpg[/img]"

	ms := imageRegex.FindAllStringSubmatch(r, -1)
	fmt.Println(ms)
}
