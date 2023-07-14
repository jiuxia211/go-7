# go net/http 库学习

先看一个例子,运行后我们可以访问localhost:8080 ，看到页面显示Hello World

```go
package main

import (
	"fmt"
	"net/http"
)

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World")
}

func main() {
	http.HandleFunc("/", index)
	http.ListenAndServe(":8080", nil)
}
```

##### http.HandleFunc

```go
func HandleFunc(pattern string, handler func (ResponseWriter, *Request))
```

第一个参数是路径 第二个参数传入一个函数

路径就不用多说了，来看第二个处理函数   `handler func (ResponseWriter, *Request)`

`ResponseWriter`是一个接口，该接口有三个方法

```go
Header() Header
	
Write([]byte) (int, error)

WriteHeader(statusCode int)
```

这三个接口的作用我们用一个handler函数来理解

```go
func handler(w http.ResponseWriter, r *http.Request) {
	// 设置响应头部的 Content-Type 字段
    //Header还有Add Set Get Del等方法
	w.Header().Set("Content-Type", "text/plain")

	// 设置响应状态码为 200 OK
	w.WriteHeader(http.StatusOK)

	// 写入响应体内容
	fmt.Fprintf(w, "Hello, World!")
}
```

不过实际上即使不加Content-Type 字段和响应状态码为 200，响应头默认都会有这两个内容

默认设置的响应头:

![image-20230702132602358](E:\Users\FA\Desktop\net_http\image-20230702132602358.png)

接下来看Request   

Request的结构体，包含了Request的所有信息，稍微看看就好

```go

type Request struct {
	Method string  //表示请求的 HTTP 方法，如 GET、POST、PUT、DELETE 等。
    URL *url.URL  //表示请求的 URL 对象
	Proto      string // 表示请求的协议版本，如"HTTP/1.0"
	ProtoMajor int    // 1
	ProtoMinor int    // 0
	Header Header     //请求头
	Body io.ReadCloser  //表示请求的主体，通常用于包含 POST 请求的数据。
	GetBody func() (io.ReadCloser, error) 
    ContentLength int64
	TransferEncoding []string
	Close bool
	Host string
	Form url.Values  //表示解析后的表单数据，如果请求是以 application/x-www-form-urlencoded 或 multipart/form-data 格式发送的。
	PostForm url.Values  //表示解析后的 POST 表单数据，仅在请求是以 application/x-www-form-urlencoded 格式发送的时候有效。
	MultipartForm *multipart.Form //表示解析后的多部分表单数据，仅在请求是以 multipart/form-data 格式发送的时候有效。
	Trailer Header
	RemoteAddr string  //表示发送请求的客户端的网络地址。
	RequestURI string
	TLS *tls.ConnectionState
	Cancel <-chan struct{}
	Response *Response
	ctx context.Context
}
```

然后我就可以看HandleFunc究竟做了什么

```go
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}
```

我们发现它直接调用了一个名为`DefaultServeMux`对象的`HandleFunc()`方法。

通过以下代码，我们可以看到`DefaultServeMux`是`ServeMux`类型的默认实例：

```go
type ServeMux struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	es    []muxEntry // slice of entries sorted from longest to shortest.
	hosts bool       // whether any patterns contain hostnames
}
type muxEntry struct {
	h       Handler
	pattern string
}
// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = &defaultServeMux

var defaultServeMux ServeMux
```

`ServeMux.HandleFunc()`方法如下

```go
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler))
}
```

很好，这个方法只是对handler进行的判空，然后又调用了`ServeMux.Handle()`方法

注意这里的`HandlerFunc(handler)`是类型转换,这是为了实现`HandlerFunc`的`ServeHTTP`方法

类型`HandlerFunc`的定义和`ServeHTTP`方法如下：

```go
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}
```

`ServeHTTP`方法不就是运行了一遍`HandlerFunc`嘛！

接下来我们看`ServeMux.Handle()`做了什么

```go
func (mux *ServeMux) Handle(pattern string, handler Handler) {
    //通过互斥锁对 ServeMux 进行加锁，保证并发安全性。
	mux.mu.Lock()
	defer mux.mu.Unlock()
    //处理路径是否为空
	if pattern == "" {
		panic("http: invalid pattern")
	}
    //处理handler处理器函数是否为空
	if handler == nil {
		panic("http: nil handler")
	}
    //检查是否存在相同路径
	if _, exist := mux.m[pattern]; exist {
		panic("http: multiple registrations for " + pattern)
	}
    //检查mux.m（string-muxEntry类型的map）是否为空
	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}
    //将路径和对应的muxEntry存储到m中
	e := muxEntry{h: handler, pattern: pattern}
	mux.m[pattern] = e
    //如果路径最后一个字符为'/'加入es（muxEntry切片）并从长到短进行排序
	if pattern[len(pattern)-1] == '/' {
		mux.es = appendSorted(mux.es, e)
	}
    //如果路径第一个字符不为'/' 则将 mux.hosts 设置为 true，表示存在主机名相关的路由规则。
	if pattern[0] != '/' {
		mux.hosts = true
	}
}

```

注意一个点，我们在`ServeMux.HandleFunc()`方法传入的是一个`HandlerFunc`类型的参数handler,但是

`ServeMux.Handle()`里的handler却是一个`Handler`类型，这是什么意思呢？让我们来看看`Handler`类型是个什么玩意

```go
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

```

`Handler` 是一个接口，它有一个`ServeHTTP`方法，go语言可以用接口作为参数，而我们需要传入一个实现该接口的type，等等，`HandlerFunc`类型不就实现了这个方法吗？一切都变得合理了起来	

总结:`http.HandleFunc` 将传入的路径和处理函数储存到`DefaultServeMux`（默认的ServeMux）中

##### http.ListenAndServe

```
http.ListenAndServe(":8080", nil)
```

点击进入该函数

```go
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

```

该函数定义了一个`Server`，让我们看看`Server`有哪些参数

```go
type Server struct {
	Addr string

	Handler Handler 

	TLSConfig *tls.Config

	ReadTimeout time.Duration

	ReadHeaderTimeout time.Duration

	WriteTimeout time.Duration

	IdleTimeout time.Duration

	MaxHeaderBytes int

	TLSNextProto map[string]func(*Server, *tls.Conn, Handler)

	ConnState func(net.Conn, ConnState)

	ErrorLog *log.Logger
	
	BaseContext func(net.Listener) context.Context
	
	ConnContext func(ctx context.Context, c net.Conn) context.Context

	inShutdown atomicBool 

	disableKeepAlives int32     
	nextProtoOnce     sync.Once 
	nextProtoErr      error    
    
	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	activeConn map[*conn]struct{}
	doneChan   chan struct{}
	onShutdown []func()

	listenerGroup sync.WaitGroup
}
```

好吧，真让人眼花缭乱，但是第二个变量似乎很熟悉，`Handler` ！，前面我们已经见过它了，`Handler` 是一个接口，它有一个`ServeHTTP`方法。

然后`ListenAndServe`还执行了`server.ListenAndServe()`并返回错误，让我们看看它吧

```go
func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}
```

可以看到`server.ListenAndServe()`先对Server是否关闭进行了判断，如果addr为空，则设为":http"（经过实测，如果addr为:http，访问localhost和localhost:80都是一样的）,然后通过`net.Listen`获得了一个叫做ln的`Listener`。老样子，先看看`Listener`是个什么鬼

```go
type Listener interface {
	Accept() (Conn, error)

	Close() error

	Addr() Addr
}
type Conn interface {
	Read(b []byte) (n int, err error)
.
	Write(b []byte) (n int, err error)

	Close() error

	LocalAddr() Addr

	RemoteAddr() Addr

	SetDeadline(t time.Time) error

	SetReadDeadline(t time.Time) error

	SetWriteDeadline(t time.Time) error
}
type Addr interface {
	Network() string 
	String() string  
}
```

可以看到`Listener`有三个方法

`Accept()`方法用于接受传入的连接，返回一个实现了`Conn`接口的连接实例和可能的错误。

`Close()`方法用于关闭监听器。

`Addr()`方法用于返回监听器的网络地址。

至于`net.Listen`具体逻辑是什么，里面涉及地址的解析，笔者网络知识不够，我们这里先跳过，先记住`Listener`的三个方法的作用

对`net.Listen`的错误进行处理后，`server.ListenAndServe()`继续执行了`srv.Serve(ln)` 

```go
func (srv *Server) Serve(l net.Listener) error {
	if fn := testHookServerServe; fn != nil {
		fn(srv, l) // call hook with unwrapped listener
	}

	origListener := l
	l = &onceCloseListener{Listener: l}
	defer l.Close()

	if err := srv.setupHTTP2_Serve(); err != nil {
		return err
	}

	if !srv.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer srv.trackListener(&l, false)

	baseCtx := context.Background()
	if srv.BaseContext != nil {
		baseCtx = srv.BaseContext(origListener)
		if baseCtx == nil {
			panic("BaseContext returned a nil context")
		}
	}

	var tempDelay time.Duration // how long to sleep on accept failure

	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	for {
		rw, err := l.Accept()
		if err != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("http: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		connCtx := ctx
		if cc := srv.ConnContext; cc != nil {
			connCtx = cc(connCtx, rw)
			if connCtx == nil {
				panic("ConnContext returned nil")
			}
		}
		tempDelay = 0
		c := srv.newConn(rw)
		c.setState(c.rwc, StateNew, runHooks) // before Serve can return
		go c.serve(connCtx)
	}
}
```

这个方法的关键在于一个无限的for循环,不停地调用`Listener.Accept()`方法接受新连接,开启新 goroutine 处理新连接.

这里有一个指数退避策略的用法。如果l.Accept()调用返回错误，我们判断该错误是不是临时性地（ne.Temporary()）。如果是临时性错误，Sleep一小段时间后重试，每发生一次临时性错误，Sleep的时间翻倍，最多Sleep 1s。获得新连接后，将其封装成一个conn对象（srv.newConn(rw)），创建一个 goroutine 运行其serve()方法。省略无关逻辑的代码如下：

```go
func (c *conn) serve(ctx context.Context) {
  for {
    w, err := c.readRequest(ctx)
    serverHandler{c.server}.ServeHTTP(w, w.req)
    w.finishRequest()
  }
}
```

`c.readRequest`对请求进行的解析,返回的`w`是`response`类型，它实现了`ResponseWriter`的三个方法，`w.req` 是`Request`类型

`serverHandler`是一个中间辅助结构，它实现了`ServeHTTP`方法

```go
type serverHandler struct {
  srv *Server
}
 
func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
  handler := sh.srv.Handler
  if handler == nil {
    handler = DefaultServeMux
  }
  handler.ServeHTTP(rw, req)	
}
```

`ServeHTTP`先从Server里获取`Handler`,再执行它的`ServeHTTP`方法，在最开始的示例中我们传入的Hander为空,所以这里`handler`会取默认值`DefaultServeMux`，但是`DefaultServeMux`是在哪实现的`Handler`接口呢，我使用了搜索大法找到了它的实现

```go
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}
```

该方法调用了`mux.Handler(r)` 方法，根据请求的路径查找相应的`Handler`，然后调用`h.ServeHTTP(w, r)`，

`mux.Handler(r)` 的实现如下(省略部分代码)

```go
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {
  host := stripHostPort(r.Host)
  return mux.handler(host, r.URL.Path)
}
 
func (mux *ServeMux) handler(host, path string) (h Handler, pattern string) {
  h, pattern = mux.match(path)
  return
}
 
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
  v, ok := mux.m[path]
  if ok {
    return v.h, v.pattern
  }
 
  for _, e := range mux.es {
    if strings.HasPrefix(path, e.pattern) {
      return e.h, e.pattern
    }
  }
  return nil, ""
}
```

在`match`方法中，首先会检查路径是否精确匹配`mux.m[path]`。如果不能精确匹配，后面的`for`循环会匹配路径的最长前缀。**只要注册了`/`根路径处理，所有未匹配到的路径最终都会交给`/`路径处理**。为了保证最长前缀优先，在注册时，会对路径进行排序。

还记得之前的`ServeMux.Handle()`方法吗，里面有如下逻辑

```go
   //如果路径最后一个字符为'/'加入es（muxEntry切片）并从长到短进行排序
	if pattern[len(pattern)-1] == '/' {
		mux.es = appendSorted(mux.es, e)
	}
```

也就是说，只有传入的路径结尾为'/'，才会加入`ServeMux`的es切片，才会去匹配最长逻辑。

举个例子，假如你传入`/greeting`，然后再访问`localhost:8080/greeting/a/b/c`和`localhost:8080/a/b/c`只会匹配到"/路径"，如果想要让`localhost:8080/greeting/a/b/c`匹配路径`/greeting`，注册路径需要改为`/greeting/`

再回到`serverHandler`的`ServeHTTP`方法

如果我们在开始时传入`Handler`，那就直接调用它的`ServeHTTP`方法。这是你只要访问localhost:8080，你在后面加入/xxxxx都只会调用`Handler`的`ServeHTTP`方法

总结：`http.ListenAndServe`用传入的端口号和处理函数创建一个`Server`,并用`Server`的`ListenAndServe`方法监听端口，接收新连接并用处理函数处理（如果处理函数为空，则使用`DefaultServeMux`（默认的ServeMux）来根据路径匹配处理函数并处理）

我们也可以创建自己的`ServeMux`来创建`Server`，用`ServeMux`对象初始化`Server`的`Handler`字段，最后调用`Server.ListenAndServe()`方法开启 Web 服务

```go
func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/", index)
  mux.Handle("/greeting", greeting("Welcome to go web frameworks"))
 
  server := &http.Server{
    Addr:         ":8080",
    Handler:      mux,
    ReadTimeout:  20 * time.Second,
    WriteTimeout: 20 * time.Second,
  }
  server.ListenAndServe()
}
```



##### 中间件

有时候需要在请求处理代码中增加一些通用的逻辑，如统计处理耗时、记录日志、捕获宕机等等。如果在每个请求处理函数中添加这些逻辑，代码很快就会变得不可维护，添加新的处理函数也会变得非常繁琐。所以就有了中间件的需求。

首先，基于函数类型`func(http.Handler) http.Handler`定义一个中间件类型：

```
type Middleware func(http.Handler) http.Handler
```

然后我们来写3个中间件，注意这三个中间件有细微的差别，但本质是一样的，因为`http.HandlerFunc`就是实现了`http.Handler`接口的一个类型

```go
//请求前后各输出一条日志
func WithLogger(handler http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    logger.Printf("path:%s process start...\n", r.URL.Path)
    defer func() {
      logger.Printf("path:%s process end...\n", r.URL.Path)
    }()
    handler.ServeHTTP(w, r)
  })
}
//统计处理耗时
func Metric(handler http.Handler) http.HandlerFunc {
  return func (w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    defer func() {
      logger.Printf("path:%s elapsed:%fs\n", r.URL.Path, time.Since(start).Seconds())
    }()
    time.Sleep(1 * time.Second)
    handler.ServeHTTP(w, r)
  }
}
//捕获可能出现的 panic
func PanicRecover(handler http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    defer func() {
      if err := recover(); err != nil {
        logger.Println(string(debug.Stack()))
      }
    }()
 
    handler.ServeHTTP(w, r)
  })
}
```

然后我们就可以这样处理

```go
mux.Handle("/", PanicRecover(WithLogger(Metric(http.HandlerFunc(index)))))
```

嵌套嵌套再嵌套。。我们来理一理逻辑

首先，经过前面的学习，我们知道`http.HandlerFunc`是一个函数类型，它的`ServeHTTP`方法就是调用其本身

从最内部开始看，它先把`index`转成`http.HandlerFunc`类型，再把它传给`Metric`,`Metric`返回了一个`http.HandlerFunc`,我们称它为M，M在请求前后各输出一条日志，然后调用了传入的`index`的`ServeHTTP`方法。接着`WithLogger`接收了M，又返回了一个`http.HandlerFunc`函数，我们称它为W，W统计处理了耗时.然后调用了M的`ServeHTTP`方法,最后`PanicRecover`接收了W，还是返回一个`http.HandlerFunc`函数，称为P，P函数捕获可能出现的 `panic`，然后调用了W的ServeHTTP方法，返回的P就是最终的`Handler`。

我相信你可能还是一脸懵，我们来理一理当我们对它发起请求时，它会怎么做。

这时就从外往里看，`PanicRecover`的`ServeHTTP`方法先执行，就是执行`PanicRecover`本身，在写入捕获 `panic`逻辑之后，调用了`WithLogger`的`ServeHTTP`执行`WithLogger`本身，接着写入统计处理耗时的逻辑后调用了`Metric`的`ServeHTTP`方法执行`Metric`本身，写入请求前后各输出一条日志设为逻辑后执行`index`的`ServeHTTP`方法执行`index`，请求的处理就结束了。

当然，一直嵌套的写法非常丑陋，我们可以写一个帮助函数

```go
func applyMiddlewares(handler http.Handler, middlewares ...Middleware) http.Handler {
  for i := len(middlewares)-1; i >= 0; i-- {
    handler = middlewares[i](handler)
  }
 
  return handler
}
```

使用帮助函数之后我们可以简化注册

```go
middlewares := []Middleware{
  PanicRecover,
  WithLogger,
  Metric,
}
mux.Handle("/", applyMiddlewares(http.HandlerFunc(index), middlewares...))
```

上面每次注册处理逻辑都需要调用一次applyMiddlewares()函数，还是略显繁琐。我们可以这样来优化，封装一个自己的ServeMux结构，然后定义一个方法Use()将中间件保存下来，重写Handle/HandleFunc将传入的http.HandlerFunc/http.Handler处理器包装中间件之后再传给底层的ServeMux.Handle()方法：

```go
type MyMux struct {
  *http.ServeMux
  middlewares []Middleware
}
 
func NewMyMux() *MyMux {
  return &MyMux{
    ServeMux: http.NewServeMux(),
  }
}
 
func (m *MyMux) Use(middlewares ...Middleware) {
  m.middlewares = append(m.middlewares, middlewares...)
}
 
func (m *MyMux) Handle(pattern string, handler http.Handler) {
  handler = applyMiddlewares(handler, m.middlewares...)
  m.ServeMux.Handle(pattern, handler)
}
 
func (m *MyMux) HandleFunc(pattern string, handler http.HandlerFunc) {
  newHandler := applyMiddlewares(handler, m.middlewares...)
  m.ServeMux.Handle(pattern, newHandler)
}
```

注册时只需要创建`MyMux`对象，调用其`Use()`方法传入要应用的中间件即可：

```go
middlewares := []Middleware{
  PanicRecover,
  WithLogger,
  Metric,
}
mux := NewMyMux()
mux.Use(middlewares...)
mux.HandleFunc("/", index)
mux.Handle("/greeting", greeting("welcome, dj"))
```

这种方式简单易用，但是也有它的问题，最大的问题是必须先设置好中间件，然后才能调用Handle/HandleFunc注册，后添加的中间件不会对之前注册的处理器/函数生效。

为了解决这个问题，我们可以改写ServeHTTP方法，在确定了处理器之后再应用中间件。这样后续添加的中间件也能生效。很多第三方库都是采用这种方式。改造这个方法定义`MyMux`类型的`ServeHTTP()`方法如下：

```go
func (m *MyMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  // ...
  h, _ := m.Handler(r)
  // 只需要加这一行即可
  h = applyMiddlewares(h, m.middlewares...)
  h.ServeHTTP(w, r)
}
```

本文的例子来源 http://t.csdn.cn/bNlxQ

