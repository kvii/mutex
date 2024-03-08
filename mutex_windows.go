// Package mutex 封装了 windows 的跨进程锁。
package mutex

import (
	"errors"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

var (
	// errWaitAbandoned 表明锁的上一任持有者在没有释放锁时就退出了。
	errWaitAbandoned = errors.New("mutex acquire: wait abandoned")
	// ErrDurationTooLong 表明传入的 duration 太长。
	ErrDurationTooLong = errors.New("mutex acquire: duration too long")
	// ErrWaitTimeout 表明等待锁的时间超过了指定的最长等待时间。
	ErrWaitTimeout = windows.WAIT_TIMEOUT
)

// 最长等待时间
const max_WAIT_MILLISECONDS = time.Duration(windows.INFINITE * time.Millisecond)

// Acquire 创建跨进程互斥锁。
// 返回 Releaser 的 Release 方法用于释放锁资源。它必须且只能被调用一次。
func Acquire(name string) (*Releaser, error) {
	return acquire(name, windows.INFINITE)
}

// AcquireWithTimeout 创建跨进程互斥锁，并指定最长等待时间。
// 返回 Releaser 的 Release 方法用于释放锁资源。它必须且只能被调用一次。
func AcquireWithTimeout(name string, timeout time.Duration) (*Releaser, error) {
	if timeout >= max_WAIT_MILLISECONDS {
		return nil, ErrDurationTooLong
	}
	return acquire(name, uint32(timeout.Milliseconds()))
}

func acquire(name string, waitMilliseconds uint32) (*Releaser, error) {
	ch := make(chan struct{})
	chE := make(chan error)

	go func() {
		// windows mutex 必须在同一个线程中操作。go 协程调度会导致线程切换，从而产生死锁。
		runtime.LockOSThread()

		defer close(chE)

		// https://learn.microsoft.com/zh-cn/windows/win32/api/synchapi/nf-synchapi-createmutexw
		mu, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(name))
		if err != nil && !errors.Is(err, syscall.ERROR_ALREADY_EXISTS) {
			chE <- err
			return
		}
		defer windows.CloseHandle(mu)

		// https://learn.microsoft.com/zh-cn/windows/win32/api/synchapi/nf-synchapi-waitforsingleobject
		rt, err := windows.WaitForSingleObject(mu, waitMilliseconds)
		if err != nil {
			chE <- err
			return
		}
		switch rt {
		case windows.WAIT_ABANDONED:
			chE <- errWaitAbandoned
			return
		case windows.WAIT_OBJECT_0:
			chE <- nil
		case uint32(windows.WAIT_TIMEOUT): // unreachable if waitMilliseconds is windows.INFINITE
			chE <- ErrWaitTimeout
			return
		default:
			panic("unreachable")
		}

		<-ch
		chE <- windows.ReleaseMutex(mu)
	}()

	err := <-chE
	isAbandoned := errors.Is(err, errWaitAbandoned)
	if err != nil && !isAbandoned {
		close(ch)
		return nil, err
	}

	return &Releaser{
		isAbandoned: isAbandoned,
		release:     func() error { close(ch); return <-chE },
	}, nil
}

// Releaser 用于释放锁资源。
type Releaser struct {
	isAbandoned bool
	release     func() error
}

// IsAbandoned 表明锁的上一任持有者是否在没有释放锁时就退出了。
// 这很可能是因为上一任持有者发生了严重错误。使用者应该检查被加锁的资源是否处于一致状态。
// 注意此时锁已经被当前使用者所持有了，使用者依然需要调用 Release 方法。
func (r *Releaser) IsAbandoned() bool {
	return r.isAbandoned
}

// Release 释放锁资源。该方法必须且只能被调用一次。
func (r *Releaser) Release() error {
	return r.release()
}
