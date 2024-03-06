// Package mutex 封装了 windows 的跨进程锁。
package mutex

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

var (
	// ErrWaitAbandoned 表明锁的上一任持有者在没有释放锁时就退出了。
	// 这很可能是因为上一任持有者发生了严重错误。使用者应该检查被加锁的资源是否处于一致状态。
	// 注意此时锁已经被当前使用者所持有了，使用者依然需要调用 release 方法。
	ErrWaitAbandoned = errors.New("mutex acquire: wait abandoned")
	// ErrDurationTooLong 表明传入的 duration 太长。
	ErrDurationTooLong = errors.New("mutex acquire: duration too long")
	// ErrWaitTimeout 表明等待锁的时间超过了指定的最长等待时间。
	ErrWaitTimeout = windows.WAIT_TIMEOUT
)

// 最长等待时间
const max_WAIT_MILLISECONDS = time.Duration(windows.INFINITE * time.Millisecond)

// Acquire 创建一个新的跨进程互斥锁。
// 返回的 release 函数用于释放锁资源。它支持被多个协程多次调用，但必须调用一次。
func Acquire(name string) (release func() error, err error) {
	return acquire(name, windows.INFINITE)
}

// AcquireWithTimeout 创建一个新的跨进程互斥锁，并指定最长等待时间。
// 返回的 release 函数用于释放锁资源。它支持被多个协程多次调用，但必须调用一次。
func AcquireWithTimeout(name string, timeout time.Duration) (release func() error, err error) {
	if timeout >= max_WAIT_MILLISECONDS {
		return nil, ErrDurationTooLong
	}
	return acquire(name, uint32(timeout.Milliseconds()))
}

func acquire(name string, waitMilliseconds uint32) (release func() error, err error) {
	ch := make(chan struct{})
	chE := make(chan error)

	go func() {
		// windows mutex 必须在同一个线程中操作。go 协程调度会导致线程切换，从而产生死锁。
		runtime.LockOSThread()

		defer close(chE)

		mu, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(name))
		if err != nil && !errors.Is(err, syscall.ERROR_ALREADY_EXISTS) {
			chE <- err
			return
		}
		defer windows.CloseHandle(mu)

		rt, err := windows.WaitForSingleObject(mu, waitMilliseconds)
		if err != nil {
			chE <- err
			return
		}
		switch rt {
		case windows.WAIT_ABANDONED:
			chE <- ErrWaitAbandoned
			return
		case windows.WAIT_OBJECT_0:
			chE <- nil
		case uint32(windows.WAIT_TIMEOUT): // unreachable
			chE <- ErrWaitTimeout
			return
		default:
			panic("unreachable")
		}

		<-ch
		chE <- windows.ReleaseMutex(mu)
	}()

	if e := <-chE; e != nil {
		return nil, e
	}

	return sync.OnceValue(func() error { close(ch); return <-chE }), nil
}
