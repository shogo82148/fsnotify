// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !plan9 && !solaris
// +build !plan9,!solaris

package fsnotify

import (
	"io/ioutil"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// An atomic counter
type counter struct {
	val int32
}

func (c *counter) increment() {
	atomic.AddInt32(&c.val, 1)
}

func (c *counter) value() int32 {
	return atomic.LoadInt32(&c.val)
}

func (c *counter) reset() {
	atomic.StoreInt32(&c.val, 0)
}

// tempMkdir makes a temporary directory
func tempMkdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "fsnotify")
	if err != nil {
		t.Fatalf("failed to create test directory: %s", err)
	}
	return dir
}

// tempMkFile makes a temporary file.
func tempMkFile(t *testing.T, dir string) string {
	f, err := ioutil.TempFile(dir, "fsnotify")
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer f.Close()
	return f.Name()
}

func TestWatch(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			"multiple creates",
			func(t *testing.T, w *Watcher, tempDir string) {
				file := filepath.Join(tempDir, "file")
				addWatch(t, w, tempDir)

				cat(t, "data", file)
				rm(t, file)

				touch(t, file)       // Recreate the file
				cat(t, "data", file) // Modify
				cat(t, "data", file) // Modify
			},
			`
				create  /file
				write   /file
				remove  /file
				create  /file
				write   /file
				write   /file
			`,
		},
		{
			"dir only",
			func(t *testing.T, w *Watcher, tempDir string) {
				beforeWatch := filepath.Join(tempDir, "beforeWatch")
				file := filepath.Join(tempDir, "file")

				touch(t, beforeWatch)
				addWatch(t, w, tempDir)

				cat(t, "data", file)
				rm(t, file)
				rm(t, beforeWatch)
			},
			`
				create /file
				write  /file
				remove /file
				remove /beforeWatch
			`,
		},
		{
			"subdir",
			func(t *testing.T, w *Watcher, tempDir string) {
				addWatch(t, w, tempDir)

				file := filepath.Join(tempDir, "file")
				dir := filepath.Join(tempDir, "sub")
				dirfile := filepath.Join(tempDir, "sub", "file2")

				mkdir(t, dir)     // Create sub-directory
				touch(t, file)    // Create a file
				touch(t, dirfile) // Create a file (Should not see this! we are not watching subdir)
				time.Sleep(200 * time.Millisecond)
				rmAll(t, dir) // Make sure receive deletes for both file and sub-directory
				rm(t, file)
			},
			`
				create /sub
				create /file
				remove /sub
				remove /file

				# Windows includes a write for the /sub dir too, two of them even(?)
				windows:
					create /sub
					create /file
					write  /sub
					write  /sub
					remove /sub
					remove /file
			`,
		},
	}

	for _, tt := range tests {
		tt := tt
		tt.run(t)
	}
}
