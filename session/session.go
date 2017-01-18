package session

import (
	"bytes"
	"encoding"
	"encoding/gob"
	"errors"
	"sync"
	"time"

	"github.com/O-C-R/auth/id"
	"github.com/garyburd/redigo/redis"
)

// Arguments: current unix timestamp (nanoseconds), rate (tokens per nanosecond), bucket capacity.
const tokenBucket = `
local bucket = redis.call('hmget', KEYS[1], '1', '2')
if(not bucket[1]) then
	bucket[1] = 0
end

if(not bucket[2]) then
	bucket[2] = tonumber(ARGV[2])
elseif(ARGV[3] > bucket[1]) then
	bucket[2] = math.min(ARGV[2], bucket[2] + (ARGV[3] - bucket[1]) * ARGV[1])
end

local ok = 0
if(bucket[2]>0) then
	bucket[2] = bucket[2] - 1
	ok = 1
end

redis.call('hmset', KEYS[1], '1', ARGV[3], '2', bucket[2])
redis.call('pexpire', KEYS[1], math.ceil((ARGV[2] - bucket[2]) / ARGV[1] / 1e3))

return ok
`

var (
	RateLimitExceededError = errors.New("rate limit exceeded")
	redisError             = errors.New("redis error")
	tokenBucketScript      = redis.NewScript(1, tokenBucket)
)

func interfaceToString(v interface{}) (string, error) {
	switch v := v.(type) {
	default:
		return "", errors.New("Must provide a string-like object")
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case encoding.TextMarshaler:
		bytes, err := v.MarshalText()
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
}

func sessionKey(sessionID id.ID) string {
	return "s" + sessionID.String()
}

func sessionToGroupKey(sessionID id.ID) string {
	return "z" + sessionID.String()
}

func groupKey(groupId string) string {
	return "g" + groupId
}

func rateLimitKey(client string) string {
	return "b" + client
}

type SessionStoreOptions struct {
	Addr, Password  string
	SessionDuration time.Duration
}

type SessionStore struct {
	pool                                          *redis.Pool
	sessionDuration, rateLimitDuration, rateLimit int64
}

func NewSessionStore(options SessionStoreOptions) (*SessionStore, error) {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 5 * time.Minute,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", options.Addr)
			if err != nil {
				return nil, err
			}

			if options.Password != "" {
				if _, err := conn.Do("AUTH", options.Password); err != nil {
					conn.Close()
					return nil, err
				}
			}

			return conn, err
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}

			_, err := conn.Do("PING")
			return err
		},
	}

	conn := pool.Get()
	defer conn.Close()

	// Load the token bucket script.
	if err := tokenBucketScript.Load(conn); err != nil {
		return nil, err
	}

	return &SessionStore{
		pool:            pool,
		sessionDuration: int64(options.SessionDuration / time.Second),
	}, nil
}

func (r *SessionStore) Session(sessionID id.ID, session interface{}) error {
	conn := r.pool.Get()
	defer conn.Close()

	reply, err := redis.Bytes(conn.Do("GET", sessionKey(sessionID)))
	if err != nil {
		return err
	}

	return gob.NewDecoder(bytes.NewBuffer(reply)).Decode(session)
}

func (r *SessionStore) SetSession(sessionID id.ID, groupId interface{}, session interface{}) error {
	conn := r.pool.Get()
	defer conn.Close()

	encodedSession := bytes.NewBuffer([]byte{})
	if err := gob.NewEncoder(encodedSession).Encode(session); err != nil {
		return err
	}

	sKey := sessionKey(sessionID)

	if _, err := conn.Do("SETEX", sKey, r.sessionDuration, encodedSession); err != nil {
		return err
	}

	if groupId != nil {
		groupIdStr, err := interfaceToString(groupId)
		if err != nil {
			return err
		}

		gKey := groupKey(groupIdStr)
		sgKey := sessionToGroupKey(sessionID)

		if _, err := conn.Do("SETEX", sgKey, r.sessionDuration, groupIdStr); err != nil {
			return err
		}

		if _, err := conn.Do("SADD", gKey, sKey, sgKey); err != nil {
			return err
		}

		if _, err := conn.Do("EXPIRE", gKey, r.sessionDuration); err != nil {
			return err
		}
	}

	return nil
}

func (r *SessionStore) InvalidateSessions(groupId interface{}) error {
	conn := r.pool.Get()
	defer conn.Close()

	groupIdStr, err := interfaceToString(groupId)
	if err != nil {
		return err
	}
	gKey := groupKey(groupIdStr)

	// TODO: use sscan for safety
	members, err := redis.Strings(conn.Do("SMEMBERS", gKey))
	if err != nil {
		return err
	}

	if _, err := conn.Do("DEL", redis.Args{}.Add(gKey).AddFlat(members)...); err != nil {
		return err
	}

	return nil
}

func (r *SessionStore) DeleteSession(sessionID id.ID) error {
	conn := r.pool.Get()
	defer conn.Close()

	sKey := sessionKey(sessionID)

	if _, err := conn.Do("DEL", sKey); err != nil {
		return err
	}

	sgKey := sessionToGroupKey(sessionID)
	groupId, err := redis.String(conn.Do("GET", sgKey))
	if err == nil {
		gKey := groupKey(groupId)

		if _, err := conn.Do("DEL", sgKey); err != nil {
			return err
		}

		if _, err := conn.Do("SREM", gKey, sKey, sgKey); err != nil {
			return err
		}
	}

	return nil
}

func (r *SessionStore) RateLimitCount(client string, bucketRate, bucketCapacity float64) error {
	conn := r.pool.Get()
	defer conn.Close()

	ok, err := redis.Int(tokenBucketScript.Do(conn, rateLimitKey(client), bucketRate, bucketCapacity, time.Now().UnixNano()))
	if err != nil {
		return err
	}

	if ok == 0 {
		return RateLimitExceededError
	}

	return nil
}
