package main

import (
	"flag"
	"fmt"

	"github.com/Avik32223/redis-server/internal/redis"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":6379", "address to listen on. ex :6379")
	flag.Parse()

	s := redis.NewServer(addr)
	fmt.Println(s.Start())
}
