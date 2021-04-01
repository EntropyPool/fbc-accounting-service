package utils

import "math/big"

// over int64 calculate add
func BigIntAdd(numstr string, num int64) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetInt64(num)
	m.Add(n, m)
	return m.String()
}

// over int64 calculate sub
func BigIntReduce(numstr string, num int64) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetInt64(-num)
	m.Add(n, m)
	return m.String()
}

func BigIntAddStr(numstr string, num string) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetString(num, 10)
	m.Add(n, m)
	return m.String()
}

// over int64 calculate sub
func BigIntReduceStr(numstr string, num string) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetString(num, 10)
	m.Sub(n, m)
	return m.String()
}

func BigIntMulStr(numstr string, num string) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetString(num, 10)
	m.Mul(n, m)
	return m.String()
}

// over int64 calculate div
func BigIntDivStr(numstr string, num string) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetString(num, 10)
	m.Div(n, m)
	return m.String()
}
