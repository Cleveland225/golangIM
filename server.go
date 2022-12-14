package main

import(
	"fmt"
	"net"
	"sync"
	"io"
	"time"
	// "github.com/axgle/mahonia"
)


// func ConvertToString(src string, srcCode string, tagCode string) string {
 
// 	srcCoder := mahonia.NewDecoder(srcCode)
 
// 	srcResult := srcCoder.ConvertString(src)
 
// 	tagCoder := mahonia.NewDecoder(tagCode)
 
// 	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
 
// 	result := string(cdata)
 
// 	return result
// }

type Server struct{
	IP string
	Port int

	//在线用户列表
	OnlineMap map[string]*User
	mapLock sync.RWMutex

	//消息广播的channel
	Message chan string
}

//创建一个server的接口
func NewServer(ip string, port int) *Server{
	server := &Server{
		IP:ip,
		Port:port,
		OnlineMap:make(map[string]*User),
		Message:make(chan string),
	}

	return server
}

//监听Message广播消息channel的goroutine，一旦有消息发送给全部在线User
func (this *Server) ListenMessage(){
	for{
		msg:=<-this.Message

		//将msg发送给全部在线的User
		this.mapLock.Lock()
		for _,cli:= range this.OnlineMap{
			cli.C<-msg
		}
		this.mapLock.Unlock()
	}
}

//广播消息的方法
func (this *Server) BroadCast(user *User, msg string){
	sendMsg:="["+user.Addr+"]"+user.Name+":"+msg

	this.Message<-sendMsg
}

func (this *Server) Handler(conn net.Conn){
	//当前链接的业务
	// fmt.Println("链接建立")

	user:=NewUser(conn,this)

	user.Online()

	//监听用户是否活跃channel
	isLive:=make(chan bool)

	//接受客户端发送的消息
	go func() {
		buf:=make([]byte, 4096)
		for {
			n,err:=conn.Read(buf)
			if n==0 {
				user.Offline()
				return
			}

			if err!=nil && err!=io.EOF{
				fmt.Println("Conn Read err:", err)
				return
			}

			//提取用户消息去除\n
			msg:=string(buf[:n-1])
			// msg=ConvertToString(msg,"gbk","utf-8")

			//用户处理信息
			user.DoMessage(msg)

			//用户任意消息，代表活跃
			isLive<-true
		}
	}()

	//当前Handler阻塞
	for {
		select{
		case <-isLive:
			//活跃，重置定时器
		case <-time.After(time.Second*3600):
			//已经超时，强制关闭
			user.SendMsg("你被踢了")
			//销毁资源
			close(user.C)
			conn.Close()

			return
		}
	}
}

//启动服务器的接口
func (this *Server) Start(){
	//socket listen
	listener,err:=net.Listen("tcp",fmt.Sprintf("%s:%d",this.IP,this.Port))
	if err!= nil{
		fmt.Println("net.Listen err:",err)
		return
	}
	//close listen socket
	defer listener.Close()

	//启动监听Message的goroutine
	go this.ListenMessage()

	for{
		//accept
		conn,err:=listener.Accept()
		if err!= nil{
			fmt.Println("listener accept err:",err)
			continue
		}

		go this.Handler(conn)
	}

}

