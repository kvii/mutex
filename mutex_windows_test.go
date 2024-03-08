package mutex

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func ExampleAcquire() {
	r, err := Acquire("kvii_mutex_example_acquire")
	if err != nil {
		panic(err)
	}
	defer r.Release()

	if r.IsAbandoned() {
		// 检查被加锁的资源是否处于一致状态
	}

	// Output:
}

func ExampleAcquireWithTimeout() {
	r, err := AcquireWithTimeout("kvii_mutex_example_acquire_with_timeout", time.Second)
	if err != nil {
		panic(err)
	}
	defer r.Release()

	if r.IsAbandoned() {
		// 检查被加锁的资源是否处于一致状态
	}

	// Output:
}

func TestAcquire(t *testing.T) {
	const name = "kvii_mutex_test_acquire"
	var wg sync.WaitGroup
	var err error
	var once sync.Once
	var i int

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, e := Acquire(name)
		if e != nil {
			once.Do(func() { err = e })
		}
		defer r.Release()

		if r.IsAbandoned() {
			t.Log("abandoned")
		}
		i++
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, e := Acquire(name)
		if e != nil {
			once.Do(func() { err = e })
		}
		defer r.Release()

		if r.IsAbandoned() {
			t.Log("abandoned")
		}
		i++
	}()

	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if i != 2 {
		t.Fatalf("expect 2, got %d", i)
	}
}

func TestAcquireWithTimeout(t *testing.T) {
	const name = "kvii_mutex_test_acquire_with_timeout"

	r1, err := Acquire(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r1.Release() })

	r2, err := AcquireWithTimeout(name, time.Second)
	if !errors.Is(err, ErrWaitTimeout) {
		if err == nil {
			_ = r2.Release()
		}
		t.Fatalf("expect ErrWaitTimeout, got %v", err)
	}
}
