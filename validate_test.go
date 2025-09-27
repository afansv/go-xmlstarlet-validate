package xmlstarlet_validate

import (
	"reflect"
	"strings"
	"testing"
)

func Test_parseValidateOutputLines(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name         string
		args         args
		wantProblems []ResultProblem
		wantValid    bool
	}{
		{
			name: "catalog1",
			args: args{
				`
data/xml/catalog.xml:11.27: Element 'countries': This element is not expected.
data/xml/catalog.xml - invalid

`,
			},
			wantProblems: []ResultProblem{
				{
					Filename: "data/xml/catalog.xml",
					Line:     11,
					Col:      27,
					Issue:    "Element 'countries': This element is not expected.",
				},
			},
			wantValid: false,
		},
		{
			name: "catalog1_win",
			args: args{
				`
C:\data\xml\catalog.xml:11.27: Element 'countries': This element is not expected.
C:\data\xml\catalog.xml - invalid

`,
			},
			wantProblems: []ResultProblem{
				{
					Filename: "C:\\data\\xml\\catalog.xml",
					Line:     11,
					Col:      27,
					Issue:    "Element 'countries': This element is not expected.",
				},
			},
			wantValid: false,
		},
		{
			name: "badxsd1",
			args: args{
				`
failed to load external entity "file:///C:/Users/afansv/AppData/Local/Temp/warehouses.xsd"
file:///C:/Users/afansv/AppData/Local/Temp/go-xmlstarlet-validate-schema-sch-2042183185.xml:4.0: Element '{http://www.w3.org/2001/XMLSchema}include': Failed to load the document 'file:///C:/Users/afansv/AppData/Local/Temp/warehouses.xsd' for inclusion.

`,
			},
			wantProblems: []ResultProblem{
				{
					Filename: "C:/Users/afansv/AppData/Local/Temp/go-xmlstarlet-validate-schema-sch-2042183185.xml",
					Line:     4,
					Col:      0,
					Issue:    "Element '{http://www.w3.org/2001/XMLSchema}include': Failed to load the document 'file:///C:/Users/afansv/AppData/Local/Temp/warehouses.xsd' for inclusion.",
				},
			},
			wantValid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProblems, gotValid := parseValidateOutputLines(strings.Split(tt.args.output, "\n"))
			if !reflect.DeepEqual(gotProblems, tt.wantProblems) {
				t.Errorf("parseValidateOutputLines() gotProblems = %v, want %v", gotProblems, tt.wantProblems)
			}
			if gotValid != tt.wantValid {
				t.Errorf("parseValidateOutputLines() gotValid = %v, want %v", gotValid, tt.wantValid)
			}
		})
	}
}
