package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// publish-subscribe patterns.

type Publisher struct {
	dialer string
}

type Subscriber struct {
	dialer string
	conn   *websocket.Conn
}

type Repeater struct {
	publisher   *Publisher
	subscribers map[*Subscriber]bool
	register    chan *Subscriber
	unregister  chan *Subscriber
}

func (r *Repeater) run() {
	for {
		select {
		case client := <-r.register:
			last_count := len(r.subscribers)
			r.subscribers[client] = true
			log.Printf("r.register count : [%d] -> [%d]", last_count, len(r.subscribers))
		case client := <-r.unregister:
			last_count := len(r.subscribers)
			if _, ok := r.subscribers[client]; ok {
				delete(r.subscribers, client)
			}
			log.Printf("r.register count : [%d] -> [%d]", last_count, len(r.subscribers))
		}
	}
}

func (r *Repeater) send_broadcast(request_route string) {
	log.Println("server_url:", r.publisher.dialer)
	server_conn, _, err := websocket.DefaultDialer.Dial(r.publisher.dialer, nil)

	if err != nil {
		log.Fatal("dial:", err)
	}
	defer server_conn.Close()

	for {
		// 接收消息websocket.BinaryMessage or websocket.TextMessage.
		message_type, message_data, err := server_conn.ReadMessage()
		if err != nil {
			log.Println("ReadMessage Error:", err)
			break
		}

		// log.Printf("Recv Message[%d]:[0x%x]bytes", message_type, len(message_data))
		// log.Printf("r.subscribers", len(r.subscribers))

		for subscriber := range r.subscribers {
			if err := subscriber.conn.WriteMessage(message_type, message_data); err != nil {
				fmt.Println("DataRepeater to ", subscriber.dialer, " WriteMessage Error:", err)

				r.unregister <- subscriber
				subscriber.conn.Close()
				break
			}
		}
	}
}

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// func GetMetadata(c *gin.Context) {
// 	request_route := c.Request.RequestURI
// 	log.Println("Recv Request  :", request_route)
// 	code := http.StatusOK

// 	url := fmt.Sprintf("http://%s%s", server_host_port, request_route)
// 	resp_data, err := utils.PostRequest(url, "")
// 	if err != nil {
// 		log.Printf("PostRequest failed,[err=%s][url=%s]", err, url)

// 		code = http.StatusInternalServerError
// 	}

// 	// log.Println("Recv Response :", resp_data)
// 	c.String(code, string(resp_data))
// }

func DoSubscribe(c *gin.Context) {
	//升级get请求为webSocket协议
	client_conn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	// defer client_conn.Close()

	request_route := c.Request.RequestURI
	// Recv Request  : /subscribe?chan=RAW_GNSS /subscribe?chan=RAW_GNSS /subscribe 127.0.0.1 127.0.0.1:40128
	log.Println("Recv Request :", request_route, c.Request.URL, c.FullPath(), c.ClientIP(), c.Request.RemoteAddr)

	if repeater, ok := repeaters[request_route]; ok {
		log.Println("find:", request_route)

		client := &Subscriber{
			dialer: request_route,
			conn:   client_conn,
		}

		repeater.register <- client

	} else {
		log.Println("not find:", request_route)

		server_url := fmt.Sprintf("ws://%s%s", server_host_port, request_route)

		server := &Publisher{
			dialer: server_url,
		}

		client := &Subscriber{
			dialer: request_route,
			conn:   client_conn,
		}

		repeater := &Repeater{
			publisher:   server,
			subscribers: make(map[*Subscriber]bool),
			register:    make(chan *Subscriber, 100),
			unregister:  make(chan *Subscriber, 100),
		}

		go repeater.run()

		repeater.register <- client
		repeaters[request_route] = repeater

		go repeater.send_broadcast(request_route)

	}
}

// 打印帮助信息
func usage() {
	log.Println(`Usage: client [-h] [-r] [-p local service port] [-s data sorter server]`)
	flag.PrintDefaults()
}

func print_parameter() {
	log.SetFlags(0)
	log.Println("Starting application...")
	log.Printf("Gin Release Mode       : [%t]", release_mode)
	log.Printf("Local Service Port     : [%d]", local_port)
	log.Printf("Sorter Service Host    : [%s]", server_host_port)
}

var (
	help             bool                                              // help info
	server_host_port string               = "127.0.0.1:2303"           // data sorter 服务地址
	repeaters        map[string]*Repeater = make(map[string]*Repeater) // 数据转发器
	local_port       int                  = 2305                       // 本地服务端口
	release_mode     bool                 = false                      // Running in "release" mode
)

func main() {

	flag.BoolVar(&help, "h", false, "useage help")
	flag.BoolVar(&release_mode, "r", false, "running in release mode")
	flag.StringVar(&server_host_port, "s", "127.0.0.1:2303", "remote service host and port")
	flag.IntVar(&local_port, "p", 2305, "local service port")
	flag.Parse()

	if help {
		usage()
		return
	}

	if release_mode {
		gin.SetMode(gin.ReleaseMode)
	}

	print_parameter()

	// 设置源地址
	utils.SourceHostPort = server_host_port

	r := gin.Default()
	r.GET("/appinfo", utils.RepeatGetRequest)
	r.GET("/sensor/list", utils.RepeatGetRequest)
	r.GET("/output/list", utils.RepeatGetRequest)

	r.POST("/enable", utils.RepeatPostRequest)
	r.POST("/start", utils.RepeatPostRequest)
	r.POST("/stop", utils.RepeatPostRequest)

	r.GET("/sensor/gnss", DoSubscribe)
	r.GET("/sensor/imu", DoSubscribe)
	r.GET("/sensor/vehicle", DoSubscribe)
	r.GET("/sensor/objectdetector", DoSubscribe)

	r.GET("/output/gnss", DoSubscribe)
	r.GET("/output/imu", DoSubscribe)
	r.GET("/output/vehicle", DoSubscribe)
	r.GET("/output/mapfusion", DoSubscribe)
	r.GET("/output/cpt", DoSubscribe)
	r.GET("/output/fsekf", DoSubscribe)
	r.GET("/output/objectdetector", DoSubscribe)

	bindAddress := fmt.Sprintf(":%d", local_port)
	r.Run(bindAddress)
}
