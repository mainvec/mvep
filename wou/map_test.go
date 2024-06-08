package wou_test

import (
	"reflect"
	"testing"

	"github.com/workoak/wop/wou"
)

func TestIterateByKey(t *testing.T) {
	type OData wou.OMap[string, string]
	testMap := OData{
		"a": "aVal",
		"x": "xVal",
		"c": "cVal",
		"b": "bVal",
	}
	want := []string{"aVal", "bVal", "cVal", "xVal"}
	var got []string
	for iter := wou.IteratorByKey(testMap); iter.HasNext(); {
		_, v := iter.Next()
		got = append(got, v)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v,wanted %v ", got, want)
	}

}

func TestIterateByValue(t *testing.T) {

	type ValStruct struct {
		Fnum  int
		FName string
	}
	type OData map[string]*ValStruct
	testMap := OData{
		"a": &ValStruct{Fnum: 4, FName: "aStruct"},
		"x": &ValStruct{Fnum: 2, FName: "xStruct"},
		"c": &ValStruct{Fnum: 3, FName: "cStruct"},
		"b": &ValStruct{Fnum: 1, FName: "bStruct"},
	}
	t.Run("iterate by fnum order", func(t *testing.T) {
		//order by Fnum
		want := []*ValStruct{testMap["b"], testMap["x"], testMap["c"], testMap["a"]}
		var got []*ValStruct
		lessFunc := func(i, j *ValStruct) bool {
			return i.Fnum < j.Fnum
		}
		for iter := wou.IterateByValue(testMap, lessFunc); iter.HasNext(); {
			_, v := iter.Next()
			got = append(got, v)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v,wanted %v ", got, want)
		}
	})

	t.Run("iterate by fname order", func(t *testing.T) {
		//order by Fnum
		want := []*ValStruct{testMap["a"], testMap["b"], testMap["c"], testMap["x"]}
		var got []*ValStruct
		lessNameFunc := func(i, j *ValStruct) bool {
			return i.FName < j.FName
		}
		for iter := wou.IterateByValue(testMap, lessNameFunc); iter.HasNext(); {
			_, v := iter.Next()
			got = append(got, v)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v,wanted %v ", got, want)
		}
	})

}
