package posturepolicy

import (
	"fmt"
	"math/big"
	"strings"
)

type decimalNumber struct {
	negative bool
	digits   string
	exponent *big.Int
}

func compareNumberStrings(left, right string) (int, error) {
	l, err := parseDecimalNumber(left)
	if err != nil {
		return 0, err
	}
	r, err := parseDecimalNumber(right)
	if err != nil {
		return 0, err
	}
	if l.digits == "0" && r.digits == "0" {
		return 0, nil
	}
	if l.negative != r.negative {
		if l.negative {
			return -1, nil
		}
		return 1, nil
	}
	comparison := compareDecimalMagnitude(l, r)
	if l.negative {
		comparison = -comparison
	}
	return comparison, nil
}

func parseDecimalNumber(value string) (decimalNumber, error) {
	out := decimalNumber{exponent: new(big.Int)}
	if value == "" {
		return out, fmt.Errorf("empty JSON number")
	}
	if value[0] == '-' {
		out.negative = true
		value = value[1:]
	}

	mantissa := value
	exponentText := "0"
	if idx := strings.IndexAny(value, "eE"); idx >= 0 {
		mantissa = value[:idx]
		exponentText = value[idx+1:]
	}
	if _, ok := out.exponent.SetString(exponentText, 10); !ok {
		return decimalNumber{}, fmt.Errorf("invalid JSON number exponent %q", exponentText)
	}

	integerPart := mantissa
	fractionPart := ""
	if idx := strings.IndexByte(mantissa, '.'); idx >= 0 {
		integerPart = mantissa[:idx]
		fractionPart = mantissa[idx+1:]
	}
	digits := integerPart + fractionPart
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		out.negative = false
		out.digits = "0"
		out.exponent.SetInt64(0)
		return out, nil
	}
	out.exponent.Sub(out.exponent, big.NewInt(int64(len(fractionPart))))
	for len(digits) > 1 && digits[len(digits)-1] == '0' {
		digits = digits[:len(digits)-1]
		out.exponent.Add(out.exponent, big.NewInt(1))
	}
	out.digits = digits
	return out, nil
}

func compareDecimalMagnitude(left, right decimalNumber) int {
	leftMagnitude := new(big.Int).Set(left.exponent)
	leftMagnitude.Add(leftMagnitude, big.NewInt(int64(len(left.digits))))
	rightMagnitude := new(big.Int).Set(right.exponent)
	rightMagnitude.Add(rightMagnitude, big.NewInt(int64(len(right.digits))))
	if comparison := leftMagnitude.Cmp(rightMagnitude); comparison != 0 {
		return comparison
	}

	width := len(left.digits)
	if len(right.digits) > width {
		width = len(right.digits)
	}
	leftDigits := left.digits + strings.Repeat("0", width-len(left.digits))
	rightDigits := right.digits + strings.Repeat("0", width-len(right.digits))
	return strings.Compare(leftDigits, rightDigits)
}
