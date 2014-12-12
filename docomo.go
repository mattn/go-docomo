package docomo

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
)

const (
	dialogueURL             = "https://api.apigw.smt.docomo.ne.jp/dialogue/v1/dialogue?APIKEY=%s"
	characterRecognitionURL = "https://api.apigw.smt.docomo.ne.jp/characterRecognition/v1/line?APIKEY=%s"
)

type Client struct {
	c       *http.Client
	apikey  string
	user    User
	context string
	mode    string
}

type User struct {
	Nickname       string `json:"nickname"`       //ユーザのニックネームを設定します。10文字以下
	NicknameY      string `json:"nickname_y"`     //ユーザのニックネームの読みを設定します。全角20文字以下(カタカナのみ)
	Sex            string `json:"sex"`            //ユーザの性別を設定します。男または女
	BloodType      string `json:"bloodtype"`      //ユーザの血液型を設定します。A、B、AB、O のいずれか
	BirthDateY     int    `json:"birthdateY"`     //ユーザの誕生日（年）を設定します。1～現在までのいずれかの整数(半角4文字以下)
	BirthDateM     int    `json:"birthdateM"`     //ユーザの誕生日（月）を設定します。1～12までのいずれかの整数
	BirthDateD     int    `json:"birthdateD"`     //ユーザの誕生日（日）を設定します。1～31までのいずれかの整数
	Age            int    `json:"age"`            //ユーザの年齢を設定します。正の整数(半角3文字以下)
	Constellations string `json:"constellations"` //ユーザの星座を設定します。牡羊座、牡牛座、双子座、蟹座、獅子座、乙女座、天秤座、蠍座、射手座、山羊座、水瓶座、魚座のいずれか
	Place          string `json:"place"`          //ユーザの地域情報を設定します。仕様書 2.4「場所リスト」に含まれるもののいずれか
}

func NewClient(apikey string, u User) *Client {
	client := new(http.Client)
	transport := new(http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	transport.Proxy = http.ProxyFromEnvironment
	client.Transport = transport
	c := new(Client)
	c.c = client
	c.apikey = apikey
	c.user = u
	return c
}

type DialogueResponse struct {
	Utt     string `json:"utt"`
	Yomi    string `json:"yomi"`
	Mode    string `json:"mode"`
	Da      string `json:"da"`
	Context string `json:"context"`
}

func (c *Client) Dialogue(utt string) (*DialogueResponse, error) {
	type post struct {
		User
		Context string `json:"context"` //システムから出力されたcontextを入力することにより会話を継続します。255文字以下
		Mode    string `json:"mode"`    //現在の対話のモード。システムから出力されたmodeを入力することによりしりとりを継続,dialogまたはsrtr　デフォルト：dialog
		Utt     string `json:"utt"`     //ユーザの発話を入力します。255文字以下
	}
	var p post
	p.User = c.user
	p.Utt = utt
	p.Context = c.context
	p.Mode = c.mode
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf(dialogueURL, c.apikey), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		var r DialogueResponse
		err = json.NewDecoder(res.Body).Decode(&r)
		if err != nil {
			return nil, err
		}
		c.context = r.Context
		c.mode = r.Mode
		return &r, nil
	}
	return nil, fmt.Errorf("Dialogue: %s", res.Status)
}

type CharacterRecognitionResult struct {
	Words struct {
		Word []struct {
			Text string `json:"@text"`
		} `json:"word"`
	} `json:"words"`
	Message struct {
		Text string `json:"@text"`
	} `json:"message"`
}

func (c *Client) CharacterRecognition(ct, filename string, data []byte) (*CharacterRecognitionResult, error) {
	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)
	part := make(textproto.MIMEHeader)
	part.Set("Content-Type", ct)
    part.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, filename))
    f, err := w.CreatePart(part)
    if err != nil {
		return nil, err
	}
	_, err = f.Write(data)
	if err != nil {
		return nil, err
	}
	w.Close()
	ct = w.FormDataContentType()
	req, err := http.NewRequest("POST", fmt.Sprintf(characterRecognitionURL, c.apikey), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ct)
	res, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		var r CharacterRecognitionResult
		err = json.NewDecoder(res.Body, os.Stdout).Decode(&r)
		if err != nil {
			return nil, err
		}
		return &r, nil
	}
	io.Copy(os.Stdout, res.Body)
	return nil, fmt.Errorf("CharacterRecognition: %s", res.Status)
}
