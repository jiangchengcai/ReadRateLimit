// @Time : 2024/12/19 14:02
// @Author :  jiang
// @File : reader
// @Software : Goland
// @Desc : to do somewhat..

package readRateLimit

import (
	"errors"
	"io"
	"time"
)

type LimitReader struct {
	lastTime      time.Time
	readSize      float64 // B
	KeepTime      int32   // 毫秒
	ticker        *time.Ticker
	totalUpTime   int
	totalUpCount  int
	Reader        ReaderSeeker
	consumingTime int
}

type ReaderSeeker interface {
	io.ReadSeeker
	io.ReaderAt
}

// 读取限速器
// readSize 每秒最大读取 单位bit
// granularity 1秒拆分多少个纳秒
func NewLimitReader(readSize, granularity uint64, reader ReaderSeeker) (*LimitReader, error) {
	if granularity == 0 {
		granularity = 1000
	}

	// 最小粒度1  time.Nanosecond
	if granularity > 1e9 {
		granularity = 1e9
	}

	readSizeb := float64(readSize*1024) / float64((granularity)*8)
	if readSizeb < 0 {
		readSizeb = 100
	}

	if reader == nil {
		return nil, errors.New("reader不能为空")
	}

	return &LimitReader{
		readSize: readSizeb,
		KeepTime: int32(time.Second.Nanoseconds()) / int32(granularity),
		Reader:   reader,
		ticker:   time.NewTicker(1 * time.Microsecond),
	}, nil
}

func (this *LimitReader) Read(b []byte) (int, error) {
	now := time.Now()
	// 限制不让读的时间
	wait := this.sleep(b)
	defer func() {
		this.lastTime = time.Now()

	}()
	// sleep函数处理的时间（除了time.sleep的时间）
	// 读写速度太快，这个时间需要很精确
	this.consumingTime = int(time.Now().Sub(now).Nanoseconds()) - int(wait)

	return this.Reader.Read(b)

}

func (this *LimitReader) ReadAt(b []byte, off int64) (n int, err error) {
	now := time.Now()
	wait := this.sleep(b)
	defer func() {
		this.lastTime = time.Now()
	}()
	// 封装之后消耗的时间 也就是自己read消耗的时间
	this.consumingTime = int(time.Now().Sub(now).Nanoseconds()) - int(wait)
	return this.Reader.ReadAt(b, off)
}

func (this *LimitReader) Seek(offset int64, whence int) (ret int64, err error) {
	return this.Reader.Seek(offset, whence)
}

func (this *LimitReader) sleep(b []byte) int64 {
	var wait time.Duration
	if !this.lastTime.IsZero() {
		if this.ticker != nil {
			this.totalUpTime += time.Now().Nanosecond() - this.lastTime.Nanosecond()
			this.totalUpCount++
			<-this.ticker.C
		}

		length := int32(len(b))
		// 等待时间毫米
		maxAwait := float64(this.KeepTime*length) / float64(this.readSize)

		wait = time.Nanosecond*time.Duration(maxAwait*1000)/1000 - time.Duration(this.consumingTime)
		if wait > 0 {
			if this.ticker == nil {
				this.ticker = time.NewTicker(wait)
			} else {
				this.ticker.Reset(wait)
			}

		} else {
			this.ticker = nil
		}
	}
	return wait.Nanoseconds()
}
