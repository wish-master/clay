package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	stringspb "github.com/utrack/clay/integration/response_vs_response_body/pb"
	stringssrv "github.com/utrack/clay/integration/response_vs_response_body/strings"
	"github.com/utrack/clay/v2/transport/httpruntime"
)

func TestEcho(t *testing.T) {

	ts := testServer()
	defer ts.Close()

	tt := []struct {
		name string
		req  stringspb.Types
	}{
		{
			name: "double",
			req:  stringspb.Types{D: 1.0},
		},
		{
			name: "float",
			req:  stringspb.Types{F: 1.0},
		},
		{
			name: "int32",
			req:  stringspb.Types{I32: 1},
		},
		{
			name: "int64",
			req:  stringspb.Types{I64: 1},
		},
		{
			name: "uint32",
			req:  stringspb.Types{Ui32: 1},
		},
		{
			name: "uint64",
			req:  stringspb.Types{Ui64: 1},
		},
		{
			name: "sint32",
			req:  stringspb.Types{Si32: 1},
		},
		{
			name: "sint64",
			req:  stringspb.Types{Si64: 1},
		},
		{
			name: "fixed32",
			req:  stringspb.Types{Fixed32: 1},
		},
		{
			name: "fixed64",
			req:  stringspb.Types{Fixed64: 1},
		},
		{
			name: "sfixed32",
			req:  stringspb.Types{Sfixed32: 1},
		},
		{
			name: "sfixed64",
			req:  stringspb.Types{Sfixed64: 1},
		},
		{
			name: "bool",
			req:  stringspb.Types{B: true},
		},
		{
			name: "string",
			req:  stringspb.Types{S: "foo"},
		},
		{
			name: "bytes",
			req:  stringspb.Types{Bytes: []byte("bar")},
		},
		{
			name: "enum",
			req:  stringspb.Types{E: stringspb.Enum_FOO},
		},
		{
			name: "time",
			req:  stringspb.Types{Time: types.TimestampNow()},
		},
		{
			name: "duration",
			req:  stringspb.Types{Duration: types.DurationProto(3 * time.Second)},
		},
		// {
		// 	name: "stdtime",
		// 	req:  strings_pb.Types{Stdtime: time.Now().UTC()},
		// },
		// {
		// 	name: "stdduration",
		// 	req:  strings_pb.Types{Stdduration: 3 * time.Second},
		// },
	}

	httpClient := ts.Client()
	client := stringspb.NewStringsHTTPClient(httpClient, ts.URL)

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%s and [%s]", tc.name, tc.name), func(t *testing.T) {
			ts.Lock()
			resp, err := client.Echo(context.Background(), &tc.req)
			echoBody, _ := ioutil.ReadAll(ts.RW)
			ts.Unlock()
			if err != nil {
				t.Fatalf("expected err <nil>, got: %q", err)
			}
			if resp == nil {
				t.Fatalf("expected non-nil response, got nil")
			}
			if !reflect.DeepEqual(*resp, tc.req) {
				t.Fatalf("expected %#v\n"+
					"got: %#v", tc.req, *resp)
			}

			ts.Lock()
			resp2, err := client.Echo2(context.Background(), &stringspb.ListTypes{List: []*stringspb.Types{&tc.req}})
			echoBody2, _ := ioutil.ReadAll(ts.RW)
			ts.Unlock()
			if err != nil {
				t.Fatalf("expected err <nil>, got: %q", err)
			}
			if resp2 == nil {
				t.Fatalf("expected non-nil response, got nil")
			}

			if !reflect.DeepEqual(*resp2.List[0], tc.req) {
				t.Fatalf("expected %#v\n"+
					"got: %#v", tc.req, *resp2.List[0])
			}

			if "["+string(echoBody)+"]" != string(echoBody2) {
				t.Fatalf("expected <response from echo2> = `[` + <response from echo> +`]`, got\n"+
					"<response from echo>  = %s\n"+
					"<response from echo2> = %s", echoBody, echoBody2)
			}
		})
	}

	from := []*CustomType{{
		A: 3,
		B: "15",
		C: 10.0,
		D: []int{20},
		E: map[string]interface{}{
			"F": CustomType{},
		},
	}}
	to := make([]*CustomType, 0)

	var buf bytes.Buffer
	marshaller := httpruntime.DefaultMarshaler(nil)
	err := marshaller.Marshal(&buf, from)
	if err != nil {
		t.Fatalf("expected err <nil>, got: %q", err)
	}
	err = marshaller.Unmarshal(&buf, &to)
	if err != nil {
		t.Fatalf("expected err <nil>, got: %q", err)
	}
	compareCustomType(t, from, to)

	a := from[0]
	err = marshaller.Marshal(&buf, a)
	if err != nil {
		t.Fatalf("expected err <nil>, got: %q", err)
	}
	var b CustomType
	err = marshaller.Unmarshal(&buf, &b)
	if err != nil {
		t.Fatalf("expected err <nil>, got: %q", err)
	}
	compareCustomType(t, []*CustomType{a}, []*CustomType{&b})
}

type CustomType struct {
	A int64                  `json:"A"`
	B string                 `json:"B"`
	C float64                `json:"C"`
	D []int                  `json:"D"`
	E map[string]interface{} `json:"E"`
}

func compareCustomType(t *testing.T, from, to []*CustomType) {
	if len(from) != len(to) {
		t.Fatalf("expected len %#v\n"+
			"got: %#v", len(from), len(to))
	}
	for i := range from {
		if from[i].A != to[i].A {
			t.Fatalf("expected A %#v\n"+
				"got: %#v", from[i].A, to[i].A)
		}
		if from[i].B != to[i].B {
			t.Fatalf("expected B %#v\n"+
				"got: %#v", from[i].B, to[i].B)
		}
		if from[i].C != to[i].C {
			t.Fatalf("expected C %#v\n"+
				"got: %#v", from[i].C, to[i].C)
		}
		if len(from[i].D) != len(from[i].D) {
			t.Fatalf("expected len D %#v\n"+
				"got: %#v", len(from[i].D), len(to[i].D))
		}
		for id := range from[i].D {
			if from[i].D[id] != to[i].D[id] {
				t.Fatalf("expected D %#v\n"+
					"got: %#v", from[i].D[id], to[i].D[id])
			}
		}
		if len(from[i].E) != len(from[i].E) {
			t.Fatalf("expected len E %#v\n"+
				"got: %#v", len(from[i].E), len(to[i].E))
		}
		for k := range from[i].E {
			_, ok := to[i].E[k]
			if !ok {
				t.Fatalf("expected key E %#v\n", k)
			}
		}
	}
}

func testServer() *Server {
	mux := http.NewServeMux()
	desc := stringssrv.NewStrings().GetDescription()
	desc.RegisterHTTP(mux)
	mux.Handle("/swagger.json", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(desc.SwaggerDef())
	}))
	ts := &Server{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ts.RW = NewRW(w)
		mux.ServeHTTP(ts.RW, req)
	}))
	return ts
}

type Server struct {
	sync.Mutex
	*httptest.Server
	RW *RW
}

func NewRW(w http.ResponseWriter) *RW {
	return &RW{
		w,
		bytes.NewBuffer([]byte{}),
	}
}

type RW struct {
	http.ResponseWriter
	*bytes.Buffer
}

func (w RW) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w RW) Write(b []byte) (int, error) {
	w.Buffer.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w RW) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}
