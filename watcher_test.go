// Copyright 2022 The fsnotify project. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsnotify

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var supportedPlatforms = []string{
	"darwin", "dragonfly", "freebsd", "openbsd", "linux",
	"netbsd", "windows",
}

func commonCreateWatcher(t *testing.T) *Watcher {
	watcher, err := NewWatcher()
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			for _, supported := range supportedPlatforms {
				if runtime.GOOS == supported {
					t.Fatalf("%s should be supported but watcher is reporting unsupported", runtime.GOOS)
				}
			}
			t.Skipf("Watcher is not supported on %s", runtime.GOOS)
		}
		t.Fatalf("Unexpected error when creating a watcher: %v", err)
	}
	return watcher
}

func TestWatcherCreateFile(t *testing.T) {
	watcher := commonCreateWatcher(t)
	defer watcher.Close()

	tmpdir, err := os.MkdirTemp("", "watcher")
	if err != nil {
		t.Errorf("Unable to create temporary directory for watching for changes: %v", err)
		return
	}
	defer os.RemoveAll(tmpdir)

	// Now watch the temporary directory
	err = watcher.Add(tmpdir)
	if err != nil {
		t.Errorf("Unable to watch the temporary directory: %v", err)
		return
	}

	select {
	case err := <-watcher.Errors:
		t.Errorf("Unexpected error: %v", err)
		return
	case event := <-watcher.Events:
		t.Errorf("Unexpected event: %v", event)
		return
	case <-time.After(100 * time.Millisecond):
		// No-op
	}

	err = os.WriteFile(filepath.Join(tmpdir, "hello"), []byte("Hello, Gophers!"), 0o666)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	select {
	case err := <-watcher.Errors:
		t.Errorf("Unexpected error: %v", err)
		return
	case event := <-watcher.Events:
		if event.Op != Create {
			t.Errorf("Unexpected event: %v", event)
			return
		}
	case <-time.After(10 * time.Second):
		t.Error("Watcher took too long to react to creation event")
		return
	}

	err = watcher.Close()
	if err != nil {
		t.Errorf("Unable to close the watcher: %v", err)
		return
	}
}

func TestWatcherCreateDirectory(t *testing.T) {
	watcher := commonCreateWatcher(t)
	defer watcher.Close()

	tmpdir, err := os.MkdirTemp("", "watcher")
	if err != nil {
		t.Errorf("Unable to create temporary directory for watching for changes: %v", err)
		return
	}
	defer os.RemoveAll(tmpdir)

	// Now watch the temporary directory
	err = watcher.Add(tmpdir)
	if err != nil {
		t.Errorf("Unable to watch the temporary directory: %v", err)
		return
	}

	select {
	case err := <-watcher.Errors:
		t.Errorf("Unexpected error: %v", err)
		return
	case event := <-watcher.Events:
		t.Errorf("Unexpected event: %v", event)
		return
	case <-time.After(100 * time.Millisecond):
		// No-op
	}

	err = os.Mkdir(filepath.Join(tmpdir, "hello"), 0o666)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	select {
	case err := <-watcher.Errors:
		t.Errorf("Unexpected error: %v", err)
		return
	case event := <-watcher.Events:
		if event.Op != Create {
			t.Errorf("Unexpected event: %v", event)
			return
		}
	case <-time.After(10 * time.Second):
		t.Error("Watcher took too long to react to creation event")
		return
	}

	err = watcher.Close()
	if err != nil {
		t.Errorf("Unable to close the watcher: %v", err)
		return
	}
}

func TestWatcherModifyFile(t *testing.T) {
	watcher := commonCreateWatcher(t)
	defer watcher.Close()

	tmpdir, err := os.MkdirTemp("", "watcher")
	if err != nil {
		t.Errorf("Unable to create temporary directory for watching for changes: %v", err)
		return
	}
	defer os.RemoveAll(tmpdir)

	err = os.Mkdir(filepath.Join(tmpdir, "hello3"), 0o777)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	filenames := []string{
		filepath.Join(tmpdir, "hello"),
		filepath.Join(tmpdir, "hello2"),
		filepath.Join(tmpdir, "hello3", "hello3"),
		filepath.Join(tmpdir, "hello4"),
	}
	for _, filename := range filenames {
		err = os.WriteFile(filename, []byte("Hello, Gophers!"), 0o666)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
	}

	// Now watch the temporary files
	for _, filename := range filenames {
		err = watcher.Add(filename)
		if err != nil {
			t.Errorf("Unable to watch the temporary file: %v", err)
			return
		}
	}

	select {
	case err := <-watcher.Errors:
		t.Errorf("Unexpected error: %v", err)
		return
	case event := <-watcher.Events:
		t.Errorf("Unexpected event: %v", event)
		return
	case <-time.After(100 * time.Millisecond):
		// No-op
	}

	// modify files
	for _, filename := range filenames {
		err = os.WriteFile(filename, []byte("Changed"), 0o666)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		select {
		case err := <-watcher.Errors:
			t.Errorf("Unexpected error: %v", err)
			return
		case event := <-watcher.Events:
			if event.Op != Write && event.Name != filename {
				t.Errorf("Unexpected event: %v", event)
				return
			}
		case <-time.After(10 * time.Second):
			t.Error("Watcher took too long to react to write event")
			return
		}
	}

	// remove files
	for _, filename := range filenames {
		err = os.Remove(filename)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
	loop:
		for {
			select {
			case err := <-watcher.Errors:
				t.Errorf("Unexpected error: %v", err)
				return
			case event := <-watcher.Events:
				if event.Op == Remove {
					break loop
				}
			case <-time.After(10 * time.Second):
				t.Errorf("Watcher took too long to react to remove event for %s", filename)
				return
			}
		}
	}

	err = watcher.Close()
	if err != nil {
		t.Errorf("Unable to close the watcher: %v", err)
		return
	}
}
