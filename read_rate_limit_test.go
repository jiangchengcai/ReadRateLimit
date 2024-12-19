// @Time : 2024/12/19 14:05
// @Author :  jiang
// @File : test_rate_limit
// @Software : Goland
// @Desc : to do somewhat..

package readRateLimit

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

var readSize uint64 = 2000 // Kb bit
var granularity uint64 = 1e9

func TestLimitReader(t *testing.T) {
	f, err := os.Open("./2.jpg")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	r, err := NewLimitReader(readSize, granularity, f)
	if err != nil {
		panic(err)
	}
	finfo, err := f.Stat()
	if err != nil {
		panic(err)
	}
	fmt.Printf("文件大小：%d byte,限制速度：%d kbit\n", finfo.Size(), readSize)

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1/file", r)
	if err != nil {
		panic(err)
	}
	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	fmt.Printf("理论耗时：%.2f s;实际耗时：%.2f s\n", float64(finfo.Size()*8)/float64(readSize*1000), time.Now().Sub(startTime).Seconds())
}

func TestHttpServer(t *testing.T) {
	server := http.Server{
		Addr: ":80",
	}
	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.OpenFile(fmt.Sprintf("./%d.jpg", time.Now().Unix()), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		defer f.Close()
		io.Copy(f, r.Body)
		return
	})

	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}

}
