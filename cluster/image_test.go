// Copyright 2014 docker-cluster authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cluster

import (
	"bytes"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/tsuru/safe"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRemoveImage(t *testing.T) {
	var called bool
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No such image", http.StatusNotFound)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server2.Close()
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	name := "tsuru/python"
	err = cluster.RemoveImage(name)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Errorf("RemoveImage(%q): Did not call node HTTP server", name)
	}
}

func TestRemoveImageNotFound(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No such image", http.StatusNotFound)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No such image", http.StatusNotFound)
	}))
	defer server2.Close()
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	name := "tsuru/python"
	err = cluster.RemoveImage(name)
	if err != docker.ErrNoSuchImage {
		t.Errorf("RemoveImage(%q): wrong error. Want %#v. Got %#v.", name, docker.ErrNoSuchImage, err)
	}
}

func TestPullImage(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pulling from 1!"))
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pulling from 2!"))
	}))
	defer server2.Close()
	var buf safe.Buffer
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	opts := docker.PullImageOptions{Repository: "tsuru/python", OutputStream: &buf}
	err = cluster.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		t.Error(err)
	}
	alternatives := []string{
		"Pulling from 1!Pulling from 2!",
		"Pulling from 2!Pulling from 1!",
	}
	if r := buf.String(); r != alternatives[0] && r != alternatives[1] {
		t.Errorf("Wrong output: Want %q. Got %q.", "Pulling from 1!Pulling from 2!", buf.String())
	}
}

func TestPullImageNotFound(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No such image", http.StatusNotFound)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No such image", http.StatusNotFound)
	}))
	defer server2.Close()
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	opts := docker.PullImageOptions{Repository: "tsuru/python", OutputStream: &buf}
	err = cluster.PullImage(opts, docker.AuthConfiguration{})
	if err == nil {
		t.Error("PullImage: got unexpected <nil> error")
	}
}

func TestPullImageSpecifyNode(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pulling from 1!"))
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pulling from 2!"))
	}))
	defer server2.Close()
	var buf safe.Buffer
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	opts := docker.PullImageOptions{Repository: "tsuru/python", OutputStream: &buf}
	err = cluster.PullImage(opts, docker.AuthConfiguration{}, "handler1")
	if err != nil {
		t.Error(err)
	}
	expected := "Pulling from 2!"
	if r := buf.String(); r != expected {
		t.Errorf("Wrong output: Want %q. Got %q.", expected, r)
	}
}

func TestPullImageSpecifyMultipleNodes(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pulling from 1!"))
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pulling from 2!"))
	}))
	defer server2.Close()
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pulling from 3!"))
	}))
	defer server3.Close()
	var buf safe.Buffer
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
		Node{ID: "handler2", Address: server3.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	opts := docker.PullImageOptions{Repository: "tsuru/python", OutputStream: &buf}
	err = cluster.PullImage(opts, docker.AuthConfiguration{}, "handler1", "handler2")
	if err != nil {
		t.Error(err)
	}
	alternatives := []string{
		"Pulling from 2!Pulling from 3!",
		"Pulling from 3!Pulling from 2!",
	}
	if r := buf.String(); r != alternatives[0] && r != alternatives[1] {
		t.Errorf("Wrong output: Want %q. Got %q.", "Pulling from 2!Pulling from 3!", r)
	}
}

func TestPushImage(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pushing to server 1!"))
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Pushing to server 2!"))
	}))
	defer server2.Close()
	var buf safe.Buffer
	cluster, err := New(nil, &mapStorage{iMap: map[string]string{"tsuru/ruby": "handler0"}},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	var auth docker.AuthConfiguration
	err = cluster.PushImage(docker.PushImageOptions{Name: "tsuru/ruby", OutputStream: &buf}, auth)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`^Pushing to server \d`)
	if !re.MatchString(buf.String()) {
		t.Errorf("Wrong output: Want %q. Got %q.", "Pushing to server [12]", buf.String())
	}
}

func TestPushImageNotFound(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No such image", http.StatusNotFound)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No such image", http.StatusNotFound)
	}))
	defer server2.Close()
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	var auth docker.AuthConfiguration
	err = cluster.PushImage(docker.PushImageOptions{Name: "tsuru/python", OutputStream: &buf}, auth)
	if err == nil {
		t.Error("PushImage: got unexpected <nil> error")
	}
}

func TestPushImageWithStorage(t *testing.T) {
	var count int
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pushed"))
	}))
	defer server2.Close()
	storage := mapStorage{iMap: map[string]string{"tsuru/python": "handler1"}}
	cluster, err := New(nil, &storage,
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	var auth docker.AuthConfiguration
	err = cluster.PushImage(docker.PushImageOptions{Name: "tsuru/python", OutputStream: &buf}, auth)
	if err != nil {
		t.Error(err)
	}
	if count > 0 {
		t.Error("PushImage with storage: should not send request to all servers, but did.")
	}
}

func TestImportImage(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("importing from 1"))
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("importing from 2"))
	}))
	defer server2.Close()
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	var buf safe.Buffer
	opts := docker.ImportImageOptions{
		Repository:   "tsuru/python",
		Source:       "http://url.to/tar",
		OutputStream: &buf,
	}
	err = cluster.ImportImage(opts)
	if err != nil {
		t.Error(err)
	}
	re := regexp.MustCompile(`^importing from \d`)
	if !re.MatchString(buf.String()) {
		t.Errorf("Wrong output: Want %q. Got %q.", "importing from [12]", buf.String())
	}
}

func TestImportImageWithAbsentFile(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "file not found", http.StatusNotFound)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "file not found", http.StatusNotFound)
	}))
	defer server2.Close()
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
		Node{ID: "handler1", Address: server2.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	var buf safe.Buffer
	opts := docker.ImportImageOptions{
		Repository:   "tsuru/python",
		Source:       "/path/to/tar",
		OutputStream: &buf,
	}
	err = cluster.ImportImage(opts)
	if err == nil {
		t.Error("ImportImage: got unexpected <nil> error")
	}
}

func TestBuildImage(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer server1.Close()
	cluster, err := New(nil, &mapStorage{},
		Node{ID: "handler0", Address: server1.URL},
	)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buildOptions := docker.BuildImageOptions{
		Name:         "tsuru/python",
		Remote:       "http://localhost/Dockerfile",
		InputStream:  nil,
		OutputStream: &buf,
	}
	err = cluster.BuildImage(buildOptions)
	if err != nil {
		t.Error(err)
	}
	_, err = cluster.storage().RetrieveImage("tsuru/python")
	if err != nil {
		t.Error(err)
	}
}
