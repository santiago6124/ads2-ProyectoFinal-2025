package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisPipeline implements the Pipeline interface using Redis pipelining
type RedisPipeline struct {
	pipe redis.Pipeliner
}

// RedisCmd implements the Cmd interface for Redis commands
type RedisCmd struct {
	cmd redis.Cmder
	err error
}

func (c *RedisCmd) Err() error {
	if c.err != nil {
		return c.err
	}
	return c.cmd.Err()
}

// RedisStringCmd implements the StringCmd interface
type RedisStringCmd struct {
	*RedisCmd
	cmd *redis.StringCmd
}

func (c *RedisStringCmd) Result() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	val, err := c.cmd.Bytes()
	return val, err
}

func (c *RedisStringCmd) Val() []byte {
	if c.err != nil {
		return nil
	}
	val, _ := c.cmd.Bytes()
	return val
}

// RedisStatusCmd implements the StatusCmd interface
type RedisStatusCmd struct {
	*RedisCmd
	cmd *redis.StatusCmd
}

func (c *RedisStatusCmd) Result() (string, error) {
	if c.err != nil {
		return "", c.err
	}
	return c.cmd.Result()
}

func (c *RedisStatusCmd) Val() string {
	if c.err != nil {
		return ""
	}
	return c.cmd.Val()
}

// RedisIntCmd implements the IntCmd interface
type RedisIntCmd struct {
	*RedisCmd
	cmd *redis.IntCmd
}

func (c *RedisIntCmd) Result() (int64, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.cmd.Result()
}

func (c *RedisIntCmd) Val() int64 {
	if c.err != nil {
		return 0
	}
	return c.cmd.Val()
}

// RedisBoolCmd implements the BoolCmd interface
type RedisBoolCmd struct {
	*RedisCmd
	cmd *redis.BoolCmd
}

func (c *RedisBoolCmd) Result() (bool, error) {
	if c.err != nil {
		return false, c.err
	}
	return c.cmd.Result()
}

func (c *RedisBoolCmd) Val() bool {
	if c.err != nil {
		return false
	}
	return c.cmd.Val()
}

// RedisStringSliceCmd implements the StringSliceCmd interface
type RedisStringSliceCmd struct {
	*RedisCmd
	cmd *redis.StringSliceCmd
}

func (c *RedisStringSliceCmd) Result() ([][]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	vals, err := c.cmd.Result()
	if err != nil {
		return nil, err
	}

	result := make([][]byte, len(vals))
	for i, val := range vals {
		result[i] = []byte(val)
	}
	return result, nil
}

func (c *RedisStringSliceCmd) Val() [][]byte {
	vals, _ := c.Result()
	return vals
}

// RedisFloatCmd implements the FloatCmd interface
type RedisFloatCmd struct {
	*RedisCmd
	cmd *redis.FloatCmd
}

func (c *RedisFloatCmd) Result() (float64, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.cmd.Result()
}

func (c *RedisFloatCmd) Val() float64 {
	if c.err != nil {
		return 0
	}
	return c.cmd.Val()
}

// Pipeline operations

func (p *RedisPipeline) Get(key string) *StringCmd {
	cmd := p.pipe.Get(context.Background(), key)
	return &RedisStringCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) Set(key string, value []byte, ttl time.Duration) *StatusCmd {
	cmd := p.pipe.Set(context.Background(), key, value, ttl)
	return &RedisStatusCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) Del(keys ...string) *IntCmd {
	cmd := p.pipe.Del(context.Background(), keys...)
	return &RedisIntCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) HSet(key string, field string, value []byte) *BoolCmd {
	cmd := p.pipe.HSet(context.Background(), key, field, value)
	return &RedisBoolCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) HGet(key string, field string) *StringCmd {
	cmd := p.pipe.HGet(context.Background(), key, field)
	return &RedisStringCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) ZAdd(key string, score float64, member []byte) *IntCmd {
	cmd := p.pipe.ZAdd(context.Background(), key, redis.Z{
		Score:  score,
		Member: member,
	})
	return &RedisIntCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) ZRange(key string, start, stop int64) *StringSliceCmd {
	cmd := p.pipe.ZRange(context.Background(), key, start, stop)
	return &RedisStringSliceCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) Expire(key string, ttl time.Duration) *BoolCmd {
	cmd := p.pipe.Expire(context.Background(), key, ttl)
	return &RedisBoolCmd{
		RedisCmd: &RedisCmd{cmd: cmd},
		cmd:      cmd,
	}
}

func (p *RedisPipeline) Exec(ctx context.Context) ([]Cmd, error) {
	cmds, err := p.pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]Cmd, len(cmds))
	for i, cmd := range cmds {
		results[i] = &RedisCmd{cmd: cmd}
	}

	return results, nil
}

func (p *RedisPipeline) Discard() error {
	p.pipe.Discard()
	return nil
}