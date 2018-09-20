package nbfmt

// func stringCompare(s1, s2 string, op object) (bool, error) {
// 	switch op.idents[0].typ {
// 	case eqIdent:
// 		return s1 == s2, nil
// 	case neqIdent:
// 		return s1 != s2, nil
// 	case ltIdent:
// 		return s1 < s2, nil
// 	case lteIdent:
// 		return s1 <= s2, nil
// 	case gtIdent:
// 		return s1 > s2, nil
// 	case gteIdent:
// 		return s1 >= s2, nil
// 	default:
// 		return false, fmt.Errorf("invalid compare operator: %v", op)
// 	}
// }

// func intCompare(i1, i2 int64, op object) (bool, error) {
// 	switch op.idents[0].typ {
// 	case eqIdent:
// 		return i1 == i2, nil
// 	case neqIdent:
// 		return i1 != i2, nil
// 	case ltIdent:
// 		return i1 < i2, nil
// 	case lteIdent:
// 		return i1 <= i2, nil
// 	case gtIdent:
// 		return i1 > i2, nil
// 	case gteIdent:
// 		return i1 >= i2, nil
// 	default:
// 		return false, fmt.Errorf("invalid compare operator: %v", op)
// 	}
// }

// func floatCompare(f1, f2 float64, op object) (bool, error) {
// 	switch op.idents[0].typ {
// 	case eqIdent:
// 		return f1 == f2, nil
// 	case neqIdent:
// 		return f1 != f2, nil
// 	case ltIdent:
// 		return f1 < f2, nil
// 	case lteIdent:
// 		return f1 <= f2, nil
// 	case gtIdent:
// 		return f1 > f2, nil
// 	case gteIdent:
// 		return f1 >= f2, nil
// 	default:
// 		return false, fmt.Errorf("invalid compare operator: %v", op)
// 	}
// }

// func boolCompare(b1, b2 bool, op object) (bool, error) {
// 	switch op.idents[0].typ {
// 	case eqIdent:
// 		return b1 == b2, nil
// 	case neqIdent:
// 		return b1 != b2, nil
// 	default:
// 		return false, fmt.Errorf("cannot compare bool values with %v operator", op)
// 	}
// }
