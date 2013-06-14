package main

import (
	"flag"
	"github.com/garyburd/redigo/redis"
	"github.com/sdming/mcache"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var redisServer = flag.String("server", "127.0.0.1:6379", "Redis server to connect to")
var httpPort = flag.String("p", ":8080", "listen port")
var hPort = flag.String("hp", "", "host post (useful for development)")
var urlMapFile = flag.String("f", "", "url map file in required format")

type RedirData struct {
	code string
	url  string
}

const (
	INTERNAL_REDIR = "I"
	EXTERNAL_REDIR = "301"
)

// <redir_code> <space> <target_url>
const FIELDS_COUNT = 2
const cacheTime = time.Second * 15

var hostsMap map[string]string
var Pool *redis.Pool
var Cache *mcache.MCache

type RedirHttpHandler struct {
	requests int64
	// TODO:
	// redis_hits
	// redis_miss
	// start_time
	// local_hits
	// local_miss
	// local_hard_served
	// local_hard_total
}

type RequestInfo struct {
	log       string
	redisTime int64
}

func (*RedirHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	reqInfo := &RequestInfo{log: r.URL.Path + ": ", redisTime: 0}
	reqInfo.handleRequest(w, r)

	delta := time.Now().Sub(startTime)
	ms := int64(delta / time.Microsecond)
	log.Println(reqInfo.log,
		" Stat: redis", reqInfo.redisTime, "total", ms, "micro sec (10^-6sec) ")
}

func (rI *RequestInfo) handleRequest(w http.ResponseWriter, r *http.Request) {
	_ = Cache

	reqUrl := r.URL.Path
	// TODO: process trailing slashes

	prefix, found := hostsMap[r.Host]
	if !found {
		rI.log += "ERROR: not found host: " + r.Host
		http.NotFound(w, r)
		return
	}

	res, err := rI.getRedir(prefix, reqUrl)
	if err != nil {
		rI.log += " WARNING: not match any"
		http.NotFound(w, r)
		return
	}

	redirUrl := res.url
	if len(r.URL.RawQuery) > 0 {
		redirUrl = redirUrl + "?" + r.URL.RawQuery
	}

	if res.code == INTERNAL_REDIR {
		rI.log += " I -> " + redirUrl
		w.Header().Add("X-Accel-Redirect", redirUrl)
	} else if res.code == EXTERNAL_REDIR {
		rI.log += " 301 -> " + redirUrl
		http.Redirect(w, r, redirUrl, http.StatusMovedPermanently)
	} else {
		rI.log += " redir error, unknown code " + res.code
		//log.Println("Error: unknown code: ", res.code)
		http.NotFound(w, r)
	}
}

func (rI *RequestInfo) getRedir(prefix, url string) (*RedirData, error) {
	if localStorage != nil {
		store, found := localStorage[prefix]
		if found {
			res, found := store[url]
			if found {
				rI.log += " *local*"
				return &res, nil
			}
		}
		rI.log += " *local miss* "
	}
	key := prefix + ":" + url
	return rI.getCacheRedisRedir(key)
}

func (rI *RequestInfo) getCacheRedisRedir(key string) (*RedirData, error) {
	var res *RedirData
	var resErr error

	resCached, cached := Cache.Get(key)
	if cached {
		res = resCached.(*RedirData)
	} else {
		res, resErr = rI.getRedisRedir(key)
		if resErr == nil {
			Cache.Set(key, res, cacheTime, mcache.AbsoluteExpiration)
		}
	}
	return res, resErr
}

func (rI *RequestInfo) getRedisRedir(key string) (*RedirData, error) {
	// go to redis
	var redisStart, redisEnd time.Time
	defer func() {
		deltaRedis := redisEnd.Sub(redisStart)
		rI.redisTime = int64(deltaRedis / time.Microsecond)
	}()

	redisStart = time.Now()
	redisConn := Pool.Get()
	// redis.ErrPoolExhausted
	defer redisConn.Close()

	// TODO:
	// check Do redis.ErrPoolExhausted
	redisData, err := redis.String(redisConn.Do("GET", key))
	redisEnd = time.Now()
	if err != nil {
		// TODO: check nil err 
		log.Println("redis error", err)
		return nil, err
	}

	fields := strings.Split(redisData, " ")
	if len(fields) != FIELDS_COUNT {
		log.Println("redis format error: ", redisData)
		return nil, err
	}
	return &RedirData{code: fields[0], url: fields[1]}, nil
}

func init() {
	flag.Usage = func() {
		log.Printf("Usage: urlmapper [flags]")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()

	Cache = mcache.NewMCache()

	hostsMap = map[string]string{
		"ua.tribuna.com" + *hPort:  "ua",
		"tribuna.com" + *hPort:     "ua",
		"www.tribuna.com" + *hPort: "ua",
		"www.sports.ru" + *hPort:   "ru",
	}

	if len(*urlMapFile) > 0 {
		go readUrlMapFileWorker(*urlMapFile)
	}
}

func main() {
	Pool = &redis.Pool{
		MaxActive:   1000,
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			d, _ := time.ParseDuration("250ms")
			c, err := redis.DialTimeout("tcp", *redisServer, d, d, d)
			if err != nil {
				log.Println("Redis: connection error")
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	http.Handle("/", &RedirHttpHandler{})
	http.ListenAndServe(*httpPort, nil)

	log.Println("end main")
}
