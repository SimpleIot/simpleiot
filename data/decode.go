package data

import (
	"reflect"
)

// Decode converts a Node to custom struct
func Decode(input NodeEdge, output interface{}) error {
	vOut := reflect.ValueOf(output).Elem()
	tOut := reflect.TypeOf(output).Elem()

	pointValues := make(map[string]reflect.Value)
	edgeValues := make(map[string]reflect.Value)

	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		pt := sf.Tag.Get("point")
		if pt != "" {
			pointValues[pt] = vOut.Field(i)
		} else {
			et := sf.Tag.Get("edgepoint")
			if et != "" {
				edgeValues[et] = vOut.Field(i)
			}
		}
	}

	setVal := func(p Point, v reflect.Value) {
		if p.Text != "" {
			v.SetString(p.Text)
		} else {
			switch v.Type().Kind() {
			case reflect.Int:
				v.SetInt(int64(p.Value))
			case reflect.Float64, reflect.Float32:
				v.SetFloat(p.Value)
			case reflect.Bool:
				v.SetBool(FloatToBool(p.Value))
			}
		}
	}

	for _, p := range input.Points {
		v, ok := pointValues[p.Type]
		if ok {
			setVal(p, v)
		}
	}

	for _, p := range input.EdgePoints {
		v, ok := edgeValues[p.Type]
		if ok {
			setVal(p, v)
		}
	}

	return nil
}