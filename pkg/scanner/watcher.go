package scanner

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	roots    []ScanRoot
	onChange func()
	mu       sync.Mutex
	timer    *time.Timer
	debounce time.Duration
}

func NewWatcher(roots []ScanRoot, debounce time.Duration, onChange func()) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  fsw,
		roots:    roots,
		onChange: onChange,
		debounce: debounce,
	}

	for _, root := range roots {
		w.addRecursive(root.Path)
	}

	return w, nil
}

func (w *Watcher) addRecursive(dir string) {
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				log.Printf("watcher: failed to watch %s: %v", path, err)
			}
		}
		return nil
	})
}

func (w *Watcher) Run(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			w.watcher.Close()
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.addRecursive(event.Name)
				}
				w.scheduleRescan()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) scheduleRescan() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, func() {
		log.Println("filesystem change detected, rescanning")
		w.onChange()
	})
}
