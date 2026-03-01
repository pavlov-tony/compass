package iplookup

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// StartWatcher starts a background goroutine to watch for database file changes.
// It watches the parent directory to handle atomic saves (like those from Vim or rsync)
// where the file's inode changes.
func (i *IPLookup) StartWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Failed to create watcher: %v", err)
		return err
	}

	// Watch the parent directory instead of the file itself.
	// This ensures the watch is not lost when the file is replaced or renamed.
	watchDir := filepath.Dir(i.dbPath)
	if err := watcher.Add(watchDir); err != nil {
		log.Errorf("Failed to watch directory %s: %v", watchDir, err)
		watcher.Close()
		return err
	}

	log.Infof("Watching directory %s for changes to %s", watchDir, i.dbPath)

	go func() {
		// Ensure the watcher is closed when the goroutine exits.
		defer watcher.Close()

		var timer *time.Timer
		// Debounce duration to avoid multiple reloads during a burst of events.
		const debounceDuration = 500 * time.Millisecond

		for {
			select {
			case <-i.quit:
				// Stop the timer if it's running and exit.
				if timer != nil {
					timer.Stop()
				}
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Filter events: we only care about our specific database file.
				if filepath.Clean(event.Name) != filepath.Clean(i.dbPath) {
					continue
				}

				// React to Write, Create, or Rename events (typical for atomic saves).
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
					// Debouncing logic: reset the timer on every relevant event.
					if timer != nil {
						timer.Stop()
					}

					timer = time.AfterFunc(debounceDuration, func() {
						log.Infof("Database %s changed, reloading...", i.dbPath)

						if err := i.Reload(); err != nil {
							// If reload fails, the plugin continues to serve the old DB instance.
							log.Errorf("Failed to reload database: %v", err)
						} else {
							log.Infof("Database reloaded successfully")
						}
					})
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Errorf("Watcher error: %v", err)
			}
		}
	}()

	return nil
}

// Reload creates a new reader instance and swaps it with the old one atomically.
func (i *IPLookup) Reload() error {
	// Initialize a fresh IPLookup instance to open the new DB file.
	newIPLookup, err := NewIPLookup(i.dbPath)
	if err != nil {
		return err
	}
	newDB := newIPLookup.db.Load()

	// Atomically swap the old DB pointer with the new one.
	oldDB := i.db.Swap(newDB)

	// Close the old DB reader if it exists.
	if oldDB != nil {
		oldDB.Close()
	}

	return nil
}

// Close shuts down the watcher and releases database resources.
func (i *IPLookup) Close() error {
	// Signal the watcher goroutine to terminate.
	close(i.quit)

	db := i.db.Load()
	if db != nil {
		return db.Close()
	}
	return nil
}
