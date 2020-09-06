package rest

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"testing"
)

func TestReadValidConfigurationFile(t *testing.T) {
	var fn string
	f, err := ioutil.TempFile("", "rest-test")
	if err != nil {
		t.Errorf("cannot create test file: %s", err.Error())
	}

	rc := NewConfiguration()
	rc.ApplicationName = "test1"
	d, _ := yaml.Marshal(&rc)
	f.Write(d)

	fn = f.Name()
	f.Close()
	defer func() {
		if err := os.Remove(fn); err != nil {
			t.Errorf("cannot remove temporary file: %s", err.Error())
		}
	}()

	conf := NewConfiguration()
	ReadConfiguration(fn, &conf)
	if conf.ApplicationName != rc.ApplicationName {
		t.Errorf("Application name is not correct expected '%s', but get '%s'", rc.ApplicationName, conf.ApplicationName)
	}
}

func TestReadInvalidConfigurationFile(t *testing.T) {
	var fn string
	f, err := ioutil.TempFile("", "rest-test")
	if err != nil {
		t.Errorf("cannot create test file: %s", err.Error())
	}

	f.WriteString("{")

	fn = f.Name()
	f.Close()
	defer func() {
		if err := os.Remove(fn); err != nil {
			t.Errorf("cannot remove temporary file: %s", err.Error())
		}
	}()

	conf := NewConfiguration()
	if err := ReadConfiguration(fn, &conf); err == nil {
		t.Errorf("No error reported, even there should be one")
	}
}
