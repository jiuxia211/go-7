package context

import (
	"encoding/json"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
)

type H map[string]any
type Context struct {
	Request            *http.Request
	Writer             http.ResponseWriter
	QueryCache         url.Values
	FormCache          url.Values
	MultipartFormCache *multipart.Form
	IsMatch            bool
	Handlers           HandlersChain
	Index              int8
}
type HandlerFunc func(c *Context)

type HandlersChain []HandlerFunc

func (c *Context) Write(msg string) {
	fmt.Fprintf(c.Writer, msg)
}
func (c *Context) Query(key string) (value string) {
	value = c.QueryCache.Get(key)
	return value
}
func (c *Context) PostForm(key string) (value string) {
	value = c.FormCache.Get(key)
	if value == "" {
		value = c.MultipartFormCache.Value[key][0]
	}
	return value
}
func (c *Context) JSON(code int, obj any) {
	c.Writer.WriteHeader(code)
	jsonData, err := json.Marshal(obj)
	if err != nil {
		panic("数据转化成JSON格式失败")
	}
	_, err = c.Writer.Write(jsonData)
	if err != nil {
		panic("写入JSON数据失败")
	}

}

const abortIndex int8 = math.MaxInt8 / 2

func (c *Context) Next() {
	c.Index++
	for c.Index < int8(len(c.Handlers)) {
		c.Handlers[c.Index](c)
		c.Index++
	}
}

// Abort 终止上下文
func (c *Context) Abort() {
	c.Index = abortIndex
}
