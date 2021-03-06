# redislocker  
使用方法:  
1.使用InitLockerInfo方法初始化redis服务器信息  
2.在需要使用互斥锁的地方用NewRedisLocker创建一个锁  
  (1)所有同名的锁视为同一个锁,因此无需共享Locker对象  
  (2)建议设置锁的有效期为一个较短的值,避免因服务器崩溃造成长时间死锁  
3.使用Locker的锁方法进行锁定,同步,方法返回结果确定是否获得锁  
  (1)TryLock只会进行一次锁尝试,成功获得锁返回true,失败返回false  
  (2)Lock会一直尝试获得锁,直到成功或者出现错误,除非必要,否则不建议使用  
  (3)LockWithTimeout会在设定的超时时间之内一直尝试获得锁,成功则返回true,超时或出错则返回false  
4.使用Locker的解锁方法解除锁定,异步,方法不等待返回结果  
  (1)Unlock解除当前的锁,并允许其他对象获得锁  
  (2)UnlockIfTimeout不做任何事情,当且仅当需要锁自动超时销毁的时候才使用该方法解锁  
  
  
建议:  
1.根据实际需求设置InitLockerInfo的connIdle参数  
2.不要全局共享Locker对象,应当在需要使用的时候创建(只要LockerName相同即可代表同一个锁)  
3.锁应该有一个超时时间,超过该时间后锁自动失效(即使不解锁也会被销毁)  
4.每个调用锁方法都需要调用的解锁方法  
5.一般情况下使用Unlock进行解锁,只有满足特殊需求的时候才使用UnlockIfTimeout解锁  

使用方法:  
```go
//初始化redis信息
func init() {
	InitLockerInfo("127.0.0.1:6379", 10)
}

//超时锁
func LockWithTimeoutAndUnlock(ret chan int, id int, t *testing.T) {
	t.Log(id, "启动")
	var locker = NewRedisLocker("LockAndUnlock123456", 30000) //锁有效期30s
	if locker != nil {
		if locker.LockWithTimeout(2000) {
			t.Log(id, "锁定并等待3秒超时")
			fmt.Println(id, "锁定", time.Now().UnixNano())
			time.Sleep(3 * time.Second)
			fmt.Println(id, "解除锁定", time.Now().UnixNano())
			locker.Unlock()
		} else {
			fmt.Println(id, "锁定失败", time.Now().UnixNano())
			t.Log(id, "锁定失败")
		}
	}
	ret <- id
	t.Log(id, "结束")
}

//阻塞锁
func LockAndUnlock(ret chan int, id int, t *testing.T) {
	t.Log(id, "启动")
	var locker = NewRedisLocker("LockAndUnlock12345", 30000) //锁有效期30s
	if locker != nil {
		if locker.Lock() {
			t.Log(id, "锁定并等待3秒超时")
			time.Sleep(3 * time.Second)
			fmt.Println(id, "锁定", time.Now().UnixNano())
			locker.Unlock()
		} else {
			fmt.Println(id, "锁定失败")
			t.Log(id, "锁定失败")
		}
	}
	ret <- id
	t.Log(id, "结束")
}

//尝试锁
func TryLockAndUnlock(ret chan int, id int, t *testing.T) {
	t.Log(id, "启动")
	var locker = NewRedisLocker("LockAndUnlock1234", 30000) //锁有效期30s
	if locker != nil {
		if locker.TryLock() {
			t.Log(id, "锁定并等待3秒超时")
			time.Sleep(3 * time.Second)
			fmt.Println(id, "锁定", time.Now().UnixNano())
			locker.Unlock()
		} else {
			fmt.Println(id, "锁定失败")
			t.Log(id, "锁定失败")
		}
	}
	ret <- id
	t.Log(id, "结束")
}

```