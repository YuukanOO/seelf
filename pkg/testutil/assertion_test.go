package testutil_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type testMock struct {
	testing.TB
	hasFailed bool
}

func (t *testMock) Fatalf(format string, args ...any) {
	// TODO: must test the error message too
	t.hasFailed = true
}

func Test_Equals(t *testing.T) {
	tests := []struct {
		expected   bool
		actual     bool
		shouldFail bool
	}{
		{true, false, true},
		{true, true, false},
		{false, false, false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.expected, test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.Equals(mock, test.expected, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_NotEquals(t *testing.T) {
	tests := []struct {
		expected   bool
		actual     bool
		shouldFail bool
	}{
		{true, true, true},
		{false, true, false},
		{false, false, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.expected, test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.NotEquals(mock, test.expected, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_DeepEquals(t *testing.T) {
	tests := []struct {
		expected   []bool
		actual     []bool
		shouldFail bool
	}{
		{[]bool{true, true}, []bool{false, true}, true},
		{[]bool{true, true}, []bool{true, true}, false},
		{[]bool{false, false}, []bool{false, true}, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.expected, test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.DeepEquals(mock, test.expected, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_IsTrue(t *testing.T) {
	tests := []struct {
		actual     bool
		shouldFail bool
	}{
		{true, false},
		{false, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.IsTrue(mock, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_IsFalse(t *testing.T) {
	tests := []struct {
		actual     bool
		shouldFail bool
	}{
		{true, true},
		{false, false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.IsFalse(mock, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_IsNil(t *testing.T) {
	tests := []struct {
		actual     any
		shouldFail bool
	}{
		{true, true},
		{nil, false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.IsNil(mock, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_IsNotNil(t *testing.T) {
	tests := []struct {
		actual     any
		shouldFail bool
	}{
		{true, false},
		{nil, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.IsNotNil(mock, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_HasLength(t *testing.T) {
	tests := []struct {
		expected   int
		actual     []int
		shouldFail bool
	}{
		{1, []int{1, 2}, true},
		{2, []int{1, 2}, false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.expected, test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.HasLength(mock, test.actual, test.expected)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_HasNChars(t *testing.T) {
	tests := []struct {
		expected   int
		actual     string
		shouldFail bool
	}{
		{5, "a long string", true},
		{2, "hi", false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.expected, test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.HasNChars(mock, test.expected, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_Contains(t *testing.T) {
	tests := []struct {
		value      string
		search     string
		shouldFail bool
	}{
		{"validation failed", "error", true},
		{"validation failed", "failed", false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.value, test.search), func(t *testing.T) {
			mock := new(testMock)

			testutil.Contains(mock, test.search, test.value)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_Match(t *testing.T) {
	tests := []struct {
		re         string
		value      string
		shouldFail bool
	}{
		{"abc", "error", true},
		{"abc", "abc", false},
		{"abc?", "ab", false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.value, test.re), func(t *testing.T) {
			mock := new(testMock)

			testutil.Match(mock, test.re, test.value)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func Test_ErrorIs(t *testing.T) {
	err := errors.New("some error")

	tests := []struct {
		expected   error
		actual     error
		shouldFail bool
	}{
		{err, errors.New("another one"), true},
		{err, err, false},
		{err, fmt.Errorf("with wrapped error %w", err), false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v %v", test.expected, test.actual), func(t *testing.T) {
			mock := new(testMock)

			testutil.ErrorIs(mock, test.expected, test.actual)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

type (
	domainEntity struct {
		event.Emitter
	}

	eventA struct {
		bus.Notification
		msg string
	}

	eventB struct {
		bus.Notification
		number int
	}
)

func (eventA) Name_() string { return "eventA" }
func (eventB) Name_() string { return "eventB" }

func Test_EventIs(t *testing.T) {
	var entity domainEntity

	entity = entity.apply(eventA{msg: "test"}).apply(eventB{number: 42})

	t.Run("should be able to retrieve an event if it exists", func(t *testing.T) {
		evt := testutil.EventIs[eventA](t, &entity, 0)
		evt2 := testutil.EventIs[eventB](t, &entity, 1)

		testutil.Equals(t, "test", evt.msg)
		testutil.Equals(t, 42, evt2.number)
	})

	t.Run("should fail if no events exists at all", func(t *testing.T) {
		mock := new(testMock)

		testutil.EventIs[eventA](mock, &domainEntity{}, 0)

		if !mock.hasFailed {
			t.Fail()
		}
	})

	t.Run("should fail if trying to access a not in range index", func(t *testing.T) {
		mock := new(testMock)

		testutil.EventIs[eventA](mock, &entity, 2)

		if !mock.hasFailed {
			t.Fail()
		}
	})

	t.Run("should fail if type does not match", func(t *testing.T) {
		mock := new(testMock)
		testutil.EventIs[eventB](mock, &entity, 0)

		if !mock.hasFailed {
			t.Fail()
		}
	})
}

func Test_HasNEvents(t *testing.T) {
	var entity domainEntity

	entity = entity.apply(eventA{msg: "test"}).apply(eventB{number: 42})

	tests := []struct {
		expected   int
		shouldFail bool
	}{
		{1, true},
		{2, false},
		{4, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.expected), func(t *testing.T) {
			mock := new(testMock)

			testutil.HasNEvents(mock, &entity, test.expected)

			if mock.hasFailed != test.shouldFail {
				t.Fail()
			}
		})
	}
}

func (d domainEntity) apply(e event.Event) domainEntity {
	event.Store(&d, e)
	return d
}
