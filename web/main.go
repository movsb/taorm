package main

import "github.com/gin-gonic/gin"

var listen = ":3676" // ".orm"

func main() {
	router := gin.Default()
	router.Static("/", "root")
	if err := router.Run(listen); err != nil {
		panic(err)
	}
}
