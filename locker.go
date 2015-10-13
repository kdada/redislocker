package locker

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"
	"unsafe"

	"github.com/garyburd/redigo/redis"
)

// 当一次Lock失败之后,会随机在RetryMinTimeInterval和RetryMaxTimeInterval之间
// 取一个随机值作为下次请求锁的时间间隔
// Lock请求锁的最小时间间隔(毫秒)
var RetryMinTimeInterval int64 = 5

// Lock请求锁的最大时间间隔(毫秒)
var RetryMaxTimeInterval int64 = 30

//redis连接池
var pool *redis.Pool

//服务器mac地址,用于作为服务器唯一标识
var macAddr string

// InitLockerInfo 初始化锁的链接信息,内部会维护一个redis连接池
// 该初始化方法必须在NewRedisLocker之前执行一次
// server:redis服务器地址
// connIdle:最大空闲连接数,该值的设置与并发量有关
func InitLockerInfo(server string, connIdle int) {
	var addrs, err = net.InterfaceAddrs()
	if err != nil || len(addrs) <= 0 {
		fmt.Println("[InitLockerInfo:29]", "获取服务器MAC地址失败", err.Error())
		return
	}
	macAddr = addrs[0].String()
	pool = redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", server)
	}, connIdle)
}

// generateLockerId 生成全局唯一id
// [mac地址]+[纳秒时间戳]+[随机数]
func generateLockerId() string {
	var x = new(byte)
	return macAddr + strconv.Itoa(int(time.Now().UnixNano())) + strconv.Itoa(int(uintptr(unsafe.Pointer(x))))
}

// 默认解锁脚本
const defaultUnlockScript = `
if redis.call("get",KEYS[1]) == ARGV[1] then
    return redis.call("del",KEYS[1])
else
    return 0
end
`

// 默认解锁脚本的SHA1校验码
var defaultUnlockScriptSHA1 = ""

// init 计算defaultUnlockScript脚本的SHA1值
func init() {
	var sha = sha1.New()
	sha.Write([]byte(defaultUnlockScript))
	defaultUnlockScriptSHA1 = fmt.Sprintf("%x", sha.Sum(nil))
}

// setRedisKV 设置指定的KV
// key:键名
// value:键值
// timeout:键的存活时间(毫秒)
// return:成功返回"OK",否则返回nil
func setRedisKV(key, value string, timeout int64) (interface{}, error) {
	if conn := pool.Get(); conn != nil {
		defer conn.Close()
		return conn.Do("SET", key, value, "NX", "PX", timeout)
	}
	return nil, errors.New("SET无法取得有效redis链接")
}

// deleteRedisKV 删除指定的KV,只有KV完全一致的时候才删除
// key:键名
// value:键值
// return:成功返回1,否则返回0,出错返回nil
func deleteRedisKV(key, value string) (interface{}, error) {
	if conn := pool.Get(); conn != nil {
		defer conn.Close()
		var result, err = conn.Do("EVALSHA", defaultUnlockScriptSHA1, 1, key, value)
		if err != nil {
			// 如果EVALSHA失败则使用EVAL执行脚本
			result, err = conn.Do("EVAL", defaultUnlockScript, 1, key, value)
		}
		return result, err
	}
	return nil, errors.New("DEL无法取得有效redis链接")
}

// sleepTimeInterval 随机休眠一段时间
// 随机时间范围[RetryMinTimeInterval,RetryMaxTimeInterval)
func sleepTimeInterval() {
	var unixNano = time.Now().UnixNano()
	var r = rand.New(rand.NewSource(unixNano))
	var randValue = RetryMinTimeInterval + r.Int63n(RetryMaxTimeInterval-RetryMinTimeInterval)
	time.Sleep(time.Duration(randValue) * time.Millisecond)
}

// 基于Redis的分布式锁
// 目前仅支持单Redis服务器
type Locker struct {
	LockerName string //锁名称
	Timeout    int64  //锁超时时间(毫秒),设置一个较小的值可以降低死锁带来的影响
	LockerId   string //锁Id,该Id为GUID
}

// NewRedisLocker 创建一个Redis锁
// 在创建锁之前必须使用InitLockerInfo先进行初始化
// 如果连接池未初始化,则无法创建Locker
// name:锁名称
// timeout:锁的超时时间(毫秒)
func NewRedisLocker(name string, timeout int64) *Locker {
	if pool == nil {
		return nil
	}
	var locker = new(Locker)
	locker.LockerName = name + ".rl"
	locker.Timeout = timeout
	locker.LockerId = generateLockerId()
	return locker
}

// TryLock 尝试锁,只会进行一次尝试
// return:true,成功取得锁,false,取得锁的过程中出现错误
func (this *Locker) TryLock() bool {
	var result, err = setRedisKV(this.LockerName, this.LockerId, this.Timeout)
	if err == nil && result == "OK" {
		return true
	}
	if err != nil {
		fmt.Println("Locker TryLock Error", err)
	}
	return false
}

// Lock 同步锁,将会一直等待,直到可以获得锁,每隔一段时间会进行一次尝试,不建议使用
// return:true,成功取得锁,false,取得锁的过程中出现错误
func (this *Locker) Lock() bool {
	for true {
		var result, err = setRedisKV(this.LockerName, this.LockerId, this.Timeout)
		if err == nil && result == "OK" {
			return true
		}
		if err != nil {
			fmt.Println("Locker Lock Error", err)
			break
		}
		sleepTimeInterval()
	}
	return false
}

// LockWithTimeout 超时锁,将会一直等待锁,直到获得锁或者超时
// timeout:超时时间(毫秒)
// return:true,成功取得锁,false,取得锁的过程中出现错误或者超时
func (this *Locker) LockWithTimeout(timeout int64) bool {
	var deadline = time.Now().UnixNano() + timeout*int64(time.Millisecond)
	var result interface{} = nil
	var err error = nil
	for deadline > time.Now().UnixNano() {
		result, err = setRedisKV(this.LockerName, this.LockerId, this.Timeout)
		if err == nil && result == "OK" {
			return true
		}
		if err != nil {
			fmt.Println("Locker LockWithTimeout Error", err)
			break
		}
		sleepTimeInterval()
	}
	return false
}

// Unlock 解除锁,将根据锁名称解除相应的锁
func (this *Locker) Unlock() {
	var _, err = deleteRedisKV(this.LockerName, this.LockerId)
	if err != nil {
		fmt.Println("Locker Unlock Error", err)
	}
}

// UnlockIfTimeout 根据超时时间自动解除锁(由redis超时完成)
// 本方法不实现,但是每个Lock必须有对应的Unlock方法
func (this *Locker) UnlockIfTimeout() {

}
