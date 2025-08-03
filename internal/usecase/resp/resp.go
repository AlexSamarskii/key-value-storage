package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

const (
	STRING  = '+'
	ERROR   = '-'
	INTEGER = ':'
	BULK    = '$'
	ARRAY   = '*'
)

type Value struct {
	Typ   string
	Str   string
	Num   int
	Bulk  string
	Array []Value
}

type Reader struct {
	reader *bufio.Reader
}

func NewReader(rd io.Reader) *Reader {
	return &Reader{reader: bufio.NewReader(rd)}
}

type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func (r *Reader) Read() (Value, error) {
	_type, err := r.reader.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch _type {
	case '+':
		return r.readString()
	case '-':
		return r.readError()
	case ':':
		return r.readInteger()
	case '$':
		return r.readBulkString()
	case '*':
		return r.readArray()
	default:
		fmt.Printf("Unknown type: %v", string(_type))
		return Value{}, nil
	}
}

func (r *Reader) readError() (Value, error) {
	str, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	return Value{Typ: "error", Str: str}, nil
}

func (r *Reader) readInteger() (Value, error) {
	str, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	num, err := strconv.Atoi(str) // Рекурсивный вызов — поддержка вложенных структур
	if err != nil {
		return Value{}, fmt.Errorf("cannot parse integer: %s", str)
	}
	return Value{Typ: "integer", Num: num, Str: str}, nil
}

func (r *Reader) readArray() (Value, error) {
	lenStr, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	arrayLen, err := strconv.Atoi(lenStr)
	if err != nil {
		return Value{}, fmt.Errorf("invalid array length: %s", lenStr)
	}

	if arrayLen == -1 {
		return Value{Typ: "null"}, nil
	}

	var elements []Value
	for i := 0; i < arrayLen; i++ {
		value, err := r.Read()
		if err != nil {
			return Value{}, err
		}
		elements = append(elements, value)
	}

	return Value{Typ: "array", Array: elements}, nil
}

func (r *Reader) readLine() (string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if len(line) < 2 || line[len(line)-2:] != "\r\n" {
		return "", fmt.Errorf("invalid line ending")
	}
	return line[:len(line)-2], nil // убираем \r\n
}

func (r *Reader) readString() (Value, error) {
	str, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	return Value{Typ: "string", Str: str}, nil
}

func (r *Reader) readBulkString() (Value, error) {
	lenStr, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	bulkLen, err := strconv.Atoi(lenStr)
	if err != nil {
		return Value{}, err
	}

	if bulkLen == -1 {
		return Value{Typ: "null"}, nil
	}

	bulk := make([]byte, bulkLen+2) // +2 для \r\n
	_, err = io.ReadFull(r.reader, bulk)
	if err != nil {
		return Value{}, err
	}

	if bulk[bulkLen] != '\r' || bulk[bulkLen+1] != '\n' {
		return Value{}, fmt.Errorf("expected \\r\\n")
	}

	return Value{Typ: "bulk", Bulk: string(bulk[:bulkLen])}, nil
}

func (v Value) marshalString() []byte {
	var bytes []byte
	bytes = append(bytes, STRING)
	bytes = append(bytes, v.Str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (v Value) marshalBulk() []byte {
	var bytes []byte
	bytes = append(bytes, BULK)
	bytes = append(bytes, strconv.Itoa(len(v.Bulk))...)
	bytes = append(bytes, '\r', '\n')
	bytes = append(bytes, v.Bulk...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (v Value) marshalArray() []byte {
	len := len(v.Array)
	var bytes []byte
	bytes = append(bytes, ARRAY)
	bytes = append(bytes, strconv.Itoa(len)...)
	bytes = append(bytes, '\r', '\n')

	for i := 0; i < len; i++ {
		bytes = append(bytes, v.Array[i].Marshal()...)
	}

	return bytes
}

func (v Value) marshalError() []byte {
	var bytes []byte
	bytes = append(bytes, ERROR)
	bytes = append(bytes, v.Str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (v Value) marshalNull() []byte {
	return []byte("$-1\r\n")
}

func (v Value) marshalInteger() []byte {
	var bytes []byte
	bytes = append(bytes, INTEGER)
	bytes = append(bytes, strconv.Itoa(v.Num)...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) Marshal() []byte {
	switch v.Typ {
	case "array":
		return v.marshalArray()
	case "bulk":
		return v.marshalBulk()
	case "string":
		return v.marshalString()
	case "null":
		return v.marshalNull()
	case "error":
		return v.marshalError()
	case "integer":
		return v.marshalInteger()
	default:
		return []byte{}
	}
}

func (w *Writer) Write(v Value) error {
	bytes := v.Marshal()

	_, err := w.writer.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}
