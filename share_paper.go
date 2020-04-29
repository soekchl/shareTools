package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	. "github.com/soekchl/myUtils"
	"github.com/soekchl/websocket"
)

/*
	---Cmd List---
	server->client
		1-init
		2-edit paper
		3-online count
		4-lock paper
		5-unlock paper
	client->server
		2-edit paper
		4-lock paper
		5-unlock paper
*/
type Message struct {
	Cmd   int    `json:"cmd"`
	Index int    `json:"index"`
	Data  string `json:"data"`
	ws    *websocket.Conn
}

type Paper struct {
	Data  string    `json:"data"` // paper data save momery
	Lock  bool      `json:"lock"` // 是否锁定
	id    int       // paper id - key
	mTime time.Time // last edit time
	ip    string    // last edit ip
}

//全局信息
var (
	users         []*websocket.Conn
	allCount      = 0
	userLock      = make(map[int]*Paper) // 一个连接同时只能锁定一个 连接断开时解锁
	paperMap      = make(map[int]*Paper)
	paperMapMutex sync.RWMutex
	sendMsg       chan *Message
)

func init() {
	sendMsg = make(chan *Message, 10)
	go sendServer()
}

func webSocket(ws *websocket.Conn) {
	Debugf("websocket ip=%v", ws.RemoteAddr().String())
	// save webSocket List
	index := addUsers(ws)
	changeOnline(1)
	sendInitData(ws)
	var err error
	// receive
	var buff string
	for {
		err = websocket.Message.Receive(ws, &buff)
		// Debug("data：", buff)
		if err != nil {
			//移除出错的链接
			break
		}

		msg := &Message{}
		err = json.Unmarshal([]byte(buff), msg)
		if err != nil {
			Errorf("解析数据异常... err=%v data=%v", err, buff)
			break
		}
		if msg.Cmd == 2 || msg.Cmd == 4 || msg.Cmd == 5 {
			msg.ws = ws
			// Debugf("cmd=%v id=%v", msg.Cmd, index)
			paperSet(msg, ws.RemoteAddr().String(), index)
			sendMsg <- msg
		}
	}
	//	close
	changeOnline(-1)
	clearPaper(index)
	users[index] = nil
}

func paperSet(msg *Message, ip string, index int) {
	paperMapMutex.Lock()
	defer paperMapMutex.Unlock()
	tmp, ok := paperMap[msg.Index]
	if !ok {
		tmp = &Paper{id: msg.Index}
	}
	if msg.Cmd == 2 && len(msg.Data) < 1 {
		delete(paperMap, msg.Index)
		return
	}
	if msg.Cmd == 2 {
		tmp.Data = msg.Data
	} else if msg.Cmd == 4 || msg.Cmd == 5 {
		tmp.Lock = msg.Cmd == 4
		clearPaper(index)
		userLock[index] = tmp
	}
	tmp.ip = ip
	tmp.mTime = time.Now()
	paperMap[msg.Index] = tmp
	// Warnf("%#v", tmp)
}

func clearPaper(index int) {
	lp, ok := userLock[index]
	if ok {
		lp.Lock = false // 解锁
		sendMsg <- &Message{Cmd: 5, Index: lp.id}
		delete(userLock, index)
	}
}

func sendServer() {
	var m *Message
	for m = range sendMsg {
		send(m)
	}
}

// send online count edit
func changeOnline(value int) {
	allCount += value
	Debugf("changeOnline online=%v value=%v", allCount, value)
	send(&Message{
		Cmd:  3,
		Data: fmt.Sprint(allCount),
	})
}

func sendInitData(ws *websocket.Conn) {
	paperMapMutex.RLock()
	defer paperMapMutex.RUnlock()

	mbuff, err := json.Marshal(&paperMap)
	if err != nil {
		Error(err)
		return
	}

	buff, err := json.Marshal(&Message{
		Cmd:  1,
		Data: string(mbuff),
	})
	if err != nil {
		Error(err)
		return
	}
	Debug(string(buff))
	websocket.Message.Send(ws, string(buff))
}

func send(msg *Message) {
	buff, err := json.Marshal(msg)
	if err != nil {
		Error(err)
		return
	}

	for _, k := range users {
		if k == nil {
			continue
		}
		if msg.ws == k { // not send me
			continue
		}
		err = websocket.Message.Send(k, string(buff))
		if err != nil {
			Error(err)
		}
	}
}

func addUsers(ws *websocket.Conn) int {
	for k, v := range users {
		if v == nil {
			users[k] = ws
			return k
		}
	}
	users = append(users, ws)
	return len(users) - 1
}

func getSharePaperHtml(address string) (hd htmlData) {
	hd.head = `
	<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
  <style>
    fieldset {
      display: block;
      margin-inline-start: 2px;
      margin-inline-end: 2px;
      padding-block-start: 0.35em;
      padding-inline-start: 0.75em;
      padding-inline-end: 0.75em;
      padding-block-end: 0.625em;
      min-inline-size: min-content;
      border-width: 2px;
      border-style: groove;
      border-color: threedface;
      border-image: initial;
    }

    legend {
      display: block;
      padding-inline-start: 2px;
      padding-inline-end: 2px;
      border-width: initial;
      border-style: none;
      border-color: initial;
      border-image: initial;
    }

    input.button {
      color: #fff;
      background-color: #409eff;
      border-color: #409eff;
      padding: 10px 15px;
      font-size: 18px;
      border-radius: 4px;
    }

    textarea {
      padding: 8px 8px;
    }
  </style>
`

	hd.body = `
<fieldset>
    <legend><b>
        <a href="https://gitee.com/soekchl/shareTools" title="Share Paper">
          <img width="20" title="Share Paper" src="https://www.easyicon.net/api/resizeApi.php?id=1109058&size=128"></a>
        共享记事本</b> <label for="" id="online">online </label></legend>
    <input class="button" type="button" id="add" value="Add"></input><br>
    <div id="texts">
    </div>
  </fieldset>
`

	hd.script = `
<script>
  add.onclick = addText
  let ws = null
  let textCount = 1
  let initFlag = true
  addText(0)
  conn()

  function addText(count, value = '', lockFlag = false) {
    if (typeof count === 'object') {
      count = textCount++
    } else if (textCount <= count) {
      textCount = count + 1
    }
    console.log('add', count)
    let tmp = document.createElement('textarea')
    tmp.id = "memory_"+count
    tmp.name = count
    tmp.style = "width:45%%; height: 100px; resize: none;"
    tmp.oninput = changeText
    tmp.placeholder = "share paper memory "+count
    tmp.onkeyup = checkDel
    tmp.disabled = lockFlag
    tmp.onfocus = lock
    tmp.onblur = unlock
    // tmp.onmouseover = console.log
    tmp.value = value
    texts.appendChild(tmp)
    // texts.children[texts.children.length - 1].focus()
  }
  function lock(e) {
    if (initFlag) {
      return
    }
    console.log(e.target.id, 'lock')
    // e.target.disabled = true
    sendData(4, e.target.name)
  }

  function unlock(e) {
    if (initFlag) {
      return
    }
    console.log(e.target.id, 'unlock')
    sendData(5, e.target.name)
  }

  function checkDel() {
    let x;
    if (window.event) // IE8 以及更早版本
    {
      x = event.keyCode;
    }
    else if (event.which) // IE9/Firefox/Chrome/Opera/Safari
    {
      x = event.which;
    }
    if (x === 8 && this.value.length < 1) {
      this.remove()
    }
  }

  function conn() {
    if (ws) {
      return
    }
    initFlag = true
     let wsUrl = "ws://%v/webSocket";
    ws = new WebSocket(wsUrl);
    try {
      ws.onopen = function () {
        console.log("open")
      }
      ws.onclose = function () {
        if (ws) {
          ws.close();
          ws = null;
        }
        console.log("close ws")
        checkReConn()
      }
      ws.onmessage = function (result) {
        var data = JSON.parse(result.data);
        console.log(data)
        switch (data.cmd) {
          case 1: initData(data); break;
          case 2: editMemory(data); break
          case 3: online.innerText = "在线人数："+data.data; break
          case 4: setMemoryStatus(data.index, true); break;
          case 5: setMemoryStatus(data.index, false); break;
        }
      }
    } catch (e) {
      console.log(e.message);
    }
  }

  function setMemoryStatus(index, lockFlag = true) {
    let tmp = document.getElementById("memory_"+(index || 0))
    if (tmp) {
      console.log(index, 'lock')
      tmp.disabled = lockFlag
    }
  }

  function editMemory(data) {
    let tmp = document.getElementById("memory_"+(data.index || 0))
    if (tmp) {
      tmp.value = data.data
      tmp.disabled = true
    } else {
      addText(data.index, data.data, true)
    }
    // tmp.focus()
  }

  function initData(data) {
    let list = JSON.parse(data.data)
    // console.log(list)
    for (let key in list) {
      const item = list[key]
      let tmp = document.getElementById("memory_"+(key || 0))
      if (tmp) {
        tmp.value = item.data
        tmp.disabled = item.lock
      } else {
        addText(+key, item.data, item.lock)
      }
    }
    initFlag = false
  }

  function checkReConn() {
    if (ws) {
      return
    }
    if (confirm("连接已断开，是否重新连接？")) {
      conn()
    }
  }

  function sendData(cmd, index, data = '') {
    let tmpData = { cmd, index: +index, data }
    if (!ws) {
      return
    }
    ws.send(JSON.stringify(tmpData))
  }

  function changeText() {
    if (!ws) {
      console.log("closed ws")
      checkReConn()
      return
    }
    sendData(2, +this.name, this.value)
  }
</script>
`
	hd.script = fmt.Sprintf(hd.script, address)
	return
}
