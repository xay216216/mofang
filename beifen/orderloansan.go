package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/tidwall/gjson"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"jindan/datastruct"
	"jindan/util"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	redisClient *redis.Client
	mgoSession  *mgo.Session
	dataBase    = "test"
	getTokenUrl = "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=ww338b579ec6e2589b&corpsecret=PunPJ7c-cvjAg_ew_JXPGWE18r_OfiGfwAFqjTqIjo0"
	weiXinUrl   = "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="
)

const (
	accessToken = "access_token"
	URL         = "localhost:27017" //mongodb连接字符串
	mgoUrl      = "mongodb://admin:password@192.168.100.114:27017/fanghuwang_db"
)

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "123456", // no password set
		DB:       0,        // use default DB
	})
	getSession()
}

func getSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial(URL)
		if err != nil {
			panic(err) //直接终止程序运行
		}
	}
	//最大连接池默认为4096
	return mgoSession.Clone()
}

// 执行脚本
func GoOrderLoan() {
	defer func() {
		if err := recover(); err != nil {
			util.DingDingNotice("markdown", "脚本任务任务崩溃了", "脚本任务任务崩溃了")
		}
	}()

	for {
		log.Println("开始执行脚本")
		getAccessToken()
		ScriptDataAnalysis()
		log.Println("Sleep ")
		time.Sleep(time.Minute * 1)
	}
}

// 执行脚本
func getAccessToken() {
	token, err := redisClient.Get(accessToken).Result()
	if err != nil {
		fmt.Printf("err type:%T\n", err)
	}
	if len(token) == 0 {
		response, err := http.Get(getTokenUrl)
		body, err := ioutil.ReadAll(response.Body) //[]uint8
		if err != nil {
			panic(err)
		}
		defer response.Body.Close()
		tokenInfo := new(datastruct.RedisAccessToken)
		err = json.Unmarshal(body, tokenInfo)
		tokenValue := tokenInfo.Access_token
		err = redisClient.Set("access_token", tokenValue, 7200*time.Second).Err()
		if err != nil {
			panic(err)
		}
	}
}

func ScriptDataAnalysis() {
	fmt.Println("timeStr:", 222)
	mgoSession.SetMode(mgo.Monotonic, true)
	db := mgoSession.DB(dataBase)
	apifinancingZJ0001Log := db.C("apifinancing_ZJ0001_log")
	timeStr := time.Now().Unix() - 86400
	fmt.Println("timeStr:", timeStr)
	iter := apifinancingZJ0001Log.Find(bson.M{"sign_status": 1, "is_update_mysql": bson.M{"$gte": 1}, "timeStr": bson.M{"$gte": timeStr}}).Iter()
	content := new(datastruct.MomgoOrderloan)

	var successCount = 0
	var failCount = 0
	//var sucessData []string
	//var failData []string //切片
	var failAssetsOrderNo string
	if iter != nil {
		for iter.Next(content) {
			Data := content.Data
			Id := content.Id
			IsUpdateMysql := content.Is_update_mysql
			AssetsOrderNo := gjson.Get(Data, "assetOrderNo").String()
			log.Println("Id:", Id)
			log.Println("IsUpdateMysql:", IsUpdateMysql)
			fmt.Println("Name:", AssetsOrderNo)
			if IsUpdateMysql == 1 {
				successCount++
			} else {
				//failData = append(failData, AssetsOrderNo)
				failAssetsOrderNo = failAssetsOrderNo + "," + AssetsOrderNo
				failCount++
			}
		}
	}
	//fasong
	postContent := "成功执行de订单条数：" + strconv.Itoa(successCount) + "\n失败执行订单条数：" + strconv.Itoa(failCount) + "\n失败的订单号为：" + failAssetsOrderNo
	formt := `
    {
        "touser" : "XiaoAYong",
        "toparty" : "3",
        "totag" : "1",
        "msgtype" : "text",
        "agentid" : 1000002,
        "text" : {
            "content" : "%s"。
        },
        "safe":0
    }`
	postBody := fmt.Sprintf(formt, postContent)
	jsonValue := []byte(postBody)
	tokenValue, err := redisClient.Get(accessToken).Result()
	fmt.Println("tokenValue:", tokenValue)
	resp, err := http.Post(weiXinUrl+tokenValue, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		panic(err)
	}
	fmt.Println("resp:", resp)
	iter.Close()
}
