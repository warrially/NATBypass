package main

import (
	// "io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const timeout = 5

func main() {
	//log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	args := os.Args
	argc := len(os.Args)

	if argc < 2 {
		log.Fatalln(`需要两个参数, 例如 "waryCS 1997 192.168.1.2:3389".`)
	}

	port := checkPort(args[1])
	var remoteAddress string
	if checkIP(args[2]) {
		remoteAddress = args[2]
	}
	split := strings.SplitN(remoteAddress, ":", 2)
	log.Println("[√]", "开始传送地址1:", remoteAddress, "到 地址2:", split[0]+":"+port)
	port2host(port, remoteAddress)

}

func checkPort(port string) string {
	PortNum, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalln("[x]", "端口必须是数字")
	}
	if PortNum < 1 || PortNum > 65535 {
		log.Fatalln("[x]", "端口范围是: [1,65535]")
	}
	return port
}

func checkIP(address string) bool {
	ipAndPort := strings.Split(address, ":")
	if len(ipAndPort) != 2 {
		log.Fatalln("[x]", "地址错误. 格式如下 [ip:port]. ")
	}
	ip := ipAndPort[0]
	port := ipAndPort[1]
	checkPort(port)
	pattern := `^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$`
	ok, err := regexp.MatchString(pattern, ip)
	if err != nil || !ok {
		log.Fatalln("[x]", "ip 错误. ")
	}
	return ok
}

func port2host(allowPort string, targetAddress string) {
	server := startServer("0.0.0.0:" + allowPort)
	for {
		conn := accept(server)
		if conn == nil {
			continue
		}
		//println(targetAddress)
		go func(targetAddress string) {
			log.Println("[+]", "开始连接服务器:["+targetAddress+"]")
			target, err := net.Dial("tcp", targetAddress)
			if err != nil {
				// temporarily unavailable, don't use fatal.
				log.Println("[x]", "连接目标地址 ["+targetAddress+"] 失败. 将在 ", timeout, " 秒后进行重试. ")
				conn.Close()
				log.Println("[←]", "close the connect at local:["+conn.LocalAddr().String()+"] and remote:["+conn.RemoteAddr().String()+"]")
				time.Sleep(timeout * time.Second)
				return
			}
			log.Println("[→]", "连接目标地址 ["+targetAddress+"] 成功.")
			forward(target, conn)
		}(targetAddress)
	}
}

func startServer(address string) net.Listener {
	log.Println("[+]", "尝试开启服务器在:["+address+"]")
	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln("[x]", "监听地址 ["+address+"] 失败.")
	}
	log.Println("[√]", "开始监听地址:["+address+"]")
	return server
	/*defer server.Close()

	for {
		conn, err := server.Accept()
		log.Println("accept a new client. remote address:[" + conn.RemoteAddr().String() +
			"], local address:[" + conn.LocalAddr().String() + "]")
		if err != nil {
			log.Println("accept a new client faild.", err.Error())
			continue
		}
		//go recvConnMsg(conn)
	}*/
}

func accept(listener net.Listener) net.Conn {
	conn, err := listener.Accept()
	if err != nil {
		log.Println("[x]", "接受连接 ["+conn.RemoteAddr().String()+"] 失败.", err.Error())
		return nil
	}
	log.Println("[√]", "接受一个新客户端. 远端地址是:["+conn.RemoteAddr().String()+"], 本地地址是:["+conn.LocalAddr().String()+"]")
	return conn
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	log.Printf("[+] 开始传输. [%s],[%s] <-> [%s],[%s] \n", conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	var wg sync.WaitGroup
	// wait tow goroutines
	wg.Add(2)
	go connCopy(conn1, conn2, &wg)
	go connCopy(conn2, conn1, &wg)
	//blocking when the wg is locked
	wg.Wait()
}

func connCopy(conn1 net.Conn, conn2 net.Conn, wg *sync.WaitGroup) {
	//TODO:log, record the data from conn1 and conn2.
	mycopy1(conn1, conn2)
	conn1.Close()
	log.Println("[←]", "断开本地连接:["+conn1.LocalAddr().String()+"] 和远程:["+conn1.RemoteAddr().String()+"]")
	//conn2.Close()
	//log.Println("[←]", "close the connect at local:["+conn2.LocalAddr().String()+"] and remote:["+conn2.RemoteAddr().String()+"]")
	wg.Done()
}


func mycopy1(conn1 net.Conn, conn2 net.Conn){
	// io.Copy(conn1, conn2)

	buff := make([]byte, 4096)

	for {
		n, err := conn1.Read(buff)
		if err != nil {
			return
		}

		for i:=0 ; i < n; i++ {
			buff[i] = ^buff[i]
		}

		conn2.Write(buff[:n])
	}



}