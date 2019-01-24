package structs

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/google/gofuzz"
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/reflectwalk"
	"github.com/pascaldekloe/goe/verify"
)

func TestCheckDefinition_Defaults(t *testing.T) {
	t.Parallel()
	def := CheckDefinition{}
	check := def.HealthCheck("node1")

	// Health checks default to critical state
	if check.Status != api.HealthCritical {
		t.Fatalf("bad: %v", check.Status)
	}
}

type walker struct {
	fields map[string]reflect.Value
}

func (w *walker) Struct(reflect.Value) error {
	return nil
}

func (w *walker) StructField(f reflect.StructField, v reflect.Value) error {
	w.fields[f.Name] = v
	return nil
}

func mapFields(obj interface{}) map[string]reflect.Value {
	w := &walker{make(map[string]reflect.Value)}
	if err := reflectwalk.Walk(obj, w); err != nil {
		panic(err)
	}
	return w.fields
}

func TestCheckDefinition_CheckType(t *testing.T) {
	t.Parallel()

	// Fuzz a definition to fill all its fields with data.
	var def CheckDefinition
	fuzz.New().Fuzz(&def)
	orig := mapFields(def)

	// Remap the ID field which changes name, and redact fields we don't
	// expect in the copy.
	orig["CheckID"] = orig["ID"]
	delete(orig, "ID")
	delete(orig, "ServiceID")
	delete(orig, "Token")

	// Now convert to a check type and ensure that all fields left match.
	chk := def.CheckType()
	copy := mapFields(chk)
	for f, vo := range orig {
		vc, ok := copy[f]
		if !ok {
			t.Fatalf("struct is missing field %q", f)
		}

		if !reflect.DeepEqual(vo.Interface(), vc.Interface()) {
			t.Fatalf("copy skipped field %q", f)
		}
	}
}

func TestCheckDefinitionToCheckType(t *testing.T) {
	t.Parallel()
	got := &CheckDefinition{
		ID:     "id",
		Name:   "name",
		Status: "green",
		Notes:  "notes",

		ServiceID:                      "svcid",
		Token:                          "tok",
		ScriptArgs:                     []string{"/bin/foo"},
		HTTP:                           "someurl",
		TCP:                            "host:port",
		Interval:                       1 * time.Second,
		DockerContainerID:              "abc123",
		Shell:                          "/bin/ksh",
		TLSSkipVerify:                  true,
		Timeout:                        2 * time.Second,
		TTL:                            3 * time.Second,
		DeregisterCriticalServiceAfter: 4 * time.Second,
	}
	want := &CheckType{
		CheckID: "id",
		Name:    "name",
		Status:  "green",
		Notes:   "notes",

		ScriptArgs:                     []string{"/bin/foo"},
		HTTP:                           "someurl",
		TCP:                            "host:port",
		Interval:                       1 * time.Second,
		DockerContainerID:              "abc123",
		Shell:                          "/bin/ksh",
		TLSSkipVerify:                  true,
		Timeout:                        2 * time.Second,
		TTL:                            3 * time.Second,
		DeregisterCriticalServiceAfter: 4 * time.Second,
	}
	verify.Values(t, "", got.CheckType(), want)
}

func TestCheckDefinition_HealthCheck(t *testing.T) {
	c := &CheckDefinition{
		ID:                             "check ID",
		Name:                           "check name",
		Notes:                          "check notes",
		ServiceID:                      "check serviceid",
		Interval:                       time.Minute * 2,
		Timeout:                        time.Minute * 3,
		DeregisterCriticalServiceAfter: time.Minute * 4,
		Status:                         api.HealthWarning,
	}

	node := "node1"
	h := c.HealthCheck(node)

	require.Equal(t, c.Name, h.Name)
	require.Equal(t, c.ID, h.CheckID)
	require.Equal(t, node, h.Node)
	require.Equal(t, c.Status, h.Status)
	require.Equal(t, c.Notes, h.Notes)
	require.Equal(t, c.ServiceID, h.ServiceID)

	require.Equal(t, c.HTTP, h.Definition.HTTP)
	require.Equal(t, c.TLSSkipVerify, h.Definition.TLSSkipVerify)
	require.Equal(t, c.Header, h.Definition.Header)
	require.Equal(t, c.Method, h.Definition.Method)
	require.Equal(t, c.TCP, h.Definition.TCP)
	require.Equal(t, c.Interval, h.Definition.Interval)
	require.Equal(t, c.Timeout, h.Definition.Timeout)
	require.Equal(t, c.DeregisterCriticalServiceAfter, h.Definition.DeregisterCriticalServiceAfter)
}
