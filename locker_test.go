package locker

import (
	"time"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/garyburd/redigo/redis"
)

func TestMain(m *testing.M) {
	InitLockerInfo("192.168.1.220:6379", 10)
	os.Exit(m.Run())
}

// 测试基本方法
func TestSetRedisKV(t *testing.T) {
	var k = "adasdasdas"
	var v = "asdasdasdasd"
	t.Log("START")
	var r, e = setRedisKV(k, v, 5000)
	if e != nil {
		t.Error(e)
		return
	}
	if r != "OK" {
		t.Error("设置失败", r)
		return
	}
	t.Log("设置完成", r, reflect.TypeOf(r).Name())
	var conn = pool.Get()
	if conn != nil {
		r, e = conn.Do("GET", k)
	}
	if e != nil {
		t.Error(e)
		return
	}
	if r == nil || r == "" {
		t.Error("获取失败", r)
		return
	}
	var rs, _ = redis.String(r, nil)
	t.Log("获取成功", rs, reflect.TypeOf(r).Name())
	r, e = deleteRedisKV(k, v)
	if e != nil {
		t.Error(e)
		return
	}
	if r == nil {
		t.Error("删除失败", r)
		return
	}
	var ri, _ = redis.Int64(r, nil)
	t.Log("删除完成", ri, reflect.TypeOf(r).Name())
	r, e = pool.Get().Do("GET", k)
	if e != nil {
		t.Error(e)
		return
	}
	if r != nil {
		t.Error("获取错误", r)
		return
	}
}

func LockWithTimeoutAndUnlock(ret chan int, id int, t *testing.T) {
	t.Log(id, "启动")
	var locker = NewRedisLocker("LockAndUnlock123456", 30000)//锁有效期30s
	if locker != nil {
		if locker.LockWithTimeout(2000) {
			t.Log(id, "锁定并等待3秒超时")
			fmt.Println(id,"锁定",time.Now().UnixNano())
			time.Sleep(3*time.Second)
			fmt.Println(id,"解除锁定",time.Now().UnixNano())
			locker.Unlock()
		} else {
			fmt.Println(id,"锁定失败",time.Now().UnixNano())
			t.Log(id, "锁定失败")
		}
	}
	ret <- id
	t.Log(id, "结束")
}
func TestLockWithTimeout(t *testing.T) {
	var count = 10
	var ret = make(chan int, count)
	for a := 0; a < count; a++ {
		go LockWithTimeoutAndUnlock(ret, a, t)
	}
	t.Log("等待执行结果")
	for a := 0; a < count; a++ {
		var id = <-ret
		t.Log(id, "执行完成")
	}
	t.Log("完毕")
}


func LockAndUnlock(ret chan int, id int, t *testing.T) {
	t.Log(id, "启动")
	var locker = NewRedisLocker("LockAndUnlock12345", 30000)//锁有效期30s
	if locker != nil {
		if locker.Lock() {
			t.Log(id, "锁定并等待3秒超时")
			time.Sleep(3*time.Second)
			fmt.Println(id,"锁定",time.Now().UnixNano())
			locker.Unlock()
		} else {
			fmt.Println(id,"锁定失败")
			t.Log(id, "锁定失败")
		}
	}
	ret <- id
	t.Log(id, "结束")
}

func TestLock(t *testing.T) {
	var count = 10
	var ret = make(chan int, count)
	for a := 0; a < count; a++ {
		go LockAndUnlock(ret, a, t)
	}
	t.Log("等待执行结果")
	for a := 0; a < count; a++ {
		var id = <-ret
		t.Log(id, "执行完成")
	}
	t.Log("完毕")
}

func TryLockAndUnlock(ret chan int, id int, t *testing.T) {
	t.Log(id, "启动")
	var locker = NewRedisLocker("LockAndUnlock1234", 30000)//锁有效期30s
	if locker != nil {
		if locker.TryLock() {
			t.Log(id, "锁定并等待3秒超时")
			time.Sleep(3*time.Second)
			fmt.Println(id,"锁定",time.Now().UnixNano())
			locker.Unlock()
		} else {
			fmt.Println(id,"锁定失败")
			t.Log(id, "锁定失败")
		}
	}
	ret <- id
	t.Log(id, "结束")
}



func TestTryLock(t *testing.T) {
	var count = 100
	var ret = make(chan int, count)
	for a := 0; a < count; a++ {
		go TryLockAndUnlock(ret, a, t)
	}
	t.Log("等待执行结果")
	for a := 0; a < count; a++ {
		var id = <-ret
		t.Log(id, "执行完成")
	}
	t.Log("完毕")
}