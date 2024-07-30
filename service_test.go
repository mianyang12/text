package test

import (
	"fmt"
	"reflect"
	"testing"
)

func (f main.Foo) Sum(args main.Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// it's not a exported Method
func (f main.Foo) sum(args main.Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: "+msg, v...))
	}
}

func TestNewService(t *testing.T) {
	var foo main.Foo
	s := newService(&foo)
	_assert(len(s.method) == 1, "wrong service Method, expect 1, but got %d", len(s.method))
	mType := s.method["Sum"]
	_assert(mType != nil, "wrong Method, Sum shouldn't nil")
}

func TestMethodType_Call(t *testing.T) {
	var foo main.Foo
	s := newService(&foo)
	mType := s.method["Sum"]

	argv := mType.newArgv()
	replyv := mType.newReplyv()
	argv.Set(reflect.ValueOf(main.Args{Num1: 1, Num2: 3}))
	err := s.call(mType, argv, replyv)
	_assert(err == nil && *replyv.Interface().(*int) == 4 && mType.NumCalls() == 1, "failed to call Foo.Sum")
}
