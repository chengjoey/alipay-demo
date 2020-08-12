package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	alipay "github.com/smartwalle/alipay/v3"
)

const (
	host = "" //服务地址域名或ip地址
	appId = "" //沙箱环境appId或正式环境
	appPrivateKeyFileName = "" // 应用私钥证书地址
	appCertPublicKeyFileName = "" // 应用公钥证书地址
	AliPayRootCertFileName = "" // 支付宝根证书地址
	AliPayPublicCertFileName = "" // 支付宝公钥证书地址
)

var privateKey = ""
var client = &alipay.Client{}

func init() {
	rand.Seed(time.Now().UnixNano())
	b, err := ioutil.ReadFile(appPrivateKeyFileName)
	if err != nil {
		log.Fatalf("Open app private failed: %v", err)
	}
	privateKey = string(b)

	client, err = alipay.New(appId, privateKey, false)
	if err != nil {
		log.Fatalf("Create alipay client err: %v", err)
	}

	client.LoadAppPublicCertFromFile(appCertPublicKeyFileName)
	client.LoadAliPayRootCertFromFile(AliPayRootCertFileName)
	client.LoadAliPayPublicCertFromFile(AliPayPublicCertFileName)
}


func createTradeNo() string {
	randBytes := make([]byte, 6)
	rand.Read(randBytes)
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x",
		randBytes[0], randBytes[1], randBytes[2],
		randBytes[3], randBytes[4], randBytes[5])
}

func generateRandomInt() int {
	result := rand.Intn(1000)
	return result
}

func generatePayUrl() (string, error) {
	p := alipay.TradePagePay{}
	p.Subject = "测试订单"
	p.TotalAmount = fmt.Sprintf("%d", generateRandomInt())
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	p.OutTradeNo = createTradeNo()
	p.NotifyURL = fmt.Sprintf("http://%s:5000/notify", host)
	p.ReturnURL = fmt.Sprintf("http://%s:5000/return", host)

	url, err := client.TradePagePay(p)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func pay(w http.ResponseWriter, r *http.Request) {
	url, err := generatePayUrl()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, url, 302)
	return
}

func returnHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println("parse form err:", err.Error())
	}
	ok, err := client.VerifySign(r.Form)
	fmt.Println(ok, err)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if !ok {
		http.Error(w, "验证失败", 400)
		return
	}
	w.Write([]byte("验证成功"))
}

func notifyHandler(w http.ResponseWriter, r *http.Request) {
	var noti, err = client.GetTradeNotification(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if noti != nil {
		fmt.Println("交易状态为:", noti.TradeStatus)
	}
	alipay.AckNotification(w)
}

func main() {
	http.HandleFunc("/pay", pay)
	http.HandleFunc("/return", returnHandler)
	http.HandleFunc("/notify", notifyHandler)
	err := http.ListenAndServe(":5000", nil)
	if err != nil {
		log.Fatalf("ListenAndServe err : %v", err)
	}
}