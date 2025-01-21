package sinks

import (
	"reflect"
	"testing"
)

func TestAzureStorageTable_flatten(t *testing.T) {
	type args struct {
		input map[string]interface{}
	}
	tests := []struct {
		name    string
		a       *AzureStorageTable
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "Must be able to flatten a map",
			a:    &AzureStorageTable{},
			args: args{
				input: map[string]interface{}{
					"key1":              "value1",
					"key2/key3":         "value2",
					"1startsWithNumber": "value3",
					"keyOutter": map[string]interface{}{
						"keyInner1": "valueInner1",
						"keyInner2": "valueInner2",
					},
					"arrayOfObjects": []interface{}{
						map[string]interface{}{
							"object0Key1": "object0Value1",
							"object0Key2": "object0Value2",
						},
						map[string]interface{}{
							"object1Key1": "object1Value1",
							"object1Key2": "object1Value2",
						},
					},
				},
			},
			want: map[string]interface{}{
				"key1":                         "value1",
				"key2_key3":                    "value2",
				"_1startsWithNumber":           "value3",
				"keyOutter_keyInner1":          "valueInner1",
				"keyOutter_keyInner2":          "valueInner2",
				"arrayOfObjects_0_object0Key1": "object0Value1",
				"arrayOfObjects_0_object0Key2": "object0Value2",
				"arrayOfObjects_1_object1Key1": "object1Value1",
				"arrayOfObjects_1_object1Key2": "object1Value2",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.flatten(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("AzureStorageTable.flatten() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AzureStorageTable.flatten() = %v, want %v", got, tt.want)
			}
		})
	}
}
