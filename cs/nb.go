package main

import (
	"io"
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
	if checkIp(args[2]) {
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

func checkIp(address string) bool {
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

func port2port(port1 string, port2 string) {
	listen1 := start_server("0.0.0.0:" + port1)
	listen2 := start_server("0.0.0.0:" + port2)
	log.Println("[√]", "监听端口:", port1, " 和 ", port2, "成功. 等待客户端连接...")
	for {
		conn1 := accept(listen1)
		conn2 := accept(listen2)
		if conn1 == nil || conn2 == nil {
			log.Println("[x]", "接收客户端失败. 将在 ", timeout, " 秒后进行重试. ")
			time.Sleep(timeout * time.Second)
			continue
		}
		forward(conn1, conn2)
	}
}

func port2host(allowPort string, targetAddress string) {
	server := start_server("0.0.0.0:" + allowPort)
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

func host2host(address1, address2 string) {
	for {
		log.Println("[+]", "尝试连接地址:["+address1+"] 和 ["+address2+"]")
		var host1, host2 net.Conn
		var err error
		for {
			host1, err = net.Dial("tcp", address1)
			if err == nil {
				log.Println("[→]", "连接 ["+address1+"] 成功.")
				break
			} else {
				log.Println("[x]", "连接目标地址 ["+address1+"] 失败. 将在 ", timeout, " 秒后进行重试. ")
				time.Sleep(timeout * time.Second)
			}
		}
		for {
			host2, err = net.Dial("tcp", address2)
			if err == nil {
				log.Println("[→]", "连接 ["+address2+"] 成功.")
				break
			} else {
				log.Println("[x]", "连接目标地址 ["+address2+"] 失败. 将在 ", timeout, " 秒后进行重试. ")
				time.Sleep(timeout * time.Second)
			}
		}
		forward(host1, host2)
	}
}

func start_server(address string) net.Listener {
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
	logFile := openLog(conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	if logFile != nil {
		w := io.MultiWriter(conn1, logFile)
		io.Copy(w, conn2)
	} else {
		io.Copy(conn1, conn2)
	}
	conn1.Close()
	log.Println("[←]", "断开本地连接:["+conn1.LocalAddr().String()+"] 和远程:["+conn1.RemoteAddr().String()+"]")
	//conn2.Close()
	//log.Println("[←]", "close the connect at local:["+conn2.LocalAddr().String()+"] and remote:["+conn2.RemoteAddr().String()+"]")
	wg.Done()
}
func openLog(address1, address2, address3, address4 string) *os.File {
	args := os.Args
	argc := len(os.Args)
	var logFileError error
	var logFile *os.File
	if argc > 5 && args[4] == "-log" {
		address1 = strings.Replace(address1, ":", "_", -1)
		address2 = strings.Replace(address2, ":", "_", -1)
		address3 = strings.Replace(address3, ":", "_", -1)
		address4 = strings.Replace(address4, ":", "_", -1)
		timeStr := time.Now().Format("2006_01_02_15_04_05") // "2006-01-02 15:04:05"
		logPath := args[5] + "/" + timeStr + args[1] + "-" + address1 + "_" + address2 + "-" + address3 + "_" + address4 + ".log"
		logPath = strings.Replace(logPath, `\`, "/", -1)
		logPath = strings.Replace(logPath, "//", "/", -1)
		logFile, logFileError = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
		if logFileError != nil {
			log.Fatalln("[x]", "log file path error.", logFileError.Error())
		}
		log.Println("[√]", "open test log file success. path:", logPath)
	}
	return logFile
}
