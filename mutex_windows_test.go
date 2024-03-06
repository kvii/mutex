package mutex

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func ExampleAcquire() {
	r, err := Acquire("kvii_mutex_example_acquire")
	if err != nil && !errors.Is(err, ErrWaitAbandoned) {
		panic(err)
	}
	defer r()

	// Output:
}

func ExampleAcquireWithTimeout() {
	r, err := AcquireWithTimeout("kvii_mutex_example_acquire_with_timeout", time.Second)
	if err != nil && !errors.Is(err, ErrWaitAbandoned) {
		panic(err)
	}
	if errors.Is(err, ErrWaitTimeout) {
		// return
	}
	defer r()

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
		if e != nil && !errors.Is(e, ErrWaitAbandoned) {
			once.Do(func() { err = e })
		}
		defer r()

		i++
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, e := Acquire(name)
		if e != nil && !errors.Is(e, ErrWaitAbandoned) {
			once.Do(func() { err = e })
		}
		defer r()
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
	if err != nil && !errors.Is(err, ErrWaitAbandoned) {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r1() })

	r2, err := AcquireWithTimeout(name, time.Second)
	if !errors.Is(err, ErrWaitTimeout) {
		if err == nil || errors.Is(err, ErrWaitAbandoned) {
			_ = r2()
		}
		t.Fatalf("expect ErrWaitTimeout, got %v", err)
	}
}
