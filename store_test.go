package main

import (
	"bytes"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "mombestpic"
	pathName := CASPathTransformFunc(key)
	expectedOriginalKey := "ab805f112a89c1f1470d75bc4c065219fbe4c5de"
	expectedPathName := "ab805/f112a/89c1f/1470d/75bc4/c0652/19fbe/4c5de"

	if pathName.original != expectedOriginalKey {
		t.Errorf("have %s want %s", pathName.original, expectedOriginalKey)
	}
	if pathName.Pathname != expectedPathName {
		t.Errorf("have %s want %s", pathName.Pathname, expectedPathName)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	key := "mySpecialPicture"
	data := []byte("some random bytes")

	if err := s.Write(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)
	if string(b) != string(data) {
		t.Errorf("want %s have %s", data, b)
	}
}





