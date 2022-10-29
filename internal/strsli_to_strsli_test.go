package strsli_to_strsli

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func doSomethingWithProlog(prolog func()) doSomethingFunc {
	return func(input string) string {
		prolog()
		return fmt.Sprintf("%s_%s", input, input)
	}
}

func doSomething(input string) string {
	return fmt.Sprintf("%s_%s", input, input)
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

const maxInputLength = 10

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomInputs(count int) []string {
	result := make([]string, count)
	for cursor := 0; cursor < count; cursor++ {
		for {
			if length := seededRand.Intn(maxInputLength); length > 0 {
				result[cursor] = generateRandomString(length)
				break
			}
		}
	}
	return result
}

type stringSliceToStringSliceTestSuite struct {
	suite.Suite
	t      *testing.T
	inputs []string
}

func TestStringSliceToStringSliceTestSuite(t *testing.T) {
	suite.Run(t, &stringSliceToStringSliceTestSuite{t: t})
}

func (s *stringSliceToStringSliceTestSuite) TestWithOneInput() {
	origGoroutineCount := runtime.NumGoroutine()

	s.inputs = []string{"kalap"}
	outputs := process(context.Background(), doSomething, 1, s.inputs)
	s.myAssert(outputs)
	waitForGoroutineNum(origGoroutineCount)
	outputs = process(context.Background(), doSomething, 1000, s.inputs)
	s.myAssert(outputs)
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithManyInput() {
	origGoroutineCount := runtime.NumGoroutine()

	s.inputs = generateRandomInputs(1000000)
	outputs := process(context.Background(), doSomething, 1, s.inputs)
	s.myAssert(outputs)
	waitForGoroutineNum(origGoroutineCount)
	outputs = process(context.Background(), doSomething, 1000, s.inputs)
	s.myAssert(outputs)
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithDeadline() {
	origGoroutineCount := runtime.NumGoroutine()

	inputCount := 10
	cancelAfterProcessed := 5

	s.inputs = generateRandomInputs(inputCount)
	ctx, cancel := context.WithCancel(context.Background())

	processedCount := atomic.Int64{}
	prolog := func() {
		procedCount := processedCount.Add(1)
		if procedCount == int64(cancelAfterProcessed) {
			cancel()
		}
		if procedCount >= int64(cancelAfterProcessed) {
			time.Sleep(10 * time.Millisecond)
		}
	}

	outputs := process(ctx, doSomethingWithProlog(prolog), 1, s.inputs)
	s.Nil(outputs)
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithNoDeadline() {
	origGoroutineCount := runtime.NumGoroutine()

	inputCount := 10
	cancelAfterProcessed := 5

	s.inputs = generateRandomInputs(inputCount)
	ctx := context.Background()

	processedCount := atomic.Int64{}
	prolog := func() {
		procedCount := processedCount.Add(1)
		if procedCount >= int64(cancelAfterProcessed) {
			time.Sleep(10 * time.Millisecond)
		}
	}

	outputs := process(ctx, doSomethingWithProlog(prolog), 1, s.inputs)
	s.myAssert(outputs)
	s.Equal(int64(inputCount), processedCount.Load())
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithDeadlineAfterMany() {
	origGoroutineCount := runtime.NumGoroutine()

	inputCount := 1_000_000
	cancelAfterProcessed := 900_000

	s.inputs = generateRandomInputs(inputCount)
	ctx, cancel := context.WithCancel(context.Background())

	processedCount := atomic.Int64{}
	prolog := func() {
		procedCount := processedCount.Add(1)
		if procedCount == int64(cancelAfterProcessed) {
			cancel()
		}
		if procedCount >= int64(cancelAfterProcessed) {
			time.Sleep(10 * time.Millisecond)
		}
	}

	outputs := process(ctx, doSomethingWithProlog(prolog), 1000, s.inputs)
	s.Nil(outputs)
	time.Sleep(5 * time.Second)
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) myAssert(result []string) {
	for index, inp := range s.inputs {
		s.Equal(fmt.Sprintf("%s_%s", inp, inp), result[index])
	}
}

// Checking runtime.NumGoroutine() is probably an overkill, because this is
// what the WaitGroup is in the tested code in the first place. Just for completeness.
func waitForGoroutineNum(expectedNum int) error {
	totalWaitMs := 5000
	waitIntervalMs := 100

	retryCount := totalWaitMs / waitIntervalMs

	var actualNum int
	for retry := 0; retry < retryCount; retry++ {
		actualNum = runtime.NumGoroutine()
		if actualNum == expectedNum {
			return nil
		}
		time.Sleep(time.Duration(waitIntervalMs) * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for %d to become %d", actualNum, expectedNum)
}
