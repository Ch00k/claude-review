package main

import (
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher   *fsnotify.Watcher
	watches   map[string]bool // Track watched files
	mu        sync.RWMutex
	callbacks map[string]func() // Callbacks per file path
}

var fileWatcher *FileWatcher

func initFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	fileWatcher = &FileWatcher{
		watcher:   watcher,
		watches:   make(map[string]bool),
		callbacks: make(map[string]func()),
	}

	// Start event processing in background
	go fileWatcher.processEvents()

	return nil
}

func (fw *FileWatcher) processEvents() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Only care about write/create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				fw.mu.RLock()
				callback, exists := fw.callbacks[event.Name]
				fw.mu.RUnlock()

				if exists && callback != nil {
					callback()
				}
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

func (fw *FileWatcher) watchFile(projectDir, filePath string, callback func()) error {
	absPath := filepath.Join(projectDir, filePath)

	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Check if already watching
	if fw.watches[absPath] {
		// Update callback
		fw.callbacks[absPath] = callback
		return nil
	}

	// Add watch
	if err := fw.watcher.Add(absPath); err != nil {
		return err
	}

	fw.watches[absPath] = true
	fw.callbacks[absPath] = callback

	log.Printf("Started watching file: %s", absPath)
	return nil
}

func (fw *FileWatcher) unwatchFile(projectDir, filePath string) error {
	absPath := filepath.Join(projectDir, filePath)

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.watches[absPath] {
		return nil
	}

	if err := fw.watcher.Remove(absPath); err != nil {
		return err
	}

	delete(fw.watches, absPath)
	delete(fw.callbacks, absPath)

	log.Printf("Stopped watching file: %s", absPath)
	return nil
}

func (fw *FileWatcher) close() error {
	if fw.watcher != nil {
		return fw.watcher.Close()
	}
	return nil
}
