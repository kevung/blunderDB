//go:build linux && (dev || production) && sheetpdftest

package gui

// #include <gtk/gtk.h>
import "C"

import "runtime"

// startGTKLoop runs a GTK main loop on a locked thread, the one Wails runs
// in the application, for the rendering test only.
func startGTKLoop() bool {
	ok := make(chan bool)
	go func() {
		runtime.LockOSThread()
		if C.gtk_init_check(nil, nil) == 0 {
			ok <- false
			return
		}
		ok <- true
		C.gtk_main()
	}()
	return <-ok
}
