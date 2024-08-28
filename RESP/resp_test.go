package resp

import (
	"errors"
	"fmt"
	gttype "github.com/BeginerAndProgresses/generalized-tools/type"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
)

/*
 * 说明：
 * 作者：吕元龙
 * 时间 2024/8/27 18:19
 */

func TestByteToInt(t *testing.T) {
	byts := []byte("123")
	strconv.ParseInt(string(byts), 10, 64)
	//fmt.Printf("%T\n")
	//fmt.Println(bytes[0])
}

func TestSplitBytes(t *testing.T) {
	s := []byte(`*2\r\n$3\r\nGET\r\n$1\r\na\r\n`)
	bytes := splitBytes(s, []byte(`\r\n`))
	for i, i2 := range bytes {
		fmt.Printf("%d:%s\n", i, string(i2))

	}
}

func TestRESPParser(t *testing.T) {
	testCase := []struct {
		name   string
		row    []byte
		res    any
		before func() any
	}{
		{
			name: "测试Array",
			row:  []byte("*2\r\n3\r\n2324\r\n"),
			res:  Array{"3", "2324"},
		},
		{
			name: "测试BulkString",
			row:  []byte("$3\r\n232\r\n"),
			res:  BulkStrings("232"),
		},
		{
			name: "測試BulkErr",
			row:  []byte("!3\r\nERR\r\n"),
			res: MultiErr(errors.New(
				"ERR")),
		},
		{
			name: "測試Verbatim",
			row:  []byte("=15\r\ntxt:Some string\r\n"),
			res: Verbatim{
				Coding: "txt",
				Data:   []byte("Some string"),
			},
		},
		{
			name: "測試簡單String",
			row:  []byte("+OK\r\n"),
			res:  "OK",
		},
		{
			name: "测试Int",
			row:  []byte(":2324\r\n"),
			res:  int64(2324),
		},
		{
			name: "測試Map",
			row:  []byte("%2\r\n+first\r\n:1\r\n+second\r\n:2\r\n"),
			res: Maps{
				"first":  int64(1),
				"second": int64(2),
			},
		},
		// 值相同，但是地址不相同
		//{
		//	name: "测试Sets",
		//	row:  []byte(`~2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n`),
		//	res:  gttype.NewHashSet[any]().Add("foo").Add("bar"),
		//},
		// 值相同，但是地址不相同
		//{
		//	name: "測試Push堆",
		//	row:  []byte(`>2\r\n1\r\n2\r\n`),
		//	res:  nil,
		//	before: func() any {
		//		res := gttype.NewHeap[any]()
		//		res.Insert("1")
		//		res.Insert("2")
		//		return res
		//	},
		//},
		{
			name: "测试負Int",
			row:  []byte(":-2324\r\n"),
			res:  int64(-2324),
		},
		{
			name: "测试正Int",
			row:  []byte(":+2324\r\n"),
			res:  int64(2324),
		},
		{
			name: "测试Nil",
			row:  []byte("_\r\n"),
			res:  nil,
		},
		{
			name: "测试Error",
			row:  []byte("-ERR\r\n"),
			res:  errors.New("ERR"),
		},
		{
			name: "测试Double",
			row:  []byte(",1.23\r\n"),
			res:  float64(1.23),
		},
		{
			name: "测试Bool",
			row:  []byte("#f\r\n"),
			res:  false,
		},
		{
			name: "测试大數",
			row:  []byte("(3492890328409238509324850943850943825024385\r\n"),
			before: func() any {
				b := new(big.Int)
				b.SetString("3492890328409238509324850943850943825024385", 10)
				return b
			},
		},
		{
			name: "测试聯合值",
			row:  []byte("*2\r\n$3\r\nget\r\n$1\r\na\r\n"),
			res:  Array{BulkStrings("get"), BulkStrings("a")},
		},
	}
	res := NewRESP()
	for _, v := range testCase {
		t.Run(v.name, func(t *testing.T) {
			if v.before != nil {
				v.res = v.before()
			}
			after, parse := res.Parse(v.row)
			if len(after) > 0 {
				t.Logf("after:%s", after)
			}
			t.Logf("sum:%v,%T", parse, parse)
			assert.Equal(t, v.res, parse, "結果應該相同")
		})
	}
}

func TestRespSvc_BuildingRedisExecuteRESP(t *testing.T) {
	testCases := []struct {
		name   string
		data   any
		res    []byte
		before func() any
	}{
		{
			name: "测试Array",
			data: Array{BulkStrings("3"), BulkStrings("2324")},
			res:  []byte("*2\r\n$1\r\n3\r\n$4\r\n2324\r\n"),
		},
		{
			name: "测试BulkString",
			data: BulkStrings("232"),
			res:  []byte("$3\r\n232\r\n"),
		},
		{
			name: "測試BulkErr",
			data: MultiErr(errors.New("ERR")),
			res:  []byte("!3\r\nERR\r\n"),
		},
		{
			name: "測試Verbatim",
			data: Verbatim{
				Coding: "txt",
				Data:   []byte("Some string"),
			},
			res: []byte("=15\r\ntxt:Some string\r\n"),
		},
		{
			name: "测试Map",
			data: Maps{
				"first":  int64(1),
				"second": int64(2),
			},
			res: []byte("%2\r\n+first\r\n:1\r\n+second\r\n:2\r\n"),
		},
		{
			name: "测试Sets",
			data: gttype.NewHashSet[any]().Add(BulkStrings("foo")).Add(BulkStrings("bar")),
			res:  []byte("~2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"),
		},
		{
			name: "测试Push堆",
			//data: gttype.NewHeap[any](),
			res: []byte(">2\r\n$1\r\n1\r\n$1\r\n2\r\n"),
			before: func() any {
				res := gttype.NewHeap[any]()
				res.Insert(BulkStrings("1"))
				res.Insert(BulkStrings("2"))
				return res
			},
		},
		{
			name: "测试Nil",
			data: nil,
			res:  []byte("_\r\n"),
		},
		{
			name: "测试Error",
			data: errors.New("ERR"),
			res:  []byte("-ERR\r\n"),
		},
		{
			name: "測試簡單String",
			data: "OK",
			res:  []byte("+OK\r\n"),
		},
		{
			name: "测试Int",
			data: int64(2324),
			res:  []byte(":2324\r\n"),
		},
		{
			name: "测试負Int",
			data: int64(-2324),
			res:  []byte(":-2324\r\n"),
		},
		{
			name: "测试正Int",
			data: uint(2324),
			res:  []byte(":2324\r\n"),
		},
		{
			name: "测试Double",
			data: 1.23,
			res:  []byte(",1.23\r\n"),
		},
		{
			name: "测试Bool",
			data: true,
			res:  []byte("#t\r\n"),
		},
		{
			name: "测试大數",
			res:  []byte("(3492890328409238509324850943850943825024385\r\n"),
			before: func() any {
				b := new(big.Int)
				b.SetString("3492890328409238509324850943850943825024385", 10)
				return b
			},
		},
	}
	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			resp := NewRESP()
			if v.before != nil {
				v.data = v.before()
			}
			res := resp.BuildingRedisExecuteRESP(v.data).Build()
			//t.Logf("sum:%v,%T", string(res), res)
			//t.Logf("expected:%v, actual:%v", string(v.res), string(res))
			assert.Equal(t, v.res, res, "結果應該相同")
		})
	}
}

func TestCopy(t *testing.T) {
	var bytes []byte
	copy(bytes, []byte("456456456"))
	t.Logf("%s", bytes)
}

func TestType(t *testing.T) {
	// nil 就是 nil
	// types.Nil{} 是結構體
	//assert.Equal(t, types.Nil{}, nil)
}

func TestNewRESP(t *testing.T) {
	resp := NewRESP()
	row := []byte("*2\r\n$3\r\nget\r\n$1\r\na\r\n")
	after, res := resp.Parse(row)
	if len(after) > 0 {
		t.Logf("after:%s", after)
	} else {
		for _, arr := range res.(Array) {
			t.Logf("%v", arr)
		}
	}

}
