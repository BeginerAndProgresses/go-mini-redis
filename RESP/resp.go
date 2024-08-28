package resp

import (
	"bytes"
	"errors"
	"fmt"
	gttype "github.com/BeginerAndProgresses/generalized-tools/type"
	"log/slog"
	"math/big"
	"reflect"
	"strconv"
)

// 参考 https://redis.io/topics/protocol

const (
	// 简单类型
	//   字符串
	typeStrSign = '+'
	//   整数
	typeIntSign = ':'
	//	 错误
	typeErrSign = '-'
	//	空
	typeNullSign = '_'
	//	 布尔
	typeBoolSign = '#'
	//	 浮点数
	typeDoublesSign = ','
	//	 大数
	typeBigNumbersSign = '('
)

const (
	// Aggregate
	//   数组
	typeArrSign = '*'
	//	 批量字符串
	typeBulkStringsSign = '$'
	//	 批量错误
	typeMultiErrSign = '!'
	//	逐字字符串
	typeVervatimSign = '='
	//	 Map
	typeMapsSign = '%'
	//	 Set
	typeSetsSign = '~'
	//	 pushes
	typePushesSign = '>'
)

type (
	//Array 对应数组类型
	Array []any

	//BulkStrings 对应批量字符串类型
	BulkStrings string

	//MultiErr 对应批量错误类型
	MultiErr struct {
		err string
	}

	//Verbatim 对应逐字字符串类型
	Verbatim struct {
		Coding string
		Data   []byte
	}

	//Maps 对应Map类型
	Maps map[any]any

	//Sets 对应Set类型
	Sets gttype.HashSet[any]

	//Pushes 對應推送类型
	Pushes gttype.MinHeap[any]
)

func (e *MultiErr) Error() string {
	return e.err
}

type RESP interface {
	Parse(data []byte) (parseAfter []byte, res any)
	BuildingRedisExecuteRESP(data any) *respSvc
	ValidRESP(resp []byte) (bool, error)
}

type respSvc struct {
	cerRESP []byte
}

// Build 構建RESP
func (r *respSvc) Build() []byte {
	var rb = make([]byte, len(r.cerRESP))
	copy(rb, r.cerRESP)
	r.clearCurRESP()
	return rb
}

// Parse 根据Row解析RESP
func (r *respSvc) Parse(data []byte) (parseAfter []byte, res any) {
	if len(data) < 2 {
		return []byte(""), nil
	}
	opttyp := data[0]

	var optdata []byte
	// 判断是否有前缀
	if !bytes.ContainsRune([]byte{
		typeArrSign,
		typeBulkStringsSign,
		typeMultiErrSign,
		typeVervatimSign,
		typeMapsSign,
		typeSetsSign,
		typePushesSign,
		typeStrSign,
		typeIntSign,
		typeErrSign,
		typeNullSign,
		typeBoolSign,
		typeDoublesSign,
		typeBigNumbersSign,
	}, rune(opttyp)) {
		optdata = make([]byte, len(data))
		copy(optdata, data)
	} else {
		optdata = make([]byte, len(data)-1)
		copy(optdata, data[1:])
	}
	before, after, ok := bytes.Cut(optdata, []byte("\r\n"))
	switch opttyp {
	// 聚合类型
	case typeArrSign:
		// 聚合类型
		nlen, err := strconv.ParseInt(string(before), 10, 64)
		if err != nil {
			slog.Error("解析批量错误类型失败", slog.Any("err", err))
			return nil, err
		}
		if nlen < 0 {
			return after, Array{}
		}
		var arr Array = make([]interface{}, nlen)
		for i := 0; i < len(arr); i++ {
			after, arr[i] = r.Parse(after)
		}
		return after, arr
	case typeBulkStringsSign:
		var bs BulkStrings
		after, res = r.Parse(after)
		// 如果是字符串类型,直接轉為字節數組
		if resStr, ok := res.(string); ok {
			bs = BulkStrings(resStr)
		} else {
			bs = ""
		}
		return after, bs
	case typeMultiErrSign:
		var me MultiErr
		after, res = r.Parse(after)
		if resStr, ok := res.(string); ok {
			me = MultiErr{
				err: resStr,
			}
		} else {
			me = MultiErr{}
		}
		return after, me
	case typeVervatimSign:
		after, res = r.Parse(after)
		var vs Verbatim
		if resStr, ok := res.(string); ok {
			// 將':'跳過
			vs = Verbatim{
				Coding: resStr[:3],
				Data:   []byte(resStr[4:]),
			}
		} else {
			vs = Verbatim{}
		}
		return after, vs
	case typeMapsSign:
		nlen, err := strconv.ParseInt(string(before), 10, 64)
		if err != nil {
			slog.Error("解析Map类型失败", slog.Any("err", err))
			return after, nil
		}
		var ms Maps = make(map[any]any, nlen)
		for i := 0; i < int(nlen); i++ {
			after, res = r.Parse(after)
			after, ms[res] = r.Parse(after)
		}
		return after, ms
	case typeSetsSign:
		nlen, err := strconv.ParseInt(string(before), 10, 64)
		if err != nil {
			slog.Error("解析Set类型失败", slog.Any("err", err))
			return after, nil
		}
		var ss Sets = gttype.NewHashSet[any]()
		for i := 0; i < int(nlen); i++ {
			after, res = r.Parse(after)
			ss.Add(res)
		}
		return after, ss
	case typePushesSign:
		nlen, err := strconv.ParseInt(string(before), 10, 64)
		if err != nil {
			slog.Error("解析推送类型失败", slog.Any("err", err))
			return after, nil
		}
		var ps Pushes = gttype.NewHeap[any]()
		for i := 0; i < int(nlen); i++ {
			after, res = r.Parse(after)
			ps.Insert(res)
		}
		return after, ps
	// 简单类型
	case typeStrSign:
		if ok {
			return after, string(before)
		}
	case typeIntSign:
		if ok {
			i, err := strconv.ParseInt(string(before), 10, 64)
			if err != nil {
				slog.Error("解析错误类型失败", slog.Any("err", err))
				return after, 0
			}
			return after, i
		}
	case typeErrSign:
		if ok {
			return after, errors.New(string(before))
		}
	case typeNullSign:
		if ok {
			return after, nil
		}
	case typeBoolSign:
		if ok {
			return after, bytes.Equal(before, []byte("t"))
		}
	case typeDoublesSign:
		if ok {
			float, err := strconv.ParseFloat(string(before), 64)
			if err != nil {
				slog.Error("解析浮点数类型失败", slog.Any("err", err))
				return after, float
			}
			return after, float
		}
	case typeBigNumbersSign:
		if ok {
			bi := new(big.Int)
			_, b := bi.SetString(string(before), 10)
			if !b {
				return after, nil
			}
			// 避免以後RESP大數定義改變，這裡使用String類型
			return after, bi
		}
	default:
		return after, string(before)
	}
	return []byte(""), nil
}

// BuildingRedisExecuteRESP 构建可以供Redis执行RESP
func (r *respSvc) BuildingRedisExecuteRESP(data any) *respSvc {
	var buffer bytes.Buffer
	switch data.(type) {
	case Array:
		arr := data.(Array)
		buffer.WriteByte(typeArrSign)
		if len(arr) == 0 {
			buffer.WriteString("-1")
		} else {
			buffer.WriteString(strconv.Itoa(len(arr)))
		}
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
		for _, v := range arr {
			r.BuildingRedisExecuteRESP(v)
		}
	case BulkStrings:
		bs := data.(BulkStrings)
		buffer.WriteByte(typeBulkStringsSign)
		buffer.WriteString(strconv.Itoa(len(bs)))
		buffer.Write([]byte("\r\n"))
		buffer.Write([]byte(bs))
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case MultiErr:
		me := data.(MultiErr)
		buffer.WriteByte(typeMultiErrSign)
		buffer.WriteString(strconv.Itoa(len(me.Error())))
		buffer.Write([]byte("\r\n"))
		buffer.Write([]byte(me.Error()))
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case Verbatim:
		vs := data.(Verbatim)
		buffer.WriteByte(typeVervatimSign)
		buffer.WriteString(strconv.Itoa(len(vs.Data) + 4))
		buffer.Write([]byte("\r\n"))
		buffer.WriteString(vs.Coding)
		buffer.WriteByte(':')
		buffer.Write(vs.Data)
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case Maps:
		mp := data.(Maps)
		buffer.WriteByte(typeMapsSign)
		buffer.WriteString(strconv.Itoa(len(mp)))
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
		for k, v := range mp {
			r.BuildingRedisExecuteRESP(k)
			r.BuildingRedisExecuteRESP(v)
		}
	case Sets:
		ss := data.(Sets)
		buffer.WriteByte(typeSetsSign)
		buffer.WriteString(strconv.Itoa(ss.Size()))
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
		anies := ss.GetData()
		for i := range anies {
			r.BuildingRedisExecuteRESP(anies[i])
		}
	case Pushes:
		ps := data.(Pushes)
		buffer.WriteByte(typePushesSign)
		buffer.WriteString(strconv.Itoa(ps.Size()))
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
		ps.ForEach(func(a any) {
			r.BuildingRedisExecuteRESP(a)
		})
	case string:
		buffer.WriteByte(typeStrSign)
		buffer.WriteString(data.(string))
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case int64, int32, int16, int8, int, uint:
		buffer.WriteByte(typeIntSign)
		v := reflect.ValueOf(data)
		switch expr := v.Kind(); expr {
		case reflect.Uint:
			buffer.WriteString(strconv.FormatUint(v.Uint(), 10))
		default:
			buffer.WriteString(strconv.FormatInt(v.Int(), 10))
		}
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case nil:
		buffer.WriteByte(typeNullSign)
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case error:
		buffer.WriteByte(typeErrSign)
		buffer.WriteString(data.(error).Error())
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case bool:
		buffer.WriteByte(typeBoolSign)
		if data.(bool) {
			buffer.Write([]byte("t"))
		} else {
			buffer.Write([]byte("f"))
		}
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case float64, float32:
		buffer.WriteByte(typeDoublesSign)
		v := reflect.ValueOf(data)
		buffer.WriteString(strconv.FormatFloat(v.Float(), 'f', -1, 64))
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	case *big.Int:
		buffer.WriteByte(typeBigNumbersSign)
		buffer.WriteString(data.(*big.Int).String())
		buffer.Write([]byte("\r\n"))
		r.cerRESP = append(r.cerRESP, buffer.Bytes()...)
	default:
		slog.Info("不支持的类型", slog.Any("data", data))
	}
	return r
}
func (r *respSvc) clearCurRESP() {
	r.cerRESP = make([]byte, 0)
}

// ValidRESP 验证RESP格式
func (r *respSvc) ValidRESP(resp []byte) (bool, error) {
	if len(resp) < 3 {
		return false, errors.New("row长度小于3")
	}
	if !bytes.HasSuffix(resp, []byte("\r\n")) {
		return false, errors.New("结尾不是\\r\\n")
	}
	before, _, _ := bytes.Cut(resp, []byte("\r\n"))
	before = before[1:]
	switch resp[0] {
	case typeArrSign:
		//if string(before) == "-1" && !ok {
		//	return true, nil
		//}
		//if ok {
		//	nlen, err := strconv.Atoi(string(before))
		//	if err != nil {
		//		return false, err
		//	}
		//	if nlen < 0 {
		//		return false, errors.New("数组长度小于0")
		//	} else if nlen == 0 {
		//
		//	} else {
		//		if !ok {
		//			return false, errors.New("数组长度不为-1，但数据为空")
		//		}
		//		for i := 0; i < nlen; i++ {
		//			, err := r.ValidRESP(after)
		//
		//			if ! {
		//				return false, errors.New("数组数据格式错误")
		//			}
		//		}
		//	}
		//}
	case typeBulkStringsSign:
	case typeMultiErrSign:
	case typeSetsSign:
	case typeVervatimSign:
	case typeMapsSign:
	case typePushesSign:
	case typeStrSign:
		return true, nil
	case typeIntSign:
		_, err := strconv.ParseInt(string(before), 10, 64)
		if err != nil {
			return false, err
		}
		return true, nil
	case typeErrSign:
		return true, nil
	case typeNullSign:
		if len(before) == 0 {
			return true, nil
		}
		return false, errors.New("不为-1\\r\\n")
	case typeBoolSign:
		if len(before) != 1 {
			return false, errors.New("bool标志不为t或f")
		}
		switch before[0] {
		case 't', 'f':
			return true, nil
		default:
			return false, errors.New("不为t或f")
		}
	case typeDoublesSign:
		_, err := strconv.ParseFloat(string(before), 64)
		if err != nil {
			return false, err
		}
		return true, nil
	case typeBigNumbersSign:
		_, b := big.NewInt(0).SetString(string(before), 10)
		if !b {
			return false, errors.New(fmt.Sprintf("%v 不是数字", string(before)))
		}
		return true, nil
	default:
		return false, errors.New(fmt.Sprintf("不支持的类型:%b", resp[0]))
	}
	return false, nil
}

func NewRESP() RESP {
	return &respSvc{
		cerRESP: make([]byte, 0),
	}
}

// splitBytes 根据分隔字节切片将字节切片分割成多个字节切片
func splitBytes(data []byte, sep []byte) [][]byte {
	var parts [][]byte
	for len(data) > 0 {
		index := bytes.Index(data, sep)
		if index == -1 {
			// 如果找不到分隔符，则直接添加剩余部分
			parts = append(parts, data)
			break
		}

		// 添加分隔符之前的部分
		parts = append(parts, data[:index])

		// 移除已经处理的部分
		data = data[index+len(sep):]
	}

	return parts
}
