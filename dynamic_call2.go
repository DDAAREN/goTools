package common

import (
	"errors"
	"reflect"
	"regexp"
)

var (
	ErrNotFunc    = errors.New("The object to bind is not the function")
	ErrParamsNum  = errors.New("The number of params is not adapted")
	ErrParamsType = errors.New("The type of the params is incorrect")
)

type Funcs map[string]reflect.Value

func NewFuncs(size int) Funcs {
	return make(Funcs, size)
}

func (f Funcs) Bind(name string, fn interface{}) (err error) {
	fv := reflect.ValueOf(fn)
	if fv.Kind() != reflect.Func {
		err = ErrNotFunc
		return
	}

	f[name] = fv
	err = nil
	return
}

func (f Funcs) Call(name string, params ...interface{}) (results []interface{}, err error) {
	defer func() {
		if err := recover(); err != nil {
			results = nil
		}
	}()

	if _, ok := f[name]; !ok {
		err = errors.New(name + " does not exist")
		return
	}

	ft := f[name].Type()
	if len(params) != ft.NumIn() {
		err = ErrParamsNum
		return
	}

	in := make([]reflect.Value, len(params))
	for i, param := range params {
		typ := ft.In(i).Kind()
		if (typ != reflect.Interface) && (typ != reflect.TypeOf(param).Kind()) {
			err = ErrParamsType
			return
		}

		in[i] = reflect.ValueOf(param)
	}

	ret = f[name].Call(in)
	for _, r := range ret {
		results = append(results, r.Interface())
	}
	return
}
