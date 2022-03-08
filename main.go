package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"text/template"
	"time"
)

const baseUrl = "https://openapi.youdao.com/api"
const outputTemplate = `
@ {{.Query}}
英: [ {{.Basic.UkPhonetic}} ]
美: [ {{.Basic.UsPhonetic}} ]
[翻译]
{{- range $i, $e :=.Translation}}
	{{$i}} . {{.}}
{{- end}}
[延伸]
{{- range $i, $e :=.Basic.Explains}}
	{{$i}} . {{.}}
{{- end}}
[网络]
{{- range $i, $e :=.Web}}
	{{$i}} . {{.Key}}
	翻译：{{range .Value}}{{.}}, {{end}}
{{- end}}
`

type DictResp struct {
	ErrorCode    string                 `json:"errorCode"`
	Query        string                 `json:"query"`
	Translation  []string               `json:"translation"`
	Basic        DictBasic              `json:"basic"`
	Web          []DictWeb              `json:"web,omitempty"`
	Lang         string                 `json:"l"`
	Dict         map[string]interface{} `json:"dict,omitempty"`
	WebDict      map[string]interface{} `json:"webdict,omitempty"`
	TSpeakUrl    string                 `json:"tSpeakUrl,omitempty"`
	SpeakUrl     string                 `json:"speakUrl,omitempty"`
	ReturnPhrase []string               `json:"returnPhrase,omitempty"`
}

type DictBasic struct {
	UsPhonetic string   `json:"us-phonetic"`
	Phonetic   string   `json:"phonetic"`
	UkPhonetic string   `json:"uk-phonetic"`
	UkSpeech   string   `json:"uk-speech"`
	UsSpeech   string   `json:"us-speech"`
	Explains   []string `json:"explains"`
}

type DictWeb struct {
	Key   string   `json:"key"`
	Value []string `json:"value"`
}

func main() {
	flag.Parse()
	q := flag.Arg(0)
	if len(q) == 0 {
		panic("please enter a word")
	}
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	APP_KEY := os.Getenv("APP_KEY")
	APP_SECRET := os.Getenv("APP_SECRET")
	if len(APP_KEY) == 0 || len(APP_SECRET) == 0 {
		panic("cannot find .env to load APP_KEY or APP_SECRET")
	}
	salt := uuid.NewString()
	client := resty.New()
	client.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	now := strconv.FormatInt(time.Now().Unix(), 10)
	r := client.R().SetFormData(map[string]string{
		"q":        q,
		"from":     "auto",
		"to":       "auto",
		"signType": "v3",
		"appKey":   APP_KEY,
		"salt":     salt,
		"curtime":  now,
		"sign":     encrypt(APP_KEY + truncate(q) + salt + now + APP_SECRET),
	})
	var dict DictResp
	r.SetResult(&dict)
	_, err = r.Post(baseUrl)
	errCode, err := strconv.Atoi(dict.ErrorCode)
	if err != nil {
		panic("cannot parse errCode")
	}
	if errCode != 0 {
		panic(fmt.Sprintf("cannot get dict: %d", errCode))
	}

	if err != nil {
		panic(err)
	}
	output(dict)
}

func encrypt(s string) string {
	res := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", res)
}

func truncate(q string) string {
	temp := []rune(q)
	qLen := len(temp)
	if qLen <= 20 {
		return q
	}
	var input []rune
	input = append(input, temp[:10]...)
	input = append(input, []rune(strconv.Itoa(qLen))...)
	input = append(input, temp[qLen-10:]...)
	return string(input)
}

func output(d DictResp) {
	t := template.New("dict")
	t, err := t.Parse(outputTemplate)
	if err != nil {
		panic(err)
	}
	if err := t.Execute(os.Stderr, d); err != nil {
		panic(err)
	}
}
