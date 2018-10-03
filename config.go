package workers

import (
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
)

const (
	defaultRetryKey        = "goretry"
	defaultScheduleJobsKey = "schedule"
	defaultmaxRetries      = 11
	defaultpollInterval    = 15
)

type config struct {
	processID    string
	Namespace    string
	PollInterval int
	RetryKey     string
	ScheduleKey  string
	RetryLimit   int
	Pool         *redis.Pool
	Fetch        func(queue string) Fetcher
}

var Config *config

func Configure(options map[string]string) {
	var poolSize int
	var namespace string

	retryKey := defaultRetryKey
	pollInterval := defaultpollInterval
	retryLimit := defaultmaxRetries

	if options["server"] == "" {
		panic("Configure requires a 'server' option, which identifies a Redis instance")
	}

	if options["process"] == "" {
		panic("Configure requires a 'process' option, which uniquely identifies this instance")
	}

	if options["pool"] == "" {
		options["pool"] = "1"
	}

	if options["namespace"] != "" {
		namespace = options["namespace"] + ":"
	}

	if seconds, err := strconv.Atoi(options["poll_interval"]); err == nil {
		pollInterval = seconds
	}

	if customRetryKey, set := options["retry_key"]; set {
		retryKey = customRetryKey
	}

	if customRetryLimit, err := strconv.Atoi(options["retry_limit"]); err == nil {
		retryLimit = customRetryLimit
	}

	poolSize, _ = strconv.Atoi(options["pool"])

	Config = &config{
		options["process"],
		namespace,
		pollInterval,
		retryKey,
		defaultScheduleJobsKey,
		retryLimit,
		&redis.Pool{
			MaxIdle:     poolSize,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", options["server"])
				if err != nil {
					return nil, err
				}
				if options["password"] != "" {
					if _, err := c.Do("AUTH", options["password"]); err != nil {
						c.Close()
						return nil, err
					}
				}
				if options["database"] != "" {
					if _, err := c.Do("SELECT", options["database"]); err != nil {
						c.Close()
						return nil, err
					}
				}
				return c, err
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		},
		func(queue string) Fetcher {
			return NewFetch(queue, make(chan *Msg), make(chan bool))
		},
	}
}
