package main

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

func main() {
	p := new(v1.Pod)
	fmt.Println(p)
}
