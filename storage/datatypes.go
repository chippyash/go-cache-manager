package storage

import (
	"strconv"
	"time"
)

const (
	TypeUnknown = iota
	TypeBoolean
	TypeInteger
	TypeInteger8
	TypeInteger16
	TypeInteger32
	TypeInteger64
	TypeUint
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64
	TypeFloat32
	TypeFloat64
	TypeString
	TypeDuration
	TypeTime
	TypeBytes
)

type DataTypes map[int]bool
var DefaultDataTypes = DataTypes{
	TypeUnknown:   true,
	TypeBoolean:   true,
	TypeInteger:   true,
	TypeInteger8:  true,
	TypeInteger16: true,
	TypeInteger32: true,
	TypeInteger64: true,
	TypeUint:      true,
	TypeUint8:     true,
	TypeUint16:    true,
	TypeUint32:    true,
	TypeUint64:    true,
	TypeFloat32:   true,
	TypeFloat64:   true,
	TypeString:    true,
	TypeDuration:  true,
	TypeTime:      true,
	TypeBytes:     true,
}

func GetType(v any) int {
	switch v.(type) {
	case string:
		return TypeString
	case int:
		return TypeInteger
	case int8:
		return TypeInteger8
	case int16:
		return TypeInteger16
	case int32:
		return TypeInteger32
	case int64:
		return TypeInteger64
	case uint:
		return TypeUint
	case uint8:
		return TypeUint8
	case uint16:
		return TypeUint16
	case uint32:
		return TypeUint32
	case uint64:
		return TypeUint64
	case float32:
		return TypeFloat32
	case float64:
		return TypeFloat64
	case time.Duration:
		return TypeDuration
	case time.Time:
		return TypeTime
	case bool:
		return TypeBoolean
	case []byte:
		return TypeBytes
	default:
		return TypeUnknown
	}
}

func GetTypedValue(t int, v string) (any, error) {
	switch t {
	case TypeString:
		return v, nil
	case TypeInteger:
		return strconv.Atoi(v)
	case TypeInteger8:
		i, err := strconv.ParseInt(v, 10, 8)
		return int8(i), err
	case TypeInteger16:
		i, err := strconv.ParseInt(v, 10, 16)
		return int16(i), err
	case TypeInteger32:
		i, err := strconv.ParseInt(v, 10, 32)
		return int32(i), err
	case TypeInteger64:
		return strconv.ParseInt(v, 10, 64)
	case TypeUint:
		i, err := strconv.ParseUint(v, 10, 64)
		return uint(i), err
	case TypeUint8:
		i, err := strconv.ParseUint(v, 10, 8)
		return uint8(i), err
	case TypeUint16:
		i, err := strconv.ParseUint(v, 10, 16)
		return uint16(i), err
	case TypeUint32:
		i, err := strconv.ParseUint(v, 10, 32)
		return uint32(i), err
	case TypeUint64:
		return strconv.ParseUint(v, 10, 64)
	case TypeFloat32:
		f, err := strconv.ParseFloat(v, 32)
		return float32(f), err
	case TypeFloat64:
		return strconv.ParseFloat(v, 64)
	case TypeDuration:
		return time.ParseDuration(v)
	case TypeTime:
		return time.Parse(time.RFC3339, v)
	case TypeBoolean:
		return strconv.ParseBool(v)
	case TypeBytes:
		return []byte(v), nil
	default:
		return v, nil
	}
}


