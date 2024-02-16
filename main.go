package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

func repeatFunc[T any, K any](done <-chan K, f func() T) <-chan T {
	stream := make(chan T)

	go func() {
		defer close(stream)

		for {
			select {
			case <-done:
				return
			case stream <- f():
			}
		}
	}()

	return stream
}

func take[T any, K any](done <-chan K, stream <-chan T, n int) <-chan T {
	takeStream := make(chan T)

	go func() {
		defer close(takeStream)

		for i := 0; i < n; i++ {
			select {
			case <-done:
				return
			case takeStream <- <-stream:
			}
		}
	}()

	return takeStream
}

func primeFinder(done <-chan int, randNum <-chan int) <-chan int {
	isPrime := func(n int) bool {
		for i := n - 1; i > 1; i-- {
			if n%i == 0 {
				return false
			}
		}
		return true
	}

	primeStream := make(chan int)

	go func() {
		defer close(primeStream)

		for {
			select {
			case <-done:
				return
			case num := <-randNum:
				if isPrime(num) {
					primeStream <- num
				}
			}
		}
	}()

	return primeStream
}

func fanIn[T any](done <-chan int, channels ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	fannedInStream := make(chan T)

	transfer := func(c <-chan T) {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			case fannedInStream <- <-c:
			}
		}
	}

	for _, c := range channels {
		wg.Add(1)
		go transfer(c)
	}

	go func() {
		wg.Wait()
		close(fannedInStream)
	}()

	return fannedInStream
}

func main() {
	start := time.Now()
	done := make(chan int)
	defer close(done)

	randNumFetcher := func() int { return rand.Intn(500000000) }
	randIntStream := repeatFunc(done, randNumFetcher)

	// Naive way to print the first 10 prime numbers
	// primeStream := primeFinder(done, randIntStream)

	// for prime := range take(done, primeStream, 10) {
	// 	fmt.Println(prime)
	// }

	// Fan Out / Fan In Pattern
	// Fan Out
	CPUCount := runtime.NumCPU()
	fmt.Println("CPU Count:", CPUCount)
	primeFinderChannels := make([]<-chan int, CPUCount)
	for i := 0; i < CPUCount; i++ {
		primeFinderChannels[i] = primeFinder(done, randIntStream)
	}

	// Fan In
	fanIn(done, primeFinderChannels...)

	for prime := range take(done, fanIn(done, primeFinderChannels...), 10) {
		fmt.Println(prime)
	}

	fmt.Println("Time taken:", time.Since(start))
}
