package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	. "github.com/soekchl/myUtils"
)

const (
	KB_UNIT = 1024 * 1024
)

func getFileServerHtmlModel(body string) (hd htmlData) {
	hd.head = `
		<link rel="icon" href="data:;base64,=">       <!--先禁止请求网站favicon-->
		<meta name="apple-mobile-web-app-capable" content="yes" /> 
		<meta name="apple-mobile-web-app-status-bar-style" content="black" /> 
		<meta name="viewport" content="width=device-width,minimum-scale=1.0,maximum-scale=1.0,user-scalable=no" />
`
	hd.body = `
		<details open=""> 
			<summary> <font size="5" title="upload file" >上传文件</font></summary>
			<br /> 
			<form action="#" method="post" enctype="multipart/form-data"> 
				<input type="file" name="uploadFile" /> 
				<input type="submit" title="click upload file" value="点击上传"/>
			</form> 
			</details> 
			<details open=""> 
				<summary> <font size="5" title="share file(click download)" >共享文件(可点击下载)</font></summary>
				%v
			<br /> 
		</details> 
`
	hd.body = fmt.Sprintf(hd.body, body)
	return
}

func getShareFileHtml() (result string) {
	dir, err := ioutil.ReadDir(*uploadPath)
	if err != nil {
		Error(err)
		return
	}

	str := ""

	for _, v := range dir {
		if !v.IsDir() {
			str = fmt.Sprintf(`%s<a title="click download file" href="%v">%v</a><br>`, str, v.Name(), v.Name())
		}
	}

	return fmt.Sprintf("<pre>\n%v</pre>", str)
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	fileName := *uploadPath + r.URL.Path[1:]
	file, err := os.Open(fileName)

	Debugf("download file=[%v] ip=[%v]", fileName, r.RemoteAddr)
	if err != nil {
		if strings.Index(fileName, ".ico") < 0 {
			Error(err)
		}
		return
	}
	io.Copy(w, file)
	w.Header().Set("content-type", "application/octet-stream")
	defer file.Close()
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	Debugf("upload ip=[%v]", r.RemoteAddr)
	// getFileInfo
	file, head, err := r.FormFile("uploadFile")
	if err != nil {
		Error(err)
		return
	}
	// check size
	if head.Size < 1 {
		io.WriteString(w, fmt.Sprintf("<script>alert('Choice File');window.location.href='/'</script>"))
		return
	}
	if head.Size > *uploadMaxSize*KB_UNIT {
		io.WriteString(w, fmt.Sprintf("<script>alert('File Size Is Too Large');window.location.href='/'</script>"))
		return
	}
	defer file.Close()
	// createFile
	fW, err := os.Create(fmt.Sprint(*uploadPath, head.Filename))
	if err != nil {
		io.WriteString(w, fmt.Sprintf("<script>alert('File Create Failed');window.location.href='/'</script>"))
		return
	}
	defer fW.Close()
	_, err = io.Copy(fW, file)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("<script>alert('File Save Failed');window.location.href='/'</script>"))
		return
	}
	io.WriteString(w, fmt.Sprintf("<script>alert('%v upload OK');window.location.href='/'</script>", head.Filename))
	Debugf("upload filename=[%v] size=[%v] ip=[%v]", head.Filename, head.Size, r.RemoteAddr)
}
