package strsli_to_strsli

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/constraints"
)

func doSomething(input string) string {
	return fmt.Sprintf("%s_%s", input, input)
}

func doSomethingWithProlog(prolog func()) doSomethingFunc {
	return func(input string) string {
		prolog()
		return doSomething(input)
	}
}

func mapToExpectedOutputs(inputs []string) []string {
	expectedOutputs := make([]string, len(inputs))
	for index, input := range inputs {
		expectedOutputs[index] = doSomething(input)
	}
	return expectedOutputs
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

func generateRandomInputs(count int64) []string {
	result := make([]string, count)
	for cursor := int64(0); cursor < count; cursor++ {
		for {
			if length := seededRand.Intn(maxInputLength); length > 0 {
				result[cursor] = generateRandomString(length)
				break
			}
		}
	}
	return result
}

var workerCounts []int = []int{1, runtime.NumCPU(), 1000}

type stringSliceToStringSliceTestSuite struct {
	suite.Suite
	t           *testing.T
	workerCount int
}

func TestStringSliceToStringSliceTestSuite(t *testing.T) {
	for index, workerCount := range workerCounts {
		fmt.Fprintf(os.Stderr, "#%02d workerCount: %d\n", index, workerCount)
		suite.Run(t, &stringSliceToStringSliceTestSuite{t: t, workerCount: workerCount})
	}
}

func (s *stringSliceToStringSliceTestSuite) TestWithOneInput() {
	origGoroutineCount := runtime.NumGoroutine()
	inputs := []string{"kalap"}
	outputs := process(context.Background(), doSomething, s.workerCount, inputs)
	s.Equal(mapToExpectedOutputs(inputs), outputs)
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithManyInput() {
	origGoroutineCount := runtime.NumGoroutine()

	inputs := generateRandomInputs(1000000)
	outputs := process(context.Background(), doSomething, s.workerCount, inputs)
	s.Equal(mapToExpectedOutputs(inputs), outputs)
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithDeadline() {
	origGoroutineCount := runtime.NumGoroutine()

	inputCount := int64(100)
	cancelAfterProcessed := int64(70)

	inputs := generateRandomInputs(inputCount)
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

	outputs := process(ctx, doSomethingWithProlog(prolog), s.workerCount, inputs)
	s.Nil(outputs)
	if s.workerCount <= runtime.NumCPU() {
		s.Less(processedCount.Load(), Min(cancelAfterProcessed+int64(runtime.NumCPU()), inputCount-2))
	}
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithNoDeadline() {
	origGoroutineCount := runtime.NumGoroutine()

	inputCount := int64(100)
	cancelAfterProcessed := int64(70)

	inputs := generateRandomInputs(inputCount)
	ctx := context.Background()

	processedCount := atomic.Int64{}
	prolog := func() {
		procedCount := processedCount.Add(1)
		if procedCount >= int64(cancelAfterProcessed) {
			time.Sleep(10 * time.Millisecond)
		}
	}

	outputs := process(ctx, doSomethingWithProlog(prolog), s.workerCount, inputs)
	s.Equal(mapToExpectedOutputs(inputs), outputs)
	s.Equal(int64(inputCount), processedCount.Load())
	waitForGoroutineNum(origGoroutineCount)
}

func (s *stringSliceToStringSliceTestSuite) TestWithDeadlineAfterMany() {
	origGoroutineCount := runtime.NumGoroutine()

	inputCount := int64(1_000_000)
	cancelAfterProcessed := int64(900_000)

	inputs := generateRandomInputs(inputCount)
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

	outputs := process(ctx, doSomethingWithProlog(prolog), s.workerCount, inputs)
	s.Nil(outputs)
	if s.workerCount <= runtime.NumCPU() {
		s.Less(processedCount.Load(), Min(cancelAfterProcessed+int64(runtime.NumCPU()), inputCount-2))
	}
	waitForGoroutineNum(origGoroutineCount)
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

func Min[O constraints.Ordered](a, b O) O {
	if a < b {
		return a
	}
	return b
}
