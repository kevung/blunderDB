//go:build linux && (dev || production)

package gui

/*
#cgo linux pkg-config: gtk+-3.0
#cgo !webkit2_41 pkg-config: webkit2gtk-4.0
#cgo webkit2_41 pkg-config: webkit2gtk-4.1
#include <stdint.h>
#include <stdlib.h>
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

extern void sheetPDFDone(uintptr_t id, char *err);

// The page is ready once the fitting script of a print sheet has marked the
// root; a page without that script (an archive page) once it has loaded.
static const char *sheet_ready_js =
	"(function(){var g=document.querySelector('meta[name=generator][content=\"blunderDB print sheet\"]');"
	"return document.readyState==='complete'&&(!g||document.documentElement.hasAttribute('data-fitted'));})()";

typedef struct {
	uintptr_t id;
	char *html;
	char *uri;
	char *freeze;
	GtkWidget *win;
	WebKitWebView *view;
	WebKitPrintOperation *op;
	char *err;
	int polls;
	gboolean done;
} sheet_job;

static gboolean sheet_free(gpointer data) {
	sheet_job *j = data;
	if (j->op) g_object_unref(j->op);
	gtk_widget_destroy(j->win);
	g_free(j->html);
	g_free(j->uri);
	g_free(j->freeze);
	g_free(j->err);
	g_free(j);
	return G_SOURCE_REMOVE;
}

static void sheet_finish(sheet_job *j, const char *err) {
	if (j->done) return;
	j->done = TRUE;
	sheetPDFDone(j->id, (char *)err);
	// Not from inside a signal of the view or of its print operation.
	g_idle_add(sheet_free, j);
}

static void sheet_print_failed(WebKitPrintOperation *op, GError *e, gpointer data) {
	sheet_job *j = data;
	if (!j->err) j->err = g_strdup(e && e->message ? e->message : "print failed");
}

static void sheet_print_finished(WebKitPrintOperation *op, gpointer data) {
	sheet_job *j = data;
	sheet_finish(j, j->err);
}

static void sheet_print(sheet_job *j) {
	GtkPrintSettings *s = gtk_print_settings_new();
	gtk_print_settings_set_printer(s, "Print to File");
	gtk_print_settings_set(s, GTK_PRINT_SETTINGS_OUTPUT_FILE_FORMAT, "pdf");
	gtk_print_settings_set(s, GTK_PRINT_SETTINGS_OUTPUT_URI, j->uri);
	GtkPaperSize *paper = gtk_paper_size_new(GTK_PAPER_NAME_A4);
	gtk_print_settings_set_paper_size(s, paper);
	gtk_print_settings_set_orientation(s, GTK_PAGE_ORIENTATION_PORTRAIT);
	// A 297 mm sheet is 1123 CSS pixels, the A4 page 1122.67: at full scale
	// the third of a pixel spills onto a blank page Chromium would drop.
	gtk_print_settings_set_scale(s, 99.8);
	GtkPageSetup *setup = gtk_page_setup_new();
	gtk_page_setup_set_paper_size(setup, paper);
	gtk_page_setup_set_orientation(setup, GTK_PAGE_ORIENTATION_PORTRAIT);
	gtk_page_setup_set_top_margin(setup, 0, GTK_UNIT_MM);
	gtk_page_setup_set_bottom_margin(setup, 0, GTK_UNIT_MM);
	gtk_page_setup_set_left_margin(setup, 0, GTK_UNIT_MM);
	gtk_page_setup_set_right_margin(setup, 0, GTK_UNIT_MM);

	j->op = webkit_print_operation_new(j->view);
	webkit_print_operation_set_print_settings(j->op, s);
	webkit_print_operation_set_page_setup(j->op, setup);
	g_signal_connect(j->op, "failed", G_CALLBACK(sheet_print_failed), j);
	g_signal_connect(j->op, "finished", G_CALLBACK(sheet_print_finished), j);
	webkit_print_operation_print(j->op);
	g_object_unref(s);
	g_object_unref(setup);
	gtk_paper_size_free(paper);
}

static gboolean sheet_poll(gpointer data);

static gboolean sheet_js_finish(sheet_job *j, GObject *src, GAsyncResult *res, gboolean *value) {
	GError *e = NULL;
	gboolean ok = FALSE;
#if WEBKIT_CHECK_VERSION(2, 40, 0)
	JSCValue *v = webkit_web_view_evaluate_javascript_finish(WEBKIT_WEB_VIEW(src), res, &e);
	if (v) {
		ok = TRUE;
		*value = jsc_value_to_boolean(v);
		g_object_unref(v);
	}
#else
	WebKitJavascriptResult *r = webkit_web_view_run_javascript_finish(WEBKIT_WEB_VIEW(src), res, &e);
	if (r) {
		ok = TRUE;
		*value = jsc_value_to_boolean(webkit_javascript_result_get_js_value(r));
		webkit_javascript_result_unref(r);
	}
#endif
	if (e) {
		sheet_finish(j, e->message);
		g_error_free(e);
		return FALSE;
	}
	if (!ok) sheet_finish(j, "script failed");
	return ok;
}

static void sheet_run_js(sheet_job *j, const char *js, GAsyncReadyCallback cb) {
#if WEBKIT_CHECK_VERSION(2, 40, 0)
	webkit_web_view_evaluate_javascript(j->view, js, -1, NULL, NULL, NULL, cb, j);
#else
	webkit_web_view_run_javascript(j->view, js, NULL, cb, j);
#endif
}

static void sheet_frozen(GObject *src, GAsyncResult *res, gpointer data) {
	sheet_job *j = data;
	gboolean v = FALSE;
	if (sheet_js_finish(j, src, res, &v)) sheet_print(j);
}

static void sheet_ready(GObject *src, GAsyncResult *res, gpointer data) {
	sheet_job *j = data;
	gboolean ready = FALSE;
	if (!sheet_js_finish(j, src, res, &ready)) return;
	if (ready) {
		sheet_run_js(j, j->freeze, sheet_frozen);
	} else if (++j->polls > 600) {
		sheet_finish(j, "the page layout did not finish");
	} else {
		g_timeout_add(50, sheet_poll, j);
	}
}

static gboolean sheet_poll(gpointer data) {
	sheet_run_js(data, sheet_ready_js, sheet_ready);
	return G_SOURCE_REMOVE;
}

static void sheet_load_changed(WebKitWebView *v, WebKitLoadEvent ev, gpointer data) {
	if (ev == WEBKIT_LOAD_FINISHED) sheet_poll(data);
}

static gboolean sheet_load_failed(WebKitWebView *v, WebKitLoadEvent ev, gchar *uri, GError *e, gpointer data) {
	sheet_finish(data, e && e->message ? e->message : "the page did not load");
	return TRUE;
}

// sheet_start runs on the GTK main loop: an off-screen window the user never
// sees, its own ephemeral context, nothing shared with the application's view.
static gboolean sheet_start(gpointer data) {
	sheet_job *j = data;
	WebKitWebContext *ctx = webkit_web_context_new_ephemeral();
	j->view = WEBKIT_WEB_VIEW(webkit_web_view_new_with_context(ctx));
	g_object_unref(ctx);
	WebKitSettings *st = webkit_web_view_get_settings(j->view);
	webkit_settings_set_print_backgrounds(st, TRUE);
	webkit_settings_set_enable_javascript(st, TRUE);
	j->win = gtk_offscreen_window_new();
	// The width of an A4 page at 96 dpi: the layout the print will keep.
	gtk_window_set_default_size(GTK_WINDOW(j->win), 794, 1123);
	gtk_container_add(GTK_CONTAINER(j->win), GTK_WIDGET(j->view));
	gtk_widget_show_all(j->win);
	g_signal_connect(j->view, "load-changed", G_CALLBACK(sheet_load_changed), j);
	g_signal_connect(j->view, "load-failed", G_CALLBACK(sheet_load_failed), j);
	webkit_web_view_load_html(j->view, j->html, "about:blank");
	return G_SOURCE_REMOVE;
}

static void sheet_pdf_start(uintptr_t id, char *html, char *uri, char *freeze) {
	sheet_job *j = g_new0(sheet_job, 1);
	j->id = id;
	j->freeze = g_strdup(freeze);
	j->html = g_strdup(html);
	j->uri = g_strdup(uri);
	g_idle_add(sheet_start, j);
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

// sheetPDFTimeout bounds a rendering whose callbacks never come: the GTK
// loop stopped, or the print backend hangs.
const sheetPDFTimeout = 90 * time.Second

var (
	sheetJobsMu sync.Mutex
	sheetJobs   = map[uintptr]chan error{}
	sheetJobSeq uintptr
)

// freezeColumnsJS pins every multi-column box as the screen laid it out:
// each child placed absolutely where its column put it, the column rules
// drawn as lines (a block running over two columns gives way to its
// children, placed one by one). WebKitGTK's print layout gives every column the width of
// the whole box, so a sheet printed as it stands keeps one column of
// several and loses the rest past the right edge. A box holding bare text,
// or text split over two columns, is left as it is.
const freezeColumnsJS = `(function () {
  var boxes = Array.prototype.filter.call(document.querySelectorAll('body *'), function (m) {
    var c = getComputedStyle(m);
    return c.columnCount !== 'auto' || c.columnWidth !== 'auto';
  });
  var plans = boxes.map(function (m) {
    var bare = Array.prototype.some.call(m.childNodes, function (n) { return n.nodeType === 3 && n.textContent.trim(); });
    var r = m.getBoundingClientRect(), c = getComputedStyle(m), split = false, kids = [], opened = [];
    (function collect(el) {
      Array.prototype.forEach.call(el.children, function (k) {
        if (k.getClientRects().length > 1) {
          var text = Array.prototype.some.call(k.childNodes, function (n) { return n.nodeType === 3 && n.textContent.trim(); });
          if (text || !k.children.length) { split = true; return; }
          opened.push(k);
          collect(k);
          return;
        }
        var kr = k.getBoundingClientRect();
        kids.push({ k: k, l: kr.left - r.left - m.clientLeft, t: kr.top - r.top - m.clientTop, w: kr.width });
      });
    })(m);
    return { m: m, skip: bare || split, w: r.width, h: r.height, kids: kids, opened: opened, c: c,
      gap: parseFloat(c.columnGap) || 0 };
  });
  plans.forEach(function (p) {
    if (p.skip) return;
    var m = p.m, s = m.style, rule = p.c.columnRuleStyle !== 'none' && parseFloat(p.c.columnRuleWidth) > 0;
    var ruleCSS = p.c.columnRuleWidth + ' ' + p.c.columnRuleStyle + ' ' + p.c.columnRuleColor;
    var lefts = {};
    s.boxSizing = 'border-box'; s.width = p.w + 'px'; s.height = p.h + 'px';
    s.columnCount = 'auto'; s.columnWidth = 'auto'; s.columnRuleStyle = 'none';
    if (p.c.position === 'static') s.position = 'relative';
    // A block that runs over two columns gives way to its children.
    p.opened.forEach(function (k) { k.style.display = 'contents'; });
    p.kids.forEach(function (x) {
      var ks = x.k.style;
      ks.position = 'absolute'; ks.margin = '0'; ks.boxSizing = 'border-box';
      ks.left = x.l + 'px'; ks.top = x.t + 'px'; ks.width = x.w + 'px';
      lefts[Math.round(x.l)] = true;
    });
    if (!rule) return;
    Object.keys(lefts).map(Number).sort(function (a, b) { return a - b; }).slice(1).forEach(function (l) {
      var d = document.createElement('div');
      d.style.cssText = 'position:absolute;top:0;bottom:0;width:0;border-left:' + ruleCSS + ';left:' + (l - p.gap / 2) + 'px';
      m.appendChild(d);
    });
  });
  return true;
})()`

//export sheetPDFDone
func sheetPDFDone(id C.uintptr_t, cerr *C.char) {
	var err error
	if cerr != nil {
		err = errors.New(C.GoString(cerr))
	}
	sheetJobsMu.Lock()
	ch, ok := sheetJobs[uintptr(id)]
	delete(sheetJobs, uintptr(id))
	sheetJobsMu.Unlock()
	if ok {
		ch <- err
	}
}

func sheetPDFAvailable() bool { return true }

// renderSheetPDF prints page to an A4 PDF in WebKitGTK, off screen. Every
// GTK call happens on the GTK main loop (the one Wails runs); this goroutine
// only waits.
func renderSheetPDF(ctx context.Context, page []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "blunderdb-sheet-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "sheet.pdf")
	uri := (&url.URL{Scheme: "file", Path: out}).String()

	ch := make(chan error, 1)
	sheetJobsMu.Lock()
	sheetJobSeq++
	id := sheetJobSeq
	sheetJobs[id] = ch
	sheetJobsMu.Unlock()
	abandon := func() {
		sheetJobsMu.Lock()
		delete(sheetJobs, id)
		sheetJobsMu.Unlock()
	}

	chtml, curi, cfreeze := C.CString(string(page)), C.CString(uri), C.CString(freezeColumnsJS)
	C.sheet_pdf_start(C.uintptr_t(id), chtml, curi, cfreeze)
	C.free(unsafe.Pointer(chtml))
	C.free(unsafe.Pointer(curi))
	C.free(unsafe.Pointer(cfreeze))

	timer := time.NewTimer(sheetPDFTimeout)
	defer timer.Stop()
	select {
	case err := <-ch:
		if err != nil {
			return nil, fmt.Errorf("PDF rendering: %w", err)
		}
	case <-ctx.Done():
		abandon()
		return nil, ctx.Err()
	case <-timer.C:
		abandon()
		return nil, errors.New("PDF rendering timed out")
	}
	return os.ReadFile(out)
}
