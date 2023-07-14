package engine

import (
	"fmt"
	"gout/context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Engine struct {
	methodTree  map[string]MethodList
	srv         *http.Server
	pool        sync.Pool
	middlewares context.HandlersChain //公有的处理函数链，用于存储中间件
}
type MethodList []method

type method struct {
	name     string
	path     string
	paramNum int
	handlers context.HandlersChain
}

// NewEngine 新建engine
func NewEngine() *Engine {
	engine := &Engine{
		srv: &http.Server{
			ReadTimeout:  20 * time.Second,
			WriteTimeout: 20 * time.Second,
		},
	}
	engine.pool.New = func() any {
		return &context.Context{Index: -1}
	}
	engine.methodTree = make(map[string]MethodList)
	engine.methodTree[http.MethodGet] = nil
	engine.methodTree[http.MethodPost] = nil
	engine.methodTree[http.MethodPut] = nil
	engine.methodTree[http.MethodDelete] = nil
	return engine
}
func (engine *Engine) GET(path string, handlers ...context.HandlerFunc) {
	engine.addRoute(http.MethodGet, path, handlers)
}
func (engine *Engine) POST(path string, handlers ...context.HandlerFunc) {
	engine.addRoute(http.MethodPost, path, handlers)
}
func (engine *Engine) PUT(path string, handlers ...context.HandlerFunc) {
	engine.addRoute(http.MethodPut, path, handlers)
}
func (engine *Engine) DELETE(path string, handlers ...context.HandlerFunc) {
	engine.addRoute(http.MethodDelete, path, handlers)
}

// 添加新的路由
func (engine *Engine) addRoute(methodName string, path string, handlers []context.HandlerFunc) {
	engine.methodTree[methodName] = append(engine.methodTree[methodName], method{
		name:     methodName,
		path:     path,
		paramNum: 0,
		handlers: handlers,
	})
	fmt.Println("[Gout]-- " + methodName + "  " + " " + path)
}

// Run 启动
func (engine *Engine) Run(addr string) {
	engine.srv.Addr = addr
	engine.srv.Handler = engine
	if err := engine.srv.ListenAndServe(); err != nil {
		panic("启动路由失败")
	}
}

// Engine的ServeHTTP 主要作用是从pool中获取上下文并将请求数据传入
func (engine *Engine) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*context.Context)
	c.Writer = rw
	c.Request = req
	engine.handleHTTPRequest(c)
	engine.pool.Put(c)

}

// 对请求进行处理，主要进行路由的匹配、query，表单数据的处理和处理函数的运行
func (engine *Engine) handleHTTPRequest(c *context.Context) {

	if c.Request.Method == http.MethodGet {
		c.QueryCache = c.Request.URL.Query()
	}
	if c.Request.Method == http.MethodPost {
		err := c.Request.ParseForm()
		if err != nil {
			panic("application/x-www-form-urlencoded解析失败" + err.Error())
		}
		c.FormCache = c.Request.Form
		err = c.Request.ParseMultipartForm(32 << 20)
		if err != nil {
			panic("multipart/form-data解析失败" + err.Error())
		}
		c.MultipartFormCache = c.Request.MultipartForm
	}

	methodName := c.Request.Method
	//匹配路由
	for _, v := range engine.methodTree[methodName] {
		if v.path == c.Request.URL.Path {
			c.IsMatch = true
			handlers := engine.combineHandlers(v.handlers)
			c.Handlers = handlers
			c.Next()
			return
		}
	}
	//未匹配成功
	for _, handler := range engine.middlewares {
		c.IsMatch = false
		handler(c)
	}
	c.Write("路由未匹配成功")
}
func (engine *Engine) combineHandlers(handlers context.HandlersChain) context.HandlersChain {
	finalSize := len(engine.middlewares) + len(handlers)
	mergedHandlers := make(context.HandlersChain, finalSize)
	copy(mergedHandlers, engine.middlewares)
	copy(mergedHandlers[len(engine.middlewares):], handlers)
	return mergedHandlers
}

// AddMiddleware 添加中间件
func (engine *Engine) AddMiddleware(handler ...context.HandlerFunc) {
	engine.middlewares = append(engine.middlewares, handler...)
}
func Logger() context.HandlerFunc {
	return func(c *context.Context) {
		code := 404
		if c.IsMatch {
			code = 200
		}
		log.Printf("[GOUT] |%v|  |%v|  \"%6v\" |%6v|", code, c.Request.Method, c.Request.URL.Path, c.Request.Header.Get("Content-Length"))
	}
}
func Recovery() context.HandlerFunc {
	return func(c *context.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 生成 HTTP 500 错误响应
				c.JSON(http.StatusInternalServerError, context.H{
					"error": "Internal Server Error",
				})
				// 打印 panic 的详细信息
				log.Println("Panic:", err)
			}
		}()
	}
}
func CORS() context.HandlerFunc {
	return func(c *context.Context) {
		method := c.Request.Method               //请求方法
		origin := c.Request.Header.Get("Origin") //请求头部
		var headerKeys []string                  // 声明请求头keys
		for k := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			// 这是允许访问所有域
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			//服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
			// 允许跨域设置
			// 可以返回其他子段
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			// 跨域关键设置 让浏览器可以解析
			c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar")
			// 缓存请求信息 单位为秒
			c.Writer.Header().Set("Access-Control-Max-Age", "172800")
			//  跨域请求是否需要带cookie信息 默认设置为true
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "false")

		}
		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "Options Request!")
		}
	}
}
