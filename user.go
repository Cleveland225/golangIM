package main

import(
	"net"
	"strings"
)

type User struct{
	Name string
	Addr string
	C chan string
	conn net.Conn

	server *Server
}

//创建一个用户API
func NewUser(conn net.Conn, server *Server) *User{
	userAddr:=conn.RemoteAddr().String()
	user:=&User{
		Name:userAddr,
		Addr:userAddr,
		C:make(chan string),
		conn:conn,

		server:server,
	}

	//启动监听当前user channel消息的goroutine	
	go user.ListenMessage()
	return user
}

//上线业务
func (this *User) Online(){

	//用户上线，将用户加入OnlineMap中
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name]=this
	this.server.mapLock.Unlock()

	//广播当前用户上线消息
	this.server.BroadCast(this, "已上线")
}

//下线业务
func (this *User) Offline(){
	//用户下线
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()

	//广播当前用户上线消息
	this.server.BroadCast(this, "下线")
}

//给当前user对应的客户端发消息
func (this *User) SendMsg(msg string){
	this.conn.Write([]byte(msg))
}

//处理消息业务
func (this *User) DoMessage(msg string){
	if msg=="list_users"{
		//查询当前在线用户
		this.server.mapLock.Lock()
		for _,user:=range this.server.OnlineMap{
			onlinemsg:="["+user.Addr+"]"+user.Name+":"+"在线...\n"
			this.SendMsg(onlinemsg)
		}
		this.server.mapLock.Unlock()
	}else if len(msg)>7 && msg[:7]=="rename|" {
		//改名
		newName:=strings.Split(msg,"|")[1]
		//判断是否重名
		_,ok:=this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前用户名被使用\n")
		}else{
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap,this.Name)
			this.server.OnlineMap[newName]=this
			this.server.mapLock.Unlock()

			this.Name=newName
			this.SendMsg("您已经更新用户名："+this.Name+"\n")
		}
	}else if len(msg)>4 && msg[:3]=="to|" {
		//消息格式私聊

		//获取对方用户名
		remoteName:=strings.Split(msg,"|")[1]
		if remoteName==""{
			this.SendMsg("消息格式不正确，请使用\"to|zxh|hello\"格式。\n")
			return
		}

		//根据用户名得到对方User对象
		remoteUser,ok:=this.server.OnlineMap[remoteName]
		if !ok{
			this.SendMsg("该用户名不存在或未上线\n")
			return
		}

		//获取消息内容，通过对方的User对象将消息内容发送
		content:=strings.Split(msg,"|")[2]
		if content==""{
			this.SendMsg("你放了个屁\n")
			return
		}
		remoteUser.SendMsg(this.Name+"私聊您说："+content+"\n")
	}else{
		this.server.BroadCast(this, msg)
	}
}

//监听当前user channel的方法，一旦有消息，就直接发给客户端
func (this *User) ListenMessage(){
	for{
		msg:=<-this.C
		this.conn.Write([]byte(msg+"\n"))
	}
}

