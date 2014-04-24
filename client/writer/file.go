package writer

import (
	"os"
	"bytes"
	"crypto/sha1"
	"fmt"
	"time"
	"io"
	"log"
	"github.com/tsileo/silokv/rolling"
	"github.com/garyburd/redigo/redis"
)

func GetDbPool() (pool *redis.Pool, err error) {
	pool = &redis.Pool{
		MaxIdle:     50,
		MaxActive: 50,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "localhost:9736")
			if err != nil {
				return nil, err
			}
			//if _, err := c.Do("AUTH", password); err != nil {
			//    c.Close()
			//    return nil, err
			//}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return
}

func SHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

type WriteResult struct {
	Hash string
	Size int
	BlobsCnt int
	SkippedCnt int
	SkippedSize int
	UploadedCnt int
	UploadedSize int
}

func FileWriter(key, path string) (*WriteResult, error) {
	writeResult := &WriteResult{}
	window := 64
	rs := rolling.New(window)
	f, err := os.Open(path)
	if err != nil {
		return writeResult, err
	}
	rpool, _ := GetDbPool()
	con := rpool.Get()
	defer con.Close()
	var buf bytes.Buffer
	fullHash := sha1.New()
	eof := false
	for {
		b := make([]byte, 1)
		_, err := f.Read(b)
		if err == io.EOF {
			eof = true
		}
		rs.Write(b)
		buf.Write(b)
		onSplit := rs.OnSplit()
		if (onSplit && (buf.Len() > 64 << 10)) || buf.Len() >= 1 << 20 || eof {
			nsha := SHA1(buf.Bytes())
			ndata := string(buf.Bytes())
			fullHash.Write(buf.Bytes())
			exists, err := redis.Bool(con.Do("BEXISTS", nsha))
			log.Printf("exists:%v", exists)
			if err != nil {
				panic(fmt.Sprintf("DB error: %v", err))
			}
			if !exists {
				rsha, err := redis.String(con.Do("BPUT", ndata))
				if err != nil {
					panic(fmt.Sprintf("DB error: %v", err))
				}
				writeResult.UploadedCnt++
				writeResult.UploadedSize += buf.Len()
				if rsha != nsha {
					panic(fmt.Sprintf("Corrupted data: %+v/%+v", rsha, nsha))
				}
			} else {
				writeResult.SkippedSize += buf.Len()
				writeResult.SkippedCnt++
			}
			con.Do("BPADD", key, writeResult.BlobsCnt, nsha)
			writeResult.Size += buf.Len()
			buf.Reset()
			writeResult.BlobsCnt++
		}
		if eof {
			break
		}
	}
	writeResult.Hash = fmt.Sprintf("%x", fullHash.Sum(nil))
	return writeResult, nil
}