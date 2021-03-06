package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
	"log"
	"strconv"
	"sync"
	"web/library"
)

var mm = melody.New()

func SocketGame() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("访问ws")

		mm.HandleRequest(c.Writer, c.Request)
	}
}

func gameBroadcast(t string, name interface{}) {
	e := Users{
		Type: t,
		Data: name,
	}
	r, err := json.Marshal(e)
	if err != nil {
		log.Printf("发生错误: %v", err)
	}
	//广播向所有会话广播文本消息
	mm.Broadcast(r)
}

// 游戏消息定义
type GameEvent struct {
	Type      string `json:"type"`      // 事件类型
	User      string `json:"user"`      // 用户名
	Uid       string `json:"uid"`       // 用户id
	PosX      string `json:"x"`         // x坐标
	PosY      string `json:"y"`         // y坐标
	Direction string `json:"direction"` // 方向
	Heart     int    `json:"heart"`     // 血量
}

type BulletEvent struct {
	Type    string      `json:"type"`    // 事件类型
	Bullets interface{} `json:"bullets"` // 数组
}

//在线用户
var OnlineUsers = make(map[string]GameEvent)

func GameInit() {
	mm.Config.MaxMessageSize = 2048*10
	lock := new(sync.Mutex)
	// 监听连接事件
	mm.HandleConnect(func(s *melody.Session) {
		// 1. 实例化连接消息
		fmt.Println("ws game 连接消息")
		uid, name, err := getQueryAuth(s)
		if err != nil {
			fmt.Println(err)
		}
		//OnlineUsers["1234567"] = GameEvent{
		//	Type:      "pos",
		//	User:      "admin",
		//	Uid:       "1234567",
		//	PosX:      "250",
		//	PosY:      "250",
		//	Direction: "0",
		//	Heart:     9,
		//}
		_, ok := OnlineUsers[uid]
		if ok == false {
			OnlineUsers[uid] = GameEvent{
				Type:      "pos",
				User:      name,
				Uid:       uid,
				PosX:      "150",
				PosY:      "150",
				Direction: "0",
				Heart:     9,
			}
		}
		fmt.Println(ok)
		fmt.Println(OnlineUsers)
		//玩家加入
		gameBroadcast("playJoin", OnlineUsers)
	})

	// 监听接收事件
	mm.HandleMessage(func(s *melody.Session, msg []byte) {
		lock.Lock()

		//fmt.Println("ws game 接收消息:", string(msg))
		var data map[string]interface{}
		err := json.Unmarshal(msg, &data)
		if err != nil {
			fmt.Println("发生错误", err)
		}
		switch data["type"] {
		//坦克位置消息
		case "pos":
			uid, name, err := getQueryAuth(s)
			if err != nil {
				fmt.Println(err)
			}
			e := GameEvent{
				Type:      "pos",
				User:      name,
				Uid:       uid,
				PosX:      data["x"].(string),
				PosY:      data["y"].(string),
				Direction: data["direction"].(string),
				Heart:     OnlineUsers[uid].Heart,
			}
			//更新数据
			OnlineUsers[uid] = e

			r, err := json.Marshal(e)
			if err != nil {
				log.Printf("发生错误: %v", err)
			}
			//广播向所有会话广播文本消息
			mm.Broadcast(r)
		//子弹位置消息
		case "bullets":
			//_, _, err := getQueryAuth(s)

			e := BulletEvent{
				Type:    "bullets",
				Bullets: data["bullets"],
			}

			r, err := json.Marshal(e)
			if err != nil {
				log.Printf("发生错误: %v", err)
			}
			//广播向所有会话广播文本消息
			mm.Broadcast(r)
		//受伤减血
		case "injured":
			//uid, name, _ := getQueryAuth(s)

			tank := OnlineUsers[data["uid"].(string)]
			heart := tank.Heart - 1
			fmt.Println("血量:", heart)
			fmt.Println(heart <= 0)
			if heart <= 0 {
				e := GameEvent{
					Type: "gameOver",
					User: tank.User,
					Uid:  tank.Uid,
				}

				r, err := json.Marshal(e)
				if err != nil {
					log.Printf("发生错误: %v", err)
				}
				fmt.Println("死掉了", e)
				mm.Broadcast(r)
			} else {
				e := GameEvent{
					Type:      "injured",
					User:      tank.User,
					Uid:       tank.Uid,
					PosX:      strconv.Itoa(library.RandInt(0, 760)),
					PosY:      strconv.Itoa(library.RandInt(0, 700)),
					Direction: tank.Direction,
					Heart:     heart, //血-1
				}
				//更新数据
				OnlineUsers[tank.Uid] = e

				r, err := json.Marshal(e)
				if err != nil {
					log.Printf("发生错误: %v", err)
				}
				fmt.Println("复活", e)
				mm.Broadcast(r)

			}

		default:
			log.Fatalf("type 错误")
		}

		lock.Unlock()
	})

	// 监听连接断开事件
	mm.HandleDisconnect(func(s *melody.Session) {
		// 断开连接消息
		fmt.Println("ws game 断开连接消息")
		uid, _, err := getQueryAuth(s)
		if err != nil {
			fmt.Println(err)
		}
		delete(OnlineUsers, uid)
		gameBroadcast("playOut", uid)
	})

	// 监听连接错误
	mm.HandleError(func(s *melody.Session, e error) {
		log.Println("ws game 发生错误", e.Error())
	})
}
