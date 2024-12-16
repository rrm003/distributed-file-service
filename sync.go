package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

const (
	downloadsDir = "./downloads"
	uploadsDir   = "./uploads"
)

func main() {
	// Create watcher for downloads directory
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer watcher.Close()

	// Channel to signal when done
	done := make(chan bool)
	var wg sync.WaitGroup

	// Start watching the downloads directory
	wg.Add(1)
	go watchDirectory(downloadsDir, watcher, &wg)

	// Handle events
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				handleEvent(event)
			case err := <-watcher.Errors:
				fmt.Println("Error:", err)
			}
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()
	done <- true
}

func watchDirectory(dir string, watcher *fsnotify.Watcher, wg *sync.WaitGroup) {
	defer wg.Done()

	err := watcher.Add(dir)
	if err != nil {
		fmt.Println("Error watching directory:", err)
		return
	}
	fmt.Println("Watching directory:", dir)

	<-make(chan struct{})
}

func handleEvent(event fsnotify.Event) {
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		fmt.Println("File created:", event.Name)
		copyFile(event.Name)
	case event.Op&fsnotify.Write == fsnotify.Write:
		fmt.Println("File updated:", event.Name)
		copyFile(event.Name)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// Check if the file exists to distinguish between rename and delete
		_, err := os.Stat(event.Name)
		if err == nil {
			fmt.Println("File renamed:", event.Name)
			renameFile(event.Name)
		} else if os.IsNotExist(err) {
			fmt.Println("File deleted:", event.Name)
			deleteFile(event.Name)
		}
	}
}

func copyFile(srcFile string) {
	destFile := filepath.Join(uploadsDir, filepath.Base(srcFile))

	src, err := os.Open(srcFile)
	if err != nil {
		fmt.Println("Error opening source file:", err)
		return
	}
	defer src.Close()

	dest, err := os.Create(destFile)
	if err != nil {
		fmt.Println("Error creating destination file:", err)
		return
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		fmt.Println("Error copying file:", err)
		return
	}

	fmt.Println("File copied to:", destFile)
}

func renameFile(oldPath string) {
	newPath := filepath.Join(uploadsDir, filepath.Base(oldPath))

	err := os.Rename(oldPath, newPath)
	if err != nil {
		fmt.Println("Error renaming file:", err)
		return
	}

	fmt.Printf("File renamed from %s to %s\n", oldPath, newPath)
}

func deleteFile(path string) {
	err := os.Remove(filepath.Join(uploadsDir, filepath.Base(path)))
	if err != nil {
		fmt.Println("Error deleting file:", err)
		return
	}

	fmt.Println("File deleted:", path)
}
