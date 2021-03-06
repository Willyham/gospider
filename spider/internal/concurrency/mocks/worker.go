package mocks

import mock "github.com/stretchr/testify/mock"

// Worker is an autogenerated mock type for the Worker type
type Worker struct {
	mock.Mock
}

// Work provides a mock function with given fields:
func (_m *Worker) Work() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
