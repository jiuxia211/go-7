package main

import (
	"gout/context"
	"gout/engine"
)

func main() {
	r := engine.NewEngine()
	r.AddMiddleware(engine.Recovery(), engine.Logger(), engine.CORS())
	//  http://localhost:8080/ping?name=John&age=25
	r.GET("/ping", func(c *context.Context) {
		c.Write("test  " + c.Query("name"))
	})
	r.GET("/recover", func(c *context.Context) {
		panic("recover测试")
	})
	r.POST("/login", func(c *context.Context) {
		c.JSON(200, context.H{
			"account":  c.PostForm("account"),
			"password": c.PostForm("password"),
		})
	})
	r.PUT("/put/test", func(c *context.Context) {
		c.JSON(200, context.H{
			"msg": "put msg",
		})
	})
	r.DELETE("/delete/test", func(c *context.Context) {
		c.JSON(200, context.H{
			"msg": "delete msg",
		})
	})

	r.Run(":8080")

}
