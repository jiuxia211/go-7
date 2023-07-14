# 基于net/http库的Gout框架

时间有点不够，7号合作了结束然后开始军训，然后每天就是早训->躺尸一下午->晚训->看代码看到昏迷

gin的路由树的构建和匹配没有完全搞懂，特别是统配符部分，15、16号有个小周末看看能不能搞明白重写一遍

### 实现思路

1.在engine中存一个`http.Server`，直接调用`func (srv *Server) ListenAndServe(`)，来启动服务

2.直接使用`map[string]MethodList` map+切片的形式存储路由，key设置为GET、POST、PUT和DELETE，接收到请求后，直接遍历`MethodList`找到相同的路径。(因为gin的路由树没搞懂，所以也没有弄通配符的解析,就是`param`)

3.和gin一样，在engine中弄一个Pool存context的对象池,处理请求直接用context里的数据

4.支持中间件、解释query，form-data，x-www-form-urlencoded数据(直接用net/http库的方法)



net/http库的学习笔记也上传一下(gin 其实也写了，卡在路由树的通配符那边)

