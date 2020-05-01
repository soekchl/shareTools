package main

import (
	"flag"
	"io"
	"net/http"

	. "github.com/soekchl/myUtils"
	"github.com/soekchl/websocket"
)

var (
	ip            = flag.String("ip", "", "Ip Address")                                      // 默认获取ip
	port          = flag.String("port", ":80", "ServerPort")                                 // 默认端口
	uploadPath    = flag.String("uploadDir", "./", "Upload Dir")                             // 目录
	uploadMaxSize = flag.Int64("uploadMaxSize", 100, "Upload File Max Size Default Unit MB") // 10MB
)

func init() {
	flag.Parse()
}

func main() {
	http.HandleFunc("/", index)
	http.Handle("/webSocket", websocket.Handler(webSocket))

	localIp := GetIp()
	addr := *ip + *port
	if *ip == "" {
		ip = &localIp
	}
	Warnf("Start Server IP--[%v]  Port--[%v]  Share Path--[%v]  Max Upload Size--[%v MB]",
		*ip,
		*port,
		*uploadPath,
		*uploadMaxSize,
	)
	Error(http.ListenAndServe(addr, nil))
}

func index(w http.ResponseWriter, r *http.Request) {
	//从请求当中判断方法
	if r.Method == "GET" {
		if len(r.URL.Path) == 1 {
			Debugf("index ip=%v", r.RemoteAddr)
			var hd []htmlData
			hd = append(hd, getFileServerHtmlModel(getShareFileHtml()))
			hd = append(hd, getSharePaperHtml(*ip+*port))
			io.WriteString(w, getRenderHtml("本地共享工具", hd))
		} else {
			downloadFile(w, r)
		}
	} else {
		uploadFile(w, r)
	}
}
